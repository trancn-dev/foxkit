# Spec: Khung chung foxkit (GĐ0)

> Nhóm: **core** · Giai đoạn: GĐ0 · Trạng thái: chưa làm
> **Làm file này đầu tiên.** Mọi spec lệnh khác đều giả định khung này đã có.

---

## 1. Mục đích

Dựng bộ khung để mọi lệnh sau dùng chung:

- cobra root + cờ `--json` toàn cục;
- package `internal/output`: in kết quả/lỗi, envelope JSON, exit code;
- `commands.json` nhúng vào binary + lệnh `manifest` + test chống lệch;
- lệnh `version`;
- helper dùng chung: `internal/winexec` (chạy lệnh Windows), `internal/fsutil` (duyệt thư mục, parse size/duration).

## 2. Khởi tạo

```powershell
mkdir foxkit; cd foxkit
go mod init foxkit
go get github.com/spf13/cobra@latest
```

Dependency duy nhất ở GĐ0–GĐ5: `github.com/spf13/cobra` (đã chốt trong kiến trúc dự án).

## 3. Cấu trúc thư mục

```
foxkit/
├── main.go                    # chỉ gọi os.Exit(cmd.Execute())
├── cmd/
│   ├── root.go
│   ├── version.go
│   ├── manifest.go
│   ├── manifest_test.go
│   └── <lệnh>.go              # mỗi lệnh 1 file
├── internal/
│   ├── output/                # output.go, errors.go, output_test.go
│   ├── winexec/               # run.go, tasklist.go
│   ├── fsutil/                # walk.go, units.go (ParseSize, ParseAge, FormatBytes)
│   ├── manifest/              # commands.json + manifest.go (go:embed)
│   └── <lệnh>/                # logic từng lệnh
└── docs/specs/                # copy các spec này vào đây để Claude Code đọc
```

`go:embed` chỉ nhúng được file nằm trong thư mục của package (hoặc thư mục con), nên **manifest đặt ở `internal/manifest/commands.json`** và embed ngay trong package đó:

```go
// internal/manifest/manifest.go
package manifest

import _ "embed"

//go:embed commands.json
var Raw []byte
```

→ Chốt: manifest nằm ở `internal/manifest/commands.json`. Launcher không đọc file này trực tiếp mà gọi `foxkit manifest --json`.

## 4. `internal/output`

### 4.1 Kiểu dữ liệu

```go
package output

type Error struct {
    Code    string `json:"code"`    // VD "PORT_INVALID"
    Message string `json:"message"` // tiếng Việt, nói rõ chuyện gì
    Hint    string `json:"hint,omitempty"` // cần làm gì tiếp
    Exit    int    `json:"-"`       // 1, 2 hoặc 3
}

func (e *Error) Error() string { return e.Message }

// Constructor ngắn gọn
func Usage(code, msg, hint string) *Error   // Exit 2
func Fail(code, msg, hint string) *Error    // Exit 1
func NeedAdmin(msg string) *Error           // Exit 3, Code "NEED_ADMIN", Hint "Chạy lại terminal bằng quyền Administrator"
```

### 4.2 In kết quả

```go
type Printer struct {
    JSON   bool
    Stdout io.Writer
    Stderr io.Writer
}

// Success: JSON → {"ok":true,"data":<data>}; text → gọi textFn(w)
func (p Printer) Success(data any, textFn func(w io.Writer) error) error

// Fail: in lỗi ra stderr, trả exit code
func (p Printer) Error(err error) int

// Stream (NDJSON) cho lệnh chạy lâu: mỗi lần gọi in 1 dòng JSON
func (p Printer) Event(v any) error
```

Quy tắc:
- `Success` ở chế độ JSON dùng `json.NewEncoder(w)` với `SetEscapeHTML(false)` (để `&`, `<` không bị thành `&`) và `SetIndent("", "  ")`.
- `Error`: nếu `err` là `*output.Error` (kiểm bằng `errors.As`) thì dùng code/exit của nó; lỗi khác coi là `INTERNAL`, exit 1, message = `err.Error()`.
- Text mode: lỗi in dạng `Lỗi: <message>` và dòng thứ hai `→ <hint>` nếu có.
- **Slice rỗng phải in `[]`, không in `null`**: khởi tạo `make([]T, 0)` trong logic, không để nil.

