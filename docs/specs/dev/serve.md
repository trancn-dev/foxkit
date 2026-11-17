# Spec: `foxkit serve`

> Nhóm: **dev** · Giai đoạn: GĐ4 · Trạng thái: chưa làm · Cần trước: [life/ip.md](../life/ip.md) (`internal/netinfo`)

## 1. Mục đích
Chạy static file server tức thì cho một thư mục: xem thử bản build Vue (`dist/`), chia file trong LAN, test HTML tĩnh. Có chế độ SPA cho Vue Router history mode.

## 2. Cú pháp
```
foxkit serve [dir] [--port 8080] [--host 0.0.0.0] [--spa] [--cors] [--no-cache] [--open] [--list] [--json]
```
| Cờ | Mặc định | Ghi chú |
|---|---|---|
| `[dir]` | `.` | |
| `--port` | 8080 | đang bận → tự thử port kế tiếp (tối đa +20), in rõ port thực tế; `--port 0` = hệ điều hành chọn |
| `--host` | `0.0.0.0` | `127.0.0.1` để chỉ máy mình truy cập |
| `--spa` | false | path không phải file → trả `index.html` |
| `--cors` | false | thêm `Access-Control-Allow-Origin: *` và xử lý `OPTIONS` |
| `--no-cache` | true | `Cache-Control: no-store` (đang dev nên mặc định tắt cache) |
| `--open` | false | mở trình duyệt mặc định (`rundll32 url.dll,FileProtocolHandler <url>`) |
| `--list` | false | cho phép liệt kê thư mục (mặc định tắt) |

`longRunning: true`.

## 3. File
```
cmd/serve.go
internal/serve/server.go
internal/serve/middleware.go
internal/serve/serve_test.go
```

## 4. Hợp đồng
```go
package serve
type Options struct { Dir, Host string; Port int; SPA, CORS, NoCache, List bool }
type Event struct {
    Event  string   `json:"event"`            // start, request, error, stop
    Time   string   `json:"time"`
    URLs   []string `json:"urls,omitempty"`   // start
    Method string   `json:"method,omitempty"`
    Path   string   `json:"path,omitempty"`
    Status int      `json:"status,omitempty"`
    Bytes  int64    `json:"bytes,omitempty"`
    Ms     int64    `json:"ms,omitempty"`
    Remote string   `json:"remote,omitempty"`
}
func Handler(opt Options) http.Handler
func Run(ctx context.Context, opt Options, emit func(Event)) error
```

## 5. Cách hoạt động
1. `root, _ := filepath.Abs(dir)`, kiểm tra là thư mục.
2. `fs := http.FileServer(http.Dir(root))`. `http.Dir` đã chặn `..` thoát ra ngoài root.
3. Middleware xếp lớp (giống middleware Laravel):
   ```
   logging( cors?( noCache?( noList?( spa?( fs ) ) ) ) )
   ```
   - `logging`: bọc `http.ResponseWriter` bằng struct tự viết ghi lại status và số byte; sau khi handler chạy xong → `emit(Event{Event:"request",...})`.
   - `noList`: path kết thúc `/` và không có `index.html` → 404.
   - `spa`: `os.Stat(filepath.Join(root, path))` không tồn tại và path **không có đuôi file** (không phải `/app.js`) → phục vụ `index.html`. File có đuôi mà thiếu → 404 thật (để thấy lỗi asset).
   - Không bao giờ phục vụ file/thư mục bắt đầu bằng `.` (`.env`, `.git`) → 404.
4. Listen: `net.Listen("tcp", host:port)`; lỗi "address in use" → thử port+1.
5. `emit(start)` với URL: `http://localhost:<port>` + mọi IP LAN từ `netinfo.LANAddrs` (khi host là `0.0.0.0`).
6. `ctx` từ `signal.NotifyContext(context.Background(), os.Interrupt)`; khi `ctx.Done()` → `srv.Shutdown(ctx 5s)` → `emit(stop)`.
7. Text mode: in log mỗi request một dòng `19:20:01  200  GET /app.js  12 KB  2ms`. `--json`: mỗi `Event` một dòng NDJSON qua `Printer.Event`.
8. Windows Firewall có thể hỏi cho phép lần đầu khi listen `0.0.0.0` — ghi chú trong output start: "Nếu điện thoại không vào được, cho phép foxkit trong Windows Firewall".

## 6. Output
Text:
```
Đang phục vụ D:\Projects\shop\dist  (SPA)
  http://localhost:8080
  http://192.168.1.23:8080
Nhấn Ctrl+C để dừng.
19:20:01  200  GET  /             1.2 KB   3ms
19:20:01  200  GET  /assets/app.js 182 KB  4ms
```
NDJSON:
```
{"event":"start","time":"...","urls":["http://localhost:8080","http://192.168.1.23:8080"]}
{"event":"request","time":"...","method":"GET","path":"/","status":200,"bytes":1234,"ms":3,"remote":"192.168.1.40"}
{"event":"stop","time":"..."}
```

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `DIR_NOT_FOUND` | | 2 | |
| `PORT_INVALID` | | 2 | |
| `SERVE_NO_PORT` | 20 port liên tiếp đều bận | 1 | Chỉ định --port khác hoặc xem foxkit port --list |
| `SERVE_SPA_NO_INDEX` | `--spa` mà không có index.html | 2 | Thư mục build Vue thường là dist/ |

## 8. Manifest
```json
{ "name": "serve", "title": "Static server", "description": "Chạy web server tĩnh cho thư mục, hỗ trợ SPA", "category": "dev",
  "args": [ { "name": "dir", "type": "dir", "required": false, "label": "Thư mục" } ],
  "flags": [
    { "name": "port", "type": "int", "default": 8080, "min": 0, "max": 65535, "label": "Port" },
    { "name": "host", "type": "string", "default": "0.0.0.0", "label": "Host" },
    { "name": "spa", "type": "bool", "default": false, "label": "Chế độ SPA" },
    { "name": "cors", "type": "bool", "default": false, "label": "Bật CORS" },
    { "name": "no-cache", "type": "bool", "default": true, "label": "Tắt cache" },
    { "name": "open", "type": "bool", "default": false, "label": "Mở trình duyệt" },
    { "name": "list", "type": "bool", "default": false, "label": "Cho liệt kê thư mục" }
  ], "admin": false, "longRunning": true, "dryRun": false, "output": "stream" }
```

## 9. Test
`httptest.NewServer(Handler(opt))` với thư mục tạm có `index.html`, `assets/app.js`, `.env`:
- `/` → index; `/assets/app.js` → 200.
- `--spa`: `/products/12` → index; `/missing.js` → 404.
- `/.env` → 404 (kể cả `/%2eenv`).
- Không `--list`: `/assets/` → 404.
- `--cors`: có header; `OPTIONS` → 204.
- Logging middleware ghi đúng status 404.

Chạy thử:
```powershell
cd D:\Projects\shop; npm run build
.\foxkit serve dist --spa --open
.\foxkit serve . --port 9000 --json
```

## 10. Tiêu chí hoàn thành
Tiêu chí chung + Ctrl+C tắt gọn (in sự kiện stop, exit 0) + điện thoại cùng Wi-Fi mở được.

## 11. Ngoài phạm vi
HTTPS, live reload, proxy API, upload file, nén gzip.

## 12. Go học được
`http.Handler`/`http.HandlerFunc`, middleware bằng hàm bọc hàm (≈ middleware Laravel `handle($request, $next)`), embed struct để ghi đè `WriteHeader`, `signal.NotifyContext`, `Server.Shutdown`.
