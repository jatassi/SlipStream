package grabs

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// GrabWait is what an unfinished Grab waits on for an Admin: the Work page's
// Waiting code, the client's error for client_error, and the root for
// no_space and root_not_ready.
type GrabWait struct {
	Code        string
	ClientError *files.ClientError
	Hold        *files.ImportHold
}

// Pipeline is what Home shows of the Grabs, read in one snapshot.
type Pipeline struct {
	// Jobs is the moving Grabs in importance order, at most rows of them.
	Jobs []View
	// Moving counts every moving Grab.
	Moving int
	// Needs is the failed Grabs that need the Admin and the waiting ones,
	// newest first, at most rows of them.
	Needs []Need
	// Failed counts the failed Grabs that need the Admin up to 100.
	Failed int
	// Waiting counts every waiting Grab.
	Waiting int
}

// Need is one Needs you entry: a failed Grab, Wait nil, or a waiting one.
type Need struct {
	View
	Wait *GrabWait
}

// moving is a Grab that moves with no wait, keyed for importance.
type moving struct {
	id      GrabID
	stop    int
	entered time.Time
}

// Stops in importance order, nearest Available first.
const (
	stopImport = iota
	stopTransfer
	stopDownload
	stopFetch
)

// less orders moving Grabs by importance: the stop nearest Available
// first, then the earliest to enter it. S40 (Requests and approval) adds
// "a Member asked for this" after the stop.
func less(a, b moving) int {
	return cmp.Or(cmp.Compare(a.stop, b.stop), a.entered.Compare(b.entered), a.id.Compare(b.id.UUID))
}

// Pipeline reads Home's Jobs and Needs you in r, each at most rows long.
// A Grab is waiting when its cell on the Work page would be, and moving
// when it is in flight with no wait and no Import file waiting on the
// Admin. An Import delay and an Indexer's fetch wait end by themselves, so
// they don't stop a Grab moving.
func (gr *Grabs) Pipeline(ctx context.Context, r storage.Reader[file], rows int) (Pipeline, error) {
	inflight, err := db.New(r).ListInFlight(ctx, inFlightStages)
	if err != nil {
		return Pipeline{}, err
	}
	open := make([]db.GrabsGrab, len(inflight))
	var importing []GrabID
	for i, row := range inflight {
		open[i] = row.GrabsGrab
		if row.GrabsGrab.Stage == Importing {
			importing = append(importing, row.GrabsGrab.ID)
		}
	}
	cells := GrabCells{InStep: gr.cfg.Roster.InStep, Router: gr.cfg.Router}
	waits, err := cells.waitsOf(ctx, r, open)
	if err != nil {
		return Pipeline{}, err
	}
	imps := map[GrabID]ImportOf{}
	if len(importing) > 0 {
		if imps, err = gr.cfg.Imports.Of(ctx, r, importing); err != nil {
			return Pipeline{}, err
		}
	}
	var jobs []moving
	type waitingGrab struct {
		id      GrabID
		created storage.Time
		wait    GrabWait
	}
	var waiting []waitingGrab
	for _, row := range inflight {
		g, im := row.GrabsGrab, imps[row.GrabsGrab.ID]
		w := waits[g.ID]
		switch {
		case im.Wait != nil:
			waiting = append(waiting, waitingGrab{g.ID, g.CreatedAt, *im.Wait})
		case !cells.InStep(g.NodeID):
			waiting = append(waiting, waitingGrab{g.ID, g.CreatedAt, GrabWait{Code: NodeAway}})
		case w.code != "":
			waiting = append(waiting, waitingGrab{g.ID, g.CreatedAt, GrabWait{Code: w.code, ClientError: w.clientError}})
		case im.Waiting > 0:
		default:
			jobs = append(jobs, moving{id: g.ID, stop: stopOf(g.Stage, im), entered: row.EnteredAt.Time()})
		}
	}
	slices.SortFunc(jobs, less)
	failed, nFailed, err := needsYouWalk(ctx, r, rows)
	if err != nil {
		return Pipeline{}, err
	}
	type needed struct {
		id      GrabID
		created storage.Time
		wait    *GrabWait
	}
	needs := make([]needed, 0, len(failed)+len(waiting))
	for _, f := range failed {
		needs = append(needs, needed{id: f.ID, created: f.CreatedAt})
	}
	for _, w := range waiting {
		needs = append(needs, needed{id: w.id, created: w.created, wait: &w.wait})
	}
	slices.SortFunc(needs, func(a, b needed) int {
		return cmp.Or(b.created.Compare(a.created), b.id.Compare(a.id.UUID))
	})
	out := Pipeline{Moving: len(jobs), Failed: nFailed, Waiting: len(waiting)}
	jobs, needs = jobs[:min(rows, len(jobs))], needs[:min(rows, len(needs))]
	ids := make([]GrabID, 0, len(jobs)+len(needs))
	for _, j := range jobs {
		ids = append(ids, j.id)
	}
	for _, n := range needs {
		ids = append(ids, n.id)
	}
	views, err := gr.viewsOf(ctx, r, ids)
	if err != nil {
		return Pipeline{}, err
	}
	for _, j := range jobs {
		out.Jobs = append(out.Jobs, views[j.id])
	}
	for _, n := range needs {
		out.Needs = append(out.Needs, Need{View: views[n.id], Wait: n.wait})
	}
	return out, nil
}

// stopOf is the stop a moving Grab at stage s is at, im its Import.
func stopOf(s Stage, im ImportOf) int {
	switch {
	case s == Importing && im.Sending > 0:
		return stopTransfer
	case s == Importing:
		return stopImport
	case s == Downloading:
		return stopDownload
	}
	return stopFetch
}

// viewsOf is the view of each of ids, read in r.
func (gr *Grabs) viewsOf(ctx context.Context, r storage.Reader[file], ids []GrabID) (map[GrabID]View, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := db.New(r).ListGrabsByID(ctx, ids)
	if err != nil {
		return nil, err
	}
	grabs, err := loadAll(ctx, r, rows)
	if err != nil {
		return nil, err
	}
	rd, err := gr.readsIn(ctx, r, grabs)
	if err != nil {
		return nil, err
	}
	views, err := gr.views(grabs, rd)
	if err != nil {
		return nil, err
	}
	out := make(map[GrabID]View, len(views))
	for _, v := range views {
		out[v.ID] = v
	}
	return out, nil
}
