//! Cửa sổ, tray, hotkey, autostart (launcher.md 6.1–6.3).
use crate::config::{self, Config};
use crate::context;
use crate::runner::CmdError;
use crate::thread::{Thread, ThreadItem};
use crate::{AppState, ContextState};
use std::str::FromStr;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Mutex;
use tauri::menu::{CheckMenuItem, MenuBuilder, MenuItem};
use tauri::tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent};
use tauri::{AppHandle, Emitter, Manager, PhysicalPosition, PhysicalSize, WebviewWindow, WindowEvent, Wry};
use tauri_plugin_autostart::ManagerExt as _;
use tauri_plugin_global_shortcut::{GlobalShortcutExt, Shortcut};

/// State giao diện: hotkey đang đăng ký, cờ tạm tắt "ẩn khi mất focus", mục tray cần đổi chữ.
#[derive(Default)]
pub struct Ui {
    pub hotkey: Mutex<Option<String>>,
    pub suspend_autohide: AtomicBool,
    hotkey_item: Mutex<Option<MenuItem<Wry>>>,
    autostart_item: Mutex<Option<CheckMenuItem<Wry>>>,
}

fn bar(app: &AppHandle) -> Option<WebviewWindow> {
    app.get_webview_window("bar")
}

/// Đặt ô nhanh ở giữa ngang, cách mép trên 25% chiều cao của màn hình đang có chuột.
fn place_bar(app: &AppHandle, w: &WebviewWindow) {
    let mon = app
        .cursor_position()
        .ok()
        .and_then(|p| app.monitor_from_point(p.x, p.y).ok().flatten())
        .or_else(|| app.primary_monitor().ok().flatten());
    let Some(m) = mon else { return };
    let size = w.outer_size().unwrap_or(PhysicalSize::new(680, 90));
    let x = m.position().x + (m.size().width as i32 - size.width as i32) / 2;
    let y = m.position().y + (m.size().height as f64 * 0.25) as i32;
    let _ = w.set_position(PhysicalPosition::new(x, y)); // không đặt được vị trí thì giữ chỗ cũ
}

/// Hiện ô nhanh. `capture`: chụp ngữ cảnh trước khi cửa sổ launcher cướp focus (chỉ khi mở bằng hotkey).
pub fn show_bar(app: &AppHandle, capture: bool) {
    let Some(w) = bar(app) else { return };
    let enabled = app.state::<AppState>().config.lock().unwrap().context;
    let ctx = if capture && enabled { context::capture_with_budget() } else { context::Context::none() };
    *app.state::<ContextState>().0.lock().unwrap() = ctx;
    if let Some(main) = app.get_webview_window("main") {
        let _ = main.hide(); // hai cửa sổ không hiện cùng lúc
    }
    place_bar(app, &w);
    let _ = w.show();
    let _ = w.set_focus();
    let _ = app.emit_to("bar", "bar-shown", ());
}

pub fn hide_bar(app: &AppHandle) {
    if let Some(w) = bar(app) {
        let _ = w.hide();
    }
}

pub fn toggle_bar(app: &AppHandle) {
    match bar(app).and_then(|w| w.is_visible().ok()) {
        Some(true) => hide_bar(app),
        _ => show_bar(app, true),
    }
}

pub fn show_main(app: &AppHandle, tab: &str) {
    hide_bar(app);
    let Some(w) = app.get_webview_window("main") else { return };
    let _ = w.unminimize();
    let _ = w.show();
    let _ = w.set_focus();
    let _ = app.emit_to("main", "show-tab", serde_json::json!({ "tab": tab }));
}

fn add_error(app: &AppHandle, line: &str, code: &str, message: &str, hint: &str) {
    let ts = std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH).map(|d| d.as_millis() as u64).unwrap_or(0);
    app.state::<Thread>().add(ThreadItem {
        id: 0,
        line: line.into(),
        status: "error".into(),
        result: Some(serde_json::json!({ "kind": "error", "code": code, "message": message, "hint": hint })),
        ts,
    });
    let _ = app.emit("thread-changed", ()); // chưa có cửa sổ nào nghe thì bỏ qua
}

