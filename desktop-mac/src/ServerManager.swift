import Foundation

class ServerManager: NSObject {
    static let shared = ServerManager()
    
    var apiBaseURL: String {
        get {
            UserDefaults.standard.string(forKey: "gerege_api_url") ?? "http://localhost:8080"
        }
        set {
            UserDefaults.standard.set(newValue, forKey: "gerege_api_url")
        }
    }
    
    var webBaseURL: String {
        get {
            UserDefaults.standard.string(forKey: "gerege_web_url") ?? "http://localhost:3000"
        }
        set {
            UserDefaults.standard.set(newValue, forKey: "gerege_web_url")
        }
    }
    
    /// Гол frame-ийн шилжилтэд зөвшөөрөгдөх гадаад origin-ууд.
    ///
    /// Платформын өөрийн web ба API-аас гадна цөөн тооны танилтын origin
    /// шаардлагатай: eID-ийн буцах хаяг гол frame-ээр дамждаг. Жагсаалтад
    /// байхгүй бүх хаяг webview дотор биш, системийн хөтчөөр нээгдэнэ.
    static let defaultExternalNavigationOrigins = [
        "https://eidmongolia.mn",
        "https://www.eidmongolia.mn",
        "https://developer.gerege.mn",
    ]

    /// Байршуулалт бүрийн нэмэлт origin-ууд (таслалаар тусгаарлана).
    ///
    /// Интеграцийн OAuth зөвшөөрлийн дэлгэц (Google, Dropbox гэх мэт) нь гол
    /// frame-ээр явдаг тул тухайн байгууллага өөрийн ашигладаг provider-ийн
    /// origin-ыг энд нэмнэ. Нэмээгүй бол урсгал системийн хөтчид үргэлжилнэ.
    var extraNavigationOrigins: [String] {
        (UserDefaults.standard.string(forKey: "gerege_nav_allowlist") ?? "")
            .split(separator: ",")
            .map { $0.trimmingCharacters(in: .whitespaces) }
            .filter { !$0.isEmpty }
    }

    private(set) var isAPIRunning = false
    private(set) var isWebRunning = false
    
    private var healthTimer: Timer?
    
    override private init() {
        super.init()
    }
    
    func startMonitoring(onChange: @escaping (Bool, Bool) -> Void) {
        checkHealth { apiOk, webOk in
            onChange(apiOk, webOk)
        }
        healthTimer?.invalidate()
        healthTimer = Timer.scheduledTimer(withTimeInterval: 5.0, repeats: true) { [weak self] _ in
            self?.checkHealth { apiOk, webOk in
                onChange(apiOk, webOk)
            }
        }
    }
    
    func checkHealth(completion: @escaping (Bool, Bool) -> Void) {
        let group = DispatchGroup()
        var apiResult = false
        var webResult = false
        
        group.enter()
        checkURL(urlString: "\(apiBaseURL)/healthz") { ok in
            apiResult = ok
            group.leave()
        }
        
        group.enter()
        checkURL(urlString: webBaseURL) { ok in
            webResult = ok
            group.leave()
        }
        
        group.notify(queue: .main) {
            self.isAPIRunning = apiResult
            self.isWebRunning = webResult
            completion(apiResult, webResult)
        }
    }
    
    // MARK: - Origin шалгалт

    /// "scheme://host:port" хэлбэрийн жиших боломжтой origin. Стандарт порт нь
    /// тодорхой бичигдсэн эсэхээс үл хамааран ижил утга өгнө.
    static func origin(of url: URL) -> String? {
        guard let scheme = url.scheme?.lowercased(), let host = url.host?.lowercased() else { return nil }
        let defaultPort = scheme == "https" ? 443 : (scheme == "http" ? 80 : -1)
        let port = url.port ?? defaultPort
        return port >= 0 ? "\(scheme)://\(host):\(port)" : "\(scheme)://\(host)"
    }

    static func origin(ofString value: String) -> String? {
        guard let url = URL(string: value) else { return nil }
        return origin(of: url)
    }

    /// Хаяг нь платформын web app-ынх мөн эсэх. Гүүрийн эрхийг зөвхөн энэ
    /// origin эзэмшинэ.
    func isAppOrigin(_ url: URL) -> Bool {
        guard let candidate = ServerManager.origin(of: url),
              let expected = ServerManager.origin(ofString: webBaseURL) else { return false }
        return candidate == expected
    }

    /// Гол frame webview дотроо очиж болох хаяг мөн эсэх.
    func isAllowedNavigation(_ url: URL) -> Bool {
        guard let candidate = ServerManager.origin(of: url) else { return false }
        var allowed = [webBaseURL, apiBaseURL].compactMap { ServerManager.origin(ofString: $0) }
        allowed += (ServerManager.defaultExternalNavigationOrigins + extraNavigationOrigins)
            .compactMap { ServerManager.origin(ofString: $0) }
        return allowed.contains(candidate)
    }

    private func checkURL(urlString: String, completion: @escaping (Bool) -> Void) {
        guard let url = URL(string: urlString) else {
            completion(false)
            return
        }
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        request.timeoutInterval = 2.5
        
        let task = URLSession.shared.dataTask(with: request) { _, response, error in
            if let httpRes = response as? HTTPURLResponse, (200...404).contains(httpRes.statusCode) {
                completion(true)
            } else {
                completion(false)
            }
        }
        task.resume()
    }
}
