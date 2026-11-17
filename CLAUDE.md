# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

# foxkit

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

## Lệnh hay dùng
```
go test ./internal/svgtrace                 # một package
go test ./internal/svgtrace -run TestRDP -v # một test
go test ./cmd                               # gồm test chống lệch manifest
go test -short ./...                        # bỏ test hiệu năng nặng
cd launcher && npm run test                 # vitest (lib TS)
cd launcher/src-tauri && cargo test && cargo clippy
cd launcher && npm run tauri build          # ra bản cài NSIS
```
`go test -race` không chạy được trên máy này (không có cgo/gcc).

## Điều cần biết (không thấy ngay khi đọc từng file)
- `cmd/root.go` tự dò `--json` từ args vì cobra báo lỗi trước khi parse cờ; lỗi kiểm đối số của cobra được quy về exit 2. Lệnh nhiều file (hash, svg) ghi lỗi vào từng item, in kết quả rồi mới trả lỗi để exit 1.
- `cmd/manifest_test.go` so cờ cobra với `internal/manifest/commands.json`: thêm cờ mà quên manifest là test đỏ. Manifest kiểu `float` chưa có, dùng `string`.
- Lệnh ghi hệ thống tách plan/apply (rename, prune, sync, branches, hosts, path). hosts có biến `FOXKIT_HOSTS` trỏ file tạm để thử mà không đụng hosts thật.
- Package trùng tên stdlib thì đổi tên: `syncdir`, `zipx`, `jsonx`, `csvx`, `textcase`, `winpath`. Go 1.27 đã có `uuid` trong stdlib nên không có `internal/uuid`.
- Launcher (`launcher/`): hai cửa sổ `bar` (Ctrl+Alt+Space mặc định, cấu hình trong `config.rs`) và `main`; thread chat chung giữ trong bộ nhớ Rust; lệnh ngoài đọc từ `foxkit.json` kèm snapshot và hash tin cậy. Logic thuần ở `src/lib/*.ts` (có vitest), phần Win32/COM ở `src-tauri/src/context.rs` chỉ kiểm tay được.
- `docs/PROGRESS.md` là nhật ký: mỗi lệnh một dòng kèm phần "Chưa kiểm" và ghi chú Go so với PHP/JS. Cập nhật khi xong mục.
- Ràng buộc an toàn khi làm tự động: không chạy `--apply` hay sửa hosts/PATH/registry/service thật; test dùng thư mục tạm và bản giả.
