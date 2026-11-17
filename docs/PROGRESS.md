# PROGRESS

Quy ước: `[x]` xong, `[ ]` chưa làm. Hash ghi ở commit kế tiếp (commit không tự biết hash của chính nó). Phiên bị ngắt thì đọc file này + `git log` rồi làm tiếp.

## Mục việc

- [x] GĐ0 foundation — 650bf57. Thêm: `Execute` tự dò `--json` trong args vì cobra lỗi trước khi parse cờ; lỗi kiểm đối số của cobra ("unknown command", "accepts ...") quy về exit 2. Test chống lệch đã thử xóa entry rồi hoàn tác.
- [x] uuid — 7dd3c3e. Spec sửa: stdlib Go 1.27 đã có package `uuid` → bỏ `internal/uuid`, không có `UUID_RANDOM_FAILED`.
- [x] pass — 7dd3c3e (chung commit với uuid; các mục nhỏ gộp commit vì manifest dùng chung một file). Spec sửa: vòng lặp đảm bảo 4 loại ký tự nâng 100 → 10000 lần (len=4 chỉ ~6% chuỗi đủ loại).
- [x] hash — fde3be2. Tiêu chí: vector chuẩn đạt; file lớn >1GB **chưa kiểm** (dùng `io.Copy` nên không nạp RAM) — thử: `.\foxkit hash D:\big.iso`. Thêm: `--check` không khớp in `khớp: false` rồi trả `HASH_MISMATCH`; item lỗi in ra stderr.
- [x] enc — (cùng commit với hash). `ENC_INPUT_TOO_LARGE` dùng chung `output.InputTooLarge` rồi đổi code.
- [x] ts — ad82327. Spec sửa: ví dụ output của `1763287727` là 2025-11-16 (Chủ Nhật), không phải 2026. **Chưa kiểm** trên máy không cài Go: thử copy `foxkit.exe` sang máy khác rồi `foxkit ts 0 --tz Asia/Ho_Chi_Minh` (đã nhúng `time/tzdata`).
- [x] case — (cùng commit với ts). `kebab` và `slug` khác nhau ở chỗ slug chỉ giữ `a-z0-9`.
- [x] port — a20367c. Đã thử thật trên listener tự tạo: find/json/kill/trống đều đúng; port 445 → `PORT_SYSTEM_PROCESS`. Lệch spec nhỏ: `List` gọi `tasklist` một lần cho mọi PID thay vì mỗi PID một lần. **Chưa kiểm**: `NEED_ADMIN` (kill process của user khác/service) — thử `.\foxkit port 3306 --kill` khi MySQL chạy dưới service mà terminal không admin.
- [x] env — (cùng commit với port). Có `ParseError` (kèm số dòng), `Set`/`WriteTo` giữ nguyên byte (kể cả CRLF) cho `lara key` ở GĐ5. **Chưa kiểm** trên dự án Laravel thật: `.\foxkit env diff D:\Projects\<dự-án>`.
- [ ] L1 launcher (nhiều commit nhỏ). Môi trường: MSVC Build Tools ở `D:\VS\BuildTools`, npm/cargo có mạng. Tauri chọn `2.x` (crate mới nhất trên crates.io là 3.0 alpha, spec chốt v2).
  - [x] L1.0 khung — commit "launcher: scaffold". Windows crate 0.62, sha2 0.11 (cargo chọn bản mới nhất tương thích). Icon tạm sinh từ chữ "fx" (thay bằng icon thật sau).
  - [x] L1.1 lib TS thuần + vitest (47 test pass): tokenize, fuzzy, resolve, ghost, placeholders, form, thread, manifest. `thread.ts` chỉ là phần xem/đồng bộ (`ThreadStore` + backend giả để test), luồng thật nằm ở Rust (L1.2).
  - [x] L1.2 Rust: config, runner (run/stream/stop), elevate, thread state — 14 test + clippy sạch. Sửa so với spec: lệnh Tauri trả `Result<_, CmdError>` (có code/message/hint) thay vì `String`; `run`/`run_stream` nhận thêm `exec: Option<ExecSpec>` để chạy lệnh ngoài; hàm vào của lib đổi tên `start()` vì trùng tên lệnh `run`. **Chưa kiểm tay**: UAC thật (`run_elevated`) và Stop của stream — làm ở bước thử tay mục 10.3.
  - [x] L1.3 Rust: context.rs — 18 test pass, clippy sạch. Logic thuần (tách tiêu đề VS Code, storage.json, URI) có test; phần Win32/COM (`IShellWindows`, `Folder2::Self_`) **chưa kiểm tay**: mục 10.3 #12-14 (Explorer chọn 2 file, VS Code 2 cửa sổ, Notepad). Ngân sách 200 ms chạy ở thread riêng.
  - [x] L1.4 Rust: scan.rs + `launcher/schema/foxkit-command.schema.json` — 31 test Rust pass, clippy sạch. Validator JSON Schema trong test là bản tối giản tự viết (không thêm crate). Đuôi program cho phép: .exe/.bat/.cmd/.com hoặc không đuôi (`python3.11` có đuôi "11" nên bị từ chối, hãy dùng `python`). Lệnh Tauri thêm: add_folder, forget_folder, folders_info, scan_log, trust_check, trust_allow (ngoài danh sách spec, cần cho `:add`/`:forget`/`:folders` và bước tin cậy).
  - [x] L1.5 UI ô nhanh: Session (điều khiển chung), Composer, Suggestions, ArgHint, HintBar, ResultView, App.vue + interpret/copy (59 test vitest, vue-tsc sạch). Mở rộng nhỏ: copy `uuid`/`pass` ra mỗi giá trị một dòng. **Chưa chạy được trong cửa sổ** cho tới L1.7 (cần `hide_window`, `show_main`, `suspend_autohide`, sự kiện `bar-shown`).
  - [ ] L1.6 UI giao diện chính 4 tab
  - [ ] L1.7 lib.rs: tray, hotkey, single-instance, autostart, hai cửa sổ; build `tauri build`
- [ ] GĐ2: size · dup · rename · find · temp · prune · todo
- [ ] GĐ3: ps · kill · which · path · hosts · ip · dns · wifi
- [ ] GĐ4: json · jwt · http · tcp · serve
- [ ] GĐ5: proj · lara · repos · branches
- [ ] GĐ6: core/winapi · disk · sysinfo · path-edit · startup · svc · clip · lock · nâng cấp port/ps/kill
- [ ] GĐ7: fake · csv · diff · down · zip · sync · svg

## Chưa kiểm / để bạn tự thử trên Windows

(điền dần)

## Ghi chú Go (so với PHP/JS)

- **`internal/`**: Go cấm package ngoài module import thư mục này, như `private` cho cả thư mục.
- **`errors.As`**: ≈ `catch (MyException $e)`; kiểm tra lỗi (có thể đã bị bọc) có phải kiểu `*output.Error` không.
- **`go:embed`**: nhúng file vào binary lúc build, ≈ bundle JSON vào file build của Vite.
- **`io.Writer`**: truyền "nơi in" vào hàm (stdout thật hoặc buffer khi test), ≈ dependency injection.
- **`[16]byte` vs slice**: mảng có độ dài cố định là một phần của kiểu, so sánh bằng `==` được (≈ giá trị, không phải tham chiếu như array PHP/JS).
- **`crypto/rand` vs `math/rand`**: ≈ `random_bytes()` vs `rand()` của PHP; mật khẩu/UUID luôn dùng `crypto/rand`. `rand.Int(max)` không lệch phân bố, còn `byte % n` thì có.
