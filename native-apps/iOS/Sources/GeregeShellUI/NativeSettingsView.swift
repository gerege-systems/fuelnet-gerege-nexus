#if os(iOS)
import GeregeShellKit
import SwiftUI
import UIKit
import WebKit

public struct NativeSettingsView: View {
    private enum SettingsPane: String, CaseIterable, Identifiable {
        case general = "Ерөнхий", connection = "Холболт", printer = "Принтер", scanner = "Сканнер"
        case serial = "Serial порт", privacy = "Нууцлал", device = "Төхөөрөмж", lockdown = "Lockdown"
        case drawer = "Cash drawer", update = "Шинэчлэлт", diagnostics = "Оношлогоо"
        var id: String { rawValue }
        var symbol: String { switch self {
        case .general: "gearshape"; case .connection: "network"; case .printer: "printer"
        case .scanner: "barcode.viewfinder"; case .serial: "cable.connector"; case .privacy: "lock.shield"
        case .device: "display"; case .lockdown: "lock.rectangle"; case .drawer: "tray"
        case .update: "arrow.triangle.2.circlepath"; case .diagnostics: "stethoscope" } }
    }

    public let formFactor: String
    public let onClose: () -> Void
    @State private var selection: SettingsPane? = .general
    @State private var status = ""
    @AppStorage("native.settings.schemaVersion") private var schemaVersion = 1
    @AppStorage("native.settings.webEndpoint") private var webEndpoint = "https://nexus.gerege.mn"
    @AppStorage("native.settings.apiEndpoint") private var apiEndpoint = "https://nexus.gerege.mn"
    @AppStorage("native.settings.printerTransport") private var printerTransport = "AirPrint"
    @AppStorage("native.settings.printerHost") private var printerHost = ""
    @AppStorage("native.settings.printerPort") private var printerPort = "9100"
    @AppStorage("native.settings.paperWidth") private var paperWidth = "80 mm"
    @AppStorage("native.settings.scannerMode") private var scannerMode = "Camera"
    @AppStorage("native.settings.serialPort") private var serialPort = ""
    @AppStorage("native.settings.baudRate") private var baudRate = "9600"
    @AppStorage("native.settings.biometric") private var biometric = true
    @AppStorage("native.settings.idleMinutes") private var idleMinutes = "5"
    @AppStorage("native.settings.deviceName") private var deviceName = UIDevice.current.name
    @AppStorage("native.settings.site") private var site = ""
    @AppStorage("native.settings.lockdown") private var lockdown = false
    @AppStorage("native.settings.reboot") private var reboot = "03:00"
    @AppStorage("native.settings.drawerPulse") private var drawerPulse = "120"
    @AppStorage("native.settings.updateChannel") private var updateChannel = "Stable"
    @AppStorage("native.settings.telemetry") private var telemetry = true
    @AppStorage("native.settings.deviceID") private var deviceID = ""
    @State private var enrollmentCode = ""
    @State private var enrollmentState = "Бүртгэгдээгүй"
    private let deviceClient = DeviceEnrollmentClient()
    private let tokenStore = DeviceTokenStore()

    public init(formFactor: String, onClose: @escaping () -> Void) { self.formFactor = formFactor; self.onClose = onClose }

    public var body: some View {
        NavigationSplitView {
            List(visibleSections, selection: $selection) { section in
                Label(section.rawValue, systemImage: section.symbol).tag(section)
            }
            .navigationTitle("Төхөөрөмж")
            .safeAreaInset(edge: .bottom) { Button("Ажлын хэсэг рүү буцах", action: onClose).buttonStyle(.bordered).padding() }
        } detail: {
            if let selection { detail(selection).navigationTitle(selection.rawValue) }
            else { VStack(spacing: 12) { Image(systemName: "gearshape.2").font(.largeTitle); Text("Ангилал сонгоно уу") }.foregroundStyle(.secondary) }
        }
        .tint(Color(red: 34/255, green: 163/255, blue: 165/255))
        .task { await refreshDeviceIdentity() }
    }

    private var visibleSections: [SettingsPane] { SettingsPane.allCases.filter {
        switch $0 {
        case .device: formFactor == "kiosk" || formFactor == "pos"
        case .lockdown: formFactor == "kiosk"
        case .drawer: formFactor == "pos"
        default: true
        }
    } }

