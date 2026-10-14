//go:build unix

package e2e

import (
	"os"
	"syscall"
	"testing"
)

func links(t *testing.T, path string) uint64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return uint64(info.Sys().(*syscall.Stat_t).Nlink) //nolint:forcetypeassert,unconvert // a Unix stat is a Stat_t; Nlink is narrower on some
}
