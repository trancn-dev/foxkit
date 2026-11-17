# Spec: `foxkit path` (đọc & kiểm tra)

> Nhóm: **dev** · Giai đoạn: GĐ3 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)
> Phần sửa PATH (`path add`, `path rm`) ở GĐ6: [dev/path-edit.md](path-edit.md).

## 1. Mục đích
Xem PATH thành danh sách dễ đọc và phát hiện vấn đề: thư mục không tồn tại, mục trùng, mục rỗng, PATH quá dài.

## 2. Cú pháp
```
foxkit path [--json]          # liệt kê
foxkit path check [--json]    # chỉ in mục có vấn đề
```

## 3. File
```
cmd/path.go          # lệnh "path" (vừa là lệnh lá liệt kê, vừa là cha của check/add/rm)
cmd/path_check.go
internal/winpath/path.go
internal/winpath/path_test.go
```
Lưu ý manifest: vì `path` vừa chạy được vừa có lệnh con, test chống lệch phải coi `path` là lệnh lá **nếu nó có `RunE`**. Cập nhật quy tắc ở foundation §10: "lệnh lá = lệnh có `Run`/`RunE`".

## 4. Hợp đồng
```go
package winpath
type Entry struct {
    Index     int    `json:"index"`
    Dir       string `json:"dir"`        // giá trị gốc
    Expanded  string `json:"expanded"`   // đã expand %VAR%
    Exists    bool   `json:"exists"`
    Duplicate bool   `json:"duplicate"`  // trùng với mục đứng trước (so sau khi expand, không phân biệt hoa thường, bỏ "\" cuối)
    Empty     bool   `json:"empty"`
}
type Report struct {
    Entries  []Entry `json:"entries"`
    Length   int     `json:"length"`    // độ dài chuỗi PATH
    Problems int     `json:"problems"`
}
func Analyze(pathEnv string) Report
```

## 5. Cách hoạt động
1. `os.Getenv("PATH")` (đây là PATH **đã gộp** User + System của process hiện tại).
2. Tách theo `;` **thủ công** (không dùng `filepath.SplitList` vì cần giữ cả mục rỗng để báo).
3. Expand `%VAR%` bằng hàm tự viết (regex `%([^%]+)%` → `os.Getenv`).
4. `os.Stat` → `Exists` (lỗi khác "không tồn tại", như thiếu quyền, coi là tồn tại).
5. So trùng bằng chuẩn hóa: `strings.ToLower(strings.TrimRight(filepath.Clean(x), `\`))`.
6. Cảnh báo `Length > 2047` (một số công cụ cũ cắt PATH dài).
7. `check`: exit 0 nếu không vấn đề; có vấn đề vẫn exit 0 (là báo cáo), `problems > 0`.

## 6. Output
```
 1  ✓  C:\Windows\system32
 2  ✓  C:\laragon\bin\php\php-8.3.12
 3  ✗  C:\Program Files\OldTool\bin            (không tồn tại)
 4  ⚠  C:\Windows\System32\                    (trùng mục 1)
...
PATH dài 1.842 ký tự · 2 vấn đề
Muốn sửa vĩnh viễn: foxkit path rm <dir> (GĐ6) hoặc Settings → Environment Variables.
```
JSON: `{ "ok": true, "data": <Report> }`

## 7. Lỗi
Không có lỗi riêng (PATH rỗng vẫn trả report rỗng).

## 8. Manifest
```json
{ "name": "path", "title": "Xem PATH", "description": "Liệt kê PATH dạng danh sách", "category": "dev",
  "args": [], "flags": [], "admin": false, "longRunning": false, "dryRun": false, "output": "table" },
{ "name": "path check", "title": "Kiểm tra PATH", "description": "Tìm mục PATH không tồn tại, trùng, rỗng", "category": "dev",
  "args": [], "flags": [], "admin": false, "longRunning": false, "dryRun": false, "output": "table" }
```

## 9. Test
`Analyze("C:\\a;;C:\\A\\;%TMPX%\\b")` với `TMPX` đặt bằng `t.Setenv` → mục 2 empty, mục 3 duplicate của mục 1, mục 4 expanded đúng.

Chạy thử: `.\foxkit path`, `.\foxkit path check --json`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + test + quy tắc lệnh lá trong manifest test đã cập nhật.

## 11. Ngoài phạm vi (ở GĐ này)
Tách User/System (cần registry), sửa PATH.

## 12. Go học được
`t.Setenv` trong test, chuẩn hóa đường dẫn Windows, cobra: lệnh vừa có `RunE` vừa có lệnh con.
