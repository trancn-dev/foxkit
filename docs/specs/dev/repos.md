# Spec: `foxkit repos`

> Nhóm: **dev** · Giai đoạn: GĐ5 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)
> Cần `git` trong PATH.

## 1. Mục đích
Quét thư mục chứa nhiều repo và cho biết repo nào đang có thay đổi chưa commit, chưa push, bị tụt so với remote. Dùng cuối ngày ("còn quên commit gì không?") hoặc trước khi cài lại máy.

## 2. Cú pháp
```
foxkit repos <dir> [--depth 3] [--dirty] [--fetch] [--json]
```
| Cờ | Mặc định | Ghi chú |
|---|---|---|
| `--depth` | 3 | độ sâu tìm thư mục `.git` |
| `--dirty` | false | chỉ hiện repo có vấn đề (dirty, ahead, behind, không có upstream) |
| `--fetch` | false | chạy `git fetch --quiet` trước để behind chính xác (chậm, cần mạng) |

## 3. File
```
cmd/repos.go
internal/git/git.go        # helper chạy git, dùng chung với branches
internal/git/status.go     # parse porcelain v2
internal/repos/repos.go
internal/git/status_test.go
```

## 4. Hợp đồng
```go
package git
func Run(ctx context.Context, dir string, args ...string) ([]byte, error) // git -C <dir> ...; lỗi gồm stderr
type Status struct {
    Branch    string `json:"branch"`     // "(detached)" nếu detached
    Upstream  string `json:"upstream"`   // "origin/main" hoặc ""
    Ahead     int    `json:"ahead"`
    Behind    int    `json:"behind"`
    Staged    int    `json:"staged"`
    Modified  int    `json:"modified"`
    Untracked int    `json:"untracked"`
    Conflicts int    `json:"conflicts"`
    Stashes   int    `json:"stashes"`
}
func ParseStatusV2(out []byte) (Status, error)

package repos
type Repo struct {
    Path   string     `json:"path"`
    Name   string     `json:"name"`
    git.Status
    LastCommit string `json:"lastCommit"` // ISO
    Error  string     `json:"error,omitempty"`
}
func Scan(ctx context.Context, dir string, depth int, fetch bool) ([]Repo, error)
```

## 5. Cách hoạt động
1. Kiểm tra `git` có trong PATH (`exec.LookPath`), không có → `GIT_NOT_FOUND`.
2. Tìm repo: `WalkDir` tới `--depth`; thấy thư mục chứa `.git` (thư mục hoặc **file** — worktree/submodule dùng file `.git`) → ghi nhận và `SkipDir` (không quét repo lồng). Bỏ `node_modules`, `vendor`.
3. Với mỗi repo (tối đa 8 song song bằng semaphore channel):
   - `--fetch`: `git fetch --quiet --no-tags` timeout 20s; lỗi (không mạng, cần đăng nhập) → ghi `Error`, vẫn tiếp tục lấy status local.
   - `git status --porcelain=v2 --branch` — định dạng dành cho máy, ổn định qua các phiên bản:
     - `# branch.head <name>` · `# branch.upstream <u>` · `# branch.ab +A -B`
     - dòng bắt đầu `1 ` hoặc `2 `: cột XY; X != `.` → staged++, Y != `.` → modified++
     - `u ` → conflicts++ · `? ` → untracked++
   - `git stash list` → đếm dòng.
   - `git log -1 --format=%cI` → LastCommit (repo chưa có commit → rỗng).
   - Đặt `GIT_OPTIONAL_LOCKS=0` trong env của lệnh git để không tạo lock khi chỉ đọc (tránh xung đột với IDE đang mở repo).
4. Sắp: repo có vấn đề trước, rồi theo tên.
5. Exit luôn 0 (báo cáo); lỗi từng repo nằm trong `Error`.

## 6. Output
```
REPO          BRANCH            TRẠNG THÁI
shop-api      feature/coupon    ✎ 3 sửa · + 1 mới · ↑2 chưa push
crm           main              ↓5 tụt so với origin/main
blog          main              ✓ sạch
tools         spike/x           ⚠ chưa có upstream · 1 stash
12 repo · 3 cần chú ý
```
JSON: `{ "ok": true, "data": { "items": [Repo...], "attention": 3 } }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `GIT_NOT_FOUND` | | 1 | Cài Git for Windows hoặc kiểm tra bằng foxkit which git |
| `DIR_NOT_FOUND` | | 2 | |

## 8. Manifest
```json
{ "name": "repos", "title": "Trạng thái các repo", "description": "Repo nào chưa commit, chưa push, bị tụt", "category": "dev",
  "args": [ { "name": "dir", "type": "dir", "required": true, "label": "Thư mục chứa repo" } ],
  "flags": [
    { "name": "depth", "type": "int", "default": 3, "min": 1, "max": 6, "label": "Độ sâu" },
    { "name": "dirty", "type": "bool", "default": false, "label": "Chỉ repo cần chú ý" },
    { "name": "fetch", "type": "bool", "default": false, "label": "Fetch trước" }
  ], "admin": false, "longRunning": false, "dryRun": false, "output": "table" }
```

## 9. Test
`ParseStatusV2` với output mẫu (bảng): sạch; có upstream +2 -1; staged + modified cùng file (`1 MM`); untracked; conflict `u UU`; detached `# branch.head (detached)`; chưa có commit `# branch.oid (initial)`.
Tích hợp (skip nếu không có git): `git init` repo tạm, tạo file → untracked 1.

Chạy thử: `.\foxkit repos D:\Projects`, `.\foxkit repos D:\Projects --dirty --fetch --json`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + parser test.

## 11. Ngoài phạm vi
Tự commit/push/pull, giao diện diff.

## 12. Go học được
Package helper dùng chung (`internal/git`), `exec.Cmd.Env = append(os.Environ(), ...)`, parse định dạng dòng có tiền tố, semaphore giới hạn số process con.
