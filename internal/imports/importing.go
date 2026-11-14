package imports

import (
	"context"

	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Importing answers, for each of items an unfinished Import of a Work
// holds, that Import: one with a file for the Item that is pending,
// waiting, sending or placing. S48 (Moves) checks it at cutover.
func Importing(ctx context.Context, r storage.Reader[file], items []works.ItemID) (map[works.ItemID]ImportID, error) {
	rows, err := db.New(r).ListImporting(ctx, items)
	if err != nil {
		return nil, err
	}
	out := make(map[works.ItemID]ImportID, len(rows))
	for _, row := range rows {
		out[row.ItemID] = row.ImportID
	}
	return out, nil
}

// holders answers, for each of items another Import holds against imp, that
// Import: a file of it sending or placing, or pending in an Import created
// before imp. Two Imports deciding in either order can't wait on each other.
func holders(ctx context.Context, r storage.Reader[file], items []works.ItemID, imp Import) (map[works.ItemID]ImportID, error) {
	rows, err := db.New(r).ListBusy(ctx, db.ListBusyParams{ItemIds: items, Except: imp.ID, Created: imp.Created})
	if err != nil {
		return nil, err
	}
	out := make(map[works.ItemID]ImportID, len(rows))
	for _, row := range rows {
		out[row.ItemID] = row.ImportID
	}
	return out, nil
}
