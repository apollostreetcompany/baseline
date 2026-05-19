import SwiftUI

struct ContentView: View {
  @EnvironmentObject private var store: AppStore
  @State private var selection: String?

  var body: some View {
    NavigationSplitView {
      SidebarView(selection: $selection)
    } detail: {
      DashboardDetailView(selection: selection)
    }
    .navigationTitle("Baseline Hotspots")
    .toolbar {
      ToolbarItemGroup {
        Button {
          Task { await store.refresh() }
        } label: {
          Label("Refresh", systemImage: "arrow.clockwise")
        }
        .disabled(store.isLoading || !store.isSignedIn)

        Button {
          Task { await store.generateInsight() }
        } label: {
          Label("Analyze", systemImage: "sparkles")
        }
        .disabled(store.isLoading || store.runs.isEmpty)
      }
    }
  }
}

struct DashboardDetailView: View {
  @EnvironmentObject private var store: AppStore
  var selection: String?

  var selectedHotspot: Hotspot? {
    guard let selection else { return nil }
    return store.hotspots.first(where: { $0.id == selection })
  }

  var body: some View {
    ScrollView {
      VStack(alignment: .leading, spacing: 20) {
        ConnectionHeader()

        if !store.isSignedIn {
          SignInPanel()
        } else {
          OverviewGrid()
          HotspotDetail(hotspot: selectedHotspot ?? store.hotspots.first)
          TimelineSection()
          InsightPanel()
        }
      }
      .padding(24)
      .frame(maxWidth: .infinity, alignment: .leading)
    }
    .background(Color(nsColor: .windowBackgroundColor))
  }
}

struct ConnectionHeader: View {
  @EnvironmentObject private var store: AppStore

  var body: some View {
    HStack(alignment: .top) {
      VStack(alignment: .leading, spacing: 6) {
        Text("Cloud MCP")
          .font(.caption.weight(.semibold))
          .foregroundStyle(.secondary)
        Text(store.statusMessage)
          .font(.title2.weight(.semibold))
        Text(store.endpointText)
          .font(.callout.monospaced())
          .foregroundStyle(.secondary)
      }
      Spacer()
      if store.isLoading {
        ProgressView()
          .controlSize(.small)
      }
    }
  }
}

struct SignInPanel: View {
  @EnvironmentObject private var store: AppStore

  var body: some View {
    VStack(alignment: .leading, spacing: 14) {
      Text("Sign In")
        .font(.title3.weight(.semibold))
      TextField("Email", text: $store.signInEmail)
        .textFieldStyle(.roundedBorder)
      HStack {
        Button("Request Magic Link") {
          Task { await store.requestMagicLink() }
        }
        .buttonStyle(.borderedProminent)
        Spacer()
      }
      TextField("Paste magic link or token", text: $store.pastedMagicLink, axis: .vertical)
        .lineLimit(3...5)
        .textFieldStyle(.roundedBorder)
      Button("Consume Link") {
        Task { await store.consumeMagicLink() }
      }
      .buttonStyle(.bordered)
    }
    .padding(18)
    .background(.regularMaterial)
    .clipShape(RoundedRectangle(cornerRadius: 8))
  }
}

struct OverviewGrid: View {
  @EnvironmentObject private var store: AppStore

  var body: some View {
    Grid(alignment: .leading, horizontalSpacing: 14, verticalSpacing: 14) {
      GridRow {
        MetricTile(title: "Plan", value: store.account.planKey.uppercased(), detail: store.account.entitlementStatus)
        MetricTile(title: "Health", value: "\(store.runs.first?.healthScore ?? 0)", detail: store.runs.first?.status ?? "no run")
        MetricTile(title: "Warnings", value: "\(store.runs.first?.warningCount ?? 0)", detail: "\(store.hotspots.count) hotspots")
        MetricTile(title: "Delta", value: BaselineFormat.signed(store.comparison?.healthDelta ?? 0), detail: "self history")
      }
    }
  }
}

