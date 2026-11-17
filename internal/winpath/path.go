package winpath

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Entry struct {
	Index     int    `json:"index"`
	Dir       string `json:"dir"`
	Expanded  string `json:"expanded"`
	Exists    bool   `json:"exists"`
	Duplicate bool   `json:"duplicate"`
	Empty     bool   `json:"empty"`
	Scope     string `json:"scope,omitempty"` // system | user | process (chỉ có khi đọc được registry)
}

type Report struct {
	Entries  []Entry `json:"entries"`
	Length   int     `json:"length"`
	Problems int     `json:"problems"`
}

// LongPath: một số công cụ cũ cắt PATH dài hơn mức này.
const LongPath = 2047

var varRe = regexp.MustCompile(`%([^%]+)%`)

// Expand thay %VAR% bằng giá trị biến môi trường; biến không có thì giữ nguyên.
func Expand(s string) string {
	return varRe.ReplaceAllStringFunc(s, func(m string) string {
		if v, ok := os.LookupEnv(m[1 : len(m)-1]); ok {
			return v
		}
		return m
	})
}

// Normalize: dạng so sánh trùng (chữ thường, bỏ "\" cuối, Clean).
func Normalize(p string) string {
	return strings.ToLower(strings.TrimRight(filepath.Clean(p), `\`))
}

// Analyze tách PATH theo ";" thủ công (giữ cả mục rỗng để báo) và đánh dấu mục có vấn đề.
func Analyze(pathEnv string) Report {
	rep := Report{Entries: []Entry{}, Length: len(pathEnv)}
	if pathEnv == "" {
		return rep
	}
	seen := map[string]bool{}
	for i, raw := range strings.Split(pathEnv, ";") {
		e := Entry{Index: i + 1, Dir: raw, Expanded: Expand(raw)}
		if strings.TrimSpace(raw) == "" {
			e.Empty = true
		} else {
			if _, err := os.Stat(e.Expanded); err == nil || !os.IsNotExist(err) {
				e.Exists = true // lỗi khác "không tồn tại" (thiếu quyền...) coi là tồn tại
			}
			key := Normalize(e.Expanded)
			e.Duplicate = seen[key]
			seen[key] = true
		}
		if e.Empty || !e.Exists || e.Duplicate {
			rep.Problems++
		}
		rep.Entries = append(rep.Entries, e)
	}
	return rep
}

// Problematic: các mục có vấn đề.
func (r Report) Problematic() []Entry {
	out := []Entry{}
	for _, e := range r.Entries {
		if e.Empty || !e.Exists || e.Duplicate {
			out = append(out, e)
		}
	}
	return out
}

// Label đánh dấu mỗi mục thuộc PATH system, user hay chỉ có trong process (do terminal/IDE thêm). System được ưu tiên khi trùng.
func (r *Report) Label(user, system string) {
	sets := map[string]string{}
	for _, p := range strings.Split(user, ";") {
		sets[Normalize(Expand(strings.TrimSpace(p)))] = "user"
	}
	for _, p := range strings.Split(system, ";") {
		sets[Normalize(Expand(strings.TrimSpace(p)))] = "system"
	}
	for i := range r.Entries {
		if r.Entries[i].Empty {
			continue
		}
		if sc, ok := sets[Normalize(r.Entries[i].Expanded)]; ok {
			r.Entries[i].Scope = sc
		} else {
			r.Entries[i].Scope = "process"
		}
	}
}
