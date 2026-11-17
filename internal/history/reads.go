package history

import (
	"context"

	"github.com/jatassi/slipstream/internal/history/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// Subjects is each of ids' stored facts, through r: notifications' own
// Read or Write. An id that is gone has none.
func Subjects(ctx context.Context, r storage.Reader[storage.Library], ids []EventID) (map[EventID][]byte, error) {
	rows, err := db.New(r).ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[EventID][]byte, len(rows))
	for _, row := range rows {
		out[row.ID] = []byte(row.Subject)
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
