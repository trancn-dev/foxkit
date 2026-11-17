package svgtrace

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var (
	white = color.NRGBA{255, 255, 255, 255}
	black = color.NRGBA{0, 0, 0, 255}
	red   = color.NRGBA{255, 0, 0, 255}
	blue  = color.NRGBA{0, 0, 255, 255}
)

func canvas(w, h int, bg color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	rect(img, 0, 0, w, h, bg)
	return img
}

func rect(img *image.NRGBA, x0, y0, x1, y1 int, c color.NRGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}

func disc(img *image.NRGBA, cx, cy, r int, c color.NRGBA) {
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			if (x-cx)*(x-cx)+(y-cy)*(y-cy) <= r*r {
				img.SetNRGBA(x, y, c)
			}
		}
	}
}

func labelsOf(img image.Image) (*grid, []uint8) {
	g := toGrid(img)
	return g, monoLabels(g, 128)
}

func TestContourRect(t *testing.T) {
	img := canvas(10, 10, white)
	rect(img, 2, 3, 7, 8, black)
	g, l := labelsOf(img)
	loops := contours(l, g.w, g.h, 0)
	if len(loops) != 1 {
		t.Fatalf("muốn 1 chu tuyến, có %d", len(loops))
	}
	if s := simplifyClosed(loops[0], 1); len(s) != 4 {
		t.Errorf("sau simplify muốn 4 đỉnh, có %d", len(s))
	}
	if a := area(loops[0]); a != 25 {
		t.Errorf("diện tích = %v, muốn 25", a)
	}
}

func TestContourHole(t *testing.T) {
	img := canvas(12, 12, white)
	rect(img, 1, 1, 11, 11, black)
	rect(img, 4, 4, 8, 8, white)
	g, l := labelsOf(img)
	loops := contours(l, g.w, g.h, 0)
	if len(loops) != 2 {
		t.Fatalf("muốn 2 chu tuyến (ngoài + lỗ), có %d", len(loops))
	}
	sum := area(loops[0]) + area(loops[1])
	if sum != 100-16 {
		t.Errorf("tổng diện tích có dấu = %v, muốn 84", sum)
	}
}

func TestContourSaddleStable(t *testing.T) {
	img := canvas(4, 4, white)
	rect(img, 1, 1, 2, 2, black)
	rect(img, 2, 2, 3, 3, black) // hai ô chạm góc chéo
	g, l := labelsOf(img)
	first := contours(l, g.w, g.h, 0)
	for i := 0; i < 20; i++ {
		again := contours(l, g.w, g.h, 0)
		if len(again) != len(first) {
			t.Fatalf("số chu tuyến đổi giữa các lần chạy: %d vs %d", len(first), len(again))
		}
		for j := range first {
			if len(first[j]) != len(again[j]) {
				t.Fatal("chu tuyến không ổn định")
			}
		}
	}
	var sum float64
	for _, p := range first {
		sum += area(p)
	}
	if sum != 2 {
		t.Errorf("tổng diện tích = %v, muốn 2", sum)
	}
}

func TestRDP(t *testing.T) {
	var line []Point
	for i := 0; i <= 10; i++ {
		line = append(line, Point{float64(i), 0})
	}
	if got := rdp(line, 0.5); len(got) != 2 {
		t.Errorf("đường thẳng: còn %d điểm, muốn 2", len(got))
	}
	bent := []Point{{0, 0}, {5, 3}, {10, 0}}
	if got := rdp(bent, 0); len(got) != 3 {
		t.Errorf("eps=0 phải giữ mọi điểm, còn %d", len(got))
	}
	if got := rdp(bent, 5); len(got) != 2 {
		t.Errorf("eps lớn: còn %d điểm, muốn 2", len(got))
	}
}

func TestMedianCutThreeColors(t *testing.T) {
	img := canvas(30, 10, red)
	rect(img, 10, 0, 20, 10, blue)
	rect(img, 20, 0, 30, 10, white)
	pal, labels, uniq := quantize(toGrid(img), 8)
	if uniq != 3 || len(pal) != 3 {
		t.Fatalf("uniq=%d pal=%d, muốn 3 và 3", uniq, len(pal))
	}
	want := map[color.NRGBA]bool{red: true, blue: true, white: true}
	for _, p := range pal {
		if !want[p] {
			t.Errorf("màu lạ %v", p)
		}
	}
	if labels[0] == labels[15] || labels[15] == labels[25] {
		t.Error("các vùng phải khác nhãn")
	}
}

func TestDespeckle(t *testing.T) {
	img := canvas(20, 20, white)
	rect(img, 2, 2, 12, 12, black)
	rect(img, 15, 15, 17, 16, black) // chấm 2px
	g, l := labelsOf(img)
	if n := despeckle(l, g.w, g.h, 4); n != 1 {
		t.Errorf("dọn %d vùng, muốn 1", n)
	}
	if a := areas(l); a[0] != 100 || a[1] != 300 {
		t.Errorf("sau dọn: mực=%d nền=%d, muốn 100/300 (không tạo lỗ)", a[0], a[1])
	}
}

