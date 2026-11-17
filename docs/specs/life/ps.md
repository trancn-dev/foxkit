# Spec: `foxkit ps`

> Nhóm: **life** · Giai đoạn: GĐ3 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md) (`internal/winexec/tasklist.go`)

## 1. Mục đích
Liệt kê process đang chạy, lọc theo tên, sắp theo RAM. Gộp các process cùng tên (Chrome, node có hàng chục process) để thấy ai ăn RAM nhất.

## 2. Cú pháp
```
foxkit ps [filter] [--sort mem|name|pid] [--top 30] [--group] [--json]
```
| Tham số/cờ | Mặc định | Ghi chú |
|---|---|---|
| `[filter]` | | chuỗi con của tên, không phân biệt hoa thường (`node`, `php`) |
| `--sort` | `mem` | |
| `--top` | 30 | 0 = tất cả |
| `--group` | false | gộp theo tên: cộng RAM, đếm số process, liệt kê PID |

## 3. File
```
cmd/ps.go
internal/ps/ps.go
internal/ps/ps_test.go
internal/winexec/tasklist.go   # đã có từ GĐ0/port
```

## 4. Hợp đồng
```go
package ps
type Proc struct {
    PID      int    `json:"pid"`
    Name     string `json:"name"`
    MemBytes int64  `json:"memBytes"`
    Session  string `json:"session"` // "Console" / "Services"
}
type Group struct {
    Name     string `json:"name"`
    Count    int    `json:"count"`
    MemBytes int64  `json:"memBytes"`
    PIDs     []int  `json:"pids"`
}
func List(ctx context.Context) ([]Proc, error)
func Filter(ps []Proc, substr string) []Proc
func GroupByName(ps []Proc) []Group
```

## 5. Cách hoạt động
1. `tasklist /fo csv /nh` (timeout 10s) → `encoding/csv`. Cột: `Image Name, PID, Session Name, Session#, Mem Usage`.
2. Mem: bỏ mọi ký tự không phải số (`"412,345 K"`, `"412.345 K"`, có thể có khoảng trắng không ngắt ` `) → ×1024.
3. Output của `tasklist` là codepage OEM: tên process chỉ có ASCII thì đúng; tên Unicode có thể sai ký tự, chấp nhận ở v1 (GĐ6 có thể đổi sang API `CreateToolhelp32Snapshot` qua x/sys).
4. Lọc, gộp, sắp, cắt top. `sort.SliceStable` để thứ tự ổn định khi bằng nhau.
5. Tổng RAM các process hiển thị ở cuối.

## 6. Output
```
PID     NAME               MEM
14820   chrome.exe         612.4 MB
9812    node.exe           401.0 MB
...
30/214 process · tổng 7.8 GB
```
`--group`:
```
NAME          COUNT   MEM
chrome.exe    38      2.9 GB
node.exe      6       1.1 GB
```
JSON: `{ "ok": true, "data": { "items": [Proc...] , "total": 214 } }` hoặc với `--group`: `{ "groups": [Group...], "total": 214 }`.

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `TASKLIST_FAILED` | tasklist lỗi | 1 | Thử chạy "tasklist" trong terminal |
| `SORT_INVALID` | | 2 | Chọn: mem, name, pid |

## 8. Manifest
```json
{ "name": "ps", "title": "Danh sách process", "description": "Process đang chạy, sắp theo RAM, gộp theo tên", "category": "system",
  "args": [ { "name": "filter", "type": "string", "required": false, "label": "Lọc theo tên" } ],
  "flags": [
    { "name": "sort", "type": "enum", "options": ["mem","name","pid"], "default": "mem", "label": "Sắp xếp" },
    { "name": "top", "type": "int", "default": 30, "min": 0, "label": "Số dòng (0 = tất cả)" },
    { "name": "group", "type": "bool", "default": false, "label": "Gộp theo tên" }
  ], "admin": false, "longRunning": false, "dryRun": false, "output": "table" }
```

## 9. Test
Parser với CSV mẫu (fixture trong `testdata/tasklist_en.csv`, `tasklist_de.csv` có dấu chấm ngăn cách nghìn): đúng PID, mem. Gộp: 3 dòng `node.exe` → 1 group count 3. Filter `NODE` khớp `node.exe`.

Chạy thử: `.\foxkit ps`, `.\foxkit ps node`, `.\foxkit ps --group --top 10 --json`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + parser test.

## 11. Ngoài phạm vi
CPU%, cây cha-con, đường dẫn exe, cập nhật liên tục kiểu `top`.

## 12. Go học được
`encoding/csv`, gom nhóm bằng `map[string]*Group`, `sort.SliceStable`, fixture trong `testdata/`.