### 4.3 Exit code

| Code | Ý nghĩa |
|---|---|
| 0 | Thành công (kể cả kết quả "không tìm thấy" hợp lệ) |
| 1 | Lỗi khi chạy |
| 2 | Sai tham số |
| 3 | Cần quyền admin |

`main.go`:
```go
func main() { os.Exit(cmd.Execute()) }
```
`cmd.Execute()` trả `int`. Đặt `SilenceErrors: true` và `SilenceUsage: true` trên root để cobra không tự in lỗi; lỗi parse cờ của cobra quy về `Usage("ARGS_INVALID", ...)` qua `SetFlagErrorFunc`.

## 5. `cmd/root.go`

- `Use: "foxkit"`, `Short: "Bộ công cụ CLI nhỏ cho Windows"`.
- Persistent flag `--json` (bool).
- Mỗi lệnh dùng `RunE` và trả `error`; không gọi `os.Exit` trong lệnh.
- Helper: `func printer(cmd *cobra.Command) output.Printer` lấy cờ `--json` và `cmd.OutOrStdout()` / `cmd.ErrOrStderr()` (để test bắt được output).
- Root đặt `CompletionOptions.DisableDefaultCmd = true` (không cần lệnh `completion` cho launcher, tránh lệch manifest).

## 6. Quy ước cho lệnh thay đổi hệ thống

- Mặc định dry-run. Cờ `--apply` mới thực thi.
- Data luôn có field `"applied": true|false`.
- Text mode khi dry-run kết thúc bằng dòng: `Đây là bản xem trước. Thêm --apply để thực hiện.`

## 7. Helper dùng chung

### 7.1 `internal/winexec`
```go
// Run chạy lệnh có timeout, trả stdout. Lỗi gói kèm stderr.
func Run(ctx context.Context, timeout time.Duration, name string, args ...string) ([]byte, error)
```
- Dùng `exec.CommandContext`.
- Trên Windows đặt `cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}` (file `run_windows.go` với build tag) để không chớp cửa sổ khi launcher gọi.
- Output của lệnh hệ thống là codepage OEM: chỉ parse phần ASCII (số, IP, tên exe). Không parse chữ có thể bị dịch.

`tasklist.go`:
```go
type Proc struct { PID int; Name string; MemBytes int64 }
func Tasklist(ctx context.Context) ([]Proc, error)          // tasklist /fo csv /nh
func ProcName(ctx context.Context, pid int) (string, error) // tasklist /fi "PID eq N" /fo csv /nh
```
Cột bộ nhớ dạng `"412,345 K"` hoặc `"412.345 K"`: bỏ mọi ký tự không phải số rồi nhân 1024.

### 7.2 `internal/fsutil`
```go
func ParseSize(s string) (int64, error)       // "1KB","10MB","2GB","500" (byte). Không phân biệt hoa thường, cơ số 1024
func ParseAge(s string) (time.Duration, error) // "90m","12h","30d","2w","1y" (y = 365 ngày; m = phút). time.ParseDuration không có d/w/y
func FormatBytes(n int64) string               // 1536 → "1.5 KB"
var DefaultSkipDirs = []string{".git", "node_modules", "vendor", ".idea", ".vscode", "storage", "dist", "build", ".next", ".nuxt"}
```
Kèm test dạng bảng.

## 7.3 Mã lỗi dùng chung

Các spec lệnh dùng lại những mã này, định nghĩa constructor sẵn trong `internal/output/errors.go`:

