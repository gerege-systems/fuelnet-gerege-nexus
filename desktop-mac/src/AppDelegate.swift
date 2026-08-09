import Cocoa
import WebKit
import UniformTypeIdentifiers

class AppDelegate: NSObject, NSApplicationDelegate {
    
    var mainWindowController: MainWindowController?
    var statusItem: NSStatusItem?
    
    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.mainMenu = MenuBuilder.setupMainMenu(appDelegate: self)
        
        setupStatusItem()
        
        mainWindowController = MainWindowController()
        mainWindowController?.showWindow(nil)
        
        NSApp.activate(ignoringOtherApps: true)
        
        // Register URL scheme handler for gerege://
        NSAppleEventManager.shared().setEventHandler(
            self,
            andSelector: #selector(handleURLEvent(_:withReplyEvent:)),
            forEventClass: AEEventClass(kInternetEventClass),
            andEventID: AEEventID(kAEGetURL)
        )
    }
    
    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        return false // Keep running in menu bar
    }
    
    func applicationShouldHandleReopen(_ sender: NSApplication, hasVisibleWindows flag: Bool) -> Bool {
        if !flag {
            mainWindowController?.showWindow(nil)
        }
        return true
    }
    
    // MARK: - Menu Bar Status Item (Tray)
    
    private func setupStatusItem() {
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        if let button = statusItem?.button {
            if #available(macOS 11.0, *) {
                button.image = NSImage(systemSymbolName: "building.columns.circle.fill", accessibilityDescription: "Gerege Nexus")
            } else {
                button.title = "Gerege"
            }
        }
        
        let menu = NSMenu()
        menu.addItem(withTitle: "Open Gerege Nexus (Mac Desktop)", action: #selector(showMainWindow), keyEquivalent: "")
        menu.addItem(NSMenuItem.separator())
        
        let appsItem = NSMenuItem(title: "Түргэн шилжих", action: nil, keyEquivalent: "")
        let appsSub = NSMenu()
        appsSub.addItem(withTitle: "Апп Стор", action: #selector(openAppStoreFromTray), keyEquivalent: "")
        appsSub.addItem(withTitle: "PDF E-Sign", action: #selector(openESignFromTray), keyEquivalent: "")
        appsSub.addItem(withTitle: "Нэхэмжлэх", action: #selector(openBillingFromTray), keyEquivalent: "")
        appsItem.submenu = appsSub
        menu.addItem(appsItem)
        
        menu.addItem(NSMenuItem.separator())
        menu.addItem(withTitle: "Тохиргоо (⌘,)", action: #selector(openPreferences), keyEquivalent: "")
        menu.addItem(withTitle: "Гарах", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "")
        
        statusItem?.menu = menu
    }
    
    // MARK: - URL Scheme Handler
    
    @objc func handleURLEvent(_ event: NSAppleEventDescriptor, withReplyEvent replyEvent: NSAppleEventDescriptor) {
        guard let urlString = event.paramDescriptor(forKeyword: keyDirectObject)?.stringValue,
              let url = URL(string: urlString) else { return }
        
        // e.g. gerege://apps/esign -> route to /esign
        let path = url.host != nil ? "/\(url.host!)\(url.path)" : url.path
        showMainWindow()
        mainWindowController?.navigateTo(path: path)
    }
    
    // MARK: - Menu Actions
    
    @objc func showMainWindow() {
        mainWindowController?.showWindow(nil)
        NSApp.activate(ignoringOtherApps: true)
    }
    
    @objc func showAbout() {
        let options: [NSApplication.AboutPanelOptionKey: Any] = [
            .credits: NSAttributedString(string: "Gerege Nexus нь төрийн ба хувийн хэвшлийн үйлчилгээ, ажлын урсгалыг нэгтгэх модульт платформ юм.\nGo 1.25 + Next.js 15 + Swift Native Mac Engine."),
            .applicationVersion: "1.0.0",
            .version: "2026.08"
        ]
        NSApp.orderFrontStandardAboutPanel(options: options)
    }
    
    @objc func checkUpdates() {
        let alert = NSAlert()
        alert.messageText = "Хамгийн сүүлийн үеийн хувилбар"
        alert.informativeText = "Та Open Gerege Nexus Mac v1.0.0 хамгийн сүүлийн үеийн хувилбарыг ашиглаж байна."
        alert.alertStyle = .informational
        alert.addButton(withTitle: "ОК")
        alert.runModal()
    }
    
    @objc func openPreferences() {
        PreferencesWindowController.shared.showWindow(nil)
        NSApp.activate(ignoringOtherApps: true)
    }
    
    @objc func newDocument() {
        showMainWindow()
        mainWindowController?.navigateTo(path: "/documents")
    }
    
    @objc func openPDFForSigning() {
        let panel = NSOpenPanel()
        if #available(macOS 11.0, *) {
            panel.allowedContentTypes = [.pdf]
        } else {
            panel.allowedFileTypes = ["pdf"]
        }
        panel.canChooseFiles = true
        panel.canChooseDirectories = false
        panel.allowsMultipleSelection = false
        panel.message = "Тоон гарын үсгээр баталгаажуулах PDF файлаа сонгоно уу"
        
        panel.begin { [weak self] result in
            if result == .OK, let url = panel.url {
                self?.showMainWindow()
                self?.mainWindowController?.navigateTo(path: "/esign?file=\(url.path.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? "")")
            }
        }
    }
    
    @objc func printDocument() {
        if let webVC = mainWindowController?.contentViewController as? WebViewController {
            let printOp = webVC.webView.printOperation(with: NSPrintInfo.shared)
            printOp.runModal(for: mainWindowController!.window!, delegate: nil, didRun: nil, contextInfo: nil)
        }
    }
    
    @objc func focusSearch() {
        showMainWindow()
        mainWindowController?.onSearchEntered()
    }
    
    @objc func reloadPage() {
        if let webVC = mainWindowController?.contentViewController as? WebViewController {
            webVC.webView.reload()
        }
    }
    
    @objc func forceReloadPage() {
        if let webVC = mainWindowController?.contentViewController as? WebViewController {
            webVC.webView.reloadFromOrigin()
        }
    }
    
    @objc func zoomIn() {
        if let webVC = mainWindowController?.contentViewController as? WebViewController {
            webVC.webView.pageZoom += 0.1
        }
    }
    
    @objc func zoomOut() {
        if let webVC = mainWindowController?.contentViewController as? WebViewController {
            webVC.webView.pageZoom = max(0.5, webVC.webView.pageZoom - 0.1)
        }
    }
    
    @objc func zoomReset() {
        if let webVC = mainWindowController?.contentViewController as? WebViewController {
            webVC.webView.pageZoom = 1.0
        }
    }
    
    @objc func toggleDevTools() {
        if let webVC = mainWindowController?.contentViewController as? WebViewController {
            webVC.webView.evaluateJavaScript("console.log('DevTools toggled');", completionHandler: nil)
        }
    }
    
    @objc func navigateToApp(_ sender: NSMenuItem) {
        if let path = sender.representedObject as? String {
            showMainWindow()
            mainWindowController?.navigateTo(path: path)
        }
    }
    
    @objc func openAppStoreFromTray() { showMainWindow(); mainWindowController?.navigateTo(path: "/apps") }
    @objc func openESignFromTray() { showMainWindow(); mainWindowController?.navigateTo(path: "/esign") }
    @objc func openBillingFromTray() { showMainWindow(); mainWindowController?.navigateTo(path: "/billing") }
    
    @objc func openDocumentation() {
        if let url = URL(string: "https://github.com/gerege-systems/open-gerege-nexus") {
            NSWorkspace.shared.open(url)
        }
    }
    
    @objc func openSupport() {
        if let url = URL(string: "https://developer.gerege.mn") {
            NSWorkspace.shared.open(url)
        }
    }
}
