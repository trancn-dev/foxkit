# Spec: `foxkit lock`

> Nhóm: **life** · Giai đoạn: GĐ6 · Cần trước: [core/winapi.md](../core/winapi.md), [life/kill.md](kill.md)
> Lệnh khó nhất của roadmap. Nên làm cuối.

## 1. Mục đích
Trả lời "process nào đang giữ file/thư mục này?" khi Windows báo *"The action can't be completed because the file is open in another program"* (xóa `node_modules`, đổi tên thư mục dự án, xóa file log). Giống PowerToys File Locksmith / Resource Monitor.

## 2. Cú pháp
```
foxkit lock <path>... [--kill] [--max-files 5000] [--json]
```
| Tham số/cờ | Ghi chú |
|---|---|
| `<path>...` | file hoặc thư mục |
| `--kill` | kill các process tìm được (dùng lại `internal/kill`, có danh sách bảo vệ) |
| `--max-files` | với thư mục: số file tối đa đăng ký kiểm tra (mặc định 5000, tối đa 50000) |

## 3. File
```
cmd/lock.go
internal/lock/rm_windows.go     # bọc Restart Manager
internal/lock/rm_other.go
internal/lock/lock.go
internal/lock/lock_test.go
```

## 4. Hợp đồng
```go
package lock
type Holder struct {
    PID       uint32 `json:"pid"`
    Name      string `json:"name"`      // tên exe
    App       string `json:"app"`       // tên hiển thị từ Restart Manager, vd "Visual Studio Code"
    Service   string `json:"service,omitempty"`
    Type      string `json:"type"`      // mainWindow|otherWindow|service|explorer|console|critical|unknown
    Path      string `json:"path,omitempty"` // đường dẫn exe (winapi.ProcessPath)
}
type Result struct {
    Paths      []string `json:"paths"`
    Checked    int      `json:"checked"`     // số file đã đăng ký
    Truncated  bool     `json:"truncated"`   // thư mục có nhiều file hơn --max-files
    Holders    []Holder `json:"holders"`
    Killed     []int    `json:"killed,omitempty"`
}
func Find(paths []string, maxFiles int) (Result, error)
```

## 5. Cách hoạt động
Dùng **Restart Manager** (`rstrtmgr.dll`) — API chính thức Windows dùng để biết app nào giữ file (installer dùng nó). Không có trong `x/sys/windows` → gọi qua `windows.NewLazySystemDLL("rstrtmgr.dll")`.

**Hàm & struct**
```
DWORD RmStartSession(DWORD *pSessionHandle, DWORD dwSessionFlags, WCHAR strSessionKey[CCH_RM_SESSION_KEY+1]);   // CCH_RM_SESSION_KEY = 32
DWORD RmRegisterResources(DWORD dwSessionHandle, UINT nFiles, LPCWSTR rgsFilenames[], UINT nApplications, RM_UNIQUE_PROCESS rgApplications[], UINT nServices, LPCWSTR rgsServiceNames[]);
DWORD RmGetList(DWORD dwSessionHandle, UINT *pnProcInfoNeeded, UINT *pnProcInfo, RM_PROCESS_INFO rgAffectedApps[], LPDWORD lpdwRebootReasons);
DWORD RmEndSession(DWORD dwSessionHandle);
```
```go
type rmUniqueProcess struct {
    ProcessId        uint32
    ProcessStartTime windows.Filetime
}
type rmProcessInfo struct {
    Process             rmUniqueProcess
    AppName             [256]uint16 // CCH_RM_MAX_APP_NAME + 1
    ServiceShortName    [64]uint16  // CCH_RM_MAX_SVC_NAME + 1
    ApplicationType     uint32      // RM_APP_TYPE: 0 Unknown, 1 MainWindow, 2 OtherWindow, 3 Service, 4 Explorer, 5 Console, 1000 Critical
    AppStatus           uint32
    TSSessionId         uint32
    Restartable         int32       // BOOL
}
```
**Lưu ý**: các hàm Rm* **trả mã lỗi trực tiếp** (DWORD, 0 = thành công), không dùng GetLastError → kiểm `r1 != 0` và chuyển `syscall.Errno(r1)`.

