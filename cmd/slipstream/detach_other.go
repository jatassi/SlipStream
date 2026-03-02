//go:build !windows

package main

import "os/exec"

func setSysProcAttrDetached(_ *exec.Cmd) {}
