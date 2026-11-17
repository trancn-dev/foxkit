# Spec: `foxkit svg`

> Nhóm: **life** (xử lý ảnh) · Giai đoạn: GĐ7 · Trạng thái: xong · Cần trước: [core/foundation.md](../core/foundation.md)
> Khó: **5** — lệnh có thuật toán nặng nhất roadmap. Làm sau cùng trong GĐ7. Không dependency (chỉ thư viện chuẩn).

## 1. Mục đích
Chuyển ảnh PNG (cũng nhận JPG, GIF) sang SVG. Có hai chế độ:
- **`trace`** (mặc định): vector hóa thật — dò đường viền các vùng màu rồi xuất thành `<path>`. Dùng cho **logo, icon, hình vẽ phẳng, chữ ký, ảnh đen trắng**.
- **`embed`**: chỉ nhúng ảnh gốc vào file SVG (base64). Không phải vector, nhưng nhanh, đúng 100%, dùng khi chỉ cần "đóng gói" thành `.svg`.

**Giới hạn cần nói rõ cho người dùng**: tracing chỉ cho kết quả tốt với ảnh ít màu, cạnh rõ. Ảnh chụp/ảnh nhiều chuyển màu sẽ ra file SVG rất nặng và xấu. Lệnh phải cảnh báo khi ảnh có quá nhiều màu (xem 5.2).

## 2. Cú pháp
```
foxkit svg <file>... [--mode trace|embed] [--out <file|dir>] [--colors 8] [--mono] [--threshold 128]
                    [--simplify 1.0] [--smooth] [--despeckle 4] [--bg keep|remove] [--scale 1] [--force] [--json]
```
| Cờ | Mặc định | Ghi chú |
|---|---|---|
| `<file>...` | bắt buộc | `.png .jpg .jpeg .gif` (gif lấy khung đầu). Nhiều file → xử lý lần lượt |
| `--mode` | `trace` | |
| `--out` | cạnh file gốc, đổi đuôi `.svg` | thư mục khi nhiều file |
| `--colors` | 8 | số màu tối đa sau khi gộp, 2–32 |
| `--mono` | false | ảnh đen trắng một màu (logo, chữ ký): bỏ qua gộp màu, dùng ngưỡng sáng; màu tô lấy từ `--fill` (mặc định `#000000`) |
| `--threshold` | 128 | 0–255, chỉ cho `--mono` |
| `--simplify` | 1.0 | dung sai Ramer–Douglas–Peucker tính bằng pixel; lớn = ít điểm, file nhẹ, kém chi tiết |
| `--smooth` | false | làm mượt đường viền thành cong Bézier thay vì gấp khúc |
| `--despeckle` | 4 | bỏ vùng nhỏ hơn N pixel (nhiễu) |
| `--bg` | `keep` | `remove`: bỏ lớp nền (màu chiếm nhiều nhất ở viền ảnh hoặc vùng alpha=0) để SVG trong suốt |
| `--scale` | 1 | nhân kích thước hiển thị (viewBox giữ theo pixel gốc) |
| `--max-pixels` | 4_000_000 | ảnh lớn hơn bị từ chối trừ khi thêm `--force` (tracing tốn bộ nhớ) |

## 3. File
```
cmd/svg.go
internal/svgtrace/load.go        # decode ảnh -> []RGBA đã chuẩn hóa alpha
internal/svgtrace/quantize.go    # gộp màu
internal/svgtrace/mask.go        # tách từng màu thành bitmask, dọn nhiễu
internal/svgtrace/contour.go     # dò đường viền
internal/svgtrace/simplify.go    # Ramer–Douglas–Peucker
internal/svgtrace/smooth.go      # Catmull-Rom -> Bézier
internal/svgtrace/write.go       # xuất SVG
internal/svgtrace/embed.go
internal/svgtrace/testdata/{logo_2color.png, icon_flat.png, photo.jpg, transparent.png}
internal/svgtrace/svgtrace_test.go
```

## 4. Hợp đồng
```go
package svgtrace
type Options struct {
    Colors int; Mono bool; Threshold uint8; Fill string
    Simplify float64; Smooth bool; Despeckle int
    RemoveBG bool; Scale float64
}
type Result struct {
    Width, Height int
    Paths    int      // số <path>
    Colors   []string // màu đã dùng, "#rrggbb"
    Points   int      // tổng số điểm sau đơn giản hóa
    SVGBytes int
    Warnings []string
}
func Load(path string) (image.Image, error)
func Trace(img image.Image, opt Options) ([]byte, Result, error)
func Embed(srcPath string) ([]byte, Result, error)
```

