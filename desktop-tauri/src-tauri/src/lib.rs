//! Open Gerege Nexus — Tauri v2 native bürkhüül.
//!
//! Архитектур: "Native Shell + Web Work Area". Бүрхүүл нь нэвтрэлт, цэс, tray,
//! төхөөрөмжийн хандалтыг эзэмшинэ; web app нь бүрхүүл дотор өөрийн chrome-оо
//! нуугаад зөвхөн ажлын муж болж рендерлэгдэнэ. Хоёрын хоорондох гэрээ нь
//! `docs/SHELL_CONTRACT.md` — өөрчлөгдөхгүй спецификац.

mod auth;
mod bridge;
mod commands;
mod config;
mod health;
mod menus;
mod shell;
mod tray;
mod windows;

use std::sync::Mutex;

use auth::AuthState;
use bridge::HttpBridge;
use config::ConfigState;
use health::HealthState;

/// Ажлын мужийн томруулалт. Tauri нь одоогийн утгыг уншиж өгдөггүй тул
/// бүрхүүл өөрөө тоолно.
pub struct ZoomState {
    value: Mutex<f64>,
}

impl ZoomState {
    fn new() -> Self {
        Self { value: Mutex::new(1.0) }
    }

    pub fn set(&self, next: f64) {
        let clamped = next.clamp(0.5, 3.0);
        match self.value.lock() {
            Ok(mut guard) => *guard = clamped,
            Err(poisoned) => *poisoned.into_inner() = clamped,
        }
    }

    pub fn step(&self, delta: f64) -> f64 {
        let current = match self.value.lock() {
            Ok(guard) => *guard,
            Err(poisoned) => *poisoned.into_inner(),
        };
        (current + delta).clamp(0.5, 3.0)
    }
}

pub struct AppState {
    pub config: ConfigState,
    pub bridge: HttpBridge,
    pub auth: AuthState,
    pub health: HealthState,
    pub zoom: ZoomState,
}

pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_window_state::Builder::default().build())
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_deep_link::init())
        // TODO(updater): tauri-plugin-updater-ийг Cargo.toml дотор нээж,
        // tauri.conf.json-ы plugins.updater дотор бодит endpoint ба minisign
        // pubkey бөглөсний дараа энд .plugin(tauri_plugin_updater::Builder::new().build())
        // нэмнэ. Түлхүүргүй updater нь гарын үсэггүй шинэчлэлт суулгах эрсдэлтэй
        // тул зориуд идэвхгүй үлдээв.
        .manage(AppState {
            config: ConfigState::load(),
            bridge: HttpBridge::new(),
            auth: AuthState::new(),
            health: HealthState::new(),
            zoom: ZoomState::new(),
        })
        .invoke_handler(tauri::generate_handler![
            commands::shell_invoke,
            commands::shell_http_result,
            commands::shell_locale,
            commands::auth_password_login,
            commands::auth_eid_start,
            commands::auth_cancel,
            commands::prefs_get,
            commands::prefs_set,
            commands::offline_retry,
            commands::open_preferences,
        ])
        .on_menu_event(|app, event| {
            menus::handle_event(app, event.id().as_ref());
        })
        .setup(|app| {
            let handle = app.handle().clone();

            // Ажлын муж эхлээд нуугдмал: нэвтрэлт бол бүрхүүлийн үүрэг.
            windows::create_main(&handle)?;

            if let Err(err) = tray::build(&handle) {
                eprintln!("setup: tray үүсгэж чадсангүй: {err}");
            }

            // Өмнөх session амьд бол шууд ажлын муж руу; үгүй бол нэвтрэлт.
            let startup = handle.clone();
            tauri::async_runtime::spawn(async move {
                if !auth::restore_session(&startup).await {
                    windows::open_login(&startup);
                    menus::schedule_rebuild(&startup);
                }
            });

            register_deep_links(&handle);
            health::start(handle);
            Ok(())
        })
        .run(tauri::generate_context!())
        .unwrap_or_else(|err| {
            // Энэ бол програм ажиллуулж чадаагүй цорын ганц цэг — үргэлжлүүлэх
            // боломжгүй тул зогсоохоос өөр сонголт байхгүй.
            eprintln!("Gerege Nexus эхэлж чадсангүй: {err}");
            std::process::exit(1);
        });
}

/// `gerege://apps/esign` → `shell:navigate {"path": "/apps/esign"}`.
fn register_deep_links(app: &tauri::AppHandle) {
    use tauri_plugin_deep_link::DeepLinkExt;

    let handle = app.clone();
    app.deep_link().on_open_url(move |event| {
        for url in event.urls() {
            let host = url.host_str().unwrap_or_default();
            let path = url.path();
            let target = if host.is_empty() {
                path.to_string()
            } else {
                format!("/{host}{path}")
            };
            windows::focus_main(&handle);
            shell::navigate(&handle, &target);
        }
    });
}
