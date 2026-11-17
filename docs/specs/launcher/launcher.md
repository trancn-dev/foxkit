# Spec: foxkit launcher (giao diện chính + ô lệnh nhanh bật bằng phím tắt, kiểu Spotlight)

> Nhóm: **launcher** (app desktop, không phải lệnh CLI) · Giai đoạn: L1 (làm song song, bắt đầu sau GĐ1) · Trạng thái: chưa làm
> Cần trước: [core/foundation.md](../core/foundation.md) (lệnh `manifest`, envelope JSON, exit code) và ít nhất các lệnh GĐ1 để có cái chạy thử.
> Khó: **4** — phần mới với người dùng là Rust/Tauri; phần UI là Vue nên quen tay. Làm đúng phạm vi bên dưới, không thêm tính năng ngoài spec.

---

## 1. Mục đích

App có **hai chế độ**, cùng một tiến trình, chạy ngầm trên khay hệ thống (tray):

1. **Giao diện chính** (cửa sổ thường, mục 6.10): mở app lên là vào **Chat đầy đủ**, gõ lệnh và xem kết quả như một cuộc chat bình thường. Có thêm các tab để duyệt lệnh bằng form, thêm/quét thư mục lệnh ngoài, chỉnh cài đặt. Bấm X hoặc thu nhỏ thì **thu về tray**, app vẫn chạy.
2. **Ô lệnh nhanh, chat mini** (mục 6.2, 6.4): khi app đã thu về tray, bấm tổ hợp phím toàn cục (hoặc click icon tray) thì hiện một ô chat nhỏ ở giữa màn hình, gõ lệnh foxkit (vd `port 8000 --kill`) và Enter. Đây là **cùng một cuộc chat** với tab Chat của giao diện chính, chỉ thu gọn: lệnh nào chạy ở đâu cũng hiện ở cả hai nơi. Có nút "Mở rộng" để nhảy sang giao diện chính, và nút "Thu nhỏ" ở giao diện chính để về lại ô nhỏ. Esc hoặc bấm ra ngoài thì ẩn ô nhỏ.

Giống cách app Claude trên desktop: thu nhỏ thì bấm vào là ra chat nhỏ, mở hẳn app thì chat đầy đủ.

Ba điểm làm ô lệnh đỡ phải gõ tay:
- **Nhận diện ngữ cảnh lúc bấm phím:** app đang focus (VS Code, Explorer...) và thư mục/file hiện tại của app đó, dùng để tự điền đối số (mục 6.8).
- **Gợi ý gõ tiếp:** ghost text mờ ngay sau con trỏ, cộng một dòng ví dụ mẫu mờ phía trên ô nhập (mục 6.4).
- **Copy kết quả** vào clipboard (mục 7).

Gợi ý chỉ là phần gõ tiếp theo cú pháp lệnh; không có gợi ý theo lệnh hay dùng hay lịch sử.

"Chat" trong yêu cầu được hiểu là **ô gõ lệnh dạng chat mini** (luồng lệnh → kết quả trong phiên đang chạy), không phải chat với AI.

## 2. Công nghệ (đã chốt)

**Tauri v2 + Vue 3 (`<script setup>`) + TypeScript + Vite.**

Lý do ngắn:
- Tauri v2 có plugin chính thức cho đúng 4 thứ cần: phím tắt toàn cục (`global-shortcut`), khởi động cùng Windows (`autostart`), chỉ một bản chạy (`single-instance`), tray (có sẵn trong core, feature `tray-icon`). Bản cài ~5–10 MB, RAM thấp vì dùng WebView2 có sẵn trên Windows 10/11.
- Không chọn Wails (Go): Wails v2 không có global hotkey/tray chính thức, v3 còn alpha. Không chọn Electron: nặng (~150 MB) cho một ô nhập.
- Vue 3 vì người dùng quen Vue 2; Composition API gần với Options API hơn là học React. TypeScript để có kiểu cho manifest, giảm lỗi khi Claude Code viết phần gợi ý/validate.

Phụ thuộc (ghi vào roadmap mục 5):

| Gói | Lý do |
|---|---|
| `tauri` 2.x (feature `tray-icon`) | Khung app |
| `tauri-plugin-global-shortcut` | Phím tắt toàn cục (bọc `RegisterHotKey`) |
| `tauri-plugin-autostart` | Khởi động cùng Windows (ghi `HKCU\...\Run`, không cần admin) |
| `tauri-plugin-single-instance` | Mở app lần 2 thì hiện cửa sổ bản đang chạy thay vì chạy thêm |
| `serde`, `serde_json` | Đọc config, trả kết quả cho frontend |
| `windows` (crate chính thức của Microsoft, bật feature theo nhu cầu: `Win32_UI_WindowsAndMessaging`, `Win32_System_Threading`, `Win32_UI_Shell`, `Win32_System_Com`) | `ShellExecuteExW` verb `runas`; lấy cửa sổ đang focus; hỏi Explorer thư mục/file đang chọn qua COM `IShellWindows` (`windows-sys` không có wrapper COM) |
| `tauri-plugin-clipboard-manager` | Đọc/ghi clipboard ổn định (`navigator.clipboard.readText` trong WebView2 bị hỏi quyền) |
| `sha2` | Băm SHA-256 file khai báo lệnh ngoài để nhớ "đã cho phép" (6.9); thư viện chuẩn Rust không có băm |
| `tauri-plugin-dialog` | Hộp thoại chọn thư mục cho `:add` |
| `url` | Đổi URI `file:///c%3A/...` trong cấu hình VS Code thành đường dẫn Windows |
| `vue` 3, `vite`, `typescript`, `@tauri-apps/api`, các gói JS của 3 plugin trên | Frontend |
| `vitest` (dev) | Test tokenizer/gợi ý/validate |

Không dùng thư viện fuzzy search hay UI kit: tự viết bộ chấm điểm ~50 dòng, CSS tay.

## 3. Phím tắt

**Mặc định: `Ctrl+Alt+Space`**, đổi được trong config.

- `Win+F1` không dùng được: Windows giữ cho Help (mở trình duyệt tìm "get help"), `RegisterHotKey` thất bại hoặc cả hai cùng chạy. Phần lớn tổ hợp `Win+...` đều bị Windows giữ.
- `Alt+Space` không dùng: là menu hệ thống của mọi cửa sổ và PowerToys Run dùng. `Win+Alt+Space` là của PowerToys Command Palette.
- `Ctrl+Space` hay bị bộ gõ (IME) dùng.

Đăng ký thất bại (tổ hợp đã bị app khác giữ): app vẫn chạy, hiện thông báo trên tray "Không đăng ký được phím tắt Ctrl+Alt+Space, có thể app khác đang dùng. Sửa `hotkey` trong file cấu hình rồi chọn Tải lại." Người dùng vẫn mở được ô lệnh bằng click icon tray.

## 4. Vị trí file

Launcher nằm trong cùng repo foxkit, thư mục `launcher/` (monorepo, để manifest và launcher đổi cùng một PR):

```
launcher/
├── package.json, vite.config.ts, tsconfig.json
├── index.html                   # ô lệnh nhanh (cửa sổ `bar`)
├── main.html                    # giao diện chính (cửa sổ `main`); Vite chạy dạng nhiều trang
├── src/                         # Vue
│   ├── main.ts
│   ├── App.vue                  # ô lệnh nhanh: ô nhập + gợi ý + luồng hội thoại mini
│   ├── main/                    # giao diện chính (mục 6.10)
│   │   ├── MainApp.vue          # khung 5 tab
│   │   ├── ChatTab.vue          # chat đầy đủ (cùng luồng với ô lệnh nhanh)
│   │   ├── CommandsTab.vue      # danh sách lệnh + form + kết quả
│   │   ├── CommandForm.vue      # form sinh từ args/flags của manifest
│   │   ├── FoldersTab.vue       # thư mục lệnh ngoài: thêm / quét lại / bỏ / log
│   │   └── SettingsTab.vue      # hotkey, autostart, autoCopy, context, foxkitPath
│   ├── components/
│   │   ├── Composer.vue         # ô nhập + ghost text + gợi ý, dùng chung cho ô nhanh và tab Chat
│   │   ├── Suggestions.vue
│   │   ├── ArgHint.vue          # dòng gợi ý cú pháp "port <port> [--kill] [--list]"
│   │   ├── ResultView.vue       # hiển thị theo output: kv / table / text / stream / error
│   │   └── HintBar.vue          # dòng mờ phía trên ô nhập: app/thư mục ngữ cảnh + ví dụ mẫu
│   ├── lib/
│   │   ├── manifest.ts          # kiểu TS của schema 1 + load
│   │   ├── tokenize.ts          # tách chuỗi gõ vào thành mảng tham số (có nháy kép)
│   │   ├── resolve.ts           # tìm lệnh (kể cả "hosts add"), gợi ý, validate
│   │   ├── fuzzy.ts             # chấm điểm khớp, bỏ dấu tiếng Việt
│   │   ├── ghost.ts             # tính ghost text (phần gõ tiếp) và ví dụ mẫu
│   │   ├── placeholders.ts      # mở rộng @dir @file @files @clip trước khi chạy
│   │   ├── form.ts              # giá trị form → mảng args (cùng quy tắc với gõ tay)
│   │   ├── thread.ts            # luồng chat dùng chung (đồng bộ qua Rust, mục 6.2)
│   │   └── *.test.ts
│   └── style.css
├── schema/
│   └── foxkit-command.schema.json   # JSON Schema của file foxkit.json
└── src-tauri/
    ├── Cargo.toml, tauri.conf.json, capabilities/default.json
    ├── icons/
    └── src/
        ├── main.rs              # gọi lib::run()
        ├── lib.rs               # dựng app: plugin, tray, cửa sổ, hotkey
        ├── config.rs            # đọc/ghi config.json
        ├── runner.rs            # chạy foxkit: thường, stream, dừng
        ├── scan.rs              # quét thư mục lệnh ngoài, scan-cache.json, trusted.json (mục 6.9)
        ├── context.rs           # chụp app đang focus + thư mục/file hiện tại lúc bấm phím (chỉ Windows, kèm context_other.rs)
        └── elevate.rs           # chạy lại bằng quyền admin (chỉ Windows)
```