func TestTransparencyAndBG(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 20, 20)) // toàn alpha 0
	if _, _, err := Trace(img, Options{Colors: 4, Simplify: 1}); err != ErrEmpty {
		t.Errorf("ảnh trong suốt: err=%v, muốn ErrEmpty", err)
	}
	rect(img, 5, 5, 15, 15, red)
	svg, res, err := Trace(img, Options{Colors: 4, Simplify: 1})
	if err != nil || res.Paths != 1 {
		t.Fatalf("err=%v paths=%d, muốn 1 path (vùng trong suốt không sinh path)", err, res.Paths)
	}
	if !strings.Contains(string(svg), "#ff0000") {
		t.Error("thiếu màu đỏ")
	}
	two := canvas(20, 20, white)
	rect(two, 5, 5, 15, 15, red)
	_, keep, _ := Trace(two, Options{Colors: 4, Simplify: 1})
	_, rm, _ := Trace(two, Options{Colors: 4, Simplify: 1, RemoveBG: true})
	if keep.Paths != 2 || rm.Paths != 1 || rm.Colors[0] != "#ff0000" {
		t.Errorf("keep=%d remove=%d %v, muốn 2 và 1 (chỉ còn màu đỏ)", keep.Paths, rm.Paths, rm.Colors)
	}
}

// Test chính: diện tích Shoelace của mọi chu tuyến một màu (lỗ trừ) ≈ số pixel màu đó.
func TestGeometryShoelace(t *testing.T) {
	img := canvas(200, 200, white)
	disc(img, 100, 100, 80, blue)
	disc(img, 100, 100, 30, white) // lỗ
	rect(img, 10, 10, 50, 40, black)
	g, l := labelsOf(img)
	for _, lab := range []uint8{0, 1} {
		px := areas(l)[lab]
		var exact, simp float64
		for _, loop := range contours(l, g.w, g.h, lab) {
			exact += area(loop)
			if s := simplifyClosed(loop, 0.5); s != nil {
				simp += area(s)
			}
		}
		if exact != float64(px) {
			t.Errorf("nhãn %d: chu tuyến thô %v ≠ %d pixel", lab, exact, px)
		}
		if d := math.Abs(simp-float64(px)) / float64(px); d > 0.02 {
			t.Errorf("nhãn %d: sai lệch %.2f%% sau simplify 0.5 (muốn < 2%%)", lab, d*100)
		}
	}
}

func TestTraceLogoRegression(t *testing.T) {
	img := canvas(512, 512, white)
	disc(img, 256, 256, 200, color.NRGBA{30, 136, 229, 255})
	disc(img, 256, 256, 90, white)
	for _, smooth := range []bool{false, true} {
		svg, res, err := Trace(img, Options{Colors: 4, Simplify: 1.5, Smooth: smooth, Despeckle: 4})
		if err != nil {
			t.Fatal(err)
		}
		if res.Paths != 2 || res.Points < 20 || res.Points > 2000 {
			t.Errorf("smooth=%v: paths=%d points=%d ngoài khoảng", smooth, res.Paths, res.Points)
		}
		if res.SVGBytes > 10*1024 {
			t.Errorf("smooth=%v: %d byte, muốn < 10 KB", smooth, res.SVGBytes)
		}
		s := string(svg)
		if !strings.HasPrefix(s, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"`) || !strings.Contains(s, `fill-rule="evenodd"`) {
			t.Errorf("SVG sai khung: %.120s", s)
		}
		if smooth && !strings.Contains(s, "C") {
			t.Error("--smooth phải sinh lệnh C")
		}
	}
}

func TestMonoFillAndScale(t *testing.T) {
	img := canvas(40, 20, white)
	rect(img, 5, 5, 20, 15, black)
	svg, res, err := Trace(img, Options{Mono: true, Threshold: 128, Fill: "#336699", Simplify: 1, Scale: 2})
	if err != nil || res.Paths != 1 {
		t.Fatalf("err=%v paths=%d", err, res.Paths)
	}
	s := string(svg)
	if !strings.Contains(s, `fill="#336699"`) || !strings.Contains(s, `width="80" height="40"`) || !strings.Contains(s, `viewBox="0 0 40 20"`) {
		t.Errorf("fill/scale sai: %s", s)
	}
}

func TestNum(t *testing.T) {
	cases := map[float64]string{1: "1", 1.5: "1.5", 1.234: "1.23", 0.001: "0", -0.001: "0", 10: "10", 100: "100"}
	for in, want := range cases {
		var b strings.Builder
		num(&b, in)
		if b.String() != want {
			t.Errorf("num(%v) = %q, muốn %q", in, b.String(), want)
		}
	}
}

func TestEmbedAndLoad(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.png")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, canvas(7, 5, red)); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	svg, res, err := Embed(p)
	if err != nil || res.Width != 7 || res.Height != 5 {
		t.Fatalf("err=%v res=%+v", err, res)
	}
	if !strings.Contains(string(svg), `href="data:image/png;base64,`) || !strings.Contains(string(svg), `viewBox="0 0 7 5"`) {
		t.Errorf("embed sai: %.150s", svg)
	}
	if _, err := Load(p); err != nil {
		t.Error(err)
	}
	if !Supported("X.JPG") || Supported("a.webp") {
		t.Error("Supported sai")
	}
}

func TestPerf1MPixel(t *testing.T) {
	if testing.Short() {
		t.Skip("bỏ qua ở -short")
	}
	img := canvas(1000, 1000, white)
	cols := []color.NRGBA{red, blue, black, {0, 200, 0, 255}, {200, 200, 0, 255}, {0, 200, 200, 255}, {200, 0, 200, 255}}
	for i, c := range cols {
		disc(img, 150+i*110, 300+(i%3)*150, 90, c)
	}
	start := time.Now()
	_, res, err := Trace(img, Options{Colors: 8, Simplify: 1, Despeckle: 4})
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Errorf("mất %v, muốn < 3s", d)
	}
	if res.Paths < 7 {
		t.Errorf("chỉ có %d path", res.Paths)
	}
}
