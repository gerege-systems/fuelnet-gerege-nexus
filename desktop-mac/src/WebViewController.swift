import Cocoa
import WebKit
import UserNotifications

class WebViewController: NSViewController, WKNavigationDelegate, WKUIDelegate, WKScriptMessageHandler, WKDownloadDelegate {

    // MARK: - Shell contract v1

    static let messageHandlerName = "geregeShell"
    static let contractVersion = "1.0"
    static let platform = "macos"
    /// ЗӨВХӨН энэ бүрхүүлд ҮНЭХЭЭР хэрэгжсэн чадварууд. Хэрэгжүүлээгүй зүйлээ
    /// зарлавал web тал fallback-аа ажиллуулах боломжгүй болно.
    static let capabilities = ["biometric", "notify", "badge", "external.open", "print.system", "fs.save"]

    static let eventNavigate = "shell:navigate"
    static let eventSearch = "shell:search"
    static let eventMenuRefresh = "shell:menu-refresh"

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
        userContentController.add(self, name: WebViewController.messageHandlerName)

        // Inject the GeregeShell contract (docs/SHELL_CONTRACT.md).
        //
        // forMainFrameOnly: true — гүүр нь зөвхөн ажлын мужид хамаарна.
        // Ямар нэг байдлаар ачаалагдсан iframe (тайлангийн embed, гуравдагч
        // этгээдийн виджет) биометр, файл, мэдэгдэлд хүрэх шаардлагагүй.
        let userScript = WKUserScript(source: WebViewController.shellScriptSource,
                                      injectionTime: .atDocumentStart,
                                      forMainFrameOnly: true)
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
    
