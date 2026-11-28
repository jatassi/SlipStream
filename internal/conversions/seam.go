package conversions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Seam is files.Config.Conversions. It holds no state of its own, so it
// is built before files; Conversions is the Controller, built after.
type Seam struct {
	Poke        *Poke
	Conversions func() *Conversions
}

var _ files.Conversions = Seam{}

// Cells is the unfinished Conversion of each made-from cell of ws.
func (s Seam) Cells(ctx context.Context, r storage.Reader[storage.Library], ws works.Among[works.WorkID]) (map[files.CellKey]files.ConversionCell, error) {
	q := db.New(r)
	rows, err := works.Read(ws, func() ([]db.ListCellsOfRow, error) {
		every, err := q.ListEveryCell(ctx)
		out := make([]db.ListCellsOfRow, len(every))
		for i, e := range every {
			out[i] = db.ListCellsOfRow(e)
		}
		return out, err
	}, func(ids []works.WorkID) ([]db.ListCellsOfRow, error) { return q.ListCellsOf(ctx, ids) })
	if err != nil {
		return nil, fmt.Errorf("read the Conversions of cells: %w", err)
	}
	now := s.Conversions().cfg.Clock()
	out := map[files.CellKey]files.ConversionCell{}
	for _, row := range rows {
		var items []works.ItemID
		if err := json.Unmarshal([]byte(row.Items), &items); err != nil {
			return nil, fmt.Errorf("read the Conversion %s: %w", row.ID, err)
		}
		v := files.ConversionCell{
			ID: row.ID.UUID, Unfinished: true, Failed: row.FailureCode != nil, Due: !row.NotBefore.Time().After(now),
			Stage: row.Stage, Node: some(row.NodeID), NodeName: row.NodeName,
		}
		if row.FailureCode != nil {
			v.Code, v.Class = *row.FailureCode, string(faults[Code(*row.FailureCode)].Class)
		}
		for _, item := range items {
			out[files.CellKey{Item: item, Edition: row.Edition, Target: row.TargetID}] = v
		}
	}
	return out, nil
}

// Holding is c-<id>-<n>.<ext> and its .part for each unfinished
// Conversion writing into a root's holding folder.
func (Seam) Holding(ctx context.Context, r storage.Reader[storage.Library]) (map[roots.ID]map[string]bool, error) {
	rows, err := db.New(r).ListHolding(ctx)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	goals, err := goalsOf(ctx, r)
	if err != nil {
		return nil, err
	}
	out := map[roots.ID]map[string]bool{}
	for _, row := range rows {
		t := goals[row.TargetID]
		if t.Conversion == nil {
			continue
		}
		if out[row.RootID] == nil {
			out[row.RootID] = map[string]bool{}
		}
		name := holdingRel(row.ID, int(row.Attempt), t.Conversion.Settings.Container).String()
		out[row.RootID][name], out[row.RootID][name+".part"] = true, true
	}
	return out, nil
}

// Leaving cancels, in d's doom Write, every unfinished Conversion of d
// as a source, before its row goes.
func (s Seam) Leaving(ctx context.Context, tx *storage.Tx[storage.Library], d files.File, _ files.ID) error {
	_, err := s.cancelSource(ctx, tx, d.ID)
	return err
}

// Changed starts again from a source whose bytes changed in place: its
// unfinished Conversions cancel, its declines go, and its Work is
// converged.
func (s Seam) Changed(ctx context.Context, tx *storage.Tx[storage.Library], cs []files.PathChange) error {
	q := db.New(tx)
	now := s.Conversions().cfg.Clock()
	for _, ch := range cs {
		if ch.Kind != files.PathReplaced {
			continue
		}
		ws, err := s.cancelSource(ctx, tx, ch.File)
		if err != nil {
			return err
		}
		n, err := q.DeleteDeclinesOfSource(ctx, ch.File)
		if err != nil {
			return fmt.Errorf("clear the source's declines: %w", err)
		}
		if len(ws) == 0 && n > 0 {
			f, err := files.PlayedOf(ctx, tx, ch.File)
			if err != nil {
				return fmt.Errorf("read the source file: %w", err)
			}
			ws = []works.WorkID{f.WorkID}
		}
		for _, w := range ws {
			if err := nudge(ctx, q, files.Ask{Work: w, Why: files.ArrivalChanged, Since: now}, BackfillID{}); err != nil {
				return err
			}
		}
	}
	s.Poke.Wake()
	return nil
}

