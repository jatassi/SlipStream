package grabsapi

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

func TestAPacksTargetsNameEachTargetOnce(t *testing.T) {
	hd, uhd := targets.TargetID{UUID: storage.NewUUID()}, targets.TargetID{UUID: storage.NewUUID()}
	var v grabs.View
	for range 3 {
		item := works.ItemID{UUID: storage.NewUUID()}
		for _, target := range []targets.TargetID{hd, uhd} {
			v.Targets = append(v.Targets, grabs.Cell{Key: files.CellKey{Item: item, Edition: targets.AnyEdition, Target: target}})
		}
	}
	got := grabOf(v)
	if want := []string{hd.String(), uhd.String()}; !slices.Equal(got.TargetIDs, want) || len(got.Targets) != 6 {
		t.Errorf("a Grab of 3 Items for 2 Targets lists targetIds %q over %d cells, want %q over 6", got.TargetIDs, len(got.Targets), want)
	}
}
