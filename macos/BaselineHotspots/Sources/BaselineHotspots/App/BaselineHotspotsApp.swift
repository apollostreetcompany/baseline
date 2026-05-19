import SwiftUI

@main
struct BaselineHotspotsApp: App {
  @StateObject private var store = AppStore()

  var body: some Scene {
    WindowGroup {
      ContentView()
        .environmentObject(store)
        .task {
          await store.loadFromKeychain()
        }
    }
    .commands {
      CommandGroup(after: .appInfo) {
        Button("Refresh Baseline") {
          Task { await store.refresh() }
        }
        .keyboardShortcut("r", modifiers: [.command])
      }
    }

    Settings {
      SettingsView()
        .environmentObject(store)
        .frame(width: 560)
    }
  }
}
