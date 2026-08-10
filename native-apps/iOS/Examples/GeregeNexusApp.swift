#if os(iOS)
import GeregeShellKit
import GeregeShellUI
import SwiftUI
import UserNotifications

final class AppDelegate: NSObject, UIApplicationDelegate, UNUserNotificationCenterDelegate {
    func application(_ application:UIApplication,didFinishLaunchingWithOptions options:[UIApplication.LaunchOptionsKey:Any]?=nil)->Bool { UNUserNotificationCenter.current().delegate=self;UNUserNotificationCenter.current().requestAuthorization(options:[.alert,.badge,.sound]){granted,_ in if granted{DispatchQueue.main.async{application.registerForRemoteNotifications()}}};return true }
    func application(_ application:UIApplication,didRegisterForRemoteNotificationsWithDeviceToken token:Data){UserDefaults.standard.set(token.map{String(format:"%02x",$0)}.joined(),forKey:"native.push.apnsToken")}
}

@main
struct GeregeNexusApp: App {
    @UIApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate
    @StateObject private var auth: AuthController
    @State private var showWorkArea = false

    init() {
        var configured = UserDefaults.standard.string(forKey: "native.settings.apiEndpoint") ?? "https://nexus.gerege.mn"
        if configured == "https://api.nexus.gerege.mn" || configured == "http://localhost:8080" {
            configured = "https://nexus.gerege.mn"
            UserDefaults.standard.set(configured, forKey: "native.settings.apiEndpoint")
        }
        let root = configured.trimmingCharacters(in: CharacterSet(charactersIn: "/"))
        let url = URL(string: root.hasSuffix("/api/v1") ? root + "/" : root + "/api/v1/")!
        _auth = StateObject(wrappedValue: AuthController(transport: URLSessionAuthTransport(apiBase: url)))
    }

    var body: some Scene {
        WindowGroup {
            Group {
                if showWorkArea {
                    NativeShellView(cookies: auth.sessionCookies,
                                    formFactor: UIDevice.current.userInterfaceIdiom == .pad ? "tablet" : "mobile") {
                        showWorkArea = false
                        auth.cancel()
                    }
                } else {
                    NativeLoginView(auth: auth)
                }
            }
            .onChange(of: auth.phase) { phase in if phase == .success { showWorkArea = true; registerPushToken() } }
            .onOpenURL { _ in /* The in-flight poll owns completion; app link only re-activates this scene. */ }
        }
    }
    private func registerPushToken(){guard let token=UserDefaults.standard.string(forKey:"native.push.apnsToken") else{return};let endpoint=UserDefaults.standard.string(forKey:"native.settings.apiEndpoint") ?? "https://nexus.gerege.mn";Task{try? await PushRegistration.registerAPNS(token:token,apiEndpoint:endpoint)}}
}
#else
@main
struct GeregeNexusAppBuildStub {
    static func main() {}
}
#endif
