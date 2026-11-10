package grabs

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// TestHeldCellsReadInWantOrder holds every cell of an Item and reads the
// held cells many times: they read in the order of the Item's cells each
// time, as the plan's rows do.
func TestHeldCellsReadInWantOrder(t *testing.T) {
	cells := []files.Cell{cell(hd, files.Missing), cell(small, files.Missing), cell(hd, files.Missing), cell(small, files.Missing)}
	rs := runSheet{planIn: planIn{item: cells}, held: map[int]db.ListCellGrabsOfWorkRow{}}
	var want []GrabID
	for k := range cells {
		id := GrabID{UUID: storage.NewUUID()}
		rs.held[k] = db.ListCellGrabsOfWorkRow{ID: id, Stage: Downloading, ReleaseTitle: "Orchard.2021.1080p.BluRay-GRP"}
		want = append(want, id)
	}
	s := sheet{in: manualIn{describe: func(title string, _ int64) string { return title }}}
	for range 50 {
		var got []GrabID
		for _, h := range s.heldIn(rs) {
			got = append(got, h.Grab)
		}
		if !slices.Equal(got, want) {
			t.Fatalf("the held cells read as Grabs %v, want %v in the Item's cell order", got, want)
		}
	}
}
