import Cocoa
import WebKit

class MainWindowController: NSWindowController, NSWindowDelegate, NSToolbarDelegate {
    
    private var webVC: WebViewController!
    private var searchField: NSSearchField!
    private var statusDotItem: NSToolbarItem!
    
    convenience init() {
        let window = NSWindow(
            contentRect: NSRect(x: 100, y: 100, width: 1280, height: 830),
            styleMask: [.titled, .closable, .miniaturizable, .resizable],
            backing: .buffered,
            defer: false
        )
        
        window.title = "Open Gerege Nexus"
        window.center()
        window.setFrameAutosaveName("GeregeNexusMainWindow")
        
        self.init(window: window)
        window.delegate = self
        
        setupContent()
        setupToolbar()
        monitorServerStatus()
    }
    
    private func setupContent() {
        guard let window = window else { return }
        webVC = WebViewController()
        window.contentViewController = webVC
    }
    
    private func setupToolbar() {
        guard let window = window else { return }
        
        let toolbar = NSToolbar(identifier: "GeregeMainToolbar")
        toolbar.allowsUserCustomization = true
        toolbar.autosavesConfiguration = true
        toolbar.displayMode = .iconOnly
        toolbar.delegate = self
        
        window.toolbar = toolbar
    }
    
    private func monitorServerStatus() {
        ServerManager.shared.startMonitoring { [weak self] apiOk, webOk in
            DispatchQueue.main.async {
                let bothOk = apiOk && webOk
                self?.statusDotItem?.toolTip = bothOk ? "Сервер хэвийн холбогдсон 🟢" : "Сервер холбогдоогүй 🔴"
                if #available(macOS 11.0, *) {
                    let iconName = bothOk ? "circle.fill" : "exclamationmark.circle.fill"
                    let config = NSImage.SymbolConfiguration(paletteColors: [bothOk ? .systemGreen : .systemRed])
                    self?.statusDotItem?.image = NSImage(systemSymbolName: iconName, accessibilityDescription: nil)?.withSymbolConfiguration(config)
                }
            }
        }
    }
    
    func reloadWebClient() {
        webVC.loadWebClient()
    }
    
    func navigateTo(path: String) {
        webVC.loadWebClient(path: path)
    }
    
    // MARK: - NSToolbarDelegate
    
    func toolbar(_ toolbar: NSToolbar, itemForItemIdentifier itemIdentifier: NSToolbarItem.Identifier, willBeInsertedIntoToolbar flag: Bool) -> NSToolbarItem? {
        
        switch itemIdentifier.rawValue {
        case "AppStoreItem":
            let item = NSToolbarItem(itemIdentifier: itemIdentifier)
            item.label = "Апп Стор"
            item.paletteLabel = "Апп Стор"
            item.toolTip = "Нэгдсэн Апп Стор"
            if #available(macOS 11.0, *) {
                item.image = NSImage(systemSymbolName: "square.grid.2x2.fill", accessibilityDescription: "Apps")
            }
            item.target = self
            item.action = #selector(openAppStore)
            return item
            
        case "ESignItem":
            let item = NSToolbarItem(itemIdentifier: itemIdentifier)
            item.label = "PDF E-Sign"
            item.paletteLabel = "PDF E-Sign"
            item.toolTip = "Тоон гарын үсгээр баталгаажуулах"
            if #available(macOS 11.0, *) {
                item.image = NSImage(systemSymbolName: "doc.badge.gearshape.fill", accessibilityDescription: "Sign")
            }
            item.target = self
            item.action = #selector(openESign)
            return item
            
        case "GovServicesItem":
            let item = NSToolbarItem(itemIdentifier: itemIdentifier)
            item.label = "Төрийн үйлчилгээ"
            item.paletteLabel = "Төрийн үйлчилгээ"
            item.toolTip = "Төрийн үйлчилгээний систем"
            if #available(macOS 11.0, *) {
                item.image = NSImage(systemSymbolName: "building.2.fill", accessibilityDescription: "Gov")
            }
            item.target = self
            item.action = #selector(openGovServices)
            return item
            
        case "SearchItem":
            let item = NSSearchToolbarItem(itemIdentifier: itemIdentifier)
            item.label = "Хайлт"
            searchField = item.searchField
            searchField.placeholderString = "Апп, цэс, үйлчилгээ хайх... (⌘F)"
            searchField.target = self
            searchField.action = #selector(onSearchEntered)
            return item
            
        case "StatusDotItem":
            let item = NSToolbarItem(itemIdentifier: itemIdentifier)
            item.label = "Төлөв"
            item.paletteLabel = "Серверийн төлөв"
            item.toolTip = "Серверийн холболт шалгаж байна..."
            if #available(macOS 11.0, *) {
                item.image = NSImage(systemSymbolName: "circle.fill", accessibilityDescription: "Status")
            }
            item.target = self
            item.action = #selector(openPreferences)
            statusDotItem = item
            return item
            
        case "ReloadItem":
            let item = NSToolbarItem(itemIdentifier: itemIdentifier)
            item.label = "Шинэчлэх"
            item.paletteLabel = "Шинэчлэх"
            item.toolTip = "Аппликейшн дахин ачаалах (⌘R)"
            if #available(macOS 11.0, *) {
                item.image = NSImage(systemSymbolName: "arrow.clockwise", accessibilityDescription: "Reload")
            }
            item.target = self
            item.action = #selector(reloadClicked)
            return item
            
        case "SettingsItem":
            let item = NSToolbarItem(itemIdentifier: itemIdentifier)
            item.label = "Тохиргоо"
            item.paletteLabel = "Тохиргоо"
            item.toolTip = "Платформын тохиргоо (⌘,)"
            if #available(macOS 11.0, *) {
                item.image = NSImage(systemSymbolName: "gearshape.fill", accessibilityDescription: "Settings")
            }
            item.target = self
            item.action = #selector(openPreferences)
            return item
            
        default:
            return nil
        }
    }
    
    func toolbarDefaultItemIdentifiers(_ toolbar: NSToolbar) -> [NSToolbarItem.Identifier] {
        return [
            NSToolbarItem.Identifier("AppStoreItem"),
            NSToolbarItem.Identifier("ESignItem"),
            NSToolbarItem.Identifier("GovServicesItem"),
            .sidebarTrackingSeparator,
            .flexibleSpace,
            NSToolbarItem.Identifier("SearchItem"),
            .flexibleSpace,
            NSToolbarItem.Identifier("StatusDotItem"),
            NSToolbarItem.Identifier("ReloadItem"),
            NSToolbarItem.Identifier("SettingsItem")
        ]
    }
    
    func toolbarAllowedItemIdentifiers(_ toolbar: NSToolbar) -> [NSToolbarItem.Identifier] {
        return toolbarDefaultItemIdentifiers(toolbar)
    }
    
    // MARK: - Actions
    
    @objc func openAppStore() { navigateTo(path: "/apps") }
    @objc func openESign() { navigateTo(path: "/esign") }
    @objc func openGovServices() { navigateTo(path: "/gov-services") }
    @objc func reloadClicked() { webVC.webView.reload() }
    @objc func openPreferences() { PreferencesWindowController.shared.showWindow(nil) }
    
    @objc func onSearchEntered() {
        let query = searchField.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
        if !query.isEmpty {
            let js = "if(window.GeregeDesktop) { window.location.href = '/apps?q=' + encodeURIComponent('\(query)'); }"
            webVC.webView.evaluateJavaScript(js, completionHandler: nil)
        }
    }
}
