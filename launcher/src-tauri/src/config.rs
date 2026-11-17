use serde::{Deserialize, Serialize};
use serde_json::{Map, Value};
use std::path::{Path, PathBuf};

/// Gợi ý riêng cho một app: tên exe (vd. `chrome.exe`) và các lệnh hiện sẵn khi mở ô nhanh lúc app đó đang focus.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Default)]
#[serde(default)]
pub struct AppHint {
    pub app: String,
    pub commands: Vec<String>,
}

/// `%APPDATA%\foxkit-launcher\config.json` (launcher.md 5.1).
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq)]
#[serde(rename_all = "camelCase", default)]
pub struct Config {
    pub hotkey: String,
    pub foxkit_path: String,
    pub autostart: bool,
    pub auto_copy: bool,
    pub context: bool,
    pub folders: Vec<String>,
    /// Gợi ý theo app (mặc định rỗng = không gợi ý gì).
    pub app_hints: Vec<AppHint>,
    /// Field lạ do người dùng tự thêm: giữ nguyên khi ghi lại.
    #[serde(flatten)]
    pub extra: Map<String, Value>,
}

impl Default for Config {
    fn default() -> Self {
        Config {
            hotkey: "Ctrl+Alt+Space".into(),
            foxkit_path: String::new(),
            autostart: true,
            auto_copy: false,
            context: true,
            folders: Vec::new(),
            app_hints: Vec::new(),
            extra: Map::new(),
        }
    }
}

/// Thư mục dữ liệu của launcher: `%APPDATA%\foxkit-launcher`.
pub fn data_dir() -> PathBuf {
    let base = std::env::var_os("APPDATA").map(PathBuf::from).unwrap_or_else(std::env::temp_dir);
    base.join("foxkit-launcher")
}

pub fn config_path(dir: &Path) -> PathBuf {
    dir.join("config.json")
}

/// Đọc config. File thiếu → tạo mặc định. JSON hỏng → dùng mặc định, KHÔNG ghi đè file, trả cảnh báo.
pub fn load(dir: &Path) -> (Config, Option<String>) {
    let path = config_path(dir);
    match std::fs::read_to_string(&path) {
        Ok(text) => match serde_json::from_str::<Config>(&text) {
            Ok(c) => (c, None),
            Err(e) => (Config::default(), Some(format!("config.json bị hỏng ({e}), đang dùng cấu hình mặc định"))),
        },
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => {
            let c = Config::default();
            let warn = save(dir, &c).err().map(|e| format!("Không tạo được config.json: {e}"));
            (c, warn)
        }
        Err(e) => (Config::default(), Some(format!("Không đọc được config.json: {e}"))),
    }
}

pub fn save(dir: &Path, cfg: &Config) -> Result<(), String> {
    std::fs::create_dir_all(dir).map_err(|e| e.to_string())?;
    let text = serde_json::to_string_pretty(cfg).map_err(|e| e.to_string())?;
    std::fs::write(config_path(dir), text).map_err(|e| e.to_string())
}

/// Tìm foxkit.exe: `foxkitPath` → cạnh launcher → (debug) thư mục src-tauri → PATH.
pub fn find_foxkit(cfg_path: &str, exe_dir: Option<&Path>) -> Option<PathBuf> {
    let configured = expand_env(cfg_path);
    if !configured.is_empty() {
        let p = PathBuf::from(&configured);
        return p.is_file().then_some(p);
    }
    let mut candidates: Vec<PathBuf> = Vec::new();
    if let Some(d) = exe_dir {
        candidates.push(d.join("foxkit.exe"));
    }
    if cfg!(debug_assertions) {
        if let Some(m) = option_env!("CARGO_MANIFEST_DIR") {
            candidates.push(Path::new(m).join("foxkit.exe"));
        }
    }
    if let Some(found) = candidates.into_iter().find(|p| p.is_file()) {
        return Some(found);
    }
    let path = std::env::var_os("PATH")?;
    std::env::split_paths(&path).map(|d| d.join("foxkit.exe")).find(|p| p.is_file())
}

/// Mở rộng `%VAR%` kiểu Windows.
pub fn expand_env(s: &str) -> String {
    let mut out = String::new();
    let mut rest = s;
    while let Some(i) = rest.find('%') {
        out.push_str(&rest[..i]);
        let after = &rest[i + 1..];
        match after.find('%') {
            Some(j) if j > 0 => match std::env::var(&after[..j]) {
                Ok(v) => {
                    out.push_str(&v);
                    rest = &after[j + 1..];
                }
                Err(_) => {
                    out.push('%');
                    rest = after;
                }
            },
            _ => {
                out.push('%');
                rest = after;
            }
        }
    }
    out.push_str(rest);
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    fn tmp(name: &str) -> PathBuf {
        let d = std::env::temp_dir().join(format!("foxkit-test-{name}-{}", std::process::id()));
        let _ = std::fs::remove_dir_all(&d);
        d
    }

    #[test]
    fn missing_file_creates_default() {
        let d = tmp("cfg-missing");
        let (c, warn) = load(&d);
        assert_eq!(c, Config::default());
        assert!(warn.is_none());
        assert!(config_path(&d).is_file());
    }

    #[test]
    fn broken_json_uses_default_and_does_not_overwrite() {
        let d = tmp("cfg-broken");
        std::fs::create_dir_all(&d).unwrap();
        std::fs::write(config_path(&d), "{ not json").unwrap();
        let (c, warn) = load(&d);
        assert_eq!(c, Config::default());
        assert!(warn.is_some());
        assert_eq!(std::fs::read_to_string(config_path(&d)).unwrap(), "{ not json");
    }

    #[test]
    fn partial_file_fills_defaults_and_keeps_unknown_fields() {
        let d = tmp("cfg-partial");
        std::fs::create_dir_all(&d).unwrap();
        std::fs::write(config_path(&d), r#"{"hotkey":"Ctrl+Alt+K","mine":42}"#).unwrap();
        let (c, _) = load(&d);
        assert_eq!(c.hotkey, "Ctrl+Alt+K");
        assert!(c.context && c.autostart);
        save(&d, &c).unwrap();
        assert!(std::fs::read_to_string(config_path(&d)).unwrap().contains("\"mine\": 42"));
    }

    #[test]
    fn app_hints_default_empty_and_roundtrip() {
        let d = tmp("cfg-hints");
        std::fs::create_dir_all(&d).unwrap();
        std::fs::write(config_path(&d), r#"{"hotkey":"Ctrl+Alt+K"}"#).unwrap();
        let (mut c, _) = load(&d);
        assert!(c.app_hints.is_empty());
        c.app_hints.push(AppHint { app: "chrome.exe".into(), commands: vec!["http".into(), "dns".into()] });
        save(&d, &c).unwrap();
        let (back, warn) = load(&d);
        assert!(warn.is_none());
        assert_eq!(back.app_hints, c.app_hints);
        assert!(std::fs::read_to_string(config_path(&d)).unwrap().contains("\"appHints\""));
    }

    #[test]
    fn expand_env_works() {
        std::env::set_var("FOXKIT_T", "xyz");
        assert_eq!(expand_env("a%FOXKIT_T%b"), "axyzb");
        assert_eq!(expand_env("100%"), "100%");
        assert_eq!(expand_env("%NOPE_NOT_SET%x"), "%NOPE_NOT_SET%x");
    }
}
