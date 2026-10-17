package works

import (
	"context"

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

// ItemsOf reads work's Items, without their fields, inside a caller's
// read: live ones first, each by position. A Content Type reads it to
// answer for every Item of a Work in one call.
func ItemsOf(ctx context.Context, r storage.Reader[file], work WorkID) ([]Item, error) {
	rows, err := db.New(r).ListItems(ctx, work)
	if err != nil {
		return nil, err
	}
	items := make([]Item, len(rows))
	for i, it := range rows {
		items[i] = bare(it)
	}
	return items, nil
}

func bare(it db.WorksItem) Item {
	return Item{ID: it.ID, LevelID: it.LevelID, Key: it.Key, Label: it.Label, Position: int(it.Position), WithdrawnAt: it.WithdrawnAt, Monitored: it.Monitored}
}
