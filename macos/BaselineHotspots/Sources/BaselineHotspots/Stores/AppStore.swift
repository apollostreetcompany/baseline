import Foundation

@MainActor
final class AppStore: ObservableObject {
  @Published var endpointText = "https://trackbaseline.com"
  @Published var sessionToken = ""
  @Published var signInEmail = ""
  @Published var pastedMagicLink = ""
  @Published var openRouterKey = ""
  @Published var openRouterModel = "openai/gpt-4.1-mini"
  @Published var account = AccountSummary()
  @Published var runs: [BaselineRun] = []
  @Published var hotspots: [Hotspot] = []
  @Published var comparison: ComparisonSummary?
  @Published var nextActions: [String] = []
  @Published var insight = ""
  @Published var statusMessage = "Not connected"
  @Published var isLoading = false

  private let keychain = KeychainStore()
  private let providerBridge = AgentProviderBridge()

  var isSignedIn: Bool { !sessionToken.isEmpty }

  func loadFromKeychain() async {
    endpointText = keychain.string(for: "endpoint").isEmpty ? endpointText : keychain.string(for: "endpoint")
    sessionToken = keychain.string(for: "sessionToken")
    openRouterKey = keychain.string(for: "openRouterKey")
    openRouterModel = keychain.string(for: "openRouterModel").isEmpty ? openRouterModel : keychain.string(for: "openRouterModel")
    if isSignedIn {
      await refresh()
    }
  }

  func saveSettings() {
    keychain.set(endpointText, for: "endpoint")
    keychain.set(sessionToken, for: "sessionToken")
    keychain.set(openRouterKey, for: "openRouterKey")
    keychain.set(openRouterModel, for: "openRouterModel")
    statusMessage = "Settings saved"
  }

  func signOut() {
    sessionToken = ""
    keychain.remove("sessionToken")
    account = AccountSummary()
    runs = []
    hotspots = []
    comparison = nil
    insight = ""
    statusMessage = "Signed out"
  }

  func requestMagicLink() async {
    guard let client = clientForAuth() else { return }
    isLoading = true
    defer { isLoading = false }
    do {
      try await client.requestMagicLink(email: signInEmail)
      statusMessage = "Magic link requested. Check email, then paste the link or token here."
    } catch {
      statusMessage = error.localizedDescription
    }
  }

  func consumeMagicLink() async {
    guard let client = clientForAuth() else { return }
    isLoading = true
    defer { isLoading = false }
    do {
      let token = try await client.consumeMagicLink(pastedMagicLink)
      sessionToken = token
      saveSettings()
      pastedMagicLink = ""
      statusMessage = "Signed in"
      await refresh()
    } catch {
      statusMessage = error.localizedDescription
    }
  }

  func refresh() async {
    guard let client = client() else { return }
    isLoading = true
    defer { isLoading = false }
    do {
      let accountPayload = try await client.callTool("baseline_account", arguments: ["action": "status"])
      let historyPayload = try await client.callTool("baseline_history", arguments: ["limit": 30])
      let hotspotsPayload = try await client.callTool("baseline_hotspots", arguments: ["limit": 50])
      let comparisonPayload = try await client.callTool("baseline_compare")
      account = BaselineParser.account(from: accountPayload)
      runs = BaselineParser.runs(from: historyPayload)
      hotspots = BaselineParser.hotspots(from: hotspotsPayload)
      comparison = BaselineParser.comparison(from: comparisonPayload)
      nextActions = BaselineParser.nextActions(from: accountPayload)
      statusMessage = "Connected to Baseline cloud MCP"
    } catch {
      statusMessage = error.localizedDescription
    }
  }

  func generateInsight() async {
    let prompt = """
    Analyze this Baseline account summary for operator action. Prioritize recurring hotspots, worsening self-history, token/model anomalies, and recovery steps.

    \(BaselineParser.compactSignal(runs: runs, hotspots: hotspots, comparison: comparison))
    """
    isLoading = true
    insight = "Analyzing..."
    defer { isLoading = false }
    if let providerInsight = await providerBridge.analyze(prompt: prompt) {
      insight = providerInsight
      statusMessage = "Insight generated with local agent provider"
      return
    }
    guard !openRouterKey.isEmpty else {
      insight = "No local provider bridge responded. Add an OpenRouter API key in Settings for fallback analysis."
      statusMessage = "Insight needs provider configuration"
      return
    }
    do {
      insight = try await OpenRouterClient(apiKey: openRouterKey, model: openRouterModel).analyze(prompt: prompt)
      statusMessage = "Insight generated with OpenRouter fallback"
    } catch {
      insight = error.localizedDescription
      statusMessage = "Insight failed"
    }
  }

  private func clientForAuth() -> BaselineMCPClient? {
    guard let endpoint = URL(string: endpointText.trimmingCharacters(in: .whitespacesAndNewlines)) else {
      statusMessage = "Endpoint URL is invalid"
      return nil
    }
    return BaselineMCPClient(endpoint: endpoint, sessionToken: sessionToken)
  }

  private func client() -> BaselineMCPClient? {
    guard !sessionToken.isEmpty else {
      statusMessage = "Sign in with a magic link first"
      return nil
    }
    return clientForAuth()
  }
}
