package notifications_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/notifications"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

const (
	nodeID    = "X'00000000000000000000000000000001'"
	accountID = "X'00000000000000000000000000000002'"
	connID    = "X'00000000000000000000000000000003'"
	scriptID  = "X'00000000000000000000000000000004'"
)

// valid is one row each table takes; each case changes one column of it.
var valid = map[string]map[string]string{
	"notifications_connection": {
		"id": scriptID, "name": "'Den script'", "provider": "'script'", "is_on": "1", "on_at": "0", "events": "'[]'",
		"warnings": "0", "posters": "1", "node_id": nodeID, "settings": "'{}'", "revision": "1", "created_at": "0",
	},
	"notifications_line": {
		"connection_id": connID, "state": "'failing'", "state_at": "0", "reason": `'{"code":"network"}'`, "attempts": "1",
		"next_at": "0", "revision": "1", "given_up": "0",
	},
	"notifications_pending": {"event_id": "1", "kind": "'imported'", "at": "10", "due_at": "10", "items": "'[\"storage.d\"]'"},
	"notifications_worse":   {"event_id": "1", "key": "'storage.d'", "worse_id": "2"},
	"notifications_delivery": {
		"event_id": "1", "connection_id": connID, "kind": "'imported'", "at": "10", "state": "'given_up'", "attempts": "0", "items": "'[]'", "accepted": "'[]'",
		"ended_at": "0", "reason": `'{"code":"too_large"}'`, "response": "'too big'",
	},
	"notifications_shown":    {"event_id": "1", "kind": "'imported'", "at": "10", "shown_at": "0", "items": "'[]'"},
	"notifications_seen":     {"account_id": accountID, "seq": "0"},
	"notifications_settings": {"id": "1", "health_wait_seconds": "0", "give_up_seconds": "1", "warn_seconds": "1"},
	"notifications_bell":     {"id": "1", "choices": "'{}'", "warnings": "1"},
}