## 5. Hợp đồng

### 5.1 Config — `%APPDATA%\foxkit-launcher\config.json`

```json
{
  "hotkey": "Ctrl+Alt+Space",
  "foxkitPath": "",
  "autostart": true,
  "autoCopy": false,
  "context": true,
  "folders": [],
  "appHints": []
}
```

- File chưa có thì tạo với giá trị mặc định trên. JSON hỏng thì dùng mặc định và báo trên tray, **không** ghi đè file người dùng.
- `autoCopy`: `true` thì mỗi lần chạy thành công tự copy kết quả (quy tắc ở mục 7).
- `context`: `false` thì tắt hẳn việc đọc app/thư mục/clipboard (mục 6.8); `@dir` v.v. báo `CONTEXT_MISSING`.
- `folders`: các thư mục quét lệnh ngoài (mục 6.9), quản lý bằng `:add` / `:forget`; launcher ghi lại file này khi đổi (giữ nguyên các field khác).
- `appHints`: gợi ý theo app, mặc định rỗng (= không gợi ý gì). Mỗi phần tử `{ "app": "chrome.exe", "commands": ["http", "dns"] }`; so tên exe không phân biệt hoa thường, thiếu `.exe` thì tự thêm. Sửa ở tab "Ứng dụng" (mục 6.10).
- `foxkitPath` rỗng thì tìm theo thứ tự: `foxkit.exe` cùng thư mục với launcher (bản cài kèm) → `foxkit.exe` trong `PATH`. Không thấy thì ô lệnh hiện lỗi `FOXKIT_NOT_FOUND` (mục 8).

### 5.2 Tauri command (Rust → frontend; dùng chung cho cả hai cửa sổ)

```rust
#[derive(Serialize)]
struct RunResult { exit_code: i32, stdout: String, stderr: String }

#[tauri::command] async fn load_manifest() -> Result<serde_json::Value, String>
#[tauri::command] async fn run(args: Vec<String>, cwd: Option<String>) -> Result<RunResult, String>   // tự thêm --json; cwd = thư mục làm việc của process
#[tauri::command] async fn run_stream(args: Vec<String>, cwd: Option<String>, on_line: Channel<StreamMsg>) -> Result<u32, String> // trả run id
#[tauri::command] fn stop(run_id: u32) -> Result<(), String>
#[tauri::command] async fn run_elevated(args: Vec<String>, cwd: Option<String>) -> Result<RunResult, String>
#[tauri::command] async fn scan(folders: Vec<String>) -> Result<ScanReport, String>          // quét, ghi scan-cache.json (6.9)
#[tauri::command] fn get_config() -> Config
#[tauri::command] fn save_config(cfg: Config) -> Result<(), String>                     // ghi file + đăng ký lại hotkey + đồng bộ autostart; lỗi hotkey trả HOTKEY_FAILED, giữ cấu hình cũ
#[tauri::command] fn thread_list() -> Vec<ThreadItem>                                      // luồng chat dùng chung (6.2)
#[tauri::command] fn thread_add(item: ThreadItem) -> u32                                   // trả id; phát sự kiện thread-changed
#[tauri::command] fn thread_update(id: u32, item: ThreadItem)                              // cập nhật mục (vd Đang chạy → kết quả)
#[tauri::command] fn thread_remove(id: u32)                                              // xóa đúng một mục khỏi luồng
#[tauri::command] fn thread_clear()                                                         // xóa cả luồng
#[tauri::command] fn show_main(app: tauri::AppHandle)                                    // hiện giao diện chính
#[tauri::command] fn get_context() -> Context                                            // ảnh chụp ngữ cảnh lúc bấm phím (mục 6.8)
#[tauri::command] fn hide_window(window: tauri::WebviewWindow) -> Result<(), String>

#[derive(Serialize)]
struct ScanReport { added: u32, changed: u32, removed: u32, skipped_files: u32, folders_missing: Vec<String> }

#[derive(Serialize)]
struct Context {
    app: String,            // tên thân thiện: "VS Code", "Explorer", hoặc tên exe bỏ .exe
    exe: String,            // đường dẫn đầy đủ của exe
    title: String,          // tiêu đề cửa sổ
    dir: Option<String>,    // thư mục hiện tại nếu xác định được
    files: Vec<String>,     // file/thư mục đang chọn (Explorer), có thể rỗng
    source: String,         // "explorer" | "vscode" | "generic" | "none"
}

#[derive(Serialize)]
#[serde(tag = "kind")]
enum StreamMsg { Line { text: String }, Stderr { text: String }, Exit { code: i32 } }
```

> **Đã chỉnh khi triển khai:** các lệnh trả `Result<_, CmdError>` với `CmdError { code, message, hint }` (thay cho `String`) để frontend hiện đúng mã lỗi ở mục 8; `run`/`run_stream` nhận thêm `exec: Option<ExecSpec>` để chạy lệnh ngoài; thêm các lệnh `show_bar`, `suspend_autohide`, `check_foxkit`, `add_folder`, `forget_folder`, `folders_info`, `scan_log`, `trust_check`, `trust_allow`; hàm vào của thư viện là `start()` (tên `run` trùng lệnh `run`).

Frontend không bao giờ ghép chuỗi lệnh: luôn truyền `args` là mảng, Rust đưa thẳng vào `Command::args`, không qua shell (không có lỗ chèn lệnh).

### 5.3 Kiểu manifest phía TS

Theo đúng schema 1 ở [foundation mục 9.1](../core/foundation.md). `load_manifest` gọi `foxkit manifest --json`, lấy `data`. `schema != 1` thì báo lỗi `MANIFEST_UNSUPPORTED`.

## 6. Cách hoạt động

### 6.1 Khởi động
1. `single-instance`: nếu đã có bản chạy, bản mới gửi tín hiệu để bản cũ **hiện giao diện chính**, rồi thoát (mở app từ Start là vào giao diện chính).
2. Đọc config, đăng ký hotkey, tạo tray, tạo hai cửa sổ `bar` (ô lệnh nhanh) và `main` (giao diện chính) đều **ẩn sẵn** (`visible: false`) để bấm phím là hiện ngay, không chờ khởi tạo WebView.
3. Gọi `load_manifest` một lần, giữ trong bộ nhớ. Chỉ tải lại khi chọn "Tải lại" trên tray.
4. Đồng bộ `autostart` trong config với plugin (bật/tắt mục Run trong registry). Autostart khởi chạy với tham số `--hidden` → không hiện cửa sổ.

### 6.2 Cửa sổ ô lệnh nhanh (`bar`)
- `tauri.conf.json`: `decorations: false`, `alwaysOnTop: true`, `skipTaskbar: true`, `resizable: false`, `transparent: false`, rộng 680 px, cao ban đầu ~90 px: dòng mờ `HintBar` (24 px) nằm **phía trên** ô nhập (64 px).
- Hiện: **chụp ngữ cảnh trước** (mục 6.8, trước khi cửa sổ launcher cướp focus), rồi đặt ở giữa ngang, cách mép trên 25% chiều cao của **màn hình đang có chuột**, `show()` + `set_focus()`, chọn toàn bộ chữ cũ trong ô (gõ đè được ngay).
- **Mặc định chỉ có ô nhập** (dòng `HintBar` + ô nhập, ~78 px). Chiều cao luôn bằng đúng nội dung (đo bằng `ResizeObserver`, nên đổi theo ngữ cảnh tải xong, gợi ý hiện/ẩn, kết quả mới), tối đa 480 px; chỉ khi vượt 480 px mới có thanh cuộn. Kết quả của các lần mở trước **không hiện lại** ở ô nhanh (vẫn còn ở tab Chat); chỉ kết quả của lệnh vừa chạy trong lần mở này mới hiện bên dưới để copy.
- **Luồng chat dùng chung** (`thread.ts`): ô nhanh và tab Chat của giao diện chính là **hai cách xem cùng một luồng**. Luồng nằm trong state của Rust (`Mutex<Vec<ThreadItem>>`), mỗi thay đổi phát sự kiện `thread-changed` để cả hai cửa sổ cập nhật. Mỗi mục gồm dòng lệnh đã gõ (bấm vào thì điền lại vào ô nhập) và kết quả (`ResultView`, mục 7) hoặc lỗi; lệnh đang chạy hiện "Đang chạy…". Giữ tối đa 100 mục, cũ nhất tự bị loại; chỉ nằm trong bộ nhớ, mất khi thoát app, **không ghi ra đĩa**. `:clear` xóa luồng; nút **Xóa** cạnh Copy/JSON chỉ xóa đúng lệnh đó, nút **Xóa chat** ở thanh trên tab Chat xóa cả luồng. Luồng không dùng để gợi ý lệnh hay gọi lại bằng phím ↑.
- Ô nhanh chỉ hiện **20 mục mới nhất**, mới nhất ở trên, ngay dưới ô nhập. Nút **Mở rộng** (góc phải ô nhập) ẩn ô nhanh và hiện giao diện chính ở tab Chat, giữ nguyên chữ đang gõ dở.
- Ẩn khi: Esc (lúc không có gợi ý đang mở), cửa sổ mất focus, bấm lại hotkey khi đang hiện. Ẩn **không** dừng lệnh đang chạy; mở lại thấy luồng cũ.