| code | exit | message mẫu | hint mẫu |
|---|---|---|---|
| `ARGS_INVALID` | 2 | (lỗi parse cờ của cobra) | Xem: foxkit <lệnh> --help |
| `DIR_NOT_FOUND` | 2 | Không tìm thấy thư mục <path> | Kiểm tra lại đường dẫn |
| `FILE_NOT_FOUND` | 2 hoặc 1* | Không tìm thấy file <path> | Kiểm tra lại đường dẫn |
| `PATH_NOT_FOUND` | 2 | Không tìm thấy <path> | Kiểm tra lại đường dẫn |
| `SIZE_INVALID` | 2 | Kích thước không hợp lệ: <s> | Ví dụ: 500KB, 10MB, 2GB |
| `AGE_INVALID` | 2 | Thời lượng không hợp lệ: <s> | Ví dụ: 90m, 12h, 7d, 2w, 1y |
| `PORT_INVALID` | 2 | Port phải là số từ 1 đến 65535 | |
| `INPUT_TOO_LARGE` | 1 | Dữ liệu vào quá lớn (giới hạn N) | |
| `GIT_NOT_FOUND` | 1 | Không tìm thấy git | Cài Git for Windows hoặc kiểm tra bằng foxkit which git |
| `NEED_ADMIN` | 3 | Không đủ quyền: <việc> | Chạy lại terminal bằng quyền Administrator |
| `CANCELED` | 1 | Đã hủy (Ctrl+C) | |
| `INTERNAL` | 1 | (err.Error()) | Báo lỗi kèm lệnh đã chạy |

\* `FILE_NOT_FOUND` là exit 2 khi file là tham số chính của lệnh; là lỗi theo từng item (exit 1) khi lệnh nhận nhiều file như `hash`.

## 8. Lệnh `version`

- Cú pháp: `foxkit version [--json]`
- Biến trong `cmd/version.go`: `var version = "dev"; var commit = "none"; var built = "unknown"`.
- Build:
  ```powershell
  go build -ldflags "-s -w -X foxkit/cmd.version=0.1.0 -X foxkit/cmd.commit=$(git rev-parse --short HEAD) -X foxkit/cmd.built=$(Get-Date -Format o)" -o foxkit.exe .
  ```
- Nếu chưa có ldflags, fallback lấy `vcs.revision`/`vcs.time` từ `debug.ReadBuildInfo()`.
- Text: `foxkit 0.1.0 (a1b2c3d, 2026-11-16T10:00:00Z, go1.25.4)`
- JSON: `{ "ok": true, "data": { "version": "0.1.0", "commit": "a1b2c3d", "built": "2026-11-16T10:00:00Z", "go": "go1.25.4" } }`

## 9. Lệnh `manifest`

- Cú pháp: `foxkit manifest [--json]`
- `--json`: in `{ "ok": true, "data": <nội dung commands.json> }` (dùng `json.RawMessage`, không decode/encode lại).
- Không `--json`: in danh sách gọn `tên  —  mô tả` theo nhóm `category`.
- Lỗi khi manifest hỏng không thể xảy ra lúc chạy vì test bắt trước (mục 10).

### 9.1 Schema `commands.json`

```json
{
  "schema": 1,
  "commands": [
    {
      "name": "port",
      "title": "Kiểm tra port",
      "description": "Xem process nào đang chiếm port, có thể kill",
      "category": "dev",
      "args":  [ { "name": "port", "type": "int", "required": true, "min": 1, "max": 65535, "label": "Port" } ],
      "flags": [ { "name": "kill", "type": "bool", "default": false, "label": "Kill process", "confirm": "Kill process đang dùng port này?" } ],
      "admin": false,
      "confirm": "",
      "longRunning": false,
      "dryRun": false,
      "output": "kv"
    }
  ]
}
```

