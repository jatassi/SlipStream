package grabs_test

import (
	"context"
	"log/slog"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/magnet"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type library = *storage.DB[storage.Library]

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

// openAt opens the library at path with every set migrated in the
// order the Node runs them.
func openAt(t *testing.T, path string) library {
	t.Helper()
	return openSets(t, path, downloadclients.Schema(), grabs.Schema(), indexers.Schema(), files.Schema())
}

func openSets(t *testing.T, path string, last ...storage.Schema[storage.Library]) library {
	t.Helper()
	schemas := append([]storage.Schema[storage.Library]{secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(),
		health.Schema(), roots.Schema()}, last...)
	lib, err := storage.OpenLibrary(t.Context(), path, schemas, nil, storage.DefaultTimings(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	return lib
}

func run(t *testing.T, lib library, stmts ...func(ctx context.Context, tx *storage.Tx[storage.Library]) error) {
	t.Helper()
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, s := range stmts {
			if err := s(ctx, tx); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func sql(query string, args ...any) func(context.Context, *storage.Tx[storage.Library]) error {
	return func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	}
}

// addAttic adds the fixture Work "The Attic" with one Item.
func addAttic(t *testing.T, lib library) works.WorkID {
	t.Helper()
	fx := contenttypetest.New()
	w, err := works.Open(t.Context(), works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx}, Source: fx.Source(), Artwork: fx,
		Changed: discard[works.Changed]{}, TypesChanged: discard[works.ContentTypesChanged]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	fx.Put("w1", contenttypetest.Record{Tree: works.Fetched{Label: "The Attic", Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}}},
	}}})
	tree, _, err := w.Add(t.Context(), identity.Admin{}, contenttypetest.ID, "w1", works.Adding{Monitor: true})
	if err != nil {
		t.Fatal(err)
	}
	return tree.ID
}

type noDownloads struct{}

func (noDownloads) Touch(...role.NodeID)                  {}
func (noDownloads) Moved() <-chan struct{}                { return nil }
func (noDownloads) Retargeted(downloadclients.Retargeted) {}
func (noDownloads) Sample(downloadclients.ID) (downloadclients.Sample, bool) {
	return downloadclients.Sample{}, false
}

func (noDownloads) Touched(...downloadclients.Touched) {}

func (noDownloads) RemoveIn(context.Context, *storage.Tx[storage.Library], []downloadclients.ID, downloadclients.RemoveReason, bool) ([]downloadclients.Result, downloadclients.Touched, error) {
	panic("nothing removes a Torrent in this test")
}

func (noDownloads) Resolve(context.Context, role.NodeID, magnet.Link, time.Time) (downloadclients.Resolution, error) {
	panic("reading a Grab asks no Node for metadata")
}

func intentOf(t *testing.T, lib library, node role.NodeID) []bencode.InfoHash {
	t.Helper()
	var out []bencode.InfoHash
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		in, err := downloadclients.Derive{Keeps: downloadclients.NoKeeps}.Intent(ctx, r, node)
		for _, tor := range in.GetTorrents() {
			out = append(out, bencode.InfoHash(tor.GetInfoHash()))
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestDeletingAWorkKeepsItsTorrentSeeding deletes a Work whose Grab
// downloaded a Torrent: the Grab goes with the Work, and the Torrent stays
// in its Node's intent.
func TestDeletingAWorkKeepsItsTorrentSeeding(t *testing.T) {
	lib := openAt(t, filepath.Join(t.TempDir(), storage.LibraryFile))
	t.Cleanup(func() { _ = lib.Close() })
	work := addAttic(t, lib)
	box, grab, torrent := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	hash := []byte("aaaaaaaaaaaaaaaaaaaa")
	run(t, lib,
		sql(`INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'seedbox', 0)`, box),
		sql(`INSERT INTO downloadclients_torrent (id, node_id, info_hash, metainfo, name, size, private, trackers, content_type, wanted,
			added_order, added_at) VALUES (?, ?, ?, x'00', 'The.Attic.1080p.WEB', 4096, 1, '[]', 'fixture', 'started', 1, 0)`, torrent, box, hash),
		sql(`INSERT INTO grabs_grab (id, work_id, item_id, edition, source, node_id, release_title, release_size,
			stage, torrent_id, created_at, updated_at)
			SELECT ?, work_id, id, 'any', 'search', ?, 'The.Attic.1080p.WEB', 4096, 'downloading', ?, 0, 0
			FROM works_item`, grab, box, torrent),
	)
	run(t, lib, sql(`DELETE FROM works_work WHERE id = ?`, work))

	var left int64
	if err := lib.QueryRowContext(t.Context(), "SELECT count(*) FROM grabs_grab").Scan(&left); err != nil || left != 0 {
		t.Fatalf("%d Grabs (%v) outlive their Work, want none", left, err)
	}
	if got := intentOf(t, lib, role.NodeID{UUID: box}); !slices.Equal(got, []bencode.InfoHash{bencode.InfoHash(hash)}) {
		t.Errorf("seedbox's intent is %v after the Work's delete, want its Torrent still seeding", got)
	}
}
