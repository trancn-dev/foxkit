package output

import "fmt"

// Error là lỗi có mã, thông điệp tiếng Việt và gợi ý bước tiếp theo.
// Position: vị trí lỗi cú pháp (lệnh json).
type Position struct {
	Line int `json:"line"`
	Col  int `json:"col"`
}

type Error struct {
	Code     string    `json:"code"`
	Message  string    `json:"message"`
	Hint     string    `json:"hint,omitempty"`
	Position *Position `json:"position,omitempty"`
	Exit     int       `json:"-"`
}

func (e *Error) Error() string { return e.Message }

// Usage: sai tham số (exit 2).
func Usage(code, msg, hint string) *Error {
	return &Error{Code: code, Message: msg, Hint: hint, Exit: 2}
}

// Fail: lỗi khi chạy (exit 1).
func Fail(code, msg, hint string) *Error {
	return &Error{Code: code, Message: msg, Hint: hint, Exit: 1}
}

// NeedAdmin: cần quyền admin (exit 3).
func NeedAdmin(msg string) *Error {
	return &Error{Code: "NEED_ADMIN", Message: msg, Hint: "Chạy lại terminal bằng quyền Administrator", Exit: 3}
}

// Các lỗi dùng chung (foundation mục 7.3).

func DirNotFound(p string) *Error {
	return Usage("DIR_NOT_FOUND", "Không tìm thấy thư mục "+p, "Kiểm tra lại đường dẫn")
}
func FileNotFound(p string) *Error {
	return Usage("FILE_NOT_FOUND", "Không tìm thấy file "+p, "Kiểm tra lại đường dẫn")
}
func PathNotFound(p string) *Error {
	return Usage("PATH_NOT_FOUND", "Không tìm thấy "+p, "Kiểm tra lại đường dẫn")
}
func SizeInvalid(s string) *Error {
	return Usage("SIZE_INVALID", "Kích thước không hợp lệ: "+s, "Ví dụ: 500KB, 10MB, 2GB")
}
func AgeInvalid(s string) *Error {
	return Usage("AGE_INVALID", "Thời lượng không hợp lệ: "+s, "Ví dụ: 90m, 12h, 7d, 2w, 1y")
}
func PortInvalid() *Error {
	return Usage("PORT_INVALID", "Port phải là số từ 1 đến 65535", "")
}
func InputTooLarge(limit string) *Error {
	return Fail("INPUT_TOO_LARGE", "Dữ liệu vào quá lớn (giới hạn "+limit+")", "")
}
func GitNotFound() *Error {
	return Fail("GIT_NOT_FOUND", "Không tìm thấy git", "Cài Git for Windows hoặc kiểm tra bằng foxkit which git")
}
func Canceled() *Error { return Fail("CANCELED", "Đã hủy (Ctrl+C)", "") }
func Internal(err error) *Error {
	return Fail("INTERNAL", err.Error(), "Báo lỗi kèm lệnh đã chạy")
}
func Argsf(format string, a ...any) *Error {
	return Usage("ARGS_INVALID", fmt.Sprintf(format, a...), "")
}
