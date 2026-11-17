# CLAUDE.md — foxkit

> Copy file này vào gốc repo `foxkit/`. Claude Code tự đọc nó ở mỗi phiên.

## Dự án
foxkit là bộ công cụ CLI nhỏ cho Windows, viết bằng Go + cobra. Mỗi chức năng là một subcommand. Sau này có launcher Tauri đọc manifest (`foxkit manifest --json`) để tự sinh button.

Người phát triển: dev Laravel/Vue 2 đang học Go. Khi giải thích Go, so sánh với PHP/JS nếu giúp dễ hiểu. Trả lời bằng tiếng Việt, ngắn gọn.

## Tài liệu
- Roadmap: `docs/foxkit-roadmap.md`
- Spec từng lệnh: `docs/specs/{core,dev,life}/<lệnh>.md`. **Luôn đọc spec trước khi làm lệnh**, và đọc các spec ghi ở dòng "Cần trước".
- Launcher (Tauri, thư mục `launcher/`): `docs/specs/launcher/launcher.md`.
- Khung chung: `docs/specs/core/foundation.md` (output, exit code, manifest). Mọi lệnh tuân theo file này.

## Kiến trúc (bắt buộc)
- `cmd/<lệnh>.go`: chỉ parse tham số, gọi logic, đưa kết quả cho `internal/output`. Không chứa logic.
- `internal/<package>/`: logic, có test.
- Mọi lệnh hỗ trợ `--json` (cờ toàn cục). Kết quả ra stdout, lỗi ra stderr.
- Exit code: 0 thành công · 1 lỗi khi chạy · 2 sai tham số · 3 cần admin. Lệnh dùng `RunE` trả `error`, không gọi `os.Exit`.
- Lỗi dùng `output.Error{Code, Message, Hint}`; message tiếng Việt, hint nói rõ cần làm gì tiếp.
- Lệnh thay đổi hệ thống mặc định dry-run, cần `--apply`.
- Slice rỗng trong JSON phải là `[]`.
- Thêm/sửa lệnh **phải** cập nhật `internal/manifest/commands.json`; `go test ./...` có test chống lệch. Mỗi entry nên có 1–2 `examples` (lấy từ mục "Chạy thử" của spec) để launcher hiện ví dụ mờ.

## Quy tắc code
- Module: `foxkit`. Tên lệnh ngắn, chữ thường.
- Theo idiom Go: gofmt, xử lý mọi error, không `_ =` bỏ qua error (trừ khi có comment giải thích).
- Chỉ dùng thư viện chuẩn khi đủ. Dependency được phép: `cobra`; từ GĐ6: `golang.org/x/sys`, `github.com/atotto/clipboard`. Thêm cái khác phải hỏi và ghi lý do vào roadmap mục 5.
- Ưu tiên Windows. Code chỉ chạy trên Windows đặt trong `*_windows.go` kèm `*_other.go` để test/vet chạy được nơi khác.
- Không thêm tính năng ngoài spec (xem mục "Ngoài phạm vi" của từng spec).
- Không parse chữ có thể bị dịch theo ngôn ngữ Windows trong output của lệnh hệ thống; không dùng `wmic`.

## Quy trình mỗi lệnh
1. Đọc spec + spec "Cần trước".
2. Viết `internal/...` + test (dạng bảng), rồi `cmd/...`, rồi manifest.
3. Chạy:
   ```
   gofmt -l .
   go vet ./...
   go test ./...
   go build -o foxkit.exe .
   ```
4. Báo cáo từng ô "Tiêu chí hoàn thành" là đạt hay chưa, kèm lệnh "Chạy thử" người dùng nên tự chạy trên Windows.
5. Nếu thực tế khác spec (API khác tên, hành vi khác), sửa spec trong cùng thay đổi và nói rõ đã sửa gì.
6. Nếu code người dùng tự viết chưa idiomatic, chỉ ra và sửa kèm một câu giải thích.
