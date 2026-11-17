mod config;
#[allow(dead_code)] // capture_with_budget được nối vào hotkey ở L1.7
mod context;
mod elevate;
mod runner;
mod scan;
mod thread;

use runner::{CmdError, ExecSpec, RunResult, Runs, StreamMsg};
use std::collections::HashSet;
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

/// Chạy `foxkit manifest --json` và trả `data` (đã kiểm schema).
async fn fetch_builtin(foxkit: Option<PathBuf>) -> Result<serde_json::Value, CmdError> {
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

fn names_of(manifest: &serde_json::Value) -> HashSet<String> {
    manifest["commands"].as_array().into_iter().flatten().filter_map(|c| c["name"].as_str().map(String::from)).collect()
}

/// Lệnh của foxkit rồi lệnh ngoài (từ snapshot, không quét lại); mỗi lệnh có `source` và `file`.
#[tauri::command]
async fn load_manifest(state: State<'_, AppState>) -> Result<serde_json::Value, CmdError> {
    let mut data = fetch_builtin(state.foxkit()).await?;
    let builtin = names_of(&data);
    let cmds = data["commands"].as_array_mut().ok_or_else(|| CmdError::new("MANIFEST_FAILED", "Manifest không có danh sách lệnh", ""))?;
    for c in cmds.iter_mut() {
        c["source"] = "builtin".into();
    }
    // lệnh ngoài trùng tên một lệnh foxkit mới thêm sau lần quét thì bỏ
    cmds.extend(scan::load_snapshot(&state.dir).all_commands().into_iter().filter(|c| c["name"].as_str().is_some_and(|n| !builtin.contains(n))));
    Ok(data)
}

fn now_secs() -> u64 {
    std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH).map(|d| d.as_secs()).unwrap_or(0)
}

/// Quét `only` (hoặc mọi thư mục nếu None), ghi snapshot + scan.log.
async fn do_scan(state: &AppState, folders: Vec<String>, only: Option<Vec<String>>) -> Result<scan::ScanReport, CmdError> {
    let builtin = names_of(&fetch_builtin(state.foxkit()).await?);
    let prev = scan::load_snapshot(&state.dir);
    let (snap, report, log) = scan::scan(&folders, only.as_deref(), &prev, &builtin, now_secs());
    scan::save_snapshot(&state.dir, &snap).map_err(|e| CmdError::new("SCAN_FAILED", format!("Không ghi được scan-cache.json: {e}"), ""))?;
    scan::write_log(&state.dir, &log);
    Ok(report)
}

/// `:scan`: quét lại các thư mục trong danh sách (mặc định toàn bộ `folders` trong config).
#[tauri::command]
async fn scan(state: State<'_, AppState>, folders: Option<Vec<String>>) -> Result<scan::ScanReport, CmdError> {
    let all = state.config.lock().unwrap().folders.clone();
    let list = folders.unwrap_or_else(|| all.clone());
    do_scan(&state, all, Some(list)).await
}

fn save_cfg(state: &AppState, cfg: &config::Config) -> Result<(), CmdError> {
    config::save(&state.dir, cfg).map_err(|e| CmdError::new("CONFIG_FAILED", format!("Không ghi được config.json: {e}"), ""))
}

/// `:add`: thêm thư mục (không trùng) vào config rồi quét riêng thư mục đó.
#[tauri::command]
async fn add_folder(state: State<'_, AppState>, path: String) -> Result<scan::ScanReport, CmdError> {
    let path = path.trim().trim_end_matches(['\\', '/']).to_string();
    let all = {
        let mut cfg = state.config.lock().unwrap();
        if !cfg.folders.iter().any(|f| scan::same_folder(f, &path)) {
            cfg.folders.push(path.clone());
            save_cfg(&state, &cfg)?;
        }
        cfg.folders.clone()
    };
    let target = all.iter().find(|f| scan::same_folder(f, &path)).cloned().unwrap_or(path);
    do_scan(&state, all, Some(vec![target])).await
}

/// `:forget`: bỏ thư mục khỏi config và xóa lệnh của nó khỏi snapshot (không đụng file nào trên đĩa).
#[tauri::command]
fn forget_folder(state: State<'_, AppState>, path: String) -> Result<bool, CmdError> {
    let removed = {
        let mut cfg = state.config.lock().unwrap();
        let before = cfg.folders.len();
        cfg.folders.retain(|f| !scan::same_folder(f, &path));
        save_cfg(&state, &cfg)?;
        cfg.folders.len() != before
    };
    let mut snap = scan::load_snapshot(&state.dir);
    snap.folders.retain(|f| !scan::same_folder(&f.path, &path));
    scan::save_snapshot(&state.dir, &snap).map_err(|e| CmdError::new("SCAN_FAILED", e, ""))?;
    Ok(removed)
}

#[derive(serde::Serialize)]
struct FolderInfo {
    path: String,
    files: usize,
    commands: usize,
    skipped: u32,
    scanned_at: u64,
    status: String,
}

/// `:folders`: mọi thư mục trong config kèm trạng thái từ snapshot.
#[tauri::command]
fn folders_info(state: State<'_, AppState>) -> Vec<FolderInfo> {
    let snap = scan::load_snapshot(&state.dir);
    state
        .config
        .lock()
        .unwrap()
        .folders
        .iter()
        .map(|p| match snap.folders.iter().find(|f| f.path == *p) {
            Some(f) => FolderInfo { path: p.clone(), files: f.files.len(), commands: f.files.iter().map(|x| x.commands.len()).sum(), skipped: f.skipped, scanned_at: f.scanned_at, status: f.status.clone() },
            None => FolderInfo { path: p.clone(), files: 0, commands: 0, skipped: 0, scanned_at: 0, status: "chưa quét".into() },
        })
        .collect()
}

#[tauri::command]
fn scan_log(state: State<'_, AppState>) -> String {
    scan::read_log(&state.dir)
}

/// Lệnh ngoài này đã được cho phép (đúng nội dung file khai báo hiện tại) chưa?
#[tauri::command]
fn trust_check(state: State<'_, AppState>, file: String) -> bool {
    scan::is_trusted(&state.dir, &file)
}

#[tauri::command]
fn trust_allow(state: State<'_, AppState>, file: String) -> Result<(), CmdError> {
    scan::trust(&state.dir, &file).map_err(|e| CmdError::new("TRUST_FAILED", e, ""))
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

/// Ảnh chụp ngữ cảnh lúc bấm hotkey (đặt bởi handler hotkey ở L1.7); `context: false` thì luôn rỗng.
#[derive(Default)]
pub struct ContextState(pub Mutex<context::Context>);

#[tauri::command]
fn get_context(state: State<'_, AppState>, ctx: State<'_, ContextState>) -> context::Context {
    if !state.config.lock().unwrap().context {
        return context::Context::none();
    }
    ctx.0.lock().unwrap().clone()
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
        .manage(ContextState::default())
        .invoke_handler(tauri::generate_handler![
            load_manifest, run, run_stream, stop, run_elevated, get_config, get_context, thread_list, thread_add, thread_update, thread_clear,
            scan, add_folder, forget_folder, folders_info, scan_log, trust_check, trust_allow
        ])
        .setup(|app| {
            let _ = app.get_webview_window("main"); // cửa sổ được dựng sẵn từ tauri.conf.json
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("lỗi khi chạy foxkit launcher");
}
