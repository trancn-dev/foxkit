use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::io::{BufRead, BufReader, Read};
use std::path::{Path, PathBuf};
use std::process::{Child, Command, Stdio};
use std::sync::atomic::{AtomicU32, Ordering};
use std::sync::{Arc, Mutex};
use tauri::ipc::Channel;

#[cfg(windows)]
const CREATE_NO_WINDOW: u32 = 0x0800_0000;

/// Lỗi trả cho frontend: có mã để hiển thị đúng message/hint (launcher.md mục 8).
#[derive(Serialize, Debug, Clone)]
pub struct CmdError {
    pub code: String,
    pub message: String,
    pub hint: String,
}

impl CmdError {
    pub fn new(code: &str, message: impl Into<String>, hint: impl Into<String>) -> Self {
        CmdError { code: code.into(), message: message.into(), hint: hint.into() }
    }
}

#[derive(Serialize, Debug, Clone)]
pub struct RunResult {
    pub exit_code: i32,
    pub stdout: String,
    pub stderr: String,
}

/// Cách chạy một lệnh ngoài (foxkit.json): chương trình thay cho foxkit.exe.
#[derive(Deserialize, Debug, Clone)]
#[serde(rename_all = "camelCase")]
pub struct ExecSpec {
    pub program: String,
    #[serde(default)]
    pub prefix: Vec<String>,
    /// true: chương trình theo chuẩn foxkit, thêm `--json` cuối cùng.
    #[serde(default)]
    pub json: bool,
    /// Thư mục chứa file foxkit.json khai báo; gốc cho đường dẫn tương đối.
    pub base_dir: String,
}

#[derive(Serialize, Clone, Debug)]
#[serde(tag = "kind")]
pub enum StreamMsg {
    Line { text: String },
    Stderr { text: String },
    Exit { code: i32 },
}

fn foxkit_not_found() -> CmdError {
    CmdError::new(
        "FOXKIT_NOT_FOUND",
        "Không tìm thấy foxkit.exe",
        "Đặt foxkit.exe cạnh launcher, thêm vào PATH, hoặc điền foxkitPath trong file cấu hình rồi chọn Tải lại",
    )
}

/// Đường dẫn tương đối (tính từ base_dir) nếu file tồn tại ở đó; không thì giữ nguyên (để tìm trong PATH).
fn resolve_path(p: &str, base: &Path) -> String {
    let expanded = crate::config::expand_env(p);
    let joined = base.join(&expanded);
    if !Path::new(&expanded).is_absolute() && joined.exists() {
        joined.to_string_lossy().into_owned()
    } else {
        expanded
    }
}

/// Dựng lệnh: foxkit.exe + args + `--json`, hoặc chương trình ngoài. Không qua shell.
pub fn build_command(foxkit: &Path, args: &[String], cwd: Option<&str>, exec: Option<&ExecSpec>) -> Command {
    let mut cmd;
    match exec {
        Some(e) => {
            let base = PathBuf::from(&e.base_dir);
            cmd = Command::new(resolve_path(&e.program, &base));
            for p in &e.prefix {
                cmd.arg(resolve_path(p, &base));
            }
            cmd.args(args);
            if e.json {
                cmd.arg("--json");
            }
        }
        None => {
            cmd = Command::new(foxkit);
            cmd.args(args).arg("--json");
        }
    }
    if let Some(d) = cwd.filter(|d| Path::new(d).is_dir()) {
        cmd.current_dir(d);
    }
    cmd.stdin(Stdio::null()).stdout(Stdio::piped()).stderr(Stdio::piped());
    #[cfg(windows)]
    {
        use std::os::windows::process::CommandExt;
        cmd.creation_flags(CREATE_NO_WINDOW);
    }
    cmd
}

fn spawn_error(e: std::io::Error, exec: Option<&ExecSpec>) -> CmdError {
    match (e.kind(), exec) {
        (std::io::ErrorKind::NotFound, Some(x)) => CmdError::new(
            "EXTERNAL_NOT_FOUND",
            format!("Không tìm thấy chương trình {}", x.program),
            "Kiểm tra đường dẫn program trong file foxkit.json, hoặc gõ :scan nếu tool đã bị đổi/xóa",
        ),
        (std::io::ErrorKind::NotFound, None) => foxkit_not_found(),
        _ => CmdError::new("SPAWN_FAILED", format!("Không chạy được lệnh: {e}"), ""),
    }
}

pub fn run_blocking(foxkit: Option<&Path>, args: &[String], cwd: Option<&str>, exec: Option<&ExecSpec>) -> Result<RunResult, CmdError> {
    let foxkit = foxkit.map(Path::to_path_buf).or_else(|| exec.map(|_| PathBuf::new())).ok_or_else(foxkit_not_found)?;
    let out = build_command(&foxkit, args, cwd, exec).output().map_err(|e| spawn_error(e, exec))?;
    Ok(RunResult {
        exit_code: out.status.code().unwrap_or(-1),
        stdout: String::from_utf8_lossy(&out.stdout).into_owned(),
        stderr: String::from_utf8_lossy(&out.stderr).into_owned(),
    })
}

