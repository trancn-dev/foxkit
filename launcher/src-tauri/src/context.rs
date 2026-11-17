//! Nhận diện ngữ cảnh lúc bấm hotkey (launcher.md 6.8): app đang focus, thư mục/file hiện tại.
//! Phần logic thuần (tách tiêu đề VS Code, đọc storage.json, đổi URI) có test; phần Win32/COM ở module `win`.
use serde::Serialize;
use serde_json::Value;

#[derive(Serialize, Clone, Debug, Default, PartialEq)]
pub struct Context {
    pub app: String,
    pub exe: String,
    pub title: String,
    pub dir: Option<String>,
    pub files: Vec<String>,
    /// "explorer" | "vscode" | "generic" | "none"
    pub source: String,
}

impl Context {
    pub fn none() -> Self {
        Context { source: "none".into(), ..Default::default() }
    }
}

/// Tên đẹp theo tên exe (không phân biệt hoa thường).
pub fn friendly_app(exe: &str) -> String {
    let file = exe.rsplit(['\\', '/']).next().unwrap_or(exe);
    match file.to_lowercase().as_str() {
        "code.exe" => "VS Code".into(),
        "code - insiders.exe" => "VS Code Insiders".into(),
        "cursor.exe" => "Cursor".into(),
        "vscodium.exe" => "VSCodium".into(),
        "explorer.exe" => "Explorer".into(),
        other => other.strip_suffix(".exe").map(|_| file[..file.len() - 4].to_string()).unwrap_or_else(|| file.to_string()),
    }
}

/// Thư mục dữ liệu của app VS Code theo exe: `%APPDATA%\<tên>`.
pub fn vscode_data_name(exe: &str) -> Option<&'static str> {
    match exe.rsplit(['\\', '/']).next()?.to_lowercase().as_str() {
        "code.exe" => Some("Code"),
        "code - insiders.exe" => Some("Code - Insiders"),
        "cursor.exe" => Some("Cursor"),
        "vscodium.exe" => Some("VSCodium"),
        _ => None,
    }
}

const VSCODE_SUFFIXES: [&str; 4] = ["Visual Studio Code - Insiders", "Visual Studio Code", "Cursor", "VSCodium"];

/// Tách tiêu đề VS Code `● file.ext - tên-thư-mục - Visual Studio Code` thành `rootName` (nếu có).
pub fn vscode_root_name(title: &str) -> Option<String> {
    let t = title.trim();
    let rest = VSCODE_SUFFIXES.iter().find_map(|s| t.strip_suffix(&format!(" - {s}")))?;
    let rest = rest.trim_start_matches(['●', '*', ' ']);
    let parts: Vec<&str> = rest.split(" - ").collect();
    let root = if parts.len() >= 2 { parts[1..].join(" - ") } else { rest.to_string() };
    (!root.is_empty()).then_some(root)
}

fn pct_decode(s: &str) -> String {
    let b = s.as_bytes();
    let mut out = Vec::with_capacity(b.len());
    let mut i = 0;
    while i < b.len() {
        if b[i] == b'%' && i + 2 < b.len() {
            if let Some(v) = s.get(i + 1..i + 3).and_then(|h| u8::from_str_radix(h, 16).ok()) {
                out.push(v);
                i += 3;
                continue;
            }
        }
        out.push(b[i]);
        i += 1;
    }
    String::from_utf8_lossy(&out).into_owned()
}

/// `file:///c%3A/My%20Dir` → `C:\My Dir`. URI không phải `file:///` (remote, WSL...) → None.
pub fn uri_to_path(uri: &str) -> Option<String> {
    let u = url::Url::parse(uri).ok()?;
    if u.scheme() != "file" || u.host_str().is_some_and(|h| !h.is_empty()) {
        return None;
    }
    let p = pct_decode(u.path());
    let p = p.trim_start_matches('/');
    let mut chars = p.chars();
    let (drive, rest) = (chars.next()?, chars.as_str());
    if !drive.is_ascii_alphabetic() || !rest.starts_with(':') {
        return None;
    }
    Some(format!("{}{}", drive.to_ascii_uppercase(), rest.replace('/', "\\")))
}

fn basename(p: &str) -> &str {
    p.trim_end_matches('\\').rsplit('\\').next().unwrap_or(p)
}

/// Chọn thư mục từ storage.json: duy nhất khớp tên → dùng; nhiều khớp → lấy `lastActiveWindow` nếu nằm trong đó; còn lại None.
pub fn pick_vscode_dir(root: &str, storage: &Value) -> Option<String> {
    let ws = storage.get("windowsState")?;
    let last = ws.pointer("/lastActiveWindow/folder").and_then(Value::as_str).and_then(uri_to_path);
    let mut folders: Vec<String> = ws
        .get("openedWindows")
        .and_then(Value::as_array)
        .into_iter()
        .flatten()
        .filter_map(|w| w.get("folder").and_then(Value::as_str))
        .filter_map(uri_to_path)
        .collect();
    folders.extend(last.clone());
    folders.sort();
    folders.dedup();
    let matches: Vec<String> = folders.into_iter().filter(|f| basename(f).eq_ignore_ascii_case(root)).collect();
    match matches.len() {
        0 => None,
        1 => matches.into_iter().next(),
        _ => last.filter(|l| matches.contains(l)),
    }
}

