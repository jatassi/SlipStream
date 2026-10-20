package grabs_test

import (
	"context"
	"log/slog"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/works"
)

type library = *storage.DB[storage.Library]

// openAt opens the library at path with grabs migrated by g, after every
// set it stands on.
func openAt(t *testing.T, path string, g storage.Schema[storage.Library]) library {
	t.Helper()
	schemas := []storage.Schema[storage.Library]{secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(),
		roots.Schema(), indexers.Schema(), downloadclients.Schema(), g, files.Schema()}
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
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx},
		Changed: discard[works.WorkChanged]{}, TypesChanged: discard[works.ContentTypesChanged]{},
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

func (noDownloads) Touch(...role.NodeID)   {}
func (noDownloads) Moved() <-chan struct{} { return nil }
func (noDownloads) Sample(downloadclients.ID) (downloadclients.Sample, bool) {
	return downloadclients.Sample{}, false
}

func grabOf(t *testing.T, lib library, id storage.UUID) grabs.View {
	t.Helper()
	v, err := grabs.New(grabs.Config{Library: lib, Downloads: noDownloads{}}).Get(t.Context(), identity.Admin{}, grabs.GrabID{UUID: id})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func intentOf(t *testing.T, lib library, node role.NodeID) []bencode.InfoHash {
	t.Helper()
	var out []bencode.InfoHash
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		in, err := downloadclients.Intent(ctx, r, node)
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

// TestMigrationTurnsGrabTorrentsIntoTorrents migrates Grabs that held
// their torrent: two Grabs of one hash on seedbox share one live Torrent,
// a Grab whose seeding a write-off dropped becomes a parted Torrent, the
// same hash on home is its own Torrent with its completion, and a Grab
// still fetching has none.
func TestMigrationTurnsGrabTorrentsIntoTorrents(t *testing.T) {
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	old := openAt(t, path, grabs.Schema().Through(4))
	addAttic(t, old)
	box, home, indexer := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	first, second, dropped, completed, fetching := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	shared, other := []byte("aaaaaaaaaaaaaaaaaaaa"), []byte("bbbbbbbbbbbbbbbbbbbb")
	const grab = `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, indexer_id, node_id, release_title, release_size,
		release_link, stage, info_hash, metainfo, download_files, failed_stage, failure_code, failure_message, seed_dropped_at,
		created_at, updated_at)
	SELECT ?, work_id, id, 'any', 'search', ?, ?, ?, 4096, 'https://site.test/dl/1', ?, ?, ?, ?, ?, ?, ?, ?, ?, ? FROM works_item`
	run(t, old,
		sql(`INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'seedbox', 0), (?, randomblob(32), 'home', 0)`, box, home),
		sql(`INSERT INTO indexers_indexer VALUES (?, 'Fixture Site', 'torznab', 'https://site.test/api', NULL, ?, '{}', 0)`, indexer, home),
		sql(grab, first, indexer, box, "The.Attic.1080p.WEB", "downloading", shared, []byte("first"), nil, nil, nil, nil, nil, 10, 10),
		sql(grab, second, indexer, box, "The.Attic.1080p.WEB.REPACK", "failed", shared, []byte("second"), nil, "probing", "probe_failed", "no", nil, 20, 20),
		sql(grab, dropped, indexer, box, "The.Attic.720p.WEB", "failed", other, []byte("dropped"), nil, "downloading", "node_written_off", "gone", 50, 5, 5),
		sql(grab, completed, indexer, home, "The.Attic.1080p.WEB", "downloading", shared, []byte("home"), `[{"Index":0,"Path":"Attic.mkv","Size":4096}]`, nil, nil, nil, nil, 30, 40),
		sql(grab, fetching, indexer, home, "The.Attic.2160p.WEB", "fetching", nil, nil, nil, nil, nil, nil, nil, 60, 60),
	)
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	lib := openAt(t, path, grabs.Schema())
	t.Cleanup(func() { _ = lib.Close() })
	for _, c := range []struct {
		grab, torrent storage.UUID
		outcome       downloadclients.Outcome
		files         []engine.File
	}{
		{first, first, downloadclients.Pending{}, nil},
		{second, first, downloadclients.Pending{}, nil},
		{dropped, dropped, downloadclients.Gone{Reason: downloadclients.NodeWrittenOff}, nil},
		{completed, completed, downloadclients.Completed{}, []engine.File{{Index: 0, Path: "Attic.mkv", Size: 4096}}},
	} {
		g := grabOf(t, lib, c.grab)
		if g.Torrent == nil || g.Torrent.ID.UUID != c.torrent || g.Torrent.Outcome != c.outcome || !slices.Equal(g.Torrent.Files, c.files) {
			t.Errorf("Grab %s reads its Torrent as %+v, want %s %#v with files %v", c.grab, g.Torrent, c.torrent, c.outcome, c.files)
		}
	}
	if g := grabOf(t, lib, fetching); g.Torrent != nil {
		t.Errorf("the fetching Grab reads the Torrent %+v, want none", g.Torrent)
	}
	if got, want := intentOf(t, lib, role.NodeID{UUID: box}), []bencode.InfoHash{bencode.InfoHash(shared)}; !slices.Equal(got, want) {
		t.Errorf("seedbox's intent is %v, want only the shared hash: the dropped Torrent is parted", got)
	}
	if got, want := intentOf(t, lib, role.NodeID{UUID: home}), []bencode.InfoHash{bencode.InfoHash(shared)}; !slices.Equal(got, want) {
		t.Errorf("home's intent is %v, want its own Torrent of the shared hash", got)
	}
	var name, trackers, contentType string
	var private, order int64
	var metainfo []byte
	err := lib.QueryRowContext(t.Context(), `SELECT name, private, trackers, content_type, added_order, metainfo FROM downloadclients_torrent WHERE id = ?`, first).
		Scan(&name, &private, &trackers, &contentType, &order, &metainfo)
	if err != nil {
		t.Fatal(err)
	}
	if name != "The.Attic.1080p.WEB" || private != 1 || trackers != "[]" || contentType != string(contenttypetest.ID) || order != 2 || string(metainfo) != "first" {
		t.Errorf("seedbox's shared Torrent is %q private=%d trackers=%s %s order %d metainfo %q, want the first Grab's release, private, no trackers, fixture, order 2",
			name, private, trackers, contentType, order, metainfo)
	}
}

// TestDeletingAWorkKeepsItsTorrentSeeding deletes a Work whose Grab
// downloaded a Torrent: the Grab goes with the Work, and the Torrent stays
// in its Node's intent.
func TestDeletingAWorkKeepsItsTorrentSeeding(t *testing.T) {
	lib := openAt(t, filepath.Join(t.TempDir(), storage.LibraryFile), grabs.Schema())
	t.Cleanup(func() { _ = lib.Close() })
	work := addAttic(t, lib)
	box, indexer, grab, torrent := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	hash := []byte("aaaaaaaaaaaaaaaaaaaa")
	run(t, lib,
		sql(`INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'seedbox', 0)`, box),
		sql(`INSERT INTO indexers_indexer VALUES (?, 'Fixture Site', 'torznab', 'https://site.test/api', NULL, ?, '{}', 0)`, indexer, box),
		sql(`INSERT INTO downloadclients_torrent (id, node_id, info_hash, metainfo, name, size, private, trackers, content_type, wanted,
			added_order, added_at) VALUES (?, ?, ?, x'00', 'The.Attic.1080p.WEB', 4096, 1, '[]', 'fixture', 'started', 1, 0)`, torrent, box, hash),
		sql(`INSERT INTO grabs_grab (id, work_id, item_id, edition, source, indexer_id, node_id, release_title, release_size, release_link,
			stage, torrent_id, created_at, updated_at)
			SELECT ?, work_id, id, 'any', 'search', ?, ?, 'The.Attic.1080p.WEB', 4096, 'https://site.test/dl/1', 'downloading', ?, 0, 0
			FROM works_item`, grab, indexer, box, torrent),
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
