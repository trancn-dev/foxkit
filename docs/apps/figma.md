# Figma (và công cụ thiết kế)

> exe: `Figma.exe` (app desktop), hoặc `chrome.exe` với tiêu đề chứa "Figma"; `Photoshop.exe`, `Inkscape` (`inkscape.exe`) · Ngữ cảnh: tên app + tiêu đề.

## 1. Lệnh có sẵn, dùng ngay (appHints)
| Lệnh | Dùng khi |
|---|---|
| `svg` | Logo/icon PNG/JPG → SVG (trace), hoặc nhúng ảnh; `--mono` cho logo/chữ ký |
| `size` | Thư mục asset nào nặng (ảnh export) |
| `dup` | Ảnh export trùng nội dung |
| `rename` | Đặt tên hàng loạt cho ảnh export (`icon-01`, `icon-02`) |
| `zip pack` | Đóng gói bộ asset giao cho dev |
| `hash` | Kiểm tra file giao nhận không đổi |
| `down` | Tải asset từ link, nối tiếp khi đứt |

Mẫu: `{ "app": "Figma.exe", "commands": ["svg", "rename", "zip pack", "size"] }`.

## 2. Tool nhỏ mới đề xuất
| Lệnh đề xuất | Việc làm | Khó | Giá trị | Dependency |
|---|---|---|---|---|
| `color` | Hex ↔ rgb ↔ hsl, độ tương phản WCAG, gợi ý màu đậm/nhạt hơn | 1 | ★★★ | stdlib |
| `px` | px ↔ rem/em/pt theo cỡ chữ gốc; tính tỉ lệ khung hình, scale `@1x/@2x/@3x` | 1 | ★★ | stdlib |
| `palette` | Rút bảng màu chủ đạo từ ảnh PNG/JPG (dùng lại median cut của `svgtrace`) | 2 | ★★★ | stdlib |
| `img` | Đổi cỡ/đổi định dạng ảnh hàng loạt cho `@1x/@2x` (xem `explorer.md`) | 2 | ★★★ | stdlib; WebP cần dependency |
| `svgmin` | Rút gọn SVG: bỏ metadata/comment, làm tròn số, gộp path | 3 | ★★ | stdlib (cẩn thận không làm hỏng hình; test so sánh hình học như `svg`) |
| `favicon` | Từ một PNG sinh bộ favicon nhiều cỡ (16–512) | 2 | ★★ | stdlib; `.ico` nhiều cỡ tự ghi header |

## 3. Việc launcher chưa làm được
- Không đọc được file/frame đang chọn trong Figma; asset export phải tự chọn bằng Explorer (`@files`). Chấp nhận.
- `svg --mono` đã có spec; đề xuất `palette` + `svg` thành chuỗi "ảnh → bảng màu + SVG" chỉ nên làm nếu người dùng thật sự cần.
- Kiểm `svg` bằng mắt trong trình duyệt/Inkscape vẫn là mục "Chưa kiểm" trong `PROGRESS.md`.
