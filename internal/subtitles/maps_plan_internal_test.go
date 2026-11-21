package subtitles

import (
	"context"
	"testing"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// TestTheMapDrainReadsThroughItsIndexes pins the map drain's reads, which
// run on every pass, to their partial indexes.
func TestTheMapDrainReadsThroughItsIndexes(t *testing.T) {
	lib := storagetest.Library(t, identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), downloadclients.Schema(), grabs.Schema(), indexers.Schema(), files.Schema(), Schema(), naming.Schema())
	for _, s := range []struct {
		name    string
		indexes []string
		read    func(ctx context.Context, q *db.Queries) error
	}{
		{"NextMaps", []string{"subtitles_file_map_due"}, func(ctx context.Context, q *db.Queries) error {
			_, err := q.NextMaps(ctx, db.NextMapsParams{RootID: roots.ID{UUID: storage.NewUUID()}, N: 2})
			return err
		}},
		{"ListMapRoots", []string{"subtitles_file_map_due"}, func(ctx context.Context, q *db.Queries) error {
			_, err := q.ListMapRoots(ctx)
			return err
		}},
	} {
		wantPlans(t, lib, s.name, s.indexes, s.read)
	}
}
