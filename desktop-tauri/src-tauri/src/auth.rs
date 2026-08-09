//! Native нэвтрэлт: имэйл/нууц үг ба eID.
//!
//! eID-ийн long-poll урсгалын логик нь `frontend/components/EIDLogin.tsx`-ийн
//! давталттай яг ижил утгатай: сервер хүсэлтийг 25 секунд хүртэл нээлттэй
//! барьдаг тул нэг зэрэг ганцхан хүсэлт явуулж, хооронд нь богино завсар
//! барина; тасарсан холболт мобайл сүлжээнд энгийн зүйл тул хэдэн удаа
//! тэвчинэ.

use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Mutex;
use std::time::{Duration, Instant};

use serde::Serialize;
use serde_json::json;
use tauri::{AppHandle, Manager};

use crate::bridge::HttpResult;
use crate::shell::{js_string_literal, MAIN_WINDOW};
use crate::AppState;

/// Хоёр шалгалтын хоорондох завсар. 25 секундын нээлттэй хүсэлтийн хажууд энэ
/// нь зөвшөөрөл өгснөөс нэвтрэх хүртэлх цорын ганц саатал тул богино байна.
const GAP: Duration = Duration::from_millis(400);
/// Дараалсан хэдэн алдааг тэвчих вэ.
const TOLERATED_FAILURES: u32 = 3;
/// Хугацааны хязгаар биш, зогсоох баталгаа: eID сесс хэзээ үхэхээ өөрөө хэлдэг
/// бөгөөд тэр хариу ирэхгүй тохиолдолд л энэ давталтыг зогсооно.
const BACKSTOP: Duration = Duration::from_secs(15 * 60);

pub const LOGIN_WINDOW: &str = "login";

#[derive(Default)]
pub struct AuthState {
    /// Оролдлого бүр тасалбар авна. Асинхрон ажил өөрийн тасалбарыг шалгасны
    /// дараа л төлөв өөрчилдөг тул цуцлагдсан оролдлого сэргэхгүй.
    ticket: AtomicU64,
    locale: Mutex<String>,
}

impl AuthState {
    pub fn new() -> Self {
        Self {
            ticket: AtomicU64::new(0),
            locale: Mutex::new("mn".to_string()),
        }
    }

    /// Шинэ тасалбар олгож, өмнөх оролдлогуудыг хүчингүй болгоно.
    fn take_ticket(&self) -> u64 {
        self.ticket.fetch_add(1, Ordering::SeqCst) + 1
    }

    fn is_current(&self, ticket: u64) -> bool {
        self.ticket.load(Ordering::SeqCst) == ticket
    }

    pub fn cancel(&self) {
        self.ticket.fetch_add(1, Ordering::SeqCst);
    }

    pub fn locale(&self) -> String {
        match self.locale.lock() {
            Ok(guard) => guard.clone(),
            Err(poisoned) => poisoned.into_inner().clone(),
        }
    }

    pub fn set_locale(&self, locale: &str) {
        let value = if locale.trim().is_empty() { "mn" } else { locale.trim() };
        match self.locale.lock() {
            Ok(mut guard) => *guard = value.to_string(),
            Err(poisoned) => *poisoned.into_inner() = value.to_string(),
        }
    }
}

/// Login цонх руу илгээх төлөв.
#[derive(Serialize)]
pub struct LoginUpdate {
    pub phase: &'static str,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub qr_svg: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub verification_code: Option<String>,
}

impl LoginUpdate {
    pub fn phase(phase: &'static str) -> Self {
        Self { phase, message: None, qr_svg: None, verification_code: None }
    }

    pub fn error(message: impl Into<String>) -> Self {
        Self {
            phase: "error",
            message: Some(message.into()),
            qr_svg: None,
            verification_code: None,
        }
    }
}

/// Login цонхны төлвийг шинэчилнэ. Гэрээний адил JSON-оор дамжина.
pub fn push_login_update(app: &AppHandle, update: &LoginUpdate) {
    let Some(window) = app.get_webview_window(LOGIN_WINDOW) else {
        return;
    };
    let payload = match serde_json::to_string(update) {
        Ok(payload) => payload,
        Err(err) => {
            eprintln!("auth: төлөв цуваа болгож чадсангүй: {err}");
            return;
        }
    };
    let script = format!(
        "window.__geregeLogin && window.__geregeLogin({});",
        js_string_literal(&payload)
    );
    if let Err(err) = window.eval(&script) {
        eprintln!("auth: login цонх руу төлөв илгээж чадсангүй: {err}");
    }
}

