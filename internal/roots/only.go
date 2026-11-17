package roots

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/roots/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Only keeps the ids among ws whose root is root, in ws' order, reading
// works.Batch ids at a time.
func Only(ctx context.Context, r storage.Reader[file], root ID, ws []works.WorkID) ([]works.WorkID, error) {
	q := db.New(r)
	in := map[works.WorkID]bool{}
	for chunk := range slices.Chunk(ws, works.Batch) {
		ids, err := q.ListWorksOfRootAmong(ctx, db.ListWorksOfRootAmongParams{RootID: root, WorkIds: chunk})
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			in[id] = true
		}
	}
	return slices.DeleteFunc(slices.Clone(ws), func(id works.WorkID) bool { return !in[id] }), nil
}
