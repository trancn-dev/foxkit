# Spec: `foxkit case`

> Nhóm: **dev** · Giai đoạn: GĐ1 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Đổi kiểu chữ cho tên biến/route/file: camel, snake, kebab, studly, title, upper, lower, slug (bỏ dấu tiếng Việt). Tương đương `Str::camel/snake/kebab/studly/slug` của Laravel.

## 2. Cú pháp
```
foxkit case <mode> <text|-> [--json]
foxkit case all <text|->        # in mọi kiểu một lượt
```
`<mode>`: `camel`, `snake`, `kebab`, `studly` (alias `pascal`), `title`, `upper`, `lower`, `slug`, `constant` (`UPPER_SNAKE`), `all`.

## 3. File
```
cmd/case.go
internal/textcase/case.go
internal/textcase/vietnamese.go     # bảng bỏ dấu
internal/textcase/case_test.go
```
(Đặt tên package `textcase` vì `case` là từ khóa Go.)

## 4. Hợp đồng
```go
package textcase
func Words(s string) []string          // tách từ
func Convert(mode, s string) (string, error)
func RemoveDiacritics(s string) string // "Đơn hàng" → "Don hang"
```

## 5. Cách hoạt động
**Tách từ (`Words`)** — trái tim của lệnh:
1. Ký tự không phải chữ/số (`unicode.IsLetter`, `unicode.IsDigit`) là ranh giới: dấu cách, `_`, `-`, `.`, `/`.
2. Ranh giới camel: chữ thường → chữ hoa (`userName` → `user`,`Name`).
3. Ranh giới viết tắt: chuỗi hoa rồi hoa+thường (`HTTPServer` → `HTTP`,`Server`; `getHTTPResponse` → `get`,`HTTP`,`Response`).
4. Chữ ↔ số **không** tách (`file2name` giữ nguyên, giống Laravel).

**Ghép**:
| mode | Quy tắc | `"user profile ID"` → |
|---|---|---|
| camel | từ đầu lower, các từ sau Title | `userProfileId` |
| studly | mọi từ Title | `UserProfileId` |
| snake | lower + `_` | `user_profile_id` |
| constant | upper + `_` | `USER_PROFILE_ID` |
| kebab | lower + `-` | `user-profile-id` |
| title | Title + dấu cách | `User Profile Id` |
| upper/lower | `strings.ToUpper/ToLower` trên chuỗi gốc | |
| slug | bỏ dấu → lower → kebab, chỉ giữ `a-z0-9` | `user-profile-id` |

"Title" một từ: rune đầu `unicode.ToUpper`, phần còn lại `ToLower`. Làm việc trên `[]rune`, không trên byte (để `đ` không bị cắt đôi).

**Bỏ dấu tiếng Việt** (`vietnamese.go`): `map[rune]rune` gồm đủ nguyên âm có dấu thường/hoa (a ă â e ê i o ô ơ u ư y × 5 dấu + gốc) và `đ→d`, `Đ→D`. Rune không có trong map giữ nguyên. Không cần `golang.org/x/text`.

`all`: trả map mọi mode.

## 6. Output
Text: kết quả. `all`:
```
camel     donHangMoi
studly    DonHangMoi
snake     don_hang_moi
...
```
Lưu ý: camel/snake/... cũng **bỏ dấu** (tên biến không nên có dấu); chỉ `title/upper/lower` giữ dấu.

JSON: `{ "ok": true, "data": { "input": "Đơn hàng mới", "mode": "slug", "result": "don-hang-moi" } }`
`all`: `"result": { "camel": "...", "snake": "...", ... }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `CASE_MODE_INVALID` | mode lạ | 2 | Chọn: camel, snake, kebab, studly, title, upper, lower, slug, constant, all |
| `CASE_NO_INPUT` | thiếu text | 2 | Ví dụ: foxkit case snake "userName" |

## 8. Manifest
```json
{ "name": "case", "title": "Đổi kiểu chữ", "description": "camel, snake, kebab, studly, slug tiếng Việt...", "category": "dev",
  "args": [
    { "name": "mode", "type": "enum", "options": ["all","camel","snake","kebab","studly","title","upper","lower","slug","constant"], "required": true, "label": "Kiểu" },
    { "name": "text", "type": "string", "required": true, "label": "Chuỗi" }
  ],
  "flags": [], "admin": false, "longRunning": false, "dryRun": false, "output": "kv" }
```

## 9. Test (bảng)
| input | mode | kỳ vọng |
|---|---|---|
| `userName` | snake | `user_name` |
| `HTTPServer` | kebab | `http-server` |
| `getHTTPResponse` | studly | `GetHttpResponse` |
| `user_profile-id` | camel | `userProfileId` |
| `Đơn hàng mới` | slug | `don-hang-moi` |
| `Tiếng Việt có dấu` | snake | `tieng_viet_co_dau` |
| `  nhiều   khoảng trắng ` | kebab | `nhieu-khoang-trang` |
| `Ưu đãi 50%!` | slug | `uu-dai-50` |
| `file2name` | snake | `file2name` |
| `` (rỗng) | camel | `` |

## 10. Tiêu chí hoàn thành
Tiêu chí chung + toàn bộ bảng test pass.

## 11. Ngoài phạm vi
Bỏ dấu ngôn ngữ khác (Pháp, Đức...), số nhiều/số ít (`Str::plural`).

## 12. Go học được
`rune` vs `byte` (Go string là byte UTF-8, `len("đ") == 2`), duyệt `for i, r := range s`, package `unicode`, `map[rune]rune`, đặt tên package tránh từ khóa.
