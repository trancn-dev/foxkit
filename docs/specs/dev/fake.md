# Spec: `foxkit fake`

> Nhóm: **dev** · Giai đoạn: GĐ7 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md), [dev/case.md](case.md) (`textcase.RemoveDiacritics`)

## 1. Mục đích
Sinh dữ liệu giả tiếng Việt để seed DB, test form, demo (giống `fake()` / Faker của Laravel, nhưng chạy tức thì từ dòng lệnh và ra JSON/CSV).

## 2. Cú pháp
```
foxkit fake <field>... [--count 10] [--format table|json|csv|ndjson] [--seed N] [--locale vi|en] [--json]
foxkit fake --list
```
`<field>`: `name`, `first`, `last`, `email`, `phone`, `address`, `city`, `company`, `username`, `uuid`, `date`, `datetime`, `int`, `bool`, `text`, `url`, `ip`, `password`. Mỗi field có thể kèm tham số: `int:1-100`, `date:2020-01-01..2026-12-31`, `text:20` (số từ), `pick:a|b|c`.

| Cờ | Mặc định | Ghi chú |
|---|---|---|
| `--count` | 10 | 1–100.000 |
| `--format` | `table` | `csv` có header; `ndjson` mỗi dòng một object (nạp thẳng vào DB tool) |
| `--seed` | ngẫu nhiên | cùng seed → cùng kết quả (test lặp lại được) |
| `--locale` | `vi` | |

## 3. File
```
cmd/fake.go
internal/fake/fake.go        # Generator, parse field spec
internal/fake/data_vi.go     # danh sách họ/tên đệm/tên/phường/quận/tỉnh/công ty
internal/fake/data_en.go
internal/fake/fake_test.go
```

## 4. Hợp đồng
```go
package fake
type Field struct { Kind string; Arg string }
type Generator struct { rng *rand.Rand; locale string }
func New(seed int64, locale string) *Generator
func ParseField(s string) (Field, error)
func (g *Generator) Value(f Field) (any, error)
func (g *Generator) Rows(fields []Field, n int) ([]map[string]any, error)
```

## 5. Cách hoạt động
1. Dùng `math/rand/v2` với `rand.New(rand.NewPCG(seed, seed))` để **có thể lặp lại** theo `--seed` (đây là dữ liệu giả, không phải bí mật nên không dùng `crypto/rand`; `password` thì dùng `internal/pass`).
2. `name` (vi): họ (Nguyễn, Trần, Lê, Phạm, Hoàng, Huỳnh, Phan, Vũ, Võ, Đặng, Bùi, Đỗ, Hồ, Ngô, Dương, Lý — có trọng số, Nguyễn ~38%) + tên đệm (Văn/Thị/Hữu/Minh/Ngọc...) + tên; tên đệm `Văn` chỉ đi với tên nam, `Thị` với nữ → sinh giới tính trước rồi chọn.
3. `email`: từ tên bỏ dấu + số ngẫu nhiên + domain (`example.com`, `example.org`, `test.vn` — **chỉ domain dành riêng cho ví dụ**, không sinh domain thật để tránh gửi nhầm).
4. `phone`: đầu số di động VN thật (`03x, 05x, 07x, 08x, 09x`) + 7 số, định dạng `09xx xxx xxx` hoặc `+84...` theo tham số `phone:intl`.
5. `address`: `số nhà + đường + phường + quận + tỉnh/thành` từ bảng nhỏ trong code (khoảng 20 tỉnh/thành lớn, mỗi nơi vài quận) — không cố chính xác tuyệt đối, chỉ đủ giống thật.
6. `date`, `datetime`: khoảng mặc định 5 năm gần nhất; định dạng ISO.
7. `pick:a|b|c`: chọn ngẫu nhiên một giá trị.
8. Cùng một hàng, field `email`/`username` dùng lại tên đã sinh ở field `name` của hàng đó nếu có (dữ liệu nhất quán).
9. Output `table` bằng `text/tabwriter`; `csv` bằng `encoding/csv`, mặc định không BOM; thêm cờ `--bom` để Excel đọc đúng tiếng Việt.

## 6. Output
```
> foxkit fake name email phone --count 3 --seed 1
NAME              EMAIL                        PHONE
Nguyễn Văn Hùng   hung.nguyen42@example.com    0912 345 678
Trần Thị Lan      lan.tran7@example.org        0357 120 884
...
```
JSON: `{ "ok": true, "data": { "count": 3, "seed": 1, "rows": [ { "name": "...", "email": "...", "phone": "..." } ] } }`
`--format ndjson`: mỗi dòng một object, không envelope.

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `FAKE_NO_FIELD` | | 2 | Ví dụ: foxkit fake name email --count 5 |
| `FAKE_FIELD_INVALID` | field lạ hoặc tham số sai | 2 | Xem danh sách: foxkit fake --list |
| `FAKE_COUNT_INVALID` | | 2 | Từ 1 đến 100000 |

## 8. Manifest
```json
{ "name": "fake", "title": "Dữ liệu giả", "description": "Sinh dữ liệu giả tiếng Việt: tên, SĐT, địa chỉ, email...", "category": "dev",
  "args": [ { "name": "field", "type": "string", "required": false, "multiple": true, "label": "Field (name, email, phone...)" } ],
  "flags": [
    { "name": "count", "type": "int", "default": 10, "min": 1, "max": 100000, "label": "Số dòng" },
    { "name": "format", "type": "enum", "options": ["table","json","csv","ndjson"], "default": "table", "label": "Định dạng" },
    { "name": "seed", "type": "int", "label": "Seed (lặp lại được)" },
    { "name": "locale", "type": "enum", "options": ["vi","en"], "default": "vi", "label": "Ngôn ngữ" },
    { "name": "bom", "type": "bool", "default": false, "label": "CSV có BOM (cho Excel)" },
    { "name": "list", "type": "bool", "default": false, "label": "Liệt kê field" }
  ], "admin": false, "longRunning": false, "dryRun": false, "output": "table" }
```

## 9. Test
- Cùng `--seed` → byte giống hệt; khác seed → khác.
- `phone` khớp regex `^0(3|5|7|8|9)\d{8}$` sau khi bỏ khoảng trắng (1000 lần).
- Tên nam không có `Thị`, nữ không có `Văn`.
- `email` luôn thuộc 3 domain ví dụ, không có ký tự có dấu.
- `int:1-5` luôn trong khoảng; `pick:a|b` chỉ ra a hoặc b.
- CSV có `--bom` bắt đầu bằng `EF BB BF`.

Chạy thử: `.\foxkit fake name email phone --count 5`, `.\foxkit fake name "int:18-60" --count 100 --format csv --bom > users.csv`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + test trên + 100.000 dòng sinh trong dưới 2 giây.

## 11. Ngoài phạm vi
Nạp thẳng vào DB, sinh theo schema bảng MySQL, ảnh đại diện, dữ liệu liên quan nhiều bảng.

## 12. Go học được
`math/rand/v2` với seed (khác `crypto/rand`), chọn có trọng số, slice của map, `encoding/csv` writer, đặt dữ liệu tĩnh lớn trong file riêng.
