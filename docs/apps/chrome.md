# Chrome / Edge (và trình duyệt khác)

> exe: `chrome.exe`, `msedge.exe`, `firefox.exe`, `brave.exe` · Ngữ cảnh launcher lấy được: tên app + **tiêu đề cửa sổ** (tên trang). Chưa có URL, chưa có văn bản bôi đen.

## 1. Lệnh có sẵn, dùng ngay (appHints)
| Lệnh | Dùng khi |
|---|---|
| `http` | Kiểm tra status, thời gian, redirect, SSL của URL đang xem (dán URL bằng `@clip`) |
| `dns` | Tra bản ghi DNS của tên miền; `dns --flush` khi vừa đổi DNS mà trình duyệt chưa nhận |
| `jwt` | Token copy từ DevTools → Application/Network |
| `json` | JSON copy từ tab Network/Response, cần format hoặc lấy một khóa |
| `tcp` | Kiểm tra cổng của server dev có mở không |

Mẫu cấu hình (không đặt mặc định): `{ "app": "chrome.exe", "commands": ["http", "dns"] }`. Người dùng đã chọn gọn hai lệnh `http`, `dns`; `jwt`/`json` chỉ thêm khi cần.

## 2. Tool nhỏ mới đề xuất
| Lệnh đề xuất | Việc làm | Vào → Ra | Khó | Giá trị | Dependency |
|---|---|---|---|---|---|
| `url` | Tách/ghép URL: liệt kê query param, bỏ `utm_*`/`fbclid`, giải mã `%xx`, đổi tham số | URL (`@clip`) → bảng param hoặc URL sạch | 1 | ★★★ | stdlib (`net/url`) |
| `color` | Đổi hex ↔ rgb ↔ hsl, tính tỉ lệ tương phản WCAG của hai màu | `#1e88e5` → các dạng + độ tương phản | 1 | ★★ | stdlib |
| `curl` | Đổi lệnh `curl ...` (copy từ DevTools "Copy as cURL") thành cờ của `foxkit http` hoặc ngược lại | chuỗi curl → dòng `http` | 2 | ★★★ | stdlib |
| `har` | Tóm tắt file HAR: request chậm nhất, lỗi 4xx/5xx, tổng dung lượng theo domain | `.har` → bảng | 2 | ★★ | stdlib (`encoding/json`) |
| `qr` | Sinh mã QR cho URL/Wi-Fi | text → ảnh PNG/SVG | 3 | ★★ | **cần dependency** (stdlib không có QR; hoặc tự viết Reed–Solomon, khó) → hỏi người dùng trước |

## 3. Việc launcher chưa làm được
- **Lấy URL tab đang mở** (để `http` tự điền, không cần copy): đọc thanh địa chỉ bằng **UI Automation** (`IUIAutomation`, crate `windows` đã có) — tìm control "Address and search bar" của cửa sổ Chrome/Edge. Khó 4; chạy trong ngân sách 200 ms của việc chụp ngữ cảnh khá chặt; thanh địa chỉ có thể ẩn ở chế độ toàn màn hình. Cách rẻ hơn: ghost `@clip` (đã có) sau khi người dùng bấm Ctrl+L, Ctrl+C.
- **Văn bản đang bôi đen** (`@sel`): chỉ làm được bằng cách giả Ctrl+C rồi đọc clipboard, rủi ro ghi đè clipboard và nháy phím như lần đổi CapsLock; **không đề xuất** trước khi có cơ chế an toàn.
- Tên trang đã có (`title`) nhưng chưa dùng; có thể hiện trong chip để biết ngữ cảnh.

Tham khảo: nhóm tiện ích "Command Palette", "Color Picker", "Environment Variables" của Microsoft PowerToys ([learn.microsoft.com/windows/powertoys](https://learn.microsoft.com/windows/powertoys)).
