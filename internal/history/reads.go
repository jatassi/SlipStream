package history

import (
	"context"

	"github.com/jatassi/slipstream/internal/history/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Stored is an event's facts and the one Work it is about, zero for none
// or many.
type Stored struct {
	Subject []byte
	Work    works.WorkID
}

// Events is each of ids' stored facts and Work, through r: notifications'
// own Read or Write. An id that is gone has none.
func Events(ctx context.Context, r storage.Reader[storage.Library], ids []EventID) (map[EventID]Stored, error) {
	rows, err := db.New(r).ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[EventID]Stored, len(rows))
	for _, row := range rows {
		out[row.ID] = Stored{Subject: []byte(row.Subject), Work: row.WorkID}
	}
	return out, nil
}

// Items is each of ids' Health items, in key order, through r.
func Items(ctx context.Context, r storage.Reader[storage.Library], ids []EventID) (map[EventID][]Item, error) {
	rows, err := db.New(r).ListItemsOf(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := map[EventID][]Item{}
	for _, row := range rows {
		out[row.EventID] = append(out[row.EventID], itemOf(row))
	}
	return out, nil
}
