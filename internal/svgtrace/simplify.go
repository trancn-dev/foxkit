package svgtrace

import "math"

// distSeg: khoảng cách từ p tới đoạn a–b.
func distSeg(p, a, b Point) float64 {
	dx, dy := b.X-a.X, b.Y-a.Y
	l2 := dx*dx + dy*dy
	if l2 == 0 {
		return math.Hypot(p.X-a.X, p.Y-a.Y)
	}
	t := math.Max(0, math.Min(1, ((p.X-a.X)*dx+(p.Y-a.Y)*dy)/l2))
	return math.Hypot(p.X-(a.X+t*dx), p.Y-(a.Y+t*dy))
}

// rdp: Ramer–Douglas–Peucker trên đường hở; giữ cả hai đầu. eps = 0 giữ mọi điểm.
func rdp(pts []Point, eps float64) []Point {
	if len(pts) < 3 {
		return pts
	}
	idx, dmax := -1, -1.0
	for i := 1; i < len(pts)-1; i++ {
		if d := distSeg(pts[i], pts[0], pts[len(pts)-1]); d > dmax {
			idx, dmax = i, d
		}
	}
	if dmax < eps {
		return []Point{pts[0], pts[len(pts)-1]}
	}
	left := rdp(pts[:idx+1], eps)
	right := rdp(pts[idx:], eps)
	return append(append([]Point{}, left[:len(left)-1]...), right...)
}

// simplifyClosed: vòng khép kín → tách tại hai điểm xa nhau nhất (điểm 0 và điểm xa nó nhất), RDP từng nửa. Còn < 3 điểm thì trả nil.
func simplifyClosed(p []Point, eps float64) []Point {
	if len(p) < 3 {
		return nil
	}
	far, best := 0, -1.0
	for i, q := range p {
		if d := math.Hypot(q.X-p[0].X, q.Y-p[0].Y); d > best {
			far, best = i, d
		}
	}
	a := rdp(p[:far+1], eps)
	b := rdp(append(append([]Point{}, p[far:]...), p[0]), eps)
	out := append(append([]Point{}, a[:len(a)-1]...), b[:len(b)-1]...)
	if len(out) < 3 {
		return nil
	}
	return out
}
