# Spec: `foxkit startup`

> Nhóm: **life** · Giai đoạn: GĐ6 · Cần trước: [core/winapi.md](../core/winapi.md)

## 1. Mục đích
Liệt kê các chương trình tự chạy khi đăng nhập Windows (máy khởi động chậm vì Docker, Teams, OneDrive, updater...). **v1 chỉ đọc.**

## 2. Cú pháp
```
foxkit startup [--json]
```

## 3. File
```
cmd/startup.go
internal/startup/startup_windows.go
internal/startup/startup_other.go
internal/startup/startup_test.go
```

## 4. Hợp đồng
```go
package startup
type Item struct {
    Name    string `json:"name"`
    Command string `json:"command"`
    Source  string `json:"source"`  // "HKCU Run", "HKLM Run", "HKLM Run (32-bit)", "Startup folder (user)", "Startup folder (all users)"
    Scope   string `json:"scope"`   // user | machine
    Enabled bool   `json:"enabled"`
    Exists  bool   `json:"exists"`  // file thực thi còn tồn tại không
}
func List() ([]Item, error)
```

## 5. Cách hoạt động
**Nguồn**
| Source | Vị trí |
|---|---|
| HKCU Run | `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` |
| HKLM Run | `HKLM\Software\Microsoft\Windows\CurrentVersion\Run` |
| HKLM Run (32-bit) | `HKLM\Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Run` |
| Startup folder (user) | `%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup` |
| Startup folder (all users) | `%ProgramData%\Microsoft\Windows\Start Menu\Programs\StartUp` |

(`RunOnce` bỏ qua vì chỉ chạy một lần.)

1. Registry: `k.ReadValueNames(-1)` → mỗi tên `GetStringValue` (kiểu SZ hoặc EXPAND_SZ; EXPAND_SZ thì `registry.ExpandString`).
2. Thư mục Startup: liệt kê file (bỏ `desktop.ini`). File `.lnk`: v1 chỉ hiện đường dẫn shortcut (đọc target cần COM `IShellLink`, ngoài phạm vi).
3. **Trạng thái bật/tắt** (do Task Manager ghi) nằm ở khóa `...\Explorer\StartupApproved\Run`, `Run32`, `StartupFolder` (dưới HKCU hoặc HKLM tương ứng): giá trị binary cùng tên; **byte đầu là số chẵn (0x02, 0x06) = bật, số lẻ (0x03) = tắt**. Không có giá trị → bật. (Quy ước này không có tài liệu chính thức, xác nhận lại trên máy thật khi làm bằng cách tắt/bật một mục trong Task Manager rồi xem khóa.)
4. `Exists`: tách đường dẫn exe khỏi command (có nháy kép thì lấy phần trong nháy, không thì đến `.exe` đầu tiên) → `os.Stat`.
5. Đọc HKLM không cần admin.
6. Sắp: bật trước, rồi theo tên.

## 6. Output
```
TRẠNG THÁI  TÊN              NGUỒN          LỆNH
bật         Docker Desktop   HKCU Run       "C:\Program Files\Docker\Docker\Docker Desktop.exe" -Autostart
bật         OneDrive         HKCU Run       "C:\Users\me\AppData\Local\Microsoft\OneDrive\OneDrive.exe" /background
tắt         Spotify          HKCU Run       ...
bật ✗       OldUpdater       HKLM Run       C:\Program Files\Old\upd.exe   (file không còn)
Tắt bớt: Task Manager → Startup apps.
```
JSON: `{ "ok": true, "data": { "items": [Item...] } }`

## 7. Lỗi
Nguồn nào lỗi thì bỏ qua và ghi cảnh báo; chỉ `STARTUP_FAILED` khi mọi nguồn đều lỗi.

## 8. Manifest
```json
{ "name": "startup", "title": "Chương trình khởi động", "description": "App tự chạy khi đăng nhập Windows", "category": "system",
  "args": [], "flags": [], "admin": false, "longRunning": false, "dryRun": false, "output": "table" }
```

## 9. Test
Logic thuần: tách exe từ command (bảng: `"C:\a b\x.exe" -y` → `C:\a b\x.exe`; `C:\x.exe /s` → `C:\x.exe`; `rundll32.exe a.dll,Run` → `rundll32.exe`), đọc byte trạng thái (`[]byte{0x03,...}` → tắt).
Registry: test Windows đọc khóa tạm do test tạo.

Chạy thử: `.\foxkit startup`, `.\foxkit startup --json`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + trạng thái bật/tắt khớp Task Manager trên máy thật.

## 11. Ngoài phạm vi
Bật/tắt mục (sửa `StartupApproved`), Scheduled Tasks chạy lúc logon, services tự khởi động (xem `svc`), đọc target `.lnk`.

## 12. Go học được
Duyệt registry (`ReadValueNames`, `GetBinaryValue`), `registry.ExpandString`, chịu lỗi từng nguồn.
