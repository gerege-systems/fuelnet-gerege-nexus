//! Bridge Contract v1 — `window.GeregeShell`.
//!
//! Гэрээ нь `docs/SHELL_CONTRACT.md`-д тодорхойлогдсон бөгөөд өөрчлөгдөхгүй
//! спецификац. Энэ модуль түүний Tauri талын хэрэгжилт: inject хийх скрипт,
//! `invoke()`-ийн диспетчер, event илгээх туслахууд.

use serde_json::{json, Value};
use tauri::{AppHandle, Manager, WebviewWindow};

use crate::config::Endpoints;

pub const CONTRACT_VERSION: &str = "1.0";

/// Гэрээгээр тогтсон event нэрс.
pub const EVENT_NAVIGATE: &str = "shell:navigate";
pub const EVENT_SEARCH: &str = "shell:search";
pub const EVENT_MENU_REFRESH: &str = "shell:menu-refresh";

/// Ажлын мужийг агуулах цонхны шошго.
pub const MAIN_WINDOW: &str = "main";

/// Build target-аас гарах платформын нэр. Гэрээний `platform` талбар ба
/// `<html data-shell="...">` атрибутын утга болно.
pub fn platform() -> &'static str {
    if cfg!(target_os = "macos") {
        "macos"
    } else if cfg!(target_os = "windows") {
        "windows"
    } else {
        "linux"
    }
}

/// ЗӨВХӨН энэ бүрхүүлд үнэхээр хэрэгжсэн чадварууд.
///
/// `biometric` энд алга: Tauri-гийн биометр залгуур нь мобайл платформынх,
/// desktop дээр хэрэгжилт байхгүй. `secure-store` мөн алга — гэрээний v1-д
/// түүнийг ашиглах method тодорхойлогдоогүй тул зарлах нь утгагүй болно.
pub fn capabilities() -> Vec<&'static str> {
    vec![
        "notify",
        "badge",
        "external.open",
        "print.system",
        "fs.save",
        "menu.native",
    ]
}

/// JS эх бичвэрт шууд суулгаж болох, бүрэн escape хийгдсэн string literal.
///
/// Native талаас JS руу орох утга бүр үүгээр эсвэл JSON-оор л явна. Гэрээний
/// аюулгүй байдлын шаардлага #3: ямар ч эх сурвалжаас ирсэн текст код болж
/// ажиллах боломжгүй байх ёстой.
pub fn js_string_literal(value: &str) -> String {
    let encoded = match serde_json::to_string(value) {
        Ok(encoded) => encoded,
        // Энгийн мөрийг JSON болгоход алдаа гарах боломж бараг байхгүй ч
        // panic хийхээс хоосон мөр буцаах нь дээр.
        Err(_) => return "\"\"".to_string(),
    };
    // JSON нь U+2028/U+2029-ийг түүхийгээр нь үлдээдэг; JS-ийн зарим парсер
    // эдгээрийг мөр таслал гэж уншина.
    encoded.replace('\u{2028}', "\\u2028").replace('\u{2029}', "\\u2029")
}

