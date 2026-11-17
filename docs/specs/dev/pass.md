# Spec: `foxkit pass`

> Nhóm: **dev** · Giai đoạn: GĐ1 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Sinh mật khẩu hoặc chuỗi ngẫu nhiên an toàn (mật khẩu DB, secret webhook, token test).

## 2. Cú pháp
```
foxkit pass [--len N] [--count N] [--charset all|alnum|alpha|num|hex] [--no-ambiguous] [--json]
```
| Cờ | Kiểu | Mặc định | Ràng buộc |
|---|---|---|---|
| `--len` | int | 24 | 4–1024 |
| `--count` | int | 1 | 1–100 |
| `--charset` | enum | `all` | xem bảng dưới |
| `--no-ambiguous` | bool | false | bỏ ký tự dễ nhầm `0 O o 1 l I |` |

| charset | Ký tự |
|---|---|
| `all` | `a-z A-Z 0-9` + `!@#$%^&*()-_=+[]{};:,.?` |
| `alnum` | `a-z A-Z 0-9` |
| `alpha` | `a-z A-Z` |
| `num` | `0-9` |
| `hex` | `0-9 a-f` |

Không dùng dấu nháy, backslash, dấu cách, `<>` trong `all` để dán vào `.env` không phải escape.

## 3. File
```
cmd/pass.go
internal/pass/pass.go
internal/pass/pass_test.go
```

## 4. Hợp đồng
```go
package pass
type Options struct { Length int; Charset string; NoAmbiguous bool }
func Generate(opt Options) (string, error)
func Alphabet(charset string, noAmbiguous bool) (string, error)
func EntropyBits(length, alphabetSize int) float64 // length * log2(alphabetSize)
```

## 5. Cách hoạt động
1. Lấy alphabet theo charset; lọc ký tự dễ nhầm nếu cần.
2. Với mỗi vị trí: `n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))` → lấy `alphabet[n]`. Cách này **không bị lệch phân bố** (khác `byte % len`).
3. Với `all` và `len >= 4`: đảm bảo có ít nhất 1 chữ thường, 1 chữ hoa, 1 số, 1 ký hiệu. Nếu thiếu thì sinh lại cả chuỗi (vòng lặp, tối đa 10000 lần; với `len=4` chỉ ~6% chuỗi đủ 4 loại nên 100 lần là không đủ).
4. Tính `entropyBits` làm tròn 1 chữ số thập phân.

## 6. Output
Text: mỗi mật khẩu một dòng.
JSON:
```json
{ "ok": true, "data": { "length": 24, "charset": "all", "entropyBits": 155.4, "items": ["q7#Vk2!pZx..."] } }
```

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `PASS_LEN_INVALID` | len ngoài 4–1024 | 2 | Ví dụ: --len 32 |
| `PASS_COUNT_INVALID` | count ngoài 1–100 | 2 | Ví dụ: --count 5 |
| `PASS_CHARSET_INVALID` | charset lạ | 2 | Chọn: all, alnum, alpha, num, hex |

## 8. Manifest
```json
{ "name": "pass", "title": "Sinh mật khẩu", "description": "Sinh mật khẩu/chuỗi ngẫu nhiên an toàn", "category": "dev",
  "args": [],
  "flags": [
    { "name": "len", "type": "int", "default": 24, "min": 4, "max": 1024, "label": "Độ dài" },
    { "name": "count", "type": "int", "default": 1, "min": 1, "max": 100, "label": "Số lượng" },
    { "name": "charset", "type": "enum", "options": ["all","alnum","alpha","num","hex"], "default": "all", "label": "Bộ ký tự" },
    { "name": "no-ambiguous", "type": "bool", "default": false, "label": "Bỏ ký tự dễ nhầm" }
  ],
  "admin": false, "longRunning": false, "dryRun": false, "output": "text" }
```

## 9. Test
- Độ dài đúng; mọi ký tự thuộc alphabet (bảng test cho từng charset).
- `all` luôn có đủ 4 loại ký tự (chạy 1000 lần).
- `--no-ambiguous` không chứa `0Oo1lI|`.
- `EntropyBits(24, 64)` = 144.

Chạy thử: `.\foxkit pass`, `.\foxkit pass --len 64 --charset hex`, `.\foxkit pass --count 3 --json`.

## 10. Tiêu chí hoàn thành
Tiêu chí chung + test mục 9.

## 11. Ngoài phạm vi
Passphrase dạng từ (diceware), kiểm tra độ mạnh mật khẩu có sẵn, copy clipboard.

## 12. Go học được
Vì sao `math/rand` không dùng cho mật khẩu (dự đoán được, ≈ `rand()` vs `random_bytes()` của PHP), `crypto/rand.Int`, `math/big`, `strings.Builder` (≈ nối chuỗi hiệu quả), `strings.ContainsAny`.
