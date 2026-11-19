# Chat (Slack, Zalo, Telegram, Teams, Discord)

> exe: `slack.exe`, `Zalo.exe`, `Telegram.exe`, `ms-teams.exe`, `Discord.exe` · Ngữ cảnh: tên app + tiêu đề. Làm việc qua clipboard (copy tin nhắn → chạy lệnh → dán lại).

## 1. Lệnh có sẵn, dùng ngay (appHints)
| Lệnh | Dùng khi |
|---|---|
| `ts` | Ai đó gửi timestamp/ngày giờ UTC: đổi sang giờ Việt Nam và nhiều múi giờ |
| `json` | Log/JSON dán vào chat: format cho dễ đọc trước khi gửi lại |
| `jwt` | Giải mã token được gửi để debug (không gửi secret) |
| `case`, `enc` | Đổi nhanh dạng tên biến, base64/url |
| `fake` | Sinh dữ liệu mẫu để gửi (tên, SĐT, email giả) |
| `uuid`, `pass`, `createpw` | Gửi mã/mật khẩu tạm (cẩn thận: `createpw --copy` để khỏi lộ trên màn hình) |
| `down` | Tải file từ link đã chia sẻ, nối tiếp khi đứt, kiểm checksum |
| `size`, `zip pack` | Gửi thư mục dự án đã nén, bỏ `node_modules`/`.env` |

Mẫu: `{ "app": "Zalo.exe", "commands": ["ts", "json", "createpw", "zip pack"] }`.

## 2. Tool nhỏ mới đề xuất
| Lệnh đề xuất | Việc làm | Khó | Giá trị | Dependency |
|---|---|---|---|---|
| `mask` | Che thông tin nhạy cảm trong text trước khi dán lên chat: token, mật khẩu, email, SĐT, `.env`; có chế độ chỉ phát hiện | 2 | ★★★ | stdlib (regex) |
| `time` | Đổi một giờ giữa các múi giờ cho cuộc họp ("9h sáng ICT = ? PST/CET") | 2 | ★★★ | stdlib (`time.LoadLocation`; Windows cần `tzdata` nhúng: `import _ "time/tzdata"`) |
| `numword` | Số → chữ tiếng Việt ("1.250.000" → "một triệu hai trăm năm mươi nghìn đồng") và ngược lại | 2 | ★★ | stdlib |
| `fmt` | Định dạng tin nhắn: code block, danh sách, trích dẫn cho Markdown của Slack/Telegram | 1 | ★ | stdlib |
| `short` | Rút gọn/nới rộng URL đã bỏ tham số theo dõi (dùng chung `url` ở `chrome.md`) | 1 | ★ | stdlib; rút gọn dịch vụ ngoài **không** làm |

## 3. Việc launcher chưa làm được
- Không đọc được tin nhắn đang chọn; luồng Electron/Qt không lộ nội dung qua Win32. Đúng hướng: người dùng copy rồi `@clip`.
- Không tự gửi/dán vào cửa sổ chat (giả phím, bị loại như ở `terminal.md`).
- `Zalo.exe` có thể không phải tên chính xác theo bản cài (Zalo PC có tiến trình `Zalo.exe` và `ZaloCall.exe`); cần kiểm tay tên exe trong tab Ứng dụng.
