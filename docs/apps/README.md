# Tool nhỏ theo từng ứng dụng

> Trạng thái: **đề xuất, chưa làm** (trừ cột "Có sẵn"). Viết trong lượt chạy đêm 18/11/2026; cần người dùng duyệt trước khi thành spec. Nối với tính năng launcher "Gợi ý theo app" (`launcher.md` mục 6.4a): cột **Có sẵn** dùng được ngay bằng tab **Ứng dụng**, mặc định vẫn là không gợi ý gì.

## 1. Bảng tổng hợp theo app
| App | exe (tên để điền ở tab Ứng dụng) | Có sẵn (nên cấu hình trước) | Đề xuất mới (ưu tiên) | File |
|---|---|---|---|---|
| Chrome / Edge | `chrome.exe`, `msedge.exe` | `http` `dns` (+ `jwt` `json` `tcp`) | `url`, `curl`, `color` | [chrome.md](chrome.md) |
| VS Code | `Code.exe`, `Cursor.exe` | `proj` `todo` `lara log` `env diff` `port` | `gitignore`, `wait-port`, `deps` | [vscode.md](vscode.md) |
| Explorer | `explorer.exe` | `size` `dup` `rename` `hash` `zip pack` `lock` | `img`, `organize`, `empty`, `tree` | [explorer.md](explorer.md) |
| Terminal | `WindowsTerminal.exe`, `powershell.exe` | `port` `which` `ps` `lock` `env get` | `wait-port`, `open`, `envs` | [terminal.md](terminal.md) |
| Notepad | `notepad.exe` | `json` `case` `enc` `ts` `clip` | `lines`, `table`, `count`, `regex` | [notepad.md](notepad.md) |
| Postman / Insomnia | `Postman.exe`, `Insomnia.exe` | `json` `jwt` `http` `fake` `uuid` | `hmac`, `curl`, `mock`, `webhook` | [postman.md](postman.md) |
| DB client | `TablePlus.exe`, `dbeaver.exe`, `heidisql.exe` | `tcp` `svc` `csv` `fake` `ts` | `sqlin`, `csv2sql`, `dsn`, `sqlfmt` | [db-client.md](db-client.md) |
| Git GUI | `GitHubDesktop.exe`, `SourceTree.exe` | `repos` `branches clean` `todo` `env diff` | `changelog`, `secrets`, `gitstat` | [git-gui.md](git-gui.md) |
| Chat | `Zalo.exe`, `slack.exe`, `Telegram.exe` | `ts` `json` `createpw` `zip pack` | `mask`, `time`, `numword` | [chat.md](chat.md) |
| Excel / Sheets | `EXCEL.EXE` | `csv` `fake` `ts` `case` | `table`, `dedupe`, `serial`, `money` | [excel-sheets.md](excel-sheets.md) |
| Figma | `Figma.exe` | `svg` `rename` `zip pack` `size` | `color`, `palette`, `img`, `favicon` | [figma.md](figma.md) |
| Docker Desktop | `Docker Desktop.exe` | `port` `tcp` `http` `env diff` `disk` | `ports`, `composeenv`, `dk` | [docker.md](docker.md) |
| Laragon / XAMPP | `laragon.exe`, `xampp-control.exe` | `svc` `port` `hosts` `lara log` `env diff` `which` | `phpver`, `vhost`, `lara perm` | [laragon-xampp.md](laragon-xampp.md) |

## 2. Đề xuất mới xếp theo giá trị / độ khó
Lệnh xuất hiện ở nhiều app được gộp một dòng. Khó 1–5 như roadmap; ★ giá trị; "Dep" là có cần dependency ngoài stdlib/danh sách cho phép (`cobra`, `x/sys`, `clipboard`) hay không.

