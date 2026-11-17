# Spec: `foxkit csv`

> Nhóm: **dev** · Giai đoạn: GĐ7 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Xem file CSV dạng bảng ngay trong terminal, thống kê nhanh, lọc cột/dòng, đổi CSV ↔ JSON. Xử lý đúng file CSV xuất từ Excel Việt Nam (BOM, dấu `;`, tiếng Việt).

## 2. Cú pháp
```
foxkit csv <file|-> [--delim auto|,|;|tab|"|"] [--no-header] [--cols a,b,3] [--where "col=value"] [--head 20] [--tail N]
                    [--stats] [--to json|ndjson|csv|tsv] [--out <file>] [--json]
```
| Cờ | Mặc định | Ghi chú |
|---|---|---|
| `--delim` | `auto` | đoán từ dòng đầu: đếm `, ; \t \|` ngoài dấu nháy, chọn dấu xuất hiện đều nhất ở 5 dòng đầu |
| `--no-header` | false | dòng đầu là dữ liệu; cột đặt tên `c1,c2,...` |
| `--cols` | tất cả | chọn theo tên hoặc số thứ tự (1-based), giữ thứ tự nhập |
| `--where` | | lọc `col=value`, `col!=v`, `col~sub` (chứa), `col>10`, `col<10`; nhiều điều kiện cách nhau `,` là AND |
| `--head`/`--tail` | `--head 20` | 0 = tất cả |
| `--stats` | false | mỗi cột: số dòng, rỗng, duy nhất, kiểu đoán (int/float/date/text), min/max (cho số/ngày) |
| `--to` | | xuất sang định dạng khác thay vì in bảng |
| `--out` | stdout | ghi ra file (không ghi đè nếu tồn tại trừ khi `--force`) |

## 3. File
```
cmd/csv.go
internal/csvx/read.go        # đọc, đoán delimiter, xử lý BOM
internal/csvx/query.go       # cols, where
internal/csvx/stats.go
internal/csvx/csvx_test.go
```

## 4. Hợp đồng
```go
package csvx
type Table struct { Header []string; Rows [][]string; Delim rune; HadBOM bool }
func Read(r io.Reader, delim rune, noHeader bool) (*Table, error)
func DetectDelim(sample []byte) rune
func (t *Table) Select(cols []string) (*Table, error)
func (t *Table) Where(expr string) (*Table, error)
type ColStat struct { Name string; Count, Empty, Unique int; Kind string; Min, Max string }
func (t *Table) Stats() []ColStat
```

## 5. Cách hoạt động
1. Bỏ BOM UTF-8 đầu file (ghi `HadBOM`). Nếu byte lạ không phải UTF-8 hợp lệ (file Excel lưu ANSI/Windows-1258) → `CSV_ENCODING` với gợi ý lưu lại bằng "CSV UTF-8" (không tự chuyển mã để tránh phải thêm dependency).
2. `csv.NewReader` với `Comma = delim`, `LazyQuotes = true`, `FieldsPerRecord = -1` (chấp nhận dòng thiếu/thừa cột; dòng thiếu được đệm chuỗi rỗng và đếm vào `warnings`).
3. File lớn: đọc streaming; `--head` dừng đọc sớm khi không cần `--stats`/`--where`/`--tail`/`--to`. Giới hạn nạp RAM 200MB khi cần toàn bộ, vượt thì `INPUT_TOO_LARGE`.
4. `Where`: tách điều kiện, so sánh số nếu cả hai vế parse được số (`strconv.ParseFloat`), không thì so chuỗi không phân biệt hoa thường. Cột không tồn tại → `CSV_COL_NOT_FOUND` kèm danh sách cột.
5. `Stats`: đoán kiểu theo ≥ 95% giá trị không rỗng; `Unique` bằng `map[string]struct{}` giới hạn 100.000 phần tử (vượt thì ghi `">100000"`).
6. `--to json`: mảng object theo header (giá trị luôn là chuỗi, trừ khi thêm `--typed` để số/bool thành kiểu thật); `ndjson` mỗi dòng một object; `csv`/`tsv` đổi delimiter và luôn ghi LF.
7. In bảng: cắt ô rộng quá 40 ký tự bằng `…`, căn theo độ rộng **rune** (tiếng Việt), không theo byte.

