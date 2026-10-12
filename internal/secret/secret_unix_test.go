//go:build !windows

package secret_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// A directory its owner may write but not read takes the renamed key and
// refuses the open that syncs it, so the sync shows in Open's result.
func TestCreatingTheKeySyncsItsDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root opens a directory whatever its mode")
	}
	lib := storagetest.Library(t, secret.Schema())
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o300); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	_, err := secret.Open(t.Context(), lib, filepath.Join(dir, secret.KeyFile))
	want := "create library.key: sync " + dir + ": open " + dir + ": permission denied"
	if err == nil || err.Error() != want {
		t.Fatalf("Open = %v, want %q", err, want)
	}
}
