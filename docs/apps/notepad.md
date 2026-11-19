# Notepad (và trình soạn thảo văn bản khác)

> exe: `notepad.exe`, `Notepad++` (`notepad++.exe`), `Obsidian.exe` · Ngữ cảnh: tên app + tiêu đề (có tên file). Không đọc được nội dung; làm việc qua **clipboard** (`@clip`).

## 1. Lệnh có sẵn, dùng ngay (appHints)
| Lệnh | Dùng với văn bản copy |
|---|---|
| `json` | Format/minify/kiểm tra JSON, `--get` lấy một khóa |
| `case` | Đổi `camelCase`/`snake_case`/`kebab`/slug tiếng Việt |
| `enc` | base64, url, html, hex |
| `diff` | So hai file hoặc hai thư mục (so hai đoạn text: xem đề xuất bên dưới) |
| `ts` | Timestamp ↔ ngày giờ |
| `csv` | Xem/lọc/đổi CSV |
| `clip` | Đọc/ghi clipboard (dán kết quả ngược lại) |
| `hash` | Băm chuỗi (`--text`) |

Mẫu: `{ "app": "notepad.exe", "commands": ["json", "case", "enc", "ts", "clip"] }`.

## 2. Tool nhỏ mới đề xuất
| Lệnh đề xuất | Việc làm | Khó | Giá trị | Dependency |
|---|---|---|---|---|
| `lines` | Thao tác theo dòng trên text/clipboard: sắp xếp, bỏ trùng, đếm, đảo, thêm tiền/hậu tố, lọc | 1 | ★★★ | stdlib |
| `count` | Đếm ký tự/từ/dòng/byte (UTF-8) và thời gian đọc | 1 | ★★ | stdlib |
| `table` | Clipboard TSV (copy từ Excel) ↔ Markdown table / CSV / JSON | 1 | ★★★ | stdlib |
| `regex` | Thử biểu thức chính quy (RE2 của Go) trên text, hiện nhóm bắt | 2 | ★★ | stdlib (lưu ý RE2 khác PCRE của PHP/JS: không có lookahead) |
| `diff --text` | So hai đoạn text (một từ clipboard, một từ file/stdin) — mở rộng `diff` hiện có | 1 | ★★ | stdlib |
| `lorem` | Văn bản giả tiếng Việt/Anh theo số từ/đoạn | 1 | ★ | stdlib |

## 3. Việc launcher chưa làm được
- Nhiều lệnh trên cần kết quả **dán ngược** vào văn bản: hiện chỉ có Copy (người dùng tự Ctrl+V). `autoCopy` đã đủ cho luồng này; không cần giả phím.
- Tiêu đề Notepad chứa tên file nhưng không có đường dẫn đầy đủ → không có `@file`. Notepad++ có thể đọc `session.xml` để biết file đang mở (khó 3, ưu tiên thấp).
