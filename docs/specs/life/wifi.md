# Spec: `foxkit wifi`

> Nhóm: **life** · Giai đoạn: GĐ3 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Xem mật khẩu các mạng Wi-Fi đã lưu trên máy (khi khách hỏi pass, hoặc cần cài cho máy khác).

## 2. Cú pháp
```
foxkit wifi [ssid] [--json]
```
Không có `ssid`: liệt kê mọi profile kèm mật khẩu. Có `ssid`: chỉ profile đó (không phân biệt hoa thường).

## 3. File
```
cmd/wifi.go
internal/wifi/wifi.go
internal/wifi/xml.go
internal/wifi/wifi_test.go
internal/wifi/testdata/profile_wpa2.xml
```

## 4. Hợp đồng
```go
package wifi
type Profile struct {
    SSID     string `json:"ssid"`
    Auth     string `json:"auth"`               // WPA2PSK, WPA3SAE, open...
    Password string `json:"password,omitempty"` // rỗng nếu mạng mở hoặc không đọc được
    Hidden   bool   `json:"hidden"`
    Error    string `json:"error,omitempty"`
}
func Export(ctx context.Context, ssid string) ([]Profile, error)
func ParseProfileXML(b []byte) (Profile, error)
```

## 5. Cách hoạt động
1. `dir, _ := os.MkdirTemp("", "foxkit-wifi-")` → `defer os.RemoveAll(dir)` (file export chứa mật khẩu dạng rõ, **phải xóa** dù lỗi).
2. `netsh wlan export profile key=clear folder="<dir>"` (thêm `name="<ssid>"` nếu có). Dùng export XML vì output `netsh wlan show profile` bị dịch theo ngôn ngữ Windows, còn XML có cấu trúc cố định.
3. Đọc mọi `*.xml` trong dir, `encoding/xml` decode vào struct:
   ```go
   type wlanProfile struct {
       Name string `xml:"name"`
       SSIDConfig struct { NonBroadcast bool `xml:"nonBroadcast"` } `xml:"SSIDConfig"`
       MSM struct { Security struct {
           AuthEncryption struct { Authentication string `xml:"authentication"` } `xml:"authEncryption"`
           SharedKey struct { KeyMaterial string `xml:"keyMaterial"`; Protected bool `xml:"protected"` } `xml:"sharedKey"`
       } `xml:"security"` } `xml:"MSM"`
   }
   ```
   Lưu ý XML có namespace mặc định; `encoding/xml` khớp theo local name nên không cần khai báo namespace.
4. `Protected == true` (không giải mã được, thường khi thiếu quyền) → `Password = ""`, `Error = "cần quyền admin để xem mật khẩu"`.
5. Máy không có card Wi-Fi / dịch vụ WLAN tắt: netsh trả lỗi → `WIFI_UNAVAILABLE`.
6. Không có profile nào → `items: []`, exit 0.
7. Sắp theo SSID.

## 6. Output
```
SSID            BẢO MẬT    MẬT KHẨU
Office-5G       WPA2PSK    s3cr3t-pass
Cafe Free       open       (mạng mở)
Home            WPA3SAE    (cần quyền admin)
```
JSON: `{ "ok": true, "data": { "items": [Profile...] } }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `WIFI_UNAVAILABLE` | không có Wi-Fi/WLAN AutoConfig tắt | 1 | Máy cần card Wi-Fi và dịch vụ WLAN AutoConfig đang chạy |
| `WIFI_NOT_FOUND` | ssid chỉ định không có | 1 | Chạy foxkit wifi để xem danh sách |

## 8. Manifest
```json
{ "name": "wifi", "title": "Mật khẩu Wi-Fi", "description": "Xem mật khẩu Wi-Fi đã lưu", "category": "network",
  "args": [ { "name": "ssid", "type": "string", "required": false, "label": "Tên Wi-Fi" } ],
  "flags": [], "admin": false, "longRunning": false, "dryRun": false, "output": "table" }
```

## 9. Test
`ParseProfileXML` với fixture: WPA2 có `keyMaterial`, mạng open (không có `sharedKey`), `protected=true`. Tạo fixture bằng tay theo mẫu XML của Windows (không commit mật khẩu thật).

Chạy thử: `.\foxkit wifi`, `.\foxkit wifi "Office-5G" --json`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + thư mục tạm luôn bị xóa (kiểm tra `%TEMP%` sau khi chạy, kể cả khi lỗi).

## 11. Ngoài phạm vi
Kết nối Wi-Fi, quét mạng xung quanh, sinh QR Wi-Fi (xem backlog `qr`).

## 12. Go học được
`encoding/xml` với struct lồng nhau và tag, `os.MkdirTemp` + `defer os.RemoveAll` cho dữ liệu nhạy cảm, `filepath.Glob`.
