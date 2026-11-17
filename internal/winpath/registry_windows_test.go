package winpath

import (
	"testing"

	"golang.org/x/sys/windows/registry"
)

// Ghi vào khóa tạm HKCU\Software\foxkit-test (tuyệt đối không đụng HKCU\Environment thật).
func TestWriteKeepsExpandType(t *testing.T) {
	const path = `Software\foxkit-test`
	k, _, err := registry.CreateKey(registry.CURRENT_USER, path, registry.ALL_ACCESS)
	if err != nil {
		t.Skip("không tạo được khóa tạm:", err)
	}
	k.Close()
	defer registry.DeleteKey(registry.CURRENT_USER, path)

	if err := writeValue(registry.CURRENT_USER, path, "Path", Raw{Value: `%USERPROFILE%\bin;C:\a`, Expand: true}); err != nil {
		t.Fatal(err)
	}
	r, err := readValue(registry.CURRENT_USER, path, "Path")
	if err != nil || !r.Expand || r.Value != `%USERPROFILE%\bin;C:\a` {
		t.Fatalf("EXPAND_SZ phải được giữ: %+v %v", r, err)
	}
	// giá trị mới có %VAR% thì phải EXPAND_SZ dù cờ gốc là SZ
	if err := writeValue(registry.CURRENT_USER, path, "Path", Raw{Value: `%SystemRoot%\x`, Expand: false}); err != nil {
		t.Fatal(err)
	}
	if r, _ := readValue(registry.CURRENT_USER, path, "Path"); !r.Expand {
		t.Error("có %VAR% thì không được ghi SZ")
	}
	if err := writeValue(registry.CURRENT_USER, path, "Path", Raw{Value: `C:\plain`}); err != nil {
		t.Fatal(err)
	}
	if r, _ := readValue(registry.CURRENT_USER, path, "Path"); r.Expand || r.Value != `C:\plain` {
		t.Errorf("không có %%: kiểu SZ: %+v", r)
	}
	if r, err := readValue(registry.CURRENT_USER, path, "NoSuchValue"); err != nil || !r.Expand || r.Value != "" {
		t.Errorf("giá trị chưa có phải là rỗng + EXPAND_SZ: %+v %v", r, err)
	}
}
