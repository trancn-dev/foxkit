# Spec: `foxkit sync`

> Nhóm: **life** · Giai đoạn: GĐ7 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md), [life/dup.md](dup.md) (`fsutil`, worker pool), [life/rename.md](rename.md) (mẫu plan/apply)

## 1. Mục đích
Đồng bộ **một chiều** thư mục A → B (backup dự án sang ổ ngoài, USB, NAS): chỉ chép file mới/đã đổi. Giống `robocopy /MIR` nhưng an toàn hơn: mặc định xem trước, và không xóa gì ở B trừ khi bảo xóa.

## 2. Cú pháp
```
foxkit sync <src> <dst> [--delete] [--exclude pat,...] [--no-default-excludes] [--compare size|hash] [--apply] [--json]
```
| Cờ | Mặc định | Ghi chú |
|---|---|---|
| `--delete` | false | xóa ở B những file không còn ở A (cần `--apply`; xóa vào thư mục `.foxkit-trash` trong B thay vì xóa thật, xem 5.6) |
| `--exclude` | | glob thêm vào danh sách loại |
| `--no-default-excludes` | false | tắt danh sách loại mặc định (`Thumbs.db`, `desktop.ini`, `*.tmp`, `$RECYCLE.BIN`, `System Volume Information`); xem 5.1 |
| `--compare` | `size` | `size` = khác size hoặc mtime lệch > 2s; `hash` = so sha256 (chậm, chắc chắn) |
| `--apply` | false | thực hiện |

## 3. File
```
cmd/sync.go
internal/sync/plan.go
internal/sync/apply.go
internal/sync/sync_test.go
```
(Đặt tên package `syncdir` để không trùng package chuẩn `sync`.)

## 4. Hợp đồng
```go
package syncdir
type Action struct {
    Op     string `json:"op"`     // copy|update|delete|mkdir
    Path   string `json:"path"`   // tương đối
    Bytes  int64  `json:"bytes"`
    Reason string `json:"reason,omitempty"` // "mới", "size khác", "mtime khác", "hash khác"
}
type Plan struct {
    Applied bool     `json:"applied"`
    Src     string   `json:"src"`
    Dst     string   `json:"dst"`
    Actions []Action `json:"actions"`
    Same    int      `json:"same"`
    CopyBytes int64  `json:"copyBytes"`
    Failed  []Failure `json:"failed"`
}
func Build(ctx context.Context, src, dst string, opt Options) (Plan, error)
func Apply(ctx context.Context, p *Plan, opt Options, emit func(Event)) error
```

## 5. Cách hoạt động
1. **Loại mặc định**: `Thumbs.db`, `desktop.ini`, `*.tmp`, `$RECYCLE.BIN`, `System Volume Information`. `node_modules`/`vendor`/`.git` **mặc định được chép** (backup phải đủ); muốn bỏ thì `--exclude node_modules,vendor`. (Khác `zip pack` vì mục đích khác.)
2. **An toàn đường dẫn**: `src` và `dst` không được trùng nhau, không được lồng nhau (một cái nằm trong cái kia) → `SYNC_OVERLAP`. `dst` là gốc ổ đĩa hoặc `C:\Windows`, `C:\Program Files`, thư mục Users gốc → `SYNC_DST_UNSAFE`.
3. **Build**: duyệt hai cây (song song), so theo đường dẫn tương đối không phân biệt hoa thường:
   - Chỉ ở A → `copy`; khác → `update`; chỉ ở B → `delete` (chỉ khi `--delete`).
   - So `size` + `mtime` (làm tròn 2 giây vì FAT/exFAT lưu mtime thô). `--compare hash` → băm cả hai khi size bằng nhau.
   - Thư mục thiếu ở B → `mkdir`.
4. Exit 0 kể cả khi còn việc cần làm ở chế độ xem trước.
5. **Apply**:
   - Chép bằng `os.Open` + `io.Copy` vào `<tên>.foxkit-tmp` trong thư mục đích rồi `os.Rename` đè (file đích không bao giờ ở trạng thái dở dang).
   - Giữ `mtime` nguồn bằng `os.Chtimes`.
   - Worker pool 4 luồng (I/O ổ ngoài không nhanh hơn nhiều với số luồng lớn).
   - Lỗi từng file (đang bị khóa, hết chỗ, quyền) → `Failed`, tiếp tục; exit 1 `SYNC_PARTIAL` cuối cùng.
   - Hết dung lượng đích → kiểm trước: `CopyBytes` > dung lượng trống của ổ đích → `DISK_FULL` ngay từ bước xem trước (dùng `fsutil` hoặc `disk` ở GĐ6; trước GĐ6 bỏ qua kiểm tra này).
