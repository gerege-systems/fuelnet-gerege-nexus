//! Native цэс — тенантын цэсний мөрүүдээс баригдана.
//!
//! Цэсний эх сурвалж нь сервер: `GET /api/v1/menus` нь тухайн тенантад
//! суулгасан аппуудын мөрүүдийг RBAC-аар шүүж, `Accept-Language`-ийн дагуу
//! орчуулаад буцаана. Бүрхүүл өөрөө ямар апп байгааг мэдэхгүй — мэдэх ч
//! ёсгүй, учир нь апп бүр тенант тус бүрд асаж унтарна.

use serde::Deserialize;
use tauri::menu::{Menu, MenuBuilder, MenuItemBuilder, SubmenuBuilder};
use tauri::{AppHandle, Manager};

use crate::shell::MAIN_WINDOW;
use crate::AppState;

/// Цэсний мөрийг дарахад ажиллах үйлдлийг ID-д нь шифрлэнэ.
const NAV_PREFIX: &str = "nav:";
const ACTION_SEARCH: &str = "act:search";
const ACTION_RELOAD: &str = "act:reload";
const ACTION_PRINT: &str = "act:print";
const ACTION_ZOOM_IN: &str = "act:zoom-in";
const ACTION_ZOOM_OUT: &str = "act:zoom-out";
const ACTION_ZOOM_RESET: &str = "act:zoom-reset";
const ACTION_PREFS: &str = "act:prefs";
const ACTION_RELOGIN: &str = "act:relogin";

#[derive(Debug, Deserialize)]
pub struct MenuRow {
    pub id: String,
    #[serde(default)]
    pub app_id: Option<String>,
    #[serde(default)]
    pub app_name: Option<String>,
    #[serde(default)]
    pub parent_id: Option<String>,
    pub label: String,
    #[serde(default)]
    pub path: Option<String>,
    #[serde(default)]
    pub icon: String,
    #[serde(default)]
    pub order: i64,
}

/// Апп бүрийн цэс: эхний замтай мөр нь аппын үндсэн хуудас.
struct AppGroup {
    name: String,
    rows: Vec<MenuRow>,
}

/// Цэсийг сервертэй дахин тааруулж, native цэсийг шинээр барина.
pub async fn rebuild(app: &AppHandle) {
    let rows = match fetch(app).await {
        Ok(rows) => rows,
        Err(err) => {
            // Цэс татагдаагүй нь бүрхүүлийг зогсоох шалтгаан биш: суурь цэс
            // (гарах, дахин ачаалах, тохиргоо) байсаар байх ёстой.
            eprintln!("menus: цэс татагдсангүй: {err}");
            Vec::new()
        }
    };
    if let Err(err) = install(app, rows) {
        eprintln!("menus: цэс суулгаж чадсангүй: {err}");
    }
}

async fn fetch(app: &AppHandle) -> Result<Vec<MenuRow>, String> {
    let state = app.state::<AppState>();
    let Some(main) = app.get_webview_window(MAIN_WINDOW) else {
        return Err("ажлын муж бэлэн болоогүй".into());
    };
    let locale = state.auth.locale();
    let result = state
        .bridge
        .request(&main, "GET", "/api/v1/menus", None, &locale)
        .await?;
    if !result.is_success() {
        return Err(result.error_message());
    }
    serde_json::from_str(&result.body).map_err(|err| err.to_string())
}

/// Мөрүүдийг апп тус бүрээр бүлэглэнэ. Дараалал нь `order`, дараа нь ирсэн
/// дараалал — сервер аль хэдийн эрэмбэлсэн байдаг ч давхар баталгаа.
fn group(rows: Vec<MenuRow>) -> Vec<AppGroup> {
    let mut groups: Vec<AppGroup> = Vec::new();
    for row in rows {
        let Some(app_id) = row.app_id.clone() else {
            continue;
        };
        let name = row
            .app_name
            .clone()
            .filter(|value| !value.is_empty())
            .unwrap_or_else(|| app_id.clone());
        match groups.iter_mut().find(|group| {
            group
                .rows
                .first()
                .and_then(|first| first.app_id.as_deref())
                .map(|id| id == app_id)
                .unwrap_or(false)
        }) {
            Some(group) => group.rows.push(row),
            None => groups.push(AppGroup { name, rows: vec![row] }),
        }
    }
    for group in &mut groups {
        group.rows.sort_by_key(|row| row.order);
        // Аппын нэрийг эхний мөрийн шошгоор нэрлэнэ — Layout.tsx-ийн адил.
        if let Some(first) = group.rows.first() {
            if !first.label.is_empty() {
                group.name = first.label.clone();
            }
        }
    }
    groups
}

