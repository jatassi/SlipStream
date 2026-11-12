package grabs

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// TestHeldCellsReadInWantOrder holds every cell of an Item and reads the
// held cells many times: they read in the order of the Item's cells each
// time, as the plan's rows do.
func TestHeldCellsReadInWantOrder(t *testing.T) {
	cells := []files.Cell{cell(hd, files.Missing), cell(small, files.Missing), cell(hd, files.Missing), cell(small, files.Missing)}
	rs := runSheet{run: Search{ItemID: item.ID}, planIn: planIn{item: cells}, held: map[int]db.ListCellGrabsOfWorkRow{}}
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

// TestASheetOfOneEpisodeIsTitledWithItsCode titles a sheet scoped to one
// Item of a Work with levels by the Item's code, as the Work page leads an
// episode with it, and names its Work once.
func TestASheetOfOneEpisodeIsTitledWithItsCode(t *testing.T) {
	season := works.Level{ID: works.LevelID{UUID: storage.NewUUID()}, Label: "Season 2"}
	episode := works.Item{
		ID: works.ItemID{UUID: storage.NewUUID()}, LevelID: season.ID,
		Fields: []works.FieldValue{{Key: "code", Kind: works.KindText, Text: "S02E10"}},
	}
	other := works.Item{ID: works.ItemID{UUID: storage.NewUUID()}, LevelID: season.ID}
	s := sheet{
		in: manualIn{
			tree: works.Tree{Summary: works.Summary{Label: "The Lantern Keepers"}, Levels: []works.Level{season}, Items: []works.Item{other, episode}},
			nouns: works.Description{
				Item: works.Noun{One: "Episode", Many: "Episodes"}, Items: works.ItemsMany,
				Fields: []works.FieldSpec{{Key: "code", Tier: works.TierItem, Kind: works.KindText, In: []works.Place{works.PlaceCode}}},
			},
		},
		runs: []runSheet{{run: Search{ItemID: episode.ID}}},
	}
	if got := s.title(); got != "S02E10 of The Lantern Keepers" {
		t.Fatalf("the sheet is titled %q, want S02E10 of The Lantern Keepers", got)
	}
}