    /// Гол frame хаашаа очиж болохыг шийднэ.
    ///
    /// Бүрхүүл нь платформын ажлын мужийг л агуулна. Гадны хуудас энэ webview-д
    /// ачаалагдвал манай cookie, session, native гүүрийн хажууд суух тул
    /// зөвшөөрөгдсөн origin-оос гадуурх бүх шилжилт системийн хөтөч рүү гарна.
    func webView(_ webView: WKWebView, decidePolicyFor navigationAction: WKNavigationAction, decisionHandler: @escaping (WKNavigationActionPolicy) -> Void) {
        guard let url = navigationAction.request.url else {
            decisionHandler(.cancel)
            return
        }
        // Дэд frame-үүд гүүрт хүрэхгүй тул энд хязгаарлахгүй — эсрэг тохиолдолд
        // тайлан, газрын зураг зэрэг embed бүхэн эвдэрнэ.
        guard navigationAction.targetFrame?.isMainFrame ?? false else {
            decisionHandler(.allow)
            return
        }
        let scheme = url.scheme?.lowercased() ?? ""
        // about:blank ба blob: нь баримт үзүүлэх, хэвлэхэд WebKit өөрөө
        // үүсгэдэг дотоод хаягууд.
        if scheme == "about" || scheme == "blob" {
            decisionHandler(.allow)
            return
        }
        if ServerManager.shared.isAllowedNavigation(url) {
            decisionHandler(.allow)
            return
        }
        decisionHandler(.cancel)
        NSWorkspace.shared.open(url)
    }

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
        // Шинэ цонх нээхгүй: манай апп доторх холбоос энэ л webview-д
        // үргэлжилнэ, бусад нь системийн хөтчид гарна.
        if let url = navigationAction.request.url {
            if ServerManager.shared.isAllowedNavigation(url) {
                webView.load(navigationAction.request)
            } else {
                NSWorkspace.shared.open(url)
            }
        }
        return nil
    }
    
    // MARK: - Shell bridge (injected script)

    /// window.GeregeShell — гэрээний web талын хэрэгжилт.
    ///
    /// Тохиргоог JS дотор мөр залгаж биш, JSON-оор дамжуулж байгаа нь энэ
    /// файлын нийтлэг дүрэм: native талаас JS руу орох бүх утга JSON-оор
    /// кодлогдоно.
    private static var shellScriptSource: String {
        let config: [String: Any] = [
            "version": contractVersion,
            "platform": platform,
            "capabilities": capabilities,
            "handler": messageHandlerName,
        ]
        let configJSON = (try? JSONSerialization.data(withJSONObject: config, options: []))
            .flatMap { String(data: $0, encoding: .utf8) } ?? "{}"

        return """
        (function () {
          if (window.GeregeShell) { return; }
          var config = JSON.parse(\(jsStringLiteral(configJSON)));
          var pending = {};
          var counter = 0;

          // Native тал БҮХ хариугаа энэ нэг цэгээр буцаана. Хоёр аргумент нь
          // хоёулаа JSON тул хариу дотор ямар текст ирсэн ч код болж
          // ажиллах боломжгүй.
          window.__geregeShellResolve = function (id, payloadJSON) {
            var entry = pending[id];
            if (!entry) { return; }
            delete pending[id];
            var response;
            try { response = JSON.parse(payloadJSON); }
            catch (e) { entry.reject(new Error('shell: буруу хариу')); return; }
            if (response && response.ok) { entry.resolve(response.value); }
            else { entry.reject(new Error((response && response.error) || 'shell: invoke амжилтгүй')); }
          };

          window.__geregeShellEmit = function (name, payloadJSON) {
            var detail = null;
            try { detail = payloadJSON ? JSON.parse(payloadJSON) : null; }
            catch (e) { return; }
            window.dispatchEvent(new CustomEvent(name, { detail: detail }));
          };

          window.GeregeShell = Object.freeze({
            version: config.version,
            platform: config.platform,
            capabilities: Object.freeze(config.capabilities.slice()),
            invoke: function (method, params) {
              return new Promise(function (resolve, reject) {
                var id = 'gs' + (++counter);
                pending[id] = { resolve: resolve, reject: reject };
                try {
                  window.webkit.messageHandlers[config.handler].postMessage({
                    id: id, method: String(method), params: params || {}
                  });
                } catch (err) {
                  delete pending[id];
                  reject(err instanceof Error ? err : new Error(String(err)));
                }
              });
            },
            on: function (name, handler) {
              var listener = function (event) { handler(event.detail); };
              window.addEventListener(name, listener);
              return function () { window.removeEventListener(name, listener); };
            }
          });
        })();
        """
    }

    /// JS эх бичвэрт шууд суулгаж болох, бүрэн escape хийгдсэн string literal.
    ///
    /// Native талаас JS руу дамжих утга бүр үүгээр л явна. Өмнө нь алдааны
    /// текстийг мөр залгаж дамжуулдаг байсан — тэр текстэд нэг хашилт орвол
    /// дурын JS ажиллуулах нүх байв.
    private static func jsStringLiteral(_ value: String) -> String {
        guard let data = try? JSONSerialization.data(withJSONObject: [value], options: []),
              let wrapped = String(data: data, encoding: .utf8),
              wrapped.count >= 2 else { return "\"\"" }
        let literal = String(wrapped.dropFirst().dropLast())
        // JSON нь U+2028/U+2029-ийг түүхийгээр нь үлдээдэг, JS-ийн хуучин
        // парсерууд эдгээрийг мөр таслал гэж уншина.
        return literal
            .replacingOccurrences(of: "\u{2028}", with: "\\u2028")
            .replacingOccurrences(of: "\u{2029}", with: "\\u2029")
    }

    // MARK: - JS Bridge (WKScriptMessageHandler)

    func userContentController(_ userContentController: WKUserContentController, didReceive message: WKScriptMessage) {
        guard message.name == WebViewController.messageHandlerName else { return }
        // Гүүр нь зөвхөн ажлын мужийнх. Скрипт нь main frame-д л inject
        // хийгддэг ч гэсэн энд дахин шалгаж байгаа нь: гүүрийг эзэмших эрхийг
        // нэг л газар шийдэж, дараагийн тохиргооны алдаа нүх болохоос
        // сэргийлнэ.
        guard message.frameInfo.isMainFrame else { return }
        // Гол frame ямар нэг замаар гуравдагч этгээдийн хуудсанд очсон бол
        // native чадварууд түүнд нээлттэй байх ёсгүй.
        guard let frameURL = message.frameInfo.request.url,
              ServerManager.shared.isAppOrigin(frameURL) else { return }
        guard let body = message.body as? [String: Any],
              let id = body["id"] as? String,
              let method = body["method"] as? String else { return }

        handle(method: method, params: body["params"] as? [String: Any] ?? [:], id: id)
    }

    private func handle(method: String, params: [String: Any], id: String) {
        switch method {
        case "biometric.authenticate":
            let reason = params["reason"] as? String ?? "Тоон гарын үсэг / Баталгаажуулалт"
            BiometricAuth.shared.authenticate(reason: reason) { [weak self] success, err in
                if success {
                    self?.resolve(id: id, value: ["authenticated": true])
                } else {
                    self?.reject(id: id, error: err ?? "Баталгаажуулалт амжилтгүй боллоо")
                }
            }

        case "notify.show":
            sendNativeNotification(title: params["title"] as? String ?? "Gerege Nexus",
                                   body: params["body"] as? String ?? "")
            resolve(id: id, value: NSNull())

        case "badge.set":
            let count = params["count"] as? Int ?? 0
            DispatchQueue.main.async {
                NSApplication.shared.dockTile.badgeLabel = count > 0 ? "\(count)" : nil
            }
            resolve(id: id, value: NSNull())

        case "external.open":
            // Гадаад хөтөч рүү юу дамжуулж байгаагаа хязгаарлана: file:// эсвэл
            // бүртгэгдсэн дурын scheme нь webview-гээс код ажиллуулах гарц.
            guard let raw = params["url"] as? String,
                  let url = URL(string: raw),
                  let scheme = url.scheme?.lowercased(),
                  ["http", "https", "mailto", "tel"].contains(scheme) else {
                reject(id: id, error: "external.open: зөвшөөрөгдөөгүй URL")
                return
            }
            DispatchQueue.main.async { NSWorkspace.shared.open(url) }
            resolve(id: id, value: NSNull())

        case "print.system":
            DispatchQueue.main.async { [weak self] in
                guard let self = self, let window = self.view.window else {
                    self?.reject(id: id, error: "print.system: цонх алга")
                    return
                }
                let operation = self.webView.printOperation(with: NSPrintInfo.shared)
                operation.runModal(for: window, delegate: nil, didRun: nil, contextInfo: nil)
                self.resolve(id: id, value: NSNull())
            }

        case "fs.saveAs":
            // Хэрэглэгчийн сонгосон газарт л бичнэ — замыг web тал заахгүй.
            let suggested = (params["filename"] as? String ?? "document") as NSString
            var payload: Data?
            if let base64 = params["base64"] as? String { payload = Data(base64Encoded: base64) }
            else if let text = params["text"] as? String { payload = text.data(using: .utf8) }
            guard let data = payload else {
                reject(id: id, error: "fs.saveAs: агуулга алга эсвэл буруу base64")
                return
            }
            DispatchQueue.main.async { [weak self] in
                let panel = NSSavePanel()
                panel.nameFieldStringValue = suggested.lastPathComponent
                panel.begin { result in
                    guard result == .OK, let url = panel.url else {
                        self?.reject(id: id, error: "fs.saveAs: цуцлагдсан")
                        return
                    }
                    do {
                        try data.write(to: url)
                        self?.resolve(id: id, value: ["path": url.path])
                    } catch {
                        self?.reject(id: id, error: error.localizedDescription)
                    }
                }
            }

        case "menu.changed":
            // macOS-ийн native цэс одоогоор статик тул дахин барих зүйл алга.
            // Гэвч мэдэгдлийг хүлээн авсан гэдгээ хэлэх нь чухал: web тал үүнийг
            // алдаа гэж бүртгэх ёсгүй.
            resolve(id: id, value: NSNull())

        default:
            // Зарлаагүй method-ыг няцаана — web тал өөрийн fallback-аа
            // ажиллуулж чадна (жишээ нь auth.reLogin → /login).
            reject(id: id, error: "shell: дэмжигдээгүй method — \(method)")
        }
    }

    // MARK: - Shell responses & events

    private func resolve(id: String, value: Any) {
        respond(id: id, payload: ["ok": true, "value": value])
    }

    private func reject(id: String, error: String) {
        respond(id: id, payload: ["ok": false, "error": error])
    }

    private func respond(id: String, payload: [String: Any]) {
        guard let data = try? JSONSerialization.data(withJSONObject: payload, options: []),
              let json = String(data: data, encoding: .utf8) else { return }
        let js = "window.__geregeShellResolve && window.__geregeShellResolve("
            + WebViewController.jsStringLiteral(id) + ", "
            + WebViewController.jsStringLiteral(json) + ");"
        DispatchQueue.main.async { [weak self] in
            self?.webView.evaluateJavaScript(js, completionHandler: nil)
        }
    }

    /// Бүрхүүлээс ажлын муж руу event илгээнэ (цэс, toolbar, deep link).
    func emit(event: String, payload: [String: Any] = [:]) {
        guard let data = try? JSONSerialization.data(withJSONObject: payload, options: []),
              let json = String(data: data, encoding: .utf8) else { return }
        let js = "window.__geregeShellEmit && window.__geregeShellEmit("
            + WebViewController.jsStringLiteral(event) + ", "
            + WebViewController.jsStringLiteral(json) + ");"
        DispatchQueue.main.async { [weak self] in
            self?.webView.evaluateJavaScript(js, completionHandler: nil)
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
