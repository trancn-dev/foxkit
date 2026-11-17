# Spec: `foxkit lara` (log, key, clear)

> Nhóm: **dev** · Giai đoạn: GĐ5 · Trạng thái: chưa làm · Cần trước: [dev/env.md](env.md), [dev/proj.md](proj.md)

## 1. Mục đích
Ba việc Laravel hay làm nhất mà không cần mở IDE:
- `lara log`: đọc `laravel.log` dễ nhìn — tách từng entry, lọc theo level/thời gian, rút gọn stack trace.
- `lara key`: sinh `APP_KEY`, có thể ghi thẳng vào `.env`.
- `lara clear`: xóa mọi cache (`optimize:clear`).

Mọi lệnh con nhận `[dir]` (mặc định thư mục hiện tại) và kiểm tra đó là dự án Laravel (có `artisan` + `composer.json` chứa `laravel/framework`), không phải thì `LARA_NOT_PROJECT`.

---

## 2. `lara log`

### Cú pháp
```
foxkit lara log [dir] [--level error] [--last 20] [--since 1h] [--grep text] [--full] [--follow] [--json]
```
| Cờ | Mặc định | Ghi chú |
|---|---|---|
| `--level` | (tất cả) | mức **tối thiểu**: debug < info < notice < warning < error < critical < alert < emergency |
| `--last` | 20 | số entry mới nhất |
| `--since` | | `fsutil.ParseAge` (`30m`, `2h`, `1d`) |
| `--grep` | | chuỗi con trong message, không phân biệt hoa thường |
| `--full` | false | in đủ stack trace (mặc định chỉ 5 frame đầu thuộc `app/`) |
| `--follow` | false | theo dõi file như `tail -f`; `longRunning` |

### File log
- Mặc định `storage/logs/laravel.log`.
- Kênh `daily`: có `laravel-YYYY-MM-DD.log` → đọc các file mới nhất theo tên (giảm dần) cho tới khi đủ `--last` hoặc vượt `--since`.

### Định dạng entry Laravel (Monolog LineFormatter)
```
[2026-11-16 18:55:02] local.ERROR: SQLSTATE[HY000] [2002] Connection refused (Connection: mysql, SQL: select ...) {"exception":"[object] (Illuminate\\Database\\QueryException(code: 2002): SQLSTATE... at D:\\Projects\\shop\\vendor\\laravel\\framework\\src\\Illuminate\\Database\\Connection.php:825)
[stacktrace]
#0 D:\\Projects\\shop\\vendor\\laravel\\framework\\src\\Illuminate\\Database\\Connection.php(779): ...
#1 ...
"}
```
Regex đầu entry: `^\[(\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:[+-]\d{2}:?\d{2})?)\] (\w+)\.(\w+): (.*)$` → time, env, level, phần đầu message. Mọi dòng không khớp thuộc entry trước.

### Hợp đồng
```go
package laralog
type Entry struct {
    Time      string   `json:"time"`
    Env       string   `json:"env"`
    Level     string   `json:"level"`      // chữ hoa như trong log
    Message   string   `json:"message"`    // dòng đầu, bỏ phần context JSON
    Exception string   `json:"exception,omitempty"` // class exception
    File      string   `json:"file,omitempty"`      // file:line nơi ném lỗi
    Trace     []string `json:"trace,omitempty"`     // frame đã rút gọn (hoặc đủ khi --full)
    Count     int      `json:"count"`      // số lần lặp liên tiếp giống hệt message (gộp)
}
func ReadLast(paths []string, filter Filter, n int) ([]Entry, error)
func Follow(ctx context.Context, path string, filter Filter, emit func(Entry)) error
```

