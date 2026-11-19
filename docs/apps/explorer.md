# Explorer (File Explorer)

> exe: `explorer.exe` (cửa sổ thư mục) · Ngữ cảnh: **thư mục hiện tại** (`@dir`) và **các file đang chọn** (`@file`, `@files`) qua COM — là app giàu ngữ cảnh nhất. Desktop/taskbar không tính.

## 1. Lệnh có sẵn, dùng ngay (appHints)
| Lệnh | Dùng với |
|---|---|
| `size` | `size @dir`: mục nào chiếm nhiều dung lượng nhất |
| `dup` | `dup @dir`: file trùng nội dung, dung lượng lãng phí |
| `rename` | Đổi tên hàng loạt cho `@files` (plan/apply, có hoàn tác) |
| `find` | Tìm theo tên, đuôi, kích thước, ngày, nội dung |
| `hash` | `hash @files`: băm file đang chọn, `--check` so checksum |
| `zip pack` / `zip unpack` / `zip list` | Nén thư mục, giải nén an toàn |
| `sync` | Sao lưu một chiều thư mục này sang ổ khác (dry-run trước) |
| `lock` | File/thư mục bị "đang được dùng"? biết process nào giữ |
| `svg` | PNG/JPG đang chọn → SVG |
| `temp`, `disk` | Dọn file tạm, dung lượng các ổ |

Mẫu: `{ "app": "explorer.exe", "commands": ["size", "dup", "rename", "hash", "zip pack", "lock"] }`.

## 2. Tool nhỏ mới đề xuất
| Lệnh đề xuất | Việc làm | Khó | Giá trị | Dependency |
|---|---|---|---|---|
| `img` | Đổi cỡ, đổi định dạng PNG/JPG/GIF, ép dung lượng JPG (đã nằm backlog roadmap) | 2 | ★★★ | stdlib (`image/*`); WebP cần dependency → hỏi |
| `organize` | Sắp file trong thư mục (ví dụ Downloads) vào thư mục con theo đuôi hoặc theo tháng; plan/apply, có hoàn tác như `rename` | 2 | ★★★ | stdlib |
| `empty` | Tìm thư mục rỗng và file 0 byte, dọn có `--apply` | 1 | ★★ | stdlib |
| `tree` | In cây thư mục (độ sâu, bỏ `node_modules`/`.git`), copy được vào docs | 1 | ★★ | stdlib |
| `flatten` | Đưa mọi file ở thư mục con lên một cấp, xử lý trùng tên; plan/apply | 2 | ★ | stdlib |
| `verify` | Đọc file `.sha256`/`SHA256SUMS` rồi kiểm cả thư mục (mở rộng `hash --check`) | 2 | ★★ | stdlib |
| `pdf` (ghép/tách) | Ghép/tách PDF | 4 | ★ | **cần dependency**; ngoài phạm vi, chỉ ghi nhận |

## 3. Việc launcher chưa làm được
- COM chạy trong ngân sách 200 ms; Explorer có nhiều tab (Windows 11) → chỉ đọc được **tab đang hiển thị**; đã đúng với cách chọn cửa sổ theo `HWND`. Nên kiểm tay với nhiều tab (mục "Chưa kiểm" trong `PROGRESS.md`).
- Thư mục ảo (This PC, Recycle Bin, ổ mạng) → `dir` rỗng; có thể hiện lý do trong chip thay vì ẩn lặng lẽ (khó 1).
- Menu chuột phải của Explorer ("Gửi tới foxkit") để mở launcher với file đã chọn: cần ghi registry shell extension, **ngoài phạm vi**, ghi nhận.

Tham khảo: PowerRename, Peek, "New+" của PowerToys ([github.com/microsoft/PowerToys](https://github.com/microsoft/PowerToys)).
