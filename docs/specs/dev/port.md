# Spec: `foxkit port`

> Nhóm: **dev** · Giai đoạn: GĐ1 · Trạng thái: chưa làm
> Tài liệu này để giao cho Claude Code triển khai. Cần làm xong [core/foundation.md](../core/foundation.md) trước. Làm đúng phạm vi bên dưới, không thêm tính năng ngoài spec.

---

## 1. Mục đích

Trả lời câu hỏi "port X đang bị process nào chiếm?" và cho phép kill process đó.
Tình huống điển hình: `php artisan serve` báo port 8000 đã được dùng, hoặc `npm run dev` báo port 5173 bận.

## 2. Cú pháp

```
foxkit port <port> [--kill] [--json]
foxkit port --list [--json]
```

| Tham số / cờ | Kiểu | Bắt buộc | Mặc định | Mô tả |
|---|---|---|---|---|
| `<port>` | int 1–65535 | Có (trừ khi `--list`) | — | Port cần kiểm tra |
| `--kill` | bool | Không | `false` | Kill process đang chiếm port |
| `--list` | bool | Không | `false` | Liệt kê mọi port TCP đang LISTEN |
| `--json` | bool (cờ toàn cục) | Không | `false` | Xuất JSON envelope |

Ràng buộc:
- `--list` không đi cùng `<port>` hoặc `--kill`, vi phạm thì trả lỗi sai tham số (exit 2).
- Chỉ xét **TCP LISTEN** (cả IPv4 và IPv6). UDP nằm ngoài phạm vi v1.

## 3. Vị trí file

```
cmd/port.go                 # cobra command: parse & validate tham số, gọi internal/port, in kết quả
internal/port/port.go       # logic: Find, List, Kill
internal/port/netstat.go    # parse output netstat -ano
internal/port/netstat_test.go
internal/winexec/tasklist.go  # tra tên/đường dẫn process theo PID (dùng lại cho ps, kill)
internal/manifest/commands.json  # thêm entry "port"
```

`cmd/port.go` **không** chứa logic: chỉ parse tham số, gọi `internal/port`, rồi đưa kết quả cho `internal/output`.

## 4. Kiểu dữ liệu (hợp đồng giữa các tầng)

```go
// internal/port
type Listener struct {
    Port     int    `json:"port"`
    Protocol string `json:"protocol"`          // "TCP"
    Address  string `json:"address"`           // "0.0.0.0", "127.0.0.1", "::"
    PID      int    `json:"pid"`
    Process  string `json:"process,omitempty"` // "php.exe"
    Path     string `json:"path,omitempty"`    // có thể rỗng nếu không đủ quyền
}

type Result struct {
    Port      int        `json:"port"`
    InUse     bool       `json:"inUse"`
    Listeners []Listener `json:"listeners"`         // 1 port có thể có nhiều dòng (IPv4 + IPv6)
    Killed    []int      `json:"killed,omitempty"`  // PID đã kill
}

func Find(ctx context.Context, port int) (Result, error)
func List(ctx context.Context) ([]Listener, error)
func Kill(pid int) error
```

## 5. Cách hoạt động

### 5.1 `Find(port)`
1. Chạy `netstat -ano -p TCP` và `netstat -ano -p TCPv6` (hoặc chỉ `netstat -ano` rồi lọc dòng bắt đầu bằng `TCP`). Timeout 5 giây qua `exec.CommandContext`.
2. Với mỗi dòng dữ liệu, tách bằng `strings.Fields`, cần đúng 5 cột: `Proto  LocalAddr  RemoteAddr  State  PID`.
3. Dòng là LISTEN khi **RemoteAddr là `0.0.0.0:0` hoặc `[::]:0`**. **Không** so chữ `LISTENING` vì cột State có thể bị dịch theo ngôn ngữ Windows.
4. Tách port từ LocalAddr bằng `strings.LastIndex(addr, ":")` (IPv6 có dạng `[::1]:8000`), bỏ `[]` khỏi phần address.
5. Lọc theo port cần tìm, loại các PID trùng nhau.
6. Với mỗi PID, tra tên process: `tasklist /fi "PID eq <pid>" /fo csv /nh`, đọc bằng `encoding/csv`, cột 0 là tên. Không tra được (process vừa thoát) thì để `Process` rỗng, không coi là lỗi.
7. Đường dẫn exe: thư viện chuẩn không có API lấy đường dẫn theo PID, nên **v1 để trống `Path`**. GĐ6 sẽ bổ sung qua `QueryFullProcessImageName`. (Không thêm dependency ở GĐ1.)
8. Không có dòng nào thì `InUse=false` và **exit 0** (đây là kết quả hợp lệ, không phải lỗi).

