//! IPC-ийн гадаргуу.
//!
//! Хоёр төрлийн дуудагч бий:
//!   * ажлын муж (remote origin) — зөвхөн `shell_invoke` ба `shell_http_result`;
//!   * бүрхүүлийн өөрийн локал цонхнууд — нэвтрэлт, тохиргооны командууд.

use base64::Engine;
use serde_json::{json, Value};
use tauri::{AppHandle, Manager, State};
use tauri_plugin_dialog::DialogExt;
use tauri_plugin_notification::NotificationExt;
use tauri_plugin_opener::OpenerExt;

use crate::bridge::HttpResult;
use crate::config::{is_dev_build, Endpoints};
use crate::shell::MAIN_WINDOW;
use crate::AppState;

/// Гэрээний `invoke()`. Дэмжигдээгүй method-ыг няцаах нь чухал: web тал
/// түүгээр өөрийн fallback-аа ажиллуулдаг.
#[tauri::command]
pub async fn shell_invoke(app: AppHandle, method: String, params: Value) -> Result<Value, String> {
    match method.as_str() {
        "notify.show" => {
            let title = params.get("title").and_then(Value::as_str).unwrap_or("Gerege Nexus");
            let body = params.get("body").and_then(Value::as_str).unwrap_or_default();
            app.notification()
                .builder()
                .title(title)
                .body(body)
                .show()
                .map_err(|err| err.to_string())?;
            Ok(Value::Null)
        }

        "badge.set" => {
            let count = params.get("count").and_then(Value::as_i64).unwrap_or(0);
            let window = app
                .get_webview_window(MAIN_WINDOW)
                .ok_or("badge.set: ажлын муж алга")?;
            // 0 нь тоолуурыг арилгана — гэрээний дагуу.
            let value = if count > 0 { Some(count) } else { None };
            window.set_badge_count(value).map_err(|err| err.to_string())?;
            Ok(Value::Null)
        }

        "external.open" => {
            let raw = params
                .get("url")
                .and_then(Value::as_str)
                .ok_or("external.open: url алга")?;
            // Гадаад хөтөч рүү юу дамжуулж байгаагаа хязгаарлана: file:// эсвэл
            // бүртгэгдсэн дурын scheme нь webview-гээс код ажиллуулах гарц.
            let allowed = ["http://", "https://", "mailto:", "tel:"]
                .iter()
                .any(|prefix| raw.to_ascii_lowercase().starts_with(prefix));
            if !allowed {
                return Err("external.open: зөвшөөрөгдөөгүй URL".into());
            }
            app.opener()
                .open_url(raw.to_string(), None::<&str>)
                .map_err(|err| err.to_string())?;
            Ok(Value::Null)
        }

        "print.system" => {
            let window = app
                .get_webview_window(MAIN_WINDOW)
                .ok_or("print.system: ажлын муж алга")?;
            window.print().map_err(|err| err.to_string())?;
            Ok(Value::Null)
        }

        "fs.saveAs" => save_as(app, &params).await,

        "menu.changed" => {
            crate::menus::schedule_rebuild(&app);
            Ok(Value::Null)
        }

        "auth.reLogin" => {
            // Нэвтрэлт бол бүрхүүлийн үүрэг: ажлын мужийг нуугаад native
            // цонхыг гаргана. Ажлын муж нь webview-д үлдэнэ — session тогтоох
            // хүсэлтүүд түүгээр дамжих ёстой.
            if let Some(main) = app.get_webview_window(MAIN_WINDOW) {
                if let Err(err) = main.hide() {
                    eprintln!("shell: ажлын мужийг нууж чадсангүй: {err}");
                }
            }
            crate::windows::open_login(&app);
            Ok(Value::Null)
        }

        other => Err(format!("shell: дэмжигдээгүй method — {other}")),
    }
}

/// Замыг web тал заахгүй: зөвхөн санал болгох нэр дамжуулна.
async fn save_as(app: AppHandle, params: &Value) -> Result<Value, String> {
    let suggested = params
        .get("filename")
        .and_then(Value::as_str)
        .unwrap_or("document");
    // Дамжуулсан нэрнээс зөвхөн сүүлийн бүрэлдэхүүнийг авна.
    let suggested = suggested
        .rsplit(['/', '\\'])
        .next()
        .filter(|value| !value.is_empty())
        .unwrap_or("document")
        .to_string();

    let payload = if let Some(encoded) = params.get("base64").and_then(Value::as_str) {
        base64::engine::general_purpose::STANDARD
            .decode(encoded)
            .map_err(|_| "fs.saveAs: base64 буруу".to_string())?
    } else if let Some(text) = params.get("text").and_then(Value::as_str) {
        text.as_bytes().to_vec()
    } else {
        return Err("fs.saveAs: агуулга алга".into());
    };

    let (sender, receiver) = tokio::sync::oneshot::channel();
    app.dialog()
        .file()
        .set_file_name(suggested)
        .save_file(move |chosen| {
            let _ = sender.send(chosen);
        });
    let chosen = receiver.await.map_err(|_| "fs.saveAs: цуцлагдсан".to_string())?;
    let path = chosen.ok_or("fs.saveAs: цуцлагдсан".to_string())?;
    let path = path.into_path().map_err(|err| err.to_string())?;
    std::fs::write(&path, payload).map_err(|err| err.to_string())?;
    Ok(json!({ "path": path.to_string_lossy() }))
}

/// Webview-ээс буцаж ирсэн HTTP хариу (`bridge.rs`).
#[tauri::command]
pub fn shell_http_result(state: State<'_, AppState>, result: HttpResult) {
    state.bridge.resolve(result);
}

/// Ажлын муж ямар хэл сонгосныг бүрхүүлд дуулгана — цэсний орчуулга түүнээс
/// хамаарна. Web тал `locale`-оо localStorage-д хадгалдаг.
#[tauri::command]
pub fn shell_locale(state: State<'_, AppState>, locale: String) {
    state.auth.set_locale(&locale);
    // Энэ дуудлага нь inject скрипт ажилласны шууд нотолгоо: түүнээс өмнө
    // гүүрээр хүсэлт илгээх нь утгагүй.
    state.bridge.mark_ready();
}

#[tauri::command]
pub async fn auth_password_login(app: AppHandle, email: String, password: String) -> Result<(), String> {
    crate::auth::password_login(app, email, password).await
}

#[tauri::command]
pub async fn auth_eid_start(app: AppHandle, national_id: Option<String>) -> Result<(), String> {
    crate::auth::eid_start(app, national_id).await
}

#[tauri::command]
pub fn auth_cancel(state: State<'_, AppState>) {
    state.auth.cancel();
}

#[tauri::command]
pub fn prefs_get(state: State<'_, AppState>) -> Value {
    let endpoints = state.config.get();
    json!({
        "api_url": endpoints.api_url,
        "web_url": endpoints.web_url,
        "editable": is_dev_build(),
    })
}

#[tauri::command]
pub fn prefs_set(app: AppHandle, api_url: String, web_url: String) -> Result<Value, String> {
    let saved = {
        let state = app.state::<AppState>();
        state.config.set(Endpoints { api_url, web_url })?
    };
    crate::windows::reload_main(&app);
    Ok(json!({ "api_url": saved.api_url, "web_url": saved.web_url }))
}

/// Offline цонхны "Дахин холбогдох".
#[tauri::command]
pub fn offline_retry(app: AppHandle) {
    crate::windows::reload_main(&app);
}

#[tauri::command]
pub fn open_preferences(app: AppHandle) {
    crate::windows::open_preferences(&app);
}

