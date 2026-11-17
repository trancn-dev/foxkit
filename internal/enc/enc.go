package enc

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html"
	"net/url"
	"strings"
)

var Modes = []string{"base64", "base64url", "hex", "url", "html"}

func Encode(mode string, in []byte) (string, error) {
	switch mode {
	case "base64":
		return base64.StdEncoding.EncodeToString(in), nil
	case "base64url":
		return base64.RawURLEncoding.EncodeToString(in), nil
	case "hex":
		return hex.EncodeToString(in), nil
	case "url":
		return url.QueryEscape(string(in)), nil
	case "html":
		return html.EscapeString(string(in)), nil
	}
	return "", fmt.Errorf("mode không hỗ trợ: %q", mode)
}

func Decode(mode string, in string) ([]byte, error) {
	in = strings.TrimSpace(in)
	switch mode {
	case "base64":
		if b, err := base64.StdEncoding.DecodeString(in); err == nil {
			return b, nil
		}
		return base64.RawStdEncoding.DecodeString(in)
	case "base64url":
		if b, err := base64.RawURLEncoding.DecodeString(in); err == nil {
			return b, nil
		}
		return base64.URLEncoding.DecodeString(in)
	case "hex":
		return hex.DecodeString(strings.Join(strings.Fields(in), ""))
	case "url":
		s, err := url.QueryUnescape(in)
		return []byte(s), err
	case "html":
		return []byte(html.UnescapeString(in)), nil
	}
	return nil, fmt.Errorf("mode không hỗ trợ: %q", mode)
}