### 5.2 `--kill`
1. Gọi `Find` trước.
2. Port trống: in thông báo port đang trống, exit 0, không làm gì thêm.
3. Với mỗi PID: chặn PID `0` và `4` (System), trả lỗi `PORT_SYSTEM_PROCESS`.
4. `os.FindProcess(pid)` rồi `.Kill()`.
5. Lỗi access denied (`errors.Is(err, os.ErrPermission)`, hoặc `syscall.Errno(5)` là ERROR_ACCESS_DENIED) thì trả `NEED_ADMIN`, exit 3.
6. Sau khi kill, chờ tối đa 2 giây (lặp `Find` mỗi 200ms) để xác nhận port đã trống, rồi điền `Killed`.

### 5.3 `--list`
Như bước 1–6 của `Find` nhưng không lọc theo port; sắp xếp tăng dần theo port.

## 6. Output

### 6.1 Text (không có `--json`)

```
> foxkit port 8000
Port 8000 đang được dùng:
  PID 12345  php.exe  127.0.0.1

> foxkit port 8000 --kill
Đã kill PID 12345 (php.exe). Port 8000 giờ đã trống.

> foxkit port 3000
Port 3000 đang trống.

> foxkit port --list
PORT   PID    PROCESS        ADDRESS
80     4      System         0.0.0.0
3306   2210   mysqld.exe     0.0.0.0
8000   12345  php.exe        127.0.0.1
```

Bảng dùng `text/tabwriter`.

### 6.2 JSON

```json
// foxkit port 8000 --json   (stdout, exit 0)
{ "ok": true, "data": {
  "port": 8000, "inUse": true,
  "listeners": [ { "port": 8000, "protocol": "TCP", "address": "127.0.0.1", "pid": 12345, "process": "php.exe" } ]
} }

// foxkit port 8000 --kill --json
{ "ok": true, "data": { "port": 8000, "inUse": false, "listeners": [], "killed": [12345] } }

// foxkit port --list --json
{ "ok": true, "data": { "items": [ { "port": 3306, "protocol": "TCP", "address": "0.0.0.0", "pid": 2210, "process": "mysqld.exe" } ] } }
```

`listeners` luôn là mảng (rỗng `[]`, không phải `null`), để launcher khỏi phải kiểm tra null.

## 7. Lỗi

Lỗi in ra **stderr**. Có `--json` thì in envelope `{ "ok": false, "error": {...} }`.

| code | Khi nào | exit | message | hint |
|---|---|---|---|---|
| `PORT_INVALID` | port không phải số hoặc ngoài 1–65535 | 2 | Port phải là số từ 1 đến 65535 | Ví dụ: foxkit port 8000 |
| `ARGS_CONFLICT` | `--list` đi cùng `<port>`/`--kill` | 2 | --list không dùng chung với port hoặc --kill | Dùng: foxkit port --list |
| `PORT_MISSING` | không có port và không có `--list` | 2 | Thiếu số port | Ví dụ: foxkit port 8000 |
| `NETSTAT_FAILED` | netstat lỗi hoặc timeout | 1 | Không đọc được danh sách kết nối | Thử chạy "netstat -ano" trong terminal để kiểm tra |
| `PORT_SYSTEM_PROCESS` | PID 0/4 | 1 | Port do Windows (System) giữ, không thể kill | Thường do IIS/http.sys: tắt dịch vụ tương ứng hoặc đổi port |
| `NEED_ADMIN` | kill bị từ chối quyền | 3 | Không đủ quyền kill PID 12345 | Chạy lại terminal bằng quyền Administrator |

