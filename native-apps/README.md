# Gerege Nexus — pure native clients

This directory (`native-apps/`) contains the native client codebases: **iOS/iPadOS** (`GeregeShellKit` SPM + SwiftUI/WKWebView), **windows** (C#/.NET 8 WPF + WebView2), **android** (Kotlin/Compose/WebView), and **macOS** (AppKit/WKWebView).

---

## 📁 Architecture Overview

```
native-apps/
├── macOS/                   # macOS Native Shell (Swift 5.10 + AppKit + WKWebView)
│   ├── main.swift           # NSApplication Entry Point
│   ├── AppDelegate.swift    # App Lifecycle & Native Menu Bar (Gerege Nexus, Удирдах, Харах)
│   ├── MainWindowController.swift # NSWindow & WKWebView Integration
│   ├── NativeIPC.swift      # WKScriptMessageHandler Native IPC Bridge
│   └── build.sh             # Swiftc Compilation Script
│
├── iOS/                     # iOS/iPadOS app + shared Swift package
│   ├── Package.swift        # GeregeShellKit, GeregeShellUI, GeregeNexusApp
│   ├── Sources/             # Native login/settings and WKWebView shell
│   └── Tests/               # Swift auth state-machine tests
│
├── windows/                 # Windows Native Shell (C# .NET 8 + WPF + WebView2)
│   ├── GeregeNexusWin.csproj # .NET 8 Project File
│   ├── App.xaml / App.xaml.cs # WPF Application Lifecycle
│   ├── MainWindow.xaml / MainWindow.xaml.cs # Native Window & WebView2 Integration
│   └── NativeIPCBridge.cs   # CoreWebView2.WebMessageReceived IPC Bridge
│
├── android/                 # Android mobile/tablet/kiosk/POS clients
│   ├── core/                # Shared auth/device behavior
│   └── app/                 # Four form-factor flavors
│
└── shared/                  # Shared Specifications & Configurations
    ├── app_config.json      # Platform settings & dev server URLs
    └── IPC_CONTRACT.md      # Bi-directional JSON IPC Message Contract Specification
```

---

## 🚀 Building & Running

### 1. macOS Native Shell (Swift + AppKit)

**Prerequisites**: macOS 12+, Xcode Command Line Tools (`swiftc`, `xcrun`)

```bash
# Build the native macOS executable
cd native-apps/macOS
./build.sh

# Run the native macOS application
./GeregeNexusNativeMac
```

### 2. Windows Native Shell (C# + WPF + WebView2)

**Prerequisites**: Windows 10/11, .NET 8 SDK

```powershell
# Build and run on Windows
cd native-apps/windows
dotnet build -p:FormFactor=Desktop
dotnet build -p:FormFactor=Kiosk
dotnet build -p:FormFactor=POS
```

### 3. Android native clients (Kotlin + Compose)

Android Studio-д `native-apps/android`-ыг нээнэ. Нэг app module дөрвөн
form-factor flavor-тай: `mobile`, `tablet`, `kiosk`, `pos`; auth state machine
нь `:core` модульд байна.

```bash
gradle :core:test
gradle :app:assembleMobileDebug
gradle :app:assembleTabletDebug :app:assembleKioskDebug :app:assemblePosDebug
```

---

## ⚡ Native Features & Principles Preserved

1. **Native Login + Web Work Area**: Password and eID push are native controls. On success the shell copies `session_token` into the webview cookie store and opens `/apps`; web `/login` is never rendered in a native client.
2. **Native Menu Bar**:
   - macOS Top Menu Bar (`Gerege Nexus`, `Удирдах`, `Харах`) with native shortcuts (`⌘L`, `⌘,`, `⌘R`, `⌘Q`).
   - Windows Native Menu Bar (`Gerege Nexus`, `Удирдах`, `Харах`) with shortcuts (`Ctrl+L`, `Ctrl+,`, `F5`).
3. **Bridge Contract v1.3**:
   - `window.GeregeShell` is injected at document start, main-frame only.
   - `auth.reLogin` returns to native login; unknown methods reject.
   - [`../docs/NATIVE_LOGIN_SPEC.md`](../docs/NATIVE_LOGIN_SPEC.md) defines the shared state machine.

## Deployment ба update суваг

- macOS: notarized app bundle + Sparkle feed; signing/notarization identity нь
  release environment-ийн secret байна.
- iOS/iPadOS: TestFlight → phased App Store rollout, APNs entitlement/profile.
- Windows: Desktop/Kiosk/POS тусдаа MSIX identity; Assigned Access template нь
  [`windows/deployment`](windows/deployment)-д байна.
- Android: Play managed publishing эсвэл EMM/private APK channel; kiosk нь
  Android Enterprise device-owner + Lock Task ашиглана.

Signing certificate, Apple team, Play service account, payment/vendor SDK нь
repository-д хадгалагдахгүй. CI нь бүх unsigned compile target-ийг шалгана;
release job нь deployment secret байгаа үед гарын үсэг зурна.
