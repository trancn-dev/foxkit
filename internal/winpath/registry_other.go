//go:build !windows

package winpath

import "errors"

// ErrUnsupported: chỉ hỗ trợ Windows (file này để go vet/go test chạy được ở nơi khác).
var ErrUnsupported = errors.New("chỉ hỗ trợ Windows")

func Read(Scope) (Raw, error)           { return Raw{}, ErrUnsupported }
func Write(Scope, Raw) error            { return ErrUnsupported }
func Broadcast() error                  { return ErrUnsupported }
func Backup(Scope, Raw) (string, error) { return "", ErrUnsupported }