**Luồng**
1. Mở rộng đầu vào: file → giữ nguyên; thư mục → `WalkDir` lấy file (không đi vào junction), tối đa `--max-files`, quá thì `Truncated = true`. Đường dẫn tuyệt đối.
2. `RmStartSession(&h, 0, &key[0])` với `key [33]uint16`; `defer RmEndSession(h)`.
3. `RmRegisterResources`: chuyển danh sách path thành `[]*uint16`, truyền `&ptrs[0]`. Đăng ký theo lô 1000 file (một lần quá nhiều có thể lỗi).
   - Giữ slice `ptrs` và các chuỗi UTF-16 sống tới sau lời gọi (dùng `runtime.KeepAlive(ptrs)`) để GC không thu hồi.
4. `RmGetList` lần 1 với `pnProcInfo = 0` → nhận `ERROR_MORE_DATA` (234) và `needed`. Cấp `make([]rmProcessInfo, needed)`, gọi lần 2. Nếu vẫn `ERROR_MORE_DATA` (process mới xuất hiện) → lặp lại tối đa 3 lần.
5. Map kết quả → `Holder`; khử trùng theo PID; `Name` từ `winapi.ProcessPath` (lấy base name), lỗi quyền thì dùng `App`.
6. `--kill`: gọi `kill.Kill` với các PID (process `critical`/`explorer` bị chặn theo danh sách bảo vệ, Explorer chỉ gợi ý restart chứ không kill).
7. Không ai giữ → `holders: []`, exit 0, text: "Không có process nào đang giữ. Nếu vẫn không xóa được: kiểm tra quyền, thuộc tính read-only, hoặc phần mềm diệt virus đang quét."

## 6. Output
```
D:\old-shop\node_modules  (đã kiểm 4.812 file)
PID     TIẾN TRÌNH   ỨNG DỤNG              LOẠI
9812    node.exe     Node.js JavaScript…   console
15020   Code.exe     Visual Studio Code    mainWindow
Gợi ý: đóng các ứng dụng trên, hoặc chạy lại với --kill.
```
JSON: `{ "ok": true, "data": <Result> }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `PATH_NOT_FOUND` | | 2 | |
| `MAX_FILES_INVALID` | | 2 | |
| `RM_FAILED` | API Restart Manager lỗi | 1 | Kèm mã lỗi Windows |
| `NEED_ADMIN` | kill bị từ chối | 3 | |

## 8. Manifest
```json
{ "name": "lock", "title": "Ai đang giữ file?", "description": "Tìm process đang khóa file/thư mục", "category": "system",
  "args": [ { "name": "path", "type": "file", "allowDir": true, "required": true, "multiple": true, "label": "File hoặc thư mục" } ],
  "flags": [
    { "name": "kill", "type": "bool", "default": false, "label": "Kill các process này", "confirm": "Kill các process đang giữ file?" },
    { "name": "max-files", "type": "int", "default": 5000, "min": 1, "max": 50000, "label": "Số file tối đa (thư mục)" }
  ], "admin": false, "longRunning": false, "dryRun": false, "output": "table" }
```

## 9. Test
- `unsafe.Sizeof(rmProcessInfo{})` đúng theo tài liệu Microsoft (tính tay: 4+8 + 512 + 128 + 4+4+4+4 = 668; kiểm thêm padding — Go căn `Filetime` theo 4 byte vì là 2 `uint32`, khớp C).
- Test Windows: tạo file tạm, mở bằng `os.OpenFile` trong chính test, chạy `Find` → holder có PID = `os.Getpid()`. Đóng file → holder rỗng.
- Thư mục 10 file, `--max-files 3` → `Checked 3`, `Truncated true`.

Chạy thử:
```powershell
# terminal 1
cd D:\tmp\demo; npm run dev
# terminal 2
.\foxkit lock D:\tmp\demo\node_modules
.\foxkit lock D:\tmp\demo\node_modules --kill
```

## 10. Tiêu chí hoàn thành
Tiêu chí chung + test tự-khóa pass + nhận ra VS Code/node đang giữ thư mục dự án thật.

## 11. Ngoài phạm vi
Liệt kê handle cấp kernel (như Sysinternals `handle.exe`, cần driver/quyền debug), mở khóa handle mà không kill process.

## 12. Go học được
Gọi DLL không có trong x/sys, struct khớp layout C có mảng WCHAR, mảng con trỏ chuỗi `[]*uint16`, `runtime.KeepAlive`, mã lỗi trả trực tiếp vs `GetLastError`, mẫu "gọi 2 lần để biết kích thước buffer" rất phổ biến trong Win32.
