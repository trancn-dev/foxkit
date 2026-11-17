# Spec: `foxkit which`

> Nhóm: **dev** · Giai đoạn: GĐ3 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Liệt kê **mọi** bản của một lệnh trong PATH theo thứ tự ưu tiên, chỉ ra bản nào đang được dùng. Bắt lỗi kinh điển: máy có cả Laragon + XAMPP, gõ `php -v` ra sai phiên bản; có 2 bản `node`, `composer`, `git`.

## 2. Cú pháp
```
foxkit which <name>... [--version] [--json]
```
| Tham số/cờ | Ghi chú |
|---|---|
| `<name>...` | `php`, `node`, `composer`... có hoặc không đuôi |
| `--version` | chạy `<exe> --version` (hoặc `-v` cho php) với timeout 3s để hiện phiên bản từng bản |

## 3. File
```
cmd/which.go
internal/which/which.go
internal/which/which_test.go
```

## 4. Hợp đồng
```go
package which
type Match struct {
    Path    string `json:"path"`
    Dir     string `json:"dir"`
    Active  bool   `json:"active"`  // bản đầu tiên = bản được dùng
    Version string `json:"version,omitempty"`
}
type Result struct {
    Name    string  `json:"name"`
    Matches []Match `json:"matches"`
}
func Find(name string, pathEnv, pathExt string, cwd string) []Match
```

## 5. Cách hoạt động
1. Lấy `PATH` (`os.Getenv`), tách bằng `filepath.SplitList`; bỏ mục rỗng; expand `%VAR%` còn sót (hiếm) bằng `os.ExpandEnv` sau khi đổi `%X%` → `${X}`.
2. `PATHEXT` (mặc định `.COM;.EXE;.BAT;.CMD` nếu trống), so không phân biệt hoa thường.
3. Thứ tự tìm giống `cmd.exe`: **thư mục hiện tại trước** (cmd.exe tìm cwd trước; PowerShell thì không — ghi chú trong output text khi có bản ở cwd), rồi từng mục PATH.
4. Trong mỗi thư mục: nếu `name` đã có đuôi thuộc PATHEXT → kiểm tra đúng tên đó; không thì thử lần lượt từng đuôi. Tìm thấy → `os.Stat` xác nhận là file.
5. Khử trùng đường dẫn (cùng thư mục xuất hiện 2 lần trong PATH) bằng `strings.EqualFold` trên đường dẫn đã `filepath.Clean`.
6. Phần tử đầu tiên `Active = true`.
7. `--version`: chạy song song, lấy dòng đầu stdout+stderr, cắt 80 ký tự. File `.bat/.cmd` chạy qua `cmd /c`.
8. Không tìm thấy → `matches: []`, exit 1 với `WHICH_NOT_FOUND` (đây là thông tin "không có trong PATH" nên coi là lỗi để script dùng được `if`).

## 6. Output
```
php
  ✓ C:\laragon\bin\php\php-8.3.12\php.exe     PHP 8.3.12 (cli)
    C:\xampp\php\php.exe                       PHP 8.0.30 (cli)
Bản đang dùng là bản đầu tiên. Muốn đổi thì sắp xếp lại PATH (foxkit path).
```
JSON: `{ "ok": true, "data": { "items": [Result...] } }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `WHICH_NO_NAME` | | 2 | Ví dụ: foxkit which php |
| `WHICH_NOT_FOUND` | không bản nào | 1 | <name> chưa có trong PATH. Kiểm tra cài đặt hoặc thêm thư mục chứa nó vào PATH |

## 8. Manifest
```json
{ "name": "which", "title": "Tìm lệnh trong PATH", "description": "Liệt kê mọi bản exe trong PATH, bản nào đang được dùng", "category": "dev",
  "args": [ { "name": "name", "type": "string", "required": true, "multiple": true, "label": "Tên lệnh" } ],
  "flags": [ { "name": "version", "type": "bool", "default": false, "label": "Hiện phiên bản" } ],
  "admin": false, "longRunning": false, "dryRun": false, "output": "table" }
```

## 9. Test
`Find` nhận `pathEnv`, `pathExt`, `cwd` nên test bằng thư mục tạm: tạo `a/php.exe`, `b/php.exe`, `b/php.bat` → thứ tự đúng, `a` active; `php.EXE` vẫn khớp; PATH lặp `a;a` → không trùng.

Chạy thử: `.\foxkit which php node composer --version`, `.\foxkit which git --json`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + test.

## 11. Ngoài phạm vi
App alias của Windows Store (`%LOCALAPPDATA%\Microsoft\WindowsApps` là reparse point; chỉ liệt kê, không phân tích), alias/function của PowerShell.

## 12. Go học được
`filepath.SplitList`, `os.Getenv`, khác biệt `exec.LookPath` (chỉ trả bản đầu) với tự duyệt, thiết kế hàm nhận đầu vào thay vì đọc biến môi trường trực tiếp để test được.
