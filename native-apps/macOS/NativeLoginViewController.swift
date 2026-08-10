import AppKit
import CoreImage

@MainActor
protocol NativeLoginDelegate: AnyObject {
    func nativeLoginDidSucceed(cookies: [HTTPCookie])
}

@MainActor
final class NativeLoginViewController: NSViewController {
    weak var delegate: NativeLoginDelegate?
    private let auth: NativeAuth
    private let email = NSTextField()
    private let password = NSSecureTextField()
    private let nationalID = NSTextField()
    private let status = NSTextField(labelWithString: "")
    private let passwordButton = NSButton(title: "Нэвтрэх", target: nil, action: nil)
    private let pushButton = NSButton(title: "eID апп руу хүсэлт илгээх", target: nil, action: nil)
    private let qrButton = NSButton(title: "eID QR харуулах", target: nil, action: nil)
    private let qrImage = NSImageView()
    private let cancelButton = NSButton(title: "Цуцлах", target: nil, action: nil)

    init(apiEndpoint: String) { auth = NativeAuth(apiEndpoint: apiEndpoint); super.init(nibName: nil, bundle: nil) }
    required init?(coder: NSCoder) { fatalError() }

    override func loadView() {
        let root = NSView()
        root.wantsLayer = true
        root.layer?.backgroundColor = NSColor(calibratedRed: 0.043, green: 0.059, blue: 0.09, alpha: 1).cgColor

        let title = NSTextField(labelWithString: "Gerege Nexus")
        title.font = .systemFont(ofSize: 30, weight: .bold)
        title.textColor = .white
        let subtitle = NSTextField(labelWithString: "Тавтай морил — native нэвтрэлт")
        subtitle.textColor = .secondaryLabelColor

        email.placeholderString = "И-мэйл"
        password.placeholderString = "Нууц үг"
        nationalID.placeholderString = "Регистрийн дугаар (АА00112233)"
        [email, password, nationalID].forEach { $0.controlSize = .large }

        passwordButton.target = self; passwordButton.action = #selector(loginPassword)
        pushButton.target = self; pushButton.action = #selector(loginPush)
        qrButton.target = self; qrButton.action = #selector(loginQR)
        cancelButton.target = self; cancelButton.action = #selector(cancel)
        passwordButton.bezelStyle = .rounded; pushButton.bezelStyle = .rounded
        status.textColor = .secondaryLabelColor
        status.maximumNumberOfLines = 3

        let divider = NSBox(); divider.boxType = .separator
        qrImage.imageScaling = .scaleProportionallyUpOrDown; qrImage.isHidden = true
        let stack = NSStackView(views: [title, subtitle, email, password, passwordButton, divider, nationalID, pushButton, qrButton, qrImage, status, cancelButton])
        stack.orientation = .vertical; stack.spacing = 14; stack.alignment = .leading
        stack.translatesAutoresizingMaskIntoConstraints = false
        root.addSubview(stack)
        [email, password, nationalID, passwordButton, pushButton, qrButton, status, cancelButton, divider].forEach {
            $0.translatesAutoresizingMaskIntoConstraints = false
            $0.widthAnchor.constraint(equalToConstant: 390).isActive = true
        }
        qrImage.translatesAutoresizingMaskIntoConstraints = false; qrImage.widthAnchor.constraint(equalToConstant: 190).isActive = true; qrImage.heightAnchor.constraint(equalToConstant: 190).isActive = true
        NSLayoutConstraint.activate([
            stack.centerXAnchor.constraint(equalTo: root.centerXAnchor),
            stack.centerYAnchor.constraint(equalTo: root.centerYAnchor),
            stack.widthAnchor.constraint(equalToConstant: 390)
        ])
        view = root
        auth.onPhase = { [weak self] in self?.render($0) }
        render(.idle)
    }

    @objc private func loginPassword() { auth.password(email: email.stringValue, password: password.stringValue) }
    @objc private func loginPush() { auth.push(nationalID: nationalID.stringValue) }
    @objc private func loginQR() { auth.qr() }
    @objc private func cancel() { auth.cancel() }

    private func render(_ phase: LoginPhase) {
        let pending: Bool
        switch phase {
        case .idle: status.stringValue = ""; pending = false
        case .starting: status.stringValue = "Хүсэлт эхлүүлж байна…"; pending = true
        case .waiting(let code, let link): status.stringValue = "eID апп дээр зөвшөөрнө үү. Баталгаажуулах код: \(code)"; pending = true; renderQR(link)
        case .success:
            status.stringValue = "Амжилттай нэвтэрлээ"; pending = false
            delegate?.nativeLoginDidSucceed(cookies: auth.sessionCookies())
        case .expired: status.stringValue = "Хүсэлтийн хугацаа дууслаа. Дахин оролдоно уу."; pending = false
        case .refused: status.stringValue = "eID хүсэлтээс татгалзлаа."; pending = false
        case .error(let message): status.stringValue = message; pending = false
        }
        passwordButton.isEnabled = !pending
        pushButton.isEnabled = !pending
        qrButton.isEnabled = !pending
        if !pending { qrImage.isHidden = true }
        cancelButton.isHidden = !pending
    }

    private func renderQR(_ link: URL?) { guard let link, let filter = CIFilter(name: "CIQRCodeGenerator") else { qrImage.isHidden = true; return }; filter.setValue(Data(link.absoluteString.utf8), forKey: "inputMessage"); filter.setValue("Q", forKey: "inputCorrectionLevel"); guard let output = filter.outputImage?.transformed(by: CGAffineTransform(scaleX: 8, y: 8)) else { return }; let rep = NSCIImageRep(ciImage: output); let image = NSImage(size: rep.size); image.addRepresentation(rep); qrImage.image = image; qrImage.isHidden = false }
}
