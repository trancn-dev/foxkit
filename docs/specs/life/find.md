# Spec: `foxkit find`

> Nhóm: **life** · Giai đoạn: GĐ2 · Trạng thái: chưa làm · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Tìm file theo tên, đuôi, kích thước, ngày sửa, và (tùy chọn) theo nội dung chữ bên trong, **trong một thư mục cụ thể**.

Không thay Everything (tìm toàn ổ tức thì nhờ đọc chỉ mục NTFS). `find` dùng khi cần lọc kết hợp nhiều điều kiện, tìm trong nội dung file, hoặc để launcher hiển thị kết quả.

## 2. Cú pháp
```
foxkit find [pattern] [dir] [--ext pdf,docx] [--type f|d] [--larger 10MB] [--smaller 1GB]
            [--newer 7d] [--older 1y] [--content "từ khóa"] [--ignore-case] [--max 200] [--hidden] [--json]
```
| Tham số/cờ | Mặc định | Ghi chú |
|---|---|---|
| `[pattern]` | (mọi tên) | Có `*`/`?` thì là glob (`*.pdf`, `hop-dong*`); không có thì là chuỗi con. Luôn không phân biệt hoa thường. Không dấu khớp có dấu: `hop dong` khớp `Hợp đồng` |
| `[dir]` | thư mục hiện tại | |
| `--ext` | | danh sách đuôi |
| `--type` | `f` | `f` file, `d` thư mục, `a` cả hai |
| `--larger`/`--smaller` | | `fsutil.ParseSize` |
| `--newer`/`--older` | | theo ngày sửa, `fsutil.ParseAge` (`7d`, `2w`, `1y`=365d) |
| `--content` | | tìm chuỗi trong nội dung (chỉ file văn bản ≤ 20MB) |
| `--ignore-case` | true khi `--content` | |
| `--max` | 200 | 1–10.000; đủ thì dừng sớm |
| `--hidden` | false | gồm file ẩn và `DefaultSkipDirs` |

## 3. File
```
cmd/find.go
internal/find/match.go     # khớp tên, glob, bỏ dấu
internal/find/content.go   # tìm trong nội dung
internal/find/find.go
internal/find/find_test.go
```

## 4. Hợp đồng
```go
package find
type Hit struct {
    Path     string    `json:"path"`
    Name     string    `json:"name"`
    IsDir    bool      `json:"isDir"`
    Bytes    int64     `json:"bytes"`
    Modified time.Time `json:"modified"`
    Line     int       `json:"line,omitempty"`    // khi --content: dòng khớp đầu tiên
    Snippet  string    `json:"snippet,omitempty"` // tối đa 160 ký tự quanh chỗ khớp
}
type Result struct {
    Dir       string `json:"dir"`
    Hits      []Hit  `json:"hits"`
    Truncated bool   `json:"truncated"` // đã chạm --max
    Scanned   int    `json:"scanned"`
}
func Run(ctx context.Context, opt Options) (Result, error)
```

## 5. Cách hoạt động
1. Duyệt bằng `filepath.WalkDir`; dừng sớm khi đủ `--max` (trả về một lỗi sentinel `errStop` từ callback rồi bỏ qua nó).
2. Khớp tên:
   - Chuẩn hóa cả pattern và tên: `textcase.RemoveDiacritics` + `strings.ToLower`.
   - Glob: `filepath.Match(pattern, name)` trên chuỗi đã chuẩn hóa.
   - Chuỗi con: `strings.Contains`.
3. Lọc type/ext/size/time trước, **lọc nội dung sau cùng** (đắt nhất).
4. Nội dung:
   - Đọc 8KB đầu; có byte `0x00` → coi là nhị phân, bỏ qua.
   - `bufio.Scanner` với buffer tối đa 1MB/dòng, dừng ở dòng khớp đầu tiên.
   - Chạy song song bằng worker pool (giống `dup`), giữ thứ tự kết quả ổn định bằng cách sắp xếp theo path ở cuối.
   - Chỉ hỗ trợ UTF-8 / ASCII. File UTF-16 (có BOM `FF FE`) → giải mã bằng `unicode/utf16` trước khi tìm (nhiều file `.txt`, `.reg` trên Windows là UTF-16).
