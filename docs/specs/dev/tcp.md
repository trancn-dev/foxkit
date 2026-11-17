# Spec: `foxkit tcp`

> Nhóm: **dev** · Giai đoạn: GĐ4 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Kiểm tra nhanh host:port có mở không (MySQL 3306, Redis 6379, SMTP 587, server staging 22). Thay cho `telnet`/`Test-NetConnection` chậm.

## 2. Cú pháp
```
foxkit tcp <host:port>... [--timeout 3s] [--json]
```
- Nhận `host:port`, `[ipv6]:port`, hoặc URL (`https://a.com` → port 443, `http://` → 80, `mysql://h` → 3306, `redis://h` → 6379, `postgres://h` → 5432).
- Tối đa 100 đích.

## 3. File
```
cmd/tcp.go
internal/tcpcheck/tcp.go
internal/tcpcheck/tcp_test.go
```

## 4. Hợp đồng
```go
package tcpcheck
type Item struct {
    Target  string `json:"target"`   // như người dùng nhập
    Addr    string `json:"addr"`     // host:port đã chuẩn hóa
    IP      string `json:"ip,omitempty"`
    Open    bool   `json:"open"`
    Ms      int64  `json:"ms"`
    Error   string `json:"error,omitempty"` // "timeout", "từ chối kết nối", "không phân giải được tên"
}
func ParseTarget(s string) (addr string, err error)
func CheckAll(ctx context.Context, targets []string, timeout time.Duration) []Item
```

## 5. Cách hoạt động
1. `ParseTarget` cho từng đích; lỗi parse → item lỗi, không dừng cả lệnh.
2. Mỗi đích một goroutine; kết quả ghi vào `items[i]` theo index (mỗi goroutine ghi ô riêng nên không cần mutex) → **giữ đúng thứ tự nhập**.
3. `net.Dialer{Timeout: timeout}.DialContext(ctx, "tcp", addr)` → thành công thì `conn.RemoteAddr()` lấy IP, `conn.Close()`.
4. Phân loại lỗi:
   - `errors.Is(err, os.ErrDeadlineExceeded)` hoặc `net.Error.Timeout()` → "timeout" (thường là firewall chặn).
   - `errors.Is(err, syscall.ECONNREFUSED)` (Windows: `WSAECONNREFUSED`, Go map về ECONNREFUSED) → "từ chối kết nối" (máy sống nhưng không có service).
   - `*net.DNSError` → "không phân giải được tên".
5. Exit: tất cả mở → 0; có đích đóng → 1 (để script `if` được), vẫn in đủ kết quả stdout và lỗi tổng `TCP_SOME_CLOSED` ra stderr.

## 6. Output
```
✓  db.local:3306       10.0.0.5   4 ms
✗  redis.local:6379    10.0.0.6   từ chối kết nối
✗  staging:22                     timeout (3s)
```
JSON: `{ "ok": true, "data": { "items": [Item...] } }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `TCP_NO_TARGET` | | 2 | Ví dụ: foxkit tcp localhost:3306 |
| `TCP_TARGET_INVALID` | (theo item) thiếu port | 1 | Dạng host:port |
| `TCP_SOME_CLOSED` | có đích không mở | 1 | Timeout thường do firewall; từ chối thường do service chưa chạy |

## 8. Manifest
```json
{ "name": "tcp", "title": "Kiểm tra port từ xa", "description": "Kiểm tra host:port có mở không", "category": "network",
  "args": [ { "name": "target", "type": "string", "required": true, "multiple": true, "label": "host:port" } ],
  "flags": [ { "name": "timeout", "type": "string", "default": "3s", "label": "Timeout" } ],
  "admin": false, "longRunning": false, "dryRun": false, "output": "table" }
```

## 9. Test
`ParseTarget` bảng (`a:1`, `[::1]:80`, `https://a.com`, `mysql://db`, `a` → lỗi). `CheckAll` với `net.Listen("tcp", "127.0.0.1:0")` (mở) + port vừa đóng (từ chối) → thứ tự giữ nguyên.

Chạy thử: `.\foxkit tcp localhost:3306 localhost:6379 google.com:443`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + 100 đích chạy song song xong trong ~1 timeout.

## 11. Ngoài phạm vi
Quét dải port, UDP, ping ICMP (cần quyền đặc biệt).

## 12. Go học được
`net.Dialer`, phân loại lỗi mạng bằng `errors.Is`/`errors.As`, goroutine ghi vào slice theo index (an toàn vì mỗi goroutine một ô) — so với `Promise.all` giữ thứ tự tự nhiên.
