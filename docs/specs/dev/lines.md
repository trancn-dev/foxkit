# Spec: `foxkit lines`

> Nhóm: **dev** · Ngoài roadmap gốc (đề xuất từ `docs/apps/`) · Trạng thái: **chưa làm, chờ duyệt** · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Thao tác theo dòng trên văn bản copy: sắp xếp, bỏ trùng, đếm, lọc, thêm tiền/hậu tố. Thay cho `sort | uniq` mà không cần mở shell.

## 2. Cú pháp
```
foxkit lines <thao tác> [--clip | --file F] [tùy chọn] [--copy] [--json]
```
| Thao tác | Việc | Tùy chọn |
|---|---|---|
| `sort` | sắp xếp | `--desc`, `--num` (theo số), `--ci` (không phân biệt hoa/thường) |
| `uniq` | bỏ dòng trùng, giữ thứ tự xuất hiện đầu | `--ci`, `--count` (kèm số lần) |
| `rev` | đảo thứ tự dòng | — |
| `trim` | cắt khoảng trắng đầu/cuối, bỏ dòng rỗng | `--keep-empty` |
| `wrap` | thêm tiền/hậu tố mỗi dòng | `--prefix`, `--suffix` |
| `grep` | giữ dòng khớp (chuỗi con; RE2 nếu `--re`) | `--re`, `--invert` |
| `count` | đếm dòng, dòng không rỗng, dòng duy nhất | — |

Nguồn đầu vào như `table`: `--clip`, `--file`, hoặc stdin. Thao tác sai → `LINES_OP_INVALID` (exit 2). `--re` sai cú pháp → `LINES_REGEX_INVALID` (exit 2, hint nhắc Go dùng RE2, không có lookahead).

## 3. Cách hoạt động
- Chuẩn hóa `\r\n` về `\n` (dùng `clip.NormalizeCRLF`); kết quả cuối dòng có đúng một `\n`.
- `sort` ổn định (`sort.SliceStable`); `--num` dòng không phải số xếp cuối.
- `count` trả cả text và JSON `{ "lines": N, "nonEmpty": N, "unique": N }`.
- JSON của thao tác khác: `{ "items": [...] }`, rỗng là `[]`.

## 4. Test (dạng bảng)
Mỗi thao tác một nhóm: dòng CRLF, trùng khác hoa/thường, số lẫn chữ, đầu vào rỗng, regex sai.

Chạy thử: `Get-Content a.txt | .\foxkit lines uniq --count`.

## 5. Ngoài phạm vi
Chỉnh sửa tại chỗ file, thay thế regex (dùng `rename`/`find`), cột/trường (dùng `table`/`csv`).
