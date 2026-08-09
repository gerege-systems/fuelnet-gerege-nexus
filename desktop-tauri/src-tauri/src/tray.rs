//! Tray дүрс.
//!
//! Swift бүрхүүл нь цонхны доод талд статус мөр зурдаг. Tauri-д native статус
//! мөр байхгүй бөгөөд ажлын мужийн доор HTML мөр нэмэх нь гэрээг зөрчинө
//! (бүрхүүл web-ийн зурагт хөндлөнгөөс оролцох болно). Тиймээс серверийн
//! төлөв энд — tray дүрсний зөвлөмж дээр — амьдарна.

use tauri::menu::{MenuBuilder, SubmenuBuilder};
use tauri::tray::{TrayIconBuilder, TrayIconEvent};
use tauri::AppHandle;

pub const TRAY_ID: &str = "gerege-tray";

const TRAY_OPEN: &str = "tray:open";
const TRAY_PREFS: &str = "tray:prefs";
const TRAY_QUIT: &str = "tray:quit";
const TRAY_NAV_PREFIX: &str = "tray-nav:";

pub fn build(app: &AppHandle) -> tauri::Result<()> {
    let mut quick = SubmenuBuilder::new(app, "Түргэн шилжих");
    for (path, label) in crate::menus::quick_links() {
        quick = quick.text(format!("{TRAY_NAV_PREFIX}{path}"), label);
    }

    let menu = MenuBuilder::new(app)
        .text(TRAY_OPEN, "Open Gerege Nexus")
        .separator()
        .item(&quick.build()?)
        .separator()
        .text(TRAY_PREFS, "Тохиргоо...")
        .text(TRAY_QUIT, "Гарах")
        .build()?;

    let icon = app.default_window_icon().cloned();
    let mut builder = TrayIconBuilder::with_id(TRAY_ID)
        .menu(&menu)
        .tooltip("Серверийн холболт шалгаж байна...")
        .show_menu_on_left_click(false)
        .on_menu_event(|app, event| handle_menu(app, event.id().as_ref()))
        .on_tray_icon_event(|tray, event| {
            // Зүүн товшилт нь цонхыг гаргана — цэс нь баруун товшилт дээр.
            if let TrayIconEvent::Click { button, .. } = event {
                if button == tauri::tray::MouseButton::Left {
                    crate::windows::focus_main(tray.app_handle());
                }
            }
        });
    if let Some(icon) = icon {
        builder = builder.icon(icon);
    }
    builder.build(app)?;
    Ok(())
}

fn handle_menu(app: &AppHandle, id: &str) {
    if let Some(path) = id.strip_prefix(TRAY_NAV_PREFIX) {
        crate::windows::focus_main(app);
        crate::shell::navigate(app, path);
        return;
    }
    match id {
        TRAY_OPEN => crate::windows::focus_main(app),
        TRAY_PREFS => crate::windows::open_preferences(app),
        TRAY_QUIT => app.exit(0),
        _ => {}
    }
}

/// Серверийн төлвийг зөвлөмж болгож харуулна.
pub fn refresh(app: &AppHandle, api_ok: bool, web_ok: bool) {
    let Some(tray) = app.tray_by_id(TRAY_ID) else {
        return;
    };
    let tooltip = match (api_ok, web_ok) {
        (true, true) => "Сервер хэвийн холбогдсон".to_string(),
        (false, true) => "API сервер холбогдоогүй".to_string(),
        (true, false) => "Web сервер холбогдоогүй".to_string(),
        (false, false) => "Сервер холбогдоогүй".to_string(),
    };
    if let Err(err) = tray.set_tooltip(Some(&tooltip)) {
        eprintln!("tray: зөвлөмж тавьж чадсангүй: {err}");
    }
}
