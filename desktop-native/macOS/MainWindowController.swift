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
    private let settings = NativeSettings.load()
    private var baseURLString: String { settings.webEndpoint.trimmingCharacters(in: CharacterSet(charactersIn: "/")) }

    public init() {
        let mask: NSWindow.StyleMask = [.titled, .closable, .miniaturizable, .resizable, .fullSizeContentView]
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
        showNativeLogin()
    }

    public func showNativeLogin() {
        guard let window, let content = window.contentView else { return }
        webView.isHidden = true
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

    public func nativeLoginDidSucceed(cookies: [HTTPCookie]) {
        let store = webView.configuration.websiteDataStore.httpCookieStore
        let group = DispatchGroup()
        for cookie in cookies {
            group.enter(); store.setCookie(cookie) { group.leave() }
        }
        group.notify(queue: .main) { [weak self] in
            guard let self else { return }
            self.loginController?.view.isHidden = true
            self.webView.isHidden = false
            self.loadRelativePath("/apps")
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
        print("[MainWindowController] Navigation failed: \(error.localizedDescription)")
    }
}
