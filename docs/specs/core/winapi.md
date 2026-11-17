# Spec: Nền tảng Win32 API (đầu GĐ6)

> Nhóm: **core** · Giai đoạn: GĐ6 · Cần trước: GĐ0–GĐ5
> Làm file này trước mọi lệnh GĐ6.

## 1. Dependency mới
```powershell
go get golang.org/x/sys@latest
```
**Lý do**: thư viện chuẩn không có registry, thông tin ổ đĩa, Service Control Manager, Restart Manager. `golang.org/x/sys` do chính đội Go duy trì, là cách chuẩn để gọi Windows API từ Go; một dependency phục vụ cả giai đoạn.

## 2. Quy ước code
- Code gọi API đặt trong file `*_windows.go` (build tag ngầm theo tên file). Với mỗi package có file windows, thêm file `*_other.go` (`//go:build !windows`) trả lỗi `ErrUnsupported` để `go vet`/`go test` vẫn chạy được trên máy khác (CI Linux).
- Hàm API **có sẵn** trong `golang.org/x/sys/windows` thì dùng trực tiếp. **Không có** thì gọi qua DLL:
  ```go
  var (
      modkernel32 = windows.NewLazySystemDLL("kernel32.dll")
      procGlobalMemoryStatusEx = modkernel32.NewProc("GlobalMemoryStatusEx")
  )
  r1, _, err := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&ms)))
  if r1 == 0 { return err }   // với API trả BOOL: 0 = thất bại, err là GetLastError
  ```
  `NewLazySystemDLL` chỉ nạp DLL từ thư mục System32 (an toàn trước DLL hijacking).
- Struct truyền cho API phải **khớp layout C**: dùng `uint32` cho DWORD, `uint64` cho DWORDLONG, `[N]uint16` cho WCHAR[N], `windows.Filetime` cho FILETIME; kiểm tra `unsafe.Sizeof` trong test so với kích thước tài liệu Microsoft ghi.
- Chuỗi: `windows.UTF16PtrFromString` (Go → API), `windows.UTF16ToString(buf[:])` (API → Go).
- Lỗi quyền: `errors.Is(err, windows.ERROR_ACCESS_DENIED)` → `output.NeedAdmin(...)`.

## 3. Helper chung `internal/winapi`
```go
package winapi
func IsAdmin() bool                         // windows.GetCurrentProcessToken().IsElevated()
func ProcessPath(pid int) (string, error)   // OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION) + QueryFullProcessImageName
func Processes() ([]Proc, error)            // CreateToolhelp32Snapshot + Process32First/Next
type Proc struct { PID, PPID int; Name string }
```

## 4. Nâng cấp lệnh cũ (tùy chọn, sau khi có helper)
| Lệnh | Nâng cấp |
|---|---|
| `port` | điền `Path` bằng `winapi.ProcessPath` |
| `ps` | thêm cột đường dẫn; tên process Unicode đúng (bỏ phụ thuộc codepage của tasklist) |
| `kill --tree` | tự duyệt cây bằng `PPID` thay vì gọi `taskkill /T` |

Mỗi nâng cấp phải giữ nguyên định dạng JSON cũ (chỉ thêm field), để launcher không vỡ.

## 5. Go học được
`unsafe.Pointer`, `uintptr`, layout bộ nhớ struct, build tag theo hệ điều hành, gọi DLL (≈ FFI của PHP 7.4+).
