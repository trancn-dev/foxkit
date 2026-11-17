package startup

import (
	"runtime"
	"testing"
)

func TestExePath(t *testing.T) {
	tests := []struct{ in, want string }{
		{`"C:\a b\x.exe" -y`, `C:\a b\x.exe`},
		{`C:\x.exe /s`, `C:\x.exe`},
		{`rundll32.exe a.dll,Run`, `rundll32.exe`},
		{`C:\Program Files\X\y.EXE /background`, `C:\Program Files\X\y.EXE`},
		{`"C:\only quote`, `C:\only quote`},
		{`script.bat arg`, `script.bat`},
		{``, ``},
	}
	for _, tc := range tests {
		if got := ExePath(tc.in); got != tc.want {
			t.Errorf("ExePath(%q) = %q, cần %q", tc.in, got, tc.want)
		}
	}
}

func TestApproved(t *testing.T) {
	tests := []struct {
		b    []byte
		want bool
	}{
		{nil, true},
		{[]byte{0x02, 0, 0, 0}, true},
		{[]byte{0x06, 0, 0, 0}, true},
		{[]byte{0x03, 0, 0, 0}, false},
		{[]byte{0x01}, false},
	}
	for _, tc := range tests {
		if got := Approved(tc.b); got != tc.want {
			t.Errorf("Approved(%v) = %v", tc.b, got)
		}
	}
}

func TestSortEnabledFirstThenName(t *testing.T) {
	items := []Item{{Name: "b", Enabled: false}, {Name: "z", Enabled: true}, {Name: "A", Enabled: true}}
	Sort(items)
	if items[0].Name != "A" || items[1].Name != "z" || items[2].Name != "b" {
		t.Fatalf("%+v", items)
	}
}

func TestListOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("chỉ Windows")
	}
	items, _, err := List()
	if err != nil || items == nil {
		t.Fatalf("%v %v", items, err)
	}
}
