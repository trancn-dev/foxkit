# foxkit — Roadmap & mục lục spec

> Cập nhật: 2026-11-17 · Trạng thái: chưa có code. Mọi lệnh và launcher đã có spec chi tiết, sẵn sàng giao cho Claude Code.

## TL;DR

- **62 lệnh chạy được** (tính cả lệnh con như `hosts add`, `zip pack`), mô tả trong **46 file spec lệnh + 2 spec khung + 1 spec launcher (49 file)**, chia **2 nhóm**: `dev` (phục vụ lập trình) và `life` (tiện ích Windows hằng ngày), triển khai theo **8 giai đoạn** GĐ0 → GĐ7.
- Mỗi lệnh có **một file spec** trong `specs/<nhóm>/<lệnh>.md`, cùng một khuôn: mục đích → cú pháp → vị trí file → kiểu dữ liệu → cách hoạt động → output text/JSON → bảng lỗi → entry manifest → test → tiêu chí hoàn thành → ngoài phạm vi → khái niệm Go học được.
- **Bắt đầu**: [core/foundation.md](specs/core/foundation.md) → [dev/uuid.md](specs/dev/uuid.md) → [dev/port.md](specs/dev/port.md).
- GĐ0–GĐ5 và GĐ7 chỉ dùng thư viện chuẩn + cobra. GĐ6 thêm `golang.org/x/sys` và `atotto/clipboard`.
- **Launcher**: app Tauri chạy ngầm, bấm `Ctrl+Alt+Space` hiện ô gõ lệnh kiểu Spotlight, gợi ý theo manifest — spec [launcher/launcher.md](specs/launcher/launcher.md). Bắt đầu ngay sau GĐ1 để chốt schema manifest sớm.
- **Prompt bản MVP** (foundation + uuid + port + launcher nhỏ): [prompt-mvp.md](prompt-mvp.md).
- Cách giao việc cho Claude Code: xem [mục 6](#6-cách-dùng-với-claude-code) và file [CLAUDE.md](CLAUDE.md) mẫu cho repo.

---

## 1. Cấu trúc thư mục kế hoạch

```
plans/
├── foxkit-roadmap.md      ← file này (tổng hợp + mục lục)
├── CLAUDE.md              ← copy vào gốc repo foxkit để Claude Code luôn đọc
└── specs/
    ├── core/              ← khung chung, không phải lệnh người dùng
    │   ├── foundation.md  (GĐ0: output, exit code, manifest, version, helper)
    │   └── winapi.md      (đầu GĐ6: x/sys, gọi DLL, helper Win32)
    ├── dev/               ← lệnh phục vụ lập trình (27 file)
    ├── life/              ← tiện ích Windows hằng ngày (19 file)
    └── launcher/          ← app Tauri (ô lệnh phím tắt), không phải lệnh CLI
        └── launcher.md
```

Khi tạo repo, copy `specs/` vào `foxkit/docs/specs/` và `CLAUDE.md` vào gốc repo.

Thư mục `dev/` / `life/` là cách chia tài liệu. Trong manifest, `category` chia mịn hơn để launcher nhóm button: `core` (version, manifest), `dev`, `life` (file & tiện ích), `system` (process, ổ đĩa, startup), `network` (ip, dns, wifi, http, tcp).

---

## 2. Mục lục theo giai đoạn

Khó: 1 (vài giờ) → 5 (vài ngày, với người mới học Go). Giá trị: ★ → ★★★.

### GĐ0 — Khung
| Spec | Nội dung | Khó |
|---|---|---|
| [core/foundation.md](specs/core/foundation.md) | cobra root, `--json`, envelope, exit code, `internal/output`, `winexec`, `fsutil`, manifest + test chống lệch, lệnh `version`, `manifest` | 2 |

### GĐ1 — MVP: tiện ích dev hằng ngày
| Lệnh | Nhóm | Việc | Khó | Giá trị |
|---|---|---|---|---|
| [uuid](specs/dev/uuid.md) | dev | UUID v4/v7 | 1 | ★★ |
| [pass](specs/dev/pass.md) | dev | Mật khẩu/chuỗi ngẫu nhiên | 1 | ★★ |
| [createpw](specs/dev/createpw.md) | dev | Mật khẩu 16 ký tự bắt đầu bằng chữ hoa, copy được (thêm sau, ngoài kế hoạch gốc) | 1 | ★★ |
| [hash](specs/dev/hash.md) | dev | md5/sha*/crc32, kiểm checksum | 2 | ★★ |
| [enc](specs/dev/enc.md) | dev | base64/hex/url/html | 1 | ★★ |
| [ts](specs/dev/ts.md) | dev | Timestamp ↔ ngày giờ | 2 | ★★ |
| [case](specs/dev/case.md) | dev | camel/snake/kebab/slug tiếng Việt | 2 | ★★ |
| [port](specs/dev/port.md) | dev | Ai chiếm port, kill | 3 | ★★★ |
| [env](specs/dev/env.md) | dev | `env diff`, `env get` | 2 | ★★★ |

### GĐ2 — File & thư mục
| Lệnh | Nhóm | Việc | Khó | Giá trị |
|---|---|---|---|---|
| [dup](specs/life/dup.md) | life | File trùng | 4 | ★★ |
| [size](specs/life/size.md) | life | Thư mục/file nặng nhất | 3 | ★★ |
| [rename](specs/life/rename.md) | life | Đổi tên hàng loạt | 3 | ★★ |
| [find](specs/life/find.md) | life | Tìm file theo tên/đuôi/size/ngày/nội dung | 3 | ★★★ |
| [temp](specs/life/temp.md) | life | Dọn %TEMP% | 2 | ★★ |
| [prune](specs/dev/prune.md) | dev | Dọn node_modules/vendor cũ | 3 | ★★★ |
| [todo](specs/dev/todo.md) | dev | Liệt kê TODO/FIXME | 2 | ★★ |

### GĐ3 — Windows qua lệnh hệ thống
| Lệnh | Nhóm | Việc | Khó | Giá trị |
|---|---|---|---|---|
| [ps](specs/life/ps.md) | life | Danh sách process | 2 | ★★ |
| [kill](specs/life/kill.md) | life | Kill theo PID/tên | 2 | ★★ |
| [which](specs/dev/which.md) | dev | Mọi bản exe trong PATH | 2 | ★★★ |
| [path](specs/dev/path.md) | dev | `path`, `path check` | 2 | ★★ |
| [hosts](specs/dev/hosts.md) | dev | `hosts list/add/rm/on/off` | 3 | ★★★ |
| [ip](specs/life/ip.md) | life | IP LAN/public | 2 | ★★ |
| [dns](specs/life/dns.md) | life | Tra DNS, flush | 2 | ★★ |
| [wifi](specs/life/wifi.md) | life | Mật khẩu Wi-Fi đã lưu | 3 | ★ |

### GĐ4 — Dev: dữ liệu & mạng
| Lệnh | Nhóm | Việc | Khó | Giá trị |
|---|---|---|---|---|
| [json](specs/dev/json.md) | dev | Format/minify/validate/get | 2 | ★★ |
| [jwt](specs/dev/jwt.md) | dev | Giải mã, verify HS* | 2 | ★★ |
| [http](specs/dev/http.md) | dev | Status, timing, redirect, SSL | 3 | ★★ |
| [tcp](specs/dev/tcp.md) | dev | host:port có mở | 2 | ★★ |
| [serve](specs/dev/serve.md) | dev | Static server, SPA | 3 | ★★★ |

### GĐ5 — Trợ lý dự án
| Lệnh | Nhóm | Việc | Khó | Giá trị |
|---|---|---|---|---|
| [proj](specs/dev/proj.md) | dev | Nhận diện dự án, scripts | 3 | ★★★ |
| [lara](specs/dev/lara.md) | dev | `lara log/key/clear` | 3 | ★★★ |
| [repos](specs/dev/repos.md) | dev | Trạng thái nhiều repo | 3 | ★★ |
| [branches](specs/dev/branches.md) | dev | `branches clean` | 2 | ★★ |

### GĐ6 — Windows sâu (Win32 API)
| Spec | Nhóm | Việc | Khó | Giá trị |
|---|---|---|---|---|
| [core/winapi.md](specs/core/winapi.md) | core | x/sys, gọi DLL, helper, nâng cấp port/ps/kill | 3 | — |
| [disk](specs/life/disk.md) | life | Dung lượng ổ | 3 | ★★ |
| [sysinfo](specs/life/sysinfo.md) | life | OS, CPU, RAM, uptime | 4 | ★ |
| [path add/rm/clean](specs/dev/path-edit.md) | dev | Sửa PATH vĩnh viễn | 4 | ★★★ |
| [startup](specs/life/startup.md) | life | App tự chạy | 3 | ★ |
| [svc](specs/dev/svc.md) | dev | Bật/tắt MySQL, Redis... | 3 | ★★ |
| [clip](specs/life/clip.md) | life | Clipboard | 2 | ★★ |
| [lock](specs/life/lock.md) | life | Ai đang giữ file | 5 | ★★★ |

### GĐ7 — Mở rộng (chỉ thư viện chuẩn, làm khi GĐ1–GĐ5 đã ổn)
| Lệnh | Nhóm | Việc | Khó | Giá trị |
|---|---|---|---|---|
| [fake](specs/dev/fake.md) | dev | Dữ liệu giả tiếng Việt để seed/test | 2 | ★★★ |
| [csv](specs/dev/csv.md) | dev | Xem/lọc/thống kê CSV, CSV ↔ JSON | 3 | ★★ |
| [diff](specs/dev/diff.md) | dev | So sánh file/thư mục | 4 | ★★ |
| [down](specs/life/down.md) | life | Tải file, resume, checksum | 3 | ★★ |
| [zip](specs/life/zip.md) | life | `zip pack/unpack/list`, chống Zip Slip | 3 | ★★ |
| [sync](specs/life/sync.md) | life | Backup một chiều A → B | 4 | ★★★ |
| [svg](specs/life/svg.md) | life | PNG/JPG → SVG (vector hóa logo/icon, hoặc nhúng ảnh) | 5 | ★★ |

### L1 — Launcher (song song, bắt đầu sau GĐ1)
| Spec | Việc | Khó |
|---|---|---|
| [launcher/launcher.md](specs/launcher/launcher.md) | Tauri v2 + Vue 3: chạy ngầm ở tray; **giao diện chính** (tab Chat đầy đủ + form lệnh theo manifest, thư mục lệnh ngoài, cài đặt) và **ô chat mini** bật bằng hotkey `Ctrl+Alt+Space`, cả hai cùng một luồng chat; gợi ý gõ tiếp (ghost text) + ví dụ mờ phía trên ô; nhận diện app đang focus (Explorer, VS Code) để tự điền thư mục/file (`@dir`, `@file`, `@clip`); copy kết quả; hiện kết quả `--json`, dryRun/confirm/stream/chạy admin, khởi động cùng Windows; chọn thư mục (vd `D:\TranCN\foxkit`) để quét nhập CLI tool tự viết (file `foxkit.json` ở gốc hoặc thư mục con trực tiếp, cùng schema manifest + khối `exec`); quét một lần, `:scan` để quét lại, không realtime | 4 |

Code đặt ở `launcher/` trong cùng repo foxkit. Lệnh mới thêm vào foxkit tự xuất hiện trong launcher qua manifest, không phải sửa launcher.

### Vì sao thứ tự này
- **GĐ1**: hàm thuần dễ test, học cú pháp; `port`, `env` cho giá trị ngay.
- **GĐ2 trước GĐ3**: duyệt file dạy goroutine/channel trên bài toán dễ kiểm chứng.
- **GĐ4**: cần `context`, timeout, HTTP — học sau khi đã quen goroutine.
- **GĐ5**: dùng lại package các GĐ trước (`env`, `textcase`, `netinfo`, `fsutil`).
- **GĐ6**: phải gọi Win32 API, dùng `unsafe`, cần dependency.
- **GĐ7**: lệnh phụ, độc lập, chỉ dùng thư viện chuẩn; làm theo nhu cầu. `sync` và `zip` nên làm trước nếu hay backup dự án; `svg` khó nhất, để sau cùng.

---

## 3. Phụ thuộc giữa các spec

Spec ghi "Cần trước" ở đầu file. Các package dùng chung:

| Package | Tạo ở | Dùng lại ở |
|---|---|---|
| `internal/output` | foundation | mọi lệnh |
| `internal/winexec` (`Run`, `Tasklist`) | foundation | port, ps, kill, dns, wifi |
| `internal/fsutil` (`DirSize`, `ParseSize`, `ParseAge`, `DefaultSkipDirs`) | foundation | dup, size, temp, prune, find, todo, lara, diff, zip, sync |
| `internal/textcase` | case | rename, find, fake |
| `internal/env` | env | lara key |
| `internal/ps` | ps | kill |
| `internal/kill` | kill | lock |
| `internal/netinfo` | ip | serve |
| `internal/ts` | ts | jwt |
| `internal/git` | repos | branches |
| `internal/winpath` | path | path-edit |
| `internal/hash` | hash | down |
| `internal/winapi` | winapi | port, ps, kill (nâng cấp), sysinfo, lock |

---

## 4. Quy ước chung (tóm tắt — chi tiết ở foundation)

- `--json`: stdout `{ "ok": true, "data": ... }`; lỗi ra stderr `{ "ok": false, "error": { "code", "message", "hint" } }`.
- Exit code: 0 thành công · 1 lỗi khi chạy · 2 sai tham số · 3 cần admin.
- Lệnh thay đổi hệ thống mặc định **dry-run**, cờ `--apply` mới thực thi; data có `"applied"`.
- Lệnh chạy lâu (`serve`, `lara log --follow`) in NDJSON.
- Manifest ở `internal/manifest/commands.json`, nhúng vào binary, phát qua `foxkit manifest --json`; `go test` báo đỏ nếu lệch với cây lệnh cobra.
- Slice rỗng in `[]`, không `null`.
- Parse output lệnh Windows không dựa vào chữ có thể bị dịch theo ngôn ngữ; tránh `wmic`.

---

## 5. Dependency

| Package | Giai đoạn | Lý do |
|---|---|---|
| `github.com/spf13/cobra` | GĐ0 | Đã chốt trong kiến trúc |
| `golang.org/x/sys` | GĐ6 | Registry, ổ đĩa, service, process, gọi DLL — stdlib không có; do đội Go duy trì |
| `github.com/atotto/clipboard` | GĐ6 | Clipboard Unicode đúng, thuần Go; tự viết dễ sai quản lý bộ nhớ |

Launcher (Tauri, Rust/JS) có bảng dependency riêng ở [launcher.md mục 2](specs/launcher/launcher.md#2-công-nghệ-đã-chốt): `tauri` 2, plugin `global-shortcut`, `autostart`, `single-instance`, `clipboard-manager`, `dialog`, `sha2`, crate `windows` (lấy cửa sổ focus, COM Explorer, runas), `url`, Vue 3, Vite, TypeScript, vitest.

Không thêm dependency nào khác nếu chưa ghi lý do vào bảng này.

---

## 6. Cách dùng với Claude Code

### 6.1 Chuẩn bị repo (một lần)
1. Tạo repo `foxkit`, copy [CLAUDE.md](CLAUDE.md) vào gốc, copy thư mục `specs/` vào `docs/specs/`.
2. Giao việc đầu tiên:
   ```
   Đọc docs/specs/core/foundation.md và làm đúng spec đó. Xong thì chạy gofmt, go vet, go test ./... và go build, báo kết quả theo mục "Tiêu chí hoàn thành".
   ```

### 6.2 Mỗi lệnh mới (lặp lại)
```
Triển khai lệnh theo docs/specs/dev/uuid.md.
- Làm đúng phạm vi spec, không thêm tính năng ở mục "Ngoài phạm vi".
- Thêm entry vào internal/manifest/commands.json đúng mục 8 của spec.
- Viết test theo mục 9, chạy gofmt/go vet/go test/go build.
- Cuối cùng liệt kê từng ô trong "Tiêu chí hoàn thành" là đạt hay chưa.
- Giải thích ngắn các khái niệm Go ở mục cuối spec, so sánh với PHP/JS.
```

### 6.3 Mẹo
- Một lệnh một phiên/một PR. Spec "Cần trước" phải xong trước.
- Spec có chỗ sai so với thực tế khi làm (API khác tên, hành vi Windows khác) → bảo Claude Code sửa luôn file spec trong cùng PR để tài liệu không lệch code.
- Phần "Chạy thử" trong spec là lệnh PowerShell để bạn tự kiểm trên Windows sau khi Claude Code làm xong.

---

## 7. Lưu ý cho launcher Tauri

Spec đầy đủ: [launcher/launcher.md](specs/launcher/launcher.md). Tóm tắt hợp đồng với CLI:

Chạy ngầm ở tray, hotkey mặc định `Ctrl+Alt+Space` (không dùng `Win+F1` vì Windows giữ cho Help; `Alt+Space` trùng menu hệ thống và PowerToys Run).

0a. Hai cửa sổ trong một app, cùng một luồng chat (lệnh → kết quả, chỉ trong bộ nhớ): **giao diện chính** (tab Chat đầy đủ, form lệnh sinh từ manifest, thư mục lệnh, cài đặt; bấm X chỉ ẩn về tray) và **ô chat mini** bật bằng hotkey hoặc click tray, giống app Claude desktop.
0. Ngữ cảnh: lúc bấm phím, launcher chụp app đang focus + thư mục/file hiện tại (Explorer, VS Code) để tự điền đối số và đặt thư mục làm việc khi chạy foxkit; manifest có thêm field tùy chọn `examples` cho dòng ví dụ mờ.
0b. Lệnh ngoài: người dùng chọn thư mục bằng `:add`, launcher quét file `foxkit.json` ở gốc và thư mục con trực tiếp (schema manifest 1 + `exec`) một lần và lưu snapshot; chỉ quét lại khi `:scan`, không realtime; hỏi xác nhận lần đầu; xem [launcher.md mục 6.9](specs/launcher/launcher.md).
1. Khởi động: gọi `foxkit manifest --json` → sinh gợi ý lệnh theo `name/title/description`, gợi ý và validate tham số theo `args`/`flags` (`type`, `required`, `min/max`, `options`, `multiple`).
2. Chạy lệnh luôn kèm `--json`; exit 0 đọc stdout, khác 0 đọc stderr.
3. Exit 3 → nút "Chạy lại với quyền admin" (ShellExecuteExW verb `runas` qua `cmd /c`, output ghi ra file tạm vì process admin không đọc được qua pipe).
4. `longRunning` / `longRunningFlag` → đọc NDJSON từng dòng, nút Stop kill process.
5. `confirm` → dòng xác nhận ngay trong ô lệnh (không dùng hộp thoại hệ thống). `dryRun` → chạy xem trước, hiện thay đổi, bấm "Áp dụng" mới chạy lại với `--apply`.
6. Spawn từ Rust với cờ `CREATE_NO_WINDOW` (`0x08000000`) để không chớp console.
7. (Để sau, ngoài phạm vi launcher v1) `proj --json` trả `scripts` → có thể sinh lệnh động theo dự án.

---

## 8. Backlog (chưa có spec)

| Lệnh | Lý do để sau |
|---|---|
| `qr` | Cần dependency sinh QR |
| `db` (dump/restore MySQL) | Phụ thuộc `mysqldump` có trong PATH, cần bàn cách lưu mật khẩu an toàn |
| `img` (resize/đổi định dạng), `svg` → PNG | stdlib chỉ có jpeg/png/gif; webp cần dependency; SVG → PNG cần rasterizer |
| `timer`/`shutdown` | Nhỏ, chỉ bọc lệnh có sẵn |
| `cron` | Cần dep parser, giá trị vừa |
| `color` | Giá trị thấp, DevToys có sẵn |
| `lorem` | Giá trị thấp |
| `shutdown <30m>` | Chỉ bọc `shutdown /s /t` |
| `empty` (thư mục rỗng) | Giá trị thấp |
| `watch <dir> -- <cmd>` | Cần `ReadDirectoryChangesW` hoặc polling, phức tạp |
| Launcher: lịch sử lệnh, ngữ cảnh từ terminal/JetBrains, màn hình cài đặt | Ngoài phạm vi launcher v1, làm khi bản v1 đã dùng ổn |
| `startup on/off`, `svc` đổi startup type | Ghi registry/SCM, rủi ro; làm sau khi bản đọc ổn định |

---

## 9. Bản đồ khái niệm Go (so với PHP/JS)

| Khái niệm | Gặp lần đầu ở | So với PHP / JS |
|---|---|---|
| Module, package, `internal/` | foundation | `internal` ≈ private cho cả thư mục |
| Kiểu lỗi riêng + `errors.As` | foundation | ≈ custom Exception + `catch (X $e)` |
| `io.Writer` để test output | foundation | ≈ dependency injection |
| `go:embed`, `-ldflags -X` | foundation | nhúng file / inject biến lúc build |
| `[16]byte`, phép bit, `fmt.Stringer` | uuid | `String()` ≈ `__toString()` |
| `crypto/rand` vs `math/rand` | pass | ≈ `random_bytes()` vs `rand()` |
| `io.Reader`, `hash.Hash`, `io.MultiWriter` | hash | ≈ stream Node |
| `[]byte` vs `string`, `utf8` | enc | string Go là byte UTF-8 như PHP, khác JS UTF-16 |
| `time` layout `2006-01-02`, `time/tzdata` | ts | thay `Y-m-d`; ≈ `Carbon::setTestNow()` khi truyền `now` |
| `rune`, `unicode` | case | `len("đ") == 2` |
| `os/exec` + timeout | port | ≈ `Symfony\Process` |
| `bufio.Scanner`, parser giữ format, `testdata/` | env | ≈ `@dataProvider` |
| Goroutine, channel, `WaitGroup`, worker pool, `-race` | dup | ≈ `Promise.all` nhưng song song thật |
| Semaphore, `container/heap` | size | heap là ví dụ điển hình của interface |
| `regexp` (RE2), plan/apply, inject hàm để test lỗi | rename | RE2 không có lookbehind như PCRE |
| `fs.SkipDir`, lỗi sentinel | prune, find | ≈ `break` khỏi callback |
| `encoding/csv`, `sort.SliceStable` | ps | |
| `encoding/xml`, `MkdirTemp` + `defer RemoveAll` | wifi | |
| Ghi file nguyên tử (tmp + rename) | hosts | |
| `httptest` | ip, http | ≈ `Http::fake()` |
| `net.Resolver`, `context.WithTimeout` | dns | ≈ `AbortController` |
| `json.Indent/Compact`, `UseNumber`, type switch | json | |
| `crypto/hmac`, `*bool` | jwt | `*bool` ≈ `?bool` / `null` |
| `httptrace`, `crypto/tls` | http | |
| `http.Handler` middleware, graceful shutdown | serve | ≈ middleware Laravel |
| Struct tag JSON, con trỏ struct + `omitempty` | proj | ≈ DTO |
| `Seek` đọc ngược, `time.Ticker` | lara | ≈ `tail -f` |
| `exec.Cmd.Env` | repos | |
| Build tag `_windows.go`, `unsafe`, LazyDLL | winapi | ≈ PHP FFI |
| `x/sys/windows/registry` | sysinfo, path-edit, startup | |
| `svc/mgr`, least privilege | svc | |
| Interface để thay implementation | clip | ≈ bind interface trong Service Container |
| Struct C phức tạp, `runtime.KeepAlive`, gọi 2 lần lấy buffer | lock | |
| (Rust/Tauri) `#[tauri::command]`, `Channel`, capabilities | launcher | command ≈ route API, `Channel` ≈ SSE |
