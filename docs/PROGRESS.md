# PROGRESS

Quy ước: `[x]` xong, `[ ]` chưa làm. Hash ghi ở commit kế tiếp (commit không tự biết hash của chính nó). Phiên bị ngắt thì đọc file này + `git log` rồi làm tiếp.

## Mục việc

- [x] GĐ0 foundation — 650bf57. Thêm: `Execute` tự dò `--json` trong args vì cobra lỗi trước khi parse cờ; lỗi kiểm đối số của cobra ("unknown command", "accepts ...") quy về exit 2. Test chống lệch đã thử xóa entry rồi hoàn tác.
- [x] uuid — 7dd3c3e. Spec sửa: stdlib Go 1.27 đã có package `uuid` → bỏ `internal/uuid`, không có `UUID_RANDOM_FAILED`.
- [x] pass — 7dd3c3e (chung commit với uuid; các mục nhỏ gộp commit vì manifest dùng chung một file). Spec sửa: vòng lặp đảm bảo 4 loại ký tự nâng 100 → 10000 lần (len=4 chỉ ~6% chuỗi đủ loại).
- [x] hash — (commit "hash, enc"). Tiêu chí: vector chuẩn đạt; file lớn >1GB **chưa kiểm** (dùng `io.Copy` nên không nạp RAM) — thử: `.\foxkit hash D:\big.iso`. Thêm: `--check` không khớp in `khớp: false` rồi trả `HASH_MISMATCH`; item lỗi in ra stderr.
- [x] enc — (cùng commit với hash). `ENC_INPUT_TOO_LARGE` dùng chung `output.InputTooLarge` rồi đổi code.
- [ ] ts
- [ ] case
- [ ] port
- [ ] env
- [ ] L1 launcher (nhiều commit nhỏ theo mục 4, 5, 6.x, 7, 8)
- [ ] GĐ2: size · dup · rename · find · temp · prune · todo
- [ ] GĐ3: ps · kill · which · path · hosts · ip · dns · wifi
- [ ] GĐ4: json · jwt · http · tcp · serve
- [ ] GĐ5: proj · lara · repos · branches
- [ ] GĐ6: core/winapi · disk · sysinfo · path-edit · startup · svc · clip · lock · nâng cấp port/ps/kill
- [ ] GĐ7: fake · csv · diff · down · zip · sync · svg

## Chưa kiểm / để bạn tự thử trên Windows

(điền dần)

## Ghi chú Go (so với PHP/JS)

- **`internal/`**: Go cấm package ngoài module import thư mục này, như `private` cho cả thư mục.
- **`errors.As`**: ≈ `catch (MyException $e)`; kiểm tra lỗi (có thể đã bị bọc) có phải kiểu `*output.Error` không.
- **`go:embed`**: nhúng file vào binary lúc build, ≈ bundle JSON vào file build của Vite.
- **`io.Writer`**: truyền "nơi in" vào hàm (stdout thật hoặc buffer khi test), ≈ dependency injection.
- **`[16]byte` vs slice**: mảng có độ dài cố định là một phần của kiểu, so sánh bằng `==` được (≈ giá trị, không phải tham chiếu như array PHP/JS).
- **`crypto/rand` vs `math/rand`**: ≈ `random_bytes()` vs `rand()` của PHP; mật khẩu/UUID luôn dùng `crypto/rand`. `rand.Int(max)` không lệch phân bố, còn `byte % n` thì có.
