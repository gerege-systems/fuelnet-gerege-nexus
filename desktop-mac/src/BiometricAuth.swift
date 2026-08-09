import Foundation
import LocalAuthentication

class BiometricAuth {
    static let shared = BiometricAuth()
    
    private init() {}
    
    func canEvaluatePolicy() -> Bool {
        let context = LAContext()
        var error: NSError?
        return context.canEvaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, error: &error)
    }
    
    func authenticate(reason: String, completion: @escaping (Bool, String?) -> Void) {
        let context = LAContext()
        context.localizedCancelTitle = "Цуцлах"
        
        var error: NSError?
        if context.canEvaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, error: &error) {
            context.evaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, localizedReason: reason) { success, authError in
                DispatchQueue.main.async {
                    if success {
                        completion(true, nil)
                    } else {
                        let msg = authError?.localizedDescription ?? "Баталгаажуулалт амжилтгүй боллоо"
                        completion(false, msg)
                    }
                }
            }
        } else {
            // Fallback to passcode authentication if biometrics unavailable
            if context.canEvaluatePolicy(.deviceOwnerAuthentication, error: &error) {
                context.evaluatePolicy(.deviceOwnerAuthentication, localizedReason: reason) { success, authError in
                    DispatchQueue.main.async {
                        if success {
                            completion(true, nil)
                        } else {
                            let msg = authError?.localizedDescription ?? "Баталгаажуулалт амжилтгүй боллоо"
                            completion(false, msg)
                        }
                    }
                }
            } else {
                let msg = error?.localizedDescription ?? "Биометрик баталгаажуулалт боломжгүй байна"
                completion(false, msg)
            }
        }
    }
}
