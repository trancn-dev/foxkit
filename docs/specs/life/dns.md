# Spec: `foxkit dns`

> Nhóm: **life** · Giai đoạn: GĐ3 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Tra bản ghi DNS (domain đã trỏ đúng server chưa, MX/TXT cho mail) và xóa cache DNS của Windows.

## 2. Cú pháp
```
foxkit dns <host> [--type A,AAAA,CNAME,MX,TXT,NS] [--server 1.1.1.1] [--json]
foxkit dns --flush [--json]
```
| Cờ | Mặc định | Ghi chú |
|---|---|---|
| `--type` | `A,AAAA,CNAME` | `all` = cả 6 loại |
| `--server` | (DNS của hệ thống) | tra qua DNS server chỉ định (so sánh khi vừa đổi DNS, chưa lan truyền) |
| `--flush` | | không đi cùng `<host>` |

## 3. File
```
cmd/dns.go
internal/dns/lookup.go
internal/dns/flush.go
internal/dns/dns_test.go
```

## 4. Hợp đồng
```go
package dns
type Records struct {
    A     []string `json:"A,omitempty"`
    AAAA  []string `json:"AAAA,omitempty"`
    CNAME string   `json:"CNAME,omitempty"`
    MX    []MX     `json:"MX,omitempty"`
    TXT   []string `json:"TXT,omitempty"`
    NS    []string `json:"NS,omitempty"`
}
type MX struct { Host string `json:"host"`; Pref uint16 `json:"pref"` }
type Result struct {
    Host    string            `json:"host"`
    Server  string            `json:"server"`  // "system" hoặc IP
    Records Records           `json:"records"`
    Errors  map[string]string `json:"errors"`  // lỗi theo từng loại, ví dụ {"MX":"không có bản ghi"}
    Ms      int64             `json:"ms"`
}
func Lookup(ctx context.Context, host string, types []string, server string) Result
func Flush(ctx context.Context) error
```

## 5. Cách hoạt động
1. Chuẩn hóa host: bỏ `http://`, `https://`, path, port (người dùng hay dán cả URL).
2. Resolver:
   - Mặc định `net.DefaultResolver`.
   - `--server`: `&net.Resolver{PreferGo: true, Dial: func(ctx, network, _ string) { return (&net.Dialer{}).DialContext(ctx, "udp", server+":53") }}`.
3. Gọi song song các loại (`LookupIP` với network `ip4`/`ip6`, `LookupCNAME`, `LookupMX`, `LookupTXT`, `LookupNS`), mỗi loại timeout 5s qua `context.WithTimeout`.
4. Lỗi "không có bản ghi" (`*net.DNSError` với `IsNotFound`) → ghi vào `Errors`, không coi là lỗi lệnh. Mọi loại đều lỗi và host không tồn tại → exit 1 `DNS_NOT_FOUND`.
5. CNAME trả về chính host (kèm dấu chấm cuối) nghĩa là không có CNAME → bỏ.
6. `--flush`: `ipconfig /flushdns` qua `winexec.Run`. Kiểm tra exit code; output bị dịch theo ngôn ngữ nên không parse chữ.

## 6. Output
```
shop.example.com  (qua DNS hệ thống, 34ms)
  A      203.0.113.10
  CNAME  shop-lb.example.net.
  MX     (không có bản ghi)
```
`--flush`: `Đã xóa cache DNS.`
JSON: `{ "ok": true, "data": <Result> }` · flush: `{ "ok": true, "data": { "flushed": true } }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `DNS_NO_HOST` | | 2 | Ví dụ: foxkit dns example.com |
| `DNS_TYPE_INVALID` | | 2 | Chọn: A, AAAA, CNAME, MX, TXT, NS, all |
| `DNS_SERVER_INVALID` | `--server` không phải IP | 2 | Ví dụ: --server 1.1.1.1 |
| `DNS_NOT_FOUND` | host không tồn tại | 1 | Kiểm tra chính tả domain |
| `DNS_FLUSH_FAILED` | | 1 | Thử chạy "ipconfig /flushdns" bằng quyền Administrator |

## 8. Manifest
```json
{ "name": "dns", "title": "Tra DNS", "description": "Tra bản ghi DNS, xóa cache DNS", "category": "network",
  "args": [ { "name": "host", "type": "string", "required": false, "label": "Domain" } ],
  "flags": [
    { "name": "type", "type": "list", "default": "A,AAAA,CNAME", "label": "Loại bản ghi" },
    { "name": "server", "type": "string", "label": "DNS server" },
    { "name": "flush", "type": "bool", "default": false, "label": "Xóa cache DNS" }
  ], "admin": false, "longRunning": false, "dryRun": false, "output": "kv" }
```

## 9. Test
Chuẩn hóa host (bảng: `https://a.com/x` → `a.com`, `a.com:8080` → `a.com`). Lookup thật đánh dấu `if testing.Short() { t.Skip() }` để không cần mạng khi `go test -short`.

Chạy thử: `.\foxkit dns google.com --type all`, `.\foxkit dns mysite.com --server 8.8.8.8`, `.\foxkit dns --flush`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + `go test -short` không cần mạng.

## 11. Ngoài phạm vi
DNS over HTTPS, truy vết đệ quy (`dig +trace`), SRV/CAA/SOA.

## 12. Go học được
`net.Resolver` tùy biến, chạy song song nhiều truy vấn và gom kết quả có mutex (hoặc channel), `errors.As` với `*net.DNSError`, `testing.Short()`.