### Cách hoạt động
1. **Đọc từ cuối**: file log có thể vài GB. Mở file, `Seek` từ cuối lùi từng khối 64KB, tách dòng, ghép entry ngược cho tới khi đủ `n` entry thỏa filter hoặc gặp entry cũ hơn `--since`. Không đọc cả file.
2. Message: cắt phần ` {"exception":...` hoặc context JSON `{...}`/`[]` ở cuối dòng đầu.
3. Exception: regex `\(([\w\\]+)\(code: \w+\): .* at (.+):(\d+)\)` trong phần exception.
4. Trace rút gọn: chỉ frame chứa `\app\` hoặc `/app/` (code của mình), tối đa 5; nếu không có frame nào thuộc app thì lấy 3 frame đầu. Rút đường dẫn về tương đối so với dir dự án.
5. Gộp entry liên tiếp cùng level + message → `Count`.
6. `--follow`: in N entry cuối rồi theo dõi: vòng lặp `Stat` mỗi 500ms; size tăng → đọc phần mới; size giảm (file bị xóa/xoay vòng) → đọc lại từ đầu. Dừng bằng Ctrl+C. (Không dùng `ReadDirectoryChangesW` cho đơn giản.) Kênh daily: sang ngày mới thì chuyển file.
7. Màu theo level trong text mode khi là terminal (ERROR đỏ, WARNING vàng).

### Output
```
[18:55:02] ERROR  SQLSTATE[HY000] [2002] Connection refused  ×3
           Illuminate\Database\QueryException  vendor/laravel/framework/src/Illuminate/Database/Connection.php:825
           #12 app/Http/Controllers/OrderController.php(42)
           #13 app/Services/OrderService.php(88)
[18:57:40] WARNING  Slow query 1200ms ...
```
Gợi ý cuối khi có lỗi kết nối DB: `→ Kiểm tra MySQL: foxkit tcp 127.0.0.1:3306 hoặc foxkit svc` (chỉ in với message chứa `[2002]`).
JSON: `{ "ok": true, "data": { "files": ["storage/logs/laravel.log"], "items": [Entry...] } }`; `--follow --json` → NDJSON mỗi entry một dòng.

---

## 3. `lara key`

```
foxkit lara key [dir] [--apply] [--force] [--json]
```
1. Sinh 32 byte `crypto/rand` → `"base64:" + base64.StdEncoding`. (Giống `php artisan key:generate` với cipher mặc định AES-256-CBC.)
2. Không `--apply`: chỉ in key.
3. `--apply`: đọc `.env` bằng `internal/env`, nếu `APP_KEY` đã có giá trị và không `--force` → lỗi `LARA_KEY_EXISTS` (đổi key làm mất phiên đăng nhập và dữ liệu đã mã hóa). Có `--force` hoặc đang rỗng → `env.Set("APP_KEY", key)` → ghi file giữ nguyên format (ghi tmp + rename, backup `.env.foxkit.bak`).
4. JSON: `{ "key": "base64:...", "applied": true, "file": ".env" }`.

---

## 4. `lara clear`

```
foxkit lara clear [dir] [--json]
```
1. Tìm `php` bằng `exec.LookPath`; không có → `LARA_PHP_NOT_FOUND` (gợi ý `foxkit which php`).
2. Chạy `php artisan optimize:clear` trong `dir` (timeout 60s), stream stdout ra màn hình ở text mode.
3. Laravel < 9 không có các bước gộp đầy đủ → nếu lệnh lỗi "command not found" thì chạy lần lượt: `config:clear`, `cache:clear`, `route:clear`, `view:clear`.
4. JSON: `{ "ran": ["php artisan optimize:clear"], "output": "...", "exitCode": 0 }`. Exit code của foxkit = 0 nếu artisan 0, ngược lại 1 `LARA_ARTISAN_FAILED`.

---

## 5. File
```
cmd/lara.go            # cha
cmd/lara_log.go, lara_key.go, lara_clear.go
internal/laralog/parse.go, tail.go, follow.go, laralog_test.go
internal/laralog/testdata/laravel.log, laravel-2026-11-15.log
internal/lara/project.go   # kiểm tra dự án Laravel, tìm thư mục log
```

## 6. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `LARA_NOT_PROJECT` | | 2 | Chạy trong thư mục dự án Laravel hoặc truyền đường dẫn |
| `LARA_LOG_NOT_FOUND` | chưa có log | 1 | Chưa có lỗi nào được ghi, hoặc LOG_CHANNEL không phải single/daily |
| `LEVEL_INVALID` | | 2 | Chọn: debug, info, notice, warning, error, critical, alert, emergency |
| `LARA_KEY_EXISTS` | | 1 | APP_KEY đã có. Thêm --force nếu chắc chắn muốn đổi |
| `LARA_PHP_NOT_FOUND` | | 1 | Kiểm tra bằng foxkit which php |
| `LARA_ARTISAN_FAILED` | | 1 | Xem output phía trên |

## 7. Manifest
```json
{ "name": "lara log", "title": "Laravel log", "description": "Đọc laravel.log: lọc level, rút gọn trace", "category": "dev",
  "args": [ { "name": "dir", "type": "dir", "required": false, "label": "Dự án" } ],
  "flags": [
    { "name": "level", "type": "enum", "options": ["debug","info","notice","warning","error","critical","alert","emergency"], "label": "Level tối thiểu" },
    { "name": "last", "type": "int", "default": 20, "min": 1, "max": 1000, "label": "Số entry" },
    { "name": "since", "type": "string", "label": "Trong vòng" },
    { "name": "grep", "type": "string", "label": "Lọc nội dung" },
    { "name": "full", "type": "bool", "default": false, "label": "Đủ stack trace" },
    { "name": "follow", "type": "bool", "default": false, "label": "Theo dõi liên tục" }
  ], "admin": false, "longRunning": false, "dryRun": false, "output": "table" },
{ "name": "lara key", "title": "Sinh APP_KEY", "description": "Sinh APP_KEY, có thể ghi vào .env", "category": "dev",
  "args": [ { "name": "dir", "type": "dir", "required": false, "label": "Dự án" } ],
  "flags": [
    { "name": "apply", "type": "bool", "default": false, "label": "Ghi vào .env" },
    { "name": "force", "type": "bool", "default": false, "label": "Ghi đè key cũ", "confirm": "Đổi APP_KEY sẽ làm mất phiên đăng nhập và dữ liệu đã mã hóa. Tiếp tục?" }
  ], "admin": false, "longRunning": false, "dryRun": true, "output": "kv" },
{ "name": "lara clear", "title": "Xóa cache Laravel", "description": "php artisan optimize:clear", "category": "dev",
  "args": [ { "name": "dir", "type": "dir", "required": false, "label": "Dự án" } ],
  "flags": [], "admin": false, "longRunning": false, "dryRun": false, "output": "text" }
