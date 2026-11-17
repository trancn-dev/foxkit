package startup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

type regSource struct {
	name, scope string
	root        registry.Key
	runPath     string
	approved    string // khóa StartupApproved tương ứng
}

const approvedBase = `Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\`

var regSources = []regSource{
	{"HKCU Run", "user", registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, approvedBase + "Run"},
	{"HKLM Run", "machine", registry.LOCAL_MACHINE, `Software\Microsoft\Windows\CurrentVersion\Run`, approvedBase + "Run"},
	{"HKLM Run (32-bit)", "machine", registry.LOCAL_MACHINE, `Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Run`, approvedBase + "Run32"},
}

func exists(command string) bool {
	exe := ExePath(command)
	if _, err := os.Stat(exe); err == nil {
		return true
	}
	_, err := exec.LookPath(exe) // vd rundll32.exe không có đường dẫn đầy đủ
	return err == nil
}

// approvedValue: trạng thái bật/tắt của một mục do Task Manager ghi ở StartupApproved.
func approvedValue(root registry.Key, path, name string) bool {
	k, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		return true
	}
	defer k.Close()
	b, _, err := k.GetBinaryValue(name)
	if err != nil {
		return true
	}
	return Approved(b)
}

func readRun(s regSource) ([]Item, error) {
	k, err := registry.OpenKey(s.root, s.runPath, registry.QUERY_VALUE)
	if err != nil {
		return nil, err
	}
	defer k.Close()
	names, err := k.ReadValueNames(-1)
	if err != nil {
		return nil, err
	}
	var out []Item
	for _, n := range names {
		v, typ, err := k.GetStringValue(n)
		if err != nil {
			continue
		}
		if typ == registry.EXPAND_SZ {
			if e, err := registry.ExpandString(v); err == nil {
				v = e
			}
		}
		out = append(out, Item{Name: n, Command: v, Source: s.name, Scope: s.scope, Enabled: approvedValue(s.root, s.approved, n), Exists: exists(v)})
	}
	return out, nil
}

func readFolder(dir, source, scope string, root registry.Key) ([]Item, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []Item
	for _, e := range entries {
		if e.IsDir() || strings.EqualFold(e.Name(), "desktop.ini") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		out = append(out, Item{Name: name, Command: p, Source: source, Scope: scope, Enabled: approvedValue(root, approvedBase+"StartupFolder", e.Name()), Exists: true})
	}
	return out, nil
}

// List gom mọi nguồn khởi động; nguồn nào lỗi thì bỏ qua và ghi cảnh báo, chỉ lỗi khi mọi nguồn đều lỗi.
func List() ([]Item, []string, error) {
	var items []Item
	var warns []string
	failed, total := 0, 0
	for _, s := range regSources {
		total++
		it, err := readRun(s)
		if err != nil {
			failed++
			warns = append(warns, fmt.Sprintf("%s: %v", s.name, err))
			continue
		}
		items = append(items, it...)
	}
	folders := []struct {
		dir, name, scope string
		root             registry.Key
	}{
		{filepath.Join(os.Getenv("APPDATA"), `Microsoft\Windows\Start Menu\Programs\Startup`), "Startup folder (user)", "user", registry.CURRENT_USER},
		{filepath.Join(os.Getenv("ProgramData"), `Microsoft\Windows\Start Menu\Programs\StartUp`), "Startup folder (all users)", "machine", registry.LOCAL_MACHINE},
	}
	for _, f := range folders {
		total++
		it, err := readFolder(f.dir, f.name, f.scope, f.root)
		if err != nil {
			failed++
			warns = append(warns, fmt.Sprintf("%s: %v", f.name, err))
			continue
		}
		items = append(items, it...)
	}
	if failed == total {
		return nil, warns, ErrAllFailed
	}
	if items == nil {
		items = []Item{}
	}
	Sort(items)
	return items, warns, nil
}
