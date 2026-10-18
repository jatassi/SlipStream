package targets

import (
	"context"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets/internal/db"
	"github.com/jatassi/slipstream/internal/works"
)

// See records in tx that an Indexer listed a release of cut covering
// item, keyed by its info hash in hex, else its title. Seeing it again
// changes nothing.
func See(ctx context.Context, tx *storage.Tx[file], item works.ItemID, cut release.CutKey, key string) error {
	return db.New(tx).InsertSighting(ctx, db.InsertSightingParams{ItemID: item, Cut: cut, Release: key, SeenAt: storage.Now()})
}

// SeenIn reads, in r, the cuts sighted for each of items. A file of a
// cut counts as seen too, which files adds.
func SeenIn(ctx context.Context, r storage.Reader[file], items []works.ItemID) (map[works.ItemID][]release.CutKey, error) {
	rows, err := db.New(r).ListSightingsOf(ctx, items)
	if err != nil {
		return nil, err
	}
	out := map[works.ItemID][]release.CutKey{}
	for _, row := range rows {
		out[row.ItemID] = append(out[row.ItemID], row.Cut)
	}
	return out, nil
}
