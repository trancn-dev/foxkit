# Spec: `foxkit svc`

> Nhóm: **dev** · Giai đoạn: GĐ6 · Cần trước: [core/winapi.md](../core/winapi.md)

## 1. Mục đích
Xem và bật/tắt Windows service hay dùng khi dev: MySQL, PostgreSQL, Redis, Apache, nginx, SQL Server, Docker. Hợp làm button "Bật MySQL"/"Tắt MySQL" trên launcher.

## 2. Cú pháp
```
foxkit svc [filter] [--all] [--json]
foxkit svc start <name> [--json]
foxkit svc stop <name> [--json]
foxkit svc restart <name> [--json]
```
| Tham số/cờ | Ghi chú |
|---|---|
| `[filter]` | chuỗi con của name hoặc display name |
| `--all` | mọi service; mặc định chỉ service khớp danh sách "dev" bên dưới **hoặc** khớp filter |
| `<name>` | tên service (không phân biệt hoa thường) hoặc display name khớp chính xác |

Danh sách "dev" mặc định (khớp chuỗi con, không phân biệt hoa thường): `mysql`, `mariadb`, `postgres`, `redis`, `memurai`, `mongo`, `apache`, `httpd`, `nginx`, `mssql`, `sqlserver`, `docker`, `elasticsearch`, `rabbitmq`, `php`, `laragon`.

## 3. File
```
cmd/svc.go, svc_start.go, svc_stop.go, svc_restart.go
internal/svc/svc_windows.go
internal/svc/svc_other.go
internal/svc/svc_test.go
```

## 4. Hợp đồng
```go
package svc
type Service struct {
    Name      string `json:"name"`
    Display   string `json:"display"`
    State     string `json:"state"`     // running|stopped|starting|stopping|paused|unknown
    StartType string `json:"startType"` // auto|auto-delayed|manual|disabled
    PID       uint32 `json:"pid,omitempty"`
}
func List(filter string, all bool) ([]Service, error)
func Start(name string, wait time.Duration) (Service, error)
func Stop(name string, wait time.Duration) (Service, error)
func Restart(name string, wait time.Duration) (Service, error)
```

## 5. Cách hoạt động
1. **Không dùng `mgr.Connect()`** để liệt kê: hàm này xin quyền `SC_MANAGER_ALL_ACCESS` nên sẽ lỗi khi không phải admin. Thay vào đó:
   ```go
   h, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT|windows.SC_MANAGER_ENUMERATE_SERVICE)
   m := &mgr.Mgr{Handle: h}
   defer m.Disconnect()
   names, err := m.ListServices()
   ```
2. Mỗi service: `windows.OpenService(h, name, windows.SERVICE_QUERY_STATUS|windows.SERVICE_QUERY_CONFIG)` → bọc `&mgr.Service{Name: name, Handle: sh}` → `s.Query()` (state, PID) và `s.Config()` (DisplayName, StartType, DelayedAutoStart). Lọc trước theo tên để không phải mở mọi service khi có filter.
3. **start**: mở với `SERVICE_START|SERVICE_QUERY_STATUS` → `s.Start()` → chờ (poll `Query` mỗi 300ms, tối đa 30s) tới `Running`. Đã chạy → trả về luôn, không lỗi.
4. **stop**: mở với `SERVICE_STOP|SERVICE_QUERY_STATUS` → `s.Control(svc.Stop)` → chờ `Stopped`. Service có service phụ thuộc đang chạy → Windows trả `ERROR_DEPENDENT_SERVICES_RUNNING` → lỗi `SVC_HAS_DEPENDENTS` (v1 không tự dừng service phụ thuộc).
5. **restart**: stop (nếu đang chạy) rồi start.
6. Quyền: start/stop cần admin với hầu hết service → `ERROR_ACCESS_DENIED` → `NEED_ADMIN` exit 3.
7. Service không tồn tại (`ERROR_SERVICE_DOES_NOT_EXIST`) → gợi ý tên gần đúng (tên chứa chuỗi đã nhập).

## 6. Output
```
TRẠNG THÁI  TÊN           HIỂN THỊ                    KHỞI ĐỘNG
running     MySQL80       MySQL80                     auto
stopped     Redis         Redis                       manual
running     com.docker.service  Docker Desktop Service  manual
```
`svc start MySQL80` → `MySQL80 đã chạy (PID 4120).`
JSON: list `{ "items": [Service...] }`; start/stop/restart `{ "service": Service, "changed": true }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `SVC_NOT_FOUND` | | 1 | Có phải ý bạn là: MySQL80? Xem danh sách: foxkit svc --all |
| `SVC_TIMEOUT` | quá 30s chưa đổi trạng thái | 1 | Xem Event Viewer hoặc log của service |
| `SVC_HAS_DEPENDENTS` | | 1 | Dừng các service phụ thuộc trước |
| `SVC_DISABLED` | start service đang disabled | 1 | Đổi Startup type trong services.msc |
| `NEED_ADMIN` | | 3 | Chạy bằng quyền Administrator |

## 8. Manifest
```json
{ "name": "svc", "title": "Services", "description": "Xem service dev: MySQL, Redis, Apache...", "category": "dev",
  "args": [ { "name": "filter", "type": "string", "required": false, "label": "Lọc" } ],
  "flags": [ { "name": "all", "type": "bool", "default": false, "label": "Mọi service" } ],
  "admin": false, "longRunning": false, "dryRun": false, "output": "table" },
{ "name": "svc start", "title": "Bật service", "description": "Start một Windows service", "category": "dev",
  "args": [ { "name": "name", "type": "string", "required": true, "label": "Tên service" } ],
  "flags": [], "admin": true, "longRunning": false, "dryRun": false, "output": "kv" }
```
`svc stop`, `svc restart` giống `svc start` (thêm `confirm` cho stop).

## 9. Test
Logic thuần: map StartType/State sang chữ (bảng), lọc danh sách dev, gợi ý tên gần đúng. Test Windows (đọc): `List("", true)` có service `Dnscache` hoặc `EventLog` (luôn có). Start/stop: test thủ công.

Chạy thử (PowerShell admin cho start/stop):
```powershell
.\foxkit svc
.\foxkit svc sql --json
.\foxkit svc stop MySQL80
.\foxkit svc start MySQL80
```

## 10. Tiêu chí hoàn thành
Tiêu chí chung + liệt kê chạy được khi **không** phải admin.

## 11. Ngoài phạm vi
Đổi startup type, cài/gỡ service, xem log service.

## 12. Go học được
`x/sys/windows/svc/mgr`, mở handle với quyền tối thiểu (nguyên tắc least privilege), vòng chờ trạng thái có timeout, `defer` đóng handle.
