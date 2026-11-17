# Spec: `foxkit rename`

> Nhóm: **life** · Giai đoạn: GĐ2 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Đổi tên hàng loạt file trong thư mục: tìm-thay thế (chuỗi hoặc regex), đổi kiểu chữ, đánh số thứ tự. Mặc định chỉ xem trước.

## 2. Cú pháp
```
foxkit rename <dir> --find <s> --replace <s> [--regex] [--ignore-case] [--ext jpg,png] [--recursive] [--dirs] [--apply] [--json]
foxkit rename <dir> --case lower|upper|slug|snake|kebab [--ext ...] [--apply]
foxkit rename <dir> --number "Trip_{n:3}" [--sort name|date|size] [--start 1] [--ext ...] [--apply]
```
Ba chế độ loại trừ nhau: `--find/--replace`, `--case`, `--number`.

| Cờ | Ghi chú |
|---|---|
| `--find`, `--replace` | `--replace` có thể rỗng (xóa chuỗi) |
| `--regex` | dùng `regexp` (RE2); `--replace` hỗ trợ `$1`, `${name}` |
| `--ignore-case` | thêm `(?i)` nếu regex; với chuỗi thường thì so không phân biệt hoa thường |
| `--case` | áp dụng cho **phần tên**, giữ đuôi file; dùng lại `internal/textcase` |
| `--number` | mẫu có `{n}` hoặc `{n:3}` (đệm 0 đủ 3 chữ số); giữ đuôi file gốc |
| `--sort` | thứ tự đánh số, mặc định `name` (so sánh tự nhiên: `img2` < `img10`) |
| `--ext` | lọc đuôi |
| `--recursive` | gồm thư mục con |
| `--dirs` | đổi tên cả thư mục (mặc định chỉ file) |
| `--apply` | thực hiện |

Mặc định find/replace chỉ áp dụng lên **tên không gồm đuôi**; thêm `--with-ext` để áp lên cả tên đầy đủ.

## 3. File
```
cmd/rename.go
internal/rename/plan.go     # tính danh sách thay đổi
internal/rename/apply.go    # thực thi an toàn
internal/rename/rename_test.go
```

## 4. Hợp đồng
```go
package rename
type Change struct {
    Dir  string `json:"dir"`
    From string `json:"from"`
    To   string `json:"to"`
}
type Conflict struct {
    To     string   `json:"to"`
    From   []string `json:"from"`
    Reason string   `json:"reason"` // "trùng tên đích", "file đã tồn tại", "tên không hợp lệ"
}
type Plan struct {
    Applied   bool       `json:"applied"`
    Changes   []Change   `json:"changes"`
    Unchanged int        `json:"unchanged"`
    Conflicts []Conflict `json:"conflicts"`
}
func Build(opt Options) (Plan, error)
func Apply(p Plan) (done int, err error)
```

## 5. Cách hoạt động
**Build**
1. Liệt kê file (theo `--recursive`, `--ext`, `--dirs`).
2. Tính tên mới cho từng file; tên không đổi → `Unchanged++`.
3. Kiểm tra tên hợp lệ trên Windows: không chứa `<>:"/\|?*`, không kết thúc bằng dấu chấm/dấu cách, không phải tên dành riêng (`CON`, `PRN`, `AUX`, `NUL`, `COM1-9`, `LPT1-9`), không rỗng.
4. Phát hiện xung đột **không phân biệt hoa thường** (NTFS): hai file cùng ra một tên; hoặc tên đích đã tồn tại và file đó không nằm trong danh sách đổi.
5. Có xung đột → `Apply` từ chối toàn bộ (`RENAME_CONFLICT`), không đổi một phần.

**Apply** (chỉ khi `--apply` và không xung đột)
1. Đổi tên 2 pha để xử lý chuỗi hoán đổi (`a→b`, `b→a`) và đổi chỉ khác hoa thường (`a.txt→A.txt`):
   - Pha 1: mọi file → tên tạm `.<from>.foxkit-tmp-<random>`.
   - Pha 2: tên tạm → tên đích.
2. Ghi nhật ký các bước đã làm; lỗi giữa chừng → **hoàn tác** các bước đã xong theo thứ tự ngược, rồi báo lỗi.
3. Thư mục được đổi tên sau cùng, từ sâu nhất lên (để đường dẫn file con không bị sai).