// cancelSource cancels every unfinished Conversion of src, answering
// their Works.
func (s Seam) cancelSource(ctx context.Context, tx *storage.Tx[storage.Library], src files.ID) ([]works.WorkID, error) {
	c := s.Conversions()
	rows, err := db.New(tx).ListOpenOfSource(ctx, src)
	if err != nil {
		return nil, fmt.Errorf("read the source's Conversions: %w", err)
	}
	var ws []works.WorkID
	for _, row := range rows {
		if err := c.advance(ctx, tx, row.ID, Event{On: Unneeded, Why: SourceGone}); err != nil {
			return nil, err
		}
		ws = append(ws, row.WorkID)
	}
	return ws, nil
}

// Remakes is each Work of ws, or every Work when ws is empty, with files
// t made with settings before version, and queues ws for a remake into
// one remake backfill unless preview. Seconds wait on the forecast.
func (s Seam) Remakes(ctx context.Context, tx *storage.Tx[storage.Library], t targets.TargetID, version int, ws []works.WorkID, preview bool) (files.RemakeImpact, error) {
	c := s.Conversions()
	q := db.New(tx)
	made, err := q.ListMadeBefore(ctx, db.ListMadeBeforeParams{TargetID: t, Version: int64(version)})
	if err != nil {
		return files.RemakeImpact{}, fmt.Errorf("read the files older settings made: %w", err)
	}
	goals, err := goalsOf(ctx, tx)
	if err != nil {
		return files.RemakeImpact{}, err
	}
	settings := goals[t].Conversion.Settings
	var out files.RemakeImpact
	at := map[works.WorkID]int{}
	for _, m := range made {
		if len(ws) > 0 && !slices.Contains(ws, m.WorkID) {
			continue
		}
		i, ok := at[m.WorkID]
		if !ok {
			i, at[m.WorkID] = len(out.Rows), len(out.Rows)
			out.Rows = append(out.Rows, files.MadeRow{Work: m.WorkID})
		}
		out.Older++
		out.Rows[i].Files++
		if m.SourceFileID.IsZero() {
			continue
		}
		src, err := c.cfg.Files.Located(ctx, tx, m.SourceFileID)
		switch {
		case errors.Is(err, files.ErrNoFile):
		case err != nil:
			return files.RemakeImpact{}, fmt.Errorf("read a source file: %w", err)
		default:
			out.Rows[i].Bytes += encode.Bytes(src.Media, settings)
		}
	}
	summaries, err := works.SummariesOf(ctx, tx, slices.Collect(maps.Keys(at)))
	if err != nil {
		return files.RemakeImpact{}, fmt.Errorf("read the Works: %w", err)
	}
	for i, r := range out.Rows {
		out.Rows[i].Added = summaries[r.Work].AddedAt.Time()
	}
	if preview || len(ws) == 0 {
		return out, nil
	}
	a := files.Ask{Why: files.ArrivalRemake, Since: c.cfg.Clock()}
	backfill, err := newBackfill(ctx, q, "remake", t, a)
	if err != nil {
		return files.RemakeImpact{}, err
	}
	for _, w := range ws {
		a.Work = w
		if err := nudge(ctx, q, a, backfill); err != nil {
			return files.RemakeImpact{}, err
		}
	}
	s.Poke.Wake()
	return out, nil
}