struct MetricTile: View {
  var title: String
  var value: String
  var detail: String

  var body: some View {
    VStack(alignment: .leading, spacing: 8) {
      Text(title)
        .font(.caption.weight(.semibold))
        .foregroundStyle(.secondary)
      Text(value)
        .font(.system(size: 28, weight: .bold, design: .rounded))
      Text(detail)
        .font(.callout)
        .foregroundStyle(.secondary)
    }
    .frame(minWidth: 130, maxWidth: .infinity, minHeight: 96, alignment: .leading)
    .padding(14)
    .background(Color(nsColor: .controlBackgroundColor))
    .clipShape(RoundedRectangle(cornerRadius: 8))
  }
}

struct HotspotDetail: View {
  var hotspot: Hotspot?

  var body: some View {
    VStack(alignment: .leading, spacing: 12) {
      Text("Hotspot Detail")
        .font(.title3.weight(.semibold))
      if let hotspot {
        Grid(alignment: .leading, horizontalSpacing: 20, verticalSpacing: 8) {
          GridRow { Text("Check").foregroundStyle(.secondary); Text(hotspot.checkID).font(.body.monospaced()) }
          GridRow { Text("Warnings").foregroundStyle(.secondary); Text("\(hotspot.warningCount) across \(hotspot.runCount) runs") }
          GridRow { Text("Latest").foregroundStyle(.secondary); Text("\(hotspot.latestStatus) in \(BaselineFormat.shortRun(hotspot.latestRunID))") }
          GridRow { Text("Slowest").foregroundStyle(.secondary); Text(BaselineFormat.ms(hotspot.maxDurationMS)) }
          GridRow { Text("Average").foregroundStyle(.secondary); Text(String(format: "%.0f", hotspot.averageScore)) }
        }
      } else {
        Text("No hotspots in the selected history window.")
          .foregroundStyle(.secondary)
      }
    }
    .padding(18)
    .background(Color(nsColor: .controlBackgroundColor))
    .clipShape(RoundedRectangle(cornerRadius: 8))
  }
}

struct TimelineSection: View {
  @EnvironmentObject private var store: AppStore

  var body: some View {
    VStack(alignment: .leading, spacing: 12) {
      Text("Run Timeline")
        .font(.title3.weight(.semibold))
      ForEach(store.runs.prefix(12)) { run in
        HStack {
          Text(BaselineFormat.shortRun(run.runID))
            .font(.body.monospaced())
            .frame(width: 120, alignment: .leading)
          Text("\(run.healthScore)")
            .font(.headline)
            .frame(width: 52, alignment: .leading)
          Text(run.status)
            .frame(width: 90, alignment: .leading)
          Text("\(run.warningCount) warnings")
            .foregroundStyle(.secondary)
          Spacer()
          Text(BaselineFormat.ms(run.durationMS))
            .foregroundStyle(.secondary)
        }
        Divider()
      }
      if store.runs.isEmpty {
        Text("No synced runs yet.")
          .foregroundStyle(.secondary)
      }
    }
  }
}

struct InsightPanel: View {
  @EnvironmentObject private var store: AppStore

  var body: some View {
    VStack(alignment: .leading, spacing: 12) {
      HStack {
        Text("LLM Insight")
          .font(.title3.weight(.semibold))
        Spacer()
        Button("Analyze") {
          Task { await store.generateInsight() }
        }
        .disabled(store.isLoading || store.runs.isEmpty)
      }
      Text(store.insight.isEmpty ? "Run analysis to ask the local agent provider first, then OpenRouter fallback if configured." : store.insight)
        .font(.body)
        .textSelection(.enabled)
        .foregroundStyle(store.insight.isEmpty ? .secondary : .primary)
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(14)
        .background(Color(nsColor: .textBackgroundColor))
        .clipShape(RoundedRectangle(cornerRadius: 8))
    }
  }
}