## 5. Cách hoạt động (chế độ `trace`)

### 5.1 Chuẩn bị
1. Decode bằng `image/png`, `image/jpeg`, `image/gif` (import blank `_ "image/jpeg"`...). `image.Decode` tự nhận định dạng.
2. Chuyển về mảng `[]color.NRGBA` phẳng (`w*h`). Pixel có alpha < 128 coi là **trong suốt** (không thuộc màu nào); alpha ≥ 128 lấy màu, bỏ alpha.
3. Từ chối ảnh vượt `--max-pixels` (`SVG_IMAGE_TOO_LARGE`).

### 5.2 Gộp màu (quantize)
- `--mono`: pixel tối hơn ngưỡng (độ sáng `0.299R + 0.587G + 0.114B < threshold`) và không trong suốt → "mực"; còn lại là nền. Chỉ một lớp, kết thúc bước này.
- Còn lại: **median cut** thành tối đa `--colors` màu (đệ quy chia hộp RGB theo kênh có biên độ lớn nhất tại trung vị), mỗi pixel gán vào màu gần nhất trong bảng (khoảng cách Euclid trong RGB; đủ dùng, không cần Lab). Lưu `labels []uint8` kích thước `w*h` (255 = trong suốt).
- Đếm số màu gốc duy nhất (dừng đếm ở 5.000): nếu > 5.000 và không `--mono` → thêm cảnh báo "Ảnh có quá nhiều màu (giống ảnh chụp); kết quả có thể xấu và rất nặng. Hãy thử --colors thấp hơn hoặc dùng --mode embed."

### 5.3 Tách lớp và dọn nhiễu
Mỗi màu → bitmask `[]bool`. Sau đó:
1. Gắn nhãn thành phần liên thông bằng BFS/union-find (4-liên thông cho nền, 8-liên thông cho mực).
2. Bỏ thành phần nhỏ hơn `--despeckle` pixel (đổi thành màu láng giềng chiếm nhiều nhất quanh nó, để không tạo lỗ hổng).
3. `--bg remove`: màu nền = màu xuất hiện nhiều nhất trên **viền ảnh** (hoặc vùng trong suốt); bỏ lớp đó khỏi đầu ra.

### 5.4 Dò đường viền (contour)
Dùng **marching squares** trên lưới `(w+1)×(h+1)` cho từng bitmask — cho đường viền khép kín, xử lý được lỗ bên trong:
1. Duyệt từng ô 2×2 tính chỉ số 4-bit; tạo các đoạn thẳng giữa các cạnh.
2. Nối các đoạn thành **chu tuyến khép kín** (map điểm đầu → đoạn; mỗi đỉnh có đúng 2 đoạn nên ghép không mơ hồ; ô yên ngựa (chỉ số 5 và 10) quyết định tách/nối theo quy ước cố định để kết quả ổn định).
3. Mỗi chu tuyến là danh sách điểm tọa độ nguyên theo góc pixel. Hướng (thuận/ngược chiều kim đồng hồ) cho biết viền ngoài hay lỗ; dùng `fill-rule="evenodd"` nên không cần tách riêng — tất cả chu tuyến cùng một màu gộp vào **một `<path>`** (`d` gồm nhiều `M ... Z`).

### 5.5 Đơn giản hóa
- **Ramer–Douglas–Peucker** với dung sai `--simplify` trên từng chu tuyến (xử lý vòng khép kín bằng cách tách tại hai điểm xa nhau nhất). Chu tuyến còn < 3 điểm bị bỏ.
- Với `--smooth`: lấy các điểm đã đơn giản hóa, chuyển **Catmull-Rom → Bézier bậc 3** (`C`) cho đoạn cong; góc nhọn (góc giữa hai đoạn < 120°; bản đầu ghi 60° làm hình vuông 90° bị phồng tròn ra ngoài khung) giữ nguyên là đỉnh (dùng `L`) để chữ và hình vuông không bị tròn mất góc.

### 5.6 Xuất SVG
```xml
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 W H" width="W*scale" height="H*scale">
  <path fill="#1e88e5" fill-rule="evenodd" d="M10 10L50 10L50 40Z..."/>
  ...
</svg>
```
- Thứ tự path: từ màu chiếm diện tích lớn nhất (vẽ trước, nằm dưới) đến nhỏ nhất.
- Tọa độ làm tròn tối đa 2 chữ số thập phân, bỏ số 0 thừa; dùng lệnh tương đối (`l`, `c`) khi ngắn hơn.
- Không ghi chú thừa, không `<?xml?>`; xuống dòng giữa các `<path>`.
- Nếu kích thước SVG > 3× file ảnh gốc → thêm cảnh báo gợi ý tăng `--simplify` hoặc giảm `--colors`.

