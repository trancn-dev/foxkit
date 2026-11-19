# Excel / Google Sheets

> exe: `EXCEL.EXE` (Excel), `chrome.exe`/`msedge.exe` (Google Sheets chạy trong trình duyệt, tiêu đề chứa "Google Sheets") · Ngữ cảnh: tên app + tiêu đề. Làm việc qua clipboard: ô copy từ Excel/Sheets là **TSV**.

## 1. Lệnh có sẵn, dùng ngay (appHints)
| Lệnh | Dùng khi |
|---|---|
| `csv` | Xem/lọc/thống kê file CSV (BOM, dấu `;`, ô nhiều dòng), đổi sang JSON/TSV |
| `fake` | Sinh dữ liệu giả tiếng Việt (CSV có BOM để Excel mở đúng dấu) |
| `ts` | Đổi cột timestamp sang ngày giờ |
| `case` | Slug/chuẩn hóa tên cột, bỏ dấu tiếng Việt |
| `dup` | Tìm file Excel trùng nội dung |
| `clip` | Đọc/ghi clipboard khi cần dán kết quả ngược |

Mẫu: `{ "app": "EXCEL.EXE", "commands": ["csv", "fake", "ts", "case"] }`.

## 2. Tool nhỏ mới đề xuất
| Lệnh đề xuất | Việc làm | Khó | Giá trị | Dependency |
|---|---|---|---|---|
| `table` | Clipboard TSV ↔ CSV/JSON/Markdown (xem `notepad.md`); là lệnh chung nhất của nhóm | 1 | ★★★ | stdlib |
| `serial` | Số serial ngày của Excel (`45000`) ↔ ngày thật, gồm lỗi năm 1900 | 1 | ★★ | stdlib |
| `money` | Định dạng tiền VND/USD (`1.250.000 ₫`), tách/ghép phần nghìn, số → chữ (dùng chung `numword`) | 2 | ★★ | stdlib |
| `dedupe` | Bỏ dòng trùng theo cột khóa trong CSV, giữ dòng đầu/cuối, báo số dòng bỏ | 2 | ★★★ | stdlib (nền `csvx`) |
| `join` | Nối hai CSV theo cột khóa (inner/left), như VLOOKUP | 3 | ★★ | stdlib |
| `xlsx` | Đọc/ghi `.xlsx` | 4 | ★★ | **cần dependency** (hoặc tự đọc zip+XML, đọc được ghi khó) → hỏi người dùng trước |

## 3. Việc launcher chưa làm được
- Không đọc được vùng ô đang chọn; chỉ qua clipboard (`@clip` đã đọc dòng đầu, **không đọc cả khối nhiều dòng** → lệnh `table`/`dedupe` cần cách nhận clipboard nhiều dòng: thêm placeholder `@clipall` hoặc cờ `--from-clip` ở foxkit, khó 1–2).
- Excel Việt Nam xuất CSV dấu `;` và mã hóa ANSI/UTF-8 BOM lẫn lộn; mục "Chưa kiểm" của `csv` ghi thiếu thử với file Excel VN thật.