fn install(app: &AppHandle, rows: Vec<MenuRow>) -> tauri::Result<()> {
    let mut builder = MenuBuilder::new(app);

    // macOS дээр програмын цэс эхэнд байх ёстой; бусад платформ дээр энэ нь
    // ердийн "Файл" цэс болж харагдана.
    #[cfg(target_os = "macos")]
    {
        let app_menu = SubmenuBuilder::new(app, "Gerege Nexus")
            .about(None)
            .separator()
            .text(ACTION_PREFS, "Тохиргоо...")
            .text(ACTION_RELOGIN, "Дахин нэвтрэх...")
            .separator()
            .services()
            .separator()
            .hide()
            .hide_others()
            .show_all()
            .separator()
            .quit()
            .build()?;
        builder = builder.item(&app_menu);
    }

    let file_menu = SubmenuBuilder::new(app, "Файл")
        .text(ACTION_PRINT, "Хэвлэх / PDF болгох...")
        .separator()
        .close_window()
        .build()?;
    builder = builder.item(&file_menu);

    let edit_menu = SubmenuBuilder::new(app, "Засах")
        .undo()
        .redo()
        .separator()
        .cut()
        .copy()
        .paste()
        .select_all()
        .separator()
        .text(ACTION_SEARCH, "Хайх...")
        .build()?;
    builder = builder.item(&edit_menu);

    let view_menu = SubmenuBuilder::new(app, "Харах")
        .text(ACTION_RELOAD, "Дахин ачаалах")
        .separator()
        .text(ACTION_ZOOM_IN, "Томсгох")
        .text(ACTION_ZOOM_OUT, "Жижигсгэх")
        .text(ACTION_ZOOM_RESET, "Хэвийн хэмжээ")
        .build()?;
    builder = builder.item(&view_menu);

    // Платформын цэс — сервер юу зөвшөөрснөөр.
    let platform_menu = SubmenuBuilder::new(app, "Платформ")
        .item(&*nav_item(app, "/apps", "Апп Стор", "grid")?)
        .item(&*nav_item(app, "/settings/appearance", "Харагдац", "palette")?)
        .item(&*nav_item(app, "/settings/apps", "Суулгасан аппууд", "settings")?)
        .build()?;
    builder = builder.item(&platform_menu);

    for group in group(rows) {
        let mut submenu = SubmenuBuilder::new(app, &group.name);
        let roots: Vec<&MenuRow> = group.rows.iter().filter(|row| row.parent_id.is_none()).collect();
        for root in roots {
            let children: Vec<&MenuRow> = group
                .rows
                .iter()
                .filter(|row| row.parent_id.as_deref() == Some(root.id.as_str()) && row.path.is_some())
                .collect();
            if children.is_empty() {
                if let Some(path) = root.path.as_deref() {
                    submenu = submenu.item(&*nav_item(app, path, &root.label, &root.icon)?);
                }
                continue;
            }
            let mut child_menu = SubmenuBuilder::new(app, &root.label);
            for child in children {
                if let Some(path) = child.path.as_deref() {
                    child_menu = child_menu.item(&*nav_item(app, path, &child.label, &child.icon)?);
                }
            }
            submenu = submenu.item(&child_menu.build()?);
        }
        builder = builder.item(&submenu.build()?);
    }

    let menu: Menu<_> = builder.build()?;
    app.set_menu(menu)?;
    Ok(())
}

/// Цэсний нэг мөр. Дарахад `shell:navigate` явуулна.
///
/// Дүрс олдвол дүрстэй мөр, эс бөгөөс энгийн мөр буцаана — тиймээс төрөл нь
/// боксолсон `IsMenuItem`.
fn nav_item<R: tauri::Runtime>(
    app: &AppHandle<R>,
    path: &str,
    label: &str,
    icon: &str,
) -> tauri::Result<Box<dyn tauri::menu::IsMenuItem<R>>> {
    let id = format!("{NAV_PREFIX}{path}");
    // Түлхүүр товчлол нь зөвхөн байнга хэрэглэгддэг цөөн зам дээр.
    let accelerator = if path == "/apps" { Some("CmdOrCtrl+Shift+A") } else { None };

    #[cfg(target_os = "macos")]
    if let Some(native) = native_icon(icon) {
        let mut builder = tauri::menu::IconMenuItemBuilder::with_id(&id, label).native_icon(native);
        if let Some(accelerator) = accelerator {
            builder = builder.accelerator(accelerator);
        }
        return Ok(Box::new(builder.build(app)?));
    }
    let _ = icon;

    let mut builder = MenuItemBuilder::with_id(&id, label);
    if let Some(accelerator) = accelerator {
        builder = builder.accelerator(accelerator);
    }
    Ok(Box::new(builder.build(app)?))
}

