package svgtrace

import (
	"image/color"
	"sort"
)

const transparentLabel = 255

type cc struct {
	c uint32 // 0xRRGGBB
	n int
}

func pack(c color.NRGBA) uint32 { return uint32(c.R)<<16 | uint32(c.G)<<8 | uint32(c.B) }

func unpack(v uint32) (r, g, b uint8) { return uint8(v >> 16), uint8(v >> 8), uint8(v) }

func channel(v uint32, ch int) uint8 {
	r, g, b := unpack(v)
	return [3]uint8{r, g, b}[ch]
}

// box: một hộp màu trong median cut.
type box struct{ items []cc }

func (b box) widest() (ch int, rng int) {
	var lo, hi [3]int
	for i := range lo {
		lo[i], hi[i] = 255, 0
	}
	for _, it := range b.items {
		for c := 0; c < 3; c++ {
			v := int(channel(it.c, c))
			lo[c], hi[c] = min(lo[c], v), max(hi[c], v)
		}
	}
	for c := 0; c < 3; c++ {
		if hi[c]-lo[c] > rng {
			ch, rng = c, hi[c]-lo[c]
		}
	}
	return
}

func (b box) mean() color.NRGBA {
	var sr, sg, sb, n int
	for _, it := range b.items {
		r, g, bl := unpack(it.c)
		sr, sg, sb, n = sr+int(r)*it.n, sg+int(g)*it.n, sb+int(bl)*it.n, n+it.n
	}
	if n == 0 {
		return color.NRGBA{A: 255}
	}
	return color.NRGBA{R: uint8(sr / n), G: uint8(sg / n), B: uint8(sb / n), A: 255}
}

// medianCut gộp các màu (kèm số pixel) thành tối đa k màu bằng cách chia hộp RGB theo kênh có biên độ lớn nhất tại trung vị (theo trọng số pixel).
func medianCut(hist []cc, k int) []color.NRGBA {
	boxes := []box{{items: hist}}
	for len(boxes) < k {
		bi, bestRange, bestCh := -1, 0, 0
		for i, b := range boxes {
			if len(b.items) < 2 {
				continue
			}
			if ch, r := b.widest(); r > bestRange {
				bi, bestRange, bestCh = i, r, ch
			}
		}
		if bi < 0 {
			break // không còn hộp nào tách được (ít màu hơn k)
		}
		b := boxes[bi]
		sort.Slice(b.items, func(i, j int) bool { return channel(b.items[i].c, bestCh) < channel(b.items[j].c, bestCh) })
		total := 0
		for _, it := range b.items {
			total += it.n
		}
		half, cut := 0, 1
		for i, it := range b.items {
			half += it.n
			if half*2 >= total {
				cut = i + 1
				break
			}
		}
		cut = max(1, min(cut, len(b.items)-1))
		boxes[bi] = box{items: b.items[:cut:cut]}
		boxes = append(boxes, box{items: b.items[cut:]})
	}
	pal := make([]color.NRGBA, len(boxes))
	for i, b := range boxes {
		pal[i] = b.mean()
	}
	return pal
}

func nearest(pal []color.NRGBA, v uint32) uint8 {
	r, g, b := unpack(v)
	best, bd := 0, 1<<30
	for i, p := range pal {
		dr, dg, db := int(r)-int(p.R), int(g)-int(p.G), int(b)-int(p.B)
		if d := dr*dr + dg*dg + db*db; d < bd {
			best, bd = i, d
		}
	}
	return uint8(best)
}

// quantize trả bảng màu và nhãn từng pixel (255 = trong suốt); uniq = số màu gốc duy nhất (dừng đếm ở 5000).
func quantize(g *grid, colors int) (pal []color.NRGBA, labels []uint8, uniq int) {
	counts := map[uint32]int{}
	for i, c := range g.px {
		if !g.clear[i] {
			counts[pack(c)]++
		}
	}
	uniq = len(counts)
	hist := make([]cc, 0, len(counts))
	for c, n := range counts {
		hist = append(hist, cc{c, n})
	}
	sort.Slice(hist, func(i, j int) bool { return hist[i].c < hist[j].c }) // thứ tự cố định → kết quả lặp lại được
	pal = medianCut(hist, colors)
	cache := make(map[uint32]uint8, len(counts))
	labels = make([]uint8, len(g.px))
	for i, c := range g.px {
		if g.clear[i] {
			labels[i] = transparentLabel
			continue
		}
		k := pack(c)
		l, ok := cache[k]
		if !ok {
			l = nearest(pal, k)
			cache[k] = l
		}
		labels[i] = l
	}
	return pal, labels, uniq
}

// monoLabels: nhãn 0 = mực (tối hơn ngưỡng và không trong suốt), 1 = nền, 255 = trong suốt.
func monoLabels(g *grid, threshold uint8) []uint8 {
	labels := make([]uint8, len(g.px))
	for i, c := range g.px {
		switch {
		case g.clear[i]:
			labels[i] = transparentLabel
		case 0.299*float64(c.R)+0.587*float64(c.G)+0.114*float64(c.B) < float64(threshold):
			labels[i] = 0
		default:
			labels[i] = 1
		}
	}
	return labels
}
