# Docker Desktop

> exe: `Docker Desktop.exe` (có dấu cách), `com.docker.backend.exe` · Ngữ cảnh: tên app + tiêu đề. Docker CLI (`docker.exe`) là chương trình ngoài.

## 1. Lệnh có sẵn, dùng ngay (appHints)
| Lệnh | Dùng khi |
|---|---|
| `port` | Cổng 80/3306/6379 bị chiếm → container không lên; `--kill` process chiếm |
| `tcp` | Container đã mở cổng chưa (`tcp localhost:3306`) |
| `http` | Healthcheck URL của service trong container |
| `ps` | Process nào (`com.docker.backend`, `vmmem`) ăn RAM; cảnh báo máy chậm |
| `env diff`, `env get` | Biến môi trường trong `.env` khớp `.env.example` trước khi `docker compose up` |
| `size`, `disk` | Ổ đĩa đầy do image/volume (`%LOCALAPPDATA%\Docker`) |
| `svc` | Dịch vụ liên quan (`com.docker.service`) |
| `serve` | Dựng web tĩnh tạm để thử reverse proxy |

Mẫu: `{ "app": "Docker Desktop.exe", "commands": ["port", "tcp", "http", "env diff", "disk"] }`.

## 2. Tool nhỏ mới đề xuất
| Lệnh đề xuất | Việc làm | Khó | Giá trị | Dependency |
|---|---|---|---|---|
| `ports` | Đọc `docker-compose.yml`, liệt kê cổng host dự kiến và cho biết cổng nào đang bị chiếm | 2 | ★★★ | stdlib; YAML không có trong stdlib → chỉ tách dòng `ports:` đơn giản, ghi rõ giới hạn (hoặc hỏi dùng `yaml.v3`) |
| `composeenv` | So biến `${VAR}` dùng trong compose với `.env` (thiếu/thừa) — mở rộng `env diff` | 2 | ★★★ | stdlib |
| `dockerignore` | Sinh `.dockerignore` từ loại dự án (dùng `proj`) | 1 | ★★ | stdlib |
| `dk` | Bọc `docker` CLI có sẵn: `dk ps` bảng gọn (tên, image, cổng, trạng thái), `dk logs <tên> --last N`; chỉ đọc, không `rm`/`prune` | 3 | ★★ | stdlib (`os/exec`); không parse chữ dịch → dùng `--format json` của docker |
| `vmmem` | Báo RAM WSL2/Docker đang giữ và cách giới hạn (`.wslconfig`) — chỉ gợi ý, không tự sửa | 2 | ★ | stdlib |

## 3. Việc launcher chưa làm được
- Không biết container/compose project đang mở; dùng `@dir` từ Explorer/VS Code thay thế.
- Không làm lệnh phá hoại (`docker system prune`, xóa volume) trong launcher; nếu thêm, phải dry-run + xác nhận như `prune`.
- Tên exe có dấu cách: tab Ứng dụng so tên file nguyên văn không phân biệt hoa thường (`docker desktop.exe`) — đã đúng; kiểm tay.