```
Lưu ý: entry `lara log` thêm `"longRunningFlag": "follow"` (xem schema ở foundation §9.1) để launcher biết khi bật `--follow` thì xử lý như lệnh stream.

## 8. Test
Fixture `testdata/laravel.log` gồm: 1 INFO một dòng; 1 ERROR có exception + 30 frame; 3 ERROR giống nhau liên tiếp; 1 entry có timestamp ISO có múi giờ; file kết thúc không có `\n`.
- Parse đúng số entry, level, exception, file:line.
- `--level warning` bỏ INFO. Gộp `Count == 3`.
- Đọc từ cuối: file 200MB tạo trong test (`-short` thì skip) chỉ đọc vài khối (đếm byte đọc qua reader bọc).
- `lara key`: key decode ra đúng 32 byte; `--apply` ghi đúng dòng, các dòng khác giữ nguyên; key đã có không `--force` → lỗi.

Chạy thử:
```powershell
cd D:\Projects\shop
.\foxkit lara log --level error --last 5
.\foxkit lara log --since 1h --grep SQLSTATE --json
.\foxkit lara log --follow
.\foxkit lara key
.\foxkit lara key --apply
.\foxkit lara clear
```

## 9. Tiêu chí hoàn thành
Tiêu chí chung + test + đọc file log thật >100MB tức thì.

## 10. Ngoài phạm vi
Log JSON formatter, Telescope/Horizon, xóa log, `artisan` lệnh khác.

## 11. Go học được
`os.File.Seek` + đọc ngược, regex nhiều nhóm, state machine ghép dòng thành entry, polling file với `time.Ticker`, dùng lại `internal/env` để ghi file.
