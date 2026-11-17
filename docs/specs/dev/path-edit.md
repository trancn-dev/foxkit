# Spec: `foxkit path add` / `foxkit path rm` / `foxkit path` (User/System)

> Nhóm: **dev** · Giai đoạn: GĐ6 · Cần trước: [dev/path.md](path.md), [core/winapi.md](../core/winapi.md)

## 1. Mục đích
Thêm/xóa thư mục trong PATH **vĩnh viễn** (User hoặc System) an toàn, và nâng cấp `foxkit path` để biết mỗi mục đến từ User hay System. Thay cho việc mở Settings → Environment Variables.

## 2. Cú pháp
```
foxkit path [--scope all|user|system] [--json]           # nâng cấp lệnh GĐ3
foxkit path add <dir> [--system] [--first] [--apply] [--json]
foxkit path rm <dir> [--system] [--apply] [--json]
foxkit path clean [--system] [--apply] [--json]           # xóa mục không tồn tại + trùng
```
| Cờ | Ghi chú |
|---|---|
| `--system` | sửa PATH System (cần admin); mặc định User |
| `--first` | thêm lên đầu (ưu tiên cao nhất); mặc định thêm cuối |
| `--apply` | ghi registry |

## 3. File
```
cmd/path_add.go, path_rm.go, path_clean.go     # cmd/path.go đã có, thêm --scope
internal/winpath/registry_windows.go
internal/winpath/registry_other.go
internal/winpath/edit.go                       # logic thuần, test được mọi nơi
internal/winpath/edit_test.go
```

## 4. Hợp đồng
```go
package winpath
type Scope string // "user" | "system"
type Raw struct { Value string; Expand bool } // Expand = kiểu REG_EXPAND_SZ
func Read(scope Scope) (Raw, error)
func Write(scope Scope, r Raw) error
func Broadcast() error // WM_SETTINGCHANGE

// Logic thuần (không đụng registry)
func Add(value, dir string, first bool) (newValue string, changed bool)
func Remove(value, dir string) (newValue string, removed int)
func Clean(value string, exists func(string) bool) (newValue string, removed []string)
```

## 5. Cách hoạt động
**Vị trí registry**
| Scope | Khóa | Giá trị |
|---|---|---|
| user | `HKCU\Environment` | `Path` |
| system | `HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment` | `Path` |

**Đọc**: `registry.OpenKey(..., registry.QUERY_VALUE)` → `k.GetStringValue("Path")` trả cả chuỗi **chưa expand** và kiểu (`registry.EXPAND_SZ` hoặc `SZ`). Không có giá trị `Path` (user mới) → chuỗi rỗng, kiểu EXPAND_SZ.

**Sửa (logic thuần)**
1. Tách `;`, giữ nguyên từng mục gốc (kể cả `%USERPROFILE%\...`).
2. So sánh trùng bằng dạng chuẩn hóa: expand `%VAR%` → `filepath.Clean` → bỏ `\` cuối → `ToLower`. **Chỉ để so sánh**, không ghi dạng đã expand.
3. `Add`: đã có → không đổi. `dir` truyền vào được chuyển thành đường dẫn tuyệt đối; nếu nằm dưới `%USERPROFILE%` thì ghi lại dạng `%USERPROFILE%\...` để giữ tính di động (chỉ với scope user).
4. `Remove`: xóa mọi mục khớp.
5. `Clean`: xóa mục rỗng, không tồn tại, và trùng (giữ lần xuất hiện đầu).
6. Ghép bằng `;`, không để `;` thừa ở đầu/cuối.

**Ghi** (`--apply`)
1. Backup giá trị cũ ra `%LOCALAPPDATA%\foxkit\backup\path-<scope>-<yyyyMMdd-HHmmss>.txt`.
2. **Giữ kiểu**: gốc là EXPAND_SZ (hoặc giá trị mới có `%`) → `SetExpandStringValue`; ngược lại `SetStringValue`. Ghi sai kiểu SZ khi có `%VAR%` là lỗi kinh điển làm hỏng PATH (biến không được expand nữa).
3. Cảnh báo nếu độ dài > 2047 ký tự; chặn nếu > 32767.
4. `Broadcast`: `SendMessageTimeoutW(HWND_BROADCAST=0xFFFF, WM_SETTINGCHANGE=0x1A, 0, "Environment", SMTO_ABORTIFHUNG=0x2, 5000, &result)` qua LazyDLL `user32.dll`. Explorer nhận → app mở **mới** sẽ thấy PATH mới.
5. System scope mà không admin → `NEED_ADMIN` (exit 3) trước khi ghi gì.

**`path` nâng cấp**: đọc cả 2 scope, mỗi `Entry` thêm `"scope": "system"|"user"|"process"` (process = có trong PATH hiện tại nhưng không thấy trong registry, ví dụ do terminal/IDE thêm vào). Windows ghép System trước rồi User.

## 6. Output
```
> foxkit path add C:\tools\bin
PATH (user) sẽ thêm vào cuối:
  + C:\tools\bin
