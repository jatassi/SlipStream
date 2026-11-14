package mediaservers_test

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

type library = *storage.DB[storage.Library]

func open(t *testing.T) library {
	t.Helper()
	lib, err := storage.OpenLibrary(t.Context(), filepath.Join(t.TempDir(), storage.LibraryFile),
		[]storage.Schema[storage.Library]{secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), roots.Schema(), mediaservers.Schema()},
		nil, storage.DefaultTimings(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lib.Close() })
	return lib
}

func exec(t *testing.T, lib library, sql string, args ...any) error {
	t.Helper()
	return lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, sql, args...)
		return err
	})
}

func must(t *testing.T, lib library, sql string, args ...any) {
	t.Helper()
	if err := exec(t, lib, sql, args...); err != nil {
		t.Fatal(err)
	}
}

func count(t *testing.T, lib library, table string) int {
	t.Helper()
	var n int
	if err := lib.QueryRowContext(t.Context(), "SELECT count(*) FROM "+table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

type fixture struct {
	lib          library
	node, root   storage.UUID
	server, file storage.UUID
}

// seed adds a Node, a root on it and a Jellyfin server it reaches, with a
// mapping, a due change, an index entry, a play and a credential fetch.
func seed(t *testing.T) fixture {
	f := fixture{lib: open(t), node: storage.NewUUID(), root: storage.NewUUID(), server: storage.NewUUID(), file: storage.NewUUID()}
	must(t, f.lib, "INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'Den', 0)", f.node)
	must(t, f.lib, "INSERT INTO roots_root (id, content_type, node_id, path, is_default) VALUES (?, 'fixture', ?, '/media/movies', 1)", f.root, f.node)
	must(t, f.lib, `INSERT INTO mediaservers_server (id, kind, name, address, identity, version, monitor_delay_ms,
		reaching_node_id, device_id, secrets_sealed, created_at, updated_at)
		VALUES (?, 'jellyfin', 'Den Jellyfin', 'http://den:8096', 'f4c1', '12.2.0', 60000, ?, 'd1', x'00', 1, 1)`, f.server, f.node)
	must(t, f.lib, `INSERT INTO mediaservers_mapping (server_id, root_id, shown, server_folder, library_key, library_name,
		library_folder, confirmed_at, confirmed_place) VALUES (?, ?, 1, '/media/movies', 'a', 'Movies', '/media/movies', 1, 1)`, f.server, f.root)
	f.change(t, "added", "Heat/Heat.mkv", nil, "due", "none")
	must(t, f.lib, `INSERT INTO mediaservers_item (server_id, path_key, server_path, library_key, file_id, item_key, top_key, seen_at)
		VALUES (?, '/media/movies/Heat/Heat.mkv', '/media/movies/Heat/Heat.mkv', 'a', ?, 'i1', 'i1', 1)`, f.server, f.file)
	f.play(t, f.server, storage.NewUUID())
	must(t, f.lib, `INSERT INTO mediaservers_credential_fetch (server_id, node_id, settings_version, first_fetched_at, fetched_at)
		VALUES (?, ?, 1, 1, 1)`, f.server, f.node)
	return f
}

func (f fixture) insertChange(t *testing.T, kind, path string, from any, state, step string) error {
	t.Helper()
	var stepAt any
	if step != "none" {
		stepAt = 1
	}
	return exec(t, f.lib, `INSERT INTO mediaservers_change (id, server_id, kind, root_id, path, from_path, file_id, state, step, step_at,
		next_attempt_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, 1)`,
		storage.NewUUID(), f.server, kind, f.root, path, from, f.file, state, step, stepAt)
}

func (f fixture) change(t *testing.T, kind, path string, from any, state, step string) {
	t.Helper()
	if err := f.insertChange(t, kind, path, from, state, step); err != nil {
		t.Fatal(err)
	}
}

func (f fixture) play(t *testing.T, server, id storage.UUID) {
	t.Helper()
	must(t, f.lib, `INSERT INTO mediaservers_play (server_id, id, session_key, server_path, user_name, player, product, decision,
		started_at, last_seen_at) VALUES (?, ?, 's1', '/media/movies/Heat/Heat.mkv', 'Sam', 'Living room TV', 'Jellyfin Web', 'transcode', 1, 1)`,
		server, id)
}

var forgotten = []string{"mediaservers_mapping", "mediaservers_change", "mediaservers_item", "mediaservers_play", "mediaservers_credential_fetch"}

func TestRemovingAServerForgetsItsChangesIndexPlaysAndHolders(t *testing.T) {
	f := seed(t)
	must(t, f.lib, "DELETE FROM mediaservers_server WHERE id = ?", f.server)
	for _, table := range forgotten {
		if n := count(t, f.lib, table); n != 0 {
			t.Errorf("%s keeps %d rows after the server was removed", table, n)
		}
	}
}

func TestDeletingARootDeletesItsMappingsAndChanges(t *testing.T) {
	f := seed(t)
	must(t, f.lib, "DELETE FROM roots_root WHERE id = ?", f.root)
	for table, want := range map[string]int{"mediaservers_mapping": 0, "mediaservers_change": 0, "mediaservers_server": 1, "mediaservers_play": 1} {
		if n := count(t, f.lib, table); n != want {
			t.Errorf("%s holds %d rows after the root was deleted, want %d", table, n, want)
		}
	}
}

func TestOneOpenRowPerPathAndOneOpenCarryPerFile(t *testing.T) {
	f := seed(t)
	if err := f.insertChange(t, "removed", "Heat/Heat.mkv", nil, "due", "none"); err == nil {
		t.Error("a second open row at a path was stored")
	}
	f.change(t, "moved", "Heat (1995)/Heat.mkv", "Heat/Heat.mkv", "prepared", "snapshotted")
	if err := f.insertChange(t, "moved", "Ran/Ran.mkv", "Heat (1995)/Heat.mkv", "due", "none"); err == nil {
		t.Error("a second open carry of one file was stored")
	}
}

func TestAPlayIdNamesAPlayOnlyOnItsServer(t *testing.T) {
	f := seed(t)
	other := storage.NewUUID()
	must(t, f.lib, `INSERT INTO mediaservers_server (id, kind, name, address, identity, version, reaching_node_id, secrets_sealed,
		created_at, updated_at) VALUES (?, 'plex', 'Living room Plex', 'http://den:32400', 'm1', '1.43.5', ?, x'00', 1, 1)`, other, f.node)
	id := storage.NewUUID()
	f.play(t, f.server, id)
	f.play(t, other, id)
	if err := exec(t, f.lib, `INSERT INTO mediaservers_play (server_id, id, session_key, server_path, user_name, player, product, decision,
		started_at, last_seen_at) VALUES (?, ?, 's2', '/x', 'Sam', 'TV', 'Plex', 'copy', 1, 1)`, f.server, id); err == nil {
		t.Error("a play id was started twice on one server")
	}
}

func TestTheSchemaRefusesAChangeNoFoldMakes(t *testing.T) {
	f := seed(t)
	for _, c := range []struct {
		name              string
		kind, path, state string
		from              any
		step              string
	}{
		{"a carry back to its origin", "moved", "Heat/Heat.mkv", "due", "Heat/Heat.mkv", "none"},
		{"a moved change with no origin", "moved", "Ran/Ran.mkv", "due", nil, "none"},
		{"a prepared refresh", "added", "Ran/Ran.mkv", "prepared", nil, "none"},
		{"a carry of a removal", "removed", "Ran/Ran.mkv", "due", "Old/Ran.mkv", "none"},
		{"a sent change with no result", "added", "Ran/Ran.mkv", "sent", nil, "none"},
		{"an unknown kind", "renamed", "Ran/Ran.mkv", "due", nil, "none"},
		{"a library refresh with a path", "library", "Ran/Ran.mkv", "due", nil, "none"},
	} {
		if err := f.insertChange(t, c.kind, c.path, c.from, c.state, c.step); err == nil {
			t.Errorf("%s was stored", c.name)
		}
	}
}
