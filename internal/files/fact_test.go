package files

import (
	"testing"

	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

func TestAMoveKeepingItsRelIsAMovedChangeFromItsRoot(t *testing.T) {
	from, to := roots.ID{UUID: storage.NewUUID()}, roots.ID{UUID: storage.NewUUID()}
	for _, c := range []struct {
		name     string
		x        fact
		kind     PathChangeKind
		fromRoot roots.ID
	}{
		{"a Move to another root", fact{root: to, fromRoot: from, from: "Heat/Heat.mkv", to: "Heat/Heat.mkv"}, PathMoved, from},
		{"a rename put back", fact{root: to, from: "Heat/Heat.mkv", to: "Heat/Heat.mkv"}, PathUnmoved, roots.ID{}},
		{"a rename in one root", fact{root: to, fromRoot: to, from: "Heat/Heat.mkv", to: "Heat (1995)/Heat.mkv"}, PathMoved, roots.ID{}},
	} {
		got, ok, err := c.x.change()
		if err != nil || !ok {
			t.Fatalf("%s: change() = %v, %v", c.name, ok, err)
		}
		if got.Kind != c.kind || got.FromRoot != c.fromRoot || got.Root != to {
			t.Errorf("%s is a %d change from root %v into %v", c.name, got.Kind, got.FromRoot, got.Root)
		}
	}
}