Đây là bản xem trước. Thêm --apply để thực hiện.

> foxkit path add C:\tools\bin --apply
Đã thêm C:\tools\bin vào PATH (user). Backup: C:\Users\me\AppData\Local\foxkit\backup\path-user-20261116-201500.txt
Lưu ý: terminal/VS Code đang mở cần mở lại mới thấy PATH mới.
```
JSON: `{ "ok": true, "data": { "scope": "user", "applied": true, "added": ["C:\\tools\\bin"], "removed": [], "backup": "...", "length": 1904 } }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `PATH_DIR_NOT_FOUND` | add thư mục không tồn tại | 1 | Kiểm tra lại, hoặc tạo thư mục trước |
| `PATH_NOT_IN_PATH` | rm mục không có | 1 | Xem danh sách: foxkit path --scope user |
| `PATH_TOO_LONG` | > 32767 | 1 | Dọn bớt bằng foxkit path clean |
| `NEED_ADMIN` | `--system` không admin | 3 | Chạy terminal bằng quyền Administrator |
| `REGISTRY_FAILED` | | 1 | |

## 8. Manifest
```json
{ "name": "path add", "title": "Thêm vào PATH", "description": "Thêm thư mục vào PATH vĩnh viễn", "category": "dev",
  "args": [ { "name": "dir", "type": "dir", "required": true, "label": "Thư mục" } ],
  "flags": [
    { "name": "system", "type": "bool", "default": false, "label": "PATH hệ thống (admin)" },
    { "name": "first", "type": "bool", "default": false, "label": "Thêm lên đầu" },
    { "name": "apply", "type": "bool", "default": false, "label": "Ghi" }
  ], "admin": false, "longRunning": false, "dryRun": true, "output": "kv" }
```
`path rm` tương tự (không `--first`); `path clean` không có arg; entry `path` thêm flag `scope` (enum `all,user,system`, mặc định `all`).

## 9. Test
Logic thuần (bảng, chạy mọi OS):
- Add trùng khác hoa thường/khác `\` cuối → không đổi.
- Add khi giá trị có `%SystemRoot%\system32` → giữ nguyên chuỗi `%SystemRoot%`.
- Remove khớp dạng expand (`t.Setenv("USERPROFILE", ...)`).
- Clean bỏ `;;`, mục không tồn tại (inject `exists`).
Registry: test trên Windows ghi vào khóa tạm `HKCU\Software\foxkit-test` (không đụng `Environment`), kiểm tra giữ kiểu EXPAND_SZ, rồi xóa khóa.

Chạy thử:
```powershell
.\foxkit path --scope user
.\foxkit path add C:\tools\bin
.\foxkit path add C:\tools\bin --apply
# mở terminal mới
.\foxkit which mytool
.\foxkit path rm C:\tools\bin --apply
```

## 10. Tiêu chí hoàn thành
Tiêu chí chung + không bao giờ ghi SZ đè EXPAND_SZ + luôn có backup.

## 11. Ngoài phạm vi
Sửa biến môi trường khác ngoài PATH, khôi phục từ backup (làm tay bằng file backup).

## 12. Go học được
`golang.org/x/sys/windows/registry`, tách logic thuần khỏi phần chạm hệ thống để test (≈ tách Service khỏi Facade), gọi `SendMessageTimeoutW` qua LazyDLL.
