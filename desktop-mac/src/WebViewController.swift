import Cocoa
import WebKit
import UserNotifications

class WebViewController: NSViewController, WKNavigationDelegate, WKUIDelegate, WKScriptMessageHandler, WKDownloadDelegate {
    
    var webView: WKWebView!
    private var progressView: NSProgressIndicator!
    private var offlineBanner: NSTextField!
    private var nativeOfflineView: NSView!
    private var offlineTitleLabel: NSTextField!
    private var offlineDescLabel: NSTextField!
    
    override func loadView() {
        let container = NSView(frame: NSRect(x: 0, y: 0, width: 1280, height: 800))
        container.autoresizingMask = [.width, .height]
        
        let config = WKWebViewConfiguration()
        let userContentController = WKUserContentController()
        
        // Register JS Handlers
        userContentController.add(self, name: "desktopBridge")
        
        // Inject GeregeDesktop global object
        let scriptSource = """
        window.GeregeDesktop = {
            isNativeMac: true,
            version: '1.0.0',
            authenticateBiometric: function(reason, callbackId) {
                window.webkit.messageHandlers.desktopBridge.postMessage({
                    action: 'authenticateBiometric',
                    reason: reason || 'Баталгаажуулалт шаардлагатай',
                    callbackId: callbackId
                });
            },
            sendNotification: function(title, body) {
                window.webkit.messageHandlers.desktopBridge.postMessage({
                    action: 'sendNotification',
                    title: title,
                    body: body
                });
            },
            setBadgeCount: function(count) {
                window.webkit.messageHandlers.desktopBridge.postMessage({
                    action: 'setBadgeCount',
                    count: count
                });
            },
            openExternal: function(url) {
                window.webkit.messageHandlers.desktopBridge.postMessage({
                    action: 'openExternal',
                    url: url
                });
            }
        };
        console.log('[Gerege Nexus Desktop] Native Mac bridge initialized.');
        """
        let userScript = WKUserScript(source: scriptSource, injectionTime: .atDocumentStart, forMainFrameOnly: false)
        userContentController.addUserScript(userScript)
        
        config.userContentController = userContentController
        config.preferences.setValue(true, forKey: "developerExtrasEnabled")
        
        webView = WKWebView(frame: container.bounds, configuration: config)
        webView.translatesAutoresizingMaskIntoConstraints = false
        webView.navigationDelegate = self
        webView.uiDelegate = self
        webView.customUserAgent = "OpenGeregeNexusNative/1.0 (macOS; NativeDesktopApp)"
        container.addSubview(webView)
        
        // Top progress indicator
        progressView = NSProgressIndicator()
        progressView.translatesAutoresizingMaskIntoConstraints = false
        progressView.style = .bar
        progressView.isIndeterminate = false
        progressView.isHidden = true
        container.addSubview(progressView)
        
        // Bottom offline status bar
        offlineBanner = NSTextField(labelWithString: "⚠️ Сервертэй холбогдож чадсангүй. Та Go server (localhost:8080) болон Web app (localhost:3000)-аа ажиллуулна уу.")
        offlineBanner.translatesAutoresizingMaskIntoConstraints = false
        offlineBanner.font = NSFont.systemFont(ofSize: 11, weight: .medium)
        offlineBanner.textColor = NSColor.white
        offlineBanner.backgroundColor = NSColor(calibratedRed: 0.85, green: 0.2, blue: 0.2, alpha: 1.0)
        offlineBanner.drawsBackground = true
        offlineBanner.alignment = .center
        offlineBanner.isHidden = true
        container.addSubview(offlineBanner)
        
        // Native Offline Empty State View
        setupOfflineView(container: container)
        
        // Setup Auto Layout Constraints
        NSLayoutConstraint.activate([
            // Progress View
            progressView.topAnchor.constraint(equalTo: container.topAnchor),
            progressView.leadingAnchor.constraint(equalTo: container.leadingAnchor),
            progressView.trailingAnchor.constraint(equalTo: container.trailingAnchor),
            progressView.heightAnchor.constraint(equalToConstant: 3),
            
            // WebView
            webView.topAnchor.constraint(equalTo: container.topAnchor),
            webView.leadingAnchor.constraint(equalTo: container.leadingAnchor),
            webView.trailingAnchor.constraint(equalTo: container.trailingAnchor),
            webView.bottomAnchor.constraint(equalTo: offlineBanner.topAnchor),
            
            // Offline Banner at bottom
            offlineBanner.leadingAnchor.constraint(equalTo: container.leadingAnchor),
            offlineBanner.trailingAnchor.constraint(equalTo: container.trailingAnchor),
            offlineBanner.bottomAnchor.constraint(equalTo: container.bottomAnchor),
            offlineBanner.heightAnchor.constraint(equalToConstant: 26),
            
            // Offline view centered
            nativeOfflineView.centerXAnchor.constraint(equalTo: container.centerXAnchor),
            nativeOfflineView.centerYAnchor.constraint(equalTo: container.centerYAnchor),
            nativeOfflineView.widthAnchor.constraint(equalToConstant: 460),
            nativeOfflineView.heightAnchor.constraint(equalToConstant: 280)
        ])
        
        // Observe loading progress
        webView.addObserver(self, forKeyPath: #keyPath(WKWebView.estimatedProgress), options: .new, context: nil)
        
        self.view = container
    }
    
    private func setupOfflineView(container: NSView) {
        nativeOfflineView = NSView()
        nativeOfflineView.translatesAutoresizingMaskIntoConstraints = false
        nativeOfflineView.wantsLayer = true
        nativeOfflineView.layer?.backgroundColor = NSColor.controlBackgroundColor.cgColor
        nativeOfflineView.layer?.cornerRadius = 16
        nativeOfflineView.layer?.borderWidth = 1
        nativeOfflineView.layer?.borderColor = NSColor.separatorColor.cgColor
        nativeOfflineView.isHidden = true
        container.addSubview(nativeOfflineView)
        
        // App icon / Symbol
        let iconView: NSImageView
        if #available(macOS 11.0, *) {
            let symbolConfig = NSImage.SymbolConfiguration(pointSize: 48, weight: .regular)
            let img = NSImage(systemSymbolName: "network.slash", accessibilityDescription: "Offline")?.withSymbolConfiguration(symbolConfig)
            iconView = NSImageView(image: img ?? NSImage())
        } else {
            iconView = NSImageView()
        }
        iconView.frame = NSRect(x: 206, y: 196, width: 48, height: 48)
        nativeOfflineView.addSubview(iconView)
        
        // Title
        offlineTitleLabel = NSTextField(labelWithString: "Сервертэй холбогдож чадсангүй")
        offlineTitleLabel.font = NSFont.boldSystemFont(ofSize: 16)
        offlineTitleLabel.alignment = .center
        offlineTitleLabel.frame = NSRect(x: 20, y: 156, width: 420, height: 24)
        nativeOfflineView.addSubview(offlineTitleLabel)
        
        // Description
        offlineDescLabel = NSTextField(labelWithString: "Web Client (localhost:3000) болон Go Backend (localhost:8080) ажиллах шаардлагатай.")
        offlineDescLabel.font = NSFont.systemFont(ofSize: 13)
        offlineDescLabel.textColor = NSColor.secondaryLabelColor
        offlineDescLabel.alignment = .center
        offlineDescLabel.lineBreakMode = .byWordWrapping
        offlineDescLabel.frame = NSRect(x: 20, y: 104, width: 420, height: 40)
        nativeOfflineView.addSubview(offlineDescLabel)
        
        // Action Buttons
        let retryBtn = NSButton(title: "Дахин холбогдох", target: self, action: #selector(retryClicked))
        retryBtn.frame = NSRect(x: 140, y: 50, width: 180, height: 32)
        retryBtn.bezelStyle = .rounded
        retryBtn.keyEquivalent = "\r"
        nativeOfflineView.addSubview(retryBtn)
        
        let prefsBtn = NSButton(title: "Тохиргоо...", target: self, action: #selector(openPrefsClicked))
        prefsBtn.frame = NSRect(x: 160, y: 16, width: 140, height: 28)
        prefsBtn.bezelStyle = .inline
        nativeOfflineView.addSubview(prefsBtn)
    }
    
    override func viewDidLoad() {
        super.viewDidLoad()
        loadWebClient()
        
        // Listen for server status
        ServerManager.shared.startMonitoring { [weak self] apiOk, webOk in
            DispatchQueue.main.async {
                let bothOk = apiOk && webOk
                self?.offlineBanner.isHidden = bothOk
                if bothOk && self?.nativeOfflineView.isHidden == false {
                    self?.nativeOfflineView.isHidden = true
                    self?.loadWebClient()
                }
            }
        }
    }
    
    deinit {
        webView?.removeObserver(self, forKeyPath: #keyPath(WKWebView.estimatedProgress))
    }
    
    func loadWebClient(path: String = "") {
        let baseURL = ServerManager.shared.webBaseURL
        let target = path.isEmpty ? baseURL : "\(baseURL)\(path.hasPrefix("/") ? "" : "/")\(path)"
        if let url = URL(string: target) {
            let request = URLRequest(url: url)
            webView.load(request)
        }
    }
    
    @objc private func retryClicked() {
        nativeOfflineView.isHidden = true
        loadWebClient()
    }
    
    @objc private func openPrefsClicked() {
        PreferencesWindowController.shared.showWindow(nil)
    }
    
    // MARK: - Navigation Delegate
    
    func webView(_ webView: WKWebView, didStartProvisionalNavigation navigation: WKNavigation!) {
        progressView.isHidden = false
        progressView.doubleValue = 0.1
    }
    
    func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        progressView.isHidden = true
        offlineBanner.isHidden = true
        nativeOfflineView.isHidden = true
    }
    
    func webView(_ webView: WKWebView, didFailProvisionalNavigation navigation: WKNavigation!, withError error: Error) {
        progressView.isHidden = true
        offlineBanner.isHidden = false
        nativeOfflineView.isHidden = false
    }
    
    func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) {
        progressView.isHidden = true
        offlineBanner.isHidden = false
        nativeOfflineView.isHidden = false
    }
    
    override func observeValue(forKeyPath keyPath: String?, of object: Any?, change: [NSKeyValueChangeKey : Any]?, context: UnsafeMutableRawPointer?) {
        if keyPath == "estimatedProgress" {
            progressView.doubleValue = webView.estimatedProgress
            if webView.estimatedProgress >= 1.0 {
                DispatchQueue.main.asyncAfter(deadline: .now() + 0.2) {
                    self.progressView.isHidden = true
                }
            }
        }
    }
    
    // MARK: - UI Delegate & Popups
    
    func webView(_ webView: WKWebView, createWebViewWith configuration: WKWebViewConfiguration, for navigationAction: WKNavigationAction, windowFeatures: WKWindowFeatures) -> WKWebView? {
        if navigationAction.targetFrame == nil || !navigationAction.targetFrame!.isMainFrame {
            if let url = navigationAction.request.url {
                NSWorkspace.shared.open(url)
            }
        }
        return nil
    }
    
    // MARK: - JS Bridge (WKScriptMessageHandler)
    
    func userContentController(_ userContentController: WKUserContentController, didReceive message: WKScriptMessage) {
        guard message.name == "desktopBridge", let body = message.body as? [String: Any] else { return }
        
        let action = body["action"] as? String ?? ""
        
        switch action {
        case "authenticateBiometric":
            let reason = body["reason"] as? String ?? "Тоон гарын үсэг / Баталгаажуулалт"
            let callbackId = body["callbackId"] as? String
            BiometricAuth.shared.authenticate(reason: reason) { [weak self] success, err in
                if let cb = callbackId {
                    let js = "window.GeregeDesktop.onBiometricResult && window.GeregeDesktop.onBiometricResult('\(cb)', \(success), '\(err ?? "")');"
                    self?.webView.evaluateJavaScript(js, completionHandler: nil)
                }
            }
            
        case "sendNotification":
            let title = body["title"] as? String ?? "Gerege Nexus"
            let bodyMsg = body["body"] as? String ?? ""
            sendNativeNotification(title: title, body: bodyMsg)
            
        case "setBadgeCount":
            let count = body["count"] as? Int ?? 0
            DispatchQueue.main.async {
                NSApplication.shared.dockTile.badgeLabel = count > 0 ? "\(count)" : nil
            }
            
        case "openExternal":
            if let urlStr = body["url"] as? String, let url = URL(string: urlStr) {
                NSWorkspace.shared.open(url)
            }
            
        default:
            break
        }
    }
    
    private func sendNativeNotification(title: String, body: String) {
        let center = UNUserNotificationCenter.current()
        center.requestAuthorization(options: [.alert, .sound, .badge]) { granted, _ in
            if granted {
                let content = UNMutableNotificationContent()
                content.title = title
                content.body = body
                content.sound = UNNotificationSound.default
                
                let request = UNNotificationRequest(identifier: UUID().uuidString, content: content, trigger: nil)
                center.add(request, withCompletionHandler: nil)
            }
        }
    }
    
    // MARK: - Download Delegate
    func webView(_ webView: WKWebView, navigationAction: WKNavigationAction, didBecome download: WKDownload) {
        download.delegate = self
    }
    
    func download(_ download: WKDownload, decideDestinationUsing response: URLResponse, suggestedFilename: String, completionHandler: @escaping (URL?) -> Void) {
        let savePanel = NSSavePanel()
        savePanel.nameFieldStringValue = suggestedFilename
        savePanel.begin { result in
            if result == .OK {
                completionHandler(savePanel.url)
            } else {
                completionHandler(nil)
            }
        }
    }
}
