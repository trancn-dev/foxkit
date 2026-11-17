# Spec: `foxkit dup`

> Nhóm: **life** · Giai đoạn: GĐ2 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Tìm file trùng nội dung trong một hoặc nhiều thư mục (ảnh, video, tài liệu bị copy nhiều lần) và báo dung lượng lãng phí. **v1 chỉ báo cáo, không xóa.**

## 2. Cú pháp
```
foxkit dup <dir>... [--min-size 1KB] [--ext jpg,png,mp4] [--workers N] [--include-hidden] [--json]
```
| Tham số/cờ | Kiểu | Mặc định | Ghi chú |
|---|---|---|---|
| `<dir>...` | dir (nhiều) | bắt buộc | quét chung, so trùng giữa các thư mục |
| `--min-size` | size | `1KB` | bỏ file nhỏ (dùng `fsutil.ParseSize`) |
| `--ext` | list | tất cả | không phân biệt hoa thường, không cần dấu chấm |
| `--workers` | int | `runtime.NumCPU()` | 1–64 |
| `--include-hidden` | bool | false | mặc định bỏ file/thư mục ẩn và `fsutil.DefaultSkipDirs` |

## 3. File
```
cmd/dup.go
internal/dup/scan.go      # duyệt thư mục, gom theo size
internal/dup/hash.go      # worker pool băm
internal/dup/dup.go       # Find(): ghép các bước
internal/dup/dup_test.go
```

## 4. Hợp đồng
```go
package dup
type Options struct { Dirs []string; MinSize int64; Exts []string; Workers int; IncludeHidden bool }
type Group struct {
    Size  int64    `json:"size"`
    Hash  string   `json:"hash"`   // sha256 toàn file
    Files []string `json:"files"`  // sắp xếp theo đường dẫn
}
type Report struct {
    Scanned     int     `json:"scanned"`
    Groups      []Group `json:"groups"`       // sắp xếp theo WastedBytes giảm dần
    WastedBytes int64   `json:"wastedBytes"`  // tổng size*(len(files)-1)
    Skipped     int     `json:"skipped"`      // file không đọc được
    DurationMs  int64   `json:"durationMs"`
}
func Find(ctx context.Context, opt Options) (Report, error)
```

## 5. Cách hoạt động (3 tầng lọc)
1. **Duyệt** bằng `filepath.WalkDir` cho từng dir. Bỏ: thư mục trong `DefaultSkipDirs`, file ẩn (tên bắt đầu `.` hoặc thuộc tính Hidden: kiểm tra qua `d.Info()` → `Sys().(*syscall.Win32FileAttributeData).FileAttributes & FILE_ATTRIBUTE_HIDDEN`, file `_windows.go`), symlink, file < min-size, sai ext. Lỗi quyền → `Skipped++` và đi tiếp (trả `nil` từ callback, với thư mục thì `fs.SkipDir`).
   - Cùng một file được liệt kê 2 lần (2 dir lồng nhau) → khử trùng bằng đường dẫn tuyệt đối đã `filepath.Clean` + `strings.ToLower`.
2. **Gom theo size**: `map[int64][]string`. Bỏ nhóm chỉ 1 file (khác size chắc chắn khác nội dung).
3. **Băm 4KB đầu** các file còn lại → gom lại theo `(size, partialHash)`, bỏ nhóm 1 file.
4. **Băm toàn file** (sha256) → gom theo `(size, fullHash)`, nhóm ≥2 file là trùng.
5. Bước 3 và 4 chạy bằng **worker pool**:
   ```
   jobs := make(chan string)        // đường dẫn
   results := make(chan hashResult) // path, hash, err
   N goroutine: for p := range jobs { results <- hashFile(p) }
   1 goroutine đẩy job rồi close(jobs)
   WaitGroup chờ worker xong rồi close(results)
   main: for r := range results { gom vào map }
   ```
   Chỉ goroutine main ghi vào map nên không cần mutex.
6. Hỗ trợ hủy: kiểm tra `ctx.Done()` trong worker; `cmd` dùng `signal.NotifyContext(ctx, os.Interrupt)` để Ctrl+C dừng gọn.
7. Text mode khi stderr là terminal: in tiến độ `Đã quét 12.345 file...` ra **stderr** (ghi đè dòng bằng `\r`), mỗi 500ms.

## 6. Output
Text:
```
Quét 18.234 file trong 4.2s. Tìm thấy 37 nhóm trùng, lãng phí 1.8 GB.

[2.4 MB × 3]
  D:\Photos\2025\IMG_0012.jpg
  D:\Photos\Backup\IMG_0012.jpg
  D:\Photos\Backup\IMG_0012 (1).jpg
...
```
JSON: `{ "ok": true, "data": <Report> }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `DUP_NO_DIR` | thiếu dir | 2 | Ví dụ: foxkit dup D:\Photos |
| `DIR_NOT_FOUND` | dir không tồn tại / không phải thư mục | 2 | Kiểm tra lại đường dẫn |
| `SIZE_INVALID` | `--min-size` sai | 2 | Ví dụ: 500KB, 10MB |
| `CANCELED` | Ctrl+C | 1 | |

## 8. Manifest
```json
{ "name": "dup", "title": "Tìm file trùng", "description": "Tìm file trùng nội dung, báo dung lượng lãng phí", "category": "life",
  "args": [ { "name": "dir", "type": "dir", "required": true, "multiple": true, "label": "Thư mục" } ],
  "flags": [
    { "name": "min-size", "type": "string", "default": "1KB", "label": "Bỏ file nhỏ hơn" },
    { "name": "ext", "type": "list", "label": "Chỉ đuôi file (jpg,png)" },
    { "name": "workers", "type": "int", "min": 1, "max": 64, "label": "Số luồng" },
    { "name": "include-hidden", "type": "bool", "default": false, "label": "Gồm file ẩn" }
  ], "admin": false, "longRunning": false, "dryRun": false, "output": "table" }
```

## 9. Test
Dựng cây file trong `t.TempDir()`:
- `a.txt` = `b.txt` = "hello" (trùng), `c.txt` = "hellp" (cùng size, khác nội dung) → 1 nhóm {a,b}.
- 2 file cùng 4KB đầu nhưng khác đuôi → không trùng (kiểm chứng tầng 4).
- File trong `node_modules/` bị bỏ qua.
- `--ext txt` lọc đúng; `--min-size 10` bỏ file 5 byte.
- Workers 1 và 8 cho kết quả giống hệt (chạy với `go test -race`).

Chạy thử: `.\foxkit dup $HOME\Downloads`, `.\foxkit dup D:\Photos E:\Backup --ext jpg,png --json`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + `go test -race ./internal/dup` sạch + quét ~50.000 file không lỗi, Ctrl+C dừng ngay.

## 11. Ngoài phạm vi
Xóa/di chuyển file trùng, ảnh gần giống (perceptual hash), so trùng theo tên.

## 12. Go học được
Goroutine, channel không buffer, `sync.WaitGroup`, mẫu worker pool + fan-in, `context` để hủy, `go test -race`. So với JS: `Promise.all` chạy đồng thời trên 1 luồng; goroutine chạy song song thật trên nhiều core nên phải lo chuyện chia sẻ dữ liệu.
