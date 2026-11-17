package clip

import (
	"strings"
	"time"

	"github.com/atotto/clipboard"
)

// Clipboard: giao diện để thay implementation khi test (≈ bind interface trong Service Container).
type Clipboard interface {
	Read() (string, error)
	Write(string) error
}

type atottoClipboard struct{}

func (atottoClipboard) Read() (string, error) { return clipboard.ReadAll() }
func (atottoClipboard) Write(s string) error  { return clipboard.WriteAll(s) }

// Default là clipboard thật (thư viện atotto/clipboard: Unicode đúng, thuần Go, không cgo).
var Default Clipboard = atottoClipboard{}

// MaxBytes: giới hạn ghi.
const MaxBytes = 10 << 20

// NormalizeCRLF đổi mọi kiểu xuống dòng sang CRLF để dán vào Notepad đúng.
func NormalizeCRLF(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.ReplaceAll(s, "\n", "\r\n")
}

// TrimOneNewline bỏ đúng một xuống dòng cuối (lệnh pipe thường tự thêm).
func TrimOneNewline(s string) string {
	if t, ok := strings.CutSuffix(s, "\r\n"); ok {
		return t
	}
	return strings.TrimSuffix(s, "\n")
}

// Retry chạy f tối đa 3 lần cách 100ms (clipboard có thể đang bị app khác giữ); trả lỗi của lần cuối.
func Retry(f func() error) error {
	var err error
	for i := 0; i < 3; i++ {
		if err = f(); err == nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return err
}
