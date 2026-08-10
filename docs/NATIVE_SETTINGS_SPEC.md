# Native Settings specification v1.0

Native Settings нь зөвхөн бүрхүүл, OS, төхөөрөмж, peripheral болон fleet
тохиргоог эзэмшинэ. Tenant, хэрэглэгч, бизнес модуль, workflow, theme зэрэг
web-д шийдэгдэх тохиргоог энд давхардуулахгүй.

## Мэдээллийн архитектур

| Section | Нийтлэг талбарууд | Form-factor |
| --- | --- | --- |
| Ерөнхий | startup, language, download folder, default route | desktop/mobile/tablet |
| Холболт | web/API endpoint, proxy, timeout, offline cache | бүгд |
| Принтер | driver, USB/network/serial, host/port, paper width, test page | desktop/kiosk/POS |
| Сканнер | keyboard wedge, camera/vendor, suffix, debounce, test scan | kiosk/POS/tablet |
| Serial ports | port, baud, data bits, parity, stop bits, reconnect | desktop/kiosk/POS |
| Cash drawer | printer pulse, direct serial, open duration, test | POS |
| Нууцлал | biometric lock, idle timeout, secure-store reset | desktop/mobile/tablet/POS |
| Device | enrollment status, device name, site, token rotate | kiosk/POS |
| Lockdown | dedicated mode, allowed exit PIN, reboot policy | kiosk |
| Update | channel, auto-download, maintenance window | бүгд |
| Diagnostics | versions, health, logs, export, peripheral tests | бүгд |

Layout нь desktop/tablet дээр scroll хийдэг зүүн sidebar + detail pane. Compact
mobile дээр `NavigationSplitView` автоматаар drill-down navigation болно.

## Хадгалалт ба нууц

- Энгийн preference: UserDefaults / .NET user config / DataStore.
- Enrollment/device token, proxy password, exit PIN: Keychain / Credential
  Manager / Android Keystore; UI-д plaintext дахин харуулахгүй.
- Тохиргоо schema version-тэй байна. Үл мэдэгдэх key-г хадгалж, downgrade үед
  арилгахгүй.
- Test action нь хадгалахаас тусдаа; printer/serial test алдаа нь тухайн
  section дотроо actionable тайлбартай харагдана.
- Web/API endpoint нь дараагийн shell startup-аас login, cookie injection,
  navigation, origin allowlist болон device enrollment-д бүхэлд нь үйлчилнэ.
  Ажиллаж буй webview-ийн origin-ийг дундаас нь сольж security boundary-г
  бүдгэрүүлэхгүй; endpoint хадгалсны дараа restart шаардлагатайг UI мэдээлнэ.

## Native–web хил

Web нь native settings-ийн UI-г зурж, нууц утгыг уншихгүй. Device capability
өөрчлөгдвөл shell `shell:capabilities-changed` event илгээж болно; web зөвхөн
шинэ capability жагсаалтаар fallback-аа шийднэ.
