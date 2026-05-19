import Foundation

struct AgentProviderBridge {
  func analyze(prompt: String) async -> String? {
    await withCheckedContinuation { continuation in
      DispatchQueue.global(qos: .userInitiated).async {
        continuation.resume(returning: runProvider(prompt: prompt))
      }
    }
  }
}

private func runProvider(prompt: String) -> String? {
  let candidates = [
    ("/opt/homebrew/bin/hermes", ["chat", "--quiet", "--query", prompt]),
    ("/usr/local/bin/hermes", ["chat", "--quiet", "--query", prompt]),
    ("/opt/homebrew/bin/openclaw", ["run", "--prompt", prompt]),
    ("/usr/local/bin/openclaw", ["run", "--prompt", prompt])
  ]
  for candidate in candidates {
    if let output = runCommand(path: candidate.0, arguments: candidate.1), !output.isEmpty {
      return output
    }
  }
  return nil
}

private func runCommand(path: String, arguments: [String]) -> String? {
  guard FileManager.default.isExecutableFile(atPath: path) else { return nil }
  let process = Process()
  process.executableURL = URL(fileURLWithPath: path)
  process.arguments = arguments
  let output = Pipe()
  let error = Pipe()
  process.standardOutput = output
  process.standardError = error
  do {
    try process.run()
  } catch {
    return nil
  }
  let deadline = DispatchTime.now() + .seconds(45)
  let semaphore = DispatchSemaphore(value: 0)
  process.terminationHandler = { _ in semaphore.signal() }
  if semaphore.wait(timeout: deadline) == .timedOut {
    process.terminate()
    return nil
  }
  let data = output.fileHandleForReading.readDataToEndOfFile()
  return String(data: data, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines)
}
