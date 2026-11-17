# Spec: `foxkit zip`

> Nhóm: **life** · Giai đoạn: GĐ7 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md), [life/size.md](size.md) (`fsutil`)

## 1. Mục đích
Nén thư mục/file thành `.zip` và giải nén, không cần cài 7-Zip/WinRAR. Có lọc bỏ `node_modules`, `.git`, `vendor` khi nén dự án để gửi đi. Chỉ định dạng ZIP (thư viện chuẩn).

## 2. Cú pháp
```
foxkit zip pack <src>... [--out <file.zip>] [--exclude pat,...] [--no-default-excludes] [--level 0-9] [--force] [--json]
foxkit zip unpack <file.zip> [--to <dir>] [--force] [--json]
foxkit zip list <file.zip> [--json]
```
| Cờ | Ghi chú |
|---|---|
| `--out` | mặc định `<tên thư mục>.zip` cạnh thư mục nguồn (nhiều nguồn thì bắt buộc chỉ định) |
| `--exclude` | glob tên hoặc đường dẫn tương đối (`*.log`, `storage/framework/cache`); thêm vào danh sách mặc định |
| `--no-default-excludes` | không tự bỏ `.git`, `node_modules`, `vendor`, `.idea`, `.vscode`, `*.log`, `.env` |
| `--level` | 0 = chỉ lưu; mặc định 6 |
| `--to` | thư mục giải nén, mặc định `<tên zip không đuôi>\` cạnh file zip |
| `--force` | cho ghi đè |

**`.env` mặc định bị loại** khi nén dự án (tránh gửi nhầm bí mật); thông báo rõ trong output những gì đã loại.

## 3. File
```
cmd/zip.go           # cha
cmd/zip_pack.go, zip_unpack.go, zip_list.go
internal/zipx/pack.go
internal/zipx/unpack.go
internal/zipx/zipx_test.go
```

## 4. Hợp đồng
```go
package zipx
type PackResult struct { Out string; Files int; Bytes, ZipBytes int64; Excluded []string; Skipped []string }
type UnpackResult struct { To string; Files int; Bytes int64 }
type Entry struct { Name string `json:"name"`; Bytes int64 `json:"bytes"`; Compressed int64 `json:"compressed"`; Modified time.Time `json:"modified"`; IsDir bool `json:"isDir"` }
func Pack(ctx context.Context, srcs []string, out string, opt PackOptions) (PackResult, error)
func Unpack(ctx context.Context, zipPath, to string, force bool) (UnpackResult, error)
func List(zipPath string) ([]Entry, error)
```

## 5. Cách hoạt động
**Pack**
1. Ghi vào `<out>.tmp` rồi `os.Rename` khi xong (lỗi giữa chừng không để lại zip hỏng). `out` không được nằm **trong** thư mục nguồn (nếu có thì tự thêm vào exclude để không nén chính nó).
2. `filepath.WalkDir` từng nguồn; đường dẫn trong zip dùng `/`, tương đối so với **cha của nguồn** (nén `D:\shop` → các mục bắt đầu bằng `shop/...`).
3. `zip.Writer.CreateHeader` với `Method: zip.Deflate` (`zip.Store` nếu `--level 0` hoặc file đã nén như `.zip .png .jpg .mp4`); đặt `Modified` theo file; tên là UTF-8 (đặt cờ `NonUTF8 = false`, Go tự đặt bit 11) để Windows Explorer đọc đúng tên tiếng Việt.
4. Level: dùng `zw.RegisterCompressor(zip.Deflate, func(w io.Writer) (io.WriteCloser, error) { return flate.NewWriter(w, level) })`.
5. Symlink/junction: bỏ qua (ghi vào `Skipped`). File không đọc được (đang bị khóa) → `Skipped`, tiếp tục.
6. Thư mục rỗng: ghi mục kết thúc bằng `/`.

**Unpack — chống Zip Slip (bắt buộc)**
1. Với mỗi entry: `dest := filepath.Join(to, filepath.FromSlash(e.Name))`; kiểm `filepath.Rel(to, dest)` **không bắt đầu bằng `..`** và `dest` nằm trong `to`; ngược lại → `ZIP_UNSAFE_PATH` và dừng toàn bộ (không giải nén dở).
2. Chặn tên có ký tự không hợp lệ trên Windows (`:` ngoài ổ đĩa, tên dành riêng `CON`, `NUL`...), đường dẫn tuyệt đối, và entry là symlink.
3. Giới hạn chống zip bomb: tổng dung lượng giải nén ≤ 20× kích thước zip hoặc ≤ 20GB (cái nhỏ hơn); vượt → `ZIP_TOO_LARGE`.
4. File đã tồn tại và không `--force` → liệt kê rồi dừng **trước khi ghi bất kỳ file nào** (`FILE_EXISTS`).
5. Ghi file qua `io.Copy` có đếm; đặt lại `Modified` bằng `os.Chtimes`.
6. Tên entry không phải UTF-8 (zip cũ từ Windows dùng CP437/CP1258): hiển thị như đọc được, không cố chuyển mã; gợi ý dùng 7-Zip nếu tên bị lỗi.

## 6. Output
```
> foxkit zip pack D:\Projects\shop
Nén 1.204 file (86 MB) → shop.zip (21 MB, nén 76%)
Đã loại: .git, node_modules, vendor, .env, 3 file *.log
```
JSON pack: `{ "ok": true, "data": { "out": "...", "files": 1204, "bytes": 90177536, "zipBytes": 22020096, "excluded": [".git","node_modules","vendor",".env"], "skipped": [] } }`
`list`: bảng tên, size gốc, size nén, ngày.

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `ZIP_NO_SOURCE` | | 2 | Ví dụ: foxkit zip pack D:\Projects\shop |
| `ZIP_NOT_FOUND` | | 2 | |
| `ZIP_INVALID` | file hỏng / không phải zip | 1 | File có thể bị tải lỗi; kiểm tra bằng foxkit hash |
| `ZIP_UNSAFE_PATH` | Zip Slip | 1 | File zip chứa đường dẫn nguy hiểm, không giải nén |
| `ZIP_TOO_LARGE` | nghi zip bomb | 1 | |
| `FILE_EXISTS` | | 1 | Thêm --force để ghi đè |

## 8. Manifest
```json
{ "name": "zip pack", "title": "Nén ZIP", "description": "Nén thư mục/file, tự bỏ node_modules, .git, .env", "category": "life",
  "args": [ { "name": "src", "type": "file", "allowDir": true, "required": true, "multiple": true, "label": "Nguồn" } ],
  "flags": [
    { "name": "out", "type": "string", "label": "File zip đầu ra" },
    { "name": "exclude", "type": "list", "label": "Loại thêm" },
    { "name": "no-default-excludes", "type": "bool", "default": false, "label": "Không tự loại .git/node_modules/.env" },
    { "name": "level", "type": "int", "default": 6, "min": 0, "max": 9, "label": "Mức nén" },
    { "name": "force", "type": "bool", "default": false, "label": "Ghi đè" }
  ], "admin": false, "longRunning": false, "dryRun": false, "output": "kv" },
{ "name": "zip unpack", "title": "Giải nén ZIP", "description": "Giải nén file zip an toàn", "category": "life",
  "args": [ { "name": "file", "type": "file", "required": true, "label": "File zip" } ],
  "flags": [
    { "name": "to", "type": "dir", "label": "Giải nén vào" },
    { "name": "force", "type": "bool", "default": false, "label": "Ghi đè" }
  ], "admin": false, "longRunning": false, "dryRun": false, "output": "kv" },
{ "name": "zip list", "title": "Xem nội dung ZIP", "description": "Liệt kê file trong zip", "category": "life",
  "args": [ { "name": "file", "type": "file", "required": true, "label": "File zip" } ],
  "flags": [], "admin": false, "longRunning": false, "dryRun": false, "output": "table" }