#[cfg(windows)]
mod win {
    use super::*;
    use windows::core::{Interface, PWSTR};
    use windows::Win32::Foundation::{CloseHandle, HWND};
    use windows::Win32::System::Com::{CoCreateInstance, CoInitializeEx, CLSCTX_ALL, COINIT_APARTMENTTHREADED};
    use windows::Win32::System::Threading::{OpenProcess, QueryFullProcessImageNameW, PROCESS_NAME_WIN32, PROCESS_QUERY_LIMITED_INFORMATION};
    use windows::Win32::UI::Shell::{Folder2, IShellFolderViewDual, IShellWindows, IWebBrowserApp, ShellWindows};
    use windows::Win32::System::Variant::VARIANT;
    use windows::Win32::UI::WindowsAndMessaging::{GetClassNameW, GetForegroundWindow, GetWindowTextW, GetWindowThreadProcessId};

    fn wide(buf: &[u16], n: usize) -> String {
        String::from_utf16_lossy(&buf[..n.min(buf.len())])
    }

    /// Lấy thư mục + file đang chọn của cửa sổ Explorer có HWND trùng. Mọi lỗi COM → None.
    fn explorer(hwnd: HWND) -> Option<(String, Vec<String>)> {
        // SAFETY: gọi COM trên thread hiện tại (apartment-threaded); các interface sống trong hàm này.
        unsafe {
            let _ = CoInitializeEx(None, COINIT_APARTMENTTHREADED); // S_FALSE khi đã khởi tạo, không phải lỗi
            let shell: IShellWindows = CoCreateInstance(&ShellWindows, None, CLSCTX_ALL).ok()?;
            let count = shell.Count().ok()?;
            for i in 0..count {
                let Ok(disp) = shell.Item(&VARIANT::from(i)) else { continue };
                let Ok(wb) = disp.cast::<IWebBrowserApp>() else { continue };
                if wb.HWND().ok()?.0 != hwnd.0 as isize {
                    continue;
                }
                let view: IShellFolderViewDual = wb.Document().ok()?.cast().ok()?;
                let dir = view.Folder().ok()?.cast::<Folder2>().ok()?.Self_().ok()?.Path().ok()?.to_string();
                let mut files = Vec::new();
                if let Ok(items) = view.SelectedItems() {
                    for k in 0..items.Count().unwrap_or(0) {
                        if let Ok(it) = items.Item(&VARIANT::from(k)) {
                            if let Ok(p) = it.Path() {
                                files.push(p.to_string());
                            }
                        }
                    }
                }
                return Some((dir, files));
            }
            None
        }
    }

    fn is_real_dir(p: &str) -> bool {
        let b = p.as_bytes();
        !p.starts_with("::") && b.len() >= 2 && b[0].is_ascii_alphabetic() && b[1] == b':'
    }

    pub fn capture() -> Context {
        // SAFETY: các hàm Win32 chỉ đọc thông tin cửa sổ; buffer sống đủ lâu; handle process đóng bằng CloseHandle.
        unsafe {
            let hwnd = GetForegroundWindow();
            if hwnd.0.is_null() {
                return Context::none();
            }
            let mut pid = 0u32;
            GetWindowThreadProcessId(hwnd, Some(&mut pid));
            if pid == 0 || pid == std::process::id() {
                return Context::none();
            }
            let mut cls = [0u16; 256];
            let n = GetClassNameW(hwnd, &mut cls) as usize;
            let class = wide(&cls, n);
            let mut tbuf = [0u16; 512];
            let n = GetWindowTextW(hwnd, &mut tbuf) as usize;
            let title = wide(&tbuf, n);
            let mut exe = String::new();
            if let Ok(h) = OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, false, pid) {
                let mut buf = [0u16; 1024];
                let mut size = buf.len() as u32;
                if QueryFullProcessImageNameW(h, PROCESS_NAME_WIN32, PWSTR(buf.as_mut_ptr()), &mut size).is_ok() {
                    exe = wide(&buf, size as usize);
                }
                let _ = CloseHandle(h);
            }
            let app = friendly_app(&exe);
            let is_explorer = app == "Explorer";
            if is_explorer && matches!(class.as_str(), "Shell_TrayWnd" | "Progman" | "WorkerW") {
                return Context::none();
            }
            let mut ctx = Context { app, exe: exe.clone(), title: title.clone(), dir: None, files: vec![], source: "generic".into() };
            if is_explorer && class == "CabinetWClass" {
                ctx.source = "explorer".into();
                if let Some((dir, files)) = explorer(hwnd) {
                    if is_real_dir(&dir) {
                        ctx.dir = Some(dir);
                        ctx.files = files;
                    }
                }
            } else if let Some(data) = vscode_data_name(&exe) {
                ctx.source = "vscode".into();
                ctx.dir = vscode_root_name(&title).and_then(|root| {
                    let appdata = std::env::var_os("APPDATA")?;
                    let p = std::path::Path::new(&appdata).join(data).join("User/globalStorage/storage.json");
                    let json: Value = serde_json::from_str(&std::fs::read_to_string(p).ok()?).ok()?;
                    pick_vscode_dir(&root, &json)
                });
            }
            ctx
        }
    }
}