### 5.7 Chế độ `embed`
`<svg ... viewBox="0 0 W H"><image width="W" height="H" href="data:image/png;base64,..."/></svg>` (đọc kích thước bằng `image.DecodeConfig`, không decode toàn ảnh). Ghi rõ trong output rằng đây **không phải vector**.

### 5.8 An toàn
- Nội dung `<path>`/màu do chương trình tự sinh từ số, không chèn chuỗi từ ảnh → không có chuyện SVG chứa script.
- File đích tồn tại và không `--force` → `FILE_EXISTS`. Ghi vào `<tên>.svg.tmp` rồi đổi tên.
- Bộ nhớ ước tính ~ `w*h*(4 + 1 + 1)` byte + bitmask; chặn ở `--max-pixels`.

## 6. Output
```
> foxkit svg logo.png --colors 4 --smooth
logo.png (512×512, 4 màu) → logo.svg
  3 path · 184 điểm · 2.1 KB (gốc 38 KB)
  Màu: #1e88e5 #ffffff #0d47a1
```
Cảnh báo (stderr khi text mode):
```
! Ảnh có quá nhiều màu (giống ảnh chụp); kết quả có thể xấu và rất nặng.
```
JSON: `{ "ok": true, "data": { "items": [ { "source": "logo.png", "out": "logo.svg", "mode": "trace", "width": 512, "height": 512, "paths": 3, "points": 184, "colors": ["#1e88e5","#0d47a1"], "svgBytes": 2150, "sourceBytes": 38912, "warnings": [] } ] } }`

## 7. Lỗi
| code | Khi nào | exit | hint |
|---|---|---|---|
| `SVG_NO_INPUT` | | 2 | Ví dụ: foxkit svg logo.png |
| `SVG_FORMAT_UNSUPPORTED` | không phải png/jpg/gif | 2 | Chỉ hỗ trợ PNG, JPG, GIF. WebP cần chuyển sang PNG trước |
| `SVG_IMAGE_INVALID` | file hỏng | 1 | Thử mở file bằng trình xem ảnh để kiểm tra |
| `SVG_IMAGE_TOO_LARGE` | > `--max-pixels` | 1 | Thu nhỏ ảnh hoặc thêm --force |
| `SVG_COLORS_INVALID` | ngoài 2–32 | 2 | Ví dụ: --colors 8 |
| `SVG_EMPTY` | ảnh toàn trong suốt/không có vùng nào sau khi dọn nhiễu | 1 | Giảm --despeckle hoặc đổi --threshold |
| `FILE_EXISTS` | | 1 | Thêm --force để ghi đè |

Nhiều file: lỗi từng file ghi vào item, các file khác vẫn chạy; exit 1 nếu có ít nhất một lỗi.

## 8. Manifest
```json
{ "name": "svg", "title": "PNG → SVG", "description": "Chuyển PNG/JPG sang SVG (vector hóa logo, icon) hoặc nhúng ảnh", "category": "life",
  "args": [ { "name": "file", "type": "file", "required": true, "multiple": true, "label": "Ảnh PNG/JPG" } ],
  "flags": [
    { "name": "mode", "type": "enum", "options": ["trace","embed"], "default": "trace", "label": "Chế độ" },
    { "name": "out", "type": "string", "label": "Lưu ra" },
    { "name": "colors", "type": "int", "default": 8, "min": 2, "max": 32, "label": "Số màu" },
    { "name": "mono", "type": "bool", "default": false, "label": "Đen trắng (logo, chữ ký)" },
    { "name": "threshold", "type": "int", "default": 128, "min": 0, "max": 255, "label": "Ngưỡng đen/trắng" },
    { "name": "simplify", "type": "string", "default": "1.0", "label": "Đơn giản hóa (px)" },
    { "name": "smooth", "type": "bool", "default": false, "label": "Làm mượt đường cong" },
    { "name": "despeckle", "type": "int", "default": 4, "min": 0, "label": "Bỏ nhiễu nhỏ hơn (px)" },
    { "name": "bg", "type": "enum", "options": ["keep","remove"], "default": "keep", "label": "Nền" },
    { "name": "force", "type": "bool", "default": false, "label": "Ghi đè / ảnh lớn" }
  ], "admin": false, "longRunning": false, "dryRun": false, "output": "table" }
```
(`simplify` để kiểu `string` vì schema chưa có kiểu `float`; nếu muốn thêm `"type": "float"` cho launcher thì cập nhật foundation §9.1 và test manifest.)

