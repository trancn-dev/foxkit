//go:build !windows

package sysinfo

func Get() (Info, error) { return Info{}, ErrUnsupported }
