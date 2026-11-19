# NIGHT-LOG — lượt chạy đêm 18/11/2026 (lịch 2:02)

> Nhật ký lượt chạy tự động theo lịch. Không có thao tác git ghi nào (không add/commit/push/stash); mọi thay đổi nằm ở working tree để người dùng duyệt.

## Giai đoạn 1 — Docs: **xong**
- Tạo `docs/apps/` gồm `README.md` (bảng tổng hợp + xếp hạng đề xuất + việc launcher) và 13 file theo app: `chrome`, `vscode`, `explorer`, `terminal`, `notepad`, `postman`, `db-client`, `git-gui`, `chat`, `excel-sheets`, `figma`, `docker`, `laragon-xampp`.
- Thêm một dòng liên kết ở `README.md` (mục Tài liệu) và `docs/foxkit-roadmap.md` (mục 8 Backlog).
- Mỗi app có: lệnh có sẵn dùng làm `appHints` (kèm mẫu cấu hình, **không** đặt mặc định), tool mới đề xuất (khó, giá trị, dependency), việc launcher chưa làm được.
- Chỉ là đề xuất; chưa viết spec/code cho lệnh mới nào.

## Giai đoạn 2 — Cải thiện/tối ưu
### Đã sửa / thêm (đều ở working tree, chưa commit)
| File | Việc |
|---|---|
| `cmd/createpw_test.go` (mới) | Test `createpw`: mặc định 16 ký tự bắt đầu bằng chữ hoa, `--len/--count`, ràng buộc exit 2, `--copy` ghi đúng mật khẩu đầu bằng clipboard giả, clipboard lỗi → exit 1 |
| `launcher/src/App.vue` | `fit()` bỏ qua lần `setSize` trùng chiều cao (ép đặt lại mỗi lần cửa sổ hiện); bọc `watch` bằng `() => void fit()` để tham số watch không biến thành `force` |
| `docs/specs/dev/createpw.md`, `docs/specs/launcher/launcher.md` | Cập nhật mô tả test và việc bỏ qua `setSize` trùng |
| `docs/PROGRESS.md` | Thêm mục kiểm đêm 18/11 |
| `docs/apps/*`, `README.md`, `docs/foxkit-roadmap.md` | Giai đoạn 1 |

### Kiểm thật chỉ-đọc (không ghi hệ thống)
- `csv` file ANSI → `Lỗi: File không phải UTF-8` + gợi ý Save As CSV UTF-8; file BOM dấu `;` → đúng.
- `http https://example.com` đủ DNS/Kết nối/TLS/Chờ/SSL (còn 38 ngày)/Header; chứng chỉ hết hạn → lỗi tiếng Việt kèm `--insecure`; `dns example.com` (A/AAAA/CNAME); `tcp example.com:443` ✓.

### Kết quả test cuối
Go: `gofmt -l .` rỗng, `go vet ./...` sạch, `go test ./...` pass. Launcher: `vue-tsc` sạch, vitest 63/63, `cargo test` 33/33.

### Việc dở / cần bạn xử lý
- **Hai tiến trình Vite của tôi còn chạy** (PID 21916 `npx vite` và 30584 `vite`), đang giữ **cổng 1420**; hệ thống chặn việc tôi dừng chúng. Hãy dừng trước khi chạy `npm run tauri dev` (cổng cố định, sẽ báo trùng): `Stop-Process -Id 21916,30584`.
- Không build lại bản release (launcher đang chạy, file bị khóa), nên `App.vue` mới chỉ có ở mã nguồn; build lại khi muốn dùng.
- `node_modules\.vite` chưa dọn (cache, vô hại).

### Làm tiếp theo yêu cầu "update tiếp"
- Viết spec (chỉ docs, chưa có code, chưa vào manifest) cho 3 lệnh đầu trong gợi ý: `docs/specs/dev/table.md`, `lines.md`, `url.md`. Trạng thái "chưa làm, chờ duyệt".

### Đề xuất lớn, chưa làm (cần bạn quyết)
1. Placeholder `@clipall` (clipboard nhiều dòng) — `table`, `lines`, `sqlin`, `mask` trong `docs/apps/` cần; hiện `@clip` chỉ lấy dòng đầu.
2. Chọn lệnh mới nào trong `docs/apps/README.md` mục 2 để viết spec đầu tiên (gợi ý: `table`, `lines`, `url`, `hmac`, `sqlin`, `color`).
3. Lấy URL tab trình duyệt qua UI Automation (khó 4) — chỉ làm nếu bạn thật sự cần.
4. Dependency chờ bạn hỏi lại: `qr`, `xlsx`, `pdf`, `dbping`, YAML cho `ports`.
