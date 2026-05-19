import Foundation

struct OpenRouterClient {
  var apiKey: String
  var model: String

  func analyze(prompt: String) async throws -> String {
    var request = URLRequest(url: URL(string: "https://openrouter.ai/api/v1/chat/completions")!)
    request.httpMethod = "POST"
    request.setValue("Bearer \(apiKey)", forHTTPHeaderField: "authorization")
    request.setValue("application/json", forHTTPHeaderField: "content-type")
    request.setValue("BaselineHotspots", forHTTPHeaderField: "x-title")
    request.httpBody = try JSONSerialization.data(withJSONObject: [
      "model": model,
      "messages": [
        ["role": "system", "content": "You analyze redacted Baseline monitoring summaries. Do not infer raw prompts, raw responses, secrets, or local paths."],
        ["role": "user", "content": prompt]
      ],
      "temperature": 0.2
    ])
    let (data, response) = try await URLSession.shared.data(for: request)
    let status = (response as? HTTPURLResponse)?.statusCode ?? 0
    let object = try JSONSerialization.jsonObject(with: data) as? [String: Any] ?? [:]
    if status >= 400 {
      throw BaselineClientError.server(BaselineParser.string((object["error"] as? [String: Any])?["message"], fallback: "OpenRouter request failed."))
    }
    let choices = object["choices"] as? [[String: Any]] ?? []
    let message = choices.first?["message"] as? [String: Any] ?? [:]
    let content = BaselineParser.string(message["content"])
    if content.isEmpty { throw BaselineClientError.server("OpenRouter returned no analysis.") }
    return content
  }
}
