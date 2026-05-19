// swift-tools-version: 6.0
import PackageDescription

let package = Package(
  name: "BaselineHotspots",
  platforms: [.macOS(.v14)],
  products: [
    .executable(name: "BaselineHotspots", targets: ["BaselineHotspots"])
  ],
  targets: [
    .executableTarget(
      name: "BaselineHotspots",
      linkerSettings: [.linkedFramework("Security")]
    )
  ]
)