### 6.3 Tray
Icon foxkit, click trái = hiện ô lệnh nhanh, double-click = hiện giao diện chính. Menu chuột phải:
- `Mở ô lệnh (Ctrl+Alt+Space)` — hiện chữ hotkey đang dùng.
- `Mở giao diện chính`.
- `Quét lại thư mục lệnh` — như `:scan` (mục 6.9).
- `Khởi động cùng Windows` — checkbox, đổi xong ghi config.
- `Tải lại` — đọc lại config, đăng ký lại hotkey, tải lại manifest.
- `Mở file cấu hình` — mở `config.json` bằng app mặc định (cài đặt thường dùng đã có trong giao diện chính).
- `Thoát`.

### 6.4 Gõ lệnh và gợi ý
1. `tokenize(input)`: tách theo khoảng trắng, nháy kép gom thành một token (`"C:\My Files"`), `\` giữ nguyên (đường dẫn Windows). Thiếu nháy đóng thì coi như đóng ở cuối và báo nhẹ trong dòng gợi ý.
   Nếu token đầu là `foxkit` (người dùng quen gõ như ở terminal) thì bỏ token đó, coi như không có.
2. `resolve(tokens)`: tìm entry manifest có `name` là **tiền tố dài nhất** của tokens (`hosts add x` khớp `"hosts add"`, không khớp `"hosts"` vì lệnh cha không có entry).
3. Chưa khớp lệnh nào → danh sách gợi ý lệnh, chấm điểm bằng `fuzzy.ts` trên `name`, `title`, `description` (đã bỏ dấu, chữ thường): khớp đầu `name` > khớp đầu từ > chứa chuỗi con > khớp rời rạc theo thứ tự. Gõ 1–2 ký tự thì chỉ lấy lệnh có `name` bắt đầu bằng chuỗi đó hoặc có từ trong `title` bắt đầu bằng chuỗi đó (gõ `d` ra `dup`, `dns`, `disk`, `diff`, `down`... chứ không ra mọi lệnh có chữ d ở giữa); từ 3 ký tự mới thêm khớp chuỗi con và khớp rời rạc. Tối đa 8 dòng, mỗi dòng: `name` đậm, `title`, `description` mờ. **Ô trống thì không liệt kê gì** (giữ ô nhanh gọn), trừ khi app đang focus có cấu hình `appHints` (mục 6.4a).
4. Đã khớp lệnh → ẩn danh sách lệnh, hiện `ArgHint`: cú pháp sinh từ `args`/`flags`, tô đậm tham số đang gõ, kèm `label`, kiểu, `min–max`. Token đang gõ bắt đầu bằng `--` → gợi ý tên cờ chưa dùng. Tham số kiểu `enum` → gợi ý `options`.
5. Phím: `↑/↓` chọn gợi ý · `Tab` điền gợi ý đang chọn (thêm dấu cách) · `Enter` chạy (nếu đang mở danh sách lệnh thì điền lệnh trước, Enter lần nữa mới chạy) · `Esc` đóng gợi ý, Esc lần nữa ẩn cửa sổ.
6. **Ghost text (gợi ý gõ tiếp):** chữ mờ nằm ngay sau con trỏ, chỉ hiện khi con trỏ ở cuối ô. `Tab` hoặc `→` chấp nhận, gõ tiếp thì tự cập nhật. Chỉ gợi ý phần **tiếp theo của cú pháp**, tính bởi `ghost.ts` theo thứ tự ưu tiên:

   | Tình huống | Ghost |
   |---|---|
   | Đang gõ dở tên lệnh, trùng đầu `name` của gợi ý đầu tiên | Phần còn lại của tên: gõ `po` → ghost `rt` |
   | Đã đủ tên lệnh, arg kế tiếp là `dir` và biết thư mục hiện tại | Đường dẫn thư mục hiện tại (có nháy kép nếu có dấu cách) |
   | Arg kế tiếp là `file` và có file đang chọn (Explorer) | File đầu tiên đang chọn; với `allowDir` mà không có file thì dùng thư mục hiện tại |
   | Arg kế tiếp kiểu `int` và clipboard là một số nguyên trong `min–max` | Số đó |
   | Arg kế tiếp `enum` | Giá trị `options` đầu tiên khớp với phần đã gõ |
   | Còn arg `required` chưa gõ, không có giá trị ngữ cảnh | Tên arg dạng `<label>` (mờ hơn, **không** nhận bằng Tab, chỉ nhắc) |
   | Đang gõ dở cờ (`--ki`) | Phần còn lại của cờ khớp đầu tiên: ghost `ll` |
   | Đủ arg bắt buộc | Cờ đầu tiên chưa dùng, dạng `--tên` (nhận bằng Tab); gõ `-` thì cả danh sách cờ mở ra để chọn |

   Giá trị lấy từ ngữ cảnh/clipboard được **ghi cụ thể ra ô** khi nhấn Tab (không để dạng `@dir`), để người dùng thấy đúng cái sẽ chạy. Không đoán giá trị cho `string`/`text` tự do.
7. **Dòng ví dụ mờ phía trên ô nhập (`HintBar`):** bên trái là chip ngữ cảnh (mục 6.8), bên phải là ví dụ:
   - Chưa khớp lệnh: ví dụ của lệnh đầu danh sách gợi ý; ô trống thì chọn ngẫu nhiên một ví dụ mỗi lần mở cửa sổ, **chỉ trong các lệnh của `appHints`** nếu app đang focus có cấu hình, nếu không thì trong mọi lệnh.
   - Đã khớp lệnh: ví dụ thứ nhất của lệnh (`examples[0]`); nếu người dùng đã gõ khớp một ví dụ thì chuyển sang ví dụ kế tiếp.
   - Hiện dạng `Ví dụ: port 8000 --kill — Kill process đang chiếm port 8000`; `Tab` ở ô trống hoặc khi chỉ có tên lệnh sẽ điền ví dụ vào ô.
   - Lệnh chưa có `examples` trong manifest thì hiện dòng cú pháp sinh từ `args`/`flags` (đã có ở `ArgHint`), không bịa ví dụ.

### 6.4a Gợi ý theo app (`appcmds.ts`)
- Cấu hình ở `config.appHints` (mục 5.1), sửa ở tab "Ứng dụng". **Mặc định không gợi ý gì**; chỉ khi app đang focus (so tên exe trong ngữ cảnh 6.8) có dòng cấu hình thì ô trống mới hiện sẵn đúng các lệnh đã chọn, theo thứ tự đã chọn (tên lệnh không có trong manifest thì bỏ qua).
- Khi đã gõ chữ, gợi ý theo cách thường (mục 6.4); `appHints` chỉ ảnh hưởng ô trống và dòng ví dụ mờ.

### 6.5 Validate trước khi chạy
Dựa trên manifest, báo lỗi ngay dưới ô nhập (chữ đỏ, không chạy):
- Thiếu arg `required`; `int` không phải số hoặc ngoài `min/max`; `enum` không thuộc `options`; cờ không có trong `flags` của lệnh; arg thừa khi arg cuối không `multiple`.
- Chỉ hỗ trợ cờ dài (`--kill`, `--out x`, `--out=x`). Manifest không khai báo cờ ngắn nên launcher không đoán.
- `--json` người dùng tự gõ thì bỏ đi (launcher luôn tự thêm). `--apply` hợp lệ khi lệnh có `dryRun` (đã nằm trong `flags` của manifest).
- Ràng buộc chéo (vd `--list` không đi cùng `<port>`) **không** kiểm ở launcher: để foxkit trả exit 2, launcher hiện lỗi như bình thường.

### 6.6 Chạy lệnh
1. `confirm` (của lệnh hoặc của cờ đang bật) → hiện dòng xác nhận ngay trong cửa sổ: "Kill process đang dùng port này? Enter = Có · Esc = Không". Không dùng hộp thoại hệ thống (cửa sổ sẽ mất focus và tự ẩn).
2. `dryRun: true` và người dùng chưa gõ `--apply` → chạy bản xem trước, hiện kết quả kèm nút **Áp dụng** (`Ctrl+Enter`) → chạy lại cùng args + `--apply`.
3. Rust spawn: `Command::new(foxkit).args(args).arg("--json")`, `creation_flags(0x08000000)` (`CREATE_NO_WINDOW`), stdin null, đọc stdout/stderr dạng byte → `String::from_utf8_lossy` (foxkit in UTF-8).
4. Lệnh thường (`run`): chờ xong, trả `exit_code/stdout/stderr`. Không có timeout; trong lúc chạy hiện "Đang chạy…" và nút **Dừng**.
5. Lệnh chạy lâu (`longRunning: true`, hoặc cờ ghi ở `longRunningFlag` đang bật) → `run_stream`: mỗi dòng stdout gửi qua `Channel` dạng `Line`, stderr dạng `Stderr`, kết thúc gửi `Exit`. **Dừng** → `stop(run_id)` gọi `Child::kill()`.
6. Mỗi lúc chỉ một lệnh chạy. Đang chạy mà Enter lệnh khác → hiện "Đang có lệnh chạy. Enter = Dừng và chạy lệnh mới · Esc = Thôi".
7. Đọc kết quả theo exit code:
   - `0` → parse stdout là envelope `{ok:true,data}` → `ResultView` theo `output`.
   - `1`, `2` → parse stderr là envelope lỗi → hiện `message` (đỏ) + `hint` (mờ).
   - `3` → như trên, thêm nút **Chạy lại với quyền admin** (mục 6.7).
   - stdout/stderr không phải JSON hợp lệ → hiện nguyên văn trong khối `pre`, kèm lỗi `BAD_OUTPUT`.
8. `Shift+Enter` = chạy rồi **copy kết quả và ẩn cửa sổ** (hợp với `uuid`, `pass`, `ts`...). Nếu lệnh lỗi hoặc cần xác nhận thì không ẩn, hiện lỗi/xác nhận như bình thường.
9. Thư mục làm việc: nếu có `Context.dir` thì spawn với `current_dir = dir` (lệnh nào mặc định theo `.` sẽ chạy đúng thư mục đang mở); không có thì dùng thư mục của launcher.

### 6.7 Chạy lại bằng quyền admin (`elevate.rs`)
Process chạy bằng `runas` không cho đọc stdout qua pipe, nên ghi ra file tạm:
1. Tạo thư mục tạm `%TEMP%\foxkit-launcher\<run-id>\`.
2. `ShellExecuteExW` với `lpVerb = "runas"`, `lpFile = "cmd.exe"`, `lpParameters = /d /s /c ""<foxkit>" <args> --json 1>"<dir>\out.json" 2>"<dir>\err.json""`, `nShow = SW_HIDE`, `fMask = SEE_MASK_NOCLOSEPROCESS`.
3. `WaitForSingleObject` → `GetExitCodeProcess` → đọc 2 file → xóa thư mục tạm → trả `RunResult` như lệnh thường.
4. Mỗi arg được bọc nháy kép. Arg chứa ký tự đặc biệt của cmd (`" & | < > ^ %`) → từ chối, lỗi `ELEVATE_UNSAFE_ARG`. Đây là đổi lấy sự đơn giản và an toàn của v1.
5. Người dùng bấm "No" ở UAC → `ShellExecuteExW` trả lỗi `ERROR_CANCELLED` (1223) → lỗi `ELEVATE_CANCELLED`.
6. Lệnh chạy lâu không hỗ trợ chạy admin ở v1 (nút không hiện).

