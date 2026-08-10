# Bridge Contract v1.3 — Native Shell + Web Work Area

Native бүрхүүл (Swift, C#, Kotlin) ба web app хоёрын хооронд байх
`window.GeregeShell` гэрээний бүртгэл.

<p>
  <img src="assets/icons/flag-mn.png" width="18" height="18" alt=""> <b>Монгол</b>
</p>

[Баримт бичгийн төв рүү буцах](README.md)

---

## 1. Зорилго

Платформ хоёр хэлбэрээр ажиллана:

- **Хөтчийн горим** — web app өөрөө бүрэн апп: толгой хэсэг, хажуугийн цэс,
  мобайл таб, нэвтрэлт бүгд түүнийх.
- **Бүрхүүлийн горим** — native бүрхүүл нь session-ий амьдралын мөчлөг
  (сэргээх, дахин нэвтрүүлэх, гарах), толгой хэсэг, цэс, хөл, төхөөрөмжийн
  хандалтыг эзэмшинэ. Web app өөрийн chrome-оо нуугаад зөвхөн **ажлын муж**
  болж рендерлэгдэнэ. Нэвтрэлтийн дэлгэц нь платформын native UI; web-ийн
  `/login` зөвхөн browser/PWA горимд үлдэнэ.

Гэрээ нь энэ хоёрын хилийг тодорхойлно. Хамгийн чухал шаардлага:
**хөтчийн горим ямар ч нөхцөлд өөрчлөгдөхгүй.** Бүрхүүл байхгүй үед
`window.GeregeShell` тодорхойлогдоогүй байна; web талын бүх helper `null`
эсвэл `false` буцаана; `data-shell` атрибут `<html>` дээр огт тавигдахгүй.

Web талын хэрэгжилт: [`frontend/lib/shell.ts`](../frontend/lib/shell.ts).
Бүрхүүлийн талын хэрэгжилтүүд: [`desktop-native/`](../desktop-native).

---

## 2. Инжекцийн дүрэм

| Дүрэм | Утга |
| --- | --- |
| Хугацаа | **Document start** — hydration эхлэхээс өмнө объект байрандаа байна. |
| Хамрах хүрээ | **Main frame only.** Дэд frame (iframe) руу скрипт inject хийхгүй. |
| Давхардал | Объект аль хэдийн байвал скрипт юу ч хийхгүй буцна. |
| Хувиршгүй байдал | `window.GeregeShell` нь `Object.freeze` хийгдсэн. |

Document start дээр inject хийх шаардлага нь зөвхөн тохь тухын асуудал биш:
`ThemeProvider` анхны рендер дээрээ `data-shell`-ийг тавьдаг тул объект хожуу
ирвэл хэрэглэгч эхлээд web-ийн харагдацыг хараад дараа нь native рүү үсэрнэ.

---

## 3. Объектын бүтэц

```ts
interface GeregeShell {
  version: string;       // гэрээний semver, одоо "1.3"
  platform: "macos" | "windows" | "ios" | "android" | "kiosk" | "pos";
  formFactor: "desktop" | "mobile" | "tablet" | "kiosk" | "pos";
  capabilities: string[];
  invoke<T>(method: string, params?: Record<string, unknown>): Promise<T>;
  on(event: string, handler: (payload: unknown) => void): () => void;
}
```

- `version` — **гэрээний** хувилбар, бүрхүүл програмын хувилбар БИШ.
- `platform` — `<html data-shell="...">` атрибутын утга болно.
- `capabilities` — тухайн бүрхүүлд **үнэхээр хэрэгжсэн** чадварууд. Энд
  байхгүй боломжийг зарлах нь web талын fallback-ыг ажиллах боломжгүй болгоно.
- `on()` — буцаах утга нь бүртгэлээ цуцлах функц (`useEffect`-ийн cleanup).

---

## 4. Capability нэрс

| Нэр | Утга |
| --- | --- |
| `biometric` | Touch ID / Windows Hello / төхөөрөмжийн эзэн танилт. |
| `notify` | Системийн мэдэгдэл. |
| `badge` | Апп дүрсэн дээрх тоолуур. |
| `external.open` | Системийн хөтчөөр гадаад хаяг нээх. |
| `print.system` | Системийн хэвлэх харилцах цонх. |
| `fs.save` | Файлыг хэрэглэгчийн сонгосон газарт хадгалах (`fs.saveAs` method). |
| `secure-store` | Keychain / Credential Manager маягийн нууц хадгалалт. |
| `menu.native` | Native цэсийг тенантын цэснээс динамикаар барих. |

Capability нэр ба method нэр нь тусдаа: capability нь *боломж*, method нь
*дуудлага*. Жишээ нь `fs.save` чадвар нь `fs.saveAs` method-оор хэрэгжинэ.

---

## 5. Method-ууд

Бүх дуудлага `invoke(method, params)` хэлбэртэй ба `Promise` буцаана.
Дэмжигдээгүй method-ыг бүрхүүл **reject** хийх ёстой — тэр үед web тал өөрийн
fallback-аа ажиллуулна.

### `auth.reLogin`

Session дуусахад нэвтрэлтийн урсгалыг эхлүүлнэ.

| | |
| --- | --- |
| Параметр | Байхгүй |
| Хариу | `null` — бүрхүүл нэвтрэлтийг барьж авсан, web тал өгөгдлөө дахин татна |
| Алдаа | Бүрхүүл нэвтрэлтийг эзэмшдэггүй бол reject; web тал `/login` руу шилжинэ |

Web тал үүнийг нэг session-д **нэг л удаа** оролдоно: дахин нэвтэрсэн ч
session хүчингүй хэвээр байвал мөчлөг үүсэхээс сэргийлнэ.

### `auth.lock`

Native biometric/PIN түгжээг гаргана. Unlock дуусах хүртэл resolve хийхгүй,
цуцалбал reject хийнэ. `auth.reLogin`, `auth.lock` нь lifecycle method учраас
capability шаардахгүй.

### `notify.show`

| | |
| --- | --- |
| Параметр | `{ title: string, body?: string }` |
| Хариу | `null` |
| Алдаа | Мэдэгдлийн зөвшөөрөл олгогдоогүй бол reject |

### `badge.set`

| | |
| --- | --- |
| Параметр | `{ count: number }` — `0` бол тоолуурыг арилгана |
| Хариу | `null` |

### `biometric.authenticate`

| | |
| --- | --- |
| Параметр | `{ reason?: string }` — хэрэглэгчид харагдах шалтгаан |
| Хариу | `{ authenticated: true }` |
| Алдаа | Цуцлагдсан, амжилтгүй, эсвэл боломжгүй үед системийн алдааны текстээр reject |

### `external.open`

| | |
| --- | --- |
| Параметр | `{ url: string }` |
| Хариу | `null` |
| Алдаа | Зөвхөн `http`, `https`, `mailto`, `tel` scheme зөвшөөрөгдөнө; бусад нь reject |

### `print.system`

| | |
| --- | --- |
| Параметр | Байхгүй |
| Хариу | `null` — харилцах цонх хаагдсаны дараа |
| Алдаа | Хэвлэх цонх нээх боломжгүй бол reject |

### `fs.saveAs`

| | |
| --- | --- |
| Параметр | `{ filename?: string, base64?: string, text?: string }` |
| Хариу | `{ path: string }` |
| Алдаа | Агуулга дутуу/буруу, хэрэглэгч цуцалсан, бичиж чадаагүй үед reject |

Замыг **web тал заахгүй** — зөвхөн санал болгох файлын нэр дамжуулна. Бүрхүүл
хэрэглэгчийн сонгосон газарт л бичнэ, дамжуулсан нэрнээс зөвхөн сүүлийн
бүрэлдэхүүнийг авна.

### `menu.changed`

Тенантын апп цэс өөрчлөгдсөнийг мэдэгдэнэ; бүрхүүл native цэсээ дахин барих
боломжтой болно.

| | |
| --- | --- |
| Параметр | `{}` |
| Хариу | `null` |

### Device method-ууд (v1.1–v1.3)

| Method | Capability | Параметр | Хариу |
| --- | --- | --- | --- |
| `escpos.print` | `escpos` | `{ text?: string, base64?: string, cut?: boolean }` | `null` |
| `escpos.drawer` | `escpos` | `{ pulse_ms?: number }` | `null` |
| `scanner.start` / `scanner.stop` | `scanner` | `{ mode?: string }` / `{}` | `null`; уншилт `shell:scan` event-ээр ирнэ |
| `serial.transact` | `serial` | `{ port?: string, baud?: number, base64: string, read_timeout_ms?: number }` | `{ base64: string }` |
| `device.identity` | `device.identity` | `{}` | Нууц token-гүй device identity |
| `camera.scan` | `camera.scan` | `{ formats?: string[] }` | `{ value: string, format: string }` |
| `kiosk.lockdown` | `kiosk.lockdown` | `{ enabled: boolean }` | `{ enabled: boolean }` |
| `telemetry.emit` | `telemetry` | `{ level: string, event: string, payload?: object }` | `null` |
| `payments.start` | `payments` | `{ amount: number, currency: string, reference: string }` | Vendor-neutral төлбөрийн үр дүн |

`payments.start` нь vendor SDK суусан adapter байхгүй үед reject хийнэ. Web тал
тэр үед өөр төлбөрийн арга руу fallback хийнэ. Device token, secure-store key,
printer raw credential зэрэг нь ямар ч method-ын хариунд орж болохгүй.

---

## 6. Event-ууд

Event нь бүрхүүлээс ажлын муж руу чиглэнэ. `on(name, handler)` нь `handler`-т
payload-ыг шууд дамжуулна.

| Event | Payload | Утга |
| --- | --- | --- |
| `shell:navigate` | `{ path: string }` | SPA router-ээр шилжинэ. Зөвхөн `/`-ээр эхэлсэн дотоод зам хүлээн авна. |
| `shell:search` | `{ query: string }` | Ажлын мужийн хайлтыг нээж, үгийг нь дамжуулна. |
| `shell:menu-refresh` | Байхгүй (`null`) | Цэсээ сервертэй дахин тааруулахыг хүснэ. |
| `shell:auth-changed` | `{ reason: string, user_id?: string }` | Session/ажилтан солигдсон; tenant data-г дахин татна. |
| `shell:capabilities-changed` | `{ capabilities: string[] }` | Peripheral/settings өөрчлөгдсөн; web fallback-аа дахин тооцно. |
| `shell:scan` | `{ value: string, format: string }` | Keyboard wedge, camera эсвэл vendor scanner уншилт. |

---

## 7. Хувилбарын дүрэм

Гэрээ **semver**-ээр явна. `GeregeShell.version` нь гэрээний хувилбар.

- **Minor (1.0 → 1.1)** — шинэ method, шинэ event, шинэ capability, эсвэл
  сонголтот параметр нэмэх. Хуучин web бүрхүүлийн шинэ хувилбартай ажиллана.
- **Major (1.x → 2.0)** — байгаа method-ын нэр, параметр, хариу, эсвэл
  event-ийн payload өөрчлөх, юм хасах. Бүрхүүл шинэ `version`-оо ЗААВАЛ
  зарлана.
- Device method нэмэхдээ түүнд харгалзах **capability-г мөн зарлана**.
  `auth.*` lifecycle method-ууд capability шаардахгүй.
- Web тал үл мэдэгдэх method-ыг дуудаж болно — reject ирнэ гэдгийг тооцсон
  fallback-тай байх ёстой. Энэ нь хуучин бүрхүүл дээр шинэ web ажиллах гол
  механизм.

---

## 8. Аюулгүй байдлын шаардлага

Бүрхүүл хэрэгжүүлэх бүрд дараах зүйлс **заавал** биелэх ёстой.

1. **Origin шалгалт.** Гүүрээр ирсэн мессежийг боловсруулахын өмнө
   илгээгч frame-ийн одоогийн хаягийн origin нь платформын web origin-той
   тохирч байгааг шалгана. Тохирохгүй бол мессежийг үл тоомсорлоно.
2. **Main frame only.** Скриптийг зөвхөн гол frame-д inject хийж, мессеж
   хүлээн авахдаа `isMainFrame`-ийг дахин шалгана. iframe нь биометр, файл,
   мэдэгдэлд хүрэхгүй.
3. **JSON serialize.** Native талаас JS руу орох **бүх** утга JSON-оор
   кодлогдоно. Алдааны текст, callback ID, хайлтын үг зэргийг JS эх бичвэрт
   мөр залгаж оруулахыг хориглоно — ямар ч эх сурвалжаас ирсэн текст код болж
   ажиллах боломжгүй байх ёстой.
4. **Хариуг нэг цэгээр.** Native хариу зөвхөн `__geregeShellResolve(id, json)`
   гэсэн нэг entry point-оор буцна. Дуудлага бүрд шинэ дэлхийн функц
   үүсгэдэггүй.
5. **Navigation allowlist.** Гол frame-ийн шилжилт зөвхөн зөвшөөрөгдсөн
   origin дотор явна: платформын web origin, API origin, түүнчлэн тодорхой
   нэрлэгдсэн танилтын origin-ууд. Бусад бүх хаяг webview дотор биш,
   **системийн хөтчөөр** нээгдэнэ.
6. **Scheme хязгаарлалт.** `external.open` нь зөвхөн `http`, `https`,
   `mailto`, `tel`-ийг хүлээн авна. `file://` болон бүртгэгдсэн дурын scheme
   нь webview-гээс код ажиллуулах гарц болно.
7. **Файлын зам.** Хадгалах байршлыг web тал сонгохгүй; зөвхөн хэрэглэгчийн
   сонгосон газар руу бичнэ.

---

## 9. Бүрхүүлийн одоогийн байдал

Хэрэгжилтүүд нь [`desktop-native/`](../desktop-native) доторх Swift/AppKit ба
C#/.NET сууриас эхэлнэ; Kotlin/Android суурь мөн энд нэмэгдэнэ. Linux нь PWA.

| Зүйл | Утга |
| --- | --- |
| `version` | `1.3` |
| `platform` | `macos`, `windows`, `ios`, `android`, `kiosk`, `pos` |
| `capabilities` | `notify`, `badge`, `external.open`, `print.system`, `fs.save` |
| Хэрэгжсэн method | `notify.show`, `badge.set`, `external.open`, `print.system`, `fs.saveAs`, `menu.changed`, `auth.reLogin` |
| Reject хийдэг | `biometric.authenticate` — desktop дээр хэрэгжилт алга; web тал өөрийн fallback-аа ажиллуулна |
| Илгээдэг event | `shell:navigate` (deep link, цэс, tray), `shell:search` (⌘/Ctrl+F) |

`menu.native` зарлагдаагүй: навигацийн цэсийг ажлын муж өөрөө зурдаг тул
бүрхүүл тенантын цэсийг native байдлаар барихаа больсон. Тиймээс
`shell:menu-refresh` мөн илгээгддэггүй — `menu.changed` дуудлагыг хүлээж авч,
хариу нь амжилттай гэдгийг л баталгаажуулна.

`secure-store` капабилити зарлагдаагүй: гэрээний v1-д түүнийг ашиглах method
тодорхойлогдоогүй тул зарлах нь дуудагдах боломжгүй амлалт болно. Хэрэглэх
шаардлагатай болбол гэрээнд method нэмж, хувилбарыг **minor** болгож өсгөнө.

### Navigation allowlist-ыг тохируулах

Гол frame-д анхдагчаар зөвшөөрөгдөх origin-ууд: Web ба API хаяг (dev горимд
Тохиргооны цонхноос, production-д compile-time тогтмол), мөн eID-ийн танилтын
origin-ууд. Байгууллага өөрийн интеграцийн OAuth зөвшөөрлийн дэлгэцийг
(Google, Dropbox гэх мэт) апп дотор үлдээхийг хүсвэл тэдгээрийн origin-ыг
нэмнэ. Нэмээгүй бол урсгал таслагдахгүй — зөвхөн системийн хөтөч дээр
үргэлжилнэ.

> **Анхаар.** Swift-ийн `WKNavigationDelegate`, WebView2-ийн
> `NavigationStarting`, Android-ийн `WebViewClient` гурвуул main-frame origin
> allowlist-ыг bridge message бүр дээр давхар шалгана.

---

## 10. Гараар шалгах хувилбарууд

Бүрхүүлийн өөрчлөлт бүрийн дараа дараах жагсаалтыг гүйцэтгэнэ.

**A. Хөтчийн горим өөрчлөгдөөгүй эсэх**

1. `cd frontend && npm run dev`, дараа нь хөтчөөр `http://localhost:3000` руу
   орж нэвтэрнэ.
2. Толгой хэсэг, хажуугийн цэс, мобайл таб бүгд урьдын адил байгааг харна.
3. DevTools → Elements: `<html>` дээр `data-shell` атрибут **байхгүй**.
4. Console: `window.GeregeShell` → `undefined`.

**B. Бүрхүүлийн горимд chrome нуугдсан эсэх**

1. Тухайн native target-ыг ажиллуулж, native дэлгэцээр нэвтэрнэ.
2. Нэвтэрсний дараа `gerege-topbar`, хажуугийн цэс, мобайл таб аль нь ч
   зурагдаагүй; зөвхөн ажлын муж ба AI туслах харагдана.
3. Апп Стороос модуль асаагаад/унтраагаад цэсний өгөгдөл шинэчлэгдэж байгааг
   шалгана — өгөгдлийн fetch нь зурагдахгүй ч ажиллаж байх ёстой.

**C. `data-shell` тавигдсан эсэх**

1. Аппын цонхон дээр баруун товшоод *Inspect Element* (dev build).
2. Elements: `<html data-shell="macos" ...>` — эсвэл `windows` / `ios` / `android`.
3. Console: `getComputedStyle(document.body).fontFamily` — Inter биш,
   системийн фонтоор эхэлсэн байна.

**D. Дэмжигдээгүй method няцаагдаж байгаа эсэх**

Console дээр:

```js
await window.GeregeShell.invoke("biometric.authenticate", { reason: "Тест" })
```

1. Promise **reject** хийнэ — desktop дээр биометр хэрэгжээгүй.
2. `window.GeregeShell.capabilities` дотор `biometric` байхгүйг шалгана: web
   тал fallback-аа ажиллуулах болзол нь тэр.
3. Хэрэгжсэнийг шалгана: `await window.GeregeShell.invoke("notify.show",
   { title: "Тест" })` → системийн мэдэгдэл гарна.

**E. iframe-ээс гүүр дуудагдахгүй эсэх**

Console дээр:

```js
const f = document.createElement("iframe");
document.body.appendChild(f);
f.contentWindow.GeregeShell;                       // undefined байх ёстой
f.contentWindow.webkit?.messageHandlers?.geregeShell; // undefined байх ёстой
```

**F. Гадаад URL webview дотор нээгдэхгүй эсэх**

1. Console дээр `window.location.href = "https://example.com"`.
2. Хуудас **системийн хөтөч** дээр нээгдэнэ; аппын цонх өөрчлөгдөхгүй үлдэнэ.
3. `window.open("https://example.com")` — мөн адил гадаад хөтчөөр нээгдэнэ.
4. Аппын дотоод холбоос (жишээ нь `/apps`) хэвийн ажиллана.

**G. Функциональ регресс байхгүй эсэх**

1. Цэсний мөрийн Хайх (⌘F) → үг бичиж Enter дарахад ажлын мужид хайлтын
   давхарга нээгдэж, үр дүн гарна.
2. Toolbar-ын Апп Стор / E-Sign / Төрийн үйлчилгээ товчнууд шилжүүлнэ.
3. Хэвлэх (⌘P), файл татах (Save panel), tray цэс, `gerege://apps` deep link
   бүгд ажиллана.
