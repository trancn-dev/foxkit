package svgtrace

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Embed nhúng nguyên file ảnh vào SVG (không phải vector).
func Embed(srcPath string) ([]byte, Result, error) {
	res := Result{Colors: []string{}, Warnings: []string{}}
	w, h, err := Size(srcPath)
	if err != nil {
		return nil, res, err
	}
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, res, err
	}
	mime := "image/png"
	switch strings.ToLower(filepath.Ext(srcPath)) {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".gif":
		mime = "image/gif"
	}
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d"><image width="%d" height="%d" href="data:%s;base64,%s"/></svg>`+"\n",
		w, h, w, h, w, h, mime, base64.StdEncoding.EncodeToString(data))
	res.Width, res.Height, res.SVGBytes = w, h, len(svg)
	return []byte(svg), res, nil
}
