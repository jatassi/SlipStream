package downloading_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

// TestLookFindsFilesByNameAndSize looks in a folder holding Orchard.mkv
// and Show/Harbor.mkv: each is found by its name with either separator at
// its size, and a wrong size or a missing file is not. A name that climbs
// out of the folder is refused.
func TestLookFindsFilesByNameAndSize(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		r := n.startRole()
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "Show"), 0o750); err != nil {
			t.Fatal(err)
		}
		for name, size := range map[string]int{"Orchard.mkv": 1000, filepath.Join("Show", "Harbor.mkv"): 500} {
			if err := os.WriteFile(filepath.Join(dir, name), make([]byte, size), 0o600); err != nil {
				t.Fatal(err)
			}
		}

		res, err := r.Look(t.Context(), &dlv1.LookRequest{Folder: dir, Entries: []*dlv1.LookEntry{
			{Name: "Orchard.mkv", Size: 1000}, {Name: "Show/Harbor.mkv", Size: 500}, {Name: `Show\Harbor.mkv`, Size: 500},
			{Name: "Orchard.mkv", Size: 999}, {Name: "Meadow.mkv", Size: 1000}, {Name: "Show", Size: 0},
		}})
		if err != nil {
			t.Fatal(err)
		}
		if want := []bool{true, true, true, false, false, false}; !slices.Equal(res.GetFound(), want) {
			t.Errorf("Look found %v, want %v", res.GetFound(), want)
		}
		_, err = r.Look(t.Context(), &dlv1.LookRequest{Folder: dir, Entries: []*dlv1.LookEntry{{Name: "../Orchard.mkv", Size: 1000}}})
		if rpc.CodeOf(err) != rpc.CodeInvalidArgument {
			t.Errorf("Look past the folder = %v, want InvalidArgument", err)
		}
	})
}
