package winapi

// Proc: một process từ snapshot.
type Proc struct {
	PID  int    `json:"pid"`
	PPID int    `json:"ppid"`
	Name string `json:"name"`
}

// Descendants trả PID của mọi process con/cháu của root (không gồm root), duyệt theo PPID.
func Descendants(procs []Proc, root int) []int {
	children := map[int][]int{}
	for _, p := range procs {
		if p.PID != p.PPID { // process 0 tự là cha của chính nó
			children[p.PPID] = append(children[p.PPID], p.PID)
		}
	}
	var out []int
	seen := map[int]bool{root: true}
	queue := []int{root}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, c := range children[cur] {
			if !seen[c] {
				seen[c] = true
				out = append(out, c)
				queue = append(queue, c)
			}
		}
	}
	return out
}
