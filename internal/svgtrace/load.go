package svgtrace

import (
	"fmt"
	"image"
	"image/color"
	_ "image/gif"  // đăng ký định dạng cho image.Decode
	_ "image/jpeg" // (blank import chạy init() của package)
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
)

// ErrFormat: định dạng không được hỗ trợ.
var ErrFormat = fmt.Errorf("chỉ hỗ trợ PNG, JPG, GIF")

// Supported kiểm đuôi file.
func Supported(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif":
		return true
	}
	return false
}

// Size đọc kích thước bằng DecodeConfig (không decode cả ảnh).
func Size(path string) (w, h int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

// Load decode PNG/JPG/GIF (GIF lấy khung đầu).
func Load(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

// grid là ảnh đã chuẩn hóa: pixel alpha < 128 coi là trong suốt, còn lại lấy màu (bỏ alpha).
type grid struct {
	w, h  int
	px    []color.NRGBA
	clear []bool
}

func toGrid(img image.Image) *grid {
	b := img.Bounds()
	g := &grid{w: b.Dx(), h: b.Dy()}
	g.px = make([]color.NRGBA, g.w*g.h)
	g.clear = make([]bool, g.w*g.h)
	for y := 0; y < g.h; y++ {
		for x := 0; x < g.w; x++ {
			c := color.NRGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA) // NRGBA = không nhân trước alpha
			i := y*g.w + x
			if c.A < 128 {
				g.clear[i] = true
				continue
			}
			c.A = 255
			g.px[i] = c
		}
	}
	return g
}
