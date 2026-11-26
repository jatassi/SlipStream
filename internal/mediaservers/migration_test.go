package mediaservers_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

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
	seedServer(t, f)
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

// seedServer adds f's Node, a root on it and a Jellyfin server.
func seedServer(t *testing.T, f fixture) {
	t.Helper()
	must(t, f.lib, "INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'Den', 0)", f.node)
	must(t, f.lib, "INSERT INTO roots_root (id, content_type, node_id, path, is_default) VALUES (?, 'fixture', ?, '/media/movies', 1)", f.root, f.node)
	must(t, f.lib, `INSERT INTO mediaservers_server (id, kind, name, address, identity, version, monitor_delay_ms,
		reaching_node_id, device_id, secrets_sealed, created_at, updated_at)
		VALUES (?, 'jellyfin', 'Den Jellyfin', 'http://den:8096', 'f4c1', '12.2.0', 60000, ?, 'd1', x'00', 1, 1)`, f.server, f.node)
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

const (
	plexServerID = "X'000000000000000000000000000000a1'"
	jellyfinID   = "X'000000000000000000000000000000a2'"
	samID        = "X'000000000000000000000000000000a3'"
)

// validMemberRows are one row each table takes; each case changes one
// column of one of them.
var validMemberRows = map[string]map[string]string{
	"plex access": {
		"id": "randomblob(16)", "server_id": plexServerID, "way": "'plex_tv'", "account_id": samID, "member_name": "'Sam'",
		"state": "'waiting'", "changed_at": "1", "offered_at": "1", "plex_id": "42", "plex_uuid": "'u42'",
	},
	"users access": {
		"id": "randomblob(16)", "server_id": jellyfinID, "way": "'users'", "account_id": samID, "member_name": "'Sam'",
		"state": "'active'", "changed_at": "1", "offered_at": "1", "server_user_id": "'f00d'", "username": "'sam'",
	},
	"server": {
		"id": "randomblob(16)", "kind": "'jellyfin'", "name": "'Attic Jellyfin'", "address": "'http://attic:8096'", "identity": "'a77c'",
		"version": "'12.2.0'", "monitor_delay_ms": "60000", "reaching_node_id": "(SELECT id FROM nodes_node)", "device_id": "'d2'",
		"secrets_sealed": "x'00'", "created_at": "1", "updated_at": "1", "member_address": "'https://attic.example.com'",
	},
	"member plex": {
		"account_id": samID, "client_identifier": "'c1'", "plex_id": "42", "plex_uuid": "'u42'", "username": "'sam'",
		"email": "'sam@plex.example'", "signed_in_at": "1", "token_sealed": "x'00'",
	},
	"signup link": { //nolint:gosec // token_hash is a random blob, not a credential
		"id": "randomblob(16)", "kind": "'signup'", "token_hash": "randomblob(32)", "email": "'kim@example.com'", "issued_at": "1",
		"expires_at": "2", "grants": `'{"mediaServers":[]}'`,
	},
}

var memberTables = map[string]string{
	"plex access": "mediaservers_member_access", "users access": "mediaservers_member_access", "server": "mediaservers_server",
	"member plex": "mediaservers_member_plex", "signup link": "identity_link",
}

func TestTheSchemaRefusesEveryAccessRowItsChecksForbid(t *testing.T) {
	f := seed(t)
	must(t, f.lib, `INSERT INTO mediaservers_server (id, kind, name, address, identity, version, reaching_node_id, secrets_sealed,
		created_at, updated_at) VALUES (`+plexServerID+`, 'plex', 'Living room Plex', 'http://den:32400', 'm1', '1.43.5', ?, x'00', 1, 1)`, f.node)
	must(t, f.lib, `INSERT INTO mediaservers_server (id, kind, name, address, identity, version, monitor_delay_ms, reaching_node_id,
		device_id, secrets_sealed, created_at, updated_at) VALUES (`+jellyfinID+`, 'jellyfin', 'Loft Jellyfin', 'http://loft:8096', 'b1',
		'12.2.0', 60000, ?, 'd3', x'00', 1, 1)`, f.node)
	must(t, f.lib, "INSERT INTO identity_account (id, email, display_name, admin, pin_hash, created_at) VALUES ("+samID+", 'sam@example.com', 'Sam', 0, '', 0)")
	for name, row := range validMemberRows {
		if err := tryRow(t, f.lib, memberTables[name], row); err != nil {
			t.Fatalf("the valid %s row is refused: %v", name, err)
		}
	}

	for _, c := range []struct {
		name, row string
		set       map[string]string
	}{
		{"a waiting users row", "users access", map[string]string{"state": "'waiting'"}},
		{"a shared users row", "users access", map[string]string{"state": "'shared'"}},
		{"a making Plex row", "plex access", map[string]string{"state": "'making'", "username": "'sam'"}},
		{"a removed Plex row", "plex access", map[string]string{"state": "'removed'"}},
		{"a waiting Plex row with no Plex account", "plex access", map[string]string{"plex_id": "NULL", "plex_uuid": "NULL"}},
		{"a Plex id with no uuid", "plex access", map[string]string{"plex_uuid": "NULL"}},
		{"a share with no Plex account", "plex access", map[string]string{"state": "'offered'", "plex_id": "NULL", "plex_uuid": "NULL", "shared_server_id": "'s1'"}},
		{"a Plex row with a server user", "plex access", map[string]string{"server_user_id": "'f00d'", "username": "'sam'"}},
		{"a users row in the Plex Home", "users access", map[string]string{"plex_home": "1"}},
		{"a users row shared by hand", "users access", map[string]string{"by_hand": "1"}},
		{"a users row with a Plex account", "users access", map[string]string{"plex_id": "42", "plex_uuid": "'u42'"}},
		{"a removed row with no user", "users access", map[string]string{"state": "'removed'", "server_user_id": "NULL", "username": "NULL"}},
		{"a making row with no name", "users access", map[string]string{"state": "'making'", "server_user_id": "NULL", "username": "NULL"}},
		{"a user id with no name", "users access", map[string]string{"username": "NULL"}},
		{"an active row with no Account", "users access", map[string]string{"account_id": "NULL"}},
		{"an offered row with no Account", "plex access", map[string]string{"account_id": "NULL", "state": "'offered'"}},
		{"an unknown way", "users access", map[string]string{"way": "'kodi'"}},
		{"an unknown state", "users access", map[string]string{"state": "'paused'"}},
		{"an unknown problem", "users access", map[string]string{"problem_code": "'oops'", "problem_detail": "'x'"}},
		{"a problem with no detail", "users access", map[string]string{"problem_code": "'failed'"}},
		{"a detail with no problem", "users access", map[string]string{"problem_detail": "'x'"}},
		{"an Address for Members on Plex", "server", map[string]string{"kind": "'plex'", "monitor_delay_ms": "NULL", "device_id": "NULL"}},
		{"an empty Address for Members", "server", map[string]string{"member_address": "''"}},
		{"a Member token with no Plex account", "member plex", map[string]string{"plex_id": "NULL", "plex_uuid": "NULL", "username": "NULL", "email": "NULL", "signed_in_at": "NULL"}},
		{"a Plex sign-in with no email", "member plex", map[string]string{"email": "NULL"}},
		{"grants on a reset link", "signup link", map[string]string{"kind": "'reset'", "account_id": samID}},
		{"grants that aren't JSON", "signup link", map[string]string{"grants": "'mediaServers'"}},
		{"grants that aren't an object", "signup link", map[string]string{"grants": "'[]'"}},
	} {
		row := maps.Clone(validMemberRows[c.row])
		maps.Copy(row, c.set)
		err := tryRow(t, f.lib, memberTables[c.row], row)
		if err == nil || !strings.Contains(err.Error(), "CHECK constraint failed") {
			t.Errorf("%s: err = %v, want a CHECK refusal", c.name, err)
		}
	}
	if err := exec(t, f.lib, "UPDATE mediaservers_server SET member_address = 'https://den.example.com' WHERE id = "+plexServerID); err == nil {
		t.Error("a Plex server stored an Address for Members")
	}
}

var errRolledBack = errors.New("rolled back")

// tryRow writes row into table in a Write it then rolls back.
func tryRow(t *testing.T, lib library, table string, row map[string]string) error {
	cols := slices.Sorted(maps.Keys(row))
	vals := make([]string, len(cols))
	for i, c := range cols {
		vals[i] = row[c]
	}
	stmt := "INSERT INTO " + table + " (" + strings.Join(cols, ", ") + ") VALUES (" + strings.Join(vals, ", ") + ")"
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
		return errRolledBack
	})
	if errors.Is(err, errRolledBack) {
		return nil
	}
	return err
}

