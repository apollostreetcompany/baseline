import Foundation

@MainActor
struct BaselineMCPClient {
  var endpoint: URL
  var sessionToken: String

  func callTool(_ name: String, arguments: [String: Any] = [:]) async throws -> [String: Any] {
    var request = URLRequest(url: endpoint.appending(path: "mcp"))
    request.httpMethod = "POST"
    request.setValue("application/json", forHTTPHeaderField: "content-type")
    request.setValue("application/json", forHTTPHeaderField: "accept")
    request.setValue("Bearer \(sessionToken)", forHTTPHeaderField: "authorization")
    request.httpBody = try JSONSerialization.data(withJSONObject: [
      "jsonrpc": "2.0",
      "id": UUID().uuidString,
      "method": "tools/call",
      "params": [
        "name": name,
        "arguments": arguments
      ]
    ])

    let (data, response) = try await URLSession.shared.data(for: request)
    let status = (response as? HTTPURLResponse)?.statusCode ?? 0
    let object = try JSONSerialization.jsonObject(with: data) as? [String: Any] ?? [:]
    if status == 401 {
      throw BaselineClientError.authenticationRequired(BaselineParser.string(object["authorization"], fallback: "Request a magic link."))
    }
    if status >= 400 {
      throw BaselineClientError.server(BaselineParser.string(object["error"], fallback: "Remote MCP call failed."))
    }
    if let error = object["error"] as? [String: Any] {
      throw BaselineClientError.server(BaselineParser.string(error["message"], fallback: "MCP error."))
    }
    let result = object["result"] as? [String: Any] ?? [:]
    if let structured = result["structuredContent"] as? [String: Any] {
      return structured
    }
    if let content = result["content"] as? [[String: Any]], let text = content.first?["text"] as? String {
      let data = Data(text.utf8)
      return (try? JSONSerialization.jsonObject(with: data) as? [String: Any]) ?? [:]
    }
    return result
  }

  func requestMagicLink(email: String) async throws {
    var request = URLRequest(url: endpoint.appending(path: "api/auth/magic-link"))
    request.httpMethod = "POST"
    request.setValue("application/json", forHTTPHeaderField: "content-type")
    request.httpBody = try JSONSerialization.data(withJSONObject: ["email": email])
    let (data, response) = try await URLSession.shared.data(for: request)
    let status = (response as? HTTPURLResponse)?.statusCode ?? 0
    if status >= 400 {
      let object = (try? JSONSerialization.jsonObject(with: data) as? [String: Any]) ?? [:]
      throw BaselineClientError.server(BaselineParser.string(object["error"], fallback: "Magic link request failed."))
    }
  }

  func consumeMagicLink(_ pastedValue: String) async throws -> String {
    let token = extractMagicToken(pastedValue)
    guard !token.isEmpty else { throw BaselineClientError.server("Paste the full magic link or token.") }
    var request = URLRequest(url: endpoint.appending(path: "api/auth/consume"))
    request.httpMethod = "POST"
    request.setValue("application/json", forHTTPHeaderField: "content-type")
    request.httpBody = try JSONSerialization.data(withJSONObject: ["token": token])
    let (data, response) = try await URLSession.shared.data(for: request)
    let status = (response as? HTTPURLResponse)?.statusCode ?? 0
    let object = (try? JSONSerialization.jsonObject(with: data) as? [String: Any]) ?? [:]
    if status >= 400 {
      throw BaselineClientError.server(BaselineParser.string(object["error"], fallback: "Magic link could not be consumed."))
    }
    let session = BaselineParser.string(object["session_token"])
    guard !session.isEmpty else { throw BaselineClientError.server("No session token returned.") }
    return session
  }

  private func extractMagicToken(_ value: String) -> String {
    let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
    if let url = URL(string: trimmed), let components = URLComponents(url: url, resolvingAgainstBaseURL: false) {
      return components.queryItems?.first(where: { $0.name == "token" })?.value ?? trimmed
    }
    return trimmed
  }
}

enum BaselineClientError: LocalizedError {
  case authenticationRequired(String)
  case server(String)

  var errorDescription: String? {
    switch self {
    case .authenticationRequired(let message): message
    case .server(let message): message
    }
  }
}
