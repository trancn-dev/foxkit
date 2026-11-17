# Spec: `foxkit temp`

> Nhóm: **life** · Giai đoạn: GĐ2 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Xem thư mục tạm của user chiếm bao nhiêu và dọn file cũ. Mặc định chỉ xem trước.

## 2. Cú pháp
```
foxkit temp [--older 7d] [--apply] [--json]
```
| Cờ | Mặc định | Ghi chú |
|---|---|---|
| `--older` | `7d` | chỉ xóa mục có thời gian sửa cũ hơn mốc này; tối thiểu `1h` |
| `--apply` | false | thực hiện xóa |

Phạm vi v1: chỉ `os.TempDir()` của user hiện tại (thường `%LOCALAPPDATA%\Temp`). **Không** đụng `C:\Windows\Temp` (cần admin, rủi ro hơn).

## 3. File
```
cmd/temp.go
internal/temp/temp.go
internal/temp/temp_test.go
```

## 4. Hợp đồng
```go
package temp
type Report struct {
    Dir          string `json:"dir"`
    Applied      bool   `json:"applied"`
    TotalBytes   int64  `json:"totalBytes"`   // toàn bộ thư mục temp
    Candidates   int    `json:"candidates"`   // số mục đủ điều kiện xóa
    CandidateBytes int64 `json:"candidateBytes"`
    Deleted      int    `json:"deleted"`
    FreedBytes   int64  `json:"freedBytes"`
    Locked       int    `json:"locked"`       // đang bị dùng, bỏ qua
}
func Run(ctx context.Context, dir string, olderThan time.Duration, apply bool, now time.Time) (Report, error)
```

## 5. Cách hoạt động
1. `dir := os.TempDir()`; kiểm tra tồn tại.
2. Duyệt **con trực tiếp** của temp:
   - File: đủ cũ → ứng viên.
   - Thư mục: dùng `fsutil.DirSize` và thời gian sửa **mới nhất** của mọi file bên trong (thư mục cũ nhưng có file mới thì giữ). Mới nhất vẫn cũ hơn mốc → ứng viên cả thư mục.
3. `--apply`: với từng ứng viên gọi `os.RemoveAll`. Lỗi (file đang mở bởi chương trình khác, thiếu quyền) → `Locked++`, đi tiếp, **không** coi là lỗi lệnh.
   - Thư mục xóa được một phần: cộng phần đã giải phóng (đo lại size trước/sau).
4. Không theo symlink/junction.
5. Exit 0 kể cả có `Locked > 0`.

## 6. Output
Dry-run:
```
Thư mục tạm: C:\Users\me\AppData\Local\Temp  (3.4 GB)
Có thể dọn 1.532 mục cũ hơn 7 ngày: 2.9 GB
Đây là bản xem trước. Thêm --apply để thực hiện.
```
Apply:
```
Đã xóa 1.491 mục, giải phóng 2.7 GB. Bỏ qua 41 mục đang được sử dụng.
```
JSON: `{ "ok": true, "data": <Report> }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `AGE_INVALID` | `--older` sai hoặc < 1h | 2 | Ví dụ: --older 7d |
| `TEMP_NOT_FOUND` | thư mục temp không tồn tại | 1 | Kiểm tra biến môi trường TEMP |

## 8. Manifest
```json
{ "name": "temp", "title": "Dọn thư mục tạm", "description": "Xem và dọn file tạm cũ của user", "category": "life",
  "args": [],
  "flags": [
    { "name": "older", "type": "string", "default": "7d", "label": "Cũ hơn" },
    { "name": "apply", "type": "bool", "default": false, "label": "Xóa thật", "confirm": "Xóa các file tạm cũ?" }
  ], "admin": false, "longRunning": false, "dryRun": true, "output": "kv" }
```

## 9. Test
Hàm `Run` nhận `dir` nên test bằng `t.TempDir()`:
- File cũ 10 ngày (dùng `os.Chtimes`) → bị xóa; file 1 ngày → giữ.
- Thư mục cũ chứa 1 file mới → giữ cả thư mục.
- File đang mở (giữ handle `os.Open` trong test, Windows không cho xóa) → `Locked == 1`, lệnh vẫn thành công. (Test này chỉ có nghĩa trên Windows: dùng `runtime.GOOS` để skip nơi khác.)

Chạy thử: `.\foxkit temp`, `.\foxkit temp --older 30d --json`, `.\foxkit temp --apply`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + chạy `--apply` trên máy thật không báo lỗi khi có file đang bị khóa.

## 11. Ngoài phạm vi
`C:\Windows\Temp`, cache trình duyệt, Windows Update cache, Thùng rác.

## 12. Go học được
`os.TempDir`, `os.Chtimes` trong test, phân biệt "lỗi của lệnh" và "lỗi từng mục" (đếm chứ không dừng), `t.Skip` theo hệ điều hành.