### 6.8 Nhận diện ngữ cảnh (`context.rs`)

**Thời điểm:** trong handler của hotkey, **trước** khi `show()` cửa sổ launcher (sau đó foreground là launcher, mất thông tin). Kết quả lưu trong state, frontend lấy bằng `get_context()` mỗi lần cửa sổ hiện. Cả bước chụp có ngân sách 200 ms (chạy ở thread riêng); quá hạn thì dùng phần đã có, phần còn lại để trống, **không** làm chậm việc hiện cửa sổ. Mở bằng click tray thì không có ngữ cảnh (`source: "none"`).

**Bước chung:**
1. `GetForegroundWindow` → `GetWindowThreadProcessId` → `OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION)` → `QueryFullProcessImageNameW` ra `exe`; `GetWindowTextW` ra `title`.
2. Bỏ qua nếu là chính launcher hoặc `explorer.exe` với class `Shell_TrayWnd`/`Progman`/`WorkerW` (taskbar, desktop) → `source: "none"`.
3. `app` lấy từ bảng tên đẹp (`Code.exe` → "VS Code", `Code - Insiders.exe` → "VS Code Insiders", `Cursor.exe` → "Cursor", `VSCodium.exe` → "VSCodium", `explorer.exe` → "Explorer"); app khác thì tên exe bỏ `.exe`.

**Explorer** (class `CabinetWClass`):
1. COM (`CoInitializeEx` apartment-threaded, cùng thread) → `IShellWindows` → duyệt từng cửa sổ, chọn cái có `HWND` trùng cửa sổ foreground.
2. `Document` → `IShellFolderViewDual` → `Folder().Self().Path()` ra `dir`; `SelectedItems()` ra `files` (đường dẫn đầy đủ).
3. Thư mục ảo (This PC, Quick access, Recycle Bin: đường dẫn bắt đầu bằng `::` hoặc không phải ổ đĩa) → `dir = None`.

**VS Code và bản fork (Code, Code - Insiders, Cursor, VSCodium):**
1. Tiêu đề mặc định có dạng `● file.ext - tên-thư-mục - Visual Studio Code`: tách theo `" - "`, bỏ `●`/`*` đầu dòng; đoạn cuối là tên app, đoạn kế cuối là `rootName` nếu có từ 3 đoạn trở lên (hoặc 2 đoạn khi chưa mở file nào và đoạn đầu không phải tên app).
2. Đọc `%APPDATA%\<Code|Code - Insiders|Cursor|VSCodium>\User\globalStorage\storage.json`, lấy `windowsState.lastActiveWindow` và `windowsState.openedWindows[]` có `folder` là URI `file:///`; đổi sang đường dẫn bằng crate `url`.
3. Chọn thư mục có tên cuối (basename) trùng `rootName`; nếu có đúng một → `dir`. Có nhiều thư mục trùng tên → lấy cái là `lastActiveWindow` nếu trùng, nếu không thì `dir = None` (chip ghi "không xác định được thư mục").
4. Workspace từ xa (`vscode-remote://`, WSL, SSH) và workspace file `.code-workspace` → `dir = None`. Người dùng đã đổi `window.title` nên không tách được → `dir = None`.
5. **Không** lấy file đang mở của VS Code (tiêu đề chỉ có tên file, không có đường dẫn); `files` rỗng.

**App khác:** chỉ có `app`, `exe`, `title`; `dir`/`files` rỗng (`source: "generic"`). Không đọc cwd của terminal ở v1.

**Dùng ngữ cảnh:**
- Chip ở `HintBar` giống nhau ở mọi app: `VS Code · foxkit` (chỉ **tên thư mục cuối**, rộng tối đa 280 px; tooltip là đường dẫn đầy đủ); chỉ có app thì chỉ hiện tên app; `source: "none"` thì ẩn chip.
- Ghost text tự điền (bảng ở 6.4 bước 6).
- Placeholder gõ tay, mở rộng ở frontend (`placeholders.ts`) trước khi gửi `args` cho Rust: `@dir` (thư mục hiện tại), `@file` (file đầu tiên đang chọn), `@files` (mở rộng thành nhiều arg, mỗi file một arg), `@clip` (nội dung clipboard, một arg). Ví dụ `size @dir`, `hash @files`, `port @clip`. `@@dir` là chữ `@dir` nguyên văn. Token `@` khác (vd email) giữ nguyên, không báo lỗi.
- `Context.dir` làm thư mục làm việc khi spawn (mục 6.6 bước 9).
- Placeholder không có giá trị → không chạy, lỗi `CONTEXT_MISSING`.

**Clipboard:** đọc bằng plugin `clipboard-manager` mỗi lần hiện cửa sổ, chỉ giữ trong bộ nhớ, cắt tối đa 500 ký tự, nhiều dòng thì chỉ lấy dòng đầu, **không** ghi log hay ghi ra đĩa. Chỉ dùng cho ghost kiểu `int` và `@clip`.

**Riêng tư và an toàn:** launcher chỉ đọc tên app, đường dẫn exe, tiêu đề cửa sổ và (khi cần) clipboard; không gửi đi đâu. `"context": false` trong config tắt toàn bộ mục này. Giá trị ngữ cảnh luôn đi qua `args` dạng mảng (không qua shell), nên đường dẫn chứa ký tự lạ vẫn an toàn; riêng chạy admin vẫn áp quy tắc `ELEVATE_UNSAFE_ARG` ở mục 6.7.

### 6.9 Lệnh ngoài: chọn thư mục, quét một lần, quét lại khi cần (`scan.rs`)

Ngoài lệnh của foxkit, launcher nạp thêm lệnh từ các CLI tool người dùng tự viết (exe, `.bat`, script...) nằm trong một hay nhiều thư mục (vd `D:\TranCN\foxkit`). Người dùng **chọn thư mục**, launcher **quét một lần** và lưu kết quả; **không cập nhật theo thời gian thực**, khi nào người dùng bấm quét lại thì mới đọc lại và cập nhật. **Chuẩn: một file `foxkit.json` dùng lại đúng schema manifest 1**, mỗi lệnh có thêm khối `exec` nói cách chạy.

