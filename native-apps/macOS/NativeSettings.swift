import Foundation

struct NativeSettings: Codable {
    var schemaVersion = 1
    var launchAtLogin = false
    var language = "mn"
    var webEndpoint = "http://localhost:3000"
    var apiEndpoint = "http://localhost:8080"
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
              let value = try? JSONDecoder().decode(Self.self, from: data) else { return Self() }
        return value
    }
    func save() {
        guard let data = try? JSONEncoder().encode(self) else { return }
        UserDefaults.standard.set(data, forKey: Self.storageKey)
    }
}
