# Spec: `foxkit sysinfo`

> Nhóm: **life** · Giai đoạn: GĐ6 · Cần trước: [core/winapi.md](../core/winapi.md)

## 1. Mục đích
Thông tin máy trong một màn hình: Windows bản nào, CPU, RAM, uptime, user, quyền admin. Tiện khi báo lỗi, hỏi hỗ trợ, hoặc kiểm tra máy mới.

## 2. Cú pháp
```
foxkit sysinfo [--json]
```

## 3. File
```
cmd/sysinfo.go
internal/sysinfo/sysinfo_windows.go
internal/sysinfo/sysinfo_other.go
internal/sysinfo/sysinfo_test.go
```

## 4. Hợp đồng
```go
package sysinfo
type Info struct {
    OS           string `json:"os"`           // "Windows 11 Pro"
    DisplayVer   string `json:"displayVersion"` // "24H2"
    Build        string `json:"build"`        // "26100.2314"
    Arch         string `json:"arch"`         // amd64 | arm64
    Hostname     string `json:"hostname"`
    User         string `json:"user"`         // DOMAIN\user
    IsAdmin      bool   `json:"isAdmin"`
    CPU          string `json:"cpu"`
    Cores        int    `json:"cores"`        // logical
    RAMBytes     uint64 `json:"ramBytes"`
    RAMFreeBytes uint64 `json:"ramFreeBytes"`
    RAMUsedPct   uint32 `json:"ramUsedPct"`
    BootTime     string `json:"bootTime"`
    Uptime       string `json:"uptime"`       // "3 ngày 4 giờ 12 phút"
    UptimeSec    int64  `json:"uptimeSec"`
}
func Get() (Info, error)
```

## 5. Cách hoạt động
| Field | Nguồn |
|---|---|
| Build major/minor/build | `windows.RtlGetVersion()` (không bị "nói dối" phiên bản như `GetVersionEx`) |
| OS, DisplayVer, UBR | registry `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion`: `ProductName`, `DisplayVersion`, `UBR` (số bản vá → build `26100.2314`), `EditionID` |
| Windows 11 | `ProductName` trên Win 11 **vẫn ghi "Windows 10 ..."** (lỗi đã biết của Microsoft) → nếu build ≥ 22000 thì thay chữ "Windows 10" bằng "Windows 11" |
| Arch | `runtime.GOARCH` là kiến trúc của binary; kiến trúc máy thật lấy `PROCESSOR_ARCHITECTURE` / `PROCESSOR_ARCHITEW6432` env |
| Hostname | `os.Hostname()` |
| User | `os/user.Current().Username` |
| IsAdmin | `winapi.IsAdmin()` |
| CPU | registry `HKLM\HARDWARE\DESCRIPTION\System\CentralProcessor\0` → `ProcessorNameString` (trim khoảng trắng thừa) |
| Cores | `runtime.NumCPU()` |
| RAM | `GlobalMemoryStatusEx`: dùng bản trong `x/sys/windows` nếu có, không thì gọi qua LazyDLL `kernel32.dll` với struct `MEMORYSTATUSEX` (phải đặt `dwLength = uint32(unsafe.Sizeof(ms))` trước khi gọi, kích thước 64 byte) |
| Uptime | `windows.GetTickCount64()` (ms) → `time.Duration`; BootTime = now − uptime |

Mỗi nguồn lỗi độc lập → field đó rỗng, không làm hỏng cả lệnh (gom cảnh báo vào stderr khi text mode).

## 6. Output
```
Windows     Windows 11 Pro 24H2 (build 26100.2314, amd64)
Máy         DEV-PC · user DEV-PC\me · không phải admin
CPU         AMD Ryzen 7 5800H with Radeon Graphics · 16 luồng
RAM         32.0 GB · trống 12.0 GB · đã dùng 63%
Bật từ      2026-11-13 15:02 (3 ngày 4 giờ 12 phút)
```
JSON: `{ "ok": true, "data": <Info> }`

## 7. Lỗi
Không có lỗi riêng; chỉ lỗi nội bộ `SYSINFO_FAILED` khi mọi nguồn đều lỗi.

## 8. Manifest
```json
{ "name": "sysinfo", "title": "Thông tin máy", "description": "Windows, CPU, RAM, uptime", "category": "system",
  "args": [], "flags": [], "admin": false, "longRunning": false, "dryRun": false, "output": "kv" }
```

## 9. Test
- `unsafe.Sizeof(memoryStatusEx{}) == 64`.
- Hàm định dạng uptime (bảng: 59s → "59 giây", 3h5m → "3 giờ 5 phút", 3d4h12m → "3 ngày 4 giờ 12 phút").
- Hàm sửa tên Windows 11 (bảng: "Windows 10 Pro" + build 26100 → "Windows 11 Pro"; build 19045 giữ nguyên).
- `Get()` trên Windows: `Cores > 0`, `RAMBytes > 0`.

Chạy thử: `.\foxkit sysinfo`, `.\foxkit sysinfo --json`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + hiển thị đúng "Windows 11" trên máy Win 11.

## 11. Ngoài phạm vi
GPU, nhiệt độ, pin, mainboard, serial (cần WMI/COM).

## 12. Go học được
Đọc registry (`registry.OpenKey`, `GetStringValue`, `GetIntegerValue`), struct khớp layout C, `os/user`, kết hợp nhiều nguồn dữ liệu chịu lỗi từng phần.
