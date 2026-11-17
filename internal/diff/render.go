package diff

import (
	"fmt"
	"strings"
)

const (
	green = "\x1b[32m"
	red   = "\x1b[31m"
	cyan  = "\x1b[36m"
	reset = "\x1b[0m"
)

// Unified in hunks theo định dạng unified diff; color=true thì tô màu (chỉ dùng khi stdout là terminal).
func Unified(aName, bName string, hunks []Hunk, color bool) string {
	paint := func(c, s string) string {
		if color {
			return c + s + reset
		}
		return s
	}
	var b strings.Builder
	fmt.Fprintf(&b, "--- %s\n+++ %s\n", aName, bName)
	for _, h := range hunks {
		b.WriteString(paint(cyan, fmt.Sprintf("@@ -%d,%d +%d,%d @@", h.AStart, h.ALen, h.BStart, h.BLen)) + "\n")
		for _, l := range h.Lines {
			text := strings.TrimSuffix(l.Text, "\r")
			switch l.Op {
			case Add:
				b.WriteString(paint(green, "+"+text) + "\n")
			case Del:
				b.WriteString(paint(red, "-"+text) + "\n")
			default:
				b.WriteString(" " + text + "\n")
			}
		}
	}
	return b.String()
}
