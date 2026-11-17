# Spec: `foxkit hash`

> Nhóm: **dev** · Giai đoạn: GĐ1 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Tính hash của file/chuỗi/stdin; kiểm tra file tải về có khớp checksum không.

## 2. Cú pháp
```
foxkit hash <file>... [--algo md5|sha1|sha256|sha512|crc32] [--check <hex>] [--json]
foxkit hash --text "chuỗi" [--algo ...]
foxkit hash -            # đọc stdin
```
| Tham số/cờ | Kiểu | Mặc định | Ghi chú |
|---|---|---|---|
| `<file>...` | file (nhiều) | — | `-` = stdin. Bắt buộc nếu không có `--text` |
| `--algo` | enum | `sha256` | |
| `--all` | bool | false | tính cả 5 thuật toán một lượt |
| `--text` | string | — | không đi cùng `<file>` |
| `--check` | string | — | chỉ dùng khi có đúng 1 nguồn; so sánh không phân biệt hoa/thường |

## 3. File
```
cmd/hash.go
internal/hash/hash.go
internal/hash/hash_test.go
```

## 4. Hợp đồng
```go
package hash
func New(algo string) (hash.Hash, error)               // dùng hash.Hash của stdlib
func Reader(r io.Reader, algos []string) (map[string]string, int64, error) // trả hex theo algo + số byte đã đọc
func File(path string, algos []string) (map[string]string, int64, error)
```

## 5. Cách hoạt động
1. Mở file (`os.Open`) → `defer f.Close()`.
2. Tạo các `hash.Hash`, gộp bằng `io.MultiWriter(h1, h2, ...)` để **đọc file một lần** cho nhiều thuật toán (`--all`).
3. `io.Copy(multiWriter, f)` → đọc theo khối, không nạp cả file vào RAM (file ISO 5GB vẫn chạy).
4. `hex.EncodeToString(h.Sum(nil))`.
5. Nhiều file: xử lý tuần tự (v1). Một file lỗi (không tồn tại) **không dừng** các file khác: ghi `error` vào item đó, cuối cùng exit 1 nếu có ít nhất 1 lỗi.
6. `--check`: `strings.EqualFold(got, want)` → `match`. Không khớp thì exit 1 với code `HASH_MISMATCH` (vẫn in kết quả ở stdout trước, rồi lỗi ở stderr).
7. crc32 dùng `crc32.NewIEEE()`, hex 8 ký tự.

## 6. Output
Text (giống định dạng `sha256sum` để quen tay):
```
9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08  setup.exe
```
`--all`:
```
setup.exe (12.3 MB)
  md5     5d41402abc4b2a76b9719d911017c592
  sha1    ...
  sha256  ...
```
JSON:
```json
{ "ok": true, "data": { "items": [
  { "source": "setup.exe", "bytes": 12897432, "hashes": { "sha256": "9f86d0..." }, "match": true }
] } }
```
`match` chỉ có khi dùng `--check`. `source` là `"stdin"` hoặc `"text"` cho hai nguồn đó.

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `HASH_NO_INPUT` | không có file, không `--text` | 2 | Ví dụ: foxkit hash setup.exe |
| `HASH_ALGO_INVALID` | algo lạ | 2 | Chọn: md5, sha1, sha256, sha512, crc32 |
| `HASH_ARGS_CONFLICT` | `--text` cùng file; `--check` với >1 nguồn | 2 | |
| `FILE_NOT_FOUND` | file không tồn tại (theo item) | 1 | Kiểm tra lại đường dẫn |
| `HASH_MISMATCH` | `--check` không khớp | 1 | File có thể tải lỗi hoặc bị sửa, hãy tải lại |

## 8. Manifest
```json
{ "name": "hash", "title": "Tính hash", "description": "Tính md5/sha1/sha256/sha512/crc32 của file hoặc chuỗi", "category": "dev",
  "args": [ { "name": "file", "type": "file", "required": false, "multiple": true, "label": "File" } ],
  "flags": [
    { "name": "algo", "type": "enum", "options": ["md5","sha1","sha256","sha512","crc32"], "default": "sha256", "label": "Thuật toán" },
    { "name": "all", "type": "bool", "default": false, "label": "Tất cả thuật toán" },
    { "name": "text", "type": "string", "label": "Chuỗi" },
    { "name": "check", "type": "string", "label": "Hash cần so" }
  ],
  "admin": false, "longRunning": false, "dryRun": false, "output": "table" }
```
(Thêm thuộc tính `multiple` cho arg vào schema ở foundation khi làm lệnh này.)

## 9. Test
Vector chuẩn với chuỗi `"abc"`:
| algo | hex |
|---|---|
| md5 | `900150983cd24fb0d6963f7d28e17f72` |
| sha1 | `a9993e364706816aba3e25717850c26c9cd0d89d` |
| sha256 | `ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad` |
| crc32 | `352441c2` |

Thêm: chuỗi rỗng; `--check` hoa/thường; file tạm bằng `t.TempDir()`.

Chạy thử:
```powershell
.\foxkit hash --text abc --all
"abc" | .\foxkit hash -      # lưu ý PowerShell thêm CRLF ở cuối → hash khác, đây là hành vi đúng
.\foxkit hash C:\Windows\notepad.exe --check <hash-đúng>
```

## 10. Tiêu chí hoàn thành
Tiêu chí chung + test vector pass + file lớn (>1GB) chạy được mà RAM không tăng đáng kể.

## 11. Ngoài phạm vi
Hash thư mục, file `.sha256` liệt kê nhiều file, bcrypt/argon2 (khác mục đích).

## 12. Go học được
Interface `io.Reader`/`io.Writer`/`hash.Hash` và vì sao `io.Copy` ghép được mọi thứ (≈ stream trong Node), `io.MultiWriter`, `defer`, gom lỗi từng item thay vì dừng ngay.