6. **`--delete`**: file thừa ở B được **chuyển vào `B\.foxkit-trash\<yyyyMMdd-HHmmss>\<đường dẫn>`** thay vì xóa thật, để sửa sai được; thư mục trash tự bị loại khỏi lần so sánh sau. Thêm `--purge-trash <ngày>` ở bản sau nếu cần (ngoài phạm vi).
7. Hủy bằng Ctrl+C: dừng gọn, không để file `.foxkit-tmp` (dọn khi thoát).
8. Tiến độ ra stderr (số file, byte, tốc độ) khi là terminal; `--json` + `--apply` in NDJSON (`longRunning`).

## 6. Output
```
Nguồn  D:\Projects        Đích  E:\Backup\Projects
  + 120 file mới (340 MB)
  ~  14 file đã đổi (22 MB)
  =  8.301 file giống nhau
  −   9 file chỉ có ở đích (giữ nguyên; dùng --delete để dọn)
Đây là bản xem trước. Thêm --apply để thực hiện.
```
JSON: `{ "ok": true, "data": <Plan> }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `DIR_NOT_FOUND` | src không có | 2 | |
| `SYNC_OVERLAP` | | 2 | Nguồn và đích không được trùng hoặc lồng nhau |
| `SYNC_DST_UNSAFE` | | 2 | Chọn một thư mục con riêng cho bản sao, ví dụ E:\Backup\Projects |
| `DISK_FULL` | | 1 | Đích không đủ chỗ: cần <X>, còn <Y> |
| `SYNC_PARTIAL` | có file lỗi | 1 | Xem mục failed; đóng chương trình đang dùng file rồi chạy lại |

## 8. Manifest
```json
{ "name": "sync", "title": "Đồng bộ thư mục", "description": "Sao lưu một chiều A → B, chỉ chép file mới/đổi", "category": "life",
  "args": [
    { "name": "src", "type": "dir", "required": true, "label": "Nguồn" },
    { "name": "dst", "type": "dir", "required": true, "label": "Đích" }
  ],
  "flags": [
    { "name": "delete", "type": "bool", "default": false, "label": "Dọn file thừa ở đích (vào thùng rác foxkit)" },
    { "name": "exclude", "type": "list", "label": "Loại" },
    { "name": "compare", "type": "enum", "options": ["size","hash"], "default": "size", "label": "Cách so sánh" },
    { "name": "apply", "type": "bool", "default": false, "label": "Thực hiện", "confirm": "Bắt đầu sao chép?" }
  ], "admin": false, "longRunning": false, "longRunningFlag": "apply", "dryRun": true, "output": "table" }
```

## 9. Test
Cây tạm A/B:
- B rỗng → toàn bộ `copy`; sau `Apply` hai cây giống nhau (hash từng file).
- Chạy lại ngay → `Actions` rỗng (idempotent).
- Sửa 1 file ở A (giữ size, đổi nội dung, đổi mtime) → `update`; `--compare hash` bắt được cả khi mtime không đổi.
- File thừa ở B không bị đụng khi không có `--delete`; có `--delete` → nằm trong `.foxkit-trash`, không mất.
- mtime lệch 1 giây (giả FAT) → coi là giống.
- `SYNC_OVERLAP` với dst nằm trong src.
- File nguồn bị khóa (mở sẵn trong test) → `Failed`, các file khác vẫn chép.
- Giả lỗi giữa chừng: không còn `.foxkit-tmp`.

Chạy thử:
```powershell
.\foxkit sync D:\Projects E:\Backup\Projects
.\foxkit sync D:\Projects E:\Backup\Projects --exclude node_modules,vendor --apply
.\foxkit sync D:\Projects E:\Backup\Projects --exclude node_modules,vendor --delete --apply
```

## 10. Tiêu chí hoàn thành
Tiêu chí chung + test idempotent + thử trên USB thật (FAT/exFAT) không báo "khác" liên tục do mtime.

## 11. Ngoài phạm vi
Đồng bộ hai chiều, theo dõi thay đổi liên tục, lập lịch, mã hóa, bản sao theo phiên bản (snapshot), chép quyền/ACL/alternate streams.

## 12. Go học được
Kiểu thiết kế plan/apply lần thứ ba (đã quen từ `rename`, `prune`), ghi nguyên tử bằng tệp tạm, so sánh `time.Time` với dung sai, đặt tên package tránh trùng thư viện chuẩn, worker pool với thao tác I/O.