**Chọn thư mục và quét:**
- Danh sách thư mục lưu ở config `folders` (mảng đường dẫn, hỗ trợ `%VAR%`). Chưa có thư mục nào thì không có lệnh ngoài.
- Quản lý ngay trong ô lệnh bằng lệnh nội bộ của launcher, bắt đầu bằng `:` để không lẫn với lệnh foxkit (có trong gợi ý/ghost/ví dụ như lệnh thường, nhãn "launcher"):

  | Lệnh | Việc |
  |---|---|
  | `:add` | Mở hộp thoại chọn thư mục (plugin `dialog`), thêm vào `folders` rồi quét thư mục đó |
  | `:add <đường dẫn>` | Như trên nhưng gõ đường dẫn; hỗ trợ `@dir` (đang đứng ở thư mục nào trong Explorer/VS Code thì thêm thư mục đó) |
  | `:scan` | **Quét lại toàn bộ** `folders` và cập nhật |
  | `:folders` | Liệt kê thư mục: đường dẫn, số file khai báo, số lệnh, lúc quét gần nhất, trạng thái (ok / không tìm thấy) |
  | `:forget <đường dẫn>` | Bỏ thư mục khỏi `folders` và xóa lệnh của nó (không xóa file nào) |

  Tray có thêm mục **Quét lại thư mục lệnh** (= `:scan`). "Tải lại" ở tray **không** quét, chỉ đọc lại config/manifest.
- Hộp thoại chọn thư mục làm cửa sổ launcher mất focus: lúc đó tắt hành vi "ẩn khi mất focus" (6.2) cho đến khi hộp thoại đóng.
- **Quét làm gì:** chỉ đọc đúng các vị trí chuẩn ở mục "Quy ước vị trí file" bên dưới (không duyệt sâu). Đọc, kiểm hợp lệ, rồi **ghi snapshot** vào `%APPDATA%\foxkit-launcher\scan-cache.json` (đường dẫn file, SHA-256 nội dung lúc quét, các lệnh đã parse, thời điểm quét). Mỗi file tối đa 256 KB, tối đa 200 lệnh ngoài tổng cộng.
- **Không quét lúc khởi động hay lúc mở cửa sổ.** Launcher luôn dùng snapshot trong `scan-cache.json`; sửa file khai báo hay thêm tool mới trên đĩa thì chưa có hiệu lực cho đến khi `:scan`.
- **Kết quả mỗi lần quét** hiện ngay trong ô lệnh: `Quét xong 3 thư mục: +2 lệnh mới, ~1 đổi, −1 không còn, 1 file bị bỏ (xem scan.log)`. Lệnh biến mất khỏi đĩa thì bị xóa khỏi snapshot (snapshot luôn khớp lần quét gần nhất).
- Thư mục không tồn tại lúc quét: giữ nguyên snapshot cũ của thư mục đó, đánh dấu "không tìm thấy" trong `:folders`.

**Quy ước vị trí file (chuẩn để launcher quét):**

Người dùng đưa vào một **thư mục gốc** (vd `D:\TranCN\foxkit`). Launcher chỉ tìm file tên **chính xác `foxkit.json`** ở hai vị trí, theo đúng thứ tự:

```
D:\TranCN\foxkit\                  ← thư mục gốc người dùng chọn
├── foxkit.json                    ① (tùy chọn) khai báo các lệnh đặt ngay ở gốc
├── tool-a\
│   ├── foxkit.json                ② khai báo lệnh của tool-a
│   └── bin\a.exe                 ← exec.program: "bin\\a.exe" (tính từ thư mục chứa foxkit.json)
├── tool-b\
│   ├── foxkit.json                ② khai báo lệnh của tool-b
│   └── run.bat
└── node_modules\, .git\, ...     ← bỏ qua
```

- ① `foxkit.json` ở gốc và ② `foxkit.json` ở **thư mục con trực tiếp** (một cấp). Không đọc cấp sâu hơn, không đọc file tên khác, không đọc `.foxkit.json` hay `*.foxkit.json`.
- Quy ước: **mỗi tool (mỗi repo con) một file `foxkit.json` ngay trong thư mục của nó**, nằm cạnh code. Thư mục gốc chính là repo chứa nhiều tool thì dùng ②; chỉ có một tool ngay ở gốc thì dùng ①. Có cả ① lẫn ② thì nạp cả hai.
- Chọn thẳng thư mục của một tool (vd `D:\TranCN\foxkit\tool-a`) cũng được: khi đó `foxkit.json` của nó là ①.
- Bỏ qua thư mục con: `node_modules`, `.git`, `target`, `vendor`, `dist`, `bin`, `obj`, và mọi thư mục bắt đầu bằng `.`.
- Mọi đường dẫn tương đối trong file (`exec.program`, `exec.prefix`) tính từ **thư mục chứa file `foxkit.json` đó**, nên đổi chỗ cả thư mục tool vẫn chạy.
- Thư mục chọn không có `foxkit.json` nào ở ①② → báo `Không thấy foxkit.json nào trong <thư mục> (chỉ tìm ở gốc và thư mục con trực tiếp)` và gợi ý xem mục định dạng file.

**Định dạng file:**

```json
{
  "$schema": "https://.../foxkit-command.schema.json",
  "schema": 1,
  "commands": [
    {
      "name": "resize",
      "title": "Đổi cỡ ảnh",
      "description": "Thu nhỏ ảnh theo chiều rộng",
      "category": "life",
      "args":  [ { "name": "input", "type": "file", "required": true, "label": "Ảnh" } ],
      "flags": [ { "name": "width", "type": "int", "min": 16, "max": 8000, "default": 1024, "label": "Chiều rộng" } ],
      "admin": false,
      "output": "text",
      "examples": [ { "cmd": "resize photo.jpg --width 800", "desc": "Thu về rộng 800 px" } ],
      "exec": { "program": "bin\\resize.exe", "prefix": [], "cwd": "file", "json": false }
    }
  ]
}
```

Các field giống [foundation mục 9.1](../core/foundation.md) (kể cả `confirm`, `dryRun`, `longRunning`, `longRunningFlag`, `examples`). Khối `exec` (bắt buộc với lệnh ngoài):

| Field | Ý nghĩa |
|---|---|
| `program` | Chương trình chạy: đường dẫn tuyệt đối, đường dẫn tương đối (tính từ thư mục chứa file `foxkit.json`) hoặc tên có trong `PATH`. Chỉ `.exe`, `.bat`, `.cmd`; script khác phải chạy qua trình thông dịch, ví dụ `"program": "powershell", "prefix": ["-NoProfile", "-File", "tools\\a.ps1"]` (đường dẫn trong `prefix` cũng tính từ thư mục file khai báo) |
| `prefix` | Mảng arg đặt trước mọi thứ, mặc định `[]` |
| `cwd` | `"file"` (thư mục chứa file khai báo, mặc định), `"context"` (thư mục hiện tại theo 6.8, không có thì như `file`) |
| `json` | `true`: chương trình tuân theo chuẩn foxkit (nhận `--json`, in envelope `{ok,data}` / `{ok:false,error}`, exit 0/1/2/3). `false` (mặc định): xử lý như văn bản |

**Cách chạy:** `program` + `prefix` + các `args` theo thứ tự khai báo + các cờ dạng `--tên giá trị` (cờ `bool` chỉ thêm `--tên` khi bật), không qua shell. Nếu `json: true` thì thêm `--json` cuối cùng. Placeholder `@dir`... mở rộng trước, như lệnh thường.

**Đọc kết quả khi `json: false`:** exit 0 → stdout nguyên văn là kết quả, hiện như `output: "text"` (khai báo `output` khác `text`/`stream` bị bỏ qua); exit ≠ 0 → stderr (nếu rỗng thì stdout) hiện là lỗi `EXTERNAL_FAILED` kèm exit code. Exit 3 không có ý nghĩa đặc biệt; muốn chạy admin thì khai `admin: true`. Lệnh `longRunning` đọc stdout từng dòng như `stream`.

**Hợp nhất vào danh sách lệnh:** `load_manifest` trả lệnh của foxkit rồi lệnh ngoài, mỗi lệnh thêm `source` (`"builtin"` / `"external"`) và `file`. Gợi ý, ghost text, ví dụ, validate dùng chung, không có đường riêng. Lệnh ngoài có nhãn nhỏ "ngoài" trong danh sách gợi ý, tooltip là đường dẫn file khai báo.

**Quy tắc `name`:** chữ thường, số, `-`, có thể có dấu cách cho lệnh con; không được trùng lệnh của foxkit hay lệnh ngoài đã nạp trước (thứ tự nạp theo thứ tự trong `folders`, trong thư mục theo đường dẫn file A→Z). Lệnh trùng bị bỏ qua và ghi vào log quét.

**Log:** `%APPDATA%\foxkit-launcher\scan.log` (ghi đè mỗi lần quét): file nào đọc được, lệnh nào nạp, lệnh/file nào bị bỏ và lý do.

**Tin cậy (lệnh ngoài chạy được chương trình tùy ý nên phải hỏi):**
- Quét xong lệnh nạp vào được, nhưng **lần chạy đầu của mỗi file khai báo** (và mỗi lần nội dung file đổi) hiện dòng xác nhận trong ô lệnh: `Cho phép chạy "<program đầy đủ đường dẫn>" từ <file>? Enter = Cho phép và nhớ · Esc = Không`.
- Đồng ý thì ghi `%APPDATA%\foxkit-launcher\trusted.json`: `{ "<đường dẫn file khai báo>": "<sha256 nội dung file>" }`. Quét lại thấy hash khác (file bị sửa) thì lần chạy kế hỏi lại. Chỉ hash file khai báo, không hash chương trình.
- Chạy nhanh bằng `Ctrl+Enter` không bỏ qua bước này.

