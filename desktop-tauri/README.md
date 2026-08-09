# Gerege Nexus — Tauri v2 бүрхүүл

**Native Shell + Web Work Area** архитектурын кросс-платформ хэрэгжилт.
Бүрхүүл нь нэвтрэлт, цэс, tray, төхөөрөмжийн хандалтыг эзэмшинэ; web app нь
бүрхүүл дотор өөрийн chrome-оо нуугаад зөвхөн **ажлын муж** болж
рендерлэгдэнэ.

<p>
  <img src="../docs/assets/icons/flag-mn.png" width="18" height="18" alt=""> <b>Монгол</b>
</p>

Гэрээ: [`docs/SHELL_CONTRACT.md`](../docs/SHELL_CONTRACT.md) — өөрчлөгдөхгүй
спецификац. Лавлагаа хэрэгжилт: [`desktop-mac/`](../desktop-mac) (Swift/AppKit).

---

## 1. Бүтэц

```
desktop-tauri/
├── ui/                     Бүрхүүлийн локал цонхнууд (build хийдэггүй, цэвэр HTML)
│   ├── login.html          Нэвтрэх — имэйл/нууц үг ба eID
│   ├── prefs.html          Серверийн хаяг (зөвхөн dev build-д засагдана)
│   ├── offline.html        Сервер холбогдоогүй үеийн дэлгэц
│   └── shell.css           Локал цонхнуудын нийтлэг хэв маяг
└── src-tauri/
    ├── tauri.conf.json     Цонх, CSP, deep link, updater placeholder
    ├── capabilities/       IPC-ийн эрхүүд + remote origin-ы жагсаалт
    └── src/
        ├── lib.rs          Эхлүүлэлт, plugin, state, deep link
        ├── shell.rs        Гэрээ: inject script, event, capability
        ├── commands.rs     IPC гадаргуу (shell_invoke ба локал командууд)
        ├── bridge.rs       API-гийн хүсэлтийн тээвэр (доорхыг үз)
        ├── auth.rs         Имэйл/нууц үг ба eID long-poll
        ├── menus.rs        Сервэрийн цэснээс native цэс
        ├── health.rs       5 секунд тутмын /healthz шалгалт
        ├── tray.rs         Tray дүрс ба серверийн төлөв
        ├── windows.rs      Цонхнууд ба навигацийн allowlist
        └── config.rs       API/Web хаягийн шийдэл
```

Ажлын муж нь **алсын Web URL**-аас ачаалагдана. `ui/` доторх файлууд нь зөвхөн
бүрхүүлийн өөрийн цонхнууд — frontend bundle биш, build алхам шаардахгүй.

---

## 2. Урьдчилсан шаардлага

| Платформ | Шаардлага |
| --- | --- |
| Бүгд | Rust 1.77+ (`rustup`), `cargo` |
| macOS | Xcode Command Line Tools (`xcode-select --install`) |
| Windows | Visual Studio Build Tools (MSVC, "Desktop development with C++"), WebView2 Runtime (Windows 11-д суулгаастай) |
| Linux | `webkit2gtk-4.1`, `libappindicator3`, `librsvg2`, `patchelf`, `libsoup-3.0` |

Debian/Ubuntu дээр:

```bash
sudo apt install libwebkit2gtk-4.1-dev build-essential curl wget file \
  libxdo-dev libssl-dev libayatana-appindicator3-dev librsvg2-dev
```

Bundle (`.dmg`, `.msi`, `.AppImage`) хийхэд Tauri CLI хэрэгтэй:

```bash
cargo install tauri-cli --version "^2"
```

---

## 3. Build ба ажиллуулах

Хөгжүүлэлтийн горим (backend ба frontend аль хэдийн ажиллаж байх ёстой):

```bash
cd desktop-tauri/src-tauri
cargo tauri dev            # эсвэл: cargo run
```

Компиляц ба шалгалт:

```bash
cargo build
cargo clippy --all-targets -- -D warnings
cargo test
```

### Серверийн хаягууд

| Горим | Эх сурвалж |
| --- | --- |
| Dev (`debug_assertions`) | Preferences цонх → `~/.config/mn.gerege.nexus.tauri/prefs.json` |
| Production | **Compile-time тогтмол** — build хийхдээ шийднэ |

```bash
GEREGE_API_URL=https://api.example.mn \
GEREGE_WEB_URL=https://app.example.mn \
cargo tauri build
```

Хаяг заагаагүй бол `http://localhost:8080` ба `http://localhost:3000`.

