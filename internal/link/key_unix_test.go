//go:build !windows

package link_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/link"
)

func TestKeyFileIsOwnerOnly(t *testing.T) {
	dir := t.TempDir()
	if _, err := link.OpenKey(dir); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "node.key"))
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("node.key mode %v, want 0600", mode)
	}
}
