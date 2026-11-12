package imports

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// TestAGrabsForcedCellRidesInAsForceOnTheFileOfItsItem grabs a Release
// past not_an_upgrade for one cell: the pending file of that cell's Item
// takes it by Force, and a file of another Item or one waiting doesn't.
func TestAGrabsForcedCellRidesInAsForceOnTheFileOfItsItem(t *testing.T) {
	e1, e2 := works.ItemID{UUID: storage.NewUUID()}, works.ItemID{UUID: storage.NewUUID()}
	hd, sd := targets.TargetID{UUID: storage.NewUUID()}, targets.TargetID{UUID: storage.NewUUID()}
	forced := files.CellKey{Item: e1, Edition: targets.AnyEdition, Target: hd}
	cells := []grabs.Cell{{Key: forced, Forced: true}, {Key: files.CellKey{Item: e1, Edition: targets.AnyEdition, Target: sd}}}
	for _, c := range []struct {
		name string
		f    File
		want []Fill
	}{
		{"its Item's pending file", File{Outcome: Pending, Items: []works.ItemID{e1}}, []Fill{{Cell: forced, Forced: true}}},
		{"another Item's pending file", File{Outcome: Pending, Items: []works.ItemID{e2}}, nil},
		{"a waiting file", File{Outcome: Waiting, Reason: &Reason{Code: NoNumber}}, nil},
	} {
		c.f.force(cells)
		if !slices.Equal(c.f.Fills, c.want) {
			t.Errorf("%s takes %+v, want %+v", c.name, c.f.Fills, c.want)
		}
	}
}
