//
//  MainWindowController.swift
//  GeregeNexusNativeMac
//
//  Created for Open Gerege Nexus Desktop Platform
//  Manages NSWindow & WKWebView Integration
//

import Foundation
import AppKit
import WebKit

public class MainWindowController: NSWindowController, WKNavigationDelegate, WKUIDelegate, NativeLoginDelegate {
    public var webView: WKWebView!
    private var ipcBridge: NativeIPCBridge?
    private var loginController: NativeLoginViewController?
    private var settingsWindowController: SettingsWindowController?
    private let ribbon = NSView()
    private let profileButton = NSButton()
    private let footer = NSView()
    private let footerStatus = NSTextField(labelWithString: "●  Холбогдож байна")
    private var profile: NativeUserProfile?
    private let ribbonHeight: CGFloat = 56
    private let footerHeight: CGFloat = 30
    private let settings = NativeSettings.load()
    private var baseURLString: String { settings.webEndpoint.trimmingCharacters(in: CharacterSet(charactersIn: "/")) }

    public init() {
        let mask: NSWindow.StyleMask = [.titled, .closable, .miniaturizable, .resizable]
        let rect = NSRect(x: 100, y: 100, width: 1280, height: 820)
        let window = NSWindow(contentRect: rect, styleMask: mask, backing: .buffered, defer: false)

        window.title = "Open Gerege Nexus Native"
        window.minSize = NSSize(width: 1024, height: 680)
        window.titlebarAppearsTransparent = false
        window.center()

        super.init(window: window)
        setupWebView()
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    private func setupWebView() {
        guard let window = window else { return }

        let config = WKWebViewConfiguration()
        let contentController = WKUserContentController()

        // Bridge Contract v1.3. The object exists before hydration and only in
        // the main frame. Native replies through one JSON-only entry point.
        let initScript = WKUserScript(
            source: """
            (() => {
              if (window.GeregeShell) return;
              const pending = new Map(), listeners = new Map(); let sequence = 0;
              window.__geregeShellResolve = (id, ok, value) => {
                const pair = pending.get(id); if (!pair) return; pending.delete(id);
                ok ? pair.resolve(value) : pair.reject(new Error(String(value)));
              };
              window.__geregeShellEmit = (name, payload) =>
                (listeners.get(name) || []).slice().forEach(fn => fn(payload));
              window.GeregeShell = Object.freeze({
                version: '1.3', platform: 'macos', formFactor: 'desktop',
                capabilities: Object.freeze(['external.open', 'print.system', 'biometric', 'device.identity']),
                invoke(method, params = {}) {
                  return new Promise((resolve, reject) => {
                    const id = String(++sequence); pending.set(id, {resolve, reject});
                    window.webkit.messageHandlers.geregeShell.postMessage({id, method, params});
                  });
                },
                on(name, handler) {
                  const list = listeners.get(name) || []; list.push(handler); listeners.set(name, list);
                  return () => { const i = list.indexOf(handler); if (i >= 0) list.splice(i, 1); };
                }
              });
              document.documentElement.setAttribute('data-shell', 'macos');
            })();
            """,
            injectionTime: .atDocumentStart,
            forMainFrameOnly: true
        )
        contentController.addUserScript(initScript)

        config.userContentController = contentController
        config.preferences.javaScriptCanOpenWindowsAutomatically = true

        webView = WKWebView(frame: window.contentView?.bounds ?? .zero, configuration: config)
        webView.autoresizingMask = [.width, .height]
        webView.navigationDelegate = self
        webView.uiDelegate = self
        webView.customUserAgent = "GeregeNexusNativeMac/1.0 (Macintosh; Intel Mac OS X)"

        // Initialize Native IPC
        ipcBridge = NativeIPCBridge(webView: webView, windowController: self)
        contentController.add(ipcBridge!, name: "geregeShell")

        window.contentView?.addSubview(webView)
        setupRibbon(in: window.contentView!)
        setupFooter(in: window.contentView!)
        showNativeLogin()
    }

    private func setupFooter(in content: NSView) {
        footer.wantsLayer = true
        footer.layer?.backgroundColor = NSColor(srgbRed: 16/255, green: 22/255, blue: 32/255, alpha: 1).cgColor
        footer.isHidden = true
        footer.frame = NSRect(x: 0, y: 0, width: content.bounds.width, height: footerHeight)
        footer.autoresizingMask = [.width, .maxYMargin]
        content.addSubview(footer)
        footerStatus.textColor = NSColor(srgbRed: 98/255, green: 217/255, blue: 212/255, alpha: 1)
        footerStatus.font = .systemFont(ofSize: 11, weight: .medium)
        let info = NSTextField(labelWithString: "macOS · Desktop   |   Shell v1.3")
        info.textColor = NSColor.white.withAlphaComponent(0.55); info.font = .systemFont(ofSize: 11)
        let settings = NSButton(title: "Тохиргоо", target: self, action: #selector(ribbonSettings)); settings.isBordered = false; settings.font = .systemFont(ofSize: 11)
        let spacer = NSView()
        let stack = NSStackView(views: [footerStatus, spacer, info, settings]); stack.orientation = .horizontal; stack.alignment = .centerY; stack.spacing = 12; stack.translatesAutoresizingMaskIntoConstraints = false
        footer.addSubview(stack)
        NSLayoutConstraint.activate([stack.leadingAnchor.constraint(equalTo: footer.leadingAnchor, constant: 14), stack.trailingAnchor.constraint(equalTo: footer.trailingAnchor, constant: -10), stack.topAnchor.constraint(equalTo: footer.topAnchor), stack.bottomAnchor.constraint(equalTo: footer.bottomAnchor)])
        spacer.setContentHuggingPriority(.defaultLow, for: .horizontal)
    }

    private func setupRibbon(in content: NSView) {
        ribbon.wantsLayer = true
        ribbon.layer?.backgroundColor = NSColor(srgbRed: 16/255, green: 22/255, blue: 32/255, alpha: 1).cgColor
        ribbon.layer?.borderWidth = 1
        ribbon.layer?.borderColor = NSColor.white.withAlphaComponent(0.08).cgColor
        ribbon.isHidden = true
        footer.isHidden = true
        ribbon.frame = NSRect(x: 0, y: content.bounds.height - ribbonHeight, width: content.bounds.width, height: ribbonHeight)
        ribbon.autoresizingMask = [.width, .minYMargin]
        content.addSubview(ribbon)

        let brand = NSTextField(labelWithString: "GEREGE / NEXUS")
        brand.font = .systemFont(ofSize: 12, weight: .bold)
        brand.textColor = NSColor(srgbRed: 98/255, green: 217/255, blue: 212/255, alpha: 1)
        let reload = ribbonButton("arrow.clockwise", "Дахин ачаалах", #selector(ribbonReload))
        let lock = ribbonButton("lock", "Түгжих", #selector(ribbonLock))
        let settings = ribbonButton("gearshape", "Тохиргоо", #selector(ribbonSettings))
        profileButton.target = self
        profileButton.action = #selector(showProfileMenu)
        profileButton.isBordered = false
        profileButton.imagePosition = .imageLeading
        profileButton.image = NSImage(systemSymbolName: "person.crop.circle.fill", accessibilityDescription: "Profile")
        profileButton.contentTintColor = .white
        profileButton.font = .systemFont(ofSize: 13, weight: .semibold)

        let spacer = NSView()
        let stack = NSStackView(views: [brand, spacer, reload, lock, settings, profileButton])
        stack.orientation = .horizontal
        stack.alignment = .centerY
        stack.spacing = 10
        stack.translatesAutoresizingMaskIntoConstraints = false
        ribbon.addSubview(stack)
        NSLayoutConstraint.activate([
            stack.leadingAnchor.constraint(equalTo: ribbon.leadingAnchor, constant: 20),
            stack.trailingAnchor.constraint(equalTo: ribbon.trailingAnchor, constant: -14),
            stack.topAnchor.constraint(equalTo: ribbon.topAnchor),
            stack.bottomAnchor.constraint(equalTo: ribbon.bottomAnchor),
            spacer.widthAnchor.constraint(greaterThanOrEqualToConstant: 20)
        ])
        spacer.setContentHuggingPriority(.defaultLow, for: .horizontal)
    }

    private func ribbonButton(_ symbol: String, _ label: String, _ action: Selector) -> NSButton {
        let button = NSButton(title: label, target: self, action: action)
        button.isBordered = false
        button.image = NSImage(systemSymbolName: symbol, accessibilityDescription: label)
        button.imagePosition = .imageLeading
        button.contentTintColor = NSColor.white.withAlphaComponent(0.82)
        button.font = .systemFont(ofSize: 12, weight: .medium)
        return button
    }

    public func showNativeLogin() {
        guard let window, let content = window.contentView else { return }
        webView.isHidden = true
        ribbon.isHidden = true
        if loginController == nil {
            let controller = NativeLoginViewController(apiEndpoint: settings.apiEndpoint)
            controller.delegate = self
            loginController = controller
        }
        guard let loginView = loginController?.view else { return }
        loginView.frame = content.bounds
        loginView.autoresizingMask = [.width, .height]
        if loginView.superview == nil { content.addSubview(loginView) }
        loginView.isHidden = false
    }

    func nativeLoginDidSucceed(cookies: [HTTPCookie], profile: NativeUserProfile) {
        self.profile = profile
        profileButton.title = profile.name.isEmpty ? "Хэрэглэгч" : profile.name
        let store = webView.configuration.websiteDataStore.httpCookieStore
        let group = DispatchGroup()
        for cookie in cookies {
            group.enter(); store.setCookie(cookie) { group.leave() }
        }
        group.notify(queue: .main) { [weak self] in
            guard let self else { return }
            self.loginController?.view.isHidden = true
            self.webView.isHidden = false
            self.ribbon.isHidden = false
            self.footer.isHidden = false
            self.layoutWorkArea()
            self.loadRelativePath("/apps")
        }
    }

    private func layoutWorkArea() {
        guard let content = window?.contentView else { return }
        webView.frame = NSRect(x: 0, y: footerHeight, width: content.bounds.width,
                               height: max(0, content.bounds.height - ribbonHeight - footerHeight))
        webView.autoresizingMask = [.width, .height]
    }

    public func showSettings() {
        if settingsWindowController == nil { settingsWindowController = SettingsWindowController() }
        settingsWindowController?.showWindow(nil)
        settingsWindowController?.window?.makeKeyAndOrderFront(nil)
    }

    @objc private func ribbonReload() { reloadPage() }
    @objc private func ribbonSettings() { showSettings() }
    @objc private func ribbonLock() { showNativeLogin() }

    @objc private func showProfileMenu() {
        let menu = NSMenu()
        let name = NSMenuItem(title: profile?.name ?? "Хэрэглэгч", action: nil, keyEquivalent: "")
        name.isEnabled = false
        menu.addItem(name)
        if let email = profile?.email, !email.isEmpty {
            let emailItem = NSMenuItem(title: email, action: nil, keyEquivalent: "")
            emailItem.isEnabled = false
            menu.addItem(emailItem)
        }
        menu.addItem(.separator())
        menu.addItem(withTitle: "Тохиргоо…", action: #selector(ribbonSettings), keyEquivalent: "")
        menu.addItem(withTitle: "Гарах", action: #selector(logout), keyEquivalent: "")
        menu.items.forEach { if $0.action != nil { $0.target = self } }
        menu.popUp(positioning: nil, at: NSPoint(x: 0, y: profileButton.bounds.height + 4), in: profileButton)
    }

    @objc public func logout() {
        Task {
            let endpoint = NativeSettings.load().apiEndpoint.trimmingCharacters(in: CharacterSet(charactersIn: "/"))
            let apiRoot = endpoint.hasSuffix("/api/v1") ? endpoint : endpoint + "/api/v1"
            if let url = URL(string: apiRoot + "/auth/logout") {
                var request = URLRequest(url: url); request.httpMethod = "POST"
                if let components = URLComponents(url: url, resolvingAgainstBaseURL: false) {
                    request.setValue("\(components.scheme ?? "https")://\(components.host ?? "")", forHTTPHeaderField: "Origin")
                }
                _ = try? await URLSession.shared.data(for: request)
            }
            await MainActor.run { self.clearSessionAndShowLogin() }
        }
    }

    private func clearSessionAndShowLogin() {
        profile = nil
        HTTPCookieStorage.shared.cookies?.filter { $0.name == "session_token" }.forEach(HTTPCookieStorage.shared.deleteCookie)
        let store = webView.configuration.websiteDataStore.httpCookieStore
        store.getAllCookies { cookies in
            let group = DispatchGroup()
            for cookie in cookies where cookie.name == "session_token" {
                group.enter(); store.delete(cookie) { group.leave() }
            }
            group.notify(queue: .main) { self.showNativeLogin() }
        }
    }

    public func loadRelativePath(_ path: String) {
        let fullURLString = baseURLString + path
        if let url = URL(string: fullURLString) {
            print("[MainWindowController] Navigating to: \(fullURLString)")
            let request = URLRequest(url: url)
            webView.load(request)
        }
    }

    public func reloadPage() {
        webView.reload()
    }

    // MARK: - WKNavigationDelegate
    public func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        footerStatus.stringValue = "●  Холбогдсон · \(webView.url?.host ?? "Nexus")"
        print("[MainWindowController] Page loaded successfully: \(webView.url?.absoluteString ?? "")")
    }

    public func webView(_ webView: WKWebView, decidePolicyFor navigationAction: WKNavigationAction,
                        decisionHandler: @escaping (WKNavigationActionPolicy) -> Void) {
        guard navigationAction.targetFrame?.isMainFrame != false,
              let url = navigationAction.request.url else {
            decisionHandler(.allow); return
        }
        let webOrigin = URL(string: baseURLString)
        let isWebOrigin = url.scheme == webOrigin?.scheme && url.host == webOrigin?.host && url.port == webOrigin?.port
        if isWebOrigin { decisionHandler(.allow); return }
        if ["http", "https", "mailto", "tel"].contains(url.scheme?.lowercased() ?? "") {
            NSWorkspace.shared.open(url)
        }
        decisionHandler(.cancel)
    }

    public func webView(_ webView: WKWebView, didFailProvisionalNavigation navigation: WKNavigation!, withError error: Error) {
        footerStatus.stringValue = "●  Холболт тасарсан"
        footerStatus.textColor = .systemRed
        print("[MainWindowController] Navigation failed: \(error.localizedDescription)")
    }
}
