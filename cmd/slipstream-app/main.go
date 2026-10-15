//go:build windows || darwin

// Command slipstream-app is the desktop launcher that shortcuts, the login
// item and the macOS bundle start: it runs the slipstream binary beside it
// as slipstream app with its own arguments. It imports only std and
// golang.org/x/sys and carries no build stamp, so its bytes change only
// with its source or the Go toolchain. Linux has none: the AppImage runs
// slipstream app itself.
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if err := start(); err != nil {
		report(err)
		os.Exit(1)
	}
}

func start() error {
	exe, err := os.Executable()
	if err == nil {
		exe, err = filepath.EvalSymlinks(exe)
	}
	if err != nil {
		return fmt.Errorf("find the launcher: %w", err)
	}
	node := filepath.Join(filepath.Dir(exe), "slipstream"+filepath.Ext(exe))
	if err := launch(node, append([]string{"app"}, os.Args[1:]...)); err != nil {
		return fmt.Errorf("start %s: %w", node, err)
	}
	return nil
}
