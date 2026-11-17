use crate::runner::{CmdError, RunResult};
use std::path::Path;

/// Ký tự đặc biệt của cmd.exe: từ chối khi chạy admin (launcher.md 6.7 bước 4).
const UNSAFE: [char; 7] = ['"', '&', '|', '<', '>', '^', '%'];

fn unsafe_arg() -> CmdError {
    CmdError::new(
        "ELEVATE_UNSAFE_ARG",
        "Tham số có ký tự không chạy được ở chế độ admin",
        "Mở terminal Administrator và chạy lệnh trực tiếp",
    )
}

/// Dựng `lpParameters` cho `cmd.exe`: `/d /s /c ""<foxkit>" "a" "b" --json 1>"<out>" 2>"<err>""`.
pub fn build_params(foxkit: &str, args: &[String], out: &Path, err: &Path) -> Result<String, CmdError> {
    let out_s = out.to_string_lossy();
    let err_s = err.to_string_lossy();
    for s in args.iter().map(String::as_str).chain([foxkit, &out_s, &err_s]) {
        if s.contains(UNSAFE) {
            return Err(unsafe_arg());
        }
    }
    let quoted: Vec<String> = args.iter().map(|a| format!("\"{a}\"")).collect();
    Ok(format!("/d /s /c \"\"{foxkit}\" {} --json 1>\"{out_s}\" 2>\"{err_s}\"\"", quoted.join(" ")))
}

#[cfg(windows)]
pub fn run_elevated(foxkit: &Path, args: &[String], cwd: Option<&str>) -> Result<RunResult, CmdError> {
    use windows::core::{w, HSTRING, PCWSTR};
    use windows::Win32::Foundation::{CloseHandle, GetLastError, WAIT_OBJECT_0};
    use windows::Win32::System::Threading::{GetExitCodeProcess, WaitForSingleObject, INFINITE};
    use windows::Win32::UI::Shell::{ShellExecuteExW, SEE_MASK_NOCLOSEPROCESS, SHELLEXECUTEINFOW};
    use windows::Win32::UI::WindowsAndMessaging::SW_HIDE;

    let id = std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH).map(|d| d.as_nanos()).unwrap_or(0);
    let dir = std::env::temp_dir().join("foxkit-launcher").join(id.to_string());
    std::fs::create_dir_all(&dir).map_err(|e| CmdError::new("ELEVATE_FAILED", format!("Không tạo được thư mục tạm: {e}"), ""))?;
    let (out_f, err_f) = (dir.join("out.json"), dir.join("err.json"));
    let result = (|| {
        let params = HSTRING::from(build_params(&foxkit.to_string_lossy(), args, &out_f, &err_f)?);
        let cwd_h = cwd.filter(|d| Path::new(d).is_dir()).map(HSTRING::from);
        let mut info = SHELLEXECUTEINFOW {
            cbSize: std::mem::size_of::<SHELLEXECUTEINFOW>() as u32,
            fMask: SEE_MASK_NOCLOSEPROCESS,
            lpVerb: w!("runas"),
            lpFile: w!("cmd.exe"),
            lpParameters: PCWSTR(params.as_ptr()),
            lpDirectory: cwd_h.as_ref().map_or(PCWSTR::null(), |h| PCWSTR(h.as_ptr())),
            nShow: SW_HIDE.0,
            ..Default::default()
        };
        // SAFETY: `info` và các HSTRING (params, cwd_h) sống tới hết khối này; hProcess do ShellExecuteExW trả về, đóng bằng CloseHandle.
        unsafe {
            if ShellExecuteExW(&mut info).is_err() {
                return Err(if GetLastError().0 == 1223 {
                    CmdError::new("ELEVATE_CANCELLED", "Đã hủy chạy quyền admin", "")
                } else {
                    CmdError::new("ELEVATE_FAILED", "Không chạy được với quyền admin", "")
                });
            }
            let h = info.hProcess;
            let mut code = 0u32;
            if WaitForSingleObject(h, INFINITE) == WAIT_OBJECT_0 {
                let _ = GetExitCodeProcess(h, &mut code); // lỗi thì giữ code 0 rồi dựa vào nội dung file
            }
            let _ = CloseHandle(h);
            let read = |p: &Path| std::fs::read(p).map(|b| String::from_utf8_lossy(&b).into_owned()).unwrap_or_default();
            Ok(RunResult { exit_code: code as i32, stdout: read(&out_f), stderr: read(&err_f) })
        }
    })();
    let _ = std::fs::remove_dir_all(&dir); // dọn thư mục tạm, lỗi xóa không đáng báo
    result
}

#[cfg(not(windows))]
pub fn run_elevated(_foxkit: &Path, _args: &[String], _cwd: Option<&str>) -> Result<RunResult, CmdError> {
    Err(CmdError::new("ELEVATE_FAILED", "Chạy admin chỉ hỗ trợ trên Windows", ""))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn params_quote_each_arg() {
        let p = build_params(r"C:\foxkit\foxkit.exe", &["hosts".into(), "add".into(), "my host".into()], Path::new(r"C:\t\out.json"), Path::new(r"C:\t\err.json")).unwrap();
        assert_eq!(p, r#"/d /s /c ""C:\foxkit\foxkit.exe" "hosts" "add" "my host" --json 1>"C:\t\out.json" 2>"C:\t\err.json"""#);
    }

    #[test]
    fn unsafe_chars_are_rejected() {
        for bad in ["a&b", "a|b", "a<b", "a>b", "a^b", "50%", "a\"b"] {
            let e = build_params("foxkit.exe", &[bad.to_string()], Path::new("o"), Path::new("e")).unwrap_err();
            assert_eq!(e.code, "ELEVATE_UNSAFE_ARG", "{bad}");
        }
    }
}