**Lỗi:** thêm `EXTERNAL_NOT_FOUND` (không tìm thấy `program`), `EXTERNAL_FAILED`, `EXTERNAL_DENIED` (người dùng từ chối) ở mục 8. Một file khai báo hỏng (JSON sai, `schema != 1`, thiếu `exec`, `name` sai quy tắc, `exec.program` có đuôi không được phép) chỉ bị bỏ cả file hoặc cả lệnh đó, **không** làm hỏng việc nạp file khác hay lệnh foxkit.

**File JSON Schema:** repo có `launcher/schema/foxkit-command.schema.json` (JSON Schema draft-07 mô tả đúng định dạng trên) để editor (VS Code) tự gợi ý khi viết file `foxkit.json` nhờ `$schema`. Test ở 10.1 dùng chính file này để kiểm.

### 6.10 Giao diện chính (`main`)

Cửa sổ thường: 960×640, đổi cỡ được, có trên taskbar khi đang mở, nhớ vị trí/cỡ lần trước. Mở bằng: tray (double-click hoặc menu), mở app lần 2, nút Mở rộng ở ô nhanh, hoặc lệnh nội bộ `:open`. **Bấm X hoặc thu nhỏ → ẩn về tray** (chặn sự kiện đóng), không thoát app; chỉ `Thoát` ở tray mới thoát. 
Giao diện chính **không dùng ngữ cảnh app đang focus** (6.8) vì chính nó đang focus; thay vào đó tab Lệnh có ô file/thư mục kèm nút chọn bằng hộp thoại hệ thống (plugin `dialog`).

**Tab "Chat"** (mặc định khi mở app lần đầu; sau đó nhớ tab đang dùng)
- Cùng luồng với ô nhanh (6.2), hiển thị đầy đủ: mục cũ ở trên, **mới nhất ở dưới**, ô nhập ở đáy như chat thông thường, tự cuộn xuống khi có mục mới. Dùng chung `Composer.vue` nên có đủ gợi ý lệnh, ghost text, ví dụ mờ, Tab/Enter/Shift+Enter, `Ctrl+Shift+C` như ô nhanh (6.4, 6.6, 7); `ResultView` rộng hơn nên bảng không cắt cột.
- Thanh trên: dòng nhắc phím tắt **lấy từ `config.hotkey`** (không ghi cứng), nút **Xóa chat** (xóa cả luồng) và nút **Thu nhỏ** (ẩn giao diện chính và hiện ô nhanh ngay tại chỗ, giữ chữ đang gõ dở).
- Không có ngữ cảnh app đang focus ở đây (xem đầu mục); `@dir`, `@file`, `@files` báo `CONTEXT_MISSING` với hint "Dùng ô lệnh nhanh bằng phím tắt khi đang ở Explorer/VS Code, hoặc gõ đường dẫn trực tiếp". `@clip` vẫn dùng được (đọc clipboard lúc gửi lệnh).

**Tab "Lệnh"**
- Trái: ô tìm (cùng `fuzzy.ts` với ô lệnh nhanh) và danh sách lệnh nhóm theo `category`, kể cả lệnh ngoài (có nhãn "ngoài"). Phải: tiêu đề + mô tả lệnh và **form sinh từ manifest**:

  | `type` | Ô nhập |
  |---|---|
  | `string`, `text` | ô một dòng / nhiều dòng |
  | `int` | ô số, giới hạn `min`/`max` |
  | `bool` | công tắc |
  | `enum` | danh sách chọn từ `options` |
  | `file`, `dir` | ô chữ + nút "Chọn…" (hộp thoại); `allowDir` cho chọn cả thư mục; `multiple` thì thêm nhiều dòng |
  | `list` | ô chữ, tách theo `separator` |

  `required` đánh dấu `*`, `default` điền sẵn, `label` làm nhãn, `min–max` hiện làm gợi ý nhỏ. Lệnh có `examples` thì hiện nút "Ví dụ" điền thử giá trị từ ví dụ đầu tiên.
- Dưới form: dòng lệnh xem trước (`foxkit port 8000 --kill`, nút copy) tính từ `form.ts` — **cùng quy tắc ghép args và cùng validate (6.5)** với gõ tay, lỗi hiện ngay dưới ô sai.
- Nút **Chạy** đi qua đúng đường chạy ở 6.6: xác nhận (`confirm`), `dryRun` rồi nút Áp dụng, Dừng cho lệnh chạy lâu, nút chạy lại bằng quyền admin khi exit 3, Copy kết quả. Kết quả hiện ngay dưới form bằng `ResultView`.

**Tab "Thư mục lệnh"** (mục 6.9)
- Danh sách thư mục trong `folders`: đường dẫn, số file khai báo, số lệnh, lúc quét gần nhất, trạng thái (ok / không tìm thấy). Nút: **Thêm thư mục…** (hộp thoại chọn, quét ngay), **Quét lại** (từng thư mục hoặc tất cả = `:scan`), **Bỏ** (= `:forget`).
- Sau mỗi lần quét hiện báo cáo `+ mới / ~ đổi / − không còn` và danh sách file bị bỏ kèm lý do (nội dung `scan.log`). Không tự quét.
- Hiện quy ước vị trí file `foxkit.json` (6.9) dạng ghi chú ngắn kèm nút "Copy mẫu" (file mẫu trống hợp lệ) để tạo cho tool mới.

**Tab "Ứng dụng"** (mục 6.4a)
- Danh sách dòng cấu hình `appHints`: ô tên exe (`chrome.exe`, hoặc nút **Chọn .exe…** để lấy tên từ file), các lệnh dạng chip (bấm × để bỏ), ô "thêm lệnh…" có danh sách tên lệnh từ manifest, nút **Xóa** dòng. Nút **Thêm app** thêm dòng trống, **Lưu** ghi `config.json` và áp dụng ngay; dòng thiếu tên app hoặc không có lệnh nào bị bỏ khi lưu. Mặc định chưa có dòng nào.

**Tab "Cài đặt"** — sửa các field của config (5.1), bấm **Lưu** thì ghi `config.json` và áp dụng ngay, không cần khởi động lại:
- `hotkey`: ô bắt phím (bấm vào rồi nhấn tổ hợp), bắt buộc có ít nhất một phím bổ trợ; tổ hợp có phím Win thì cảnh báo "thường bị Windows giữ". Đăng ký lỗi → báo `HOTKEY_FAILED` tại chỗ và **giữ hotkey cũ**.
- `autostart`, `autoCopy`, `context`: công tắc.
- `foxkitPath`: ô chữ + nút chọn file; hiện kết quả `foxkit version` để kiểm đúng chưa.
- Không có theme, ngôn ngữ, hay cài đặt nào khác ở v1.

## 7. Hiển thị kết quả (`ResultView.vue`)

| `output` | Dữ liệu | Cách hiện |
|---|---|---|
| `kv` | `data` là object | Bảng 2 cột khóa/giá trị. Giá trị là mảng object → bảng con; object lồng → JSON thu gọn một dòng |
| `table` | `data.items` là mảng object | Bảng, cột lấy theo khóa của phần tử đầu (giữ thứ tự khóa). Mảng rỗng → "Không có kết quả" |
| `text` | `data` là string, hoặc object | String → `pre`; object → JSON định dạng |
| `stream` | Mỗi dòng NDJSON | Thêm dần từng dòng (dòng parse được thì hiện JSON thu gọn), tự cuộn xuống cuối, giữ tối đa 1000 dòng gần nhất |

- Mỗi kết quả có nút **Copy** (quy tắc bên dưới), nút **JSON** để xem envelope gốc và nút **Xóa** để bỏ riêng lệnh đó khỏi luồng chat. Copy ghi bằng plugin `clipboard-manager` (`writeText`), hiện "Đã copy" 1,5 giây.
- Phím tắt: `Ctrl+Shift+C` copy kết quả của lệnh vừa chạy (khi ô nhập không có chữ đang được bôi chọn); `Shift+Enter` chạy + copy + ẩn (mục 6.6 bước 8).
- Config `autoCopy: true` thì lệnh thành công nào cũng tự copy theo cùng quy tắc, trừ lệnh `stream` và lệnh có `dryRun` chưa `--apply`.
- Quy tắc "copy gì": `data` là string, hoặc `kv` chỉ có một khóa → copy giá trị đó (vd `uuid`, `pass`); `table` → TSV (dán được vào Excel/Sheets); còn lại → JSON `data` đã định dạng.
- Không hiểu `output` (phiên bản manifest mới hơn) → hiện JSON định dạng.

## 8. Lỗi của launcher

Hiện dưới ô nhập, cùng kiểu với lỗi foxkit (message + hint).

