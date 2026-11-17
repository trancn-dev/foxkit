# Spec: `foxkit json`

> Nhóm: **dev** · Giai đoạn: GĐ4 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Format đẹp, minify, kiểm tra hợp lệ (báo đúng dòng/cột lỗi), lấy giá trị theo đường dẫn từ JSON (response API, file config, `composer.json`).

## 2. Cú pháp
```
foxkit json <file|-> [--min] [--indent 2] [--sort-keys] [--get <path>] [--validate] [--json]
foxkit json --text '<json>' ...
```
| Cờ | Ghi chú |
|---|---|
| (mặc định) | format đẹp, indent 2 |
| `--min` | minify |
| `--indent` | 0–8; `--indent tab` dùng tab |
| `--sort-keys` | sắp key theo ABC (mất thứ tự gốc, phải bật tay) |
| `--get` | đường dẫn: `data.items.0.id`, `["key.with.dot"]`, `items.#` (độ dài mảng), `items.*.id` (lấy id của mọi phần tử) |
| `--validate` | chỉ kiểm tra, không in nội dung |
| `--text` | JSON truyền thẳng |

Đầu vào giới hạn 50MB.

## 3. File
```
cmd/json.go
internal/jsonx/format.go
internal/jsonx/path.go
internal/jsonx/errpos.go
internal/jsonx/jsonx_test.go
```
(Package `jsonx` để không trùng `encoding/json`.)

## 4. Hợp đồng
```go
package jsonx
type SyntaxErr struct { Line, Col int; Msg string; Context string } // Context: đoạn quanh chỗ lỗi + dấu ^
func Validate(b []byte) *SyntaxErr
func Pretty(b []byte, indent string) ([]byte, error)
func Minify(b []byte) ([]byte, error)
func SortKeys(b []byte, indent string) ([]byte, error)
func Get(b []byte, path string) (any, error)
```

## 5. Cách hoạt động
1. Bỏ BOM UTF-8 đầu file nếu có (file JSON tạo trên Windows hay có).
2. **Validate**: `json.Valid`; không hợp lệ thì `json.Unmarshal` vào `any` để lấy `*json.SyntaxError`, dùng `Offset` đếm `\n` → dòng, cột. `Context`: dòng chứa lỗi, cắt ±40 ký tự, kèm dòng thứ hai có `^` chỉ cột.
3. **Pretty / Minify**: dùng `json.Indent` / `json.Compact` **trên byte gốc** → giữ nguyên thứ tự key và giữ nguyên số (không biến `1.0` thành `1`, không làm tròn số lớn như ID 19 chữ số). Đây là lý do không decode vào `map`.
4. **SortKeys**: decode với `dec.UseNumber()` (giữ số chính xác) vào `any` → `json.MarshalIndent` (map được Go tự sắp key).
5. **Get**: decode `UseNumber` vào `any`; tách path; duyệt: `map[string]any` → key; `[]any` → index (âm = đếm từ cuối); `#` → len; `*` → map qua mọi phần tử. Không tìm thấy → `JSON_PATH_NOT_FOUND` kèm phần path đã đi được.
   - Kết quả là chuỗi → text mode in chuỗi trần (không nháy) để dùng trong script; object/array → in JSON đẹp.
6. Output dùng `SetEscapeHTML(false)`.

## 6. Output
- Format/minify: in JSON kết quả.
- `--validate` hợp lệ: `JSON hợp lệ (12.4 KB).`
- Không hợp lệ:
```
JSON không hợp lệ ở dòng 12, cột 5: invalid character '}' looking for beginning of object key string
    "name": "shop",}
                   ^
```
- `--json`: `{ "ok": true, "data": { "valid": true, "result": <giá trị hoặc chuỗi đã format> } }`. Lỗi cú pháp: envelope lỗi `JSON_INVALID` với `message` gồm dòng/cột, thêm field `"position": {"line":12,"col":5}` trong error.

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `JSON_NO_INPUT` | | 2 | Ví dụ: foxkit json data.json |
| `JSON_ARGS_CONFLICT` | `--min` + `--indent`/`--sort-keys`... | 2 | |
| `JSON_INVALID` | | 1 | Sửa tại dòng N cột M |
| `JSON_PATH_INVALID` | cú pháp path sai | 2 | Ví dụ: data.items.0.id |
| `JSON_PATH_NOT_FOUND` | | 1 | Đã đi tới "data.items", không có "0" |
| `INPUT_TOO_LARGE` | > 50MB | 1 | |

## 8. Manifest
```json
{ "name": "json", "title": "JSON", "description": "Format, minify, kiểm tra, lấy giá trị JSON", "category": "dev",
  "args": [ { "name": "file", "type": "file", "required": false, "label": "File JSON" } ],
  "flags": [
    { "name": "text", "type": "text", "label": "Hoặc dán JSON" },
    { "name": "min", "type": "bool", "default": false, "label": "Minify" },
    { "name": "indent", "type": "string", "default": "2", "label": "Indent" },
    { "name": "sort-keys", "type": "bool", "default": false, "label": "Sắp key" },
    { "name": "get", "type": "string", "label": "Lấy theo đường dẫn" },
    { "name": "validate", "type": "bool", "default": false, "label": "Chỉ kiểm tra" }
  ], "admin": false, "longRunning": false, "dryRun": false, "output": "text" }
```

## 9. Test
- Pretty giữ thứ tự `{"b":1,"a":2}`; giữ `12345678901234567890` nguyên vẹn; giữ `1.0`.
- Lỗi `{"a":1,}` → dòng 1, cột 8.
- Lỗi ở dòng 3 của JSON nhiều dòng → line 3.
- Get: `a.b.0` , `items.#`, `items.*.id`, `items.-1`, `["x.y"]`.
- BOM được bỏ.

Chạy thử:
```powershell
.\foxkit json composer.json --get require.laravel/framework
curl.exe -s https://api.github.com/repos/golang/go | .\foxkit json - --get stargazers_count
.\foxkit json bad.json --validate; echo $LASTEXITCODE
```
Lưu ý: key `laravel/framework` có `/` không phải dấu chấm nên đường dẫn `require.laravel/framework` vẫn hợp lệ.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + test thứ tự key và số lớn.

## 11. Ngoài phạm vi
JSONPath/jq đầy đủ (filter điều kiện), JSON5/comment, chuyển YAML/CSV.

## 12. Go học được
`encoding/json` ở mức sâu: `json.RawMessage`, `Decoder.UseNumber`, `json.Indent/Compact`, `*json.SyntaxError`; type switch `switch v := x.(type)` để duyệt dữ liệu động (≈ duyệt array PHP lồng nhau nhưng phải kiểm kiểu rõ ràng).
