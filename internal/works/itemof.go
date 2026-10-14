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
	return Summary(w), Item{ID: it.ID, LevelID: it.LevelID, Key: it.Key, Label: it.Label, Position: int(it.Position), WithdrawnAt: it.WithdrawnAt, Monitored: it.Monitored}, nil
}
