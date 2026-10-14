//go:build unix

package e2e

import (
	"os"
	"syscall"
	"testing"
)

func linkCount(t *testing.T, path string) uint64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("stat %s: %T, want a Stat_t", path, info.Sys())
	}
	return widen(st.Nlink)
}

// widen takes Nlink's type, which is uint64 on linux-amd64 and narrower
// elsewhere, so the conversion is never redundant on one target only.
func widen[T uint16 | uint32 | uint64](n T) uint64 { return uint64(n) }
