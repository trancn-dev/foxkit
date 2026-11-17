# Spec: `foxkit uuid`

> Nhóm: **dev** · Giai đoạn: GĐ1 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)
> **Nên làm lệnh này đầu tiên sau khung**: đơn giản, chạy thông cả luồng `cmd → internal → output → manifest`.

## 1. Mục đích
Sinh UUID v4 (ngẫu nhiên) hoặc v7 (có thứ tự theo thời gian) để dán vào seed, test, migration.

## 2. Cú pháp
```
foxkit uuid [--v 4|7] [--count N] [--upper] [--no-dash] [--json]
```
| Cờ | Kiểu | Mặc định | Ràng buộc |
|---|---|---|---|
| `--v` | enum `4`,`7` | `4` | khác 4/7 → `UUID_VERSION_INVALID` |
| `--count` | int | `1` | 1–1000 → ngoài khoảng `UUID_COUNT_INVALID` |
| `--upper` | bool | false | chữ hoa |
| `--no-dash` | bool | false | bỏ dấu `-` (32 ký tự hex) |

## 3. File
```
cmd/uuid.go
internal/uuid/uuid.go
internal/uuid/uuid_test.go
```

## 4. Hợp đồng
```go
package uuid
type UUID [16]byte
func NewV4() (UUID, error)
func NewV7(now time.Time) (UUID, error)  // truyền now để test được
func (u UUID) String() string            // dạng 8-4-4-4-12, chữ thường
func (u UUID) Version() int
```
> **Đã kiểm (Go 1.27):** thư viện chuẩn có package `uuid` (`uuid.NewV4()`, `uuid.NewV7()`, `String()`), nên **không có `internal/uuid`**; `cmd/uuid.go` dùng thẳng, không có lỗi `UUID_RANDOM_FAILED` (hàm không trả error). Test nằm ở `cmd/uuid_test.go`.

Trước khi viết, kiểm tra bản Go đang dùng đã có package UUID trong thư viện chuẩn chưa (`go doc` hoặc trang release notes). **Có thì dùng luôn và bỏ `internal/uuid`**; chưa có thì tự viết như dưới.

## 5. Cách hoạt động
**v4**
1. Đọc 16 byte từ `crypto/rand` (`rand.Read`). Lỗi thì trả `UUID_RANDOM_FAILED`.
2. `b[6] = (b[6] & 0x0f) | 0x40` (version 4).
3. `b[8] = (b[8] & 0x3f) | 0x80` (variant RFC 9562).

**v7**
1. `ms := now.UnixMilli()`; ghi 48 bit cao vào `b[0..5]` (big-endian).
2. 10 byte còn lại lấy từ `crypto/rand`.
3. `b[6] = (b[6] & 0x0f) | 0x70`; `b[8] = (b[8] & 0x3f) | 0x80`.
4. Khi `--count > 1`, các UUID sinh trong cùng 1 ms vẫn phải tăng dần: sắp xếp lại danh sách theo byte sau khi sinh (cách đơn giản, đủ dùng cho CLI).

**String**: `hex.Encode` rồi chèn `-` ở vị trí 8, 12, 16, 20. `--upper`/`--no-dash` xử lý ở tầng `cmd`.

## 6. Output
Text: mỗi UUID một dòng (dễ pipe).
```
> foxkit uuid --v 7 --count 2
01937a2e-8f4c-7b3a-9d21-3c5e8f0a1b2c
01937a2e-8f4c-7c10-a2b4-77e1d09c5f31
```
JSON:
```json
{ "ok": true, "data": { "version": 7, "count": 2, "items": ["01937a2e-8f4c-7b3a-9d21-3c5e8f0a1b2c", "01937a2e-8f4c-7c10-a2b4-77e1d09c5f31"] } }
```

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `UUID_VERSION_INVALID` | `--v` khác 4/7 | 2 | Dùng --v 4 hoặc --v 7 |
| `UUID_COUNT_INVALID` | count ngoài 1–1000 | 2 | Ví dụ: --count 10 |
| `UUID_RANDOM_FAILED` | crypto/rand lỗi | 1 | Thử chạy lại |

## 8. Manifest
```json
{ "name": "uuid", "title": "Sinh UUID", "description": "Sinh UUID v4 hoặc v7", "category": "dev",
  "args": [],
  "flags": [
    { "name": "v", "type": "enum", "options": ["4","7"], "default": "4", "label": "Phiên bản" },
    { "name": "count", "type": "int", "default": 1, "min": 1, "max": 1000, "label": "Số lượng" },
    { "name": "upper", "type": "bool", "default": false, "label": "Chữ hoa" },
    { "name": "no-dash", "type": "bool", "default": false, "label": "Bỏ dấu gạch" }
  ],
  "admin": false, "longRunning": false, "dryRun": false, "output": "text" }
```

## 9. Test
- Regex `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$` cho v4 (chạy 1000 lần).
- v7: ký tự đầu nhóm 3 là `7`; 48 bit đầu bằng `now.UnixMilli()` đã truyền vào.
- v7 với 2 thời điểm `t1 < t2` → `String()` của t1 nhỏ hơn t2 (so chuỗi).
- 10.000 UUID v4 không trùng.

Chạy thử:
```powershell
.\foxkit uuid
.\foxkit uuid --v 7 --count 5
.\foxkit uuid --upper --no-dash --json
.\foxkit uuid --v 5; echo $LASTEXITCODE   # 2
```

## 10. Tiêu chí hoàn thành
Tiêu chí chung (foundation §12) + test mục 9 pass + entry manifest có mặt.

## 11. Ngoài phạm vi
v1/v3/v5/v6, ULID, copy vào clipboard.

## 12. Go học được
Mảng cố định `[16]byte` (khác slice), phép bit `&` `|`, `crypto/rand`, `encoding/hex`, method trên kiểu tự định nghĩa (`func (u UUID) String()` ≈ `__toString()` của PHP, và `fmt` tự gọi nó vì thỏa interface `fmt.Stringer`).
