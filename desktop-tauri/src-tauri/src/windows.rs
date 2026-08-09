//! Цонхнууд ба навигацийн бодлого.

use tauri::{AppHandle, Manager, WebviewUrl, WebviewWindow, WebviewWindowBuilder};
use tauri_plugin_opener::OpenerExt;

use crate::auth::LOGIN_WINDOW;
use crate::config;
use crate::shell::MAIN_WINDOW;
use crate::AppState;

pub const OFFLINE_WINDOW: &str = "offline";
pub const PREFS_WINDOW: &str = "prefs";

const MAIN_WIDTH: f64 = 1280.0;
const MAIN_HEIGHT: f64 = 830.0;

/// Ажлын мужийн цонх.
///
/// Нуугдмал төлөвт үүсгэнэ: нэвтрэлт дуустал хэрэглэгч web-ийн нэвтрэх
/// хуудсыг харах ёсгүй — тэр ажлыг native login цонх хийнэ. Гэхдээ webview нь
/// ачаалагдсан байх шаардлагатай, учир нь session тогтоох HTTP хүсэлтүүд
/// түүгээр дамжина (`bridge.rs` дахь тайлбарыг үз).
pub fn create_main(app: &AppHandle) -> tauri::Result<WebviewWindow> {
    let endpoints = app.state::<AppState>().config.get();
    let url = endpoints
        .web_url
        .parse()
        .map_err(|_| tauri::Error::WebviewNotFound)?;

    let handle = app.clone();
    let web_base = endpoints.web_url.clone();

    WebviewWindowBuilder::new(app, MAIN_WINDOW, WebviewUrl::External(url))
        .title("Open Gerege Nexus")
        .inner_size(MAIN_WIDTH, MAIN_HEIGHT)
        .min_inner_size(900.0, 600.0)
        .visible(false)
        .initialization_script(crate::shell::init_script(&endpoints))
        .on_navigation(move |target| {
            let target = target.to_string();
            // Дотоод схемүүд — WebKit/WebView2 өөрсдөө үүсгэдэг хаягууд.
            if target.starts_with("about:") || target.starts_with("blob:") || target.starts_with("tauri://")
            {
                return true;
            }
            if config::same_origin(&target, &web_base) {
                return true;
            }
            // Гадны хуудас энэ webview-д ачаалагдвал манай session, гүүрийн
            // хажууд суух тул системийн хөтөч рүү гаргана.
            if let Err(err) = handle.opener().open_url(target.clone(), None::<&str>) {
                eprintln!("windows: гадаад хаяг нээгдсэнгүй ({target}): {err}");
            }
            false
        })
        .build()
}

/// Native нэвтрэлтийн цонх.
pub fn open_login(app: &AppHandle) {
    if let Some(window) = app.get_webview_window(LOGIN_WINDOW) {
        let _ = window.show();
        let _ = window.set_focus();
        return;
    }
    let result = WebviewWindowBuilder::new(app, LOGIN_WINDOW, WebviewUrl::App("login.html".into()))
        .title("Нэвтрэх — Gerege Nexus")
        .inner_size(460.0, 620.0)
        .resizable(false)
        .center()
        .build();
    if let Err(err) = result {
        eprintln!("windows: login цонх нээгдсэнгүй: {err}");
    }
}

/// Тохиргооны цонх. Production build-д хаягууд түгжээтэй тул зөвхөн уншина.
pub fn open_preferences(app: &AppHandle) {
    if let Some(window) = app.get_webview_window(PREFS_WINDOW) {
        let _ = window.show();
        let _ = window.set_focus();
        return;
    }
    let result = WebviewWindowBuilder::new(app, PREFS_WINDOW, WebviewUrl::App("prefs.html".into()))
        .title("Тохиргоо — Gerege Nexus")
        .inner_size(520.0, 380.0)
        .resizable(false)
        .center()
        .build();
    if let Err(err) = result {
        eprintln!("windows: тохиргооны цонх нээгдсэнгүй: {err}");
    }
}

/// Сервер холбогдоогүй үеийн цонх.
///
/// Native alert биш: alert нь дарагдмагц алга болдог бөгөөд юу хийхээ хэлдэггүй.
/// Swift бүрхүүлийн offline дэлгэцтэй ижил санаа — юу ажиллах ёстойг тайлбарлаж,
/// дахин холбогдох, тохиргоо гэсэн хоёр гарц өгнө.
fn open_offline(app: &AppHandle) {
    if let Some(window) = app.get_webview_window(OFFLINE_WINDOW) {
        let _ = window.show();
        return;
    }
    let result = WebviewWindowBuilder::new(app, OFFLINE_WINDOW, WebviewUrl::App("offline.html".into()))
        .title("Холболт — Gerege Nexus")
        .inner_size(480.0, 320.0)
        .resizable(false)
        .center()
        .build();
    if let Err(err) = result {
        eprintln!("windows: offline цонх нээгдсэнгүй: {err}");
    }
}

fn close_offline(app: &AppHandle) {
    if let Some(window) = app.get_webview_window(OFFLINE_WINDOW) {
        if let Err(err) = window.close() {
            eprintln!("windows: offline цонх хаагдсангүй: {err}");
        }
    }
}

/// Health хяналтын үр дүнг цонхнуудад тусгана.
pub fn reflect_health(app: &AppHandle, api_ok: bool, web_ok: bool) {
    if api_ok && web_ok {
        close_offline(app);
        return;
    }
    open_offline(app);
}

/// Ажлын мужийг урд гаргана. Нэвтрээгүй бол login цонхыг гаргана — нэвтрэлт
/// бол бүрхүүлийн үүрэг.
pub fn focus_main(app: &AppHandle) {
    if let Some(login) = app.get_webview_window(LOGIN_WINDOW) {
        let _ = login.show();
        let _ = login.set_focus();
        return;
    }
    if let Some(main) = app.get_webview_window(MAIN_WINDOW) {
        let _ = main.show();
        let _ = main.set_focus();
    }
}

/// Ажлын мужийг тохиргоонд заасан Web хаяг руу дахин ачаална.
pub fn reload_main(app: &AppHandle) {
    let endpoints = app.state::<AppState>().config.get();
    if let Some(main) = app.get_webview_window(MAIN_WINDOW) {
        let script = format!(
            "window.location.replace({});",
            crate::shell::js_string_literal(&endpoints.web("/"))
        );
        if let Err(err) = main.eval(&script) {
            eprintln!("windows: дахин ачаалж чадсангүй: {err}");
        }
    }
}
