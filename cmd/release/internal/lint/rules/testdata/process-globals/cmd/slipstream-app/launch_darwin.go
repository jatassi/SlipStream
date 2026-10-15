package main

import (
	"os"
	"syscall"
)

func launch(node string, args []string) error {
	return syscall.Exec(node, append([]string{node}, args...), os.Environ())
}