// schemaThrough is set's migrations up to and including version, read from
// dir, as a library made by an earlier build holds them.
func schemaThrough(t *testing.T, set, dir string, version int) storage.Schema[storage.Library] {
	t.Helper()
	files := fstest.MapFS{}
	for v := 1; v <= version; v++ {
		matches, err := filepath.Glob(filepath.Join(dir, fmt.Sprintf("%05d_*.sql", v)))
		if err != nil || len(matches) != 1 {
			t.Fatalf("migration %d of %s: %v %v", v, set, matches, err)
		}
		b, err := os.ReadFile(matches[0])
		if err != nil {
			t.Fatal(err)
		}
		files["schema/"+filepath.Base(matches[0])] = &fstest.MapFile{Data: b}
	}
	return storage.NewSchema[storage.Library](set, files)
}

func TestALibraryFromBeforeMemberAccessKeepsItsRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	openAt := func(schemas ...storage.Schema[storage.Library]) library {
		lib, err := storage.OpenLibrary(t.Context(), path, schemas, nil, storage.DefaultTimings(), slog.New(slog.DiscardHandler))
		if err != nil {
			t.Fatal(err)
		}
		return lib
	}
	before := openAt(secret.Schema(), schemaThrough(t, "identity", "../identity/schema", 3), nodes.Schema(), works.Schema(), roots.Schema(),
		schemaThrough(t, "mediaservers", "schema", 4))
	f := fixture{lib: before, node: storage.NewUUID(), root: storage.NewUUID(), server: storage.NewUUID()}
	seedServer(t, f)
	must(t, before, `INSERT INTO mediaservers_server (id, kind, name, address, identity, version, reaching_node_id, secrets_sealed,
		created_at, updated_at) VALUES (`+plexServerID+`, 'plex', 'Living room Plex', 'http://den:32400', 'm1', '1.43.5', ?, x'00', 1, 1)`, f.node)
	must(t, before, `INSERT INTO mediaservers_plex_account (id, client_identifier, key_sealed, jwt_sealed, jwt_expires_at, account_name,
		plex_pass, signed_in_at) VALUES (1, 'c1', x'00', x'00', 9, 'jackson', 1, 1)`)
	must(t, before, `INSERT INTO identity_link (id, kind, token_hash, email, issued_at, expires_at)
		VALUES (randomblob(16), 'signup', randomblob(32), 'kim@example.com', 1, 2)`)
	if err := before.Close(); err != nil {
		t.Fatal(err)
	}

	lib := openAt(secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), roots.Schema(), mediaservers.Schema())
	t.Cleanup(func() { _ = lib.Close() })
	var shown, addressed, plexIDs, grants int
	err := lib.QueryRowContext(t.Context(), `SELECT
		(SELECT count(*) FROM mediaservers_server WHERE members_see = 1),
		(SELECT count(*) FROM mediaservers_server WHERE member_address IS NOT NULL),
		(SELECT count(*) FROM mediaservers_plex_account WHERE plex_id IS NOT NULL),
		(SELECT count(*) FROM identity_link WHERE grants IS NOT NULL)`).Scan(&shown, &addressed, &plexIDs, &grants)
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, lib, "mediaservers_server"); n != 2 || shown != 0 || addressed != 0 {
		t.Errorf("after the migration %d servers are kept, %d shown to Members and %d with an Address for Members; want 2, 0 and 0", n, shown, addressed)
	}
	if n := count(t, lib, "mediaservers_plex_account"); n != 1 || plexIDs != 0 {
		t.Errorf("after the migration %d Plex sign-ins are kept, %d with a Plex id; want 1 with none until renewal", n, plexIDs)
	}
	if n := count(t, lib, "identity_link"); n != 1 || grants != 0 {
		t.Errorf("after the migration %d Signup links are kept, %d with grants; want 1 with none", n, grants)
	}
}
