package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"foxkit/internal/fsutil"
	"foxkit/internal/output"
	"foxkit/internal/svgtrace"

	"github.com/spf13/cobra"
)

type svgItem struct {
	Source      string   `json:"source"`
	Out         string   `json:"out,omitempty"`
	Mode        string   `json:"mode"`
	Width       int      `json:"width"`
	Height      int      `json:"height"`
	Paths       int      `json:"paths"`
	Points      int      `json:"points"`
	Colors      []string `json:"colors"`
	SVGBytes    int      `json:"svgBytes"`
	SourceBytes int64    `json:"sourceBytes"`
	Warnings    []string `json:"warnings"`
	Error       string   `json:"error,omitempty"`
	code        string
}

// svgOutPath: đích cho một file nguồn.
func svgOutPath(src, out string, multi bool) string {
	name := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src)) + ".svg"
	switch {
	case out == "":
		return filepath.Join(filepath.Dir(src), name)
	case multi || fsutil.IsDir(out):
		return filepath.Join(out, name)
	}
	return out
}

// writeAtomic ghi vào .tmp rồi đổi tên.
func writeAtomic(path string, data []byte, force bool) error {
	if !force && fsutil.Exists(path) {
		return os.ErrExist
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp) // dọn file tạm; lỗi dọn không quan trọng bằng lỗi rename đang trả
		return err
	}
	return nil
}

func svgOne(src, out, mode string, opt svgtrace.Options, maxPixels int, force bool) svgItem {
	it := svgItem{Source: src, Out: out, Mode: mode, Colors: []string{}, Warnings: []string{}}
	fail := func(code, msg string) svgItem { it.code, it.Error = code, msg; return it }
	st, err := os.Stat(src)
	if err != nil {
		return fail("FILE_NOT_FOUND", "Không tìm thấy "+src)
	}
	it.SourceBytes = st.Size()
	if !svgtrace.Supported(src) {
		return fail("SVG_FORMAT_UNSUPPORTED", "Không hỗ trợ định dạng của "+src+" (chỉ PNG, JPG, GIF; WebP cần chuyển sang PNG trước)")
	}
	if !force && fsutil.Exists(out) {
		return fail("FILE_EXISTS", out+" đã tồn tại (thêm --force để ghi đè)")
	}
	w, h, err := svgtrace.Size(src)
	if err != nil {
		return fail("SVG_IMAGE_INVALID", "Không đọc được ảnh "+src+": "+err.Error())
	}
	if !force && w*h > maxPixels {
		return fail("SVG_IMAGE_TOO_LARGE", fmt.Sprintf("Ảnh %d×%d vượt %s pixel (thu nhỏ ảnh hoặc thêm --force)", w, h, fsutil.FormatInt(maxPixels)))
	}
	var data []byte
	var res svgtrace.Result
	if mode == "embed" {
		data, res, err = svgtrace.Embed(src)
	} else {
		im, lerr := svgtrace.Load(src)
		if lerr != nil {
			return fail("SVG_IMAGE_INVALID", "Ảnh hỏng: "+lerr.Error())
		}
		data, res, err = svgtrace.Trace(im, opt)
	}
	switch {
	case errors.Is(err, svgtrace.ErrEmpty):
		return fail("SVG_EMPTY", "Không còn vùng nào để vẽ (giảm --despeckle hoặc đổi --threshold)")
	case err != nil:
		return fail("SVG_IMAGE_INVALID", err.Error())
	}
	if it.SourceBytes > 0 && int64(res.SVGBytes) > 3*it.SourceBytes && mode == "trace" {
		res.Warnings = append(res.Warnings, "SVG lớn hơn 3 lần ảnh gốc; thử tăng --simplify hoặc giảm --colors.")
	}
	if err := writeAtomic(out, data, force); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fail("FILE_EXISTS", out+" đã tồn tại (thêm --force để ghi đè)")
		}
		return fail("SVG_WRITE_FAILED", "Không ghi được "+out+": "+err.Error())
	}
	it.Width, it.Height, it.Paths, it.Points = res.Width, res.Height, res.Paths, res.Points
	it.Colors, it.SVGBytes, it.Warnings = res.Colors, res.SVGBytes, res.Warnings
	return it
}

