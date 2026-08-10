#if os(iOS)
import GeregeShellKit
import SwiftUI

public struct NativeLoginView: View {
    @ObservedObject private var auth: AuthController
    @Environment(\.openURL) private var openURL
    @State private var email = ""
    @State private var password = ""
    @State private var nationalID = ""

    public init(auth: AuthController) { self.auth = auth }

    public var body: some View {
        GeometryReader { geometry in
            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    Text("GEREGE / NEXUS")
                        .font(.caption.weight(.bold)).tracking(2).foregroundStyle(Color.geregeTeal)
                    Text("Таны баталгаатай\nажлын орчин")
                        .font(.system(.largeTitle, design: .rounded, weight: .semibold))
                        .foregroundStyle(.white).padding(.bottom, 10)

                    TextField(String(localized: "auth.field.email", bundle: .module), text: $email).textContentType(.emailAddress)
                        .keyboardType(.emailAddress).textInputAutocapitalization(.never).loginField()
                    SecureField(String(localized: "auth.field.password", bundle: .module), text: $password).textContentType(.password).loginField()
                    Button(String(localized: "auth.action.admin_sign_in", bundle: .module)) { auth.password(email: email, password: password) }
                        .buttonStyle(GeregePrimaryButton()).disabled(isPending)

                    HStack { Rectangle().frame(height: 1); Text("eID Mongolia").font(.caption); Rectangle().frame(height: 1) }
                        .foregroundStyle(Color.white.opacity(0.18)).padding(.vertical, 10)

                    TextField(String(localized: "auth.eid.reg_number", bundle: .module), text: $nationalID)
                        .textInputAutocapitalization(.characters).autocorrectionDisabled().loginField()
                    Button(String(localized: "auth.eid.send_request", bundle: .module)) { auth.push(nationalID: nationalID) }
                        .buttonStyle(GeregePrimaryButton()).disabled(isPending)
                    Button(String(localized: "auth.action.app_to_app", bundle: .module)) {
                        auth.appToApp(callbackURL: "https://nexus.gerege.mn/auth/eid/callback")
                    }.buttonStyle(GeregeSecondaryButton()).disabled(isPending)

                    status
                    if isPending { Button(String(localized: "auth.action.cancel", bundle: .module), action: auth.cancel).frame(maxWidth: .infinity, alignment: .trailing) }
                }
                .frame(maxWidth: geometry.size.width > 700 ? 520 : 420, alignment: .leading)
                .padding(28).frame(maxWidth: .infinity, minHeight: geometry.size.height)
            }
        }
        .background(Color.geregeNavy.ignoresSafeArea())
        .preferredColorScheme(.dark)
        .onChange(of: auth.phase) { phase in
            if case .waiting(_, let link?) = phase { openURL(link) }
        }
    }

    private var isPending: Bool {
        if case .starting = auth.phase { return true }
        if case .waiting = auth.phase { return true }
        return false
    }

    @ViewBuilder private var status: some View {
        let message: String = switch auth.phase {
        case .idle: ""
        case .starting: String(localized: "auth.message.starting", bundle: .module)
        case .waiting(let code, _): "eID апп дээрх кодтой тулгана уу  ·  \(code)"
        case .success: String(localized: "auth.message.success", bundle: .module)
        case .expired: String(localized: "auth.message.expired", bundle: .module)
        case .refused: String(localized: "auth.message.refused", bundle: .module)
        case .error(let error): error
        }
        if !message.isEmpty {
            Text(message).font(.body.monospacedDigit()).foregroundStyle(.white)
                .frame(maxWidth: .infinity, alignment: .leading).padding(16)
                .background(Color.geregeTrustStrip, in: RoundedRectangle(cornerRadius: 8))
                .accessibilityLabel(message)
        }
    }
}

private extension View {
    func loginField() -> some View { self.padding(14).background(.white.opacity(0.07), in: RoundedRectangle(cornerRadius: 8)).overlay(RoundedRectangle(cornerRadius: 8).stroke(.white.opacity(0.16))) }
}
private struct GeregePrimaryButton: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label.fontWeight(.semibold).frame(maxWidth: .infinity).padding(14)
            .background(Color.geregeTeal.opacity(configuration.isPressed ? 0.72 : 1), in: RoundedRectangle(cornerRadius: 8)).foregroundStyle(Color.geregeNavy)
    }
}
private struct GeregeSecondaryButton: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label.frame(maxWidth: .infinity).padding(13)
            .overlay(RoundedRectangle(cornerRadius: 8).stroke(Color.geregeTeal)).foregroundStyle(Color.geregeTeal)
    }
}
private extension Color {
    static let geregeNavy = Color(red: 11/255, green: 15/255, blue: 23/255)
    static let geregeTeal = Color(red: 98/255, green: 217/255, blue: 212/255)
    static let geregeTrustStrip = Color(red: 23/255, green: 35/255, blue: 52/255)
}
#endif
