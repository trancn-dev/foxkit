# Spec: `foxkit todo`

> Nhóm: **dev** · Giai đoạn: GĐ2 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Liệt kê các ghi chú `TODO`, `FIXME`, `HACK`, `XXX` trong mã nguồn dự án để không bị quên.

## 2. Cú pháp
```
foxkit todo [dir] [--tags TODO,FIXME,HACK,XXX] [--ext php,vue,js,ts,go,blade.php] [--author] [--json]
```
| Cờ | Mặc định | Ghi chú |
|---|---|---|
| `--tags` | `TODO,FIXME,HACK,XXX` | so khớp **đúng chữ hoa** như quy ước |
| `--ext` | `php,js,ts,jsx,tsx,vue,go,css,scss,blade.php,py,sql,md` | khớp hậu tố tên file |
| `--author` | false | thêm tác giả từ `git blame` (chậm hơn) |

## 3. File
```
cmd/todo.go
internal/todo/todo.go
internal/todo/todo_test.go
```

## 4. Hợp đồng
```go
package todo
type Item struct {
    File   string `json:"file"`             // đường dẫn tương đối so với dir, dùng "/"
    Line   int    `json:"line"`
    Tag    string `json:"tag"`
    Text   string `json:"text"`             // phần sau tag, đã trim, tối đa 200 ký tự
    Author string `json:"author,omitempty"`
}
type Report struct {
    Dir    string         `json:"dir"`
    Items  []Item         `json:"items"`    // sắp theo file rồi line
    Counts map[string]int `json:"counts"`   // {"TODO":12,"FIXME":3}
    Files  int            `json:"files"`    // số file đã quét
}
```

## 5. Cách hoạt động
1. Duyệt `dir`, bỏ `fsutil.DefaultSkipDirs` + `public/build`, `public/vendor`, `bootstrap/cache`, `coverage`. Bỏ file `.min.js`, `.min.css`, file > 2MB.
2. Mỗi file: worker pool đọc bằng `bufio.Scanner`, đặt `scanner.Buffer(make([]byte, 64*1024), 1024*1024)` để dòng dài không làm lỗi `token too long`.
3. Regex biên dịch **một lần**: `\b(TODO|FIXME|HACK|XXX)\b[:(\s]?\s*(.*)` (thay danh sách theo `--tags`, dùng `regexp.QuoteMeta`).
   - Chỉ nhận khi tag nằm sau dấu comment trên cùng dòng: `//`, `#`, `/*`, `*`, `<!--`, `{{--`. Tránh bắt nhầm chuỗi như `$todoList` hay `'TODO'` trong code.
   - Bỏ phần đóng comment cuối dòng (`*/`, `-->`, `--}}`).
4. `--author`: sau khi có danh sách, chạy `git blame -L <line>,<line> --porcelain <file>` cho từng item (giới hạn 8 song song), lấy dòng `author `. Không phải repo git → bỏ qua im lặng.
5. Đếm theo tag.

## 6. Output
```
app/Http/Controllers/OrderController.php
  42  TODO   validate coupon trước khi tạo đơn
  88  FIXME  race condition khi 2 request cùng lúc
resources/js/components/Cart.vue
  15  HACK   tạm ẩn nút khi đang load

Tổng: TODO 12 · FIXME 3 · HACK 1 (trong 245 file)
```
JSON: `{ "ok": true, "data": <Report> }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `DIR_NOT_FOUND` | | 2 | |
| `TAGS_INVALID` | tag rỗng hoặc có ký tự lạ | 2 | Ví dụ: --tags TODO,FIXME |

## 8. Manifest
```json
{ "name": "todo", "title": "Liệt kê TODO", "description": "Tìm TODO/FIXME/HACK trong mã nguồn", "category": "dev",
  "args": [ { "name": "dir", "type": "dir", "required": false, "label": "Thư mục dự án" } ],
  "flags": [
    { "name": "tags", "type": "list", "default": "TODO,FIXME,HACK,XXX", "label": "Tag" },
    { "name": "ext", "type": "list", "label": "Đuôi file" },
    { "name": "author", "type": "bool", "default": false, "label": "Kèm tác giả (git blame)" }
  ], "admin": false, "longRunning": false, "dryRun": false, "output": "table" }
```

## 9. Test (bảng dòng)
| dòng | kỳ vọng |
|---|---|
| `// TODO: validate coupon` | TODO, `validate coupon` |
| `# FIXME(tran): lỗi múi giờ` | FIXME, `lỗi múi giờ` (bỏ `(tran):`) |
| `/* HACK tạm thời */` | HACK, `tạm thời` |
| `{{-- TODO đổi icon --}}` | TODO, `đổi icon` |
| `$todoList = [];` | không khớp |
| `echo 'TODO';` | không khớp |
| dòng 2MB không xuống dòng | không lỗi |

Chạy thử: `.\foxkit todo D:\Projects\shop`, `.\foxkit todo --tags FIXME --author --json`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + bảng test.

## 11. Ngoài phạm vi
Tạo issue GitHub từ TODO, theo dõi TODO mới theo commit.

## 12. Go học được
`regexp.MustCompile` ở cấp package (biên dịch 1 lần), `regexp.QuoteMeta`, `bufio.Scanner.Buffer`, `filepath.ToSlash`, kết hợp worker pool với lệnh ngoài (`git`).
