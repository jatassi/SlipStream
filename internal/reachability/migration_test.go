package reachability_test

import (
	"context"
	"testing"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/reachability"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

type schemaFixture struct {
	lib          *storage.DB[storage.Library]
	node, server storage.UUID
	ui, exposed  storage.UUID
}

func (f schemaFixture) exec(t *testing.T, sql string, args ...any) error {
	t.Helper()
	return f.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, sql, args...)
		return err
	})
}

func (f schemaFixture) must(t *testing.T, sql string, args ...any) {
	t.Helper()
	if err := f.exec(t, sql, args...); err != nil {
		t.Fatal(err)
	}
}

func (f schemaFixture) count(t *testing.T, table string) int {
	t.Helper()
	var n int
	if err := f.lib.QueryRowContext(t.Context(), "SELECT count(*) FROM "+table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// seedSchema holds a Node, a Plex on it, the UI's Exposure and the Plex's,
// with a result and a dismissal for the Plex.
func seedSchema(t *testing.T) schemaFixture {
	t.Helper()
	f := schemaFixture{
		lib:  storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), roots.Schema(), mediaservers.Schema(), reachability.Schema()),
		node: storage.NewUUID(), server: storage.NewUUID(), ui: storage.NewUUID(), exposed: storage.NewUUID(),
	}
	f.must(t, "INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'atlas', 0)", f.node)
	f.must(t, `INSERT INTO mediaservers_server (id, kind, name, address, identity, version, reaching_node_id, secrets_sealed, created_at, updated_at)
		VALUES (?, 'plex', 'Plex', 'http://atlas:32400', 'm1', '1.43.5', ?, x'00', 1, 1)`, f.server, f.node)
	f.must(t, "INSERT INTO reachability_exposure (id, kind, wanted_at) VALUES (?, 'ui-portal', 1)", f.ui)
	f.must(t, "INSERT INTO reachability_exposure (id, kind, server_id, public, wanted_at) VALUES (?, 'media-server', ?, 'https://plex.example.org', 1)", f.exposed, f.server)
	f.must(t, "INSERT INTO reachability_result (exposure_id, judged, verdict, tested, judged_at, failing_at) VALUES (?, '{}', 'fixable', 1, 5, 4)", f.exposed)
	f.must(t, "INSERT INTO reachability_dismissed (kind, server_id, dismissed_at) VALUES ('media-server', ?, 1)", f.server)
	return f
}

func TestTheSchemaRefusesRowsNoReaderParses(t *testing.T) {
	f := seedSchema(t)
	other := storage.NewUUID()
	for _, c := range []struct {
		name string
		sql  string
		args []any
	}{
		{"a second UI Exposure", "INSERT INTO reachability_exposure (id, kind, wanted_at) VALUES (?, 'ui-portal', 1)", []any{other}},
		{"a second Exposure of one server", "INSERT INTO reachability_exposure (id, kind, server_id, public, wanted_at) VALUES (?, 'media-server', ?, 'https://x.example.org', 1)", []any{other, f.server}},
		{"a Media Server Exposure with no public address", "INSERT INTO reachability_exposure (id, kind, server_id, wanted_at) VALUES (?, 'media-server', ?, 1)", []any{other, f.server}},
		{"a UI Exposure with a public address", "INSERT INTO reachability_exposure (id, kind, public, wanted_at) VALUES (?, 'ui-portal', 'https://x.example.org', 1)", []any{other}},
		{"server facts with no time", "UPDATE reachability_exposure SET server_facts = '{}' WHERE id = ?", []any{f.exposed}},
		{"a Reachable result failing", "UPDATE reachability_result SET verdict = 'reachable' WHERE exposure_id = ?", []any{f.exposed}},
		{"failing after the judgment", "UPDATE reachability_result SET failing_at = 6 WHERE exposure_id = ?", []any{f.exposed}},
		{"an unknown verdict", "UPDATE reachability_result SET verdict = 'maybe' WHERE exposure_id = ?", []any{f.exposed}},
		{"a UI dismissal naming a server", "INSERT INTO reachability_dismissed (kind, server_id, dismissed_at) VALUES ('ui-portal', ?, 1)", []any{f.server}},
		{"a second dismissal of one server", "INSERT INTO reachability_dismissed (kind, server_id, dismissed_at) VALUES ('media-server', ?, 1)", []any{f.server}},
		{"a zero warning wait", "UPDATE reachability_settings SET warn_after_seconds = 0", nil},
		{"a negative change wait", "UPDATE reachability_settings SET change_wait_seconds = -1", nil},
		{"STUN servers that aren't a list", "UPDATE reachability_settings SET stun_servers = '\"stun.example.org:3478\"'", nil},
		{"a Public4 of six bytes", "INSERT INTO reachability_node (node_id, facts, observed_at, public4) VALUES (?, '{}', 1, x'000000000000')", []any{f.node}},
		{"a second settings row", "INSERT INTO reachability_settings (id) VALUES (2)", nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			if err := f.exec(t, c.sql, c.args...); err == nil {
				t.Error("stored")
			}
		})
	}
	f.must(t, "UPDATE reachability_settings SET change_wait_seconds = 0, stun_servers = '[]'")
}

func TestRemovingAServerForgetsItsExposureResultAndDismissal(t *testing.T) {
	f := seedSchema(t)
	f.must(t, "DELETE FROM mediaservers_server WHERE id = ?", f.server)
	for table, want := range map[string]int{"reachability_exposure": 1, "reachability_result": 0, "reachability_dismissed": 0} {
		if n := f.count(t, table); n != want {
			t.Errorf("%s holds %d rows after the server was removed, want %d", table, n, want)
		}
	}
	f.must(t, "INSERT INTO reachability_result (exposure_id, judged, verdict, tested, judged_at) VALUES (?, '{}', 'reachable', 0, 1)", f.ui)
	f.must(t, "DELETE FROM reachability_exposure WHERE id = ?", f.ui)
	if n := f.count(t, "reachability_result"); n != 0 {
		t.Errorf("removing an Exposure left %d results", n)
	}
}
