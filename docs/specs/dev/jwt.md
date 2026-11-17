# Spec: `foxkit jwt`

> Nhóm: **dev** · Giai đoạn: GĐ4 · Trạng thái: chưa làm · Cần trước: [dev/ts.md](ts.md) (dùng `ts.Relative`)

## 1. Mục đích
Giải mã JWT (Sanctum không dùng JWT nhưng Passport, Firebase, Auth0, API bên thứ ba thì có) để xem payload, hạn dùng, và tùy chọn kiểm tra chữ ký HS256/384/512.

## 2. Cú pháp
```
foxkit jwt <token|-> [--secret <s>] [--json]
```
| Tham số/cờ | Ghi chú |
|---|---|
| `<token>` | chấp nhận cả chuỗi có tiền tố `Bearer ` (tự bỏ) |
| `--secret` | kiểm tra chữ ký HMAC; alg khác HS* → báo không hỗ trợ verify |

**Cảnh báo bảo mật trong text output**: token và secret có thể nằm trong lịch sử terminal; nên dùng `-` và dán qua stdin.

## 3. File
```
cmd/jwt.go
internal/jwt/jwt.go
internal/jwt/jwt_test.go
```

## 4. Hợp đồng
```go
package jwt
type Decoded struct {
    Header    map[string]any `json:"header"`
    Payload   map[string]any `json:"payload"`
    Signature string         `json:"signature"`           // base64url gốc
    Times     map[string]TimeInfo `json:"times"`          // exp, iat, nbf nếu có
    Expired   *bool          `json:"expired"`             // nil nếu không có exp
    Verified  *bool          `json:"verified"`            // nil nếu không verify
    Warnings  []string       `json:"warnings"`            // "alg=none", "không có exp"
}
type TimeInfo struct { Unix int64 `json:"unix"`; ISO string `json:"iso"`; Relative string `json:"relative"` }
func Decode(token string, now time.Time) (Decoded, error)
func VerifyHMAC(token, secret string) (bool, error)
```

## 5. Cách hoạt động
1. Trim, bỏ `Bearer `. Tách `.` → phải đúng 3 phần (JWE 5 phần → `JWT_UNSUPPORTED`).
2. Header, payload: `base64.RawURLEncoding.DecodeString` (thử thêm bản có padding nếu lỗi) → `json.Unmarshal` với `UseNumber`.
3. Claim thời gian `exp`, `iat`, `nbf` (kiểu `json.Number`) → `TimeInfo` theo giờ local.
4. `Expired = now > exp`. Thêm cảnh báo: `alg` = `none`; không có `exp`; `nbf` trong tương lai.
5. `VerifyHMAC`: alg `HS256/384/512` → `hmac.New(sha256.New, []byte(secret))` trên `header.payload` (chuỗi gốc, không decode lại) → so bằng `hmac.Equal` (so sánh thời gian hằng, tránh timing attack). Secret dạng `base64:...` (kiểu Laravel) → decode trước.

## 6. Output
```
Header   { "alg": "HS256", "typ": "JWT" }
Payload  {
  "sub": "42",
  "name": "Demo",
  "exp": 1763290000
}
exp      2026-11-16 20:46:40  (sau 37 phút)   ✓ còn hạn
iat      2026-11-16 19:46:40  (23 phút trước)
Chữ ký   ✓ hợp lệ với secret đã cho
```
JSON: `{ "ok": true, "data": <Decoded> }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `JWT_INVALID` | không đủ 3 phần / base64 / JSON lỗi | 1 | Token JWT có dạng xxx.yyy.zzz |
| `JWT_UNSUPPORTED` | JWE, hoặc verify alg không phải HS* | 1 | Chỉ hỗ trợ verify HS256/HS384/HS512 |
| `JWT_SIGNATURE_INVALID` | `--secret` sai | 1 | Kiểm tra lại secret |

Token hết hạn **không** phải lỗi (exit 0, `expired: true`).

## 8. Manifest
```json
{ "name": "jwt", "title": "Giải mã JWT", "description": "Xem header, payload, hạn dùng, kiểm tra chữ ký HS*", "category": "dev",
  "args": [ { "name": "token", "type": "text", "required": true, "label": "Token" } ],
  "flags": [ { "name": "secret", "type": "string", "label": "Secret (để verify)" } ],
  "admin": false, "longRunning": false, "dryRun": false, "output": "kv" }
```

## 9. Test
Token mẫu từ jwt.io (`HS256`, secret `your-256-bit-secret`) → verify true; đổi 1 ký tự payload → false. `alg: none` → warning. exp quá khứ/tương lai với `now` cố định. `Bearer ` được bỏ.

Chạy thử: `.\foxkit jwt eyJhbGciOi...`, `Get-Clipboard | .\foxkit jwt -`, `.\foxkit jwt <token> --secret xxx --json`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + test vector jwt.io.

## 11. Ngoài phạm vi
Verify RS/ES (cần public key/JWKS), tạo/ký JWT mới.

## 12. Go học được
`crypto/hmac`, `hmac.Equal`, `base64.RawURLEncoding`, con trỏ `*bool` để phân biệt "false" với "không có" (≈ `null` trong PHP/JS), dùng lại package khác (`ts`).
