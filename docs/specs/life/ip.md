# Spec: `foxkit ip`

> Nhóm: **life** · Giai đoạn: GĐ3 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Xem IP LAN của máy (để mở web đang dev trên điện thoại khi chạy `vite --host`, `php artisan serve --host=0.0.0.0`) và IP public khi cần.

## 2. Cú pháp
```
foxkit ip [--public] [--all] [--v6] [--json]
```
| Cờ | Mặc định | Ghi chú |
|---|---|---|
| `--public` | false | gọi dịch vụ ngoài để lấy IP public (gửi request ra Internet nên phải bật tay) |
| `--all` | false | gồm card ảo (vEthernet/Hyper-V/WSL, VirtualBox, VMware, Docker), card đang down |
| `--v6` | false | gồm IPv6 (mặc định chỉ IPv4; IPv6 link-local `fe80::` luôn ẩn trừ khi `--all`) |

## 3. File
```
cmd/ip.go
internal/netinfo/ip.go          # dùng lại ở serve (GĐ4)
internal/netinfo/public.go
internal/netinfo/ip_test.go
```

## 4. Hợp đồng
```go
package netinfo
type Addr struct {
    Iface   string `json:"iface"`   // "Wi-Fi", "Ethernet"
    IP      string `json:"ip"`
    Prefix  int    `json:"prefix"`  // 24
    V6      bool   `json:"v6"`
    Virtual bool   `json:"virtual"`
    MAC     string `json:"mac"`
}
type Result struct {
    LAN     []Addr `json:"lan"`
    Primary string `json:"primary"`           // IP được dùng để ra Internet
    Public  string `json:"public,omitempty"`
}
func LANAddrs(includeVirtual, includeV6 bool) ([]Addr, error)
func PrimaryIP() (string, error)
func PublicIP(ctx context.Context) (string, error)
func IsVirtual(ifaceName string) bool
```

## 5. Cách hoạt động
1. `net.Interfaces()` → bỏ loopback (`FlagLoopback`), bỏ card không `FlagUp` (trừ `--all`).
2. `iface.Addrs()` → ép kiểu `*net.IPNet` → lấy IP và prefix (`Mask.Size()`).
3. `IsVirtual`: tên chứa (không phân biệt hoa thường) `vethernet`, `virtualbox`, `vmware`, `hyper-v`, `docker`, `wsl`, `loopback`, `bluetooth`, `tailscale`, `zerotier`. (Tailscale/ZeroTier thật ra hữu ích nhưng không phải LAN; vẫn hiện khi `--all`.)
4. `PrimaryIP`: mở UDP "kết nối" tới `8.8.8.8:80` (`net.Dial("udp", ...)` — **không gửi gói nào**, chỉ để hệ điều hành chọn route) rồi đọc `LocalAddr()`. Đánh dấu IP này trong danh sách.
5. `PublicIP`: GET `https://api.ipify.org` timeout 5s, kiểm tra kết quả là IP hợp lệ; lỗi thì thử `https://icanhazip.com`. Lỗi cả hai → `IP_PUBLIC_FAILED` (vẫn in phần LAN, exit 1).

## 6. Output
```
LAN
  Wi-Fi      192.168.1.23/24   ← dùng để ra Internet
  Ethernet   10.0.0.15/24
Public       113.161.x.x
Mở trên điện thoại: http://192.168.1.23:<port>
```
JSON: `{ "ok": true, "data": <Result> }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `IP_LIST_FAILED` | `net.Interfaces` lỗi | 1 | |
| `IP_PUBLIC_FAILED` | không lấy được IP public | 1 | Kiểm tra kết nối Internet hoặc proxy |

## 8. Manifest
```json
{ "name": "ip", "title": "Xem IP", "description": "IP LAN và IP public", "category": "network",
  "args": [],
  "flags": [
    { "name": "public", "type": "bool", "default": false, "label": "Lấy IP public" },
    { "name": "all", "type": "bool", "default": false, "label": "Gồm card ảo" },
    { "name": "v6", "type": "bool", "default": false, "label": "Gồm IPv6" }
  ], "admin": false, "longRunning": false, "dryRun": false, "output": "kv" }
```

## 9. Test
`IsVirtual` dạng bảng (`"vEthernet (WSL)"` → true, `"Wi-Fi"` → false). `PublicIP` test bằng `httptest.NewServer` (cho phép inject URL). Phần interface thật: test thủ công.

Chạy thử: `.\foxkit ip`, `.\foxkit ip --public --json`, `.\foxkit ip --all --v6`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + mặc định không gửi request ra ngoài khi không có `--public`.

## 11. Ngoài phạm vi
Gateway, DNS server đang dùng, tốc độ mạng, quét LAN.

## 12. Go học được
Package `net`, type assertion `addr.(*net.IPNet)` (≈ `instanceof`), `httptest.NewServer` để test HTTP (≈ `Http::fake()` của Laravel), mẹo UDP dial.