| code | Khi nào | message | hint |
|---|---|---|---|
| `FOXKIT_NOT_FOUND` | Không tìm thấy foxkit.exe | Không tìm thấy foxkit.exe | Đặt foxkit.exe cạnh launcher, thêm vào PATH, hoặc điền `foxkitPath` trong file cấu hình rồi chọn Tải lại |
| `MANIFEST_FAILED` | `foxkit manifest --json` lỗi / không phải JSON | Không đọc được danh sách lệnh từ foxkit | Chạy `foxkit manifest --json` trong terminal để xem lỗi |
| `MANIFEST_UNSUPPORTED` | `schema` khác 1 | foxkit dùng manifest phiên bản mới hơn launcher | Cập nhật launcher |
| `UNKNOWN_COMMAND` | Enter khi không khớp lệnh nào | Không có lệnh "<gõ>" | Chọn một lệnh trong danh sách gợi ý |
| `ARG_INVALID` | Validate mục 6.5 thất bại | (theo từng trường hợp, vd "Port phải là số từ 1 đến 65535") | Cú pháp từ `ArgHint` |
| `BAD_OUTPUT` | Output không phải envelope JSON | foxkit trả kết quả không đúng định dạng | Báo lỗi kèm nội dung bên dưới |
| `ELEVATE_CANCELLED` | Bấm No ở UAC | Đã hủy chạy quyền admin | — |
| `ELEVATE_UNSAFE_ARG` | Arg có ký tự đặc biệt khi chạy admin | Tham số có ký tự không chạy được ở chế độ admin | Mở terminal Administrator và chạy lệnh trực tiếp |
| `CONTEXT_MISSING` | Gõ `@dir`/`@file`/`@files`/`@clip` mà không có giá trị (hoặc `context: false`) | Không xác định được <thư mục hiện tại / file đang chọn / clipboard> | Bấm phím tắt khi đang ở cửa sổ Explorer hoặc VS Code đã mở thư mục, hoặc gõ đường dẫn trực tiếp |
| `EXTERNAL_NOT_FOUND` | Lệnh ngoài có `exec.program` không tồn tại | Không tìm thấy chương trình <program> của lệnh <name> | Kiểm tra đường dẫn `program` trong <file>, hoặc gõ `:scan` nếu tool đã bị đổi/xóa |
| `EXTERNAL_FAILED` | Lệnh ngoài `json:false` trả exit ≠ 0 | Lệnh <name> kết thúc với mã <code> | Xem nội dung lỗi bên dưới |
| `EXTERNAL_DENIED` | Người dùng từ chối ở bước tin cậy | Chưa cho phép chạy lệnh ngoài này | Chạy lại và chọn Cho phép |
| `HOTKEY_FAILED` | Đăng ký hotkey lỗi (thông báo tray) | Không đăng ký được phím tắt <hotkey> | Sửa `hotkey` trong file cấu hình rồi chọn Tải lại |

## 9. Manifest

Launcher **không** thêm entry vào `commands.json`. Mọi thứ cần (`args`, `flags`, `confirm`, `dryRun`, `longRunning`, `longRunningFlag`, `admin`, `output`) đã có trong schema 1, trừ **một field mới, tùy chọn**: `examples` (ví dụ mẫu cho dòng mờ ở 6.4 bước 7). Đã bổ sung vào [foundation mục 9.1](../core/foundation.md) và vẫn là `schema: 1` vì field này chỉ thêm, bỏ qua được. Launcher phải chạy bình thường khi lệnh không có `examples`. Nếu khi làm thấy thiếu field khác, sửa foundation.md trong cùng PR, không tự đoán ở launcher.

## 10. Test

### 10.1 Unit test frontend (`vitest`, dạng bảng)

| File | Case |
|---|---|
| `tokenize.test.ts` | `port 8000` → `["port","8000"]` · `find "C:\My Files" --ext pdf` → path giữ nguyên `\` và dấu cách · nháy thiếu → token cuối gom đến hết · nhiều dấu cách liền → bỏ token rỗng |
| `resolve.test.ts` | `hosts add a b` khớp `"hosts add"` · `hosts` (không có entry) → gợi ý `hosts add/rm/...` · `--` gợi ý cờ chưa dùng · `--out=x` tách đúng · `port 70000` → `ARG_INVALID` · `port --nope` → cờ không tồn tại · arg thừa khi không `multiple` |
| `ghost.test.ts` | `po` → ghost `rt` · `port ` + clipboard `8000` → ghost `8000`; clipboard `70000` hoặc `abc` → không ghost số · `size ` + dir `C:\My Files` → ghost `"C:\My Files"` · `hash ` + files chọn → file đầu · đủ arg → ghost cờ chưa dùng · `string` tự do → không ghost · không có `examples` → không bịa ví dụ |
| `placeholders.test.ts` | `size @dir` thay đúng · `hash @files` thành nhiều arg · `@@dir` → `@dir` · `a@b.com` giữ nguyên · không có giá trị → `CONTEXT_MISSING` · `context:false` → `CONTEXT_MISSING` |
| `scan.rs` (Rust, `cargo test`) | Dùng `testdata/scan/`: chỉ nhận `foxkit.json` ở gốc và thư mục con trực tiếp · `a/b/foxkit.json` (2 cấp) không nhận · `x.foxkit.json` và `foxkit.config.json` không nhận (chỉ đúng tên `foxkit.json`) · `node_modules`/`.git`/`bin` bị bỏ qua · chọn thư mục của một tool → file ở gốc là ① · không có file nào → báo đúng thông điệp · `:add` trùng thư mục → không thêm 2 lần · quét lại: file mới → `+`, sửa → `~`, xóa → `−`, snapshot khớp đĩa · không quét thì snapshot không đổi dù đĩa đổi · thư mục biến mất → giữ snapshot cũ, báo không tìm thấy · file hợp lệ nạp đủ lệnh · JSON hỏng bỏ cả file nhưng file khác vẫn nạp · `schema: 2` bỏ file · thiếu `exec` bỏ lệnh · `name` trùng lệnh foxkit hoặc lệnh ngoài trước đó → bỏ, ghi log · `exec.program` đuôi `.ps1` → bỏ · hash đổi → hỏi lại · mọi file mẫu hợp lệ phải qua `foxkit-command.schema.json` |
| `form.test.ts` | Giá trị form → args giống hệt gõ tay: `port` + `8000` + kill bật → `["port","8000","--kill"]` · bool tắt → không có cờ · giá trị trống không tạo cờ · `multiple` nhiều dòng → nhiều arg · đường dẫn có dấu cách giữ nguyên một arg · thiếu `required` / ngoài `min–max` → cùng thông báo với `resolve.ts` |
| `thread.test.ts` | Thêm mục vào luồng chung · quá 100 mục → bỏ cái cũ nhất · ô nhanh chỉ lấy 20 mục mới nhất, đảo thứ tự (mới ở trên) còn tab Chat giữ thứ tự thời gian · `:clear` xóa hết · mục "đang chạy" chuyển thành kết quả đúng mục · nhận sự kiện `thread-changed` thì cập nhật |
| `fuzzy.test.ts` | `prt` khớp `port` · `doi ten` khớp title "Đổi tên hàng loạt" · khớp đầu `name` xếp trên chứa chuỗi con |

Dùng một `commands.json` mẫu trong `src/lib/testdata/`, không gọi foxkit thật.

### 10.2 Unit test Rust
- `config.rs`: file thiếu → tạo mặc định; JSON hỏng → mặc định, không ghi đè.
- `context.rs`: hàm thuần tách tiêu đề VS Code (`● a.ts - foxkit - Visual Studio Code` → `foxkit`; chỉ `Visual Studio Code` → không có folder; tiêu đề có `-` trong tên thư mục), chọn thư mục từ `storage.json` mẫu (1 khớp / nhiều khớp có `lastActiveWindow` / nhiều khớp không có / remote URI), đổi URI `file:///c%3A/My%20Dir` → `C:\My Dir`. Phần gọi Win32/COM tách riêng, không test tự động.
- `elevate.rs`: hàm dựng `lpParameters` (tách riêng, không gọi WinAPI) → bọc nháy đúng; arg có `&`/`%` → `ELEVATE_UNSAFE_ARG`.

### 10.3 Chạy thử thủ công (Windows)

```powershell
# ở gốc repo foxkit
go build -o launcher\src-tauri\foxkit.exe .
cd launcher
npm install
npm run tauri dev
```

