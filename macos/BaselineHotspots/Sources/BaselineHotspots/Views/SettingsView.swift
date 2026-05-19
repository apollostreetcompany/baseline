import SwiftUI

struct SettingsView: View {
  @EnvironmentObject private var store: AppStore

  var body: some View {
    Form {
      Section("Baseline Cloud") {
        TextField("Worker URL", text: $store.endpointText)
        SecureField("Session token", text: $store.sessionToken)
        HStack {
          Button("Save") {
            store.saveSettings()
          }
          Button("Sign Out") {
            store.signOut()
          }
        }
      }

      Section("OpenRouter Fallback") {
        SecureField("API key", text: $store.openRouterKey)
        TextField("Model", text: $store.openRouterModel)
        Button("Save OpenRouter Settings") {
          store.saveSettings()
        }
      }
    }
    .formStyle(.grouped)
    .padding(18)
  }
}