## 6. Output
```
id  name              email                  city
1   Nguyễn Văn Hùng   hung@example.com       Hà Nội
2   Trần Thị Lan      lan@example.org        TP.HCM
… 1.998 dòng nữa (dùng --head 0 để xem hết)
```
`--stats`:
```
CỘT    KIỂU   DÒNG  RỖNG  DUY NHẤT  MIN         MAX
id     int    2000  0     2000      1           2000
city   text   2000  14    12
```
JSON: `{ "ok": true, "data": { "header": [...], "rows": [[...]], "total": 2000, "delim": ",", "bom": true, "warnings": [] } }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `CSV_NO_INPUT` | | 2 | Ví dụ: foxkit csv users.csv |
| `CSV_ENCODING` | không phải UTF-8 | 1 | Mở bằng Excel → Save As → "CSV UTF-8 (Comma delimited)" |
| `CSV_COL_NOT_FOUND` | | 2 | Các cột có: ... |
| `CSV_WHERE_INVALID` | | 2 | Ví dụ: --where "city=Hà Nội,age>18" |
| `FILE_EXISTS` | `--out` trùng file | 1 | Thêm --force để ghi đè |

## 8. Manifest
```json
{ "name": "csv", "title": "CSV", "description": "Xem, lọc, thống kê, đổi CSV ↔ JSON", "category": "dev",
  "args": [ { "name": "file", "type": "file", "required": true, "label": "File CSV" } ],
  "flags": [
    { "name": "delim", "type": "string", "default": "auto", "label": "Dấu phân cách" },
    { "name": "no-header", "type": "bool", "default": false, "label": "Không có dòng tiêu đề" },
    { "name": "cols", "type": "list", "label": "Cột" },
    { "name": "where", "type": "string", "label": "Điều kiện lọc" },
    { "name": "head", "type": "int", "default": 20, "min": 0, "label": "Số dòng đầu" },
    { "name": "stats", "type": "bool", "default": false, "label": "Thống kê cột" },
    { "name": "to", "type": "enum", "options": ["json","ndjson","csv","tsv"], "label": "Xuất sang" },
    { "name": "out", "type": "string", "label": "Ghi ra file" }
  ], "admin": false, "longRunning": false, "dryRun": false, "output": "table" }
```

## 9. Test
- `DetectDelim` bảng: `a,b,c` → `,`; `a;b;c` → `;`; `"a;b",c,d` → `,` (dấu trong nháy bị bỏ qua).
- BOM + CRLF + ô có xuống dòng trong nháy.
- Dòng thiếu cột → đệm, có warning.
- `--where "age>18,city=Hà Nội"`; `--cols 2,name`.
- Tiếng Việt: căn cột không lệch (so độ rộng rune).
- `--to json` giữ đúng ký tự có dấu, không escape thành `\uXXXX`.

Chạy thử: `.\foxkit csv users.csv`, `.\foxkit csv users.csv --where "city=Hà Nội" --cols name,email`, `.\foxkit csv export.csv --to json --out export.json`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + test + mở file CSV Excel tiếng Việt xuất thật không bị lỗi font.

## 11. Ngoài phạm vi
Sửa/ghi từng ô, join nhiều file, chuyển mã Windows-1258, xlsx.

## 12. Go học được
`encoding/csv` (và các cờ `LazyQuotes`, `FieldsPerRecord`), đọc streaming + dừng sớm, `strconv.ParseFloat`, độ rộng chuỗi theo rune (`utf8.RuneCountInString`), parser điều kiện nhỏ.
