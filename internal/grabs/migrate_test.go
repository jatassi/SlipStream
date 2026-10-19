package grabs_test

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

// TestTheFetchNodeIsTheHomeNodeOfEachFetchedGrab migrates a library whose
// Grabs predate fetch_node_id: a Grab with a .torrent was fetched through
// its node_id, and one still fetching has no fetch Node yet.
func TestTheFetchNodeIsTheHomeNodeOfEachFetchedGrab(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	open := func(g storage.Schema[storage.Library]) *storage.DB[storage.Library] {
		t.Helper()
		schemas := []storage.Schema[storage.Library]{secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(),
			roots.Schema(), indexers.Schema(), downloadclients.Schema(), g, files.Schema()}
		lib, err := storage.OpenLibrary(ctx, path, schemas, nil, storage.DefaultTimings(), slog.New(slog.DiscardHandler))
		if err != nil {
			t.Fatal(err)
		}
		return lib
	}
	old := open(grabs.Schema().Through(3))
	fx := contenttypetest.New()
	w, err := works.Open(ctx, works.Config{
		Library: old, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx},
		Changed: discard[works.WorkChanged]{}, TypesChanged: discard[works.ContentTypesChanged]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	fx.Put("w1", contenttypetest.Record{Tree: works.Fetched{Label: "The Attic", Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}}},
	}}})
	if _, _, err := w.Add(ctx, identity.Admin{}, contenttypetest.ID, "w1", works.Adding{Monitor: true}); err != nil {
		t.Fatal(err)
	}
	home, indexer, fetched, fetching := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	err = old.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'home', 0)`, home); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO indexers_indexer VALUES (?, 'Fixture Site', 'torznab', 'https://site.test/api', NULL, ?, '{}', 0)`, indexer, home); err != nil {
			return err
		}
		const grab = `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, indexer_id, node_id, release_title, release_size,
			release_link, stage, info_hash, metainfo, created_at, updated_at)
		SELECT ?, work_id, id, 'any', 'search', ?, ?, 'The.Attic.1080p.WEB', 4096, 'https://site.test/dl/1', ?, ?, ?, 0, 0 FROM works_item`
		if _, err := tx.ExecContext(ctx, grab, fetched, indexer, home, "downloading", []byte("01234567890123456789"), []byte{0}); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, grab, fetching, indexer, home, "fetching", nil, nil)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	lib := open(grabs.Schema())
	t.Cleanup(func() { _ = lib.Close() })
	for id, want := range map[storage.UUID]*storage.UUID{fetched: &home, fetching: nil} {
		var got *storage.UUID
		if err := lib.QueryRowContext(ctx, "SELECT fetch_node_id FROM grabs_grab WHERE id = ?", id).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if (got == nil) != (want == nil) || got != nil && *got != *want {
			t.Fatalf("fetch_node_id of %s = %v, want %v", id, got, want)
		}
	}
}
