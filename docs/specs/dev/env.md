# Spec: `foxkit env`

> Nhóm: **dev** · Giai đoạn: GĐ1 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Kiểm tra `.env` so với `.env.example` (thiếu biến nào, thừa biến nào, biến nào rỗng) và đọc nhanh một key. Hay dùng sau khi `git pull` có người thêm biến mới.

## 2. Cú pháp
```
foxkit env diff [dir] [--file .env] [--example .env.example] [--json]
foxkit env get <KEY> [dir] [--file .env] [--show] [--json]
```
| Tham số/cờ | Mặc định | Ghi chú |
|---|---|---|
| `[dir]` | thư mục hiện tại | |
| `--file` | `.env` | đường dẫn tương đối với dir |
| `--example` | `.env.example` | |
| `--show` | false | `get` in giá trị thật; mặc định che nếu key có vẻ bí mật |

## 3. File
```
cmd/env.go             # lệnh cha "env" (chỉ nhóm)
cmd/env_diff.go
cmd/env_get.go
internal/env/parse.go
internal/env/diff.go
internal/env/parse_test.go
```
`internal/env` sẽ được `lara key` (GĐ5) dùng lại → API phải hỗ trợ **ghi lại file giữ nguyên format** (xem 4).

## 4. Hợp đồng
```go
package env

type Line struct {
    Raw     string // dòng gốc, giữ nguyên khi ghi lại
    Key     string // rỗng nếu là comment/dòng trống
    Value   string // đã bỏ nháy, đã xử lý escape trong nháy kép
    Line    int    // số dòng (1-based)
}
type File struct { Lines []Line }

func Parse(r io.Reader) (*File, error)
func ParseFile(path string) (*File, error)
func (f *File) Get(key string) (string, bool)
func (f *File) Keys() []string                  // theo thứ tự xuất hiện
func (f *File) Set(key, value string)           // sửa dòng cũ hoặc thêm cuối file (dùng cho GĐ5)
func (f *File) WriteTo(w io.Writer) (int64, error)

type Diff struct {
    Missing []string `json:"missing"` // có ở example, không có ở .env
    Extra   []string `json:"extra"`   // có ở .env, không có ở example
    Empty   []string `json:"empty"`   // có ở .env nhưng rỗng, trong khi example có giá trị
}
func Compare(env, example *File) Diff
func IsSecretKey(key string) bool // chứa KEY, SECRET, PASSWORD, PASS, TOKEN, PRIVATE (không phân biệt hoa thường)
```

## 5. Cách hoạt động
**Parse từng dòng** (`bufio.Scanner`):
1. `TrimSpace`; rỗng hoặc bắt đầu `#` → dòng không có key.
2. Bỏ tiền tố `export ` nếu có.
3. Tách tại dấu `=` **đầu tiên**. Key phải khớp `^[A-Za-z_][A-Za-z0-9_.]*$`, không khớp thì báo lỗi kèm số dòng (`ENV_PARSE_FAILED`).
4. Value:
   - Bắt đầu `"`: đọc tới `"` đóng không bị escape; xử lý `\n`, `\"`, `\\`. Hỗ trợ giá trị nhiều dòng trong nháy kép (đọc tiếp dòng sau tới khi gặp nháy đóng).
   - Bắt đầu `'`: lấy nguyên văn tới `'` đóng.
   - Không nháy: cắt comment ` #` (dấu cách + `#`) phía sau, rồi `TrimSpace`.
5. Không expand `${VAR}` (giữ nguyên chuỗi); ghi chú điều này trong output text của `get` nếu value chứa `${`.
6. Key trùng: lấy giá trị **cuối cùng** (giống phpdotenv), đưa vào cảnh báo `duplicates`.

**diff**: dùng `map[string]bool` cho tập key; kết quả sắp xếp theo thứ tự trong example (Missing, Empty) và trong .env (Extra). Slice rỗng phải là `[]`.

**get**: không tìm thấy key → exit 1 `ENV_KEY_NOT_FOUND`. Key bí mật và không `--show` → value hiển thị `abc***` (3 ký tự đầu) hoặc `***` nếu ngắn hơn 8 ký tự.

