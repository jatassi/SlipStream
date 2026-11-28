package files

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

func TestAFailedDownloadIsWaitedOnOnlyByTheTargetsMadeFromIt(t *testing.T) {
	item := works.Item{ID: works.ItemID{UUID: storage.NewUUID()}, Monitored: true}
	target := func() targets.Target { return targets.Target{ID: targets.TargetID{UUID: storage.NewUUID()}} }
	uhd, hd, mobile, tiny := target(), target(), target(), target()
	cell := func(t targets.Target, s State, sourcing targets.Sourcing, from *targets.TargetID) Cell {
		return Cell{Item: item, Edition: targets.AnyEdition, Target: t, State: s, Released: true, Sourcing: sourcing, From: from}
	}
	cells := []Cell{
		cell(uhd, Failed, targets.Download, nil),
		cell(hd, Missing, targets.Download, nil),
		cell(mobile, Missing, targets.DownloadThenDerive, &uhd.ID),
		cell(tiny, Missing, targets.Derive, &uhd.ID),
	}
	got := needs([]works.Item{item}, cells, nil, true)
	if len(got) != 1 || got[0].Kind != NeedFailed || got[0].Target != uhd.ID || !slices.Equal(got[0].Waiting, []targets.TargetID{mobile.ID, tiny.ID}) {
		t.Fatalf("Needs you = %+v, want UHD's failure waited on by Mobile and Tiny, made from it, and not by HD", got)
	}
}
