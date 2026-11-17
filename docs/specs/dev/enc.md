# Spec: `foxkit enc`

> Nhóm: **dev** · Giai đoạn: GĐ1 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Encode/decode nhanh: base64, base64url, hex, URL, HTML entity.

## 2. Cú pháp
```
foxkit enc <mode> <text|-> [--decode] [--json]
```
| Tham số/cờ | Kiểu | Ghi chú |
|---|---|---|
| `<mode>` | enum `base64`,`base64url`,`hex`,`url`,`html` | bắt buộc |
| `<text>` | text | `-` = đọc stdin (đọc hết, giới hạn 10MB) |
| `--decode` / `-d` | bool | mặc định encode |
| `--file` | file | đọc nội dung từ file thay cho `<text>` (hữu ích để base64 một ảnh) |

## 3. File
```
cmd/enc.go
internal/enc/enc.go
internal/enc/enc_test.go
```

## 4. Hợp đồng
```go
package enc
func Encode(mode string, in []byte) (string, error)
func Decode(mode string, in string) ([]byte, error)
```

## 5. Cách hoạt động
| mode | Encode | Decode |
|---|---|---|
| base64 | `base64.StdEncoding.EncodeToString` | `StdEncoding.DecodeString`; nếu lỗi thử `RawStdEncoding` (thiếu `=`) |
| base64url | `base64.RawURLEncoding` | thử `RawURLEncoding` rồi `URLEncoding` |
| hex | `hex.EncodeToString` | `hex.DecodeString` (bỏ khoảng trắng trước) |
| url | `url.QueryEscape` | `url.QueryUnescape` |
| html | `html.EscapeString` | `html.UnescapeString` |

- Trước khi decode: `strings.TrimSpace` (bỏ xuống dòng thừa khi pipe).
- Decode ra byte không phải UTF-8 hợp lệ (`utf8.Valid`): text mode in cảnh báo ra stderr `Kết quả là dữ liệu nhị phân` và in dạng hex; JSON đặt `"binary": true` và `result` là base64 của dữ liệu đó.
- Đọc stdin: `io.ReadAll(io.LimitReader(os.Stdin, 10<<20+1))`, vượt 10MB → `ENC_INPUT_TOO_LARGE`.
- Text mode in kết quả **không có xuống dòng thêm** khi stdout không phải terminal (để pipe tiếp chính xác); là terminal thì thêm `\n`. Kiểm tra bằng `os.Stdout.Stat()` và `Mode()&os.ModeCharDevice`.

## 6. Output
Text: chỉ in kết quả.
JSON:
```json
{ "ok": true, "data": { "mode": "base64", "decode": false, "binary": false, "result": "aGVsbG8=" } }
```

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `ENC_MODE_INVALID` | mode lạ | 2 | Chọn: base64, base64url, hex, url, html |
| `ENC_NO_INPUT` | thiếu text và --file | 2 | Ví dụ: foxkit enc base64 "hello" |
| `ENC_DECODE_FAILED` | chuỗi không hợp lệ | 1 | Kiểm tra chuỗi có đúng định dạng <mode> không |
| `ENC_INPUT_TOO_LARGE` | > 10MB | 1 | Dùng --file với file nhỏ hơn 10MB |

## 8. Manifest
```json
{ "name": "enc", "title": "Encode / Decode", "description": "base64, base64url, hex, url, html", "category": "dev",
  "args": [
    { "name": "mode", "type": "enum", "options": ["base64","base64url","hex","url","html"], "required": true, "label": "Kiểu" },
    { "name": "text", "type": "text", "required": false, "label": "Nội dung" }
  ],
  "flags": [
    { "name": "decode", "type": "bool", "default": false, "label": "Decode" },
    { "name": "file", "type": "file", "label": "Từ file" }
  ],
  "admin": false, "longRunning": false, "dryRun": false, "output": "text" }
```

## 9. Test
Bảng round-trip `Decode(Encode(x)) == x` với: `""`, `"hello"`, `"Xin chào đ"`, byte `0x00..0xff`.
Case cố định: base64 `"hello"` → `aGVsbG8=`; url `"a b&c"` → `a+b%26c`; html `"<a>"` → `&lt;a&gt;`; base64 decode thiếu padding `aGVsbG8` → `hello`.

Chạy thử: `.\foxkit enc base64 "Xin chào"`, `.\foxkit enc base64 -d WGluIGNow6Bv`, `.\foxkit enc base64 --file logo.png --json`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + test mục 9.

## 11. Ngoài phạm vi
Mã hóa (AES...), gzip, base32, encode tên file Unicode.

## 12. Go học được
`[]byte` vs `string` (string bất biến, giống PHP string là byte; khác JS string là UTF-16), `unicode/utf8`, `io.LimitReader`, kiểm tra stdout có phải terminal.
