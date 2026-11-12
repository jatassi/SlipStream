package imports

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Cells is files.Inflight: grabs' cells, each of an importing Grab waiting
// on what its Import's file for it waits on, and every holding path an
// unfinished Import's file claims. Stateless, so the composition root
// builds it before grabs.New and imports.New.
type Cells struct{ Grabs grabs.GrabCells }

func (c Cells) GrabCells(ctx context.Context, r storage.Reader[file], ws works.Among[works.WorkID]) (map[files.CellKey]files.GrabCell, error) {
	cells, err := c.Grabs.GrabCells(ctx, r, ws)
	if err != nil || len(cells) == 0 {
		return cells, err
	}
	var ids []grabs.GrabID
	for _, cell := range cells {
		if cell.InFlight() {
			ids = append(ids, grabs.GrabID{UUID: cell.Grab})
		}
	}
	imps, err := importsOfGrabs(ctx, r, ids)
	if err != nil {
		return nil, err
	}
	for key, cell := range cells {
		imp, ok := imps[cell.Grab]
		if !ok || !cell.InFlight() {
			continue
		}
		if hold := imp.holdOn(key); hold != "" {
			cell.Waiting = string(hold)
		}
		cell.Doing.Step = imp.stepOn(key)
		cells[key] = cell
	}
	return cells, nil
}

// holdOn is the import wait of a file of im that holds cell k, "" for
// none.
func (im Import) holdOn(k files.CellKey) HoldCode {
	for _, f := range im.Files {
		if f.Hold != nil && f.holds(k) {
			return f.Hold.Code
		}
	}
	return ""
}

// stepOn is the step the furthest file of im that holds cell k is at, ""
// when none holds it.
func (im Import) stepOn(k files.CellKey) files.Step {
	var out files.Step
	for _, f := range im.Files {
		if !f.holds(k) {
			continue
		}
		s := f.step()
		if slices.Index(steps, s) > slices.Index(steps, out) {
			out = s
		}
	}
	return out
}

// steps orders a file's steps from the one furthest from the library.
var steps = []files.Step{"", files.StepReading, files.StepWaiting, files.StepSending, files.StepPlacing}

// step is what a file that holds a cell does for it: a file whose probe
// hold the Admin must answer waits for the Admin as a waiting file does.
func (f *File) step() files.Step {
	switch {
	case f.Outcome == Pending && f.Hold != nil && f.Hold.offered(), f.Outcome == Waiting:
		return files.StepWaiting
	case f.Outcome == Pending:
		return files.StepReading
	case f.Outcome == Sending:
		return files.StepSending
	}
	return files.StepPlacing
}

// importsOfGrabs is the Import of each of ids that has one, by its Grab,
// its unfinished one first, read in r.
func importsOfGrabs(ctx context.Context, r storage.Reader[file], ids []grabs.GrabID) (map[storage.UUID]Import, error) {
	out := map[storage.UUID]Import{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := db.New(r).ListImportsOfGrabs(ctx, ids)
	if err != nil {
		return nil, err
	}
	imps, err := allImports(ctx, r, rows)
	if err != nil {
		return nil, err
	}
	for _, imp := range imps {
		if imp.Finished.IsZero() || out[imp.Grab.UUID].ID.IsZero() {
			out[imp.Grab.UUID] = imp
		}
	}
	return out, nil
}

// Holding is every path in a root's holding folder an unfinished Import's
// file claims, its Dest from the decision on while it isn't final or
// failed for the Admin to retry.
func (c Cells) Holding(ctx context.Context, r storage.Reader[file]) (map[roots.ID]map[string]bool, error) {
	rows, err := db.New(r).ListHolding(ctx)
	if err != nil {
		return nil, err
	}
	out := map[roots.ID]map[string]bool{}
	for _, row := range rows {
		if out[row.RootID] == nil {
			out[row.RootID] = map[string]bool{}
		}
		out[row.RootID][*row.Dest] = true
	}
	return out, nil
}

func (c Cells) ReleaseTitles(ctx context.Context, r storage.Reader[file]) (map[storage.UUID]string, error) {
	return c.Grabs.ReleaseTitles(ctx, r)
}
