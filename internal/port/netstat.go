package port

import (
	"sort"
	"strconv"
	"strings"
)

// ParseNetstat đọc output `netstat -ano` và trả các socket TCP đang LISTEN (chưa có tên process).
// LISTEN được nhận ra bằng RemoteAddr = 0.0.0.0:0 hoặc [::]:0, không so chữ LISTENING vì có thể bị dịch.
func ParseNetstat(out string) []Listener {
	items := make([]Listener, 0)
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) != 5 || !strings.EqualFold(f[0], "TCP") {
			continue
		}
		if f[2] != "0.0.0.0:0" && f[2] != "[::]:0" {
			continue
		}
		i := strings.LastIndex(f[1], ":")
		if i < 0 {
			continue
		}
		port, err := strconv.Atoi(f[1][i+1:])
		if err != nil {
			continue
		}
		pid, err := strconv.Atoi(f[4])
		if err != nil {
			continue
		}
		addr := strings.Trim(f[1][:i], "[]")
		items = append(items, Listener{Port: port, Protocol: "TCP", Address: addr, PID: pid})
	}
	sort.SliceStable(items, func(a, b int) bool { return items[a].Port < items[b].Port })
	return items
}
