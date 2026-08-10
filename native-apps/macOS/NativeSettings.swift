import Foundation

struct NativeSettings: Codable {
    var schemaVersion = 2
    var launchAtLogin = false
    var language = "mn"
    var webEndpoint = "https://nexus.gerege.mn"
    var apiEndpoint = "https://nexus.gerege.mn"
    var printerTransport = "USB"
    var printerHost = ""
    var printerPort = 9100
    var serialPort = ""
    var baudRate = 9600
    var paperWidth = "80 mm"
    var scannerMode = "Keyboard wedge"
    var scannerSuffix = "Enter"
    var biometricLock = true
    var idleLockMinutes = 5
    var updateChannel = "Stable"
    var telemetry = true
    var deviceName = Host.current().localizedName ?? "Mac"
    var site = ""
    var deviceID = ""

    static let storageKey = "mn.gerege.nexus.native-settings.v1"
    static func load() -> NativeSettings {
        guard let data = UserDefaults.standard.data(forKey: storageKey),
              var value = try? JSONDecoder().decode(Self.self, from: data) else { return Self() }
        if value.schemaVersion < 2 {
            if value.webEndpoint == "http://localhost:3000" { value.webEndpoint = "https://nexus.gerege.mn" }
            if value.apiEndpoint == "http://localhost:8080" { value.apiEndpoint = "https://nexus.gerege.mn" }
            value.schemaVersion = 2
            value.save()
        }
        return value
    }
    func save() {
        guard let data = try? JSONEncoder().encode(self) else { return }
        UserDefaults.standard.set(data, forKey: Self.storageKey)
    }
}
