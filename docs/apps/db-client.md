# DB client (TablePlus, DBeaver, HeidiSQL, MySQL Workbench, phpMyAdmin)

> exe: `TablePlus.exe`, `dbeaver.exe`, `heidisql.exe`, `MySQLWorkbench.exe` · Ngữ cảnh: tên app + tiêu đề (thường có tên kết nối/database). Làm việc qua clipboard.

## 1. Lệnh có sẵn, dùng ngay (appHints)
| Lệnh | Dùng khi |
|---|---|
| `svc` | MySQL/PostgreSQL/Redis service có đang chạy không; `svc start MySQL80` (cần admin) |
| `tcp` | `tcp 127.0.0.1:3306` cổng DB có mở không |
| `port` | Ai chiếm cổng 3306/5432 |
| `csv` | Xem/lọc/thống kê file CSV xuất từ DB, đổi sang JSON |
| `fake` | Sinh dữ liệu giả tiếng Việt (CSV) để import thử |
| `ts` | Đổi cột timestamp ↔ ngày giờ |
| `uuid` | Sinh UUID (v4/v7) cho khóa chính |
| `json` | Format cột JSON copy từ ô dữ liệu |
| `diff` | So hai file `.sql` hoặc hai bản dump |
| `env get` | Đọc `DB_HOST`, `DB_PORT` trong `.env` dự án |

Mẫu: `{ "app": "TablePlus.exe", "commands": ["tcp", "svc", "csv", "fake", "ts", "json"] }`.

## 2. Tool nhỏ mới đề xuất
| Lệnh đề xuất | Việc làm | Khó | Giá trị | Dependency |
|---|---|---|---|---|
| `sqlin` | Clipboard nhiều dòng → `IN ('a','b','c')` (tự thoát nháy, tự nhận số/chuỗi) | 1 | ★★★ | stdlib |
| `csv2sql` | CSV → câu `INSERT` (hoặc `CREATE TABLE` đoán kiểu), chọn dialect MySQL/PostgreSQL | 2 | ★★★ | stdlib (nền `csvx`) |
| `dsn` | Phân tích/ghép chuỗi kết nối (`mysql://user:pass@host:3306/db`), che mật khẩu khi in | 1 | ★★ | stdlib (`net/url`) |
| `sqlfmt` | Định dạng câu SQL (xuống dòng theo mệnh đề, viết hoa từ khóa) | 3 | ★★ | stdlib (tự viết tokenizer; không bao phủ hết dialect, ghi rõ giới hạn) |
| `dbping` | Kiểm tra đăng nhập DB bằng thông tin trong `.env` dự án (không in mật khẩu) | 3 | ★★ | **cần driver** (`go-sql-driver/mysql`…): hỏi người dùng trước; thay thế rẻ: `tcp` + đọc `.env` |
| `dump` | Bọc `mysqldump`/`pg_dump` có sẵn trong máy, đặt tên file theo ngày | 2 | ★★ | stdlib (`os/exec`); mặc định dry-run in lệnh, không tự chạy dump |

## 3. Việc launcher chưa làm được
- Không biết **kết nối/database đang mở** ngoài tiêu đề cửa sổ; đọc file cấu hình kết nối (TablePlus/DBeaver lưu mật khẩu mã hóa) là **không nên** (rủi ro bảo mật), nên chỉ dùng tiêu đề để hiện trong chip.
- Kết quả lớn (nhiều nghìn dòng) từ `csv2sql`/`sqlin`: `ResultView` đã giới hạn bảng; nên có `--out` ghi file thay vì in hết.
