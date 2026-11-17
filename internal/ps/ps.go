package ps

import (
	"context"
	"sort"
	"strings"

	"foxkit/internal/winexec"
)

type Proc struct {
	PID      int    `json:"pid"`
	Name     string `json:"name"`
	MemBytes int64  `json:"memBytes"`
	Session  string `json:"session"`
}

type Group struct {
	Name     string `json:"name"`
	Count    int    `json:"count"`
	MemBytes int64  `json:"memBytes"`
	PIDs     []int  `json:"pids"`
}

func convert(in []winexec.Proc) []Proc {
	out := make([]Proc, len(in))
	for i, p := range in {
		out[i] = Proc{PID: p.PID, Name: p.Name, MemBytes: p.MemBytes, Session: p.Session}
	}
	return out
}

// List: process đang chạy qua tasklist.
func List(ctx context.Context) ([]Proc, error) {
	ps, err := winexec.Tasklist(ctx)
	if err != nil {
		return nil, err
	}
	return convert(ps), nil
}

// Parse đọc CSV của tasklist (dùng cho test với fixture nhiều ngôn ngữ).
func Parse(csv string) ([]Proc, error) {
	ps, err := winexec.ParseTasklist(csv)
	if err != nil {
		return nil, err
	}
	return convert(ps), nil
}

// Filter giữ process có tên chứa substr (không phân biệt hoa thường); substr rỗng giữ tất cả.
func Filter(ps []Proc, substr string) []Proc {
	out := make([]Proc, 0, len(ps))
	q := strings.ToLower(substr)
	for _, p := range ps {
		if strings.Contains(strings.ToLower(p.Name), q) {
			out = append(out, p)
		}
	}
	return out
}

// Sort sắp xếp ổn định theo mem (giảm dần), name hoặc pid.
func Sort(ps []Proc, by string) {
	sort.SliceStable(ps, func(i, j int) bool {
		switch by {
		case "name":
			return strings.ToLower(ps[i].Name) < strings.ToLower(ps[j].Name)
		case "pid":
			return ps[i].PID < ps[j].PID
		}
		return ps[i].MemBytes > ps[j].MemBytes
	})
}

// GroupByName gộp theo tên (không phân biệt hoa thường): cộng RAM, đếm, liệt kê PID.
func GroupByName(ps []Proc) []Group {
	idx := map[string]*Group{}
	var order []*Group
	for _, p := range ps {
		k := strings.ToLower(p.Name)
		g, ok := idx[k]
		if !ok {
			g = &Group{Name: p.Name, PIDs: []int{}}
			idx[k] = g
			order = append(order, g)
		}
		g.Count++
		g.MemBytes += p.MemBytes
		g.PIDs = append(g.PIDs, p.PID)
	}
	out := make([]Group, len(order))
	for i, g := range order {
		out[i] = *g
	}
	return out
}

// SortGroups: mem (giảm dần), name, hoặc pid (PID đầu tiên của nhóm).
func SortGroups(gs []Group, by string) {
	sort.SliceStable(gs, func(i, j int) bool {
		switch by {
		case "name":
			return strings.ToLower(gs[i].Name) < strings.ToLower(gs[j].Name)
		case "pid":
			return gs[i].PIDs[0] < gs[j].PIDs[0]
		}
		return gs[i].MemBytes > gs[j].MemBytes
	})
}
