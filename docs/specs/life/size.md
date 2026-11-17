# Spec: `foxkit size`

> Nhóm: **life** · Giai đoạn: GĐ2 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Trả lời "thư mục nào/file nào đang chiếm ổ đĩa?" (giống `du` / WinDirStat bản dòng lệnh).

## 2. Cú pháp
```
foxkit size [dir] [--top 20] [--files] [--json]
```
| Tham số/cờ | Mặc định | Ghi chú |
|---|---|---|
| `[dir]` | thư mục hiện tại | |
| `--top` | 20 | 1–500 |
| `--files` | false | liệt kê **file** lớn nhất trong cả cây thay vì thư mục con trực tiếp |

## 3. File
```
cmd/size.go
internal/size/size.go
internal/size/size_test.go
internal/fsutil/walk.go     # DirSize() dùng chung với temp, prune
```

## 4. Hợp đồng
```go
package fsutil
type DirStat struct { Bytes int64; Files int; Dirs int; Denied int }
func DirSize(ctx context.Context, root string) (DirStat, error)

package size
type Item struct {
    Name  string `json:"name"`
    Path  string `json:"path"`
    Bytes int64  `json:"bytes"`
    Files int    `json:"files"`
    IsDir bool   `json:"isDir"`
    Pct   float64 `json:"pct"` // % so với tổng, 1 chữ số thập phân
}
type Report struct {
    Path   string `json:"path"`
    Total  int64  `json:"total"`
    Items  []Item `json:"items"`
    Denied int    `json:"denied"`
}
func ByChildren(ctx context.Context, dir string, top int) (Report, error)
func LargestFiles(ctx context.Context, dir string, top int) (Report, error)
```

## 5. Cách hoạt động
**ByChildren (mặc định)**
1. `os.ReadDir(dir)` lấy con trực tiếp.
2. File con: size từ `Info().Size()`.
3. Thư mục con: mỗi thư mục một goroutine gọi `DirSize`, giới hạn đồng thời bằng semaphore `make(chan struct{}, runtime.NumCPU()*2)`.
4. `DirSize` dùng `filepath.WalkDir`, cộng size file; lỗi quyền → `Denied++` và bỏ qua. Không theo symlink/junction (WalkDir mặc định không theo; kiểm tra thêm `d.Type()&fs.ModeSymlink` và `ModeIrregular` để bỏ qua).
5. Sắp xếp giảm dần theo Bytes, cắt `top`, gộp phần còn lại thành item `"(còn lại N mục)"`.

**LargestFiles (`--files`)**
1. Duyệt toàn cây (1 goroutine là đủ, I/O là nút thắt).
2. Giữ **top N bằng min-heap** (`container/heap`) thay vì lưu mọi file rồi sort: bộ nhớ O(N) dù cây có triệu file.

Tiến độ in ra stderr như `dup` khi là terminal.

## 6. Output
Text (có thanh tỷ lệ):
```
D:\Projects  — 45.0 GB

  12.1 GB  26.9%  ██████░░░░░░░░░░░░░░  shop-api\
   8.4 GB  18.7%  ████░░░░░░░░░░░░░░░░  old-crm\
   ...
(2 thư mục không đọc được do thiếu quyền)
```
JSON: `{ "ok": true, "data": <Report> }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `DIR_NOT_FOUND` | | 2 | Kiểm tra lại đường dẫn |
| `TOP_INVALID` | ngoài 1–500 | 2 | Ví dụ: --top 50 |

## 8. Manifest
```json
{ "name": "size", "title": "Dung lượng thư mục", "description": "Thư mục/file nào chiếm nhiều dung lượng nhất", "category": "life",
  "args": [ { "name": "dir", "type": "dir", "required": false, "label": "Thư mục" } ],
  "flags": [
    { "name": "top", "type": "int", "default": 20, "min": 1, "max": 500, "label": "Số mục" },
    { "name": "files", "type": "bool", "default": false, "label": "Liệt kê file lớn nhất" }
  ], "admin": false, "longRunning": false, "dryRun": false, "output": "table" }
```

## 9. Test
Cây tạm: `a/` (3 file 100B), `b/c/` (1 file 1000B), `x.bin` 50B → thứ tự `b, a, x.bin`, total 1350. `--files --top 2` → file 1000B rồi 100B. Heap giữ đúng top N với dữ liệu ngẫu nhiên (so với sort toàn bộ).

Chạy thử: `.\foxkit size $HOME`, `.\foxkit size C:\ --top 10`, `.\foxkit size D:\Projects --files --top 30 --json`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + quét `C:\Users\<me>` không crash khi gặp thư mục bị cấm quyền.

## 11. Ngoài phạm vi
Kích thước thực trên đĩa (cluster, nén NTFS), giao diện cây tương tác.

## 12. Go học được
Semaphore bằng buffered channel, `container/heap` (phải tự cài interface `heap.Interface` — ví dụ điển hình về interface Go), `sort.Slice`, `strings.Repeat` vẽ thanh.
