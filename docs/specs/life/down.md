# Spec: `foxkit down`

> Nhóm: **life** · Giai đoạn: GĐ7 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md), [dev/hash.md](../dev/hash.md) (`internal/hash`)

## 1. Mục đích
Tải file từ URL về máy với thanh tiến độ, tiếp tục được khi đứt mạng, kiểm tra checksum. Dùng để tải ISO, installer, tài nguyên lớn mà không cần trình duyệt.

## 2. Cú pháp
```
foxkit down <url> [--out <file|dir>] [--sha256 <hex>] [--header "K: V"]... [--resume] [--force] [--retries 3] [--timeout 30s] [--json]
```
| Cờ | Mặc định | Ghi chú |
|---|---|---|
| `--out` | thư mục hiện tại | thư mục → tên file lấy theo `Content-Disposition`, không có thì phần cuối của path URL, không có nữa thì `download` |
| `--sha256` | | kiểm tra sau khi tải; sai thì xóa file và báo lỗi |
| `--resume` | true | tiếp tục từ file `.part` nếu server hỗ trợ `Range` |
| `--force` | false | ghi đè file đã tồn tại |
| `--retries` | 3 | thử lại khi lỗi mạng (đợi 1s, 2s, 4s) |
| `--timeout` | 30s | timeout **kết nối và chờ header**, không phải toàn bộ lần tải |

`longRunning: true` ở chế độ `--json` (NDJSON sự kiện tiến độ).

## 3. File
```
cmd/down.go
internal/down/down.go
internal/down/progress.go
internal/down/down_test.go
```

## 4. Hợp đồng
```go
package down
type Options struct { URL, Out string; Headers http.Header; Resume, Force bool; Retries int; Timeout time.Duration; SHA256 string }
type Event struct { Event string `json:"event"` /* start|progress|retry|done */; Bytes, Total int64; BytesPerSec int64; Path string; Attempt int }
type Result struct { Path string `json:"path"`; Bytes int64 `json:"bytes"`; DurationMs int64 `json:"durationMs"`; SHA256 string `json:"sha256"`; Resumed bool `json:"resumed"` }
func Download(ctx context.Context, opt Options, emit func(Event)) (Result, error)
```

## 5. Cách hoạt động
1. Chỉ cho `http`/`https`. Header `User-Agent: foxkit/<version>`.
2. Tên file: lấy từ `Content-Disposition` (`mime.ParseMediaType`), làm sạch bằng `filepath.Base` và **loại ký tự không hợp lệ trên Windows** (`<>:"/\|?*`) để server độc hại không ghi ra ngoài thư mục đích (path traversal).
3. Tải vào `<tên>.part`; xong thì `os.Rename` sang tên cuối. File đích đã tồn tại và không `--force` → `FILE_EXISTS` (trước khi tải).
4. **Resume**: có `.part` kích thước N → gửi `Range: bytes=N-`. Trả `206` → ghi nối tiếp (kiểm `Content-Range` bắt đầu đúng N). Trả `200` (server bỏ qua Range) → tải lại từ đầu. Trả `416` → file `.part` đã đủ hoặc lỗi → xóa và tải lại.
5. Gửi `If-Range` với ETag/Last-Modified đã lưu trong file phụ `<tên>.part.meta` (JSON nhỏ) để không nối nhầm khi file trên server đã đổi.
6. Sao chép bằng `io.Copy` qua một `io.Reader` bọc để đếm byte và phát `progress` mỗi 250ms (tốc độ trung bình trượt 3 giây). Tính sha256 song song bằng `io.TeeReader` (khi resume phải băm lại phần `.part` đã có trước).
7. Lỗi mạng giữa chừng → `retry` (tiếp tục bằng Range); lỗi 4xx (trừ 408/429) không thử lại; 5xx có thử lại.
8. `ctx` hủy bằng Ctrl+C: giữ nguyên `.part` để lần sau tiếp tục, thoát với `CANCELED`.
9. Theo redirect tối đa 10 lần; **không** gửi header `Authorization` sang host khác khi redirect (Go đã làm mặc định, giữ nguyên).
10. Thanh tiến độ vẽ trên stderr khi là terminal: `[███████░░░░░] 58%  1.2 GB / 2.1 GB  14.2 MB/s  còn 1m02s`. Không biết `Total` (không có Content-Length) → chỉ hiện số byte và tốc độ.

## 6. Output
Text cuối: `Đã tải ubuntu.iso (2.1 GB) trong 2m31s → D:\Downloads\ubuntu.iso  sha256 ab12…`
JSON: `{ "ok": true, "data": <Result> }`; với NDJSON: mỗi `Event` một dòng, dòng cuối `{"event":"done",...}`.

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `URL_INVALID` | | 2 | Ví dụ: foxkit down https://example.com/file.zip |
| `FILE_EXISTS` | | 1 | Thêm --force để ghi đè |
| `HTTP_STATUS` | 4xx/5xx | 1 | Server trả <mã>; kiểm tra lại URL |
| `DOWN_CHECKSUM` | sha256 sai | 1 | File có thể tải lỗi hoặc bị thay đổi; đã xóa file, hãy tải lại |
| `HTTP_TIMEOUT`, `HTTP_CONNECT_FAILED` | | 1 | Kiểm tra mạng, thử lại với --retries |
| `DISK_FULL` | hết chỗ | 1 | Giải phóng dung lượng: foxkit size, foxkit temp |
| `CANCELED` | | 1 | File dở dang giữ lại, chạy lại cùng lệnh để tiếp tục |

## 8. Manifest
```json
{ "name": "down", "title": "Tải file", "description": "Tải file từ URL, tiếp tục khi đứt, kiểm tra checksum", "category": "life",
  "args": [ { "name": "url", "type": "string", "required": true, "label": "URL" } ],
  "flags": [
    { "name": "out", "type": "dir", "label": "Lưu vào" },
    { "name": "sha256", "type": "string", "label": "SHA-256 mong đợi" },
    { "name": "resume", "type": "bool", "default": true, "label": "Tiếp tục nếu dở dang" },
    { "name": "force", "type": "bool", "default": false, "label": "Ghi đè" },
    { "name": "retries", "type": "int", "default": 3, "min": 0, "max": 10, "label": "Số lần thử lại" }
  ], "admin": false, "longRunning": true, "dryRun": false, "output": "stream" }
```

## 9. Test
`httptest.NewServer` với handler tự viết:
- Tải đủ, đúng sha256.
- Server hỗ trợ Range: ngắt kết nối giữa chừng (handler đóng conn sau N byte) → retry và nối tiếp đúng, file cuối giống nguồn.
- Server bỏ qua Range (trả 200) → tải lại từ đầu, không hỏng file.
- `Content-Disposition: attachment; filename="..\..\evil.exe"` → tên an toàn.
- sha256 sai → file bị xóa, lỗi đúng code.
- 404 → không retry; 503 → retry rồi báo lỗi.

Chạy thử: `.\foxkit down https://example.com/big.zip --out D:\Downloads`, ngắt Ctrl+C rồi chạy lại để thấy resume.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + test resume + tải thử file lớn trên mạng thật.

## 11. Ngoài phạm vi
Tải song song nhiều luồng, torrent, FTP, đăng nhập/cookie phức tạp, tải hàng loạt từ danh sách.

## 12. Go học được
`io.Reader` bọc để đếm tiến độ, `io.TeeReader`, HTTP Range/resume, `mime.ParseMediaType`, an toàn đường dẫn khi nhận tên file từ bên ngoài, thử lại với backoff.