/// Các lệnh đang chạy ở chế độ stream, để `stop` kill được.
#[derive(Default)]
pub struct Runs {
    next: AtomicU32,
    children: Mutex<HashMap<u32, Arc<Mutex<Child>>>>,
}

/// Đọc từng dòng (chịu được byte không phải UTF-8) và gọi `f`.
fn read_lines(r: impl Read, mut f: impl FnMut(String)) {
    let mut br = BufReader::new(r);
    let mut buf = Vec::new();
    loop {
        buf.clear();
        match br.read_until(b'\n', &mut buf) {
            Ok(0) | Err(_) => break,
            Ok(_) => f(String::from_utf8_lossy(&buf).trim_end_matches(['\r', '\n']).to_string()),
        }
    }
}

pub fn run_stream(
    runs: &Arc<Runs>,
    foxkit: Option<&Path>,
    args: &[String],
    cwd: Option<&str>,
    exec: Option<&ExecSpec>,
    on_line: Channel<StreamMsg>,
) -> Result<u32, CmdError> {
    let foxkit = foxkit.map(Path::to_path_buf).or_else(|| exec.map(|_| PathBuf::new())).ok_or_else(foxkit_not_found)?;
    let mut child = build_command(&foxkit, args, cwd, exec).spawn().map_err(|e| spawn_error(e, exec))?;
    let stdout = child.stdout.take();
    let stderr = child.stderr.take();
    let id = runs.next.fetch_add(1, Ordering::SeqCst) + 1;
    let child = Arc::new(Mutex::new(child));
    runs.children.lock().unwrap().insert(id, child.clone());

    let (c1, c2, c3) = (on_line.clone(), on_line.clone(), on_line);
    let h1 = stdout.map(|o| std::thread::spawn(move || read_lines(o, |text| drop(c1.send(StreamMsg::Line { text })))));
    let h2 = stderr.map(|e| std::thread::spawn(move || read_lines(e, |text| drop(c2.send(StreamMsg::Stderr { text })))));
    let runs = runs.clone();
    std::thread::spawn(move || {
        for h in [h1, h2].into_iter().flatten() {
            let _ = h.join(); // luồng đọc chỉ kết thúc khi pipe đóng; lỗi panic của luồng đọc không ảnh hưởng exit code
        }
        let code = child.lock().unwrap().wait().ok().and_then(|s| s.code()).unwrap_or(-1);
        runs.children.lock().unwrap().remove(&id);
        let _ = c3.send(StreamMsg::Exit { code }); // frontend đóng cửa sổ thì không còn ai nhận
    });
    Ok(id)
}

/// Dừng một lệnh stream (kill). Lệnh đã kết thúc thì bỏ qua.
pub fn stop(runs: &Runs, id: u32) {
    let child = runs.children.lock().unwrap().get(&id).cloned();
    if let Some(c) = child {
        let _ = c.lock().unwrap().kill(); // đã thoát sẵn thì kill báo lỗi, không sao
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn args_of(c: &Command) -> Vec<String> {
        c.get_args().map(|a| a.to_string_lossy().into_owned()).collect()
    }

    #[test]
    fn foxkit_command_appends_json() {
        let c = build_command(Path::new("foxkit.exe"), &["port".into(), "8000".into()], None, None);
        assert_eq!(args_of(&c), ["port", "8000", "--json"]);
    }

    #[test]
    fn external_command_order_and_json_flag() {
        let spec = ExecSpec { program: "tool".into(), prefix: vec!["-x".into()], json: false, base_dir: ".".into() };
        let c = build_command(Path::new(""), &["a".into(), "--w".into(), "1".into()], None, Some(&spec));
        assert_eq!(args_of(&c), ["-x", "a", "--w", "1"]);
        let spec = ExecSpec { json: true, ..spec };
        assert_eq!(args_of(&build_command(Path::new(""), &[], None, Some(&spec))), ["-x", "--json"]);
    }

    #[test]
    fn args_are_not_interpreted_by_shell() {
        let c = build_command(Path::new("foxkit.exe"), &["a & b".into(), "$(x)".into()], None, None);
        assert_eq!(args_of(&c)[..2], ["a & b", "$(x)"]);
    }

    #[test]
    fn missing_foxkit_is_reported() {
        let e = run_blocking(None, &[], None, None).unwrap_err();
        assert_eq!(e.code, "FOXKIT_NOT_FOUND");
    }

    #[test]
    fn missing_external_program_is_reported() {
        let spec = ExecSpec { program: "definitely-not-a-real-program-xyz".into(), prefix: vec![], json: false, base_dir: ".".into() };
        let e = run_blocking(None, &[], None, Some(&spec)).unwrap_err();
        assert_eq!(e.code, "EXTERNAL_NOT_FOUND");
    }

    #[cfg(windows)]
    #[test]
    fn runs_external_cmd_and_captures_stdout() {
        let spec = ExecSpec { program: "cmd".into(), prefix: vec!["/c".into(), "echo".into()], json: false, base_dir: ".".into() };
        let r = run_blocking(None, &["xin".into(), "chao".into()], None, Some(&spec)).unwrap();
        assert_eq!(r.exit_code, 0);
        assert!(r.stdout.contains("xin chao"));
    }
}