func init() {
	c := &cobra.Command{
		Use:   "svg <file>...",
		Short: "Chuyển PNG/JPG/GIF sang SVG (trace vector hoặc embed)",
		RunE: func(c *cobra.Command, files []string) error {
			f := c.Flags()
			mode, _ := f.GetString("mode")
			out, _ := f.GetString("out")
			colors, _ := f.GetInt("colors")
			mono, _ := f.GetBool("mono")
			threshold, _ := f.GetInt("threshold")
			fill, _ := f.GetString("fill")
			simplify, _ := f.GetString("simplify")
			smooth, _ := f.GetBool("smooth")
			despeckle, _ := f.GetInt("despeckle")
			bg, _ := f.GetString("bg")
			scale, _ := f.GetFloat64("scale")
			maxPixels, _ := f.GetInt("max-pixels")
			force, _ := f.GetBool("force")

			if len(files) == 0 {
				return output.Usage("SVG_NO_INPUT", "Thiếu file ảnh", "Ví dụ: foxkit svg logo.png")
			}
			if mode != "trace" && mode != "embed" {
				return output.Usage("ARGS_INVALID", "--mode phải là trace hoặc embed", "")
			}
			if bg != "keep" && bg != "remove" {
				return output.Usage("ARGS_INVALID", "--bg phải là keep hoặc remove", "")
			}
			if colors < 2 || colors > 32 {
				return output.Usage("SVG_COLORS_INVALID", "--colors phải từ 2 đến 32", "Ví dụ: --colors 8")
			}
			if threshold < 0 || threshold > 255 {
				return output.Usage("ARGS_INVALID", "--threshold phải từ 0 đến 255", "")
			}
			simp, err := strconv.ParseFloat(simplify, 64)
			if err != nil || simp < 0 {
				return output.Usage("ARGS_INVALID", "--simplify phải là số không âm", "Ví dụ: --simplify 1.5")
			}
			if despeckle < 0 || scale <= 0 || maxPixels <= 0 {
				return output.Usage("ARGS_INVALID", "--despeckle, --scale, --max-pixels không hợp lệ", "")
			}
			multi := len(files) > 1
			if multi && out != "" && fsutil.Exists(out) && !fsutil.IsDir(out) {
				return output.Usage("ARGS_INVALID", "Nhiều file thì --out phải là thư mục", "")
			}
			opt := svgtrace.Options{Colors: colors, Mono: mono, Threshold: uint8(threshold), Fill: fill,
				Simplify: simp, Smooth: smooth, Despeckle: despeckle, RemoveBG: bg == "remove", Scale: scale}

			items := make([]svgItem, 0, len(files))
			var first *svgItem
			for _, src := range files {
				it := svgOne(src, svgOutPath(src, out, multi), mode, opt, maxPixels, force)
				items = append(items, it)
				if it.Error != "" && first == nil {
					first = &items[len(items)-1]
				}
			}
			err = printer(c).Success(map[string]any{"items": items}, func(w io.Writer) error {
				for _, it := range items {
					if it.Error != "" {
						fmt.Fprintf(c.ErrOrStderr(), "Lỗi %s: %s\n", it.Source, it.Error)
						continue
					}
					if it.Mode == "embed" {
						fmt.Fprintf(w, "%s (%d×%d) → %s\n  Nhúng ảnh gốc, %s — không phải vector\n", it.Source, it.Width, it.Height, it.Out, fsutil.FormatBytes(int64(it.SVGBytes)))
					} else {
						fmt.Fprintf(w, "%s (%d×%d, %d màu) → %s\n  %d path · %d điểm · %s (gốc %s)\n  Màu: %s\n",
							it.Source, it.Width, it.Height, len(it.Colors), it.Out, it.Paths, it.Points,
							fsutil.FormatBytes(int64(it.SVGBytes)), fsutil.FormatBytes(it.SourceBytes), strings.Join(it.Colors, " "))
					}
					for _, wn := range it.Warnings {
						fmt.Fprintf(c.ErrOrStderr(), "! %s\n", wn)
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
			if first != nil {
				if !multi {
					if first.code == "SVG_FORMAT_UNSUPPORTED" {
						return output.Usage(first.code, first.Error, svgHint(first.code))
					}
					return output.Fail(first.code, first.Error, svgHint(first.code))
				}
				return output.Fail(first.code, "Có file chuyển thất bại (xem chi tiết ở trên)", svgHint(first.code))
			}
			return nil
		},
	}
	f := c.Flags()
	f.String("mode", "trace", "trace|embed")
	f.String("out", "", "File hoặc thư mục đầu ra")
	f.Int("colors", 8, "Số màu tối đa (2-32)")
	f.Bool("mono", false, "Ảnh đen trắng một màu (dùng ngưỡng sáng)")
	f.Int("threshold", 128, "Ngưỡng sáng cho --mono (0-255)")
	f.String("fill", "#000000", "Màu tô cho --mono")
	f.String("simplify", "1.0", "Dung sai Ramer–Douglas–Peucker (pixel)")
	f.Bool("smooth", false, "Làm mượt thành đường cong Bézier")
	f.Int("despeckle", 4, "Bỏ vùng nhỏ hơn N pixel")
	f.String("bg", "keep", "keep|remove")
	f.Float64("scale", 1, "Nhân kích thước hiển thị")
	f.Int("max-pixels", 4_000_000, "Từ chối ảnh lớn hơn số pixel này (trừ --force)")
	f.Bool("force", false, "Ghi đè / bỏ giới hạn pixel")
	rootCmd.AddCommand(c)
}

func svgHint(code string) string {
	switch code {
	case "FILE_EXISTS":
		return "Thêm --force để ghi đè"
	case "SVG_FORMAT_UNSUPPORTED":
		return "Chỉ hỗ trợ PNG, JPG, GIF. WebP cần chuyển sang PNG trước"
	case "SVG_IMAGE_INVALID":
		return "Thử mở file bằng trình xem ảnh để kiểm tra"
	case "SVG_IMAGE_TOO_LARGE":
		return "Thu nhỏ ảnh hoặc thêm --force"
	case "SVG_EMPTY":
		return "Giảm --despeckle hoặc đổi --threshold"
	}
	return ""
}
