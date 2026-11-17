# Spec: `foxkit disk`

> Nhóm: **life** · Giai đoạn: GĐ6 · Cần trước: [core/winapi.md](../core/winapi.md)

## 1. Mục đích
Xem nhanh các ổ đĩa: loại, nhãn, dung lượng, còn trống, cảnh báo ổ sắp đầy.

## 2. Cú pháp
```
foxkit disk [--all] [--warn 90] [--json]
```
| Cờ | Mặc định | Ghi chú |
|---|---|---|
| `--all` | false | gồm ổ CD/ổ không có đĩa/ổ mạng mất kết nối |
| `--warn` | 90 | % đã dùng để đánh dấu cảnh báo |

## 3. File
```
cmd/disk.go
internal/disk/disk_windows.go
internal/disk/disk_other.go
internal/disk/disk_test.go
```

## 4. Hợp đồng
```go
package disk
type Drive struct {
    Root       string  `json:"root"`        // "C:\\"
    Type       string  `json:"type"`        // fixed|removable|network|cdrom|ram|unknown
    Label      string  `json:"label"`
    FS         string  `json:"fs"`          // NTFS, exFAT
    TotalBytes uint64  `json:"totalBytes"`
    FreeBytes  uint64  `json:"freeBytes"`   // dung lượng user hiện tại được dùng (tính quota)
    UsedPct    float64 `json:"usedPct"`
    Warn       bool    `json:"warn"`
    Error      string  `json:"error,omitempty"`
}
func List(all bool, warnPct float64) ([]Drive, error)
```

## 5. Cách hoạt động
1. `windows.GetLogicalDriveStrings` → buffer `[]uint16` dài 254, tách theo ký tự `0` → `["C:\\","D:\\",...]`.
2. `windows.GetDriveType(root)` → map `DRIVE_FIXED`(3)→fixed, `DRIVE_REMOVABLE`(2), `DRIVE_REMOTE`(4)→network, `DRIVE_CDROM`(5), `DRIVE_RAMDISK`(6).
3. `windows.GetVolumeInformation` → label, tên file system. Ổ CD trống / ổ mạng mất kết nối trả lỗi → ghi `Error` ("không có đĩa"/"không truy cập được"), ẩn trừ khi `--all`.
4. `windows.GetDiskFreeSpaceEx(root, &freeAvail, &total, &totalFree)` → dùng `freeAvail` làm `FreeBytes`.
5. Gọi API trên ổ mạng có thể treo vài giây → mỗi ổ chạy trong goroutine với timeout 3s; quá hạn → `Error: "timeout"`.
6. Sắp theo Root.

## 6. Output
```
Ổ    LOẠI     NHÃN     FS     DUNG LƯỢNG   TRỐNG     ĐÃ DÙNG
C:\  fixed    Windows  NTFS   476 GB       38 GB     92%  ██████████████████░░  ⚠
D:\  fixed    Data     NTFS   931 GB       512 GB    45%  █████████░░░░░░░░░░░
Ổ C:\ sắp đầy. Thử: foxkit temp --apply, foxkit prune <thư mục dự án>, foxkit size C:\Users
```
JSON: `{ "ok": true, "data": { "items": [Drive...] } }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `WARN_INVALID` | ngoài 1–100 | 2 | |
| `DISK_LIST_FAILED` | API lỗi | 1 | |

## 8. Manifest
```json
{ "name": "disk", "title": "Ổ đĩa", "description": "Dung lượng các ổ, cảnh báo ổ sắp đầy", "category": "system",
  "args": [],
  "flags": [
    { "name": "all", "type": "bool", "default": false, "label": "Gồm ổ không sẵn sàng" },
    { "name": "warn", "type": "int", "default": 90, "min": 1, "max": 100, "label": "Ngưỡng cảnh báo %" }
  ], "admin": false, "longRunning": false, "dryRun": false, "output": "table" }
```

## 9. Test
Hàm tách chuỗi multi-sz (`"C:\\\x00D:\\\x00\x00"`) dạng bảng; tính `UsedPct`/`Warn`. Gọi API thật: test chỉ chạy trên Windows (`disk_windows_test.go`), kiểm `C:\` có trong danh sách và `TotalBytes > 0`.

Chạy thử: `.\foxkit disk`, `.\foxkit disk --all --json`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + ổ mạng mất kết nối không làm treo lệnh.

## 11. Ngoài phạm vi
SMART/sức khỏe ổ, phân vùng, BitLocker.

## 12. Go học được
Gọi API x/sys trực tiếp, buffer `[]uint16`, timeout cho lời gọi đồng bộ bằng goroutine + `select` với `time.After`.