| # | Lệnh | App cần | Khó | Giá trị | Dep |
|---|---|---|---|---|---|
| 1 | `table` (TSV ↔ CSV/JSON/Markdown) | Excel, Notepad, DB | 1 | ★★★ | không |
| 2 | `lines` (sort/uniq/đếm/lọc dòng) | Notepad, DB | 1 | ★★★ | không |
| 3 | `url` (tách/ghép/sạch tham số) | Chrome, Chat | 1 | ★★★ | không |
| 4 | `hmac` | Postman | 1 | ★★★ | không |
| 5 | `sqlin` (dòng → `IN (...)`) | DB client | 1 | ★★★ | không |
| 6 | `color` (hex/rgb/hsl, tương phản) | Chrome, Figma | 1 | ★★★ | không |
| 7 | `wait-port` | Terminal, VS Code | 1 | ★★ | không |
| 8 | `curl` (curl ↔ `http`) | Chrome, Postman | 2 | ★★★ | không |
| 9 | `mask` (che bí mật trong text) | Chat, Git GUI | 2 | ★★★ | không |
| 10 | `gitignore`, `license` | VS Code, Git GUI | 2 | ★★★ | không |
| 11 | `img` (đổi cỡ/định dạng) | Explorer, Figma | 2 | ★★★ | WebP thì có |
| 12 | `organize`, `empty`, `tree` | Explorer | 1–2 | ★★★ | không |
| 13 | `csv2sql`, `dedupe` (nền `csvx`) | DB, Excel | 2 | ★★★ | không |
| 14 | `time` (đổi múi giờ họp) | Chat | 2 | ★★★ | không (`time/tzdata`) |
| 15 | `changelog` | Git GUI | 2 | ★★★ | không |
| 16 | `palette` (màu chủ đạo, dùng median cut của `svgtrace`) | Figma | 2 | ★★★ | không |
| 17 | `phpver` | Laragon | 2 | ★★★ | không |
| 18 | `ports`, `composeenv` | Docker | 2 | ★★★ | YAML đầy đủ thì có |
| 19 | `secrets` (quét khóa lộ) | Git GUI | 3 | ★★★ | không |
| 20 | `vhost` (vhost + hosts, plan/apply, admin) | Laragon | 3 | ★★★ | không |
| 21 | `numword`, `money`, `serial` | Chat, Excel | 1–2 | ★★ | không |
| 22 | `regex`, `count`, `dsn`, `open`, `envs`, `har`, `webhook`, `mock` | nhiều app | 1–3 | ★★ | không |
| 23 | `sqlfmt`, `deps`, `svgmin`, `lara routes`, `dk` | rải rác | 3 | ★★ | không |
| — | `qr`, `xlsx`, `pdf`, `dbping` | Chrome, Excel, Explorer, DB | 3–4 | ★★ | **có**: hỏi người dùng trước |

## 3. Việc cần sửa ở launcher (không phải tool mới)
| Việc | Hướng | Khó | Giá trị |
|---|---|---|---|
| Nhận clipboard **nhiều dòng** (`@clipall`) cho `table`, `lines`, `sqlin`, `mask` | thêm placeholder; hiện `@clip` chỉ lấy dòng đầu | 1–2 | ★★★ |
| Lấy URL tab trình duyệt | UI Automation đọc thanh địa chỉ | 4 | ★★ |
| `@dir` cho terminal | đọc cwd tiến trình con | 4 | ★★ |
| Đường dẫn repo của Git GUI, dự án của Laragon | đọc file cấu hình app (không có tài liệu chính thức, dễ gãy) | 3 | ★ |
| Mở ô nhanh với file đang chọn từ menu chuột phải Explorer | ghi registry shell extension | 4 | ★ — ngoài phạm vi |
| Giả phím (`@sel`, dán ngược) | **không làm**: đã từng đổi trạng thái CapsLock của người dùng | — | — |

## 4. Quy ước khi biến đề xuất thành lệnh
Mỗi lệnh đi theo quy trình ở `CLAUDE.md`: viết spec ở `docs/specs/<nhóm>/<lệnh>.md` (mục "Ngoài phạm vi" rõ ràng), `internal/<pkg>` có test dạng bảng, `cmd/<lệnh>.go` mỏng, thêm `commands.json` kèm `examples`. Lệnh thay đổi hệ thống mặc định dry-run. Dependency mới phải ghi lý do vào roadmap mục 5 và hỏi trước.

## 5. Tham khảo
- Microsoft PowerToys: Command Palette, PowerRename, Color Picker, Peek, Environment Variables ([learn.microsoft.com/windows/powertoys](https://learn.microsoft.com/windows/powertoys), [github.com/microsoft/PowerToys](https://github.com/microsoft/PowerToys)). Chỉ lấy ý tưởng nhóm chức năng, không sao chép nội dung.
