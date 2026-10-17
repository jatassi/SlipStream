package shellapi_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/shellapi"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

type pipeline struct{ g *grabs.Grabs }

func (p pipeline) Grabs() (*grabs.Grabs, bool) { return p.g, true }

// seed inserts one Grab in stage, with the columns the stage's CHECKs ask
// for, against the one Item in the library.
const seed = `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, indexer_id, node_id,
	release_title, release_size, release_link, stage, info_hash, metainfo, path, failure_code, created_at, updated_at)
SELECT ?, work_id, id, 'standard', 'search', ?, ?, 'The.Attic.1080p.WEB', 4096, 'https://site.test/dl/1', ?,
	CASE WHEN ? = 'fetching' THEN NULL ELSE randomblob(20) END,
	CASE WHEN ? = 'fetching' THEN NULL ELSE x'00' END,
	CASE WHEN ? IN ('transferring', 'importing', 'imported') THEN 'The Attic.mkv' END,
	CASE WHEN ? = 'failed' THEN 'download_failed' END,
	0, 0
FROM works_item`

func TestActivityCountsTheGrabsNeitherImportedNorFailed(t *testing.T) {
	ctx := t.Context()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(),
		roots.Schema(), indexers.Schema(), grabs.Schema(), files.Schema())
	fx := contenttypetest.New()
	w, err := works.Open(ctx, works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx},
		Changed: discard[works.WorkChanged]{}, TypesChanged: discard[works.ContentTypesChanged]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	fx.Put("w1", contenttypetest.Record{Tree: works.Fetched{Label: "The Attic", Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}}},
	}}})
	var admin identity.Admin
	if _, _, err := w.Add(ctx, admin, contenttypetest.ID, "w1", true); err != nil {
		t.Fatal(err)
	}
	s := ownertest.Serve(t, lib, func(r *api.Router) { shellapi.Routes(r, pipeline{grabs.New(grabs.Config{Library: lib})}) })

	s.Expect(t, http.MethodGet, "/api/v1/counts", "", http.StatusOK, `{"activity":0}`)
	home, indexer := storage.NewUUID(), storage.NewUUID()
	err = lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO nodes_node VALUES (?, randomblob(32), 'home', 0)`, home); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO indexers_indexer VALUES (?, 'Fixture Site', 'torznab', 'https://site.test/api', NULL, ?, '{}', 0)`, indexer, home); err != nil {
			return err
		}
		for _, stage := range []string{"fetching", "fetching", "downloading", "probing", "transferring", "importing", "imported", "imported", "failed"} {
			if _, err := tx.ExecContext(ctx, seed, storage.NewUUID(), indexer, home, stage, stage, stage, stage, stage); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	s.Expect(t, http.MethodGet, "/api/v1/counts", "", http.StatusOK, `{"activity":6}`)
}
