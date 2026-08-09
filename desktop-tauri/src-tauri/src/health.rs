//! Серверийн эрүүл мэндийн хяналт.
//!
//! 5 секунд тутам API ба Web серверийг татаж, амьд эсэхийг шалгана.
//!
//! API-гийн эрүүл мэндийн хаяг нь `/health` (`server.go`) — өмнө нь энд
//! `/healthz` бичигдсэн байсан бөгөөд тэр нь 404 буцаадаг. 404-ийг "хүрч
//! байна" гэж тоолдог байсан тул алдаа нь нуугдаж, шалгалт нь бодитоор зөвхөн
//! порт нээлттэй эсэхийг хардаг байв. Одоо API-гаас 2xx шаардана.
//!
//! Web талд бол статус нь чухал биш: Next.js-ийн үндсэн хаяг 404 буцааж болох
//! ч сервер нь амьд. Тэнд хариу ирсэн эсэх нь л хангалттай.
//!
//! Энэ шалгалт нь webview-гээр биш, Rust-аас шууд явна: ажлын муж ачаалагдаж
//! чадаагүй үед л энэ мэдээлэл хамгийн их хэрэгтэй байдаг.

use std::sync::atomic::{AtomicBool, Ordering};
use std::time::Duration;

use tauri::AppHandle;

const INTERVAL: Duration = Duration::from_secs(5);
const PROBE_TIMEOUT: Duration = Duration::from_millis(2500);

#[derive(Default)]
pub struct HealthState {
    api_ok: AtomicBool,
    web_ok: AtomicBool,
}

impl HealthState {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn snapshot(&self) -> (bool, bool) {
        (
            self.api_ok.load(Ordering::Relaxed),
            self.web_ok.load(Ordering::Relaxed),
        )
    }

    fn store(&self, api_ok: bool, web_ok: bool) {
        self.api_ok.store(api_ok, Ordering::Relaxed);
        self.web_ok.store(web_ok, Ordering::Relaxed);
    }
}

/// Хяналтыг эхлүүлнэ. Төлөв өөрчлөгдөх бүрд tray-гийн зөвлөмж ба offline
/// цонхны харагдац шинэчлэгдэнэ.
pub fn start(app: AppHandle) {
    tauri::async_runtime::spawn(async move {
        let client = match reqwest::Client::builder().timeout(PROBE_TIMEOUT).build() {
            Ok(client) => client,
            Err(err) => {
                eprintln!("health: HTTP клиент үүсгэж чадсангүй: {err}");
                return;
            }
        };
        let mut previous: Option<(bool, bool)> = None;
        loop {
            let endpoints = {
                let state = tauri::Manager::state::<crate::AppState>(&app);
                state.config.get()
            };
            let api_ok = probe(&client, &endpoints.api("/health"), true).await;
            let web_ok = probe(&client, &endpoints.web_url, false).await;

            {
                let state = tauri::Manager::state::<crate::AppState>(&app);
                state.health.store(api_ok, web_ok);
            }

            if previous != Some((api_ok, web_ok)) {
                previous = Some((api_ok, web_ok));
                crate::tray::refresh(&app, api_ok, web_ok);
                crate::windows::reflect_health(&app, api_ok, web_ok);
            }

            tokio::time::sleep(INTERVAL).await;
        }
    });
}

/// `strict` үед зөвхөн 2xx-ийг хүлээн авна; эс бөгөөс хариу ирсэн нь хангалттай.
async fn probe(client: &reqwest::Client, url: &str, strict: bool) -> bool {
    match client.get(url).send().await {
        Ok(response) => {
            if strict {
                response.status().is_success()
            } else {
                response.status().as_u16() < 500
            }
        }
        Err(_) => false,
    }
}
