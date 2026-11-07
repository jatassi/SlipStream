package works

import (
	"context"
	"errors"
	"slices"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works/internal/db"
)

// ItemOf reads one Item, without its fields, and its Work's summary inside
// a caller's read. An unknown id is storage.ErrNoRows.
func ItemOf(ctx context.Context, r storage.Reader[file], id ItemID) (Summary, Item, error) {
	q := db.New(r)
	it, err := q.GetItem(ctx, id)
	if err != nil {
		return Summary{}, Item{}, err
	}
	w, err := q.GetWork(ctx, it.WorkID)
	if err != nil {
		return Summary{}, Item{}, err
	}
	return Summary(w), bare(it), nil
}

// ItemsOf reads each of ws' Items, without their fields, inside a
// caller's read: per Work, live ones first, each by position. A Content
// Type reads it to answer for every Item of many Works in one call.
func ItemsOf(ctx context.Context, r storage.Reader[file], ws Among[WorkID]) (map[WorkID][]Item, error) {
	q := db.New(r)
	rows, err := Read(ws, func() ([]db.WorksItem, error) { return q.ListEveryItem(ctx) }, func(ids []WorkID) ([]db.WorksItem, error) {
		return q.ListItemsOfWorks(ctx, ids)
	})
	if err != nil {
		return nil, err
	}
	of := make(map[WorkID][]db.WorksItem, len(ws.IDs()))
	for _, it := range rows {
		of[it.WorkID] = append(of[it.WorkID], it)
	}
	out := make(map[WorkID][]Item, len(of))
	for id, items := range of {
		slices.SortFunc(items, siblings)
		out[id] = make([]Item, len(items))
		for i, it := range items {
			out[id][i] = bare(it)
		}
	}
	return out, nil
}

// Summaries reads every Work's summary inside a caller's read, by sort
// label.
func Summaries(ctx context.Context, r storage.Reader[file]) ([]Summary, error) {
	rows, err := db.New(r).ListWorks(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Summary, len(rows))
	for i, row := range rows {
		out[i] = Summary(row)
	}
	return out, nil
}

// Held is the Content Types holding a Work, inside a caller's read.
func Held(ctx context.Context, r storage.Reader[file]) ([]TypeID, error) {
	return db.New(r).HeldTypes(ctx)
}

func bare(it db.WorksItem) Item {
	return Item{ID: it.ID, LevelID: it.LevelID, Key: it.Key, Label: it.Label, Position: int(it.Position), WithdrawnAt: it.WithdrawnAt, Monitored: it.Monitored}
}

// SummaryOf reads Work id's summary inside a caller's read, ErrNoWork
// when no Work has it.
func SummaryOf(ctx context.Context, r storage.Reader[file], id WorkID) (Summary, error) {
	w, err := db.New(r).GetWork(ctx, id)
	if errors.Is(err, storage.ErrNoRows) {
		return Summary{}, ErrNoWork
	}
	return Summary(w), err
}
