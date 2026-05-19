import Foundation

struct AccountSummary: Equatable {
  var accountID: String = ""
  var planKey: String = "free"
  var status: String = "unknown"
  var entitlementStatus: String = "inactive"
  var monitoringEnabled: Bool = false
}

struct BaselineRun: Identifiable, Equatable {
  var id: String { runID }
  var runID: String
  var createdAt: String
  var workspace: String
  var agentKind: String
  var status: String
  var healthScore: Int
  var warningCount: Int
  var durationMS: Int
  var mode: String
}

struct Hotspot: Identifiable, Equatable {
  var id: String { checkID }
  var checkID: String
  var kind: String
  var warningCount: Int
  var runCount: Int
  var maxDurationMS: Int
  var latestStatus: String
  var latestRunID: String
  var averageScore: Double
}

struct ComparisonSummary: Equatable {
  var latestRunID: String
  var previousRunID: String
  var healthDelta: Int
  var warningDelta: Int
  var latestStatus: String
  var previousStatus: String
}

enum BaselineParser {
  static func account(from payload: [String: Any]) -> AccountSummary {
    let account = payload["account"] as? [String: Any] ?? [:]
    let entitlement = payload["entitlement"] as? [String: Any] ?? [:]
    return AccountSummary(
      accountID: string(account["id"]),
      planKey: string(account["plan_key"], fallback: "free"),
      status: string(account["status"], fallback: "unknown"),
      entitlementStatus: string(entitlement["status"], fallback: "inactive"),
      monitoringEnabled: bool(entitlement["monitoring_enabled"])
    )
  }

  static func runs(from payload: [String: Any]) -> [BaselineRun] {
    let rows = payload["runs"] as? [[String: Any]] ?? []
    return rows.map { row in
      BaselineRun(
        runID: string(row["run_id"], fallback: "unknown"),
        createdAt: string(row["created_at"]),
        workspace: string(row["workspace"], fallback: "workspace"),
        agentKind: string(row["agent_kind"], fallback: "agent"),
        status: string(row["status"], fallback: "unknown"),
        healthScore: int(row["health_score"]),
        warningCount: int(row["warning_count"]),
        durationMS: int(row["duration_ms"]),
        mode: string(row["mode"], fallback: "unknown")
      )
    }
  }

  static func hotspots(from payload: [String: Any]) -> [Hotspot] {
    let rows = payload["hotspots"] as? [[String: Any]] ?? []
    return rows.map { row in
      Hotspot(
        checkID: string(row["check_id"], fallback: "unknown"),
        kind: string(row["kind"]),
        warningCount: int(row["warning_count"]),
        runCount: int(row["run_count"]),
        maxDurationMS: int(row["max_duration_ms"]),
        latestStatus: string(row["latest_status"], fallback: "unknown"),
        latestRunID: string(row["latest_run_id"]),
        averageScore: double(row["average_score"])
      )
    }
  }

  static func comparison(from payload: [String: Any]) -> ComparisonSummary? {
    guard let row = payload["comparison"] as? [String: Any] else { return nil }
    return ComparisonSummary(
      latestRunID: string(row["latest_run_id"]),
      previousRunID: string(row["previous_run_id"]),
      healthDelta: int(row["health_delta"]),
      warningDelta: int(row["warning_delta"]),
      latestStatus: string(row["latest_status"], fallback: "unknown"),
      previousStatus: string(row["previous_status"], fallback: "unknown")
    )
  }

  static func nextActions(from payload: [String: Any]) -> [String] {
    payload["next_actions"] as? [String] ?? []
  }

  static func compactSignal(runs: [BaselineRun], hotspots: [Hotspot], comparison: ComparisonSummary?) -> String {
    let latest = runs.first
    let hotspotLines = hotspots.prefix(8).map {
      "- \($0.checkID): \($0.warningCount) warnings across \($0.runCount) runs, latest \($0.latestStatus), max \($0.maxDurationMS)ms"
    }.joined(separator: "\n")
    let comparisonLine: String
    if let comparison {
      comparisonLine = "Latest \(comparison.latestRunID) vs \(comparison.previousRunID): health delta \(comparison.healthDelta), warning delta \(comparison.warningDelta)."
    } else {
      comparisonLine = "No self-history comparison yet."
    }
    return """
    Baseline cloud summary only. Do not assume raw prompts or raw responses are available.
    Latest run: \(latest?.runID ?? "none"), score \(latest?.healthScore ?? 0), status \(latest?.status ?? "unknown"), warnings \(latest?.warningCount ?? 0).
    \(comparisonLine)
    Hotspots:
    \(hotspotLines.isEmpty ? "- none" : hotspotLines)
    """
  }

  static func string(_ value: Any?, fallback: String = "") -> String {
    if let string = value as? String { return string }
    if let value { return String(describing: value) }
    return fallback
  }

  static func int(_ value: Any?) -> Int {
    if let int = value as? Int { return int }
    if let double = value as? Double { return Int(double.rounded()) }
    if let string = value as? String { return Int(string) ?? 0 }
    return 0
  }

  static func double(_ value: Any?) -> Double {
    if let double = value as? Double { return double }
    if let int = value as? Int { return Double(int) }
    if let string = value as? String { return Double(string) ?? 0 }
    return 0
  }

  static func bool(_ value: Any?) -> Bool {
    if let bool = value as? Bool { return bool }
    if let int = value as? Int { return int != 0 }
    if let string = value as? String { return ["true", "1", "yes"].contains(string.lowercased()) }
    return false
  }
}
