package rename

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"foxkit/internal/textcase"
)

type Change struct {
	Dir  string `json:"dir"`
	From string `json:"from"`
	To   string `json:"to"`
}

type Conflict struct {
	To     string   `json:"to"`
	From   []string `json:"from"`
	Reason string   `json:"reason"`
}

type Plan struct {
	Applied   bool       `json:"applied"`
	Changes   []Change   `json:"changes"`
	Unchanged int        `json:"unchanged"`
	Conflicts []Conflict `json:"conflicts"`
}

type Options struct {
	Dir        string
	Find       string
	Replace    string
	Regex      bool
	IgnoreCase bool
	WithExt    bool
	Case       string // lower|upper|slug|snake|kebab
	Number     string // mẫu có {n} hoặc {n:3}
	Sort       string // name|date|size
	Start      int
	Exts       []string
	Recursive  bool
	Dirs       bool
}

type entry struct {
	dir, name string
	isDir     bool
	size      int64
	mod       int64
}

func (e entry) path() string { return filepath.Join(e.dir, e.name) }

var numberRe = regexp.MustCompile(`\{n(?::(\d+))?\}`)

// ValidatePattern kiểm mẫu --number có {n} hoặc {n:3}.
func ValidatePattern(p string) bool { return numberRe.MatchString(p) }

func list(opt Options) ([]entry, error) {
	exts := map[string]bool{}
	for _, e := range opt.Exts {
		exts[strings.ToLower(strings.TrimPrefix(strings.TrimSpace(e), "."))] = true
	}
	var out []entry
	add := func(dir string, d fs.DirEntry) {
		if d.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
			return
		}
		if d.IsDir() {
			if opt.Dirs {
				out = append(out, entry{dir: dir, name: d.Name(), isDir: true})
			}
			return
		}
		if len(exts) > 0 && !exts[strings.ToLower(strings.TrimPrefix(filepath.Ext(d.Name()), "."))] {
			return
		}
		e := entry{dir: dir, name: d.Name()}
		if info, err := d.Info(); err == nil {
			e.size, e.mod = info.Size(), info.ModTime().UnixNano()
		}
		out = append(out, e)
	}
	if !opt.Recursive {
		des, err := os.ReadDir(opt.Dir)
		if err != nil {
			return nil, err
		}
		for _, d := range des {
			add(opt.Dir, d)
		}
		return out, nil
	}
	err := filepath.WalkDir(opt.Dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // mục không đọc được thì bỏ qua
		}
		if p != opt.Dir {
			add(filepath.Dir(p), d)
		}
		if d.Type()&fs.ModeSymlink != 0 && d.IsDir() {
			return fs.SkipDir
		}
		return nil
	})
	return out, err
}

func splitName(e entry, withExt bool) (stem, ext string) {
	if e.isDir || withExt {
		return e.name, ""
	}
	ext = filepath.Ext(e.name)
	return strings.TrimSuffix(e.name, ext), ext
}

// naturalLess: so sánh tự nhiên, "img2" < "img10", không phân biệt hoa thường.
func naturalLess(a, b string) bool {
	a, b = strings.ToLower(a), strings.ToLower(b)
	for a != "" && b != "" {
		if unicode.IsDigit(rune(a[0])) && unicode.IsDigit(rune(b[0])) {
			ia, ib := 0, 0
			for ia < len(a) && unicode.IsDigit(rune(a[ia])) {
				ia++
			}
			for ib < len(b) && unicode.IsDigit(rune(b[ib])) {
				ib++
			}
			na, _ := strconv.ParseUint(a[:ia], 10, 64) // chuỗi dài quá giới hạn thì bằng 0, vẫn có thứ tự ổn định ở bước sau
			nb, _ := strconv.ParseUint(b[:ib], 10, 64)
			if na != nb {
				return na < nb
			}
			a, b = a[ia:], b[ib:]
			continue
		}
		if a[0] != b[0] {
			return a[0] < b[0]
		}
		a, b = a[1:], b[1:]
	}
	return len(a) < len(b)
}

var reserved = map[string]bool{"CON": true, "PRN": true, "AUX": true, "NUL": true}

// ValidName: tên hợp lệ trên Windows.
func ValidName(n string) bool {
	if n == "" || strings.ContainsAny(n, `<>:"/\|?*`) || strings.HasSuffix(n, ".") || strings.HasSuffix(n, " ") {
		return false
	}
	for _, r := range n {
		if r < 32 {
			return false
		}
	}
	base := strings.ToUpper(strings.SplitN(n, ".", 2)[0])
	if reserved[base] {
		return false
	}
	if len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9' {
		return false
	}
	return true
}

