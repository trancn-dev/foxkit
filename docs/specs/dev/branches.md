# Spec: `foxkit branches`

> Nhóm: **dev** · Giai đoạn: GĐ5 · Trạng thái: chưa làm · Cần trước: [dev/repos.md](repos.md) (`internal/git`)

## 1. Mục đích
Dọn branch local đã merge (sau vài tháng làm feature branch, `git branch` dài cả trang). Mặc định chỉ xem trước.

## 2. Cú pháp
```
foxkit branches clean [dir] [--base main] [--gone] [--keep develop,staging] [--apply] [--json]
```
| Cờ | Mặc định | Ghi chú |
|---|---|---|
| `[dir]` | `.` | |
| `--base` | tự đoán | branch đích để xét đã merge: `origin/HEAD` → `main` → `master` → `develop` (cái nào tồn tại trước) |
| `--gone` | false | thêm branch có upstream đã bị xóa trên remote (PR merge kiểu squash không hiện "merged") |
| `--keep` | | branch luôn giữ, thêm vào danh sách bảo vệ |
| `--apply` | false | |

## 3. File
```
cmd/branches.go          # cha
cmd/branches_clean.go
internal/branches/clean.go
internal/branches/clean_test.go
```

## 4. Hợp đồng
```go
package branches
type Candidate struct {
    Name   string `json:"name"`
    Reason string `json:"reason"` // "merged" | "gone"
    Last   string `json:"last"`   // ngày commit cuối
}
type Plan struct {
    Applied bool        `json:"applied"`
    Base    string      `json:"base"`
    Delete  []Candidate `json:"delete"`
    Kept    []string    `json:"kept"`     // bị bảo vệ
    Failed  []string    `json:"failed"`
}
func Build(ctx context.Context, dir, base string, gone bool, keep []string) (Plan, error)
func Apply(ctx context.Context, dir string, p *Plan) error
```

## 5. Cách hoạt động
1. Kiểm tra là repo (`git rev-parse --is-inside-work-tree`).
2. Đoán base nếu không truyền: `git symbolic-ref --short refs/remotes/origin/HEAD` → bỏ `origin/`; lỗi thì thử `main`, `master`, `develop` bằng `git show-ref --verify --quiet refs/heads/<b>`.
3. Danh sách bảo vệ: branch hiện tại (`git branch --show-current`), base, `main`, `master`, `develop`, `dev`, `staging`, `production`, `release/*`, `--keep`.
4. Merged: `git branch --merged <base> --format=%(refname:short)`.
5. `--gone`: `git for-each-ref refs/heads --format="%(refname:short)|%(upstream:track)"`, chọn dòng có `[gone]`. (Nên chạy `git fetch --prune` trước — chỉ gợi ý trong output, không tự chạy.)
6. Ngày commit cuối: `git log -1 --format=%cs <branch>`.
7. `--apply`:
   - merged → `git branch -d <b>` (an toàn, git tự từ chối nếu chưa merge).
   - gone → `git branch -D <b>` (bắt buộc vì squash merge nên git không nhận là đã merge); manifest có `confirm` riêng.
   - Lỗi từng branch → `Failed`, tiếp tục.
8. Không đụng branch remote.

## 6. Output
```
Base: main
Sẽ xóa (merged):
  feature/old-login     2026-06-02
  fix/typo              2026-08-11
Sẽ xóa (gone trên remote):
  feature/coupon-v1     2026-09-30
Giữ: develop, main (đang đứng)
Đây là bản xem trước. Thêm --apply để thực hiện.
```
JSON: `{ "ok": true, "data": <Plan> }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `GIT_NOT_FOUND` | | 1 | |
| `GIT_NOT_REPO` | | 2 | Chạy trong thư mục repo hoặc truyền đường dẫn |
| `BRANCH_BASE_NOT_FOUND` | | 2 | Chỉ định --base |
| `BRANCHES_PARTIAL` | có branch xóa lỗi | 1 | Xem mục failed |

## 8. Manifest
```json
{ "name": "branches clean", "title": "Dọn branch đã merge", "description": "Xóa branch local đã merge hoặc đã mất trên remote", "category": "dev",
  "args": [ { "name": "dir", "type": "dir", "required": false, "label": "Repo" } ],
  "flags": [
    { "name": "base", "type": "string", "label": "Branch gốc" },
    { "name": "gone", "type": "bool", "default": false, "label": "Gồm branch đã mất trên remote" },
    { "name": "keep", "type": "list", "label": "Luôn giữ" },
    { "name": "apply", "type": "bool", "default": false, "label": "Xóa thật", "confirm": "Xóa các branch đã liệt kê?" }
  ], "admin": false, "longRunning": false, "dryRun": true, "output": "table" }
```

## 9. Test
Tích hợp với repo tạm (skip nếu không có git): tạo `main`, `feature/a` (merge vào main), `feature/b` (chưa merge), đứng ở `feature/b` → Build: delete = [feature/a]; `feature/b` không bị xóa vì đang đứng & chưa merge; `develop` (nếu tạo) nằm trong kept. Apply → `feature/a` biến mất.

Chạy thử: `.\foxkit branches clean`, `.\foxkit branches clean --gone`, `.\foxkit branches clean --apply`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + không bao giờ xóa branch hiện tại/base/bảo vệ.

## 11. Ngoài phạm vi
Xóa branch remote, chạy nhiều repo một lượt (có thể kết hợp `repos` sau).

## 12. Go học được
Dùng lại `internal/git`, glob match tên branch (`path.Match("release/*", b)`), thiết kế plan/apply giống `rename`.