## 9. Test
Tạo ảnh test **bằng code** trong test (không phụ thuộc file ngoài) bằng `image.NewNRGBA` + vẽ hình, kèm vài file thật trong `testdata/`.

Đơn vị:
- **Marching squares**: ảnh 1 hình chữ nhật đặc → đúng 1 chu tuyến 4 đỉnh sau `simplify`; hình có 1 lỗ → 2 chu tuyến (ngoài + trong); hai hình chạm góc chéo (ô yên ngựa) → kết quả ổn định giữa các lần chạy.
- **RDP**: đường thẳng nhiều điểm → còn 2 điểm; dung sai 0 giữ nguyên mọi điểm.
- **Median cut**: ảnh 3 màu thuần → đúng 3 màu, không sai lệch.
- **Despeckle**: chấm 2px bị bỏ, hình lớn giữ nguyên, không tạo lỗ.
- **Trong suốt**: pixel alpha=0 không sinh path; `--bg remove` bỏ lớp nền.

Tính đúng đắn (round-trip): render lại SVG ngược thành bitmap để so sánh là cần rasterizer (không có trong stdlib). Thay bằng kiểm **hình học**: diện tích tổng các chu tuyến của một màu (công thức Shoelace, trừ lỗ) so với số pixel của màu đó trong ảnh gốc → sai lệch < 2% với `--simplify 0.5`. Đây là test chính cho thuật toán.

Hồi quy: ảnh `logo_2color.png` → số path và số điểm nằm trong khoảng cố định (golden bound, không so từng byte).

Hiệu năng: ảnh 1000×1000 (1 triệu px), 8 màu trong dưới 3 giây, bộ nhớ dưới 300 MB (`go test -bench` + `-benchmem`).

Chạy thử (Windows):
```powershell
.\foxkit svg logo.png
.\foxkit svg logo.png --colors 4 --smooth --simplify 1.5
.\foxkit svg signature.png --mono --threshold 150 --bg remove
.\foxkit svg photo.jpg --mode embed
start logo.svg      # mở bằng trình duyệt để xem kết quả
```
Gợi ý kiểm bằng mắt: mở file `.svg` bằng trình duyệt, zoom 800% xem đường viền có mượt, và đặt cạnh ảnh gốc.

## 10. Tiêu chí hoàn thành
- [ ] Tiêu chí chung (gofmt, vet, test, build, manifest).
- [ ] Test hình học (Shoelace) pass.
- [ ] Logo 2 màu 512×512 ra SVG dưới 10 KB, nhìn giống ảnh gốc khi mở trong trình duyệt.
- [ ] Ảnh chụp bị cảnh báo thay vì im lặng ra file khổng lồ.
- [ ] SVG mở được trong trình duyệt, Inkscape (người dùng tự kiểm).

## 11. Ngoài phạm vi
Vector hóa ảnh chụp chất lượng cao (cần thuật toán như Potrace/VTracer), phát hiện hình cơ bản (tròn, chữ nhật) để thay path bằng `<circle>`/`<rect>`, nhận diện chữ (OCR), đường nét đơn (centerline) cho nét vẽ, SVG → PNG (cần rasterizer; xem backlog), WebP/AVIF/HEIC, gradient.

## 12. Hướng nâng cấp nếu không hài lòng chất lượng
Nếu tracing tự viết không đạt, phương án thay thế (cần cân nhắc dependency, hỏi người dùng trước):
1. Gọi **Potrace** có sẵn bằng `os/exec` (chuẩn cho ảnh đen trắng, chất lượng cao; người dùng phải cài `potrace.exe`).
2. Dùng **VTracer** (Rust, có CLI, hỗ trợ ảnh màu).
Lệnh `svg` nên có cờ `--engine builtin|potrace|vtracer` (mặc định `builtin`) để mở đường, nhưng **v1 chỉ làm `builtin`**.

## 13. Go học được
Package `image` và `image/color` (`NRGBA` vs `RGBA`, alpha nhân trước), decode nhiều định dạng bằng blank import, thuật toán trên lưới 2D, union-find, đệ quy RDP, công thức Shoelace để kiểm thử hình học, `strings.Builder`/`strconv.AppendFloat` để sinh chuỗi nhanh không cấp phát thừa, benchmark `-benchmem`. So sánh: giống làm việc với GD/Imagick của PHP, nhưng ở đây tự đi từng pixel nên thấy rõ chi phí bộ nhớ.