/// eID-ийн `device_link_url`-ыг QR болгоно. Login цонх нь ямар ч JS сангаас
/// хамаарахгүй байхын тулд зургийг Rust талд SVG болгож өгнө.
fn qr_svg(value: &str) -> Option<String> {
    use qrcode::render::svg;
    let code = qrcode::QrCode::with_error_correction_level(value, qrcode::EcLevel::M).ok()?;
    Some(
        code.render::<svg::Color>()
            .min_dimensions(190, 190)
            .quiet_zone(true)
            .build(),
    )
}

/// Нэвтрэлт амжилттай болсны дараах шилжилт.
///
/// Session cookie нь webview дотор аль хэдийн байрлачихсан (хүсэлтийг тэр
/// өөрөө хийсэн) тул энд зөвхөн цонхнуудыг солиод ажлын мужийг /apps руу
/// чиглүүлнэ.
async fn finish_login(app: &AppHandle) {
    if let Some(login) = app.get_webview_window(LOGIN_WINDOW) {
        if let Err(err) = login.close() {
            eprintln!("auth: login цонх хаагдсангүй: {err}");
        }
    }
    if let Some(main) = app.get_webview_window(MAIN_WINDOW) {
        if let Err(err) = main.show() {
            eprintln!("auth: гол цонх нээгдсэнгүй: {err}");
        }
        if let Err(err) = main.set_focus() {
            eprintln!("auth: гол цонх фокус авсангүй: {err}");
        }
    }
    crate::shell::navigate(app, "/apps");
    crate::menus::rebuild(app).await;
}

/// Имэйл/нууц үгээр нэвтрэх.
pub async fn password_login(app: AppHandle, email: String, password: String) -> Result<(), String> {
    let state = app.state::<AppState>();
    let ticket = state.auth.take_ticket();
    push_login_update(&app, &LoginUpdate::phase("starting"));

    let Some(main) = app.get_webview_window(MAIN_WINDOW) else {
        return Err("Ажлын муж бэлэн болоогүй байна".into());
    };
    let locale = state.auth.locale();
    let result = state
        .bridge
        .request(
            &main,
            "POST",
            "/api/v1/auth/login",
            Some(json!({ "email": email, "password": password })),
            &locale,
        )
        .await?;

    if !state.auth.is_current(ticket) {
        return Ok(());
    }
    if !result.is_success() {
        let message = result.error_message();
        push_login_update(&app, &LoginUpdate::error(message.clone()));
        return Err(message);
    }
    finish_login(&app).await;
    Ok(())
}

/// eID урсгалыг эхлүүлээд хяналтын давталтыг ажиллуулна.
///
/// `national_id` хоосон бол QR (push) хувилбар.
pub async fn eid_start(app: AppHandle, national_id: Option<String>) -> Result<(), String> {
    let state = app.state::<AppState>();
    let ticket = state.auth.take_ticket();
    push_login_update(&app, &LoginUpdate::phase("starting"));

    let Some(main) = app.get_webview_window(MAIN_WINDOW) else {
        return Err("Ажлын муж бэлэн болоогүй байна".into());
    };
    let locale = state.auth.locale();

    // callbackUrl нь хөтчийн буцах хаягт зориулагдсан. Бүрхүүл дотор буцах
    // хуудас байхгүй тул хоосон явуулна — сервер энэ талбарыг сонголтоор
    // авдаг.
    let (path, body) = match national_id.as_deref().map(str::trim).filter(|v| !v.is_empty()) {
        Some(id) => (
            "/api/v1/auth/eid/start-id",
            json!({ "national_id": id.to_uppercase(), "callbackUrl": "" }),
        ),
        None => ("/api/v1/auth/eid/start", json!({ "callbackUrl": "" })),
    };

    let started = state.bridge.request(&main, "POST", path, Some(body), &locale).await?;
    if !state.auth.is_current(ticket) {
        return Ok(());
    }
    if !started.is_success() {
        let message = started.error_message();
        push_login_update(&app, &LoginUpdate::error(message.clone()));
        return Err(message);
    }

    let payload = started.json()?;
    let session_id = payload
        .get("session_id")
        .and_then(|v| v.as_str())
        .unwrap_or_default()
        .to_string();
    if session_id.is_empty() {
        let message = "eID сесс эхлээгүй байна".to_string();
        push_login_update(&app, &LoginUpdate::error(message.clone()));
        return Err(message);
    }

    push_login_update(
        &app,
        &LoginUpdate {
            phase: "waiting",
            message: None,
            qr_svg: payload
                .get("device_link_url")
                .and_then(|v| v.as_str())
                .and_then(qr_svg),
            verification_code: payload
                .get("verification_code")
                .and_then(|v| v.as_str())
                .map(str::to_string),
        },
    );

    let app_for_watch = app.clone();
    tauri::async_runtime::spawn(async move {
        watch_eid(app_for_watch, session_id, ticket).await;
    });
    Ok(())
}

