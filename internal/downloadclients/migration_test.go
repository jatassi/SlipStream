package downloadclients_test

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"

	dc "github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// TestAReleaseFaultsRebuildKeepsEveryTorrentColumn migrates a Torrent with
// seeding totals and a Retry through 00007's rebuild, then removes it with
// each new reason.
func TestAReleaseFaultsRebuildKeepsEveryTorrentColumn(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	openWith := func(clients storage.Schema[storage.Library]) library {
		t.Helper()
		lib, err := storage.OpenLibrary(ctx, path, []storage.Schema[storage.Library]{
			secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), health.Schema(), roots.Schema(), clients, indexers.Schema(),
		}, nil, storage.DefaultTimings(), slog.New(slog.DiscardHandler))
		if err != nil {
			t.Fatal(err)
		}
		return lib
	}
	old := openWith(dc.Schema().Through(6))
	node, torrent := addNode(t, old, "Den"), storage.NewUUID()
	write(t, old, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO downloadclients_torrent (id, node_id, info_hash, metainfo, name, size, private, trackers,
			content_type, wanted, added_order, added_at, uploaded, announced_seconds, retry, fault, fault_at)
			VALUES (?, ?, randomblob(20), x'00', 'Heat.1995.1080p.BluRay', 1024, 1, '[]', 'fixture', 'started', 1, 0, 512, 3600,
			'{"at":1}', 'unregistered', 5)`, torrent, node)
		return err
	})
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	lib := openWith(dc.Schema())
	t.Cleanup(func() { _ = lib.Close() })
	var got string
	err := lib.QueryRowContext(ctx, `SELECT name || ' ' || uploaded || ' ' || announced_seconds || ' ' || retry || ' ' || fault
		FROM downloadclients_torrent WHERE id = ?`, torrent).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if want := `Heat.1995.1080p.BluRay 512 3600 {"at":1} unregistered`; got != want {
		t.Fatalf("after the migration the Torrent reads %q, want %q", got, want)
	}
	for _, reason := range []dc.RemoveReason{dc.ReleaseFaulted, dc.Canceled} {
		write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, "UPDATE downloadclients_torrent SET removed_at = 9, removed_reason = ? WHERE id = ?", reason, torrent)
			return err
		})
	}
	err = lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, "UPDATE downloadclients_torrent SET removed_reason = 'stalled' WHERE id = ?", torrent)
		return err
	})
	if err == nil {
		t.Fatal("the removal reason stalled was stored")
	}
}

// TestTheAdoptedRebuildKeepsEveryTorrentColumn migrates a removed Torrent
// with a client error count through 00010's rebuild: it keeps every
// column and reads as grabbed, and only an adopted Torrent may lack a
// Content Type.
func TestTheAdoptedRebuildKeepsEveryTorrentColumn(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	openWith := func(clients storage.Schema[storage.Library]) library {
		t.Helper()
		lib, err := storage.OpenLibrary(ctx, path, []storage.Schema[storage.Library]{
			secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), health.Schema(), roots.Schema(), clients, indexers.Schema(),
		}, nil, storage.DefaultTimings(), slog.New(slog.DiscardHandler))
		if err != nil {
			t.Fatal(err)
		}
		return lib
	}
	old := openWith(dc.Schema().Through(9))
	node, torrent := addNode(t, old, "Den"), storage.NewUUID()
	write(t, old, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO downloadclients_torrent (id, node_id, info_hash, metainfo, name, size, private, trackers,
			content_type, wanted, added_order, added_at, removed_at, removed_reason, removed_by_name, client_error_ms, client_error_at)
			VALUES (?, ?, randomblob(20), x'00', 'Heat.1995.1080p.BluRay', 1024, 1, '[]', 'fixture', 'started', 1, 0, 7, 'admin', 'Ada', 4000, 6)`,
			torrent, node)
		return err
	})
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	lib := openWith(dc.Schema())
	t.Cleanup(func() { _ = lib.Close() })
	var got string
	err := lib.QueryRowContext(ctx, `SELECT name || ' ' || content_type || ' ' || removed_by_name || ' ' || client_error_ms || ' ' ||
		client_error_at || ' ' || adopted || ' ' || coalesce(withheld_at, 'none') FROM downloadclients_torrent WHERE id = ?`, torrent).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if want := "Heat.1995.1080p.BluRay fixture Ada 4000 6 0 none"; got != want {
		t.Fatalf("after the migration the Torrent reads %q, want %q", got, want)
	}
	err = lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, "UPDATE downloadclients_torrent SET content_type = NULL WHERE id = ?", torrent)
		return err
	})
	if err == nil {
		t.Fatal("a grabbed Torrent lost its Content Type")
	}
	write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, "UPDATE downloadclients_torrent SET adopted = 1, content_type = NULL WHERE id = ?", torrent)
		return err
	})
}
