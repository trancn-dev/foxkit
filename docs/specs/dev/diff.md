# Spec: `foxkit diff`

> Nhóm: **dev** · Giai đoạn: GĐ7 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md), [life/size.md](../life/size.md) (`fsutil`)

## 1. Mục đích
So sánh hai file (hiện khác biệt theo dòng) hoặc hai thư mục (file nào thêm/xóa/đổi). Dùng khi so `.env`, file config giữa môi trường, hai bản build, hai bản deploy.

## 2. Cú pháp
```
foxkit diff <a> <b> [--context 3] [--ignore-space] [--ignore-eol] [--ignore-case] [--names-only] [--exclude pat,...] [--json]
```
`<a>`, `<b>` cùng là file hoặc cùng là thư mục; khác loại → `DIFF_TYPE_MISMATCH`.

| Cờ | Ghi chú |
|---|---|
| `--context` | số dòng ngữ cảnh quanh chỗ khác (mặc định 3) |
| `--ignore-space` | bỏ qua khác biệt khoảng trắng đầu/cuối dòng và gộp khoảng trắng liên tiếp |
| `--ignore-eol` | coi `\r\n` = `\n` (mặc định bật vì file Windows/Linux lẫn lộn; tắt bằng `--ignore-eol=false`) |
| `--names-only` | thư mục: chỉ liệt kê tên file khác, không diff nội dung |
| `--exclude` | glob tên bỏ qua; mặc định thêm `fsutil.DefaultSkipDirs` |

## 3. File
```
cmd/diff.go
internal/diff/lines.go        # thuật toán diff dòng
internal/diff/dir.go
internal/diff/render.go       # unified format
internal/diff/diff_test.go
```

## 4. Hợp đồng
```go
package diff
type Op string // "equal","add","del"
type Line struct { Op Op `json:"op"`; A int `json:"a,omitempty"`; B int `json:"b,omitempty"`; Text string `json:"text"` }
type Hunk struct { AStart, ALen, BStart, BLen int; Lines []Line }
type FileDiff struct { Path string; Status string /* same|changed|added|removed|binary */; Added, Removed int; Hunks []Hunk }
type Result struct { A, B string; Identical bool; Files []FileDiff; Added, Removed, Changed int }
func Lines(a, b []string, opt Options) []Line   // dãy op đầy đủ
func ToHunks(ops []Line, context int) []Hunk
func Dirs(a, b string, opt Options) (Result, error)
```

## 5. Cách hoạt động
1. **Diff dòng** bằng thuật toán Myers (O(ND)), tự cài đặt ~60 dòng (không thêm dependency). Với file > 20.000 dòng thì rút gọn bằng cách cắt tiền tố/hậu tố giống nhau trước.
2. Chuẩn hóa trước khi so sánh theo các cờ `--ignore-*`, nhưng **in ra dòng gốc**.
3. Nhận diện nhị phân (có byte `0x00` trong 8KB đầu) → `Status: binary`, so bằng hash sha256, không in nội dung.
4. File > 10MB → chỉ so hash, báo `changed`/`same`.
5. **Thư mục**: duyệt cả hai cây, ghép theo đường dẫn tương đối (so sánh không phân biệt hoa thường trên Windows). Chỉ ở A → `removed`; chỉ ở B → `added`; ở cả hai: cùng size + mtime thì thử hash, khác thì diff dòng (trừ `--names-only`).
6. Kết quả `Identical` khi không có khác biệt nào; **exit 0 cả khi có khác biệt** (là kết quả hợp lệ), thêm `--exit-code` để trả 1 khi khác (dùng trong script).
7. Màu (khi terminal): dòng thêm xanh `+`, xóa đỏ `-`, tiêu đề hunk xanh lơ `@@ -a,b +c,d @@`.

## 6. Output
```
--- a/.env
+++ b/.env.production
@@ -3,5 +3,5 @@
 APP_ENV=local
-APP_DEBUG=true
+APP_DEBUG=false
 APP_URL=http://localhost
```
Thư mục:
```
M  config/app.php        (+2 −1)
A  config/horizon.php
D  config/old.php
B  public/logo.png       (nhị phân, khác)
Tổng: 1 sửa · 1 thêm · 1 xóa · 1 nhị phân
```
JSON: `{ "ok": true, "data": <Result> }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `DIFF_NOT_FOUND` | a hoặc b không tồn tại | 2 | Kiểm tra lại đường dẫn |
| `DIFF_TYPE_MISMATCH` | file vs thư mục | 2 | Hai đối số phải cùng là file hoặc cùng là thư mục |
| `DIFF_DIFFERENT` | `--exit-code` và có khác biệt | 1 | |

## 8. Manifest
```json
{ "name": "diff", "title": "So sánh", "description": "So sánh hai file hoặc hai thư mục", "category": "dev",
  "args": [
    { "name": "a", "type": "file", "allowDir": true, "required": true, "label": "Bên trái" },
    { "name": "b", "type": "file", "allowDir": true, "required": true, "label": "Bên phải" }
  ],
  "flags": [
    { "name": "context", "type": "int", "default": 3, "min": 0, "max": 100, "label": "Dòng ngữ cảnh" },
    { "name": "ignore-space", "type": "bool", "default": false, "label": "Bỏ qua khoảng trắng" },
    { "name": "ignore-eol", "type": "bool", "default": true, "label": "Bỏ qua CRLF/LF" },
    { "name": "names-only", "type": "bool", "default": false, "label": "Chỉ liệt kê tên" },
    { "name": "exit-code", "type": "bool", "default": false, "label": "Exit 1 nếu khác" }
  ], "admin": false, "longRunning": false, "dryRun": false, "output": "text" }
```

## 9. Test
- Bảng cặp (a, b) → dãy op mong đợi: thêm cuối, xóa đầu, đổi giữa, hai file giống, một file rỗng.
- Kết quả diff **áp lại** lên a phải ra đúng b (property test với dữ liệu ngẫu nhiên).
- `--ignore-eol`: `a\r\nb` = `a\nb`.
- Hunk gộp khi hai chỗ khác cách nhau ≤ 2×context.
- Thư mục: tạo hai cây tạm có added/removed/changed/binary.

Chạy thử: `.\foxkit diff .env .env.production`, `.\foxkit diff D:\deploy\v1 D:\deploy\v2 --names-only`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + property test áp-lại + diff hai file 50.000 dòng dưới 1 giây.

## 11. Ngoài phạm vi
Diff theo từ/ký tự, merge 3 chiều, áp patch, so sánh ảnh.

## 12. Go học được
Tự cài thuật toán (Myers) bằng slice và vòng lặp, kiểu `string`-enum (`type Op string`), property test (so kết quả với một "oracle" đơn giản), chuẩn hóa dữ liệu trước khi so sánh nhưng giữ bản gốc để in.
