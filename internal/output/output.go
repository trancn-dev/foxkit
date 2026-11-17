package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Printer in kết quả/lỗi theo chế độ text hoặc JSON.
type Printer struct {
	JSON   bool
	Stdout io.Writer
	Stderr io.Writer
}

type envelope struct {
	OK    bool   `json:"ok"`
	Data  any    `json:"data,omitempty"`
	Error *Error `json:"error,omitempty"`
}

func encode(w io.Writer, v any, indent bool) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if indent {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(v)
}

// Success: JSON → {"ok":true,"data":...}; text → textFn.
func (p Printer) Success(data any, textFn func(w io.Writer) error) error {
	if p.JSON {
		// data có thể là nil/slice: bọc bằng struct riêng để luôn có khóa "data".
		return encode(p.Stdout, struct {
			OK   bool `json:"ok"`
			Data any  `json:"data"`
		}{true, data}, true)
	}
	if textFn == nil {
		return nil
	}
	return textFn(p.Stdout)
}

// Error in lỗi ra stderr và trả exit code.
func (p Printer) Error(err error) int {
	var oe *Error
	if !errors.As(err, &oe) {
		oe = Internal(err)
	}
	if p.JSON {
		_ = encode(p.Stderr, envelope{OK: false, Error: oe}, true) // lỗi ghi stderr không có chỗ báo tiếp
	} else {
		fmt.Fprintf(p.Stderr, "Lỗi: %s\n", oe.Message)
		if oe.Hint != "" {
			fmt.Fprintf(p.Stderr, "→ %s\n", oe.Hint)
		}
	}
	return oe.Exit
}

// Event in một dòng NDJSON (lệnh chạy lâu).
func (p Printer) Event(v any) error { return encode(p.Stdout, v, false) }

// DryRunNote là dòng cuối text mode khi dry-run.
const DryRunNote = "Đây là bản xem trước. Thêm --apply để thực hiện."
