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
        root.appearance = NSAppearance(named: .darkAqua)
        root.layer?.backgroundColor = Palette.navy.cgColor

        let eyebrow = NSTextField(labelWithString: "G E R E G E  /  N E X U S")
        eyebrow.font = .systemFont(ofSize: 13, weight: .bold)
        eyebrow.textColor = Palette.teal
        let title = NSTextField(labelWithString: "Таны баталгаатай\nажлын орчин")
        title.font = .systemFont(ofSize: 36, weight: .semibold)
        title.textColor = .white
        title.maximumNumberOfLines = 2
        let subtitle = NSTextField(labelWithString: "Gerege Nexus-д үргэлжлүүлэхийн тулд нэвтэрнэ үү.")
        subtitle.font = .systemFont(ofSize: 14, weight: .regular)
        subtitle.textColor = Palette.muted

        styleField(email, placeholder: "И-мэйл")
        styleField(password, placeholder: "Нууц үг")
        styleField(nationalID, placeholder: "Регистрийн дугаар (АА00112233)")

        passwordButton.target = self; passwordButton.action = #selector(loginPassword)
        pushButton.target = self; pushButton.action = #selector(loginPush)
        qrButton.target = self; qrButton.action = #selector(loginQR)
        cancelButton.target = self; cancelButton.action = #selector(cancel)
        styleButton(passwordButton, primary: true)
        styleButton(pushButton, primary: true)
        styleButton(qrButton, primary: false)
        styleButton(cancelButton, primary: false)
        status.font = .systemFont(ofSize: 13, weight: .medium)
        status.textColor = Palette.muted
        status.maximumNumberOfLines = 3

        let divider = NSBox(); divider.boxType = .separator
        qrImage.imageScaling = .scaleProportionallyUpOrDown; qrImage.isHidden = true
        let eIDLabel = NSTextField(labelWithString: "eID MONGOLIA")
        eIDLabel.font = .systemFont(ofSize: 11, weight: .semibold)
        eIDLabel.textColor = Palette.muted
        let stack = NSStackView(views: [eyebrow, title, subtitle, email, password, passwordButton, divider, eIDLabel, nationalID, pushButton, qrButton, qrImage, status, cancelButton])
        stack.orientation = .vertical; stack.spacing = 12; stack.alignment = .leading
        stack.translatesAutoresizingMaskIntoConstraints = false
        root.addSubview(stack)
        stack.setCustomSpacing(8, after: eyebrow)
        stack.setCustomSpacing(20, after: subtitle)
        stack.setCustomSpacing(24, after: passwordButton)
        stack.setCustomSpacing(10, after: divider)
        [email, password, nationalID, passwordButton, pushButton, qrButton, status, cancelButton, divider].forEach {
            $0.translatesAutoresizingMaskIntoConstraints = false
            $0.widthAnchor.constraint(equalToConstant: 440).isActive = true
        }
        qrImage.translatesAutoresizingMaskIntoConstraints = false; qrImage.widthAnchor.constraint(equalToConstant: 190).isActive = true; qrImage.heightAnchor.constraint(equalToConstant: 190).isActive = true
        NSLayoutConstraint.activate([
            stack.centerXAnchor.constraint(equalTo: root.centerXAnchor),
            stack.centerYAnchor.constraint(equalTo: root.centerYAnchor),
            stack.widthAnchor.constraint(equalToConstant: 440)
        ])
        view = root
        auth.onPhase = { [weak self] in self?.render($0) }
        render(.idle)
    }

    private func styleField(_ field: NSTextField, placeholder: String) {
        field.controlSize = .large
        field.font = .systemFont(ofSize: 15)
        field.textColor = .white
        field.drawsBackground = true
        field.backgroundColor = Palette.surface
        field.isBezeled = true
        field.bezelStyle = .roundedBezel
        field.focusRingType = .default
        field.placeholderAttributedString = NSAttributedString(
            string: placeholder,
            attributes: [.foregroundColor: Palette.muted]
        )
        field.heightAnchor.constraint(equalToConstant: 46).isActive = true
    }

    private func styleButton(_ button: NSButton, primary: Bool) {
        button.isBordered = false
        button.wantsLayer = true
        button.layer?.cornerRadius = 9
        button.layer?.borderWidth = primary ? 0 : 1.5
        button.layer?.borderColor = Palette.teal.cgColor
        button.layer?.backgroundColor = (primary ? Palette.teal : .clear).cgColor
        button.contentTintColor = primary ? Palette.navy : Palette.teal
        button.font = .systemFont(ofSize: 15, weight: .semibold)
        button.heightAnchor.constraint(equalToConstant: 46).isActive = true
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

private enum Palette {
    static let navy = NSColor(srgbRed: 11/255, green: 15/255, blue: 23/255, alpha: 1)
    static let surface = NSColor(srgbRed: 29/255, green: 34/255, blue: 43/255, alpha: 1)
    static let teal = NSColor(srgbRed: 98/255, green: 217/255, blue: 212/255, alpha: 1)
    static let muted = NSColor(srgbRed: 166/255, green: 175/255, blue: 188/255, alpha: 1)
}