## 8. Entry trong `internal/manifest/commands.json`

```json
{
  "name": "port",
  "title": "Kiểm tra port",
  "description": "Xem process nào đang chiếm port, có thể kill",
  "category": "dev",
  "args": [
    { "name": "port", "type": "int", "required": false, "min": 1, "max": 65535, "label": "Port" }
  ],
  "flags": [
    { "name": "kill", "type": "bool", "default": false, "label": "Kill process", "confirm": "Kill process đang dùng port này?" },
    { "name": "list", "type": "bool", "default": false, "label": "Liệt kê mọi port đang mở" }
  ],
  "admin": false,
  "longRunning": false,
  "output": "kv"
}
```

## 9. Test

### 9.1 Unit test (`internal/port/netstat_test.go`, dạng bảng)
Parser nhận string mẫu, không gọi netstat thật. Các case:

| Case | Input (một dòng) | Kỳ vọng |
|---|---|---|
| IPv4 listen | `TCP    127.0.0.1:8000    0.0.0.0:0    LISTENING    12345` | port 8000, addr 127.0.0.1, pid 12345 |
| IPv6 listen | `TCP    [::]:5173    [::]:0    LISTENING    9812` | port 5173, addr `::` |
| State bị dịch | `TCP    0.0.0.0:80    0.0.0.0:0    ABHÖREN    4` | vẫn nhận là listen |
| Không phải listen | `TCP    127.0.0.1:52000    127.0.0.1:8000    ESTABLISHED    777` | bỏ qua |
| Dòng header/rác | `  Proto  Local Address ...` / dòng trống | bỏ qua, không lỗi |
| Trùng PID | 2 dòng IPv4 + IPv6 cùng PID | `Listeners` có 2 phần tử, tra tasklist 1 lần |

Thêm test cho parse port từ tham số: `"8000"` ok, `"0"`, `"70000"`, `"abc"` → `PORT_INVALID`.

### 9.2 Chạy thử thủ công (Windows)

```powershell
go build -o foxkit.exe .

# 1. Mở một server để chiếm port
python -m http.server 8000      # hoặc: php -S 127.0.0.1:8000

# 2. Ở terminal khác
.\foxkit port 8000
.\foxkit port 8000 --json
.\foxkit port 8000 --kill
.\foxkit port 8000              # phải báo đang trống
.\foxkit port --list
.\foxkit port 70000; echo $LASTEXITCODE   # phải ra 2
.\foxkit port 80 --kill                   # nếu System giữ port 80 → PORT_SYSTEM_PROCESS
```

## 10. Tiêu chí hoàn thành

- [ ] `go build`, `go vet`, `gofmt -l .` sạch; `go test ./...` pass.
- [ ] Ba chế độ (`<port>`, `--kill`, `--list`) chạy đúng như mục 6 trên Windows.
- [ ] Mọi lỗi ở mục 7 có đúng code, exit code, ra stderr.
- [ ] `--json` luôn là JSON hợp lệ, `listeners`/`items` không bao giờ là `null`.
- [ ] Entry `port` có trong `internal/manifest/commands.json`, test chống lệch manifest pass.
- [ ] Không thêm dependency mới.

## 11. Ngoài phạm vi (không làm ở v1)

UDP; lọc theo address; đường dẫn exe (để GĐ6); kill cả cây process; tự động đổi port.

## 12. Khái niệm Go học được

- `os/exec` + `exec.CommandContext` có timeout (≈ `Symfony\Process` có `setTimeout`).
- Parse text với `strings.Fields`, `strconv.Atoi`, xử lý lỗi từng bước thay vì try/catch.
- `encoding/csv` đọc output `tasklist`.
- `errors.Is` để nhận lỗi quyền (≈ `catch (PermissionException $e)`).
- Test dạng bảng (≈ `@dataProvider` của PHPUnit), tách parser khỏi phần gọi lệnh để test được mà không cần Windows.
