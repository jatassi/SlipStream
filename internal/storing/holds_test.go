package storing

import (
	"path/filepath"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/storage"
)

func TestHoldsIsAPathInARootADownloadsFolderOrTheScratchFolder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newTestDisk(t)
		id := storage.NewUUID()
		root, downloads, elsewhere := marked(t, id), t.TempDir(), t.TempDir()
		d.downloads = []string{downloads}
		d.scratch = dir(t, elsewhere, "converting")
		s := newStore(d)
		running(t, s, intentOf(rootAt(id, root, false))).next(t)
		for _, c := range []struct {
			name string
			path string
			want bool
		}{
			{"a file in the root", write(t, filepath.Join(root, "Heat (1995)", "Heat.mkv"), "x"), true},
			{"a file in a downloads folder", write(t, filepath.Join(downloads, "Heat.1995", "Heat.mkv"), "x"), true},
			{"a file elsewhere", write(t, filepath.Join(elsewhere, "Heat.mkv"), "x"), false},
			{"a source copy in the scratch folder", write(t, filepath.Join(d.scratch, "in", "s-Heat.mkv"), "x"), true},
			{"an output not yet written in the scratch folder", filepath.Join(d.scratch, "out", "c-1-1.mkv"), true},
			{"a file in a sibling of the scratch folder", write(t, filepath.Join(elsewhere, "converting-old", "Heat.mkv"), "x"), false},
			{"a path climbing out of the root", filepath.Join(root, "..", filepath.Base(elsewhere), "Heat.mkv"), false},
			{"a relative path", "Heat.mkv", false},
		} {
			if got := s.Holds(t.Context(), c.path); got != c.want {
				t.Errorf("Holds of %s = %v, want %v", c.name, got, c.want)
			}
		}
	})
}
