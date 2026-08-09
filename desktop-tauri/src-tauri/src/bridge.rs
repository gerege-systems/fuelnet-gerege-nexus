//! API руу хийх хүсэлтийн тээвэр.
//!
//! Яагаад Rust-аас шууд `reqwest`-ээр биш, ажлын мужийн webview-гээр дамжуулж
//! байна вэ:
//!
//! Платформын session нь `session_token` HttpOnly cookie бөгөөд API-гийн
//! origin-д харьяалагддаг. Web app (`frontend/lib/api.ts`) зөвхөн `credentials:
//! "include"`-ээр ажилладаг — Authorization толгой ашигладаггүй. Тэгэхээр
//! cookie нь webview-ийн өөрийнх нь cookie store-д байх ёстой, түүнийг гаднаас
//! бичих API нь Tauri-д (болон wry-д) байхгүй. Cookie-г тэнд оруулах цорын ганц
//! зөв арга бол Set-Cookie-г тухайн webview өөрөө хүлээж авах — тиймээс
//! нэвтрэлтийн HTTP хүсэлтүүд түүгээр дамжина.
//!
//! Урсгалын логик (давталт, завсар, алдааны тэвчил) бүхэлдээ Rust талд байна;
//! webview зөвхөн тээвэр.

use std::collections::HashMap;
use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};
use std::sync::Mutex;
use std::time::Duration;

use serde::{Deserialize, Serialize};
use serde_json::json;
use tauri::WebviewWindow;
use tokio::sync::oneshot;

use crate::shell::js_string_literal;

/// Сервер нь /auth/eid/poll-ыг 25 секунд хүртэл нээлттэй барьдаг тул хүлээлт нь
/// түүнээс тодорхой урт байх ёстой.
const REQUEST_TIMEOUT: Duration = Duration::from_secs(40);
/// Ажлын муж ачаалагдахыг хүлээх дээд хугацаа. Хуудас ачаалагдтал inject
/// хийсэн скрипт байхгүй тул түүнээс өмнөх eval чимээгүй алга болж, хүсэлт
/// 40 секунд өлгөөстэй үлддэг байсан.
const READY_TIMEOUT: Duration = Duration::from_secs(20);
const READY_POLL: Duration = Duration::from_millis(50);

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct HttpResult {
    pub id: String,
    pub ok: bool,
    pub status: u16,
    #[serde(default)]
    pub body: String,
    #[serde(default)]
    pub error: Option<String>,
}

impl HttpResult {
    pub fn is_success(&self) -> bool {
        self.ok && (200..300).contains(&self.status)
    }

    /// Хариуны биеэс алдааны мессежийг гаргана. Backend `{"error": "..."}`
    /// хэлбэрээр буцаадаг (`httpx.Error`).
    pub fn error_message(&self) -> String {
        if let Some(err) = &self.error {
            return err.clone();
        }
        if let Ok(value) = serde_json::from_str::<serde_json::Value>(&self.body) {
            if let Some(message) = value.get("error").and_then(|v| v.as_str()) {
                return message.to_string();
            }
        }
        format!("Хүсэлт амжилтгүй ({})", self.status)
    }

    pub fn json(&self) -> Result<serde_json::Value, String> {
        serde_json::from_str(&self.body).map_err(|err| err.to_string())
    }
}

#[derive(Default)]
pub struct HttpBridge {
    pending: Mutex<HashMap<String, oneshot::Sender<HttpResult>>>,
    counter: AtomicU64,
    /// Ажлын мужийн inject скрипт ажилласан эсэх.
    ready: AtomicBool,
}

impl HttpBridge {
    pub fn new() -> Self {
        Self::default()
    }

    /// Inject скрипт өөрийгөө зарлахад дуудагдана.
    pub fn mark_ready(&self) {
        self.ready.store(true, Ordering::SeqCst);
    }

    /// Хуудас дахин ачаалагдахад скрипт дахин ажиллах хүртэл гүүр хаалттай.
    pub fn mark_not_ready(&self) {
        self.ready.store(false, Ordering::SeqCst);
    }

    async fn wait_ready(&self) -> bool {
        let deadline = tokio::time::Instant::now() + READY_TIMEOUT;
        while !self.ready.load(Ordering::SeqCst) {
            if tokio::time::Instant::now() >= deadline {
                return false;
            }
            tokio::time::sleep(READY_POLL).await;
        }
        true
    }

    /// Webview-ээр дамжуулан API руу хүсэлт илгээж, хариуг хүлээнэ.
    pub async fn request(
        &self,
        window: &WebviewWindow,
        method: &str,
        path: &str,
        body: Option<serde_json::Value>,
        locale: &str,
    ) -> Result<HttpResult, String> {
        if !self.wait_ready().await {
            return Err("bridge: ажлын муж бэлэн болсонгүй".into());
        }
        let id = format!("gh{}", self.counter.fetch_add(1, Ordering::Relaxed));
        let (sender, receiver) = oneshot::channel();

        match self.pending.lock() {
            Ok(mut pending) => {
                pending.insert(id.clone(), sender);
            }
            Err(_) => return Err("bridge: дотоод төлөв гэмтсэн".into()),
        }

        // Сервер өөрийн орчуулгыг Accept-Language-аар шийддэг тул цэсний
        // шошгууд бүрхүүлийн хэлтэй таарахын тулд толгойг зөв дамжуулна.
        let request = json!({
            "id": id,
            "method": method,
            "path": path,
            "headers": {
                "Content-Type": "application/json",
                "Accept-Language": locale,
            },
            "body": body.map(|value| value.to_string()),
        });

        let script = format!(
            "window.__geregeShellFetch && window.__geregeShellFetch({});",
            js_string_literal(&request.to_string())
        );
        if let Err(err) = window.eval(&script) {
            self.forget(&id);
            return Err(format!("bridge: хүсэлт илгээж чадсангүй: {err}"));
        }

        match tokio::time::timeout(REQUEST_TIMEOUT, receiver).await {
            Ok(Ok(result)) => Ok(result),
            Ok(Err(_)) => {
                self.forget(&id);
                Err("bridge: хариу тасарлаа".into())
            }
            Err(_) => {
                self.forget(&id);
                Err("bridge: хугацаа хэтэрлээ".into())
            }
        }
    }

    /// Webview-ээс ирсэн хариуг хүлээж байгаа хүсэлттэй холбоно.
    pub fn resolve(&self, result: HttpResult) {
        let sender = match self.pending.lock() {
            Ok(mut pending) => pending.remove(&result.id),
            Err(_) => None,
        };
        if let Some(sender) = sender {
            // Хүлээгч тал аль хэдийн явсан байж болно (цуцлагдсан нэвтрэлт).
            let _ = sender.send(result);
        }
    }

    fn forget(&self, id: &str) {
        if let Ok(mut pending) = self.pending.lock() {
            pending.remove(id);
        }
    }
}
