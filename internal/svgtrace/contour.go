package svgtrace

type Point struct{ X, Y float64 }

// Hướng đi dọc cạnh ô lưới, theo chiều kim đồng hồ trên màn hình (y hướng xuống): phải, xuống, trái, lên.
var delta = [4][2]int{{1, 0}, {0, 1}, {-1, 0}, {0, -1}}

// contours dò đường viền khép kín của vùng nhãn l: mỗi cạnh pixel tiếp giáp ngoài vùng là một cạnh có hướng sao cho vùng nằm bên phải
// (tương đương marching squares trên lưới (w+1)×(h+1)). Viền ngoài đi theo chiều kim đồng hồ, lỗ ngược lại.
// Điểm yên ngựa (hai pixel chạm góc chéo) giải quyết bằng quy ước cố định: ưu tiên rẽ phải, rồi đi thẳng, rồi rẽ trái → kết quả ổn định.
func contours(labels []uint8, w, h int, l uint8) [][]Point {
	stride := w + 1
	mask := make([]uint8, stride*(h+1)) // bit d = có cạnh đi ra theo hướng d từ đỉnh này
	on := func(x, y int) bool { return x >= 0 && y >= 0 && x < w && y < h && labels[y*w+x] == l }
	any := false
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if labels[y*w+x] != l {
				continue
			}
			any = true
			if !on(x, y-1) {
				mask[y*stride+x] |= 1 << 0 // cạnh trên, đi sang phải
			}
			if !on(x+1, y) {
				mask[y*stride+x+1] |= 1 << 1 // cạnh phải, đi xuống
			}
			if !on(x, y+1) {
				mask[(y+1)*stride+x+1] |= 1 << 2 // cạnh dưới, đi sang trái
			}
			if !on(x-1, y) {
				mask[(y+1)*stride+x] |= 1 << 3 // cạnh trái, đi lên
			}
		}
	}
	if !any {
		return nil
	}
	var loops [][]Point
	limit := 4 * stride * (h + 1)
	for v := range mask {
		for mask[v] != 0 {
			start := v
			dir := 0
			for mask[v]&(1<<dir) == 0 {
				dir++
			}
			sx, sy := start%stride, start/stride
			pts := []Point{{float64(sx), float64(sy)}}
			cur, firstDir := start, dir
			for steps := 0; steps < limit; steps++ {
				mask[cur] &^= 1 << dir
				cx, cy := cur%stride+delta[dir][0], cur/stride+delta[dir][1]
				next := cy*stride + cx
				if next == start && mask[next]&(1<<firstDir) == 0 {
					break
				}
				nd := -1
				for _, t := range [3]int{(dir + 1) % 4, dir, (dir + 3) % 4} { // rẽ phải, thẳng, rẽ trái
					if mask[next]&(1<<t) != 0 {
						nd = t
						break
					}
				}
				if nd < 0 { // đã quay về điểm xuất phát
					break
				}
				if nd != dir {
					pts = append(pts, Point{float64(cx), float64(cy)}) // chỉ ghi các điểm góc
				}
				cur, dir = next, nd
			}
			loops = append(loops, pts)
		}
	}
	return loops
}

// area: diện tích có dấu (Shoelace). Với màn hình y hướng xuống, viền ngoài đi theo chiều kim đồng hồ cho giá trị dương, lỗ cho giá trị âm.
func area(p []Point) float64 {
	var s float64
	for i := range p {
		j := (i + 1) % len(p)
		s += p[i].X*p[j].Y - p[j].X*p[i].Y
	}
	return s / 2
}
