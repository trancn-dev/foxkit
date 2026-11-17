//go:build !windows

package winapi

import "errors"

// ErrUnsupported: API này chỉ có trên Windows (file này để go vet/go test chạy được ở nơi khác).
var ErrUnsupported = errors.New("chỉ hỗ trợ Windows")

func IsAdmin() bool                   { return false }
func ProcessPath(int) (string, error) { return "", ErrUnsupported }
func Processes() ([]Proc, error)      { return nil, ErrUnsupported }