## 6. Output
`diff` text:
```
.env so với .env.example
Thiếu (2):  MAIL_MAILER, REDIS_HOST
Thừa  (1):  OLD_API_KEY
Rỗng  (1):  APP_KEY
```
Không có khác biệt: `.env khớp với .env.example ✓`.

`diff` JSON:
```json
{ "ok": true, "data": { "file": ".env", "example": ".env.example",
  "missing": ["MAIL_MAILER","REDIS_HOST"], "extra": ["OLD_API_KEY"], "empty": ["APP_KEY"], "duplicates": [] } }
```
`get` JSON: `{ "ok": true, "data": { "key": "DB_HOST", "value": "127.0.0.1", "masked": false, "line": 12 } }`

Exit code của `diff`: luôn 0 nếu đọc được file (có khác biệt vẫn là kết quả hợp lệ).

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `ENV_FILE_NOT_FOUND` | thiếu .env | 1 | Tạo bằng: copy .env.example .env |
| `ENV_EXAMPLE_NOT_FOUND` | thiếu example | 1 | Dùng --example để chỉ file mẫu khác |
| `ENV_PARSE_FAILED` | dòng sai cú pháp | 1 | Kiểm tra dòng N trong <file> |
| `ENV_KEY_NOT_FOUND` | get không thấy | 1 | Xem danh sách key bằng foxkit env diff |

## 8. Manifest
```json
{ "name": "env diff", "title": "So sánh .env", "description": "So .env với .env.example: thiếu, thừa, rỗng", "category": "dev",
  "args": [ { "name": "dir", "type": "dir", "required": false, "label": "Thư mục dự án" } ],
  "flags": [
    { "name": "file", "type": "string", "default": ".env", "label": "File env" },
    { "name": "example", "type": "string", "default": ".env.example", "label": "File mẫu" }
  ], "admin": false, "longRunning": false, "dryRun": false, "output": "kv" },
{ "name": "env get", "title": "Đọc biến .env", "description": "Đọc giá trị một key trong .env", "category": "dev",
  "args": [
    { "name": "key", "type": "string", "required": true, "label": "Key" },
    { "name": "dir", "type": "dir", "required": false, "label": "Thư mục dự án" }
  ],
  "flags": [
    { "name": "file", "type": "string", "default": ".env", "label": "File env" },
    { "name": "show", "type": "bool", "default": false, "label": "Hiện giá trị bí mật" }
  ], "admin": false, "longRunning": false, "dryRun": false, "output": "kv" }
```

## 9. Test
Parser (bảng):
| dòng | key | value |
|---|---|---|
| `APP_NAME=Laravel` | APP_NAME | `Laravel` |
| `APP_NAME="My App"` | APP_NAME | `My App` |
| `DB_PASSWORD='p@ss#1'` | DB_PASSWORD | `p@ss#1` |
| `MAIL_FROM=a@b.com # comment` | MAIL_FROM | `a@b.com` |
| `export FOO=bar` | FOO | `bar` |
| `EMPTY=` | EMPTY | `` |
| `URL=${APP_URL}/api` | URL | `${APP_URL}/api` |
| `# comment` | (không key) | |
| `KEY="dòng1\ndòng2"` | KEY | 2 dòng |

Round-trip: `Parse` rồi `WriteTo` không đổi byte nào (kể cả CRLF). `Set` key đã có chỉ sửa đúng dòng đó.
Diff: dùng `.env.example` mặc định của Laravel 11 làm fixture trong `testdata/`.

Chạy thử (trong một dự án Laravel):
```powershell
.\foxkit env diff
.\foxkit env diff D:\Projects\shop-api --json
.\foxkit env get DB_HOST
.\foxkit env get DB_PASSWORD           # bị che
.\foxkit env get DB_PASSWORD --show
```

## 10. Tiêu chí hoàn thành
Tiêu chí chung + test parser/round-trip + chạy đúng trên dự án Laravel thật.

## 11. Ngoài phạm vi
`env set` từ CLI (để `lara key` dùng nội bộ trước), expand biến, mã hóa `.env` (`env:encrypt`).

## 12. Go học được
`bufio.Scanner`, viết parser bằng tay (state đơn giản), `map` làm tập hợp (≈ `array_diff_key`), lệnh con cobra, thư mục `testdata/` (Go tự bỏ qua khi build), giữ nguyên dữ liệu gốc để ghi lại không làm hỏng file.