fn hotkey_failed(hotkey: &str) -> CmdError {
    CmdError::new("HOTKEY_FAILED", format!("Không đăng ký được phím tắt {hotkey}"), "Sửa hotkey trong tab Cài đặt hoặc file cấu hình rồi chọn Tải lại")
}

/// Đăng ký hotkey mới rồi bỏ hotkey cũ. Lỗi → giữ nguyên hotkey cũ.
pub fn apply_hotkey(app: &AppHandle, new: &str) -> Result<(), CmdError> {
    let ui = app.state::<Ui>();
    let old = ui.hotkey.lock().unwrap().clone();
    if old.as_deref() == Some(new) {
        return Ok(());
    }
    let sc = Shortcut::from_str(new).map_err(|_| hotkey_failed(new))?;
    app.global_shortcut().register(sc).map_err(|_| hotkey_failed(new))?;
    if let Some(o) = old.and_then(|o| Shortcut::from_str(&o).ok()) {
        let _ = app.global_shortcut().unregister(o); // hotkey cũ đã mất thì không có gì để bỏ
    }
    *ui.hotkey.lock().unwrap() = Some(new.to_string());
    if let Some(item) = ui.hotkey_item.lock().unwrap().as_ref() {
        let _ = item.set_text(format!("Mở ô lệnh ({new})"));
    }
    Ok(())
}

/// Bật/tắt khởi động cùng Windows theo config.
pub fn sync_autostart(app: &AppHandle, on: bool) {
    let al = app.autolaunch();
    let _ = if on { al.enable() } else { al.disable() }; // ghi HKCU\...\Run; lỗi (vd bị chặn) thì giữ nguyên trạng thái cũ
    if let Some(item) = app.state::<Ui>().autostart_item.lock().unwrap().as_ref() {
        let _ = item.set_checked(on);
    }
}

fn set_tray_tip(app: &AppHandle, tip: &str) {
    if let Some(t) = app.tray_by_id("foxkit") {
        let _ = t.set_tooltip(Some(tip));
    }
}

/// Đọc lại config, đăng ký lại hotkey, báo các cửa sổ tải lại manifest (tray "Tải lại").
pub fn reload(app: &AppHandle) {
    let state = app.state::<AppState>();
    let (cfg, warn) = config::load(&state.dir);
    *state.config.lock().unwrap() = cfg.clone();
    if let Some(w) = warn {
        add_error(app, "config", "CONFIG_BROKEN", &w, "Sửa hoặc xóa file config.json");
    }
    register_configured_hotkey(app, &cfg);
    sync_autostart(app, cfg.autostart);
    let _ = app.emit("reload", ());
}

fn register_configured_hotkey(app: &AppHandle, cfg: &Config) {
    match apply_hotkey(app, &cfg.hotkey) {
        Ok(()) => set_tray_tip(app, &format!("foxkit — {}", cfg.hotkey)),
        Err(e) => {
            set_tray_tip(app, &format!("foxkit — {}. {}", e.message, e.hint));
            add_error(app, &format!("hotkey {}", cfg.hotkey), &e.code, &format!("{}, có thể app khác đang dùng", e.message), &e.hint);
        }
    }
}