```

## 9. Test
- Round-trip: pack rồi unpack ra thư mục khác → cây file giống (so hash từng file, kể cả tên tiếng Việt `Đơn hàng.txt` và thư mục rỗng).
- Exclude mặc định hoạt động, `--no-default-excludes` tắt được.
- **Zip Slip**: tạo zip có entry `../../evil.txt` và `C:\evil.txt` → `ZIP_UNSAFE_PATH`, không file nào được ghi.
- Zip bomb giả (tỉ lệ nén cực cao) → `ZIP_TOO_LARGE`.
- Lỗi giữa chừng khi pack (nguồn bị xóa) → không để lại `.tmp`/zip dở.
- Tên đường dẫn trong zip luôn dùng `/`.

Chạy thử: `.\foxkit zip pack D:\Projects\shop`, `.\foxkit zip list shop.zip`, `.\foxkit zip unpack shop.zip --to D:\tmp\shop`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + test Zip Slip pass + Windows Explorer mở được zip do foxkit tạo, tên tiếng Việt đúng.

## 11. Ngoài phạm vi
Mã hóa mật khẩu, 7z/rar/tar.gz, zip nhiều phần, thêm file vào zip có sẵn.

## 12. Go học được
`archive/zip`, `compress/flate`, `filepath.Rel` để chống path traversal (lỗ hổng Zip Slip), ghi file tạm rồi đổi tên, giới hạn tài nguyên khi xử lý dữ liệu không tin cậy.
