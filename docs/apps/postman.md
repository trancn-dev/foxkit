# Postman / Insomnia / Bruno (client API)

> exe: `Postman.exe`, `Insomnia.exe`, `Bruno.exe` · Ngữ cảnh: tên app + tiêu đề. Làm việc qua clipboard.

## 1. Lệnh có sẵn, dùng ngay (appHints)
| Lệnh | Dùng khi |
|---|---|
| `http` | Kiểm tra nhanh URL ngoài Postman: status, thời gian, redirect, SSL, `--insecure` cho cert tự ký |
| `json` | Format/lấy giá trị JSON response, kiểm tra body trước khi gửi |
| `jwt` | Xem header/payload/hạn dùng token, kiểm chữ ký HS* với secret |
| `enc` | base64/url encode cho Basic auth, query |
| `hash` | md5/sha256 cho chữ ký request |
| `uuid`, `pass` | Sinh id/khóa thử |
| `fake` | Sinh dữ liệu giả tiếng Việt (tên, SĐT, email) nhiều dòng JSON/CSV cho body mẫu |
| `serve` | Dựng nhanh server tĩnh/SPA có CORS làm nơi thử |
| `ts` | Đổi timestamp trong response |

Mẫu: `{ "app": "Postman.exe", "commands": ["json", "jwt", "http", "fake", "uuid"] }`.

## 2. Tool nhỏ mới đề xuất
| Lệnh đề xuất | Việc làm | Khó | Giá trị | Dependency |
|---|---|---|---|---|
| `hmac` | Tính HMAC-SHA256/SHA1 (chữ ký webhook, API ký request) từ khóa + chuỗi | 1 | ★★★ | stdlib (`crypto/hmac`) |
| `curl` | Chuyển đổi lệnh curl ↔ cờ `foxkit http` (xem `chrome.md`) | 2 | ★★★ | stdlib |
| `mock` | Server giả từ một file JSON mô tả route → response (mở rộng `serve`), có độ trễ giả | 3 | ★★ | stdlib (`net/http`) |
| `har` | Tóm tắt file HAR (request chậm, lỗi, dung lượng) | 2 | ★★ | stdlib |
| `webhook` | Mở server tạm in ra mọi request đến (thử webhook ở máy mình, không tunnel) | 2 | ★★ | stdlib; tunnel ra ngoài cần dịch vụ khác, ngoài phạm vi |
| `b64json` | Giải base64 lồng JSON nhiều tầng (payload JWT, message queue) | 1 | ★ | stdlib |

## 3. Việc launcher chưa làm được
- Không đọc được URL/body đang soạn trong Postman (ứng dụng Electron, không có UI Automation tốt). Dùng clipboard; không có hướng rẻ hơn.
- `http --header "Authorization: ..."` chứa bí mật: nên tránh lưu lệnh vào lịch sử; luồng chat hiện **chỉ trong bộ nhớ**, đã phù hợp; nhắc trong docs của `http`.
