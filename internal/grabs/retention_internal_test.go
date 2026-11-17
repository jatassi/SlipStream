package grabs

import (
	"context"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// PruneOlderThan runs the Finished Grabs step's batches over Grabs that
// finished before before, limit a batch, with between run after each
// batch's Write.
func (gr *Grabs) PruneOlderThan(ctx context.Context, before time.Time, limit int, between func()) (housekeeping.Tally, error) {
	return housekeeping.Drain(ctx, limit, func(ctx context.Context, after finishedAt, limit int) (housekeeping.Swept[finishedAt], error) {
		s, err := gr.sweep(ctx, before, after, limit)
		between()
		return s, err
	})
}

// aCell is a cell of one Item at Target target.
type aCell struct {
	item   works.ItemID
	target targets.TargetID
}

func newCell() aCell {
	return aCell{works.ItemID{UUID: storage.NewUUID()}, targets.TargetID{UUID: storage.NewUUID()}}
}

// grabOf is a Grab of the cell, in created order where it is listed.
func (c aCell) grabOf(stage Stage, code Code, settled bool) db.ListTargetsOfWorksRow {
	row := db.ListTargetsOfWorksRow{ID: GrabID{UUID: storage.NewUUID()}, ItemID: c.item, Edition: targets.AnyEdition, Stage: stage, Source: FromSearch, TargetID: c.target}
	if stage == Failed {
		row.FailureCode, row.FailedStage = &code, new(Downloading)
	}
	if settled {
		row.Settled = new(db.SettledPlaced)
	}
	return row
}

func finished(row db.ListTargetsOfWorksRow, torrent downloadclients.ID) db.ListFinishedBeforeRow {
	return db.ListFinishedBeforeRow{ID: row.ID, TorrentID: torrent, Stage: row.Stage, FailureCode: row.FailureCode, Source: row.Source}
}

func TestEachReasonKeepsAFinishedGrab(t *testing.T) {
	seeding, needsYou, importing, shadowed, prunable := newCell(), newCell(), newCell(), newCell(), newCell()
	torrent := downloadclients.ID{UUID: storage.NewUUID()}
	rows := []db.ListTargetsOfWorksRow{
		seeding.grabOf(Imported, "", true),
		needsYou.grabOf(Failed, ImportFailed, false),
		importing.grabOf(Imported, "", true),
		shadowed.grabOf(Failed, ImportFailed, false),
		shadowed.grabOf(Imported, "", true),
		prunable.grabOf(Failed, ImportFailed, false),
		prunable.grabOf(Imported, "", true),
	}
	cands := []db.ListFinishedBeforeRow{
		finished(rows[0], torrent), finished(rows[1], downloadclients.ID{}), finished(rows[2], downloadclients.ID{}),
		finished(rows[4], downloadclients.ID{}), finished(rows[5], downloadclients.ID{}), finished(rows[6], downloadclients.ID{}),
	}
	got := retained(cands, batchFacts{
		live:      map[downloadclients.ID]bool{torrent: true},
		importing: map[GrabID]bool{rows[2].ID: true},
		cells:     cellReads{rows: rows},
	})
	want := map[GrabID]why{rows[0].ID: forSeeding, rows[1].ID: forNeedsYou, rows[2].ID: forImporting, rows[4].ID: forShadows}
	if len(got) != len(want) {
		t.Fatalf("retained keeps %v, want %v", got, want)
	}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("Grab %s is kept for %b, want %b", id, got[id], w)
		}
	}
}

func TestAGrabKeptForTwoReasonsHasBoth(t *testing.T) {
	c := newCell()
	torrent := downloadclients.ID{UUID: storage.NewUUID()}
	row := c.grabOf(Failed, ImportFailed, false)
	got := retained([]db.ListFinishedBeforeRow{finished(row, torrent)}, batchFacts{
		live: map[downloadclients.ID]bool{torrent: true}, cells: cellReads{rows: []db.ListTargetsOfWorksRow{row}},
	})
	if got[row.ID] != forSeeding|forNeedsYou {
		t.Fatalf("a seeding failed Grab its cell reads is kept for %b, want seeding and needs you, %b", got[row.ID], forSeeding|forNeedsYou)
	}
}

func TestAFailedGrabSearchedAgainOrCanceledNeedsNoOne(t *testing.T) {
	again, canceled := newCell(), newCell()
	rows := []db.ListTargetsOfWorksRow{again.grabOf(Failed, DownloadFailed, false), canceled.grabOf(Failed, Canceled, false)}
	got := retained([]db.ListFinishedBeforeRow{finished(rows[0], downloadclients.ID{}), finished(rows[1], downloadclients.ID{})},
		batchFacts{cells: cellReads{rows: rows}})
	if len(got) != 0 {
		t.Fatalf("retained keeps %v, want neither a Grab searched again nor a canceled one", got)
	}
}

func TestPruningBothGrabsOfACellTogetherKeepsNeither(t *testing.T) {
	c := newCell()
	rows := []db.ListTargetsOfWorksRow{c.grabOf(Failed, ImportFailed, false), c.grabOf(Imported, "", true)}
	got := retained([]db.ListFinishedBeforeRow{finished(rows[0], downloadclients.ID{}), finished(rows[1], downloadclients.ID{})},
		batchFacts{cells: cellReads{rows: rows}})
	if len(got) != 0 {
		t.Fatalf("retained keeps %v, want both of the cell's Grabs to go together", got)
	}
}
