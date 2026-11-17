mod config;
mod elevate;
mod runner;
mod thread;

use runner::{CmdError, ExecSpec, RunResult, Runs, StreamMsg};
use std::path::PathBuf;
use std::sync::{Arc, Mutex};
use tauri::ipc::Channel;
use tauri::{AppHandle, Emitter, Manager, State};
use thread::{Thread, ThreadItem};

/// State chung của app.
pub struct AppState {
    pub config: Mutex<config::Config>,
    pub dir: PathBuf,
}

impl AppState {
    /// Đường dẫn foxkit.exe theo config hiện tại.
    fn foxkit(&self) -> Option<PathBuf> {
        let path = self.config.lock().unwrap().foxkit_path.clone();
        let exe_dir = std::env::current_exe().ok().and_then(|p| p.parent().map(PathBuf::from));
        config::find_foxkit(&path, exe_dir.as_deref())
    }
}

#[tauri::command]
async fn load_manifest(state: State<'_, AppState>) -> Result<serde_json::Value, CmdError> {
    let foxkit = state.foxkit();
    let bad = || CmdError::new("MANIFEST_FAILED", "Không đọc được danh sách lệnh từ foxkit", "Chạy `foxkit manifest --json` trong terminal để xem lỗi");
    let res = tauri::async_runtime::spawn_blocking(move || runner::run_blocking(foxkit.as_deref(), &["manifest".into()], None, None))
        .await
        .map_err(|_| bad())??;
    if res.exit_code != 0 {
        return Err(bad());
    }
    let v: serde_json::Value = serde_json::from_str(&res.stdout).map_err(|_| bad())?;
    let data = v.get("data").cloned().ok_or_else(bad)?;
    if data.get("schema").and_then(|s| s.as_i64()) != Some(1) {
        return Err(CmdError::new("MANIFEST_UNSUPPORTED", "foxkit dùng manifest phiên bản mới hơn launcher", "Cập nhật launcher"));
    }
    Ok(data)
}

#[tauri::command]
async fn run(state: State<'_, AppState>, args: Vec<String>, cwd: Option<String>, exec: Option<ExecSpec>) -> Result<RunResult, CmdError> {
    let foxkit = state.foxkit();
    tauri::async_runtime::spawn_blocking(move || runner::run_blocking(foxkit.as_deref(), &args, cwd.as_deref(), exec.as_ref()))
        .await
        .map_err(|e| CmdError::new("SPAWN_FAILED", e.to_string(), ""))?
}

#[tauri::command]
async fn run_stream(
    state: State<'_, AppState>,
    runs: State<'_, Arc<Runs>>,
    args: Vec<String>,
    cwd: Option<String>,
    exec: Option<ExecSpec>,
    on_line: Channel<StreamMsg>,
) -> Result<u32, CmdError> {
    runner::run_stream(&runs, state.foxkit().as_deref(), &args, cwd.as_deref(), exec.as_ref(), on_line)
}

#[tauri::command]
fn stop(runs: State<'_, Arc<Runs>>, run_id: u32) -> Result<(), String> {
    runner::stop(&runs, run_id);
    Ok(())
}

#[tauri::command]
async fn run_elevated(state: State<'_, AppState>, args: Vec<String>, cwd: Option<String>) -> Result<RunResult, CmdError> {
    let foxkit = state.foxkit().ok_or_else(|| CmdError::new("FOXKIT_NOT_FOUND", "Không tìm thấy foxkit.exe", "Đặt foxkit.exe cạnh launcher hoặc điền foxkitPath"))?;
    tauri::async_runtime::spawn_blocking(move || elevate::run_elevated(&foxkit, &args, cwd.as_deref()))
        .await
        .map_err(|e| CmdError::new("ELEVATE_FAILED", e.to_string(), ""))?
}

#[tauri::command]
fn get_config(state: State<'_, AppState>) -> config::Config {
    state.config.lock().unwrap().clone()
}

#[tauri::command]
fn thread_list(t: State<'_, Thread>) -> Vec<ThreadItem> {
    t.list()
}

#[tauri::command]
fn thread_add(app: AppHandle, t: State<'_, Thread>, item: ThreadItem) -> u32 {
    let id = t.add(item);
    let _ = app.emit("thread-changed", ()); // chưa có cửa sổ nào nghe thì bỏ qua
    id
}

#[tauri::command]
fn thread_update(app: AppHandle, t: State<'_, Thread>, id: u32, item: ThreadItem) {
    t.update(id, item);
    let _ = app.emit("thread-changed", ());
}

#[tauri::command]
fn thread_clear(app: AppHandle, t: State<'_, Thread>) {
    t.clear();
    let _ = app.emit("thread-changed", ());
}

pub fn start() {
    let dir = config::data_dir();
    let (cfg, _warning) = config::load(&dir);
    tauri::Builder::default()
        .manage(AppState { config: Mutex::new(cfg), dir })
        .manage(Arc::new(Runs::default()))
        .manage(Thread::default())
        .invoke_handler(tauri::generate_handler![
            load_manifest, run, run_stream, stop, run_elevated, get_config, thread_list, thread_add, thread_update, thread_clear
        ])
        .setup(|app| {
            let _ = app.get_webview_window("main"); // cửa sổ được dựng sẵn từ tauri.conf.json
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("lỗi khi chạy foxkit launcher");
}
