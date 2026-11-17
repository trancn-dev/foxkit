package svgtrace

import "math"

// sharp: góc tại cur giữa hai đoạn < 120° thì giữ làm đỉnh nhọn.
func sharp(prev, cur, next Point) bool {
	ax, ay := prev.X-cur.X, prev.Y-cur.Y
	bx, by := next.X-cur.X, next.Y-cur.Y
	la, lb := math.Hypot(ax, ay), math.Hypot(bx, by)
	if la == 0 || lb == 0 {
		return true
	}
	c := (ax*bx + ay*by) / (la * lb)
	return math.Acos(math.Max(-1, math.Min(1, c))) < 2*math.Pi/3
}

// seg: một đoạn của đường dẫn; nếu Curve thì C1, C2 là điểm điều khiển Bézier bậc 3, ngược lại là đoạn thẳng tới P.
type seg struct {
	Curve  bool
	C1, C2 Point
	P      Point
}

// toSegs đổi vòng khép kín thành các đoạn (bắt đầu từ p[0]). Không smooth: toàn đoạn thẳng.
// Smooth: Catmull-Rom → Bézier (tiếp tuyến tại điểm = (kế − trước)/2), đỉnh nhọn có tiếp tuyến 0.
func toSegs(p []Point, smooth bool) []seg {
	n := len(p)
	segs := make([]seg, 0, n)
	tan := make([]Point, n)
	if smooth {
		for i := range p {
			prev, next := p[(i+n-1)%n], p[(i+1)%n]
			if !sharp(prev, p[i], next) {
				tan[i] = Point{(next.X - prev.X) / 2, (next.Y - prev.Y) / 2}
			}
		}
	}
	for i := 1; i <= n; i++ {
		a, b := p[i-1], p[i%n]
		ta, tb := tan[i-1], tan[i%n]
		if !smooth || (ta == Point{} && tb == Point{}) {
			segs = append(segs, seg{P: b})
			continue
		}
		segs = append(segs, seg{Curve: true,
			C1: Point{a.X + ta.X/3, a.Y + ta.Y/3},
			C2: Point{b.X - tb.X/3, b.Y - tb.Y/3},
			P:  b})
	}
	return segs
}
