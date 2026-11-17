# Spec: `foxkit prune`

> Nhóm: **dev** · Giai đoạn: GĐ2 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md), [life/size.md](../life/size.md) (dùng `fsutil.DirSize`)

## 1. Mục đích
Tìm và xóa `node_modules`, `vendor` của các dự án lâu không đụng tới để lấy lại dung lượng (giống npkill). Cần thì chạy `npm install` / `composer install` lại. Mặc định chỉ xem trước.

## 2. Cú pháp
```
foxkit prune <dir> [--target node_modules,vendor] [--older 30d] [--depth 4] [--apply] [--json]
```
| Tham số/cờ | Mặc định | Ghi chú |
|---|---|---|
| `<dir>` | bắt buộc | thư mục chứa nhiều dự án, ví dụ `D:\Projects` |
| `--target` | `node_modules,vendor` | chỉ cho phép trong danh sách an toàn: `node_modules`, `vendor`, `.next`, `.nuxt`, `dist`, `.turbo`, `.cache` |
| `--older` | `30d` | dự án "không đụng tới" lâu hơn mốc này; `0` = mọi dự án |
| `--depth` | 4 | độ sâu tối đa tìm dự án tính từ `<dir>` |
| `--apply` | false | |

## 3. File
```
cmd/prune.go
internal/prune/scan.go
internal/prune/prune.go
internal/prune/prune_test.go
```

## 4. Hợp đồng
```go
package prune
type Item struct {
    Path         string    `json:"path"`         // ...\shop\node_modules
    Project      string    `json:"project"`      // ...\shop
    Target       string    `json:"target"`       // node_modules
    Bytes        int64     `json:"bytes"`
    LastActivity time.Time `json:"lastActivity"` // thời gian sửa mới nhất của file dự án (ngoài target)
    Deleted      bool      `json:"deleted"`
    Error        string    `json:"error,omitempty"`
}
type Report struct {
    Applied    bool   `json:"applied"`
    Items      []Item `json:"items"`      // sắp xếp Bytes giảm dần
    TotalBytes int64  `json:"totalBytes"`
    FreedBytes int64  `json:"freedBytes"`
    Skipped    int    `json:"skipped"`    // dự án còn mới
}
```

## 5. Cách hoạt động
1. `WalkDir` từ `<dir>`, giới hạn `--depth`. Gặp thư mục tên nằm trong target **và** thư mục cha có file đánh dấu dự án tương ứng → ghi nhận rồi `fs.SkipDir` (không đi vào trong, đây là chỗ tiết kiệm thời gian lớn nhất):
   | target | file đánh dấu ở thư mục cha |
   |---|---|
   | `node_modules` | `package.json` |
   | `vendor` | `composer.json` (tránh xóa nhầm `vendor` của Go hay dự án khác) |
   | `.next`, `.nuxt`, `dist`, `.turbo`, `.cache` | `package.json` |
   Cũng bỏ qua `.git` và **không bao giờ** đi vào một target lồng trong target khác.
2. `LastActivity` của dự án: thời gian sửa mới nhất trong số các file đánh dấu và file nguồn ở **cấp đầu** của dự án (`package.json`, `composer.json`, lock file, `.env`, `src/`, `app/`, `resources/` — chỉ stat thư mục/file đó, không đệ quy), cộng thêm `HEAD` trong `.git` nếu có (`.git/HEAD`, `.git/index` đổi mỗi lần commit/checkout). **Không** dùng thời gian của chính `node_modules`.
3. Lọc theo `--older`; dự án còn mới → `Skipped++`.
4. Tính size từng item song song (semaphore) bằng `fsutil.DirSize`.
5. `--apply`: `os.RemoveAll`. Windows hay gặp:
   - File read-only (vd trong package có `.git`) → lỗi access denied. Cách xử lý: hàm `removeAllForce` duyệt và `os.Chmod(p, 0666)` từng file rồi thử lại một lần.
   - Đường dẫn dài >260 ký tự: Go tự xử lý với đường dẫn tuyệt đối, không cần làm gì thêm.
   - File đang bị process khác giữ (`npm run dev` đang chạy) → ghi `Error`, gợi ý `foxkit lock <path>` hoặc tắt dev server.
6. Exit 0 nếu không có lỗi xóa; có item lỗi → exit 1 (`PRUNE_PARTIAL`) sau khi in report.

## 6. Output
```
Tìm thấy 14 thư mục trong 9 dự án không đụng tới hơn 30 ngày (tổng 9.0 GB):

  1.8 GB  node_modules  D:\Projects\old-crm          (sửa lần cuối 2026-03-02)
  1.1 GB  vendor        D:\Projects\old-crm          (sửa lần cuối 2026-03-02)
  ...
Bỏ qua 6 dự án còn hoạt động.
Đây là bản xem trước. Thêm --apply để thực hiện.
```
JSON: `{ "ok": true, "data": <Report> }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `DIR_NOT_FOUND` | | 2 | |
| `PRUNE_TARGET_INVALID` | target ngoài danh sách an toàn | 2 | Cho phép: node_modules, vendor, .next, .nuxt, dist, .turbo, .cache |
| `AGE_INVALID` | | 2 | Ví dụ: --older 60d |
| `PRUNE_PARTIAL` | có mục xóa lỗi | 1 | Tắt dev server/IDE đang mở dự án rồi chạy lại |

## 8. Manifest
```json
{ "name": "prune", "title": "Dọn node_modules/vendor", "description": "Xóa node_modules, vendor của dự án lâu không dùng", "category": "dev",
  "args": [ { "name": "dir", "type": "dir", "required": true, "label": "Thư mục dự án" } ],
  "flags": [
    { "name": "target", "type": "list", "default": "node_modules,vendor", "label": "Loại thư mục" },
    { "name": "older", "type": "string", "default": "30d", "label": "Không đụng tới hơn" },
    { "name": "depth", "type": "int", "default": 4, "min": 1, "max": 10, "label": "Độ sâu" },
    { "name": "apply", "type": "bool", "default": false, "label": "Xóa thật", "confirm": "Xóa các thư mục đã liệt kê?" }
  ], "admin": false, "longRunning": false, "dryRun": true, "output": "table" }
```

## 9. Test
Cây tạm:
- `p1/package.json` (cũ 60 ngày) + `p1/node_modules/x/y.js` → ứng viên.
- `p2/package.json` (mới) + `node_modules` → skipped.
- `go-proj/go.mod` + `vendor/` (không có composer.json) → **không** đụng.
- `p3/node_modules/a/node_modules/` → chỉ ghi nhận cấp ngoài.
- `--apply` xóa được file read-only.

Chạy thử:
```powershell
.\foxkit prune D:\Projects
.\foxkit prune D:\Projects --older 90d --json
.\foxkit prune D:\Projects --target node_modules --apply
```

## 10. Tiêu chí hoàn thành
Tiêu chí chung + test + chạy trên thư mục dự án thật: không đi vào trong `node_modules` (thời gian quét < vài giây).

## 11. Ngoài phạm vi
Cache toàn cục (`npm cache`, `~/.composer/cache`), tự `npm install` lại.

## 12. Go học được
`fs.SkipDir`, dùng lại package (`fsutil`), xử lý lỗi Windows đặc thù (`os.Chmod` bỏ read-only), báo lỗi một phần (partial failure) mà vẫn trả dữ liệu.