/// Цэсний дүрсийн нэрийг платформын дүрс рүү буулгана.
///
/// macOS дээр л native дүрсний сан байдаг; Windows/Linux дээр Tauri-гийн цэс
/// дүрсийг зөвхөн растер зургаар авдаг тул нэр олдохгүй үед дүрсгүй үлдээх нь
/// хамгийн тогтвортой зан төлөв.
#[cfg(target_os = "macos")]
fn native_icon(name: &str) -> Option<tauri::menu::NativeIcon> {
    use tauri::menu::NativeIcon;
    Some(match name {
        "users" => NativeIcon::User,
        "folder" | "archive" | "files" => NativeIcon::Folder,
        "refresh-cw" => NativeIcon::Refresh,
        "settings" | "sliders" => NativeIcon::Advanced,
        "shield-check" | "key-round" => NativeIcon::LockLocked,
        "network" | "webhook" | "share-2" => NativeIcon::Network,
        _ => return None,
    })
}

/// Цэсний үйлдлүүдийг гүйцэтгэнэ.
pub fn handle_event(app: &AppHandle, id: &str) {
    if let Some(path) = id.strip_prefix(NAV_PREFIX) {
        crate::shell::navigate(app, path);
        crate::windows::focus_main(app);
        return;
    }
    match id {
        ACTION_SEARCH => {
            crate::windows::focus_main(app);
            // Хайх үгийг native талд цуглуулахгүй: ажлын муж өөрийн индекстэй
            // тул хайлтын UI-г нь нээгээд өгнө.
            crate::shell::search(app, "");
        }
        ACTION_RELOAD => {
            if let Some(main) = app.get_webview_window(MAIN_WINDOW) {
                if let Err(err) = main.eval("window.location.reload();") {
                    eprintln!("menus: дахин ачаалж чадсангүй: {err}");
                }
            }
        }
        ACTION_PRINT => {
            if let Some(main) = app.get_webview_window(MAIN_WINDOW) {
                if let Err(err) = main.print() {
                    eprintln!("menus: хэвлэх цонх нээгдсэнгүй: {err}");
                }
            }
        }
        ACTION_ZOOM_IN => zoom(app, 0.1),
        ACTION_ZOOM_OUT => zoom(app, -0.1),
        ACTION_ZOOM_RESET => set_zoom(app, 1.0),
        ACTION_PREFS => crate::windows::open_preferences(app),
        ACTION_RELOGIN => crate::windows::open_login(app),
        _ => {}
    }
}

/// Tauri нь одоогийн zoom-ыг уншиж өгдөггүй тул бүрхүүл өөрөө тоолно.
fn zoom(app: &AppHandle, delta: f64) {
    let state = app.state::<AppState>();
    let next = state.zoom.step(delta);
    set_zoom(app, next);
}

fn set_zoom(app: &AppHandle, value: f64) {
    let state = app.state::<AppState>();
    state.zoom.set(value);
    if let Some(main) = app.get_webview_window(MAIN_WINDOW) {
        if let Err(err) = main.set_zoom(value) {
            eprintln!("menus: zoom тохируулж чадсангүй: {err}");
        }
    }
}

/// Web талаас `menu.changed` ирэхэд цэсийг дахин татна.
pub fn schedule_rebuild(app: &AppHandle) {
    let app = app.clone();
    tauri::async_runtime::spawn(async move {
        rebuild(&app).await;
        crate::shell::menu_refresh(&app);
    });
}

/// Цэс, tray-д хэрэглэгдэх туслах: цэсний мөрөөс шууд шилжих зам.
pub fn quick_links() -> Vec<(&'static str, &'static str)> {
    vec![
        ("/apps", "Апп Стор"),
        ("/esign", "PDF E-Sign"),
        ("/billing", "Нэхэмжлэх"),
    ]
}
