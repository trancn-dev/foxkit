# Spec: `foxkit ts`

> Nhóm: **dev** · Giai đoạn: GĐ1 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Đổi qua lại timestamp Unix ↔ ngày giờ, xem nhiều múi giờ, tính "cách đây bao lâu". Hay dùng khi đọc DB, log, JWT `exp`.

## 2. Cú pháp
```
foxkit ts [value] [--tz <IANA>] [--json]
```
| Tham số/cờ | Kiểu | Mặc định | Ghi chú |
|---|---|---|---|
| `[value]` | string | giờ hiện tại | xem bảng nhận diện |
| `--tz` | string | `Local` | ví dụ `Asia/Ho_Chi_Minh`, `UTC`, `America/New_York` |

Nhận diện `value` (theo thứ tự):
| Dạng | Ví dụ | Hiểu là |
|---|---|---|
| số 1–11 chữ số | `1763287727` | giây |
| số 12–14 chữ số | `1763287727000` | mili giây |
| số 15–17 chữ số | | micro giây |
| số 18–19 chữ số | | nano giây |
| RFC3339 | `2026-11-16T19:08:47Z`, `...+07:00` | giữ múi giờ trong chuỗi |
| `Y-m-d H:i:s` | `2026-11-16 19:08:47` | giờ theo `--tz` (giống cột DATETIME MySQL) |
| `Y-m-d` | `2026-11-16` | 00:00:00 theo `--tz` |
| `now` | | hiện tại |

## 3. File
```
cmd/ts.go
internal/ts/ts.go
internal/ts/ts_test.go
```
Trong `cmd/ts.go` (hoặc `main.go`) thêm `import _ "time/tzdata"`.

## 4. Hợp đồng
```go
package ts
type Info struct {
    Unix     int64  `json:"unix"`
    UnixMs   int64  `json:"unixMs"`
    ISO      string `json:"iso"`       // RFC3339 theo --tz
    UTC      string `json:"utc"`
    Local    string `json:"local"`     // "2026-11-16 19:08:47" theo --tz
    Zone     string `json:"zone"`      // "Asia/Ho_Chi_Minh (+07:00)"
    Weekday  string `json:"weekday"`   // "Thứ Hai"
    Relative string `json:"relative"`  // "3 giờ trước" / "sau 2 ngày"
    Detected string `json:"detected"`  // "seconds","milliseconds","rfc3339","datetime","date","now"
}
func Parse(value string, loc *time.Location, now time.Time) (time.Time, string, error)
func Describe(t time.Time, loc *time.Location, now time.Time) Info
func Relative(t, now time.Time) string
```

## 5. Cách hoạt động
1. `loc, err := time.LoadLocation(tz)`; lỗi → `TS_TZ_INVALID`.
2. `Parse`: dùng `strconv.ParseInt` cho dạng số; `time.Parse(time.RFC3339, ...)`; `time.ParseInLocation("2006-01-02 15:04:05", v, loc)`.
3. `Relative`: lấy `d := now.Sub(t)`; chọn đơn vị lớn nhất ≥1 trong: năm (365 ngày), tháng (30 ngày), ngày, giờ, phút, giây. `< 5 giây` → "vừa xong". Âm → "sau X ...".
4. Weekday tiếng Việt: mảng `[7]string{"Chủ Nhật","Thứ Hai",...}` theo `t.Weekday()`.
5. Không tham số: dùng `now`.

**Vì sao cần `time/tzdata`:** trên Windows, `time.LoadLocation` tìm dữ liệu múi giờ trong thư mục cài Go; máy không cài Go (ví dụ máy chạy launcher) sẽ lỗi. Import này nhúng dữ liệu vào binary (~450KB).

## 6. Output
Text:
```
> foxkit ts 1763287727 --tz Asia/Ho_Chi_Minh
Unix      1763287727   (ms: 1763287727000)
Giờ       2025-11-16 17:08:47  Chủ Nhật  Asia/Ho_Chi_Minh (+07:00)
ISO       2025-11-16T17:08:47+07:00
UTC       2025-11-16T10:08:47Z
Cách đây  1 năm trước
```
JSON: `{ "ok": true, "data": <Info> }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `TS_PARSE_FAILED` | không nhận diện được value | 2 | Dạng hỗ trợ: 1763287727, 2026-11-16 19:08:47, 2026-11-16T19:08:47Z |
| `TS_TZ_INVALID` | tz lạ | 2 | Dùng tên IANA, ví dụ Asia/Ho_Chi_Minh |

## 8. Manifest
```json
{ "name": "ts", "title": "Chuyển timestamp", "description": "Unix timestamp ↔ ngày giờ, nhiều múi giờ", "category": "dev",
  "args": [ { "name": "value", "type": "string", "required": false, "label": "Timestamp hoặc ngày giờ" } ],
  "flags": [ { "name": "tz", "type": "string", "default": "Local", "label": "Múi giờ" } ],
  "admin": false, "longRunning": false, "dryRun": false, "output": "kv" }
```

## 9. Test
Truyền `now` cố định (`2026-11-16T12:00:00Z`). Bảng:
| value | tz | kỳ vọng |
|---|---|---|
| `0` | UTC | `1970-01-01T00:00:00Z` |
| `1763287727000` | UTC | detected `milliseconds`, unix `1763287727` |
| `2026-11-16 19:08:47` | Asia/Ho_Chi_Minh | utc `2026-11-16T12:08:47Z` |
| `2026-11-16T19:08:47+07:00` | UTC | utc `2026-11-16T12:08:47Z` |
| `abc` | — | `TS_PARSE_FAILED` |
Relative: `now-3h` → "3 giờ trước", `now+2d` → "sau 2 ngày", `now-2s` → "vừa xong".

## 10. Tiêu chí hoàn thành
Tiêu chí chung + test + chạy được trên máy Windows không cài Go (copy exe sang máy khác hoặc tạm đổi tên thư mục Go để thử).

## 11. Ngoài phạm vi
Cộng trừ ngày (`+3d`), format tùy chỉnh, lịch âm.

## 12. Go học được
Layout thời gian dùng ngày mẫu `2006-01-02 15:04:05` (thay cho `Y-m-d H:i:s` của PHP), `time.Location`, `time.Duration`, blank import `_` để chạy `init()` của package, truyền `now` vào hàm để test được (≈ `Carbon::setTestNow()`).
