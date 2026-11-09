package storing

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

// tree is every file under folder but the marker, as the volume spells
// it.
func tree(t *testing.T, folder string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(folder, func(p string, de fs.DirEntry, err error) error {
		if err != nil || de.IsDir() || de.Name() == MarkerName {
			return err
		}
		out = append(out, filepath.ToSlash(strings.TrimPrefix(p, folder+string(filepath.Separator))))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func holdsFilm(t *testing.T, path string) {
	t.Helper()
	if b, err := os.ReadFile(path); err != nil || string(b) != "film" {
		t.Errorf("%s holds %q, %v; want film", path, b, err)
	}
}

func TestACaseOnlyMoveOnAFoldingVolumeRespellsTheFile(t *testing.T) {
	for _, c := range []struct{ from, to string }{
		{"heat.mkv", "Heat.mkv"},
		{"heat (1995)/x.mkv", "Heat (1995)/x.mkv"},
		{"heat (1995)/heat.mkv", "Heat (1995)/Heat.mkv"},
	} {
		synctest.Test(t, func(t *testing.T) {
			w := newWalking(t, nil)
			w.disk.foldCase = true
			before, err := os.Stat(w.file(t, c.from, "film"))
			if err != nil {
				t.Fatal(err)
			}
			for range 2 {
				if moved, err := w.remote.Move(t.Context(), w.at, mustRel(t, c.from), mustRel(t, c.to), false); err != nil || moved.Size != 4 {
					t.Fatalf("%s to %s: Move = %+v, %v; want 4 bytes", c.from, c.to, moved, err)
				}
			}
			if got := tree(t, w.folder); !slices.Equal(got, []string{c.to}) {
				t.Errorf("%s to %s: the root holds %v, want [%s]", c.from, c.to, got, c.to)
			}
			to := filepath.Join(w.folder, filepath.FromSlash(c.to))
			if after, err := os.Stat(to); err != nil || !os.SameFile(before, after) {
				t.Errorf("%s to %s: %s is not the file moved: %v", c.from, c.to, to, err)
			}
			holdsFilm(t, to)
		})
	}
}

func TestACaseOnlyMoveCutShortFinishesOnRetry(t *testing.T) {
	for _, c := range []struct{ from, to, left string }{
		{"heat.mkv", "Heat.mkv", "Heat.mkv" + recaseSuffix},
		{"heat (1995)/x.mkv", "Heat (1995)/x.mkv", "Heat (1995)" + recaseSuffix + "/x.mkv"},
		{"heat (1995)/heat.mkv", "Heat (1995)/Heat.mkv", "Heat (1995)/Heat.mkv" + recaseSuffix},
	} {
		synctest.Test(t, func(t *testing.T) {
			w := newWalking(t, nil)
			w.disk.foldCase = true
			w.file(t, c.left, "film")
			if _, err := w.remote.Move(t.Context(), w.at, mustRel(t, c.from), mustRel(t, c.to), false); err != nil {
				t.Fatalf("%s to %s from %s: Move = %v", c.from, c.to, c.left, err)
			}
			if got := tree(t, w.folder); !slices.Equal(got, []string{c.to}) {
				t.Errorf("%s to %s from %s: the root holds %v, want [%s]", c.from, c.to, c.left, got, c.to)
			}
			holdsFilm(t, filepath.Join(w.folder, filepath.FromSlash(c.to)))
		})
	}
}

func TestAMoveOntoAHardLinkRemovesTheSource(t *testing.T) {
	for _, fold := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			w := newWalking(t, nil)
			w.disk.foldCase = fold
			heat := w.file(t, "heat.mkv", "film")
			if err := os.Link(heat, filepath.Join(w.folder, "Ronin.mkv")); err != nil {
				t.Skipf("the volume makes no hard links: %v", err)
			}
			if _, err := w.remote.Move(t.Context(), w.at, mustRel(t, "heat.mkv"), mustRel(t, "Ronin.mkv"), false); err != nil {
				t.Fatalf("folding %v: Move = %v", fold, err)
			}
			if got := tree(t, w.folder); !slices.Equal(got, []string{"Ronin.mkv"}) {
				t.Errorf("folding %v: the root holds %v, want [Ronin.mkv]", fold, got)
			}
			holdsFilm(t, filepath.Join(w.folder, "Ronin.mkv"))
		})
	}
}

func TestACaseOnlyMoveOnAVolumeThatKeepsCaseMakesASecondFolder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		if hostFolds(t, w.folder) {
			t.Skip("this host's temp folder ignores case")
		}
		w.file(t, "heat (1995)/x.mkv", "film")
		w.file(t, "heat (1995)/y.mkv", "another film")
		if _, err := w.remote.Move(t.Context(), w.at, mustRel(t, "heat (1995)/x.mkv"), mustRel(t, "Heat (1995)/x.mkv"), false); err != nil {
			t.Fatal(err)
		}
		if got := tree(t, w.folder); !slices.Equal(got, []string{"Heat (1995)/x.mkv", "heat (1995)/y.mkv"}) {
			t.Errorf("the root holds %v, want x.mkv in a second folder", got)
		}
		holdsFilm(t, filepath.Join(w.folder, "Heat (1995)", "x.mkv"))
	})
}

func TestAFolderRespellWaitsForAWalkOfItsRoot(t *testing.T) {
	w := newWalking(t, nil)
	w.disk.foldCase = true
	w.file(t, "a/notes.txt", "notes")
	w.file(t, "heat (1995)/x.mkv", "film")
	release := w.disk.hang(filepath.Join(w.folder, "a"))
	walked := make(chan Page, 1)
	go func() {
		p, err := w.remote.Walk(t.Context(), w.at, Cursor{}, 1000)
		if err != nil {
			t.Error(err)
		}
		walked <- p
	}()
	for w.disk.count("readdir", filepath.Join(w.folder, "a")) == 0 {
		time.Sleep(time.Millisecond)
	}
	moved := make(chan error, 1)
	go func() {
		_, err := w.remote.Move(t.Context(), w.at, mustRel(t, "heat (1995)/x.mkv"), mustRel(t, "Heat (1995)/x.mkv"), false)
		moved <- err
	}()
	time.Sleep(100 * time.Millisecond)
	if got := tree(t, w.folder); !slices.Contains(got, "heat (1995)/x.mkv") {
		t.Errorf("mid-walk the root holds %v, want heat (1995) as the walk found it", got)
	}
	release()
	if p := <-walked; !slices.Equal(paths(p), []string{"heat (1995)/x.mkv"}) {
		t.Errorf("the walk found %v, want [heat (1995)/x.mkv]", paths(p))
	}
	if err := <-moved; err != nil {
		t.Fatal(err)
	}
	if got := tree(t, w.folder); !slices.Equal(got, []string{"Heat (1995)/x.mkv", "a/notes.txt"}) {
		t.Errorf("the root holds %v, want Heat (1995)/x.mkv", got)
	}
}
