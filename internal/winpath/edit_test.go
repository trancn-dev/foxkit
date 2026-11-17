package winpath

import (
	"strings"
	"testing"
)

func TestAddKeepsRawAndDedupes(t *testing.T) {
	t.Setenv("MYTOOLS", `C:\tools`)
	tests := []struct {
		name, value, dir string
		first            bool
		want             string
		changed          bool
	}{
		{"thêm cuối", `C:\a;C:\b`, `C:\c`, false, `C:\a;C:\b;C:\c`, true},
		{"thêm đầu", `C:\a;C:\b`, `C:\c`, true, `C:\c;C:\a;C:\b`, true},
		{"trùng khác hoa thường và \\ cuối", `C:\Tools\Bin;C:\b`, `c:\tools\bin\`, false, `C:\Tools\Bin;C:\b`, false},
		{"trùng qua %VAR%", `%MYTOOLS%;C:\b`, `C:\tools`, false, `%MYTOOLS%;C:\b`, false},
		{"giữ nguyên %SystemRoot%", `%SystemRoot%\system32;C:\a`, `C:\new`, false, `%SystemRoot%\system32;C:\a;C:\new`, true},
		{"PATH rỗng", ``, `C:\x`, false, `C:\x`, true},
		{"bỏ ; thừa", `;C:\a;;`, `C:\b`, false, `C:\a;C:\b`, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, changed := Add(tc.value, tc.dir, tc.first)
			if got != tc.want || changed != tc.changed {
				t.Fatalf("got %q,%v want %q,%v", got, changed, tc.want, tc.changed)
			}
		})
	}
}

func TestRemoveMatchesExpandedForm(t *testing.T) {
	t.Setenv("USERPROFILE", `C:\Users\me`)
	got, n := Remove(`%USERPROFILE%\bin;C:\a;c:\users\ME\bin\`, `C:\Users\me\bin`)
	if got != `C:\a` || n != 2 {
		t.Fatalf("%q %d", got, n)
	}
	if got, n := Remove(`C:\a`, `C:\zzz`); got != `C:\a` || n != 0 {
		t.Fatalf("%q %d", got, n)
	}
}

func TestClean(t *testing.T) {
	exists := func(p string) bool { return !strings.Contains(p, "gone") }
	got, removed := Clean(`C:\a;;C:\gone;C:\A\;C:\b`, exists)
	if got != `C:\a;C:\b` || len(removed) != 2 { // mục rỗng bị bỏ im lặng, không tính vào danh sách đã xóa
		t.Fatalf("%q %v", got, removed)
	}
}

func TestPortableAndNeedsExpand(t *testing.T) {
	t.Setenv("USERPROFILE", `C:\Users\me`)
	if got := Portable(`C:\Users\me\bin`); got != `%USERPROFILE%\bin` {
		t.Errorf("%q", got)
	}
	if got := Portable(`C:\Users\meow\bin`); got != `C:\Users\meow\bin` {
		t.Errorf("tiền tố giống nhưng khác thư mục: %q", got)
	}
	if !NeedsExpand(Raw{Value: `%A%\x`}) || !NeedsExpand(Raw{Expand: true}) || NeedsExpand(Raw{Value: `C:\x`}) {
		t.Error("NeedsExpand sai")
	}
}
