import Cocoa
import WebKit

class PreferencesWindowController: NSWindowController {
    static let shared = PreferencesWindowController()
    
    private var apiUrlField: NSTextField!
    private var webUrlField: NSTextField!
    private var touchIdSwitch: NSButton!
    private var statusLabel: NSTextField!
    
    convenience init() {
        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 480, height: 320),
            styleMask: [.titled, .closable],
            backing: .buffered,
            defer: false
        )
        window.title = "Тохиргоо - Open Gerege Nexus"
        window.center()
        self.init(window: window)
        setupUI()
    }
    
    private func setupUI() {
        guard let window = window else { return }
        
        let container = NSView(frame: NSRect(x: 0, y: 0, width: 480, height: 320))
        
        let titleLabel = NSTextField(labelWithString: "Платформын тохиргоо")
        titleLabel.font = NSFont.boldSystemFont(ofSize: 15)
        titleLabel.frame = NSRect(x: 24, y: 270, width: 432, height: 24)
        container.addSubview(titleLabel)
        
        // API URL
        let apiLabel = NSTextField(labelWithString: "Backend API URL (Go Server):")
        apiLabel.frame = NSRect(x: 24, y: 226, width: 200, height: 20)
        container.addSubview(apiLabel)
        
        apiUrlField = NSTextField(frame: NSRect(x: 24, y: 200, width: 432, height: 24))
        apiUrlField.stringValue = ServerManager.shared.apiBaseURL
        container.addSubview(apiUrlField)
        
        // Web URL
        let webLabel = NSTextField(labelWithString: "Web Client URL (Next.js):")
        webLabel.frame = NSRect(x: 24, y: 166, width: 200, height: 20)
        container.addSubview(webLabel)
        
        webUrlField = NSTextField(frame: NSRect(x: 24, y: 140, width: 432, height: 24))
        webUrlField.stringValue = ServerManager.shared.webBaseURL
        container.addSubview(webUrlField)
        
        // Touch ID toggle
        touchIdSwitch = NSButton(checkboxWithTitle: "PDF E-Sign болон sensitive үйлдлүүдэд Touch ID ашиглах", target: nil, action: nil)
        touchIdSwitch.frame = NSRect(x: 24, y: 100, width: 432, height: 24)
        let touchIdEnabled = UserDefaults.standard.bool(forKey: "gerege_use_touchid")
        touchIdSwitch.state = touchIdEnabled ? .on : .off
        container.addSubview(touchIdSwitch)
        
        // Status indicator
        statusLabel = NSTextField(labelWithString: "Сүлжээний төлөв шалгаж байна...")
        statusLabel.font = NSFont.systemFont(ofSize: 12)
        statusLabel.textColor = NSColor.secondaryLabelColor
        statusLabel.frame = NSRect(x: 24, y: 64, width: 300, height: 20)
        container.addSubview(statusLabel)
        
        // Buttons
        let saveButton = NSButton(title: "Хадгалах", target: self, action: #selector(saveClicked))
        saveButton.frame = NSRect(x: 366, y: 16, width: 90, height: 32)
        saveButton.bezelStyle = .rounded
        saveButton.keyEquivalent = "\r"
        container.addSubview(saveButton)
        
        let cancelButton = NSButton(title: "Цуцлах", target: self, action: #selector(cancelClicked))
        cancelButton.frame = NSRect(x: 270, y: 16, width: 90, height: 32)
        cancelButton.bezelStyle = .rounded
        container.addSubview(cancelButton)
        
        window.contentView = container
        updateStatus()
    }
    
    private func updateStatus() {
        ServerManager.shared.checkHealth { [weak self] apiOk, webOk in
            let apiStr = apiOk ? "🟢 API бэлэн" : "🔴 API холбогдоогүй"
            let webStr = webOk ? "🟢 Web бэлэн" : "🔴 Web холбогдоогүй"
            self?.statusLabel.stringValue = "\(apiStr) | \(webStr)"
        }
    }
    
    @objc private func saveClicked() {
        let newApi = apiUrlField.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
        let newWeb = webUrlField.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
        
        if !newApi.isEmpty { ServerManager.shared.apiBaseURL = newApi }
        if !newWeb.isEmpty { ServerManager.shared.webBaseURL = newWeb }
        
        UserDefaults.standard.set(touchIdSwitch.state == .on, forKey: "gerege_use_touchid")
        
        window?.close()
        
        // Reload active webview in main window
        if let mainVC = NSApplication.shared.windows.first?.contentViewController as? MainWindowController {
            mainVC.reloadWebClient()
        }
    }
    
    @objc private func cancelClicked() {
        window?.close()
    }
}
