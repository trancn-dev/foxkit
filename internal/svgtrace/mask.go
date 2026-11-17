package svgtrace

// despeckle gộp mọi thành phần liên thông (4-liên thông cùng nhãn) nhỏ hơn minSize pixel vào nhãn láng giềng chiếm nhiều nhất quanh nó,
// để không tạo lỗ hổng. Chạy một lượt; trả số thành phần đã dọn.
func despeckle(labels []uint8, w, h, minSize int) int {
	if minSize <= 1 {
		return 0
	}
	visited := make([]bool, len(labels))
	var queue []int
	removed := 0
	for start := range labels {
		if visited[start] {
			continue
		}
		l := labels[start]
		queue = append(queue[:0], start)
		visited[start] = true
		var comp []int
		neighbors := map[uint8]int{}
		for qi := 0; qi < len(queue); qi++ {
			i := queue[qi]
			comp = append(comp, i)
			x, y := i%w, i/w
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= w || ny >= h {
					continue
				}
				j := ny*w + nx
				if labels[j] == l {
					if !visited[j] {
						visited[j] = true
						queue = append(queue, j)
					}
				} else {
					neighbors[labels[j]]++
				}
			}
		}
		if len(comp) >= minSize || len(neighbors) == 0 {
			continue
		}
		best, bn := l, -1
		for lab, n := range neighbors {
			if n > bn || (n == bn && lab < best) { // hòa thì chọn nhãn nhỏ hơn để kết quả ổn định
				best, bn = lab, n
			}
		}
		for _, i := range comp {
			labels[i] = best
		}
		removed++
	}
	return removed
}

// borderMode: nhãn xuất hiện nhiều nhất trên viền ảnh (dùng cho --bg remove).
func borderMode(labels []uint8, w, h int) uint8 {
	cnt := map[uint8]int{}
	for x := 0; x < w; x++ {
		cnt[labels[x]]++
		cnt[labels[(h-1)*w+x]]++
	}
	for y := 0; y < h; y++ {
		cnt[labels[y*w]]++
		cnt[labels[y*w+w-1]]++
	}
	best, bn := uint8(transparentLabel), -1
	for l, n := range cnt {
		if n > bn || (n == bn && l < best) {
			best, bn = l, n
		}
	}
	return best
}

// areas: số pixel của từng nhãn.
func areas(labels []uint8) [256]int {
	var a [256]int
	for _, l := range labels {
		a[l]++
	}
	return a
}
