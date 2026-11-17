# Spec: `foxkit hosts`

> Nhóm: **dev** · Giai đoạn: GĐ3 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Quản lý file hosts cho domain local (`shop.test`, `api.local`) mà không phải mở Notepad bằng quyền admin.

## 2. Cú pháp
```
foxkit hosts list [--managed] [--json]
foxkit hosts add <domain>... [--ip 127.0.0.1] [--apply]
foxkit hosts rm <domain>... [--apply]
foxkit hosts on <domain>... [--apply]      # bỏ comment dòng
foxkit hosts off <domain>... [--apply]     # comment dòng lại
```
| Cờ | Ghi chú |
|---|---|
| `--managed` | chỉ liệt kê dòng do foxkit thêm |
| `--ip` | IPv4 hoặc IPv6 hợp lệ (`net.ParseIP`) |
| `--apply` | ghi file (cần admin) |

Biến môi trường `FOXKIT_HOSTS` ghi đè đường dẫn file (để test và để dùng thử an toàn).

## 3. File
```
cmd/hosts.go             # lệnh cha
cmd/hosts_list.go, hosts_add.go, hosts_rm.go, hosts_toggle.go
internal/hosts/parse.go
internal/hosts/edit.go
internal/hosts/write.go
internal/hosts/hosts_test.go
```

## 4. Hợp đồng
```go
package hosts
type Line struct {
    Raw      string
    IP       string   // rỗng nếu không phải dòng entry
    Domains  []string // một dòng có thể nhiều domain
    Enabled  bool     // false nếu dòng entry bị comment "# 127.0.0.1 x"
    Managed  bool     // có hậu tố "# foxkit"
    Comment  string   // comment cuối dòng (ngoài "# foxkit")
}
type File struct { Lines []Line; CRLF bool; BOM bool }
type Entry struct { // cho output
    IP string `json:"ip"`; Domain string `json:"domain"`; Enabled bool `json:"enabled"`; Managed bool `json:"managed"`; Line int `json:"line"`
}
type Change struct { Action string `json:"action"`; Domain string `json:"domain"`; IP string `json:"ip"`; Line int `json:"line"` } // add, remove, enable, disable, skip

func Load(path string) (*File, error)
func (f *File) Entries() []Entry
func (f *File) Add(ip string, domains []string) []Change
func (f *File) Remove(domains []string) []Change
func (f *File) SetEnabled(domains []string, on bool) []Change
func Save(path string, f *File) (backup string, err error)
func DefaultPath() string // %SystemRoot%\System32\drivers\etc\hosts
```

## 5. Cách hoạt động
**Parse**
1. Đọc toàn bộ; ghi nhận BOM UTF-8 và kiểu xuống dòng (CRLF nếu có `\r\n`).
2. Mỗi dòng: trim; bắt đầu `#` → thử parse phần sau `#` như entry (để nhận dòng bị tắt); không thì là comment thường.
3. Entry: tách `strings.Fields`, field đầu phải là IP hợp lệ, các field sau đến khi gặp `#` là domain.

**Sửa (luôn sửa trong bộ nhớ, chỉ ghi khi `--apply`)**
- `add`: domain đã tồn tại với **cùng IP và đang bật** → `skip`. Tồn tại nhưng tắt → bật lại (`enable`). Tồn tại với IP khác → lỗi `HOSTS_DOMAIN_EXISTS` (gợi ý `rm` trước). Không có → thêm dòng mới cuối file: `127.0.0.1 shop.test # foxkit`. Domain hợp lệ: `^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*$`.
- `rm`: chỉ xóa domain khỏi dòng; dòng hết domain thì xóa dòng. Cho phép xóa cả dòng không phải foxkit thêm nhưng ghi `managed: false` trong change để người dùng thấy.
- `on/off`: đổi `Enabled`. Dòng có nhiều domain mà chỉ tắt 1 → tách domain đó ra dòng riêng rồi tắt.

**Ghi an toàn (`Save`)**
1. Copy file gốc sang `hosts.foxkit.bak` (ghi đè bản backup cũ).
2. Ghi nội dung mới ra `hosts.foxkit.tmp` cùng thư mục, giữ CRLF/BOM.
3. `os.Rename(tmp, hosts)` (cùng ổ nên nguyên tử).
4. Lỗi quyền ở bước bất kỳ → xóa tmp, trả `NEED_ADMIN` (exit 3).
5. Lưu ý: một số phần mềm diệt virus chặn ghi hosts → lỗi access denied ngay cả khi là admin; hint nên nhắc điều này.

