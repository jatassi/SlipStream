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
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("stat %s: %T, want a Stat_t", path, info.Sys())
	}
	return uint64(st.Nlink) //nolint:unconvert // Nlink is narrower on some Unixes
}
