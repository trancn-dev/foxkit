# Spec: `foxkit table`

> Nhóm: **dev** · Ngoài roadmap gốc (đề xuất từ `docs/apps/`) · Trạng thái: **chưa làm, chờ duyệt** · Cần trước: [core/foundation.md](../core/foundation.md), [csv.md](csv.md)

## 1. Mục đích
Đổi bảng copy từ Excel/Google Sheets (TSV) sang Markdown, CSV hoặc JSON, và ngược lại. Dùng khi dán bảng vào README, PR, tài liệu.

## 2. Cú pháp
```
foxkit table [--clip | --file F] [--from tsv|csv|md] [--to md|csv|tsv|json] [--no-header] [--copy] [--json]
```
| Cờ | Mặc định | Ràng buộc |
|---|---|---|
| `--clip` | tự chọn | đọc clipboard (nhiều dòng) |
| `--file` | — | đọc file; không có `--clip`/`--file` thì đọc stdin |
| `--from` | tự dò (có tab → tsv, có `\|` đầu dòng → md, còn lại csv) | khác danh sách → `TABLE_FORMAT_INVALID` (exit 2) |
| `--to` | `md` | như trên |
| `--no-header` | false | dòng đầu là dữ liệu; tiêu đề thành `col1..colN` |
| `--copy` | false | ghi kết quả vào clipboard |

## 3. Cách hoạt động
- Tách dòng/ô bằng `encoding/csv` (dấu phân tách `\t` hoặc `,`), dùng lại `internal/csvx` cho BOM/dấu phân tách. Markdown tự phân tích `|`.
- Ô có `|` khi ra Markdown thì thoát thành `\|`; xuống dòng trong ô đổi thành `<br>`.
- Căn cột Markdown theo độ rộng chữ (đếm rune, không đếm byte, để tiếng Việt thẳng cột).
- Số cột không đều: điền ô trống tới số cột lớn nhất.
- JSON ra là mảng object theo tiêu đề; slice rỗng là `[]`.

## 4. Lỗi
| code | Khi nào | exit |
|---|---|---|
| `TABLE_FORMAT_INVALID` | `--from`/`--to` sai | 2 |
| `TABLE_EMPTY` | không có dữ liệu | 1 |
| `CLIP_READ_FAILED` / `CLIP_WRITE_FAILED` | clipboard lỗi | 1 |

## 5. Test (dạng bảng)
TSV → md có tiếng Việt, ô chứa `|`, dòng thiếu cột, `--no-header`, md → csv, csv → json, đầu vào rỗng.

Chạy thử (Windows): copy vài ô từ Excel rồi `.\foxkit table --clip --to md`.

## 6. Ngoài phạm vi
Đọc/ghi `.xlsx` (cần dependency, xem `docs/apps/excel-sheets.md`), sắp xếp/lọc cột, công thức.