> **Анхаар.** Ажлын муж IPC ашиглахын тулд түүний origin нь
> `src-tauri/capabilities/default.json`-ы `remote.urls` дотор байх ёстой. Энэ
> жагсаалт нь compile-time. Өөр origin руу байршуулж байгаа бол тэр файлыг
> хамт засна.

### Code signing — TODO

- **macOS**: Developer ID Application гэрчилгээ, `codesign`, дараа нь Apple-ийн
  notarization. Одоогийн байдлаар гарын үсэг зураагүй build гарна.
- **Windows**: Authenticode гэрчилгээ (`signtool`). SmartScreen-ий анхааруулга
  гарахгүй байхын тулд EV гэрчилгээ шаардлагатай.
- **Linux**: AppImage-д GPG гарын үсэг.

### Auto-update — TODO

`tauri-plugin-updater` нь **зориуд идэвхгүй**. Идэвхжүүлэхийн тулд:

1. `src-tauri/Cargo.toml` доторх `tauri-plugin-updater` мөрийг нээх.
2. `tauri.conf.json` → `plugins.updater` дотор бодит `endpoints` ба minisign
   `pubkey` бөглөх (одоо `TODO_` утгатай).
3. `src-tauri/src/lib.rs` дотор `.plugin(tauri_plugin_updater::Builder::new().build())` нэмэх.

Түлхүүргүй updater нь гарын үсэггүй шинэчлэлт суулгах эрсдэлтэй тул түлхүүр
бэлэн болтол идэвхжүүлэхгүй.

---

## 4. Гэрээний хэрэгжилт

| Зүйл | Утга |
| --- | --- |
| `version` | `1.0` |
| `platform` | build target-аас: `macos` / `windows` / `linux` |
| `capabilities` | `notify`, `badge`, `external.open`, `print.system`, `fs.save`, `menu.native` |
| Хэрэгжсэн method | `notify.show`, `badge.set`, `external.open`, `print.system`, `fs.saveAs`, `menu.changed`, `auth.reLogin` |
| Reject хийдэг | `biometric.authenticate` — desktop дээр хэрэгжилт алга |
| Илгээдэг event | `shell:navigate` (цэс, tray, deep link), `shell:search` (⌘/Ctrl+F), `shell:menu-refresh` (цэс дахин баригдсаны дараа) |

### Гэрээнд дутуу зүйл (тайлагнав, дур мэдэн нэмээгүй)

- **`secure-store`** — гэрээний v1-д энэ чадварыг ашиглах method
  тодорхойлогдоогүй (`SHELL_METHODS` дотор secure store-ын нэр алга). Method
  байхгүй капабилити зарлах нь худал мэдээлэл тул зарлаагүй. Хэрэглэх бол
  гэрээнд `secure.get` / `secure.set` / `secure.delete` нэмж, хувилбарыг
  **minor** болгож өсгөх ёстой.
- **`biometric`** — гэрээнд method бий (`biometric.authenticate`), гэвч
  Tauri-гийн биометр залгуур нь мобайл платформынх. Desktop дээр хэрэгжилт
  байхгүй тул капабилити зарлаагүй; дуудвал reject хийнэ.

---

## 5. Нэвтрэлт ба session — яагаад ийм замаар вэ

Платформын session нь `session_token` **HttpOnly** cookie бөгөөд **API-гийн
origin**-д харьяалагддаг. Web app (`frontend/lib/api.ts`) зөвхөн
`credentials: "include"`-ээр ажилладаг — `Authorization` толгой ашигладаггүй.

Тэгэхээр cookie нь ажлын мужийн webview-ийн cookie store-д байх ёстой, харин
түүнийг **гаднаас бичих API Tauri-д (болон wry-д) байхгүй**. Cookie-г тэнд
оруулах цорын ганц зөв арга бол `Set-Cookie`-г тухайн webview өөрөө хүлээж авах.

Тиймээс:

- **Урсгалын логик бүхэлдээ Rust талд**: eID-ийн 25 секундын long-poll, 400 мс
  завсар, 3 удаагийн алдааны тэвчил, 15 минутын зогсоох баталгаа — бүгд
  `src/auth.rs` дотор, `frontend/components/EIDLogin.tsx`-тэй ижил утгаар.
- **HTTP тээвэр нь webview-гээр**: `src/bridge.rs` нь хүсэлтийг JSON болгож
  ажлын мужид дамжуулж, хариуг IPC-ээр буцааж авна. Ингэснээр cookie зөв
  газартаа очно.
- Нэвтрэх хүртэл ажлын мужийн цонх **нуугдмал** байна (webview нь ачаалагдсан
  — тээврийн суваг нээлттэй байх ёстой), хэрэглэгч native login цонхыг л харна.

