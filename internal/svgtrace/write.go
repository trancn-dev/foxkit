package svgtrace

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"sort"
	"strconv"
	"strings"
)

type Options struct {
	Colors    int
	Mono      bool
	Threshold uint8
	Fill      string
	Simplify  float64
	Smooth    bool
	Despeckle int
	RemoveBG  bool
	Scale     float64
}

type Result struct {
	Width, Height int
	Paths         int
	Colors        []string
	Points        int
	SVGBytes      int
	Warnings      []string
}

// ErrEmpty: không còn vùng nào để vẽ.
var ErrEmpty = errors.New("không có vùng nào để vector hóa")

func hex(c color.NRGBA) string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

// num ghi số với tối đa 2 chữ số thập phân, bỏ số 0 thừa.
func num(b *strings.Builder, v float64) {
	var buf [32]byte
	s := strconv.AppendFloat(buf[:0], v, 'f', 2, 64)
	if i := strings.IndexByte(string(s), '.'); i >= 0 {
		for s[len(s)-1] == '0' {
			s = s[:len(s)-1]
		}
		if s[len(s)-1] == '.' {
			s = s[:len(s)-1]
		}
	}
	if string(s) == "-0" {
		s = s[1:]
	}
	b.Write(s)
}

func pt(b *strings.Builder, p Point) {
	num(b, p.X)
	b.WriteByte(' ')
	num(b, p.Y)
}

// Trace vector hóa ảnh ít màu thành SVG.
func Trace(img image.Image, opt Options) ([]byte, Result, error) {
	g := toGrid(img)
	res := Result{Width: g.w, Height: g.h, Colors: []string{}, Warnings: []string{}}
	if g.w == 0 || g.h == 0 {
		return nil, res, ErrEmpty
	}
	var pal []color.NRGBA
	var labels []uint8
	if opt.Mono {
		labels = monoLabels(g, opt.Threshold)
		pal = []color.NRGBA{parseFill(opt.Fill)}
	} else {
		var uniq int
		pal, labels, uniq = quantize(g, opt.Colors)
		if uniq > 5000 {
			res.Warnings = append(res.Warnings, "Ảnh có quá nhiều màu (giống ảnh chụp); kết quả có thể xấu và rất nặng. Hãy thử --colors thấp hơn hoặc dùng --mode embed.")
		}
	}
	despeckle(labels, g.w, g.h, opt.Despeckle)

	skip := -1 // nhãn nền bị bỏ
	if opt.RemoveBG && !opt.Mono {
		if m := borderMode(labels, g.w, g.h); m != transparentLabel {
			skip = int(m)
		}
	}
	ar := areas(labels)
	order := make([]int, 0, len(pal))
	for l := range pal {
		if ar[l] > 0 && l != skip {
			order = append(order, l)
		}
	}
	sort.Slice(order, func(i, j int) bool { // lớn trước (nằm dưới); hòa thì theo nhãn cho ổn định
		if ar[order[i]] != ar[order[j]] {
			return ar[order[i]] > ar[order[j]]
		}
		return order[i] < order[j]
	})

	scale := opt.Scale
	if scale <= 0 {
		scale = 1
	}
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 `)
	b.WriteString(strconv.Itoa(g.w) + " " + strconv.Itoa(g.h) + `" width="`)
	num(&b, float64(g.w)*scale)
	b.WriteString(`" height="`)
	num(&b, float64(g.h)*scale)
	b.WriteString("\">\n")
	for _, l := range order {
		var d strings.Builder
		for _, loop := range contours(labels, g.w, g.h, uint8(l)) {
			sp := simplifyClosed(loop, opt.Simplify)
			if sp == nil {
				continue
			}
			res.Points += len(sp)
			d.WriteByte('M')
			pt(&d, sp[0])
			for _, s := range toSegs(sp, opt.Smooth) {
				if s.Curve {
					d.WriteByte('C')
					pt(&d, s.C1)
					d.WriteByte(' ')
					pt(&d, s.C2)
					d.WriteByte(' ')
				} else {
					d.WriteByte('L')
				}
				pt(&d, s.P)
			}
			d.WriteByte('Z')
		}
		if d.Len() == 0 {
			continue
		}
		// shortcut: toạ độ tuyệt đối (M/L/C), chưa dùng lệnh tương đối; đổi khi cần SVG nhẹ hơn nữa.
		c := hex(pal[l])
		b.WriteString(`<path fill="` + c + `" fill-rule="evenodd" d="` + d.String() + "\"/>\n")
		res.Paths++
		res.Colors = append(res.Colors, c)
	}
	if res.Paths == 0 {
		return nil, res, ErrEmpty
	}
	b.WriteString("</svg>\n")
	res.SVGBytes = b.Len()
	return []byte(b.String()), res, nil
}

func parseFill(s string) color.NRGBA {
	s = strings.TrimPrefix(s, "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil || len(s) != 6 {
		return color.NRGBA{A: 255}
	}
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 255}
}