/// Document start дээр inject хийх скрипт.
///
/// Гол frame-д л ажиллана: Tauri-гийн initialization script нь дэд frame-д
/// хүрэхгүй бөгөөд доорх `window.top !== window` шалгалт нэмэлт хамгаалалт.
pub fn init_script(endpoints: &Endpoints) -> String {
    let config = json!({
        "version": CONTRACT_VERSION,
        "platform": platform(),
        "capabilities": capabilities(),
        "apiBase": endpoints.api_url,
    });
    let config_json = config.to_string();

    format!(
        r#"
(function () {{
  // Гэрээ зөвхөн ажлын мужид хамаарна. Дэд frame нь native чадваруудад
  // хүрэх шаардлагагүй.
  if (window.top !== window) {{ return; }}
  if (window.GeregeShell) {{ return; }}

  var config = JSON.parse({config});
  var internals = window.__TAURI_INTERNALS__;
  function ipc(command, payload) {{
    if (!internals || typeof internals.invoke !== 'function') {{
      return Promise.reject(new Error('shell: IPC боломжгүй'));
    }}
    return internals.invoke(command, payload);
  }}

  // Native тал бүх хариугаа энэ нэг цэгээр буцаана (гэрээний шаардлага #4).
  window.__geregeShellEmit = function (name, payloadJSON) {{
    var detail = null;
    try {{ detail = payloadJSON ? JSON.parse(payloadJSON) : null; }}
    catch (e) {{ return; }}
    window.dispatchEvent(new CustomEvent(name, {{ detail: detail }}));
  }};

  // Бүрхүүлийн дотоод суваг: session cookie нь HttpOnly бөгөөд API origin-д
  // харьяалагддаг тул түүнийг зөвхөн энэ webview өөрөө хүлээж авч чадна.
  // Гэрээний хэсэг БИШ — web app үүнийг дуудахгүй.
  window.__geregeShellFetch = function (requestJSON) {{
    var req;
    try {{ req = JSON.parse(requestJSON); }}
    catch (e) {{ return; }}
    var init = {{ method: req.method || 'GET', credentials: 'include', headers: req.headers || {{}} }};
    if (req.body !== null && req.body !== undefined) {{ init.body = req.body; }}
    fetch(config.apiBase + req.path, init)
      .then(function (res) {{
        return res.text().then(function (text) {{
          return {{ id: req.id, ok: true, status: res.status, body: text }};
        }});
      }})
      .catch(function (err) {{
        return {{ id: req.id, ok: false, status: 0, body: '', error: String((err && err.message) || err) }};
      }})
      .then(function (result) {{ ipc('shell_http_result', {{ result: result }}).catch(function () {{}}); }});
  }};

  // Цэсний шошгыг сервер орчуулдаг тул бүрхүүл хэрэглэгчийн сонгосон хэлийг
  // мэдэх ёстой. Web app хэлээ localStorage-д хадгалдаг — гэрээнд хэл дамжуулах
  // method байхгүй тул бүрхүүл өөрөө уншиж авна.
  try {{
    ipc('shell_locale', {{ locale: window.localStorage.getItem('locale') || 'mn' }})
      .catch(function () {{}});
  }} catch (e) {{ /* localStorage хаалттай байж болно */ }}

  window.GeregeShell = Object.freeze({{
    version: config.version,
    platform: config.platform,
    capabilities: Object.freeze(config.capabilities.slice()),
    invoke: function (method, params) {{
      return ipc('shell_invoke', {{ method: String(method), params: params || {{}} }});
    }},
    on: function (name, handler) {{
      var listener = function (event) {{ handler(event.detail); }};
      window.addEventListener(name, listener);
      return function () {{ window.removeEventListener(name, listener); }};
    }}
  }});
}})();
"#,
        config = js_string_literal(&config_json)
    )
}

/// Бүрхүүлээс ажлын муж руу event илгээнэ.
///
/// Tauri-гийн event системийг биш, гэрээний `__geregeShellEmit`-ийг ашиглаж
/// байгаа нь санаатай: macOS-ийн лавлагаа хэрэгжилттэй яг ижил зам, бөгөөд
/// remote хуудсанд Tauri-гийн JS API байхыг шаардахгүй.
pub fn emit(window: &WebviewWindow, event: &str, payload: Value) {
    let payload_json = payload.to_string();
    let script = format!(
        "window.__geregeShellEmit && window.__geregeShellEmit({}, {});",
        js_string_literal(event),
        js_string_literal(&payload_json)
    );
    if let Err(err) = window.eval(&script) {
        eprintln!("shell: event илгээж чадсангүй ({event}): {err}");
    }
}

/// Гол цонх руу event илгээнэ. Цонх байхгүй бол чимээгүй өнгөрнө — deep link
/// нь цонх үүсэхээс өмнө ирж болно.
pub fn emit_to_main(app: &AppHandle, event: &str, payload: Value) {
    if let Some(window) = app.get_webview_window(MAIN_WINDOW) {
        emit(&window, event, payload);
    }
}

/// `shell:navigate` — зөвхөн апп доторх зам. "//host" нь протокол-харьцангуй
/// гадаад хаяг тул хасагдана.
pub fn navigate(app: &AppHandle, path: &str) {
    if !path.starts_with('/') || path.starts_with("//") {
        return;
    }
    emit_to_main(app, EVENT_NAVIGATE, json!({ "path": path }));
}

pub fn search(app: &AppHandle, query: &str) {
    emit_to_main(app, EVENT_SEARCH, json!({ "query": query }));
}

pub fn menu_refresh(app: &AppHandle) {
    emit_to_main(app, EVENT_MENU_REFRESH, Value::Null);
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn literal_escapes_quotes_and_newlines() {
        assert_eq!(js_string_literal("a\"b"), "\"a\\\"b\"");
        assert_eq!(js_string_literal("a\nb"), "\"a\\nb\"");
    }

    #[test]
    fn literal_escapes_line_separators() {
        assert_eq!(js_string_literal("a\u{2028}b"), "\"a\\u2028b\"");
    }

    #[test]
    fn contract_does_not_claim_unimplemented_capabilities() {
        let declared = capabilities();
        assert!(!declared.contains(&"biometric"));
        assert!(!declared.contains(&"secure-store"));
    }
}
