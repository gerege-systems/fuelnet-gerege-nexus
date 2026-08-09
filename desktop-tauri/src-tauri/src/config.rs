//! Сервэрийн хаягуудын шийдэл.
//!
//! Production build-д хаягууд нь compile-time тогтмол: суулгасан хэрэглэгч
//! бүрхүүлээ өөр сервер рүү чиглүүлэх боломжгүй байх нь аюулгүй байдлын
//! шаардлага — навигацийн allowlist, capability-ийн remote origin бүгд эдгээр
//! хаяг дээр тулгуурладаг. Dev build-д л Preferences цонхоор солигдоно.

use std::path::PathBuf;
use std::sync::Mutex;

pub const DEFAULT_API_URL: &str = "http://localhost:8080";
pub const DEFAULT_WEB_URL: &str = "http://localhost:3000";

/// Build хийхдээ `GEREGE_API_URL=... cargo build` гэж дамжуулна.
pub fn compiled_api_url() -> &'static str {
    option_env!("GEREGE_API_URL").unwrap_or(DEFAULT_API_URL)
}

pub fn compiled_web_url() -> &'static str {
    option_env!("GEREGE_WEB_URL").unwrap_or(DEFAULT_WEB_URL)
}

/// Dev build мөн эсэх. Тохиргоог зөвхөн энэ горимд солиж болно.
pub fn is_dev_build() -> bool {
    cfg!(debug_assertions)
}

#[derive(Clone, Debug, serde::Serialize, serde::Deserialize)]
pub struct Endpoints {
    pub api_url: String,
    pub web_url: String,
}

impl Default for Endpoints {
    fn default() -> Self {
        Self {
            api_url: compiled_api_url().to_string(),
            web_url: compiled_web_url().to_string(),
        }
    }
}

impl Endpoints {
    /// Сүүлийн налуу зураасыг хасна — хаягуудыг залгахад давхар "//" гарахгүй.
    fn normalized(mut self) -> Self {
        while self.api_url.ends_with('/') {
            self.api_url.pop();
        }
        while self.web_url.ends_with('/') {
            self.web_url.pop();
        }
        self
    }

    pub fn api(&self, path: &str) -> String {
        format!("{}{}", self.api_url, path)
    }

    pub fn web(&self, path: &str) -> String {
        format!("{}{}", self.web_url, path)
    }
}

/// Ажиллаж байх хугацааны тохиргоо. Tauri-гийн `State` болж хадгалагдана.
pub struct ConfigState {
    inner: Mutex<Endpoints>,
}

impl ConfigState {
    pub fn load() -> Self {
        // Production-д файлыг огт уншихгүй: диск дээрх файл нь бүрхүүлийг
        // танихгүй сервер рүү чиглүүлэх хамгийн хялбар арга байх байсан.
        let endpoints = if is_dev_build() {
            read_prefs_file().unwrap_or_default()
        } else {
            Endpoints::default()
        };
        Self {
            inner: Mutex::new(endpoints.normalized()),
        }
    }

    /// Тохиргооны хуулбар. Mutex хордсон (panic-д өртсөн) тохиолдолд ч
    /// бүрхүүл ажиллах ёстой тул compile-time утга руу ухарна.
    pub fn get(&self) -> Endpoints {
        match self.inner.lock() {
            Ok(guard) => guard.clone(),
            Err(poisoned) => poisoned.into_inner().clone(),
        }
    }

    /// Dev build дээр л амжилттай болно. Буцаах утга нь хадгалагдсан эсэх.
    pub fn set(&self, next: Endpoints) -> Result<Endpoints, String> {
        if !is_dev_build() {
            return Err("Тохиргоо зөвхөн хөгжүүлэлтийн build дээр солигдоно".into());
        }
        let next = next.normalized();
        validate(&next)?;
        match self.inner.lock() {
            Ok(mut guard) => *guard = next.clone(),
            Err(poisoned) => *poisoned.into_inner() = next.clone(),
        }
        write_prefs_file(&next)?;
        Ok(next)
    }
}

fn validate(endpoints: &Endpoints) -> Result<(), String> {
    for url in [&endpoints.api_url, &endpoints.web_url] {
        let lowered = url.to_ascii_lowercase();
        if !lowered.starts_with("http://") && !lowered.starts_with("https://") {
            return Err(format!("Хаяг http эсвэл https байх ёстой: {url}"));
        }
    }
    Ok(())
}

fn prefs_path() -> Option<PathBuf> {
    dirs::config_dir().map(|dir| dir.join("mn.gerege.nexus.tauri").join("prefs.json"))
}

fn read_prefs_file() -> Option<Endpoints> {
    let path = prefs_path()?;
    let raw = std::fs::read_to_string(path).ok()?;
    serde_json::from_str(&raw).ok()
}

fn write_prefs_file(endpoints: &Endpoints) -> Result<(), String> {
    let path = prefs_path().ok_or("Тохиргооны хавтас олдсонгүй")?;
    if let Some(parent) = path.parent() {
        std::fs::create_dir_all(parent).map_err(|err| err.to_string())?;
    }
    let body = serde_json::to_string_pretty(endpoints).map_err(|err| err.to_string())?;
    std::fs::write(path, body).map_err(|err| err.to_string())
}

/// "scheme://host:port" — жиших боломжтой origin. Стандарт порт бичигдсэн
/// эсэхээс үл хамааран ижил утга буцаана (desktop-mac-ийн ServerManager-тэй
/// ижил дүрэм).
pub fn origin_of(url: &str) -> Option<String> {
    let (scheme, rest) = url.split_once("://")?;
    let scheme = scheme.to_ascii_lowercase();
    if scheme != "http" && scheme != "https" {
        return None;
    }
    // Замын хэсэг, query, fragment-ийг хаяна.
    let authority = rest
        .split(['/', '?', '#'])
        .next()
        .unwrap_or_default()
        .to_ascii_lowercase();
    if authority.is_empty() {
        return None;
    }
    // Хэрэглэгчийн нэр бүхий хаяг (user@host) — host хэсгийг л авна.
    let authority = authority.rsplit('@').next().unwrap_or(&authority).to_string();
    let default_port = if scheme == "https" { 443 } else { 80 };
    let (host, port) = match authority.rsplit_once(':') {
        // IPv6 ([::1]) дотор давхар цэг байдаг тул хаалт хаагдсан эсэхийг хардана.
        Some((host, port)) if !host.ends_with(']') => match port.parse::<u16>() {
            Ok(parsed) => (host.to_string(), parsed),
            Err(_) => (authority.clone(), default_port),
        },
        _ => (authority.clone(), default_port),
    };
    if host.is_empty() {
        return None;
    }
    Some(format!("{scheme}://{host}:{port}"))
}

/// Хаяг нь заасан origin-д харьяалагдах эсэх.
pub fn same_origin(url: &str, base: &str) -> bool {
    match (origin_of(url), origin_of(base)) {
        (Some(left), Some(right)) => left == right,
        _ => false,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn origin_ignores_default_port_notation() {
        assert_eq!(origin_of("http://localhost:80/apps"), origin_of("http://localhost"));
        assert_eq!(origin_of("https://a.mn:443"), origin_of("https://a.mn/x?y=1"));
    }

    #[test]
    fn origin_rejects_non_http_schemes() {
        assert!(origin_of("file:///etc/passwd").is_none());
        assert!(origin_of("gerege://apps").is_none());
    }

    #[test]
    fn same_origin_separates_ports() {
        assert!(!same_origin("http://localhost:8080", "http://localhost:3000"));
        assert!(same_origin("http://localhost:3000/apps", "http://localhost:3000"));
    }
}
