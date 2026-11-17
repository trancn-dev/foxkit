# PROGRESS

Quy ước: `[x]` xong (kèm commit + ghi chú), `[ ]` chưa làm. Phiên bị ngắt thì đọc file này + `git log` rồi làm tiếp.

## Mục việc

- [x] GĐ0 core/foundation — commit `foundation:` (hash: xem git log). Tiêu chí đạt: gofmt/vet/test/build sạch, version+manifest text/JSON, lỗi cobra exit 2 có envelope, test chống lệch đã thử xóa entry rồi hoàn tác. Thêm: Execute dò `--json` trong args vì cobra lỗi trước khi parse cờ.
- [ ] GĐ1: uuid · pass · hash · enc · ts · case · port · env
- [ ] L1 launcher (nhiều commit nhỏ theo mục 4, 5, 6.x, 7, 8)
- [ ] GĐ2: size · dup · rename · find · temp · prune · todo
- [ ] GĐ3: ps · kill · which · path · hosts · ip · dns · wifi
- [ ] GĐ4: json · jwt · http · tcp · serve
- [ ] GĐ5: proj · lara · repos · branches
- [ ] GĐ6: core/winapi · disk · sysinfo · path-edit · startup · svc · clip · lock · nâng cấp port/ps/kill
- [ ] GĐ7: fake · csv · diff · down · zip · sync · svg

## Ghi chú Go (so với PHP/JS)

- **`internal/`**: Go cấm package ngoài module import thư mục này, như `private` cho cả thư mục.
- **`errors.As`**: ≈ `catch (MyException $e)`; kiểm tra lỗi (có thể đã bị bọc) có phải kiểu `*output.Error` không.
- **`go:embed`**: nhúng file vào binary lúc build, ≈ bundle JSON vào file build của Vite.
- **`io.Writer`**: truyền "nơi in" vào hàm (stdout thật hoặc buffer khi test), ≈ dependency injection.
