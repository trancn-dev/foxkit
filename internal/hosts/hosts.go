package hosts

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Line struct {
	Raw     string
	IP      string   // rỗng nếu không phải dòng entry
	Domains []string // một dòng có thể nhiều domain
	Enabled bool     // false nếu dòng entry bị comment "# 127.0.0.1 x"
	Managed bool     // có hậu tố "# foxkit"
	Comment string   // comment cuối dòng (ngoài "# foxkit")
}

type File struct {
	Lines        []Line
	CRLF         bool
	BOM          bool
	FinalNewline bool
}

type Entry struct {
	IP      string `json:"ip"`
	Domain  string `json:"domain"`
	Enabled bool   `json:"enabled"`
	Managed bool   `json:"managed"`
	Line    int    `json:"line"`
}

// Change.Action: add, remove, enable, disable, skip, và hai trạng thái lỗi cho cmd: conflict (domain đã có với IP khác), notfound.
type Change struct {
	Action  string `json:"action"`
	Domain  string `json:"domain"`
	IP      string `json:"ip"`
	Line    int    `json:"line"`
	Managed bool   `json:"managed"`
}

const tag = "foxkit"

var domainRe = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*$`)

func ValidDomain(d string) bool { return domainRe.MatchString(d) }
func ValidIP(ip string) bool    { return net.ParseIP(ip) != nil }

// DefaultPath: FOXKIT_HOSTS (để test/dùng thử) hoặc file hosts hệ thống.
func DefaultPath() string {
	if p := os.Getenv("FOXKIT_HOSTS"); p != "" {
		return p
	}
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = `C:\Windows`
	}
	return filepath.Join(root, "System32", "drivers", "etc", "hosts")
}

func parseLine(raw string) Line {
	l := Line{Raw: raw}
	t := strings.TrimSpace(raw)
	disabled := strings.HasPrefix(t, "#")
	body := t
	if disabled {
		body = strings.TrimSpace(strings.TrimPrefix(t, "#"))
	}
	main := body
	if i := strings.Index(body, "#"); i >= 0 {
		main = body[:i]
		c := strings.TrimSpace(body[i+1:])
		if c == tag {
			l.Managed = true
		} else {
			l.Comment = c
		}
	}
	f := strings.Fields(main)
	if len(f) >= 2 && ValidIP(f[0]) {
		l.IP, l.Domains, l.Enabled = f[0], f[1:], !disabled
		return l
	}
	l.Managed, l.Comment = false, ""
	return l
}

func (l *Line) render() {
	s := l.IP + " " + strings.Join(l.Domains, " ")
	if l.Managed {
		s += " # " + tag
	}
	if l.Comment != "" {
		s += " # " + l.Comment
	}
	if !l.Enabled {
		s = "# " + s
	}
	l.Raw = s
}

// Parse đọc nội dung hosts, nhớ BOM/kiểu xuống dòng để ghi lại giống hệt.
func Parse(data []byte) *File {
	f := &File{}
	s := string(data)
	if strings.HasPrefix(s, "\xef\xbb\xbf") {
		f.BOM = true
		s = s[3:]
	}
	f.CRLF = strings.Contains(s, "\r\n")
	if s == "" {
		return f
	}
	f.FinalNewline = strings.HasSuffix(s, "\n")
	s = strings.TrimSuffix(s, "\n")
	for _, raw := range strings.Split(s, "\n") {
		f.Lines = append(f.Lines, parseLine(strings.TrimSuffix(raw, "\r")))
	}
	return f
}

func Load(path string) (*File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(b), nil
}

// Bytes dựng lại nội dung file (không sửa gì thì giống hệt bản gốc).
func (f *File) Bytes() []byte {
	eol := "\n"
	if f.CRLF {
		eol = "\r\n"
	}
	parts := make([]string, len(f.Lines))
	for i, l := range f.Lines {
		parts[i] = l.Raw
	}
	s := strings.Join(parts, eol)
	if f.FinalNewline && len(parts) > 0 {
		s += eol
	}
	if f.BOM {
		s = "\xef\xbb\xbf" + s
	}
	return []byte(s)
}

func (f *File) Entries() []Entry {
	out := []Entry{}
	for i, l := range f.Lines {
		for _, d := range l.Domains {
			out = append(out, Entry{IP: l.IP, Domain: d, Enabled: l.Enabled, Managed: l.Managed, Line: i + 1})
		}
	}
	return out
}

// find: dòng đầu tiên chứa domain (không phân biệt hoa thường) và vị trí domain trong dòng.
func (f *File) find(domain string) (line, idx int) {
	for i, l := range f.Lines {
		for j, d := range l.Domains {
			if strings.EqualFold(d, domain) {
				return i, j
			}
		}
	}
	return -1, -1
}

// Add thêm domain vào cuối file (có "# foxkit"); bật lại nếu đang tắt cùng IP; đã bật cùng IP thì skip; khác IP thì conflict.
func (f *File) Add(ip string, domains []string) []Change {
	var out []Change
	for _, d := range domains {
		i, _ := f.find(d)
		switch {
		case i < 0:
			l := Line{IP: ip, Domains: []string{d}, Enabled: true, Managed: true}
			l.render()
			f.Lines = append(f.Lines, l)
			out = append(out, Change{"add", d, ip, len(f.Lines), true})
		case f.Lines[i].IP != ip:
			out = append(out, Change{"conflict", d, f.Lines[i].IP, i + 1, f.Lines[i].Managed})
		case f.Lines[i].Enabled:
			out = append(out, Change{"skip", d, ip, i + 1, f.Lines[i].Managed})
		default:
			out = append(out, f.SetEnabled([]string{d}, true)...)
		}
	}
	return out
}

// Remove xóa domain khỏi dòng; dòng hết domain thì xóa dòng.
func (f *File) Remove(domains []string) []Change {
	var out []Change
	for _, d := range domains {
		i, j := f.find(d)
		if i < 0 {
			out = append(out, Change{Action: "notfound", Domain: d})
			continue
		}
		l := &f.Lines[i]
		out = append(out, Change{"remove", d, l.IP, i + 1, l.Managed})
		l.Domains = append(l.Domains[:j:j], l.Domains[j+1:]...)
		if len(l.Domains) == 0 {
			f.Lines = append(f.Lines[:i:i], f.Lines[i+1:]...)
		} else {
			l.render()
		}
	}
	return out
}

// SetEnabled bật/tắt domain. Dòng nhiều domain mà chỉ đổi một → tách domain đó ra dòng riêng.
func (f *File) SetEnabled(domains []string, on bool) []Change {
	var out []Change
	act := "disable"
	if on {
		act = "enable"
	}
	for _, d := range domains {
		i, j := f.find(d)
		if i < 0 {
			out = append(out, Change{Action: "notfound", Domain: d})
			continue
		}
		l := f.Lines[i]
		if l.Enabled == on {
			out = append(out, Change{"skip", d, l.IP, i + 1, l.Managed})
			continue
		}
		if len(l.Domains) == 1 {
			f.Lines[i].Enabled = on
			f.Lines[i].render()
			out = append(out, Change{act, d, l.IP, i + 1, l.Managed})
			continue
		}
		// tách: dòng gốc giữ các domain còn lại, domain này ra dòng mới ngay sau
		rest := append(append([]string{}, l.Domains[:j]...), l.Domains[j+1:]...)
		f.Lines[i].Domains = rest
		f.Lines[i].render()
		nl := Line{IP: l.IP, Domains: []string{d}, Enabled: on, Managed: l.Managed}
		nl.render()
		f.Lines = append(f.Lines[:i+1], append([]Line{nl}, f.Lines[i+1:]...)...)
		out = append(out, Change{act, d, l.IP, i + 2, l.Managed})
	}
	return out
}

// Save ghi nguyên tử: backup `.foxkit.bak` → ghi `.foxkit.tmp` → rename. Lỗi quyền → ErrPermission (bọc).
func Save(path string, f *File) (backup string, err error) {
	dir := filepath.Dir(path)
	backup = filepath.Join(dir, filepath.Base(path)+".foxkit.bak")
	tmp := filepath.Join(dir, filepath.Base(path)+".foxkit.tmp")
	if old, rerr := os.ReadFile(path); rerr == nil {
		if err := os.WriteFile(backup, old, 0o644); err != nil {
			return "", err
		}
	} else if !errors.Is(rerr, os.ErrNotExist) {
		return "", rerr
	} else {
		backup = ""
	}
	if err := os.WriteFile(tmp, f.Bytes(), 0o644); err != nil {
		_ = os.Remove(tmp) // dọn file tạm dở dang; lỗi xóa không quan trọng bằng lỗi ghi
		return backup, err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return backup, err
	}
	return backup, nil
}
