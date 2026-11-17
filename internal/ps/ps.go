package ps

import (
	"context"
	"sort"
	"strings"

	"foxkit/internal/winapi"
	"foxkit/internal/winexec"
)

type Proc struct {
	PID      int    `json:"pid"`
	Name     string `json:"name"`
	MemBytes int64  `json:"memBytes"`
	Session  string `json:"session"`
	PPID     int    `json:"ppid,omitempty"` // thêm ở GĐ6 (winapi), JSON cũ giữ nguyên
	Path     string `json:"path,omitempty"`
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

// List: process đang chạy. RAM/session lấy từ tasklist; tên (Unicode đúng), PPID và đường dẫn exe bổ sung bằng winapi
// khi có (không có/thiếu quyền thì giữ dữ liệu của tasklist, không phải lỗi).
func List(ctx context.Context) ([]Proc, error) {
	tl, err := winexec.Tasklist(ctx)
	if err != nil {
		return nil, err
	}
	ps := convert(tl)
	snap, err := winapi.Processes()
	if err != nil {
		return ps, nil
	}
	byPID := make(map[int]winapi.Proc, len(snap))
	for _, p := range snap {
		byPID[p.PID] = p
	}
	for i := range ps {
		if w, ok := byPID[ps[i].PID]; ok {
			ps[i].PPID = w.PPID
			if w.Name != "" {
				ps[i].Name = w.Name
			}
		}
		ps[i].Path, _ = winapi.ProcessPath(ps[i].PID) // process hệ thống: thiếu quyền → để trống
	}
	return ps, nil
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
