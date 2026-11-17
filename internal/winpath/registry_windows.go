package winpath

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func location(scope Scope) (registry.Key, string, error) {
	switch scope {
	case User:
		return registry.CURRENT_USER, `Environment`, nil
	case System:
		return registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`, nil
	}
	return 0, "", fmt.Errorf("scope không hợp lệ: %q", scope)
}

// readValue đọc giá trị chuỗi chưa expand cùng kiểu của nó; giá trị chưa có → rỗng, EXPAND_SZ.
func readValue(root registry.Key, path, name string) (Raw, error) {
	k, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		return Raw{}, err
	}
	defer k.Close()
	v, typ, err := k.GetStringValue(name)
	if errors.Is(err, registry.ErrNotExist) {
		return Raw{Expand: true}, nil
	}
	if err != nil {
		return Raw{}, err
	}
	return Raw{Value: v, Expand: typ == registry.EXPAND_SZ}, nil
}

// writeValue ghi giữ đúng kiểu: EXPAND_SZ cho giá trị gốc là EXPAND_SZ hoặc có %VAR%, ngược lại SZ.
func writeValue(root registry.Key, path, name string, r Raw) error {
	k, err := registry.OpenKey(root, path, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if NeedsExpand(r) {
		return k.SetExpandStringValue(name, r.Value)
	}
	return k.SetStringValue(name, r.Value)
}

func Read(scope Scope) (Raw, error) {
	root, path, err := location(scope)
	if err != nil {
		return Raw{}, err
	}
	return readValue(root, path, "Path")
}

func Write(scope Scope, r Raw) error {
	root, path, err := location(scope)
	if err != nil {
		return err
	}
	return writeValue(root, path, "Path", r)
}

// Broadcast gửi WM_SETTINGCHANGE("Environment") để Explorer/app mở mới thấy PATH mới.
func Broadcast() error {
	user32 := windows.NewLazySystemDLL("user32.dll")
	proc := user32.NewProc("SendMessageTimeoutW")
	env, err := windows.UTF16PtrFromString("Environment")
	if err != nil {
		return err
	}
	const (
		hwndBroadcast   = 0xFFFF
		wmSettingChange = 0x1A
		smtoAbortIfHung = 0x2
	)
	var result uintptr
	r1, _, callErr := proc.Call(hwndBroadcast, wmSettingChange, 0, uintptr(unsafe.Pointer(env)), smtoAbortIfHung, 5000, uintptr(unsafe.Pointer(&result)))
	if r1 == 0 {
		return callErr
	}
	return nil
}

// Backup ghi giá trị cũ ra %LOCALAPPDATA%\foxkit\backup\path-<scope>-<yyyyMMdd-HHmmss>.txt.
func Backup(scope Scope, r Raw) (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "foxkit", "backup")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	p := filepath.Join(dir, fmt.Sprintf("path-%s-%s.txt", scope, time.Now().Format("20060102-150405")))
	return p, os.WriteFile(p, []byte(r.Value), 0o600)
}