fn build_tray(app: &AppHandle, cfg: &Config) -> tauri::Result<()> {
    let ui = app.state::<Ui>();
    let open_bar = MenuItem::with_id(app, "bar", format!("Mở ô lệnh ({})", cfg.hotkey), true, None::<&str>)?;
    let open_main = MenuItem::with_id(app, "main", "Mở giao diện chính", true, None::<&str>)?;
    let rescan = MenuItem::with_id(app, "scan", "Quét lại thư mục lệnh", true, None::<&str>)?;
    let autostart = CheckMenuItem::with_id(app, "autostart", "Khởi động cùng Windows", true, cfg.autostart, None::<&str>)?;
    let reload_i = MenuItem::with_id(app, "reload", "Tải lại", true, None::<&str>)?;
    let open_cfg = MenuItem::with_id(app, "config", "Mở file cấu hình", true, None::<&str>)?;
    let quit = MenuItem::with_id(app, "quit", "Thoát", true, None::<&str>)?;
    *ui.hotkey_item.lock().unwrap() = Some(open_bar.clone());
    *ui.autostart_item.lock().unwrap() = Some(autostart.clone());
    let menu = MenuBuilder::new(app).items(&[&open_bar, &open_main, &rescan, &autostart, &reload_i, &open_cfg, &quit]).build()?;

    let mut b = TrayIconBuilder::with_id("foxkit").tooltip("foxkit").menu(&menu).show_menu_on_left_click(false);
    if let Some(icon) = app.default_window_icon() {
        b = b.icon(icon.clone());
    }
    b.on_menu_event(|app, e| match e.id().as_ref() {
        "bar" => show_bar(app, false),
        "main" => show_main(app, "chat"),
        "scan" => {
            show_main(app, "chat");
            // quét như `:scan`: frontend của cửa sổ chính chạy lệnh nội bộ khi nhận sự kiện này
            let _ = app.emit_to("main", "run-line", ":scan");
        }
        "autostart" => {
            let st = app.state::<AppState>();
            let on = {
                let mut c = st.config.lock().unwrap();
                c.autostart = !c.autostart;
                let _ = config::save(&st.dir, &c); // không ghi được thì thay đổi chỉ có hiệu lực tới lần thoát
                c.autostart
            };
            sync_autostart(app, on);
        }
        "reload" => reload(app),
        "config" => {
            let p = config::config_path(&app.state::<AppState>().dir);
            let _ = std::process::Command::new("cmd").args(["/c", "start", "", &p.to_string_lossy()]).spawn(); // mở bằng app mặc định
        }
        "quit" => app.exit(0),
        _ => {}
    })
    .on_tray_icon_event(|tray, e| match e {
        TrayIconEvent::Click { button: MouseButton::Left, button_state: MouseButtonState::Up, .. } => show_bar(tray.app_handle(), false),
        TrayIconEvent::DoubleClick { button: MouseButton::Left, .. } => show_main(tray.app_handle(), "chat"),
        _ => {}
    })
    .build(app)?;
    Ok(())
}

/// Dựng tray, hotkey, hành vi hai cửa sổ; gọi trong `setup`.
pub fn setup(app: &AppHandle, startup_warning: Option<String>) -> tauri::Result<()> {
    let cfg = app.state::<AppState>().config.lock().unwrap().clone();
    build_tray(app, &cfg)?;
    register_configured_hotkey(app, &cfg);
    sync_autostart(app, cfg.autostart);
    if let Some(w) = startup_warning {
        add_error(app, "config", "CONFIG_BROKEN", &w, "Sửa hoặc xóa file config.json");
    }

    if let Some(w) = bar(app) {
        let handle = app.clone();
        w.on_window_event(move |e| {
            // ẩn khi mất focus, trừ lúc đang mở hộp thoại chọn thư mục
            if matches!(e, WindowEvent::Focused(false)) && !handle.state::<Ui>().suspend_autohide.load(Ordering::SeqCst) {
                hide_bar(&handle);
            }
        });
    }
    if let Some(w) = app.get_webview_window("main") {
        let win = w.clone();
        w.on_window_event(move |e| match e {
            WindowEvent::CloseRequested { api, .. } => {
                api.prevent_close(); // X chỉ ẩn về tray
                let _ = win.hide();
            }
            WindowEvent::Resized(_) if win.is_minimized().unwrap_or(false) => {
                let _ = win.hide(); // thu nhỏ cũng về tray
            }
            _ => {}
        });
    }
    // Mở app từ Start/nhấp đúp = vào giao diện chính; autostart (--hidden) thì chạy ngầm.
    if !std::env::args().any(|a| a == "--hidden") {
        show_main(app, "chat");
    }
    Ok(())
}
