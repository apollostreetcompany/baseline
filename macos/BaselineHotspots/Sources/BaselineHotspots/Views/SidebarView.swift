import SwiftUI

struct SidebarView: View {
  @EnvironmentObject private var store: AppStore
  @Binding var selection: String?

  var body: some View {
    List(selection: $selection) {
      Section("Account") {
        LabeledContent("Status", value: store.account.status)
        LabeledContent("Plan", value: store.account.planKey)
        LabeledContent("Monitoring", value: store.account.monitoringEnabled ? "enabled" : "off")
      }

      Section("Hotspots") {
        ForEach(store.hotspots) { hotspot in
          VStack(alignment: .leading, spacing: 4) {
            Text(hotspot.checkID)
              .font(.callout.weight(.semibold))
              .lineLimit(1)
            Text("\(hotspot.warningCount) warnings")
              .font(.caption)
              .foregroundStyle(.secondary)
          }
          .tag(hotspot.id)
        }
        if store.hotspots.isEmpty {
          Text("No hotspots")
            .foregroundStyle(.secondary)
        }
      }

      Section("Next Actions") {
        ForEach(store.nextActions, id: \.self) { action in
          Text(action)
            .font(.caption)
            .foregroundStyle(.secondary)
        }
      }
    }
    .navigationSplitViewColumnWidth(min: 260, ideal: 300, max: 360)
  }
}