1. Bấm `Ctrl+Alt+Space` → ô lệnh hiện giữa màn hình, focus sẵn. Bấm lại → ẩn.
2. Gõ `uu` → gợi ý `uuid`, Tab, Enter → hiện UUID, bấm Copy, dán thử.
3. `python -m http.server 8000`, rồi gõ `port 8000` → bảng kv; `port 8000 --kill` → hỏi xác nhận, Enter → đã kill.
4. `port 70000` → lỗi đỏ ngay, không chạy.
5. Một lệnh `dryRun` (vd `rename`) → xem trước, `Ctrl+Enter` áp dụng.
6. `serve .` → dòng log chạy dần, bấm Dừng → process tắt (kiểm `foxkit port --list`).
7. Lệnh trả exit 3 → nút admin → UAC → có kết quả; bấm No ở UAC → `ELEVATE_CANCELLED`.
8. Bấm ra ngoài → ẩn. Mở app lần 2 từ Start → không có bản thứ hai, ô lệnh hiện lên.
9. Tắt/bật "Khởi động cùng Windows", đăng xuất/đăng nhập → app chạy ngầm, không bật cửa sổ.
10. Sửa `hotkey` thành tổ hợp đang bị app khác giữ → Tải lại → thấy thông báo `HOTKEY_FAILED`, click tray vẫn mở được.
11. Đổi tên foxkit.exe → Tải lại → `FOXKIT_NOT_FOUND`.
12. Mở Explorer vào một thư mục, chọn 2 file, bấm hotkey → chip hiện `Explorer · <thư mục>`; gõ `hash ` → ghost là file đầu; gõ `hash @files` → băm cả 2 file.
13. Mở VS Code với một thư mục dự án, bấm hotkey → chip `VS Code · <đường dẫn dự án>`; gõ `proj ` hoặc `size ` rồi Tab → tự điền đúng đường dẫn. Mở 2 cửa sổ VS Code khác thư mục → mỗi cái ra đúng thư mục của nó. Mở VS Code không có thư mục → chip chỉ có tên app.
14. Bấm hotkey ở app lạ (Notepad) → chip chỉ có tên app; gõ `size @dir` → `CONTEXT_MISSING`.
15. Copy `8000` trong Notepad, bấm hotkey, gõ `port ` → ghost `8000`, Tab điền. Gõ `port @clip` cũng chạy được.
16. `uuid` rồi `Ctrl+Shift+C`, dán thử → đúng UUID. `pass` rồi `Shift+Enter` → cửa sổ ẩn, dán được mật khẩu. `ps` → Copy ra TSV dán vào Excel được.
17. Ô trống → dòng mờ phía trên hiện một ví dụ; gõ `port` → hiện `Ví dụ: port 8000 --kill — ...`; gõ `po` → ghost `rt`, Tab điền.
18. Đặt `"context": false`, Tải lại → chip biến mất, mọi `@...` báo `CONTEXT_MISSING`.
19. Tạo thư mục thử, trong đó một file `foxkit.json` khai báo lệnh `hello` chạy `cmd /c echo xin chao`. Gõ `:add` → hộp thoại chọn thư mục (cửa sổ launcher không tự ẩn), chọn → báo `+1 lệnh mới`. Gõ `hel` thấy lệnh có nhãn "ngoài"; Enter → hỏi cho phép → Enter → hiện `xin chao`; chạy lại không hỏi nữa.
20. Sửa title trong file, thêm thư mục con `tool-b` với `foxkit.json` thứ hai, **chưa** `:scan` → mở lại ô lệnh vẫn thấy bản cũ (không realtime). Gõ `:scan` → báo `+1 mới, ~1 đổi`, danh sách cập nhật; sửa nội dung file → lần chạy kế hỏi cho phép lại. Xóa một file khai báo rồi `:scan` → lệnh biến mất. Khai báo lệnh tên `port` → bị bỏ vì trùng. Đổi tên thư mục → `:scan` rồi `:folders` báo "không tìm thấy", lệnh cũ giữ nguyên. `:forget` bỏ thư mục và lệnh của nó. Khởi động lại launcher → lệnh ngoài vẫn còn (từ snapshot, không quét).
21. Mở giao diện chính từ tray: tab Lệnh chọn `port` → form có ô Port và công tắc Kill; dòng lệnh xem trước đổi theo; nhập 70000 → lỗi ngay dưới ô; Chạy → kết quả hiện dưới form. Bấm X → cửa sổ ẩn, app vẫn chạy (tray còn); double-click tray → mở lại đúng tab cũ.
22. Mở app → vào tab Chat. Gõ `uuid`, `ts now`, `port 8000` → ba cặp lệnh/kết quả, mới nhất ở dưới. Bấm Thu nhỏ → ô nhanh hiện đúng ba mục đó (mới nhất ở trên). Chạy thêm `pass` ở ô nhanh, bấm Mở rộng → tab Chat có thêm mục đó và chữ gõ dở được giữ. Bấm X → app về tray; hotkey → ô nhanh vẫn còn luồng. `:clear` xóa ở cả hai nơi. Thoát hẳn rồi mở lại → luồng trống.
23. Tab Thư mục lệnh: Thêm thư mục, Quét lại, Bỏ hoạt động như `:add`/`:scan`/`:forget`; log hiển thị file bị bỏ.
24. Tab Cài đặt: đổi hotkey sang `Ctrl+Alt+K`, Lưu → hotkey mới dùng ngay, hotkey cũ hết tác dụng; nhập tổ hợp đang bị app khác giữ → báo lỗi và hotkey cũ vẫn dùng được; tắt `context` → chip biến mất không cần khởi động lại.

## 11. Tiêu chí hoàn thành

- [ ] `npm run test` (vitest) và `cargo test` pass; `cargo clippy` không cảnh báo; `npm run tauri build` ra bản cài.
- [ ] Hotkey mặc định `Ctrl+Alt+Space` hiện/ẩn ô lệnh; đổi được trong tab Cài đặt (áp dụng ngay) hoặc qua config + Tải lại.
- [ ] Giao diện chính 5 tab đúng mục 6.10 (mở vào tab Chat), form tab Lệnh sinh hoàn toàn từ manifest; bấm X chỉ ẩn về tray.
- [ ] Ô nhanh và tab Chat là **cùng một luồng chat** (6.2): đồng bộ hai chiều, nút Mở rộng/Thu nhỏ giữ chữ đang gõ, chỉ nằm trong bộ nhớ, `:clear`.
- [ ] Tray đủ các mục ở 6.3; autostart chạy ngầm không hiện cửa sổ; single-instance đúng.
- [ ] Gợi ý lệnh, gợi ý cờ, `ArgHint` và validate đúng mục 6.4–6.5, sinh hoàn toàn từ manifest (thêm lệnh mới vào foxkit không phải sửa launcher).
- [ ] Ghost text đúng bảng 6.4 bước 6; dòng ví dụ mờ phía trên ô nhập đúng bước 7; không có gợi ý theo lịch sử/lệnh hay dùng.
- [ ] Nhận diện đúng app đang focus, thư mục Explorer + file đang chọn, thư mục VS Code (mục 6.8); placeholder `@dir @file @files @clip` hoạt động; chụp ngữ cảnh không làm chậm việc hiện cửa sổ quá 200 ms.
- [ ] Copy kết quả: nút Copy, `Ctrl+Shift+C`, `Shift+Enter`, `autoCopy` đúng quy tắc mục 7; `context: false` tắt hẳn việc đọc ngữ cảnh/clipboard.
- [ ] Bốn kiểu `output` hiện đúng mục 7; `confirm`, `dryRun`, Dừng, chạy admin chạy đúng.
- [ ] Không chớp cửa sổ console khi chạy lệnh.
- [ ] Chọn thư mục (`:add`, `:folders`, `:forget`), quét và quét lại thủ công (`:scan`, tray) với báo cáo `+/~/−`, snapshot `scan-cache.json` dùng lại khi khởi động, **không** quét hay cập nhật tự động, xác nhận tin cậy, `scan.log` đúng mục 6.9; file hỏng không ảnh hưởng file khác; `foxkit-command.schema.json` có và khớp.
- [ ] Mọi lỗi mục 8 có đúng code và message.
- [ ] Không có dependency ngoài bảng mục 2.

## 12. Ngoài phạm vi (không làm ở v1)

Chat với AI hoặc hiểu ngôn ngữ tự nhiên; lịch sử lệnh / gõ ↑ để gọi lại / gợi ý theo lệnh hay dùng; tự hoàn thành đường dẫn file bằng cách duyệt đĩa và hộp thoại chọn file; nhận diện thư mục hiện tại của terminal (cmd, PowerShell, Windows Terminal), JetBrains, Notepad++ và app khác (chỉ có Explorer và VS Code/Cursor/VSCodium); lấy file đang mở trong VS Code; lấy chữ đang bôi đen trong app; cài đặt ngoài danh sách ở tab Cài đặt (6.10); lưu luồng hội thoại ra đĩa; theme / trong suốt / hiệu ứng mờ; ghim lệnh yêu thích; chạy nhiều lệnh cùng lúc; cập nhật lệnh ngoài theo thời gian thực hoặc tự quét khi khởi động/mở cửa sổ; lệnh `:` khác ngoài add/scan/folders/forget; cài lệnh ngoài từ URL/kho lệnh; lệnh ngoài nhận ngữ cảnh qua biến môi trường; lệnh ngoài `.ps1` chạy trực tiếp không qua `prefix`; chạy admin cho lệnh chạy lâu; tự cập nhật app; nhãn đa ngôn ngữ; button/lệnh động theo `proj --json` (ghi ở roadmap, để sau).

## 13. Khái niệm học được

- **Rust/Tauri command**: hàm `#[tauri::command]` ≈ route API trong Laravel, frontend gọi bằng `invoke("run", { args })` ≈ `axios.post`. `Result<T, String>` ≈ trả JSON hoặc ném exception.
- **`Channel` của Tauri** ≈ Server-Sent Events: Rust đẩy từng dòng, Vue nhận qua callback.
- **`std::process::Command`** ≈ `exec.Command` của Go / `Symfony\Process`; `creation_flags` là chỗ riêng Windows.
- **Vue 3 Composition API** so với Vue 2: `ref()`/`computed()` thay `data`/`computed` trong object, `<script setup>` thay `export default {}`; logic tách ra file `.ts` thuần nên test được không cần mount component.
- **Hash để nhận biết file đổi** (`sha2`): ≈ `md5_file()`/`hash_file()` của PHP dùng làm cache key; ở đây dùng làm "con dấu đã duyệt".
- **COM / `IShellWindows`**: cách Explorer cho app khác hỏi "cửa sổ nào đang mở thư mục nào"; ≈ gọi một API nội bộ của hệ điều hành thay vì đọc màn hình. Crate `windows` bọc các interface này thành kiểu Rust.
- **Capabilities** của Tauri v2: quyền gọi plugin phải khai báo trong `capabilities/default.json` (≈ middleware/policy) — quên khai thì `invoke` báo lỗi permission.