/// Chụp ngữ cảnh với ngân sách 200 ms (chạy ở thread riêng); quá hạn thì trả `none`.
pub fn capture_with_budget() -> Context {
    #[cfg(windows)]
    {
        let (tx, rx) = std::sync::mpsc::channel();
        std::thread::spawn(move || {
            let _ = tx.send(win::capture()); // người nhận đã bỏ cuộc (quá hạn) thì không sao
        });
        rx.recv_timeout(std::time::Duration::from_millis(200)).unwrap_or_else(|_| Context::none())
    }
    #[cfg(not(windows))]
    {
        Context::none()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn title_parsing() {
        assert_eq!(vscode_root_name("● a.ts - foxkit - Visual Studio Code").as_deref(), Some("foxkit"));
        assert_eq!(vscode_root_name("a.ts - foxkit - Visual Studio Code").as_deref(), Some("foxkit"));
        assert_eq!(vscode_root_name("foxkit - Visual Studio Code").as_deref(), Some("foxkit"));
        assert_eq!(vscode_root_name("Visual Studio Code"), None);
        assert_eq!(vscode_root_name("a.ts - my - dir - Visual Studio Code").as_deref(), Some("my - dir"));
        assert_eq!(vscode_root_name("x.rs - proj-1 - Cursor").as_deref(), Some("proj-1"));
        assert_eq!(vscode_root_name("x - y - Visual Studio Code - Insiders").as_deref(), Some("y"));
        assert_eq!(vscode_root_name("Untitled - Notepad"), None);
    }

    #[test]
    fn uri_conversion() {
        assert_eq!(uri_to_path("file:///c%3A/My%20Dir").as_deref(), Some(r"C:\My Dir"));
        assert_eq!(uri_to_path("file:///d%3A/Code/foxkit").as_deref(), Some(r"D:\Code\foxkit"));
        assert_eq!(uri_to_path("vscode-remote://wsl%2Bubuntu/home/x"), None);
        assert_eq!(uri_to_path("file://server/share/x"), None);
        assert_eq!(uri_to_path("not a uri"), None);
    }

    fn storage(last: &str, opened: &[&str]) -> Value {
        json!({ "windowsState": {
            "lastActiveWindow": { "folder": last },
            "openedWindows": opened.iter().map(|f| json!({ "folder": f })).collect::<Vec<_>>()
        }})
    }

    #[test]
    fn picks_directory_from_storage() {
        let one = storage("file:///d%3A/work/foxkit", &["file:///d%3A/work/other"]);
        assert_eq!(pick_vscode_dir("foxkit", &one).as_deref(), Some(r"D:\work\foxkit"));
        // nhiều khớp, có lastActiveWindow trong đó
        let many = storage("file:///d%3A/b/app", &["file:///d%3A/a/app", "file:///d%3A/b/app"]);
        assert_eq!(pick_vscode_dir("app", &many).as_deref(), Some(r"D:\b\app"));
        // nhiều khớp, lastActiveWindow không nằm trong đó
        let none = storage("file:///d%3A/z/zzz", &["file:///d%3A/a/app", "file:///d%3A/b/app"]);
        assert_eq!(pick_vscode_dir("app", &none), None);
        // remote bị bỏ qua
        let remote = storage("vscode-remote://ssh-remote%2Bx/home/app", &[]);
        assert_eq!(pick_vscode_dir("app", &remote), None);
        assert_eq!(pick_vscode_dir("app", &json!({})), None);
    }

    #[test]
    fn friendly_names() {
        assert_eq!(friendly_app(r"C:\Users\x\AppData\Local\Programs\Microsoft VS Code\Code.exe"), "VS Code");
        assert_eq!(friendly_app(r"C:\p\Cursor.exe"), "Cursor");
        assert_eq!(friendly_app(r"C:\Windows\explorer.exe"), "Explorer");
        assert_eq!(friendly_app(r"C:\Windows\notepad.exe"), "notepad");
        assert_eq!(friendly_app(""), "");
    }
}