Sau khi ghi: in nhắc "Trình duyệt có thể cache DNS, chạy `foxkit dns --flush` nếu chưa thấy thay đổi".

## 6. Output
`list`:
```
IP           DOMAIN          TRẠNG THÁI
127.0.0.1    shop.test       bật   (foxkit)
127.0.0.1    api.local       tắt   (foxkit)
10.0.0.5     db.internal     bật
```
`add` dry-run:
```
+ 127.0.0.1  shop.test
= 127.0.0.1  api.local (đã có, bỏ qua)
Đây là bản xem trước. Thêm --apply để thực hiện.
```
JSON `list`: `{ "ok": true, "data": { "file": "C:\\Windows\\System32\\drivers\\etc\\hosts", "entries": [Entry...] } }`
JSON sửa: `{ "ok": true, "data": { "applied": true, "changes": [Change...], "backup": "...\\hosts.foxkit.bak" } }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `HOSTS_DOMAIN_INVALID` | | 2 | Ví dụ: shop.test |
| `HOSTS_IP_INVALID` | | 2 | Ví dụ: --ip 127.0.0.1 |
| `HOSTS_DOMAIN_EXISTS` | trùng domain khác IP | 1 | Xóa trước: foxkit hosts rm <domain> --apply |
| `HOSTS_NOT_FOUND` | rm/on/off domain không có | 1 | Xem danh sách: foxkit hosts list |
| `NEED_ADMIN` | ghi bị từ chối | 3 | Chạy terminal bằng quyền Administrator. Nếu vẫn lỗi, phần mềm diệt virus có thể đang khóa file hosts |

## 8. Manifest
```json
{ "name": "hosts list", "title": "Xem hosts", "description": "Liệt kê domain trong file hosts", "category": "dev",
  "args": [], "flags": [ { "name": "managed", "type": "bool", "default": false, "label": "Chỉ dòng foxkit thêm" } ],
  "admin": false, "longRunning": false, "dryRun": false, "output": "table" },
{ "name": "hosts add", "title": "Thêm domain local", "description": "Thêm domain vào file hosts", "category": "dev",
  "args": [ { "name": "domain", "type": "string", "required": true, "multiple": true, "label": "Domain" } ],
  "flags": [
    { "name": "ip", "type": "string", "default": "127.0.0.1", "label": "IP" },
    { "name": "apply", "type": "bool", "default": false, "label": "Ghi file" }
  ], "admin": true, "longRunning": false, "dryRun": true, "output": "table" }
```
`hosts rm`, `hosts on`, `hosts off`: giống `hosts add` nhưng không có `--ip`.

## 9. Test
Dùng `FOXKIT_HOSTS` trỏ file tạm. Fixture `testdata/hosts_default` (file hosts mặc định Windows, CRLF):
- Parse không mất dòng nào; `Save` không thay đổi gì → byte giống hệt.
- `add shop.test` → thêm cuối với `# foxkit`, giữ CRLF.
- `add` lần 2 → skip.
- `off` 1 domain trong dòng 2 domain → tách dòng.
- `rm` domain cuối của dòng → xóa dòng.
- Thư mục tmp không ghi được → lỗi đúng code.

Chạy thử (PowerShell admin):
```powershell
.\foxkit hosts list
.\foxkit hosts add shop.test api.shop.test
.\foxkit hosts add shop.test api.shop.test --apply
ping shop.test        # phải ra 127.0.0.1
.\foxkit hosts off shop.test --apply
.\foxkit hosts rm shop.test api.shop.test --apply
```

## 10. Tiêu chí hoàn thành
Tiêu chí chung + round-trip byte-for-byte + luôn có backup trước khi ghi.

## 11. Ngoài phạm vi
Profile hosts (bật/tắt nhóm), tải danh sách chặn quảng cáo, tự xin quyền admin (UAC) — launcher làm việc đó.

## 12. Go học được
Parser giữ nguyên định dạng, ghi file nguyên tử (tmp + rename), `net.ParseIP`, `os.Getenv` để override cấu hình cho test, lệnh con cobra dùng chung cờ.
