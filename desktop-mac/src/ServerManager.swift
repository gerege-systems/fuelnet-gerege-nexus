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