func TestTheSchemaRefusesEveryRowItsChecksForbid(t *testing.T) {
	ctx := t.Context()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), history.Schema(), notifications.Schema())
	exec(t, lib,
		"INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES ("+nodeID+", randomblob(32), 'Den', 0)",
		"INSERT INTO identity_account (id, email, display_name, admin, pin_hash, created_at) VALUES ("+accountID+", 'sam@example.com', 'Sam', 1, '', 0)",
		"INSERT INTO notifications_connection (id, name, provider, is_on, on_at, events, warnings, posters, settings, revision, created_at) VALUES ("+connID+", 'Family', 'discord', 1, 0, '[]', 0, 1, '{}', 1, 0)",
		"INSERT INTO history_event (kind, at, subject) VALUES ('imported', 10, '{}'), ('imported', 20, '{}')",
	)
	for table, row := range valid {
		if err := tryRow(ctx, lib, table, row); err != nil {
			t.Fatalf("the valid %s row is refused: %v", table, err)
		}
	}

	for _, c := range []struct {
		name, table string
		set         map[string]string
	}{
		{"an empty connection name", "notifications_connection", map[string]string{"name": "''"}},
		{"a 101-character connection name", "notifications_connection", map[string]string{"name": "'" + strings.Repeat("a", 101) + "'"}},
		{"is_on other than 0 or 1", "notifications_connection", map[string]string{"is_on": "2"}},
		{"events that aren't an array", "notifications_connection", map[string]string{"events": "'{}'"}},
		{"warnings other than 0 or 1", "notifications_connection", map[string]string{"warnings": "2"}},
		{"posters other than 0 or 1", "notifications_connection", map[string]string{"posters": "2"}},
		{"settings that aren't an object", "notifications_connection", map[string]string{"settings": "'[]'"}},
		{"a connection revision below 1", "notifications_connection", map[string]string{"revision": "0"}},
		{"a script with no Node", "notifications_connection", map[string]string{"node_id": "NULL"}},
		{"a Node on a provider other than script", "notifications_connection", map[string]string{"provider": "'discord'"}},

		{"a line state outside the four", "notifications_line", map[string]string{"state": "'sent'"}},
		{"a line reason that isn't an object", "notifications_line", map[string]string{"reason": "'[]'"}},
		{"negative line attempts", "notifications_line", map[string]string{"attempts": "-1"}},
		{"a line revision below 1", "notifications_line", map[string]string{"revision": "0"}},
		{"a negative given-up count", "notifications_line", map[string]string{"given_up": "-1"}},
		{"an ok line with a reason", "notifications_line", map[string]string{"state": "'ok'", "attempts": "0"}},
		{"a paused line without a reason", "notifications_line", map[string]string{"state": "'paused'", "attempts": "0", "reason": "NULL"}},
		{"a waiting line with attempts", "notifications_line", map[string]string{"state": "'waiting'"}},

		{"a pending row with no kind", "notifications_pending", map[string]string{"kind": "''"}},
		{"a pending row due before it happened", "notifications_pending", map[string]string{"due_at": "9"}},
		{"pending items that aren't an array", "notifications_pending", map[string]string{"items": "'{}'"}},

		{"a Worse from an earlier entry", "notifications_worse", map[string]string{"event_id": "2", "worse_id": "1"}},

		{"a delivery state stored as sent", "notifications_delivery", map[string]string{"state": "'sent'"}},
		{"negative delivery attempts", "notifications_delivery", map[string]string{"attempts": "-1"}},
		{"delivery items that aren't an array", "notifications_delivery", map[string]string{"items": "'{}'"}},
		{"accepted that isn't an array", "notifications_delivery", map[string]string{"accepted": "'{}'"}},
		{"a delivery reason that isn't an object", "notifications_delivery", map[string]string{"reason": "'[]'"}},
		{"a given_up delivery without a reason", "notifications_delivery", map[string]string{"reason": "NULL"}},
		{"a given_up delivery that never ended", "notifications_delivery", map[string]string{"ended_at": "NULL"}},
		{"a due delivery that ended", "notifications_delivery", map[string]string{"state": "'due'", "reason": "NULL", "response": "NULL"}},
		{"a due delivery with a reason", "notifications_delivery", map[string]string{"state": "'due'", "ended_at": "NULL", "response": "NULL"}},
		{"a due delivery with a response", "notifications_delivery", map[string]string{"state": "'due'", "ended_at": "NULL", "reason": "NULL"}},

		{"shown items that aren't an array", "notifications_shown", map[string]string{"items": "'{}'"}},
		{"a negative seen cursor", "notifications_seen", map[string]string{"seq": "-1"}},

		{"a second settings row", "notifications_settings", map[string]string{"id": "2"}},
		{"a negative Health wait", "notifications_settings", map[string]string{"health_wait_seconds": "-1"}},
		{"a zero give-up time", "notifications_settings", map[string]string{"give_up_seconds": "0"}},
		{"a zero warn time", "notifications_settings", map[string]string{"warn_seconds": "0"}},

		{"a second bell row", "notifications_bell", map[string]string{"id": "2"}},
		{"bell choices that aren't an object", "notifications_bell", map[string]string{"choices": "'[]'"}},
		{"bell warnings other than 0 or 1", "notifications_bell", map[string]string{"warnings": "2"}},
	} {
		row := maps.Clone(valid[c.table])
		maps.Copy(row, c.set)
		err := tryRow(ctx, lib, c.table, row)
		if err == nil || !strings.Contains(err.Error(), "CHECK constraint failed") {
			t.Errorf("%s: err = %v, want a CHECK refusal", c.name, err)
		}
	}
}

var errRolledBack = errors.New("rolled back")

// tryRow writes row into table in a Write it then rolls back, replacing a
// singleton row's values where the table holds one.
func tryRow(ctx context.Context, lib *storage.DB[storage.Library], table string, row map[string]string) error {
	cols := slices.Sorted(maps.Keys(row))
	vals := make([]string, len(cols))
	for i, c := range cols {
		vals[i] = row[c]
	}
	verb := "INSERT"
	if table == "notifications_settings" || table == "notifications_bell" {
		verb = "REPLACE"
	}
	stmt := verb + " INTO " + table + " (" + strings.Join(cols, ", ") + ") VALUES (" + strings.Join(vals, ", ") + ")"
	err := lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
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

func exec(t *testing.T, lib *storage.DB[storage.Library], stmts ...string) {
	t.Helper()
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, s := range stmts {
			if _, err := tx.ExecContext(ctx, s); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
