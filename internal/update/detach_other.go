//go:build !windows

package update

import "os/exec"

func setSysProcAttrDetached(_ *exec.Cmd) {}
