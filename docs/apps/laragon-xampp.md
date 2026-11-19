# Laragon / XAMPP / WAMP (môi trường PHP-Laravel cục bộ)

> exe: `laragon.exe`, `xampp-control.exe`, `wampmanager.exe` · Ngữ cảnh: tên app + tiêu đề. Đây là nhóm phù hợp nhất với người dùng Laravel của dự án, và `foxkit` đã có sẵn nhiều lệnh cho nhóm này.

## 1. Lệnh có sẵn, dùng ngay (appHints)
| Lệnh | Dùng khi |
|---|---|
| `svc` | Xem MySQL/Apache/Redis chạy chưa; `svc start|stop|restart` (cần admin) |
| `port` | 80/443/3306 bị chiếm (Skype, IIS, MySQL trùng) → `--kill` an toàn |
| `hosts` | Thêm/tắt/bật domain ảo `myapp.test` (`list|add|rm|on|off`, dry-run + backup) |
| `lara log` | Đọc `storage/logs/laravel.log`, lọc theo level, rút gọn trace |
| `lara key`, `lara clear` | Sinh `APP_KEY`, `optimize:clear` |
| `env diff`, `env get` | So `.env` với `.env.example` |
| `proj` | Nhận diện Laravel/Vue, phiên bản PHP/Node, scripts |
| `which`, `path` | `php`/`composer` chạy bản nào, PATH có đúng thư mục Laragon |
| `dns` | `dns --flush` sau khi sửa hosts |
| `prune` | Dọn `vendor`, `node_modules` của dự án cũ |
| `serve` | Chạy nhanh thư mục tĩnh/SPA |

Mẫu: `{ "app": "laragon.exe", "commands": ["svc", "port", "hosts", "lara log", "env diff", "which"] }`.

## 2. Tool nhỏ mới đề xuất
| Lệnh đề xuất | Việc làm | Khó | Giá trị | Dependency |
|---|---|---|---|---|
| `vhost` | Cho một thư mục dự án + tên miền `.test`: sinh cấu hình vhost Apache/Nginx và dòng `hosts`, dry-run, backup, cần admin để ghi | 3 | ★★★ | stdlib; đụng file hệ thống → bắt buộc plan/apply |
| `phpver` | Liệt kê mọi bản PHP/Composer/Node trong PATH và thư mục Laragon, bản nào đang dùng, cảnh báo lệch với `composer.json` (`require.php`) | 2 | ★★★ | stdlib (nền `which` + `proj`) |
| `lara routes` | Liệt kê route từ `php artisan route:list --json` dạng bảng gọn | 2 | ★★ | stdlib (`os/exec` gọi `php`); phụ thuộc bản Laravel |
| `lara perm` | Kiểm quyền ghi `storage/`, `bootstrap/cache` (lỗi phổ biến sau khi clone) | 1 | ★★ | stdlib |
| `dbdump` | Bọc `mysqldump` của Laragon, đặt tên theo ngày (xem `db-client.md`) | 2 | ★★ | stdlib; mặc định dry-run |
| `lara tail` | Đã có `lara log --follow`; có thể thêm nhóm `queue`/`schedule` log | 2 | ★ | stdlib |

## 3. Việc launcher chưa làm được
- Không biết **thư mục dự án** từ cửa sổ Laragon; dùng `@dir` ở Explorer/VS Code. Laragon lưu đường dẫn gốc www trong `laragon.ini`; đọc được (khó 2) nhưng chỉ cho biết thư mục gốc, không phải dự án đang làm việc.
- Lệnh cần admin (`svc`, `hosts --apply`) vẫn đi qua luồng UAC của launcher; nút "Chạy lại với quyền admin" đã có.
- **Chưa kiểm** thật: `svc stop|start` MySQL, `hosts add --apply` — nằm trong danh sách để người dùng tự chạy (xem `PROGRESS.md`).
