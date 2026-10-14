package fsinfo_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/fakes/internal/fsinfo"
)

func TestStat(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "a")
	if err := os.WriteFile(path, []byte("abc"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(path, filepath.Join(dir, "b")); err != nil {
		t.Fatal(err)
	}
	got, err := fsinfo.Stat(path)
	want := fsinfo.Info{Path: path, Size: 3, Links: 2, BLAKE3: "6437b3ac38465133ffb63b75273a8db548c558465d79db03fd359c6cd5bd9d85"}
	if err != nil || got != want {
		t.Fatalf("Stat = %+v, %v; want %+v", got, err, want)
	}
	if _, err := fsinfo.Stat(dir); err == nil {
		t.Fatal("Stat of a directory succeeded, want it refused")
	}
}