/// Нэг удаад нэг шалгалт. Интервалаар давталт хийвэл нээлттэй хүсэлтүүд
/// овоолж, хостын холболтын хязгаарыг барьдаг.
async fn watch_eid(app: AppHandle, session_id: String, ticket: u64) {
    let deadline = Instant::now() + BACKSTOP;
    let mut failures = 0u32;

    loop {
        let state = app.state::<AppState>();
        if !state.auth.is_current(ticket) {
            return;
        }
        if Instant::now() >= deadline {
            push_login_update(&app, &LoginUpdate::phase("expired"));
            return;
        }
        let Some(main) = app.get_webview_window(MAIN_WINDOW) else {
            push_login_update(&app, &LoginUpdate::error("Ажлын муж хаагдсан байна"));
            return;
        };
        let locale = state.auth.locale();

        let outcome = state
            .bridge
            .request(
                &main,
                "POST",
                "/api/v1/auth/eid/poll",
                Some(json!({ "session_id": session_id })),
                &locale,
            )
            .await;

        if !state.auth.is_current(ticket) {
            return;
        }

        match outcome {
            Ok(result) if result.is_success() => {
                failures = 0;
                if let Some(next) = handle_poll_state(&app, &result, ticket).await {
                    if next {
                        return;
                    }
                } else {
                    return;
                }
            }
            Ok(result) => {
                // Сервер нь баталгаажаагүй иргэнийг 401-ээр хэлдэг: энэ бол
                // сүлжээний алдаа биш, эцсийн хариу.
                push_login_update(&app, &LoginUpdate::error(result.error_message()));
                return;
            }
            Err(err) => {
                failures += 1;
                if failures > TOLERATED_FAILURES {
                    push_login_update(&app, &LoginUpdate::error(err));
                    return;
                }
            }
        }

        tokio::time::sleep(GAP).await;
    }
}

/// Poll-ын хариуг тайлбарлана.
///
/// `None` — давталт дуусах ёстой; `Some(false)` — үргэлжилнэ; `Some(true)` —
/// амжилттай дууссан.
async fn handle_poll_state(app: &AppHandle, result: &HttpResult, ticket: u64) -> Option<bool> {
    let payload = result.json().ok()?;
    let state_name = payload.get("state").and_then(|v| v.as_str()).unwrap_or_default();
    match state_name {
        "COMPLETE" => {
            let state = app.state::<AppState>();
            if !state.auth.is_current(ticket) {
                return Some(true);
            }
            // Дахин ямар нэг оролдлого энэ урсгалыг сэргээхгүй байхын тулд
            // тасалбарыг урагшлуулна.
            state.auth.cancel();
            push_login_update(app, &LoginUpdate::phase("success"));
            finish_login(app).await;
            Some(true)
        }
        "EXPIRED" => {
            push_login_update(app, &LoginUpdate::phase("expired"));
            None
        }
        "REFUSED" => {
            push_login_update(app, &LoginUpdate::phase("refused"));
            None
        }
        _ => Some(false),
    }
}

/// Өмнөх session амьд эсэхийг шалгана.
///
/// Cookie нь webview-ийн байнгын санд үлддэг тул аппыг дахин нээх бүрд
/// нэвтрэх шаардлагагүй. Хариу 200 бол ажлын мужийг шууд гаргана.
pub async fn restore_session(app: &AppHandle) -> bool {
    let state = app.state::<AppState>();
    let Some(main) = app.get_webview_window(MAIN_WINDOW) else {
        return false;
    };
    let locale = state.auth.locale();
    let Ok(result) = state
        .bridge
        .request(&main, "GET", "/api/v1/auth/me", None, &locale)
        .await
    else {
        return false;
    };
    if !result.is_success() {
        return false;
    }
    if let Err(err) = main.show() {
        eprintln!("auth: гол цонх нээгдсэнгүй: {err}");
    }
    // Цонх нь Web URL-ын үндэс дээр ачаалагддаг бөгөөд тэр нь нэвтрэлтийн
    // landing хуудас. Session амьд байхад хэрэглэгчийг тэнд үлдээх нь
    // "нэвтэрсэн атлаа нэвтрэх хуудас харах" болно.
    crate::shell::navigate(app, "/apps");
    crate::menus::rebuild(app).await;
    true
}
