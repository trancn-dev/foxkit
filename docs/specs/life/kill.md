# Spec: `foxkit kill`

> Nhóm: **life** · Giai đoạn: GĐ3 · Trạng thái: chưa làm · Cần trước: [life/ps.md](ps.md)

## 1. Mục đích
Kill process theo PID hoặc theo tên (kill hết `node.exe` bị treo sau khi tắt terminal).

## 2. Cú pháp
```
foxkit kill <pid|name>... [--tree] [--json]
```
| Tham số/cờ | Ghi chú |
|---|---|
| `<pid\|name>...` | toàn số → PID; còn lại → tên. Tên thiếu `.exe` thì tự thêm. So tên không phân biệt hoa thường, khớp **chính xác** (không phải chuỗi con, tránh kill nhầm) |
| `--tree` | kill cả process con (vd `npm run dev` → `node` → `esbuild`) |

## 3. File
```
cmd/kill.go
internal/kill/kill.go
internal/kill/kill_test.go
```

## 4. Hợp đồng
```go
package kill
type Target struct { PID int `json:"pid"`; Name string `json:"name"` }
type Failure struct { Target; Code string `json:"code"`; Message string `json:"message"` }
type Result struct {
    Killed []Target  `json:"killed"`
    Failed []Failure `json:"failed"`
}
func Resolve(ctx context.Context, args []string) ([]Target, error) // tên → danh sách PID qua ps.List
func Kill(ctx context.Context, targets []Target, tree bool) Result
```

## 5. Cách hoạt động
1. `Resolve`: một lần gọi `ps.List`, map tên → các PID. Tên không có process nào → lỗi `PROC_NOT_FOUND` cho tên đó (thêm vào `Failed`, không dừng các tên khác).
2. Danh sách chặn (không bao giờ kill, trả `PROC_PROTECTED`): PID 0, 4, PID của chính foxkit (`os.Getpid()`) và process cha (`os.Getppid()`), cùng các tên: `system`, `smss.exe`, `csrss.exe`, `wininit.exe`, `winlogon.exe`, `services.exe`, `lsass.exe`, `svchost.exe`, `explorer.exe`, `dwm.exe`.
3. Không `--tree`: `os.FindProcess(pid)` → `.Kill()`.
4. `--tree`: `taskkill /PID <pid> /T /F` qua `winexec.Run` (Go không có sẵn API cây process; GĐ6 có thể thay bằng API).
5. Access denied → `NEED_ADMIN` trong `Failed`.
6. Exit code: tất cả thành công → 0; có ít nhất 1 thất bại do quyền và không có lỗi khác → 3; còn lại có thất bại → 1. Vẫn in `Result` đầy đủ ra stdout.

## 6. Output
```
Đã kill: node.exe (9812), node.exe (10344)
Thất bại: MsMpEng.exe (3120) — cần quyền admin
```
JSON: `{ "ok": true, "data": <Result> }` khi exit 0; exit khác 0 thì stdout vẫn có data và stderr có envelope lỗi tổng `KILL_PARTIAL`/`NEED_ADMIN`.

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `KILL_NO_TARGET` | thiếu tham số | 2 | Ví dụ: foxkit kill node |
| `PROC_NOT_FOUND` | (theo item) | 1 | Xem danh sách bằng foxkit ps |
| `PROC_PROTECTED` | (theo item) | 1 | Process hệ thống, không được kill |
| `NEED_ADMIN` | | 3 | Chạy lại bằng quyền Administrator |
| `KILL_PARTIAL` | lỗi tổng khi có item thất bại | 1 | |

## 8. Manifest
```json
{ "name": "kill", "title": "Kill process", "description": "Kill process theo PID hoặc tên", "category": "system",
  "args": [ { "name": "target", "type": "string", "required": true, "multiple": true, "label": "PID hoặc tên" } ],
  "flags": [ { "name": "tree", "type": "bool", "default": false, "label": "Kill cả process con" } ],
  "admin": false, "confirm": "Kill các process này?", "longRunning": false, "dryRun": false, "output": "kv" }
```

## 9. Test
`Resolve` với danh sách ps giả (inject): `node` → 2 PID; `NODE.EXE` giống `node`; `svchost` → `PROC_PROTECTED`; `abc` → `PROC_NOT_FOUND`. Phần kill thật: test thủ công.

Chạy thử:
```powershell
Start-Process notepad; Start-Process notepad
.\foxkit kill notepad
.\foxkit kill explorer          # phải bị chặn
```

## 10. Tiêu chí hoàn thành
Tiêu chí chung + không thể kill process trong danh sách chặn.

## 11. Ngoài phạm vi
Kill theo cửa sổ, theo port (đã có `port --kill`), chờ process thoát êm (gửi WM_CLOSE).

## 12. Go học được
Thiết kế kết quả một phần (`Killed`/`Failed`), map exit code từ nhiều lỗi, inject dependency để test (truyền hàm `listFn` thay vì gọi thẳng `ps.List`).
