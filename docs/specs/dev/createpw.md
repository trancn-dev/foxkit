# Spec: `foxkit createpw`

> Nhóm: **dev** · Ngoài roadmap gốc (thêm theo yêu cầu) · Trạng thái: xong · Cần trước: [core/foundation.md](../core/foundation.md), [pass.md](pass.md)

## 1. Mục đích
Sinh nhanh mật khẩu kiểu KeePassXC: **16 ký tự gồm A-Z a-z 0-9 và ký hiệu, ký tự đầu luôn là chữ hoa**, copy được vào clipboard ngay. Không thay `pass` (giữ nguyên hành vi `pass`).

## 2. Cú pháp
```
foxkit createpw [--len N] [--count N] [--no-ambiguous] [--copy] [--json]
```
| Cờ | Mặc định | Ràng buộc |
|---|---|---|
| `--len` | 16 | 4–256, ngoài khoảng → `PASS_LEN_INVALID` (exit 2) |
| `--count` | 1 | 1–100, ngoài khoảng → `PASS_COUNT_INVALID` (exit 2) |
| `--no-ambiguous` | false | bỏ `0 O o 1 l I \|` |
| `--copy` | false | copy mật khẩu **đầu tiên** vào clipboard; lỗi → `CLIP_WRITE_FAILED` (exit 1) |

## 3. Cách hoạt động
- Dùng lại `pass.Generate` với bộ ký tự `all` (đảm bảo đủ 4 loại: chữ hoa, chữ thường, số, ký hiệu `!@#$%^&*()-_=+[]{};:,.?`).
- `pass.GenerateStartUpper` sinh lại cho tới khi ký tự đầu là chữ hoa (trung bình ≈ 3 lần), nên phân bố của các ký tự còn lại vẫn đều, không thiên lệch. Nguồn ngẫu nhiên là `crypto/rand`.

## 4. Output
Text: mỗi mật khẩu một dòng; với `--copy` thêm một dòng ra stderr "Đã copy mật khẩu đầu tiên vào clipboard".
JSON: `{ "ok": true, "data": { "length": 16, "copied": false, "items": ["Y7gvDem9bNd:=X2B"] } }`.

## 5. Test
`TestGenerateStartUpper` (600 mẫu, có/không `--no-ambiguous`): đúng độ dài, ký tự đầu A–Z, đủ 4 loại ký tự, không còn ký tự dễ nhầm. `cmd/createpw_test.go` kiểm đầu ra, ràng buộc `--len`/`--count` (exit 2) và `--copy` bằng clipboard giả (ghi đúng mật khẩu đầu tiên, lỗi clipboard → exit 1). `--copy` với clipboard thật vẫn **chưa** thử vì ghi đè clipboard của người dùng.

Chạy thử (Windows):
```powershell
.\foxkit createpw
.\foxkit createpw --copy          # rồi Ctrl+V để dán thử
.\foxkit createpw --len 20 --count 3
```

## 6. Ngoài phạm vi
Passphrase (từ ngẫu nhiên), tùy chọn bộ ký tự, lưu/quản lý mật khẩu (xem `pass` cho bộ ký tự khác).
