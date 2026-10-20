package works

import (
	"context"
	"errors"

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

// ItemsOf reads each of works' Items, without their fields, inside a
// caller's read: per Work, live ones first, each by position. A Content
// Type reads it to answer for every Item of many Works in one call.
func ItemsOf(ctx context.Context, r storage.Reader[file], works []WorkID) (map[WorkID][]Item, error) {
	rows, err := db.New(r).ListItemsOfWorks(ctx, works)
	if err != nil {
		return nil, err
	}
	out := make(map[WorkID][]Item, len(works))
	for _, it := range rows {
		out[it.WorkID] = append(out[it.WorkID], bare(it))
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
