package main

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// launch starts node and returns. CREATE_NO_WINDOW gives node a console
// with no window, which its ffmpeg children inherit, so none of them flash
// one.
func launch(node string, args []string) error {
	cmd := exec.CommandContext(context.Background(), node, args...) //nolint:gosec // G204: the slipstream binary beside the launcher
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("release: %w", err)
	}
	return nil
}

// report shows err in a message box, since a GUI program has no console.
func report(err error) {
	text, _ := windows.UTF16PtrFromString(err.Error())
	caption, _ := windows.UTF16PtrFromString("SlipStream")
	_, _ = windows.MessageBox(0, text, caption, windows.MB_OK|windows.MB_ICONERROR)
}
