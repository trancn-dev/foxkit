# Spec: `foxkit clip`

> Nhóm: **life** · Giai đoạn: GĐ6 · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Đọc/ghi clipboard từ dòng lệnh, Unicode đúng (tiếng Việt). Dùng để nối lệnh: `foxkit uuid | foxkit clip`, `foxkit clip | foxkit json -`.

## 2. Cú pháp
```
foxkit clip                 # đọc clipboard ra stdout
foxkit clip <text>          # ghi text vào clipboard
foxkit clip -               # ghi stdin vào clipboard
foxkit clip --clear
```

## 3. Dependency
`github.com/atotto/clipboard`.
**Lý do**: API clipboard Win32 (`OpenClipboard`, `GlobalAlloc`, `GlobalLock`, `SetClipboardData`) dễ sai ở phần ai sở hữu bộ nhớ sau khi gọi; thư viện này nhỏ, thuần Go (không cgo), xử lý UTF-16 đúng. Các cách thay thế đều kém hơn: `clip.exe` làm hỏng ký tự tiếng Việt khi nhận UTF-8, gọi PowerShell `Get/Set-Clipboard` chậm vài trăm ms mỗi lần.

## 4. File
```
cmd/clip.go
internal/clip/clip.go     # bọc atotto để đổi sang implementation khác dễ dàng
internal/clip/clip_test.go
```

## 5. Hợp đồng
```go
package clip
type Clipboard interface { Read() (string, error); Write(string) error }
var Default Clipboard = atottoClipboard{}
```

## 6. Cách hoạt động
1. Đọc: `clipboard.ReadAll()`. Clipboard không có text (đang chứa ảnh/file) → `CLIP_NO_TEXT`.
2. Ghi: chuẩn hóa xuống dòng sang CRLF (để dán vào Notepad đúng); bỏ một `\n` cuối nếu dữ liệu đến từ stdin (lệnh pipe thường thêm newline).
3. Clipboard đang bị app khác giữ → thư viện trả lỗi → thử lại 3 lần cách 100ms → `CLIP_BUSY`.
4. Giới hạn ghi 10MB.
5. Text mode đọc: in nguyên văn; khi stdout là terminal thì thêm `\n` cuối nếu chưa có.

## 7. Output
JSON đọc: `{ "ok": true, "data": { "text": "...", "length": 42 } }` (length tính theo rune)
JSON ghi: `{ "ok": true, "data": { "written": true, "length": 42 } }`

## 8. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `CLIP_NO_TEXT` | | 1 | Clipboard đang chứa ảnh hoặc file, không phải chữ |
| `CLIP_BUSY` | | 1 | Đóng ứng dụng đang giữ clipboard rồi thử lại |
| `INPUT_TOO_LARGE` | | 1 | |

## 9. Manifest
```json
{ "name": "clip", "title": "Clipboard", "description": "Đọc/ghi clipboard", "category": "life",
  "args": [ { "name": "text", "type": "text", "required": false, "label": "Nội dung (để trống = đọc)" } ],
  "flags": [ { "name": "clear", "type": "bool", "default": false, "label": "Xóa clipboard" } ],
  "admin": false, "longRunning": false, "dryRun": false, "output": "text" }
```

## 10. Test
Dùng implementation giả (`fakeClipboard` lưu trong biến) để test tầng cmd: chuẩn hóa CRLF, bỏ newline cuối từ stdin, giới hạn kích thước. Test thật (Windows, ghi rồi đọc lại `"Xin chào đ 🚀"`) đánh dấu chỉ chạy khi có biến `FOXKIT_TEST_CLIPBOARD=1` (vì ghi đè clipboard của người dùng).

Chạy thử:
```powershell
.\foxkit uuid | .\foxkit clip -
.\foxkit clip
.\foxkit clip "Đơn hàng mới"
.\foxkit clip | .\foxkit case slug -
```

## 11. Tiêu chí hoàn thành
Tiêu chí chung + tiếng Việt và emoji đi qua clipboard nguyên vẹn.

## 12. Ngoài phạm vi
Ảnh/file trong clipboard, lịch sử clipboard (Win+V đã có), cờ `--copy` cho các lệnh khác (chỉ làm khi có yêu cầu).

## 13. Go học được
Interface để thay thế implementation khi test (≈ bind interface trong Service Container Laravel), retry đơn giản, lần đầu thêm dependency bên thứ ba có lý do.
