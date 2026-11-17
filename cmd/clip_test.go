package cmd

import (
	"errors"
	"strings"
	"testing"

	"foxkit/internal/clip"
)

type fakeClip struct {
	text string
	fail bool
}

func (f *fakeClip) Read() (string, error) {
	if f.fail {
		return "", errors.New("không có text")
	}
	return f.text, nil
}

func (f *fakeClip) Write(s string) error {
	if f.fail {
		return errors.New("bận")
	}
	f.text = s
	return nil
}

func withFake(t *testing.T) *fakeClip {
	f := &fakeClip{}
	old := clip.Default
	clip.Default = f
	t.Cleanup(func() { clip.Default = old })
	return f
}

func TestClipWriteReadAndNormalize(t *testing.T) {
	f := withFake(t)
	if _, _, code := execute("clip", "dòng 1\ndòng 2"); code != 0 || f.text != "dòng 1\r\ndòng 2" {
		t.Fatalf("code %d text %q", code, f.text)
	}
	o, _, code := execute("clip")
	if code != 0 || o != "dòng 1\r\ndòng 2" {
		t.Fatalf("đọc lại: %d %q", code, o)
	}
	o, _, _ = execute("clip", "--json")
	if !strings.Contains(o, `"length": 14`) {
		t.Fatalf("length tính theo rune: %s", o)
	}
	if _, _, code := execute("clip", "--clear"); code != 0 || f.text != "" {
		t.Fatalf("clear: %d %q", code, f.text)
	}
}

func TestClipErrors(t *testing.T) {
	f := withFake(t)
	f.fail = true
	if _, e, code := execute("clip", "x"); code != 1 || !strings.Contains(e, "Clipboard đang bị") {
		t.Fatalf("CLIP_BUSY: %d %q", code, e)
	}
	if _, e, code := execute("clip"); code != 1 || !strings.Contains(e, "Không đọc được chữ") {
		t.Fatalf("CLIP_NO_TEXT: %d %q", code, e)
	}
}
