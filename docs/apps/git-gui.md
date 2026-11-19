# Git GUI (GitHub Desktop, SourceTree, GitKraken, TortoiseGit)

> exe: `GitHubDesktop.exe`, `SourceTree.exe`, `gitkraken.exe`, `TortoiseGitProc.exe` · Ngữ cảnh: tên app + tiêu đề (có tên repo với GitHub Desktop). Chưa có đường dẫn repo.

## 1. Lệnh có sẵn, dùng ngay (appHints)
| Lệnh | Dùng khi |
|---|---|
| `repos` | Nhìn một lượt mọi repo trong thư mục dự án: chưa commit, chưa push, bị tụt |
| `branches clean` | Dọn branch local đã merge hoặc đã mất trên remote (dry-run trước, có `--apply`) |
| `todo` | TODO/FIXME còn sót trước khi commit |
| `diff` | So hai file/thư mục ngoài git (bản sao lưu, bản triển khai) |
| `env diff` | Tránh quên key mới trong `.env.example` trước khi commit |
| `zip pack` | Đóng gói bản phát hành tự bỏ `.git`, `node_modules`, `.env` |
| `hash` | Kiểm checksum file release |

Mẫu: `{ "app": "GitHubDesktop.exe", "commands": ["repos", "branches clean", "todo", "env diff"] }`.

## 2. Tool nhỏ mới đề xuất
| Lệnh đề xuất | Việc làm | Khó | Giá trị | Dependency |
|---|---|---|---|---|
| `changelog` | Sinh CHANGELOG từ `git log` giữa hai tag, nhóm theo `feat/fix/...` (Conventional Commits) | 2 | ★★★ | stdlib (qua `internal/git`) |
| `gitstat` | Thống kê repo: số commit theo tác giả/tuần, file đổi nhiều nhất, kích thước `.git` | 2 | ★★ | stdlib |
| `secrets` | Quét thư mục/diff chuẩn bị commit tìm khóa lộ (AWS key, token, `.env`, private key) bằng regex | 3 | ★★★ | stdlib; tỉ lệ báo nhầm cần cảnh báo rõ, chỉ báo, không sửa |
| `stale` | Liệt kê branch/ tag cũ không đụng quá N ngày kể cả remote | 2 | ★★ | stdlib |
| `gitignore` | Sinh `.gitignore` (xem `vscode.md`) | 2 | ★★★ | stdlib |

## 3. Việc launcher chưa làm được
- Không biết **repo đang mở** → các lệnh trên phải nhận đường dẫn tay hoặc `@dir` (chỉ có ở Explorer/VS Code). GitHub Desktop lưu danh sách repo trong `%APPDATA%\GitHub Desktop\` (khó 3, không có tài liệu chính thức → có thể gãy); đề xuất chỉ dùng tên repo trong tiêu đề để khớp với kết quả `repos`.
- Mọi thao tác ghi (`branches clean --apply`) vẫn phải xác nhận trong ô nhanh; **không** thêm lệnh tự commit/push từ launcher.
