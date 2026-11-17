package cmd

import (
	"bytes"
	"io"

	"foxkit/internal/output"

	"github.com/spf13/cobra"
)

// readLimited đọc hết r, tối đa limit byte; vượt thì trả lỗi INPUT_TOO_LARGE.
func readLimited(r io.Reader, limit int64, limitText string) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, output.Internal(err)
	}
	if int64(len(b)) > limit {
		return nil, output.InputTooLarge(limitText)
	}
	return b, nil
}

// readStdin đọc stdin của lệnh (c.InOrStdin để test thay được).
// Bỏ BOM UTF-8 đầu vào: PowerShell 5.1 thêm BOM khi pipe chuỗi vào chương trình ngoài.
func readStdin(c *cobra.Command, limit int64, limitText string) ([]byte, error) {
	b, err := readLimited(c.InOrStdin(), limit, limitText)
	return bytes.TrimPrefix(b, []byte("\xef\xbb\xbf")), err
}
