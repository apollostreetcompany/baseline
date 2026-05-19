import Foundation

enum BaselineFormat {
  static func ms(_ value: Int) -> String {
    if value >= 1000 {
      return String(format: "%.1fs", Double(value) / 1000)
    }
    return "\(value)ms"
  }

  static func signed(_ value: Int) -> String {
    value > 0 ? "+\(value)" : "\(value)"
  }

  static func shortRun(_ id: String) -> String {
    id.replacingOccurrences(of: "run_", with: "").prefix(12).description
  }
}
