# VS Code (và Cursor, VSCodium)

> exe: `Code.exe`, `Code - Insiders.exe`, `Cursor.exe`, `VSCodium.exe` · Ngữ cảnh: tên app, **thư mục dự án** (`@dir`, đọc từ `storage.json`), không có file đang mở.

## 1. Lệnh có sẵn, dùng ngay (appHints)
| Lệnh | Dùng khi |
|---|---|
| `proj` | Nhận diện loại dự án, phiên bản, scripts chạy được (`proj @dir`) |
| `todo` | Quét TODO/FIXME/HACK trong dự án |
| `lara log` | Xem `laravel.log` của dự án Laravel đang mở |
| `lara key`, `lara clear` | Sinh `APP_KEY`, `optimize:clear` |
| `env diff` | So `.env` với `.env.example` |
| `repos` / `branches clean` | Repo nào chưa commit/push; dọn branch đã merge |
| `size`, `prune` | Thư mục nào nặng; dọn `node_modules`/`vendor` lâu không dùng |
| `port` | Dev server đang chiếm cổng nào |
| `serve` | Chạy nhanh web tĩnh/SPA cho thư mục dự án |
| `zip pack` | Nén dự án để gửi, tự bỏ `node_modules`, `.git`, `.env` |

Mẫu: `{ "app": "Code.exe", "commands": ["proj", "lara log", "env diff", "todo", "port"] }`.

## 2. Tool nhỏ mới đề xuất
| Lệnh đề xuất | Việc làm | Khó | Giá trị | Dependency |
|---|---|---|---|---|
| `gitignore` | Sinh `.gitignore` từ mẫu nhúng sẵn (Laravel, Node, Go, Vue, JetBrains…), gộp nhiều mẫu, dry-run | 2 | ★★★ | stdlib (`go:embed`) |
| `license` | Sinh file LICENSE (MIT/Apache/GPL…) với tên + năm | 1 | ★★ | stdlib |
| `deps` | Liệt kê phiên bản các gói trong `composer.lock`/`package-lock.json` đối chiếu `composer.json`/`package.json` (gói nào lệch, gói trùng), **offline** | 3 | ★★ | stdlib |
| `lorem` | Sinh văn bản/JSON giả (đã có `fake`; gom thêm đoạn văn tiếng Việt) | 1 | ★ | stdlib — có thể gộp vào `fake` thay vì lệnh mới |
| `wait-port` | Chờ tới khi một cổng mở (dev server khởi động xong) rồi thoát, dùng trong script | 1 | ★★ | stdlib (làm trên nền `tcp`) |

## 3. Việc launcher chưa làm được
- Không biết **file đang mở**/dòng đang chọn (tiêu đề chỉ có tên file). Hướng khả thi: tiện ích mở rộng VS Code nhỏ ghi `~/.foxkit/vscode-context.json` mỗi lần đổi file; **ngoài phạm vi stdlib/launcher hiện tại**, cần quyết định riêng.
- Workspace từ xa (WSL/SSH) và file `.code-workspace` → `dir` rỗng. Có thể thêm đọc `workspaces` trong `storage.json` cho `.code-workspace` (khó 2).
- Mở thư mục trong VS Code từ launcher (`code <dir>`): lệnh ngoài `foxkit.json` làm được, không cần sửa foxkit.
