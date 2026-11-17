# foxkit

Bộ công cụ dòng lệnh nhỏ cho Windows, viết bằng Go + cobra. Mỗi chức năng là một subcommand. Kèm launcher Tauri (`launcher/`) tự sinh nút từ `foxkit manifest --json`.

## Dùng nhanh

```powershell
go build -o foxkit.exe .
.\foxkit uuid
.\foxkit port 8000
.\foxkit size D:\Projects --top 10
.\foxkit svg logo.png --colors 4 --smooth
.\foxkit manifest --json      # danh sách lệnh cho launcher
```

Quy ước chung:
- `--json` (cờ toàn cục): kết quả ra stdout dạng `{"ok":true,"data":...}`, lỗi ra stderr.
- Exit code: `0` thành công · `1` lỗi khi chạy · `2` sai tham số · `3` cần quyền admin.
- Lệnh thay đổi hệ thống mặc định chỉ **xem trước (dry-run)**; thêm `--apply` mới ghi thật.
- Thông báo lỗi bằng tiếng Việt, kèm gợi ý bước tiếp theo.

## Các lệnh

| Nhóm | Lệnh |
|---|---|
| Dev | `uuid` `pass` `hash` `enc` `ts` `case` `port` `env` `prune` `todo` `which` `path` `hosts` `json` `jwt` `serve` `proj` `repos` `branches` `lara` `diff` `csv` `fake` `svc` |
| Hệ thống | `ps` `kill` `lock` `startup` `disk` `sysinfo` |
| Mạng | `ip` `dns` `wifi` `http` `tcp` |
| Đời sống | `size` `dup` `rename` `find` `temp` `sync` `zip` `down` `clip` `svg` |

Chạy `.\foxkit <lệnh> --help` để xem cờ. Spec từng lệnh nằm ở `docs/specs/<nhóm>/<lệnh>.md`.

## Phát triển

```powershell
gofmt -l .
go vet ./...
go test ./...
go build -o foxkit.exe .
```

Thêm hoặc sửa lệnh thì phải cập nhật `internal/manifest/commands.json`; `go test ./...` có test chống lệch.

Launcher:

```powershell
cd launcher
npm install
npm run test                      # vitest
cd src-tauri; cargo test          # test Rust
cd ..; npm run tauri build        # ra bản cài NSIS
```

Phím tắt mặc định của ô nhanh là `Ctrl+Alt+Space`, đổi được trong tab Cài đặt.

## Tài liệu

- Lộ trình: `docs/foxkit-roadmap.md`
- Khung chung (output, exit code, manifest): `docs/specs/core/foundation.md`
- Launcher: `docs/specs/launcher/launcher.md`
- Tiến độ, mục chưa kiểm: `docs/PROGRESS.md`