| Field | Ý nghĩa |
|---|---|
| `name` | Đường dẫn lệnh, lệnh con cách bằng dấu cách: `"hosts add"` |
| `category` | `core`, `dev`, `life`, `system`, `network` |
| `args[].type` / `flags[].type` | `string`, `text`, `int`, `bool`, `enum` (+`options`), `file`, `dir`, `list` (+`separator`, mặc định `,`) |
| `required`, `default`, `min`, `max`, `options` | Ràng buộc để launcher validate trước khi chạy |
| `multiple` | Arg nhận nhiều giá trị (`foxkit hash a.txt b.txt`) |
| `allowDir` | Với type `file`: cho chọn cả thư mục (vd `lock`) |
| `admin` | Lệnh luôn cần admin |
| `confirm` | Câu hỏi xác nhận trước khi chạy (lệnh hoặc flag) |
| `longRunning` | Output NDJSON, chạy đến khi bị dừng |
| `longRunningFlag` | Tên cờ biến lệnh thành chạy lâu khi bật (vd `"follow"` của `lara log`) |
| `dryRun` | Lệnh có cờ `--apply`; launcher chạy xem trước rồi mới hỏi áp dụng |
| `output` | `kv`, `table`, `text`, `stream` |
| `examples` | (tùy chọn, tối đa 3) ví dụ mẫu cho launcher hiện mờ phía trên ô gõ: `[ { "cmd": "port 8000 --kill", "desc": "Kill process đang chiếm port 8000" } ]`. `cmd` bắt đầu bằng đúng `name` của lệnh, không có tiền tố `foxkit` và không có `--json`. Thêm field này không đổi `schema` |

Lệnh cha chỉ để nhóm (ví dụ `hosts`, `lara`, không có `RunE`) **không** có entry riêng; chỉ lệnh chạy được mới có entry.

### 9.2 Entry cho `version` và `manifest`

```json
{ "name": "version", "title": "Phiên bản", "description": "Phiên bản foxkit", "category": "core",
  "args": [], "flags": [], "admin": false, "longRunning": false, "dryRun": false, "output": "kv" },
{ "name": "manifest", "title": "Manifest", "description": "Danh sách lệnh cho launcher", "category": "core",
  "args": [], "flags": [], "admin": false, "longRunning": false, "dryRun": false, "output": "text" }
```

## 10. `cmd/manifest_test.go` (chống lệch)

1. Decode `manifest.Raw` vào struct; fail nếu JSON hỏng hoặc `schema != 1`.
2. Duyệt đệ quy `rootCmd.Commands()`, bỏ qua `help` và lệnh ẩn, lấy **lệnh chạy được** (`cmd.Runnable()`: có `Run`/`RunE`; một lệnh như `path` vừa chạy được vừa có lệnh con thì vẫn tính), tên = `cmd.CommandPath()` bỏ tiền tố `foxkit `.
3. Fail nếu: có lệnh trong cobra mà thiếu trong manifest; có entry trong manifest mà không có trong cobra; với mỗi lệnh, tập flag cục bộ (`cmd.LocalFlags()`, trừ `help`) khác tập `flags[].name`.
4. Kiểm `category`, `type`, `output` thuộc danh sách hợp lệ ở bảng trên.
5. Nếu có `examples`: tối đa 3 phần tử, mỗi `cmd` bắt đầu bằng `name` của lệnh và không chứa `--json`.

Thông báo fail phải nói rõ cần sửa gì, ví dụ: `lệnh "uuid" có trong cobra nhưng thiếu trong internal/manifest/commands.json`.

## 11. Test & chạy thử

```powershell
gofmt -l .        # phải rỗng
go vet ./...
go test ./...
go build -o foxkit.exe .
.\foxkit version
.\foxkit version --json
.\foxkit manifest --json
.\foxkit khongco; echo $LASTEXITCODE     # lệnh lạ → exit 2
```

## 12. Tiêu chí hoàn thành

- [ ] Build/test/vet/gofmt sạch.
- [ ] `version`, `manifest` chạy đúng cả text và `--json`.
- [ ] Lỗi từ cobra (lệnh lạ, cờ sai) ra stderr với exit 2, có envelope khi `--json`.
- [ ] `manifest_test.go` fail khi cố tình xóa một entry (thử rồi hoàn tác).
- [ ] `internal/output`, `internal/fsutil` có test dạng bảng.

## 13. Khái niệm Go học được

- Module, package, `internal/` (Go cấm import `internal` từ ngoài module, giống `private` cho cả thư mục).
- `go:embed`, `-ldflags -X`.
- Kiểu lỗi tự định nghĩa + `errors.As` (≈ custom Exception class trong Laravel).
- `io.Writer` để in vào stdout hoặc buffer khi test (≈ dependency injection).
- Build tag `_windows.go`.