func newNames(opt Options, es []entry) ([]string, error) {
	out := make([]string, len(es))
	switch {
	case opt.Number != "":
		idx := make([]int, len(es))
		for i := range idx {
			idx[i] = i
		}
		sort.SliceStable(idx, func(a, b int) bool {
			x, y := es[idx[a]], es[idx[b]]
			switch opt.Sort {
			case "date":
				return x.mod < y.mod
			case "size":
				return x.size < y.size
			}
			return naturalLess(x.name, y.name)
		})
		for rank, i := range idx {
			n := opt.Start + rank
			_, ext := splitName(es[i], false)
			name := numberRe.ReplaceAllStringFunc(opt.Number, func(m string) string {
				width := 0
				if sub := numberRe.FindStringSubmatch(m); sub[1] != "" {
					width, _ = strconv.Atoi(sub[1])
				}
				return fmt.Sprintf("%0*d", width, n)
			})
			out[i] = name + ext
		}
	case opt.Case != "":
		for i, e := range es {
			stem, ext := splitName(e, opt.WithExt)
			var s string
			switch opt.Case {
			case "lower":
				s = strings.ToLower(stem)
			case "upper":
				s = strings.ToUpper(stem)
			default:
				c, err := textcase.Convert(opt.Case, stem)
				if err != nil {
					return nil, err
				}
				s = c
			}
			out[i] = s + ext
		}
	default:
		var re *regexp.Regexp
		var err error
		pat := opt.Find
		if !opt.Regex {
			pat = regexp.QuoteMeta(pat)
		}
		if opt.IgnoreCase {
			pat = "(?i)" + pat
		}
		if re, err = regexp.Compile(pat); err != nil {
			return nil, err
		}
		for i, e := range es {
			stem, ext := splitName(e, opt.WithExt)
			if opt.Regex {
				stem = re.ReplaceAllString(stem, opt.Replace)
			} else {
				stem = re.ReplaceAllLiteralString(stem, opt.Replace)
			}
			out[i] = stem + ext
		}
	}
	return out, nil
}

// Build tính danh sách thay đổi và xung đột (không phân biệt hoa thường như NTFS). Không đụng vào đĩa ngoài việc đọc.
func Build(opt Options) (Plan, error) {
	es, err := list(opt)
	if err != nil {
		return Plan{}, err
	}
	names, err := newNames(opt, es)
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{Changes: []Change{}, Conflicts: []Conflict{}}
	type item struct {
		e  entry
		to string
	}
	var changing []item
	for i, e := range es {
		if names[i] == e.name {
			plan.Unchanged++
			continue
		}
		changing = append(changing, item{e, names[i]})
	}
	// thư mục đổi sau cùng, từ sâu nhất lên
	sort.SliceStable(changing, func(a, b int) bool {
		x, y := changing[a], changing[b]
		if x.e.isDir != y.e.isDir {
			return !x.e.isDir
		}
		if x.e.isDir {
			return strings.Count(x.e.path(), string(os.PathSeparator)) > strings.Count(y.e.path(), string(os.PathSeparator))
		}
		return false
	})

	sources := map[string]bool{}
	for _, c := range changing {
		sources[strings.ToLower(c.e.path())] = true
	}
	byTarget := map[string][]string{}
	var order []string
	for _, c := range changing {
		target := filepath.Join(c.e.dir, c.to)
		key := strings.ToLower(target)
		if _, ok := byTarget[key]; !ok {
			order = append(order, key)
		}
		byTarget[key] = append(byTarget[key], c.e.path())
		plan.Changes = append(plan.Changes, Change{Dir: c.e.dir, From: c.e.name, To: c.to})
		if !ValidName(c.to) {
			plan.Conflicts = append(plan.Conflicts, Conflict{To: target, From: []string{c.e.path()}, Reason: "tên không hợp lệ"})
			continue
		}
		if _, err := os.Lstat(target); err == nil && !sources[key] {
			plan.Conflicts = append(plan.Conflicts, Conflict{To: target, From: []string{c.e.path()}, Reason: "file đã tồn tại"})
		}
	}
	for _, key := range order {
		if from := byTarget[key]; len(from) > 1 {
			plan.Conflicts = append(plan.Conflicts, Conflict{To: filepath.Join(filepath.Dir(from[0]), filepath.Base(key)), From: from, Reason: "trùng tên đích"})
		}
	}
	return plan, nil
}
