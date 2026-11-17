package grabs

import (
	"context"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Retention is the Finished Grabs step: imported and failed Grabs older
// than Keep history for, less those still in use. Deleting a Grab deletes
// its stages, Items, cells, carry and search joins, and leaves its
// Blocklist entries, files, import records, Imports and failure counts.
func (gr *Grabs) Retention() housekeeping.Step {
	return housekeeping.Aged("finished-grabs", "Finished Grabs", setting.Noun{One: "Grab", Many: "Grabs"}, "that are still in use", gr.sweep)
}

// finishedAt is the last finished Grab a batch read, in the walk's order.
type finishedAt struct {
	At storage.Time
	ID GrabID
}

// why is each reason a finished Grab past Keep history for stays, one bit
// a reason.
type why uint8

const (
	// forSeeding: its torrent is still in a Download Client.
	forSeeding why = 1 << iota
	// forNeedsYou: it failed, needs the Admin, isn't searched again and a
	// cell reads from it.
	forNeedsYou
	// forImporting: an unfinished Import names it.
	forImporting
	// forShadows: a cell reads from it, and without it another Grab of
	// that cell would be Current.
	forShadows
)

// batchFacts is what one batch's candidates are kept by, read for them
// alone.
type batchFacts struct {
	live      map[downloadclients.ID]bool
	importing map[GrabID]bool
	cells     cellReads
}

// sweep is one Write: up to limit finished Grabs older than before, after
// after, deleting those retained keeps no reason for.
func (gr *Grabs) sweep(ctx context.Context, before time.Time, after finishedAt, limit int) (housekeeping.Swept[finishedAt], error) {
	var (
		out  housekeeping.Swept[finishedAt]
		gone []works.WorkID
	)
	err := gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		out, gone = housekeeping.Swept[finishedAt]{}, nil
		q := db.New(tx)
		cands, err := q.ListFinishedBefore(ctx, db.ListFinishedBeforeParams{
			Before: storage.FromTime(before), AfterAt: after.At, AfterID: after.ID, Limit: int64(limit),
		})
		if err != nil || len(cands) == 0 {
			return err
		}
		f, err := gr.factsOf(ctx, tx, cands)
		if err != nil {
			return err
		}
		keep := retained(cands, f)
		var ids []GrabID
		for _, c := range cands {
			if keep[c.ID] == 0 {
				ids = append(ids, c.ID)
				gone = append(gone, c.WorkID)
			}
		}
		if len(ids) > 0 {
			if err := q.DeleteGrabs(ctx, ids); err != nil {
				return err
			}
		}
		last := cands[len(cands)-1]
		out = housekeeping.Swept[finishedAt]{Seen: len(cands), Deleted: len(ids), Last: finishedAt{At: last.UpdatedAt, ID: last.ID}}
		return nil
	})
	if err != nil {
		return out, err
	}
	slices.SortFunc(gone, func(a, b works.WorkID) int { return a.Compare(b.UUID) })
	for _, w := range slices.Compact(gone) {
		gr.cfg.Changed.Publish(ctx, Changed{WorkID: w.String()})
	}
	return out, nil
}

// factsOf reads, in r, what keeps any of cands.
func (gr *Grabs) factsOf(ctx context.Context, r storage.Reader[file], cands []db.ListFinishedBeforeRow) (batchFacts, error) {
	var torrents []downloadclients.ID
	ids, ws := make([]GrabID, len(cands)), make([]works.WorkID, len(cands))
	for i, c := range cands {
		ids[i], ws[i] = c.ID, c.WorkID
		if !c.TorrentID.IsZero() {
			torrents = append(torrents, c.TorrentID)
		}
	}
	var (
		f   batchFacts
		err error
	)
	if f.live, err = downloadclients.Live(ctx, r, torrents); err != nil {
		return f, err
	}
	imps, err := gr.cfg.Imports.Of(ctx, r, ids)
	if err != nil {
		return f, err
	}
	f.importing = make(map[GrabID]bool, len(imps))
	for id, im := range imps {
		f.importing[id] = !im.Finished
	}
	f.cells, err = readCells(ctx, r, ws)
	return f, err
}

// retained is why each of cands stays, from one batch's facts. A Grab
// that no reason keeps is absent.
func retained(cands []db.ListFinishedBeforeRow, f batchFacts) map[GrabID]why {
	current, again := f.cells.readsOf()
	out := map[GrabID]why{}
	gone := map[GrabID]bool{}
	for _, c := range cands {
		var w why
		if !c.TorrentID.IsZero() && f.live[c.TorrentID] {
			w |= forSeeding
		}
		if f.importing[c.ID] {
			w |= forImporting
		}
		g := Grab{Stage: c.Stage, Source: c.Source}
		if c.FailureCode != nil {
			g.Failed = &Failure{Code: *c.FailureCode}
		}
		if g.NeedsYou() && !again[c.ID] && current[c.ID] {
			w |= forNeedsYou
		}
		if w == 0 {
			gone[c.ID] = true
			continue
		}
		out[c.ID] = w
	}
	shadowing(f.cells.rows, gone, out)
	return out
}

// shadowing moves from gone into kept, as forShadows, each Grab a cell
// reads from that another Grab of the cell, left once gone is deleted,
// would take over unsettled. Keeping one can keep another, so it runs
// until nothing moves.
func shadowing(rows []db.ListTargetsOfWorksRow, gone map[GrabID]bool, kept map[GrabID]why) {
	before := readFrom(rows)
	for {
		left := slices.DeleteFunc(slices.Clone(rows), func(row db.ListTargetsOfWorksRow) bool { return gone[row.ID] })
		after := readFrom(left)
		moved := false
		for key, row := range before {
			if next, ok := after[key]; gone[row.ID] && ok && next.Settled == nil {
				delete(gone, row.ID)
				kept[row.ID] |= forShadows
				moved = true
			}
		}
		if !moved {
			return
		}
	}
}