Локал `ui/` цонхнууд API руу шууд хандахгүй: backend-ийн CORS зөвхөн web
origin-ыг зөвшөөрдөг тул `tauri://localhost`-оос ирсэн хүсэлт хаагдана.

---

## 6. Аюулгүй байдал

| Хэмжүүр | Хэрэгжилт |
| --- | --- |
| Navigation allowlist | Гол frame зөвхөн Web URL-ын origin дотор. Бусад бүх хаяг системийн хөтчөөр (`windows.rs::create_main`) |
| Bridge зөвхөн main frame | Init script нь `window.top !== window` үед юу ч хийхгүй; Tauri-гийн IPC эрх нь `capabilities`-ээр origin-д уягдсан |
| Remote origin | `capabilities/default.json` → `remote.urls`; жагсаалтад байхгүй origin IPC дуудаж чадахгүй |
| CSP | `tauri.conf.json` → `app.security.csp`; локал цонхнуудад `frame-src 'none'`, `object-src 'none'` |
| JSON serialize | Native-аас JS руу орох бүх утга JSON-оор (`shell::js_string_literal`) — текст залгаж код үүсгэхгүй |
| `external.open` | Зөвхөн `http`, `https`, `mailto`, `tel` |
| `fs.saveAs` | Замыг web заахгүй; зөвхөн хэрэглэгчийн сонгосон файл руу бичнэ |
| Production хаяг | Compile-time тогтмол — суулгасан бүрхүүл танихгүй сервер рүү чиглэхгүй |

---

## 7. Гараар шалгах жагсаалт

**A. Нэвтрэлт → cookie → /apps**

1. Backend (`:8080`) ба frontend (`:3000`) ажиллуулна.
2. `cargo tauri dev` — native login цонх гарч, ажлын муж нуугдмал байна.
3. Имэйл/нууц үгээр нэвтэрнэ → login цонх хаагдаж, гол цонх гарч ирнэ.
4. Ажлын муж `/apps` дээр, хэрэглэгчийн өгөгдөл ачаалагдсан байна (өөрөөр
   хэлбэл session cookie webview-д зөв тавигдсан).
5. eID хувилбар: "eID QR" таб → QR гарна → eID аппаараа баталгаажуулна →
   ижил үр дүн. "eID дугаар" таб → регистрийн дугаар → push хүсэлт.

**B. Цэс сервэрээс баригдана**

1. Нэвтэрсний дараа цэсний мөрөнд тенантад суулгасан аппууд гарч ирнэ.
2. Апп Стороос модуль асаах/унтраах → web `menu.changed` дуудна → native цэс
   дахин баригдана.
3. Цэсний мөр дарахад ажлын муж бүтэн дахин ачаалагдалгүй шилжинэ.
4. Хэлээ солиод дахин ачаалахад цэсний шошго тухайн хэлээр гарна.

**C. Chromeless mode ба `data-shell`**

1. Ажлын мужид `gerege-topbar`, хажуугийн цэс, мобайл таб аль нь ч зурагдахгүй.
2. Ажлын муж дээр баруун товшоод *Inspect* (dev build) → `<html data-shell="macos">`
   (эсвэл `windows` / `linux`).
3. Console: `window.GeregeShell.capabilities` → дээрх жагсаалт.
4. `await window.GeregeShell.invoke("biometric.authenticate")` → **reject**.

**D. Гадаад URL webview-д нээгдэхгүй**

1. Console: `window.location.href = "https://example.com"`.
2. Хуудас **системийн хөтөч** дээр нээгдэж, аппын цонх өөрчлөгдөхгүй.
3. Аппын дотоод холбоос (`/apps`) хэвийн ажиллана.

**E. Deep link**

1. macOS: `open "gerege://apps"` · Windows: `start gerege://apps` ·
   Linux: `xdg-open gerege://apps`.
2. Цонх урд гарч, ажлын муж `/apps` руу шилжинэ.
3. `gerege://apps/esign` → `/apps/esign`.

**F. Хайлт ба бусад**

1. ⌘/Ctrl+F → ажлын мужид хайлтын давхарга нээгдэнэ (`shell:search`).
2. Tray дүрс дээр хулганаа авчрахад серверийн төлөв гарна; backend-ээ
   унтраахад 5 секундын дотор өөрчлөгдөж, offline цонх гарна.
3. Хэвлэх (⌘/Ctrl+P цэснээс), tray-гийн "Түргэн шилжих", "Дахин нэвтрэх..."
   бүгд ажиллана.
