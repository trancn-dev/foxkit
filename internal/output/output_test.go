package output

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestSuccess(t *testing.T) {
	tests := []struct {
		name string
		json bool
		data any
		want string
	}{
		{"json no escape", true, map[string]string{"u": "a&b<c"}, `"u": "a&b<c"`},
		{"json empty slice", true, []int{}, `"data": []`},
		{"text", false, nil, "xin chào"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			p := Printer{JSON: tc.json, Stdout: &out}
			err := p.Success(tc.data, func(w io.Writer) error { _, e := io.WriteString(w, "xin chào"); return e })
			if err != nil || !strings.Contains(out.String(), tc.want) {
				t.Fatalf("got %q err %v, want contains %q", out.String(), err, tc.want)
			}
		})
	}
}

func TestError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		json     bool
		wantExit int
		want     string
	}{
		{"usage text", Usage("X", "sai", "làm này"), false, 2, "Lỗi: sai\n→ làm này\n"},
		{"admin", NeedAdmin("việc"), false, 3, "Lỗi: việc"},
		{"plain error", errors.New("hỏng"), true, 1, `"code": "INTERNAL"`},
		{"json", Fail("A", "m", ""), true, 1, `"ok": false`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var errb bytes.Buffer
			got := Printer{JSON: tc.json, Stderr: &errb}.Error(tc.err)
			if got != tc.wantExit || !strings.Contains(errb.String(), tc.want) {
				t.Fatalf("exit %d out %q", got, errb.String())
			}
		})
	}
}

func TestEvent(t *testing.T) {
	var out bytes.Buffer
	if err := (Printer{Stdout: &out}).Event(map[string]int{"a": 1}); err != nil || out.String() != "{\"a\":1}\n" {
		t.Fatalf("got %q %v", out.String(), err)
	}
}
