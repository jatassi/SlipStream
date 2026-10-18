//go:build linux || darwin

package storing

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestOSDiskListsLinksToFoldersAsFolders(t *testing.T) {
	ctx, d, dir := t.Context(), newOSDisk(t), t.TempDir()
	movies := filepath.Join(dir, "Movies")
	for _, p := range []string{movies, filepath.Join(dir, ".cache"), filepath.Join(dir, "@eaDir")} {
		if err := os.Mkdir(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(movies, filepath.Join(dir, "films")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "gone"), filepath.Join(dir, "broken")); err != nil {
		t.Fatal(err)
	}
	entries, err := d.ReadDir(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []Entry{
		{Name: ".cache", Dir: true, Hidden: true, Readable: true},
		{Name: "@eaDir", Dir: true, Hidden: true, Readable: true},
		{Name: "Movies", Dir: true, Readable: true},
		{Name: "broken", Readable: true},
		{Name: "films", Dir: true, Readable: true},
	}
	if !slices.Equal(entries, want) {
		t.Errorf("ReadDir = %+v, want %+v", entries, want)
	}
	viaLink, err := d.Stat(ctx, filepath.Join(dir, "films"))
	if err != nil {
		t.Fatal(err)
	}
	direct, _ := d.Stat(ctx, movies)
	if !viaLink.Dir() || !viaLink.Same(direct) {
		t.Error("Stat through a link is not the folder it names")
	}
}

func TestOSDiskUnreadableAndUnwritableFolders(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads and writes every folder")
	}
	ctx, d, dir := t.Context(), newOSDisk(t), t.TempDir()
	locked, ro := filepath.Join(dir, "locked"), filepath.Join(dir, "ro")
	if err := os.Mkdir(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(ro, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700); _ = os.Chmod(ro, 0o700) }) //nolint:gosec // TempDir's cleanup needs to enter and empty them
	entries, err := d.ReadDir(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []Entry{{Name: "locked", Dir: true}, {Name: "ro", Dir: true, Readable: true}}
	if !slices.Equal(entries, want) {
		t.Errorf("ReadDir = %+v, want %+v", entries, want)
	}
	if ok, err := d.Writable(ctx, ro); ok || err != nil {
		t.Errorf("Writable(0555 folder) = %v, %v; want false", ok, err)
	}
}
