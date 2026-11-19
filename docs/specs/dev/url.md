# Spec: `foxkit url`

> Nhóm: **dev** · Ngoài roadmap gốc (đề xuất từ `docs/apps/`) · Trạng thái: **chưa làm, chờ duyệt** · Cần trước: [core/foundation.md](../core/foundation.md)

## 1. Mục đích
Tách, làm sạch và sửa URL: xem từng tham số query, bỏ tham số theo dõi (`utm_*`, `fbclid`, `gclid`), giải mã `%xx`.

## 2. Cú pháp
```
foxkit url <url> [--clean] [--decode] [--set k=v]... [--del k]... [--copy] [--json]
```
Không có `<url>` thì đọc clipboard (dòng đầu). Không cờ nào: in bảng các phần (scheme, host, port, path, query, fragment) và từng tham số.

| Cờ | Việc |
|---|---|
| `--clean` | bỏ `utm_*`, `fbclid`, `gclid`, `mc_eid`, `ref` |
| `--decode` | giải mã `%xx` trong path/query để đọc |
| `--set k=v` | đặt/ghi đè tham số (lặp được) |
| `--del k` | xóa tham số (lặp được) |

Có `--clean/--set/--del` thì kết quả là **URL mới** (một dòng, dễ pipe/copy); `--set` thiếu `=` → `URL_SET_INVALID` (exit 2). URL không parse được hoặc thiếu scheme+host → `URL_INVALID` (exit 2, hint "ví dụ: https://example.com/a?b=1").

## 3. Cách hoạt động
- `net/url` (`url.Parse`, `Query()`); giữ thứ tự tham số gốc khi ghép lại (không dùng `Encode()` vì nó sắp xếp theo khóa): tự ghép từ `RawQuery` tách bằng `&`.
- Không thêm `https://` ngầm; URL không có scheme báo lỗi như trên.
- JSON: `{ "scheme","host","port","path","fragment","params":[{"key","value"}] }`; `params` rỗng là `[]`.

## 4. Test (dạng bảng)
URL đủ phần, tham số lặp khóa, `%` tiếng Việt, `--clean` giữ nguyên thứ tự, `--set` ghi đè, URL thiếu scheme, chuỗi rác.

Chạy thử: `.\foxkit url "https://a.com/x?utm_source=fb&id=5" --clean`.

## 5. Ngoài phạm vi
Mở URL trong trình duyệt, kiểm tra sống/chết (đã có `http`), rút gọn URL (cần mạng ngoài).
