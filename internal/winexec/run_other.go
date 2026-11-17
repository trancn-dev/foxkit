//go:build !windows

package winexec

import "os/exec"

func hideWindow(*exec.Cmd) {}
