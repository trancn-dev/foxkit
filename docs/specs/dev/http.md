# Spec: `foxkit http`

> Nhóm: **dev** · Giai đoạn: GĐ4 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Kiểm tra nhanh một URL: status, thời gian từng pha (DNS, kết nối, TLS, chờ phản hồi), chuỗi redirect, header, hạn chứng chỉ SSL. Dùng khi debug site chậm/chết, kiểm tra SSL sắp hết hạn.

## 2. Cú pháp
```
foxkit http <url> [--method GET] [--header "K: V"]... [--data <body>] [--timeout 10s]
                  [--no-follow] [--insecure] [--body] [--json]
```
| Cờ | Mặc định | Ghi chú |
|---|---|---|
| `<url>` | | thiếu scheme → thêm `https://` |
| `--method` | GET | GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS |
| `--header` | | lặp nhiều lần |
| `--data` | | body; bắt đầu `@` → đọc từ file (`@payload.json`); tự đặt `Content-Type: application/json` nếu body là JSON hợp lệ và chưa có header |
| `--timeout` | 10s | `time.ParseDuration` |
| `--no-follow` | false | không theo redirect |
| `--insecure` | false | bỏ kiểm tra chứng chỉ (dev local với self-signed) |
| `--body` | false | in body (tối đa 64KB, cắt bớt kèm thông báo) |

## 3. File
```
cmd/http.go
internal/httpcheck/check.go
internal/httpcheck/trace.go
internal/httpcheck/check_test.go
```

## 4. Hợp đồng
```go
package httpcheck
type Timing struct {
    DNSMs, ConnectMs, TLSMs, TTFBMs, TotalMs int64 // json tag camelCase: dnsMs...
}
type TLSInfo struct {
    Subject  string    `json:"subject"`
    Issuer   string    `json:"issuer"`
    Expires  time.Time `json:"expires"`
    DaysLeft int       `json:"daysLeft"`
    Version  string    `json:"version"` // "TLS 1.3"
}
type Hop struct { URL string `json:"url"`; Status int `json:"status"` }
type Result struct {
    URL       string            `json:"url"`
    FinalURL  string            `json:"finalUrl"`
    Status    int               `json:"status"`
    Proto     string            `json:"proto"`   // HTTP/2.0
    Timing    Timing            `json:"timing"`
    Redirects []Hop             `json:"redirects"`
    Headers   map[string]string `json:"headers"` // key lower-case, nhiều giá trị nối bằng ", "
    Bytes     int64             `json:"bytes"`
    TLS       *TLSInfo          `json:"tls,omitempty"`
    Body      string            `json:"body,omitempty"`
}
func Check(ctx context.Context, opt Options) (Result, error)
```

## 5. Cách hoạt động
1. Tạo `http.Client` riêng (không dùng `DefaultClient`): `Timeout` = `--timeout`; `CheckRedirect` ghi lại từng hop (status lấy từ `req.Response.StatusCode`), dừng sau 10 hop; `--no-follow` → `return http.ErrUseLastResponse`.
2. `Transport` clone từ `http.DefaultTransport` (giữ proxy từ biến môi trường), `--insecure` → `TLSClientConfig.InsecureSkipVerify = true`.
3. Đo thời gian bằng `httptrace.ClientTrace` gắn vào context của request **cuối cùng**: `DNSStart/DNSDone`, `ConnectStart/ConnectDone`, `TLSHandshakeStart/Done`, `GotFirstResponseByte`. Kết nối dùng lại (`GotConn.Reused`) → các pha bằng 0.
4. Đọc body bằng `io.Copy(io.Discard, ...)` để đếm byte (hoặc giữ 64KB đầu nếu `--body`), luôn `defer resp.Body.Close()`.
5. TLS: `resp.TLS.PeerCertificates[0]` → `NotAfter`, `DaysLeft`; Subject lấy CN hoặc SAN đầu.
6. Exit code: luôn 0 nếu nhận được response (kể cả 404/500 — đó là kết quả). Không kết nối được/timeout → exit 1.
7. User-Agent: `foxkit/<version>`.

## 6. Output
```
GET https://shop.example.com → 200 OK  (HTTP/2.0, 18.4 KB)

  Redirect   http://shop.example.com → 301
  DNS        12 ms
  Kết nối    30 ms
  TLS        45 ms
  Chờ        180 ms
  Tổng       241 ms

  SSL        Let's Encrypt R11 · hết hạn 2027-02-01 (còn 77 ngày) · TLS 1.3
  Header     content-type: text/html; charset=UTF-8
             server: nginx
             ...
```
`DaysLeft < 14` → in cảnh báo màu vàng (chỉ text mode).
JSON: `{ "ok": true, "data": <Result> }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `URL_INVALID` | | 2 | Ví dụ: foxkit http https://example.com |
| `HEADER_INVALID` | thiếu `:` | 2 | Dạng "Key: Value" |
| `HTTP_TIMEOUT` | | 1 | Tăng --timeout hoặc kiểm tra server |
| `HTTP_CONNECT_FAILED` | DNS/kết nối lỗi | 1 | Kiểm tra domain bằng foxkit dns <host> |
| `HTTP_TLS_FAILED` | chứng chỉ không hợp lệ | 1 | Dev local self-signed thì thêm --insecure |
| `TOO_MANY_REDIRECTS` | >10 | 1 | |

## 8. Manifest
```json
{ "name": "http", "title": "Kiểm tra URL", "description": "Status, thời gian, redirect, SSL của một URL", "category": "network",
  "args": [ { "name": "url", "type": "string", "required": true, "label": "URL" } ],
  "flags": [
    { "name": "method", "type": "enum", "options": ["GET","HEAD","POST","PUT","PATCH","DELETE","OPTIONS"], "default": "GET", "label": "Method" },
    { "name": "header", "type": "list", "separator": "\n", "label": "Header (mỗi dòng một)" },
    { "name": "data", "type": "text", "label": "Body" },
    { "name": "timeout", "type": "string", "default": "10s", "label": "Timeout" },
    { "name": "no-follow", "type": "bool", "default": false, "label": "Không theo redirect" },
    { "name": "insecure", "type": "bool", "default": false, "label": "Bỏ kiểm tra SSL" },
    { "name": "body", "type": "bool", "default": false, "label": "In body" }
  ], "admin": false, "longRunning": false, "dryRun": false, "output": "kv" }
```

## 9. Test
`httptest.NewServer` / `NewTLSServer`:
- 200 với header → parse đúng.
- `/a` 301 → `/b` 302 → `/c` 200 → 2 hop.
- `--no-follow` → status 301, không hop.
- Handler `time.Sleep(2s)` + timeout 500ms → `HTTP_TIMEOUT`.
- TLS server → `TLS != nil`; client không `--insecure` → `HTTP_TLS_FAILED` (vì cert test tự ký).

Chạy thử: `.\foxkit http google.com`, `.\foxkit http https://localhost:8443 --insecure`, `.\foxkit http https://api.example.com/login --method POST --data '{"email":"a@b.c"}' --body`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + test httptest.

## 11. Ngoài phạm vi
Benchmark nhiều request (dùng `hey`/`k6`), lưu cookie, HTTP/3, upload multipart.

## 12. Go học được
`net/http` client tùy biến, `httptrace`, `crypto/tls`/`x509`, `context` với timeout, `httptest` (≈ `Http::fake()`), luôn đóng `resp.Body` (rò rỉ kết nối nếu quên).
