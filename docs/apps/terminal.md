# Terminal / PowerShell

> exe: `WindowsTerminal.exe`, `powershell.exe`, `pwsh.exe`, `cmd.exe` · Ngữ cảnh: chỉ tên app + tiêu đề; **chưa biết thư mục đang đứng (cwd)**.

## 1. Lệnh có sẵn, dùng ngay (appHints)
| Lệnh | Dùng khi |
|---|---|
| `which` | Gõ `php`/`node` thì chạy bản nào trong PATH (liệt kê mọi bản) |
| `path check`, `path` | PATH có mục chết, trùng, rỗng? |
| `port` | Ai đang chiếm cổng; `--kill` để giải phóng |
| `ps`, `kill` | Process nào ngốn RAM; kill theo PID/tên (có chặn process hệ thống) |
| `env get`, `env diff` | Đọc/so `.env` mà không in lộ bí mật |
| `lock` | "File is being used by another process" → ai giữ |
| `hosts`, `dns`, `ip` | Debug mạng nhanh |
| `ts`, `uuid`, `pass`, `createpw` | Tiện ích sinh/đổi nhanh khi đang gõ lệnh |

Mẫu: `{ "app": "WindowsTerminal.exe", "commands": ["port", "which", "ps", "lock", "env get"] }`.

## 2. Tool nhỏ mới đề xuất
| Lệnh đề xuất | Việc làm | Khó | Giá trị | Dependency |
|---|---|---|---|---|
| `wait-port` | Chờ cổng mở/đóng với timeout; thoát 0/1 cho script (`foxkit wait-port 3306 && ...`) | 1 | ★★ | stdlib |
| `open` | Mở file/thư mục/URL bằng app mặc định, hoặc mở Explorer tại thư mục (`open .`) | 1 | ★★ | stdlib + `x/sys` (`ShellExecute`) |
| `envs` | Liệt kê biến môi trường của user/máy có lọc, che giá trị nhạy cảm; so hai tập | 2 | ★★ | stdlib + `x/sys/windows/registry` |
| `retry` | Chạy lại một lệnh tối đa N lần đến khi exit 0 (mạng chập chờn) | 1 | ★ | stdlib (`os/exec`); lưu ý quy tắc an toàn khi chạy lệnh tùy ý |
| `alias` | Quản lý alias PowerShell trong `$PROFILE` có backup | 3 | ★ | stdlib — sửa file hệ thống nên phải plan/apply; ưu tiên thấp |

## 3. Việc launcher chưa làm được
- **Lấy cwd của shell** để `@dir` hoạt động trong terminal: đọc PEB của process con (`NtQueryInformationProcess` + `ReadProcessMemory`) — khó 4, phụ thuộc phiên bản Windows, cần quyền đọc process; Windows Terminal có nhiều tab nên phải tìm đúng process con đang focus. Hướng thay thế đơn giản: người dùng gõ `pwd | clip` rồi dùng `@clip`.
- Chạy lệnh **trong** cửa sổ terminal hiện tại (dán lệnh vào): cần giả phím/`SendInput`, **không đề xuất** (đã gây đổi CapsLock khi thử).
- `cmd.exe` mở bằng `conhost` cũ có thể báo tên exe khác; kiểm lại khi cấu hình `appHints`.