## 6. Output
Text:
```
IMG_001.JPG  →  trip_001.jpg
IMG_002.JPG  →  trip_002.jpg
2 file sẽ đổi tên, 5 file giữ nguyên.
Đây là bản xem trước. Thêm --apply để thực hiện.
```
JSON: `{ "ok": true, "data": <Plan> }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `RENAME_MODE_REQUIRED` | không chọn chế độ nào | 2 | Dùng --find/--replace, --case hoặc --number |
| `RENAME_MODE_CONFLICT` | chọn nhiều chế độ | 2 | |
| `REGEX_INVALID` | regex lỗi | 2 | Go dùng RE2: không hỗ trợ lookbehind (?<=) |
| `RENAME_PATTERN_INVALID` | `--number` thiếu `{n}` | 2 | Ví dụ: --number "Trip_{n:3}" |
| `RENAME_CONFLICT` | có xung đột khi `--apply` | 1 | Xem mục conflicts, sửa mẫu rồi chạy lại |
| `RENAME_FAILED` | lỗi giữa chừng (đã hoàn tác) | 1 | File có thể đang mở trong chương trình khác, thử foxkit lock |

## 8. Manifest
```json
{ "name": "rename", "title": "Đổi tên hàng loạt", "description": "Tìm-thay thế, đổi kiểu chữ, đánh số tên file", "category": "life",
  "args": [ { "name": "dir", "type": "dir", "required": true, "label": "Thư mục" } ],
  "flags": [
    { "name": "find", "type": "string", "label": "Tìm" },
    { "name": "replace", "type": "string", "label": "Thay bằng" },
    { "name": "regex", "type": "bool", "default": false, "label": "Regex" },
    { "name": "ignore-case", "type": "bool", "default": false, "label": "Không phân biệt hoa thường" },
    { "name": "with-ext", "type": "bool", "default": false, "label": "Áp dụng cả đuôi file" },
    { "name": "case", "type": "enum", "options": ["lower","upper","slug","snake","kebab"], "label": "Đổi kiểu chữ" },
    { "name": "number", "type": "string", "label": "Mẫu đánh số ({n:3})" },
    { "name": "sort", "type": "enum", "options": ["name","date","size"], "default": "name", "label": "Sắp xếp khi đánh số" },
    { "name": "start", "type": "int", "default": 1, "label": "Số bắt đầu" },
    { "name": "ext", "type": "list", "label": "Chỉ đuôi file" },
    { "name": "recursive", "type": "bool", "default": false, "label": "Gồm thư mục con" },
    { "name": "dirs", "type": "bool", "default": false, "label": "Đổi tên cả thư mục" },
    { "name": "apply", "type": "bool", "default": false, "label": "Thực hiện" }
  ], "admin": false, "longRunning": false, "dryRun": true, "output": "table" }
```

## 9. Test
- Hoán đổi `a.txt↔b.txt` qua 2 lần find/replace phức tạp → Apply thành công.
- `a.txt → A.txt` (`--case upper` với `--with-ext` tắt → `A.txt`).
- Hai file ra cùng tên → conflict, Apply không đổi gì.
- Tên đích `CON.txt`, `a?.txt` → conflict "tên không hợp lệ".
- Regex `(\d+)` → `photo_$1`.
- Đánh số tự nhiên: `img2, img10, img1` → thứ tự `img1, img2, img10`.
- Mô phỏng lỗi pha 2 (inject hàm rename giả trả lỗi) → mọi file trở về tên gốc.

Chạy thử:
```powershell
.\foxkit rename D:\Photos\Trip --find IMG_ --replace trip_
.\foxkit rename D:\Photos\Trip --number "DaLat_{n:3}" --sort date
.\foxkit rename D:\Photos\Trip --number "DaLat_{n:3}" --sort date --apply
```

## 10. Tiêu chí hoàn thành
Tiêu chí chung + test hoàn tác + không bao giờ ghi đè file đang có.

## 11. Ngoài phạm vi
Undo sau khi đã chạy xong (lưu lịch sử), đổi tên theo EXIF ngày chụp, đổi tên theo nội dung.

## 12. Go học được
`regexp` (RE2 khác PCRE của PHP `preg_replace`), `ReplaceAllString` với `$1`, tách tên/đuôi bằng `filepath.Ext`, inject hàm (`var renameFn = os.Rename`) để test lỗi (≈ mock trong PHPUnit), thiết kế "plan rồi apply".