5. Không tìm trong `.docx/.pdf` (là file nén/nhị phân) ở v1.

## 6. Output
Text:
```
D:\Docs\Hop dong\Hợp đồng thuê nhà 2026.pdf     1.2 MB   2026-10-02
D:\Docs\Scan\hop-dong-dien.pdf                  640 KB   2026-08-15
2 kết quả (đã quét 4.812 mục)
```
Với `--content`:
```
D:\Projects\shop\routes\api.php:42   Route::post('/coupon', [CouponController::class, 'apply']);
```
JSON: `{ "ok": true, "data": <Result> }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `DIR_NOT_FOUND` | | 2 | |
| `PATTERN_INVALID` | glob sai (`[` không đóng) | 2 | Ví dụ: "*.pdf" hoặc "hop dong" |
| `SIZE_INVALID`, `AGE_INVALID` | | 2 | Ví dụ: 10MB, 7d |
| `FIND_NO_CRITERIA` | không pattern, không cờ lọc nào | 2 | Cần ít nhất một điều kiện tìm |

Không có kết quả → exit 0, `hits: []`.

## 8. Manifest
```json
{ "name": "find", "title": "Tìm file", "description": "Tìm file theo tên, đuôi, kích thước, ngày, nội dung", "category": "life",
  "args": [
    { "name": "pattern", "type": "string", "required": false, "label": "Tên (hỗ trợ *, ?)" },
    { "name": "dir", "type": "dir", "required": false, "label": "Thư mục" }
  ],
  "flags": [
    { "name": "ext", "type": "list", "label": "Đuôi file" },
    { "name": "type", "type": "enum", "options": ["f","d","a"], "default": "f", "label": "Loại" },
    { "name": "larger", "type": "string", "label": "Lớn hơn" },
    { "name": "smaller", "type": "string", "label": "Nhỏ hơn" },
    { "name": "newer", "type": "string", "label": "Sửa trong vòng" },
    { "name": "older", "type": "string", "label": "Cũ hơn" },
    { "name": "content", "type": "string", "label": "Chứa nội dung" },
    { "name": "ignore-case", "type": "bool", "default": true, "label": "Không phân biệt hoa thường" },
    { "name": "max", "type": "int", "default": 200, "min": 1, "max": 10000, "label": "Tối đa" },
    { "name": "hidden", "type": "bool", "default": false, "label": "Gồm file ẩn" }
  ], "admin": false, "longRunning": false, "dryRun": false, "output": "table" }
```

## 9. Test
- `hop dong` khớp `Hợp Đồng.pdf`; `*.PDF` khớp `a.pdf`.
- `--larger 1KB --smaller 1MB` lọc đúng biên.
- `--newer 7d` với file đổi mtime bằng `os.Chtimes`.
- `--content` bỏ qua file nhị phân; tìm được trong file UTF-16 LE có BOM.
- `--max 3` → đúng 3 kết quả, `truncated: true`.

Chạy thử:
```powershell
.\foxkit find "hop dong" D:\Docs --ext pdf
.\foxkit find --larger 1GB --older 1y D:\
.\foxkit find "*.php" D:\Projects\shop --content "CouponController"
```

## 10. Tiêu chí hoàn thành
Tiêu chí chung + test + dừng sớm thật sự khi đủ `--max` (không duyệt hết cây).

## 11. Ngoài phạm vi
Chỉ mục/tìm tức thì toàn ổ, tìm trong PDF/Office, regex nội dung, thay thế nội dung.

## 12. Go học được
`filepath.Match`, lỗi sentinel để dừng `WalkDir` sớm (≈ `break` khỏi callback), phát hiện nhị phân, `unicode/utf16`, dùng lại package `textcase` từ GĐ1.