    @ViewBuilder private func detail(_ section: SettingsPane) -> some View {
        Form {
            Section("DEVICE CONSOLE") {
                switch section {
                case .general:
                    LabeledContent("Form factor", value: formFactor); LabeledContent("Хэл", value: "mn")
                case .connection:
                    TextField("Web endpoint", text: $webEndpoint).textInputAutocapitalization(.never)
                    TextField("API endpoint", text: $apiEndpoint).textInputAutocapitalization(.never)
                    test("Холболт шалгах", "Web/API холболтыг шалгаж байна…")
                case .printer:
                    Picker("Холболт", selection: $printerTransport) { ForEach(["AirPrint", "Network", "Bluetooth", "Vendor SDK"], id: \.self) { Text($0) } }
                    TextField("IP / host", text: $printerHost); TextField("TCP port", text: $printerPort).keyboardType(.numberPad)
                    Picker("Цаас", selection: $paperWidth) { Text("58 mm").tag("58 mm"); Text("80 mm").tag("80 mm") }
                    test("Туршилтын баримт хэвлэх", printerHost.isEmpty && printerTransport == "Network" ? "Printer host оруулна уу" : "Хэвлэх хүсэлт бэлэн боллоо")
                case .scanner:
                    Picker("Унших горим", selection: $scannerMode) { ForEach(["Camera", "Bluetooth HID", "Vendor SDK"], id: \.self) { Text($0) } }
                    test("Туршилтын код унших", "Камер/сканнерын input хүлээж байна…")
                case .serial:
                    TextField("Accessory / port", text: $serialPort); Picker("Baud", selection: $baudRate) { ForEach(["9600", "19200", "38400", "57600", "115200"], id: \.self) { Text($0) } }
                    test("Accessory шалгах", serialPort.isEmpty ? "Accessory сонгоогүй" : "\(serialPort)-ыг шалгаж байна…")
                case .privacy:
                    Toggle("Face ID / Touch ID түгжээ", isOn: $biometric); TextField("Идэвхгүй үед түгжих (минут)", text: $idleMinutes).keyboardType(.numberPad)
                    test("Keychain session цэвэрлэх…", "Биометрик баталгаажуулалт шаардлагатай")
                case .device:
                    TextField("Төхөөрөмжийн нэр", text: $deviceName); TextField("Байршил / site", text: $site)
                    TextField("Enrollment code", text: $enrollmentCode).textInputAutocapitalization(.characters).autocorrectionDisabled()
                    LabeledContent("Device ID", value: deviceID.isEmpty ? "—" : deviceID); LabeledContent("Enrollment", value: enrollmentState)
                    Button("Нэг удаагийн кодоор бүртгэх") { Task { await enrollDevice() } }.disabled(enrollmentCode.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                case .lockdown:
                    Toggle("Single App Mode", isOn: $lockdown); TextField("Өдөр тутмын reboot", text: $reboot)
                    test("Managed mode шалгах", "MDM supervision шаардлагатай")
                case .drawer:
                    TextField("Printer pulse (ms)", text: $drawerPulse).keyboardType(.numberPad); test("Шургуулга нээх", "ESC/POS drawer pulse: \(drawerPulse) ms")
                case .update:
                    Picker("Суваг", selection: $updateChannel) { ForEach(["Stable", "Pilot", "Internal"], id: \.self) { Text($0) } }; Toggle("Оношлогооны мэдээлэл", isOn: $telemetry)
                    test("Шинэчлэлт шалгах", "\(updateChannel) сувгийг шалгаж байна…")
                case .diagnostics:
                    LabeledContent("iOS/iPadOS", value: UIDevice.current.systemVersion); LabeledContent("Shell contract", value: "1.3")
                    LabeledContent("WebKit", value: "WKWebView"); test("Лог export хийх…", "Оношлогооны snapshot бэлэн боллоо")
                }
            }
            if !status.isEmpty { Section("Төлөв") { Text(status).foregroundStyle(.secondary).monospacedDigit() } }
            Section { Button("Хадгалах") { schemaVersion = 1; status = "Хадгаллаа" }.frame(maxWidth: .infinity) }
        }
    }

    private func test(_ title: String, _ message: String) -> some View { Button(title) { status = message } }

    @MainActor private func refreshDeviceIdentity() async {
        guard formFactor == "kiosk" || formFactor == "pos", let token = try? tokenStore.load() else { return }
        do { let identity = try await deviceClient.identity(apiEndpoint: apiEndpoint, token: token); deviceID = identity.id; deviceName = identity.name; enrollmentState = identity.status }
        catch { enrollmentState = "Token хүчингүй эсвэл сервер холбогдохгүй байна" }
    }

    @MainActor private func enrollDevice() async {
        status = "Төхөөрөмжийг бүртгэж байна…"
        do {
            let enrolled = try await deviceClient.enroll(apiEndpoint: apiEndpoint, code: enrollmentCode, name: deviceName, platform: "ios", formFactor: formFactor, site: site, appVersion: Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "dev", osVersion: UIDevice.current.systemVersion)
            try tokenStore.save(enrolled.deviceToken); deviceID = enrolled.deviceID; enrollmentCode = ""; enrollmentState = "ACTIVE"; status = "Төхөөрөмж амжилттай бүртгэгдлээ"
        } catch { status = error.localizedDescription }
    }
}

public struct NativeShellView: View {
    public let cookies: [SessionCookie]
    public let formFactor: String
    public let onRelogin: @MainActor () -> Void
    @State private var settings = false
    @State private var route = "/apps"
    public init(cookies: [SessionCookie], formFactor: String, onRelogin: @escaping @MainActor () -> Void) { self.cookies = cookies; self.formFactor = formFactor; self.onRelogin = onRelogin }
    public var body: some View {
        if settings { NativeSettingsView(formFactor: formFactor) { settings = false } }
        else { ShellWebView(cookies: cookies, formFactor: formFactor, route: route, onRelogin: onRelogin)
            .safeAreaInset(edge: .bottom) {
                HStack(spacing: 4) {
                    tab("square.grid.2x2", "Аппууд", "/apps")
                    tab("doc.text", "Баримт", "/documents")
                    tab("person.2", "Харилцагч", "/contacts")
                    Button { settings = true } label: { Label("Тохиргоо", systemImage: "gearshape").frame(maxWidth: .infinity) }
                }
                .labelStyle(.iconOnly).font(.title3).padding(.horizontal, 8).frame(height: 54).background(.bar)
            }
        }
    }

    private func tab(_ icon: String, _ title: String, _ path: String) -> some View {
        Button { route = path } label: {
            Label(title, systemImage: icon).frame(maxWidth: .infinity)
                .foregroundStyle(route == path ? Color.accentColor : Color.secondary)
        }
        .accessibilityLabel(title)
    }
}
#endif
