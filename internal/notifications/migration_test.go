package notifications_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/history/historytest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/notifications"
	"github.com/jatassi/slipstream/internal/notifications/notificationstest"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
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

// golden is one owner's stored subject from its render goldens, with the
// Title main renders it under.
type golden struct {
	subject, title string
}

func readGolden(t *testing.T, path string) golden {
	t.Helper()
	raw, err := os.ReadFile(path + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil {
		t.Fatal(err)
	}
	words, err := os.ReadFile(path + ".txt")
	if err != nil {
		t.Fatal(err)
	}
	title, _, _ := strings.Cut(strings.TrimPrefix(string(words), "Title: "), "\n")
	return golden{subject: compact.String(), title: title}
}

// sqlText is s as a SQL string literal.
func sqlText(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// seeded is a library as notifications left it before History.
type seeded struct {
	admin   storage.UUID
	keyFile string
	g       map[string]golden
}

// seedBeforeTheMove writes one event of each kind into the library at
// path: released ones, an owed Imported, a waiting raise whose storage.d a
// Worse replaced in place, bell rows with a deleted event above them, and
// an Admin who read the first.
func seedBeforeTheMove(t *testing.T, path string) seeded {
	t.Helper()
	ctx := t.Context()
	log := slog.New(slog.DiscardHandler)
	old, err := storage.OpenLibrary(ctx, path, []storage.Schema[storage.Library]{
		secret.Schema(), identity.Schema(), mail.Schema(), nodes.Schema(), notifications.Schema().Through(1),
	}, nil, storage.DefaultTimings(), log)
	if err != nil {
		t.Fatal(err)
	}
	g := map[string]golden{}
	for name, p := range map[string]string{
		"grabbed": "../grabs/testdata/notices/grabbed-manual", "grab_failed": "../grabs/testdata/notices/grab-failed-name_taken",
		"imported": "../imports/testdata/notices/imported-one", "upgraded": "../imports/testdata/notices/upgraded-one",
		"scan": "../imports/testdata/notices/works-added-by-scan", "works_added": "../library/testdata/notices/works-added",
		"deleted_one": "../library/testdata/notices/works-deleted-one", "deleted_many": "../library/testdata/notices/works-deleted-many",
	} {
		g[name] = readGolden(t, p)
	}
	now := time.Now()
	at := func(d time.Duration) string { return strconv.FormatInt(now.Add(d).UnixMicro(), 10) }
	released := func(id int, kind, subject string, d time.Duration) string {
		return fmt.Sprintf("INSERT INTO notifications_event (id, kind, at, due_at, released_at, subject) VALUES (%d, '%s', %s, %s, %s, %s)",
			id, kind, at(d), at(d), at(d), sqlText(subject))
	}
	admin, conn := storage.NewUUID(), storage.NewUUID()
	keyFile := filepath.Join(t.TempDir(), secret.KeyFile)
	keys, err := secret.Open(ctx, old, keyFile)
	if err != nil {
		t.Fatal(err)
	}
	token := keys.Sealer().Seal([]byte(`{"token":"`+notificationstest.Token("family")+`"}`), secret.NewColumn("notifications_connection.secrets_sealed").Row(conn.String()))
	err = old.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO notifications_connection (id, name, provider, is_on, on_at, events, warnings, posters, settings, secrets_sealed, revision, created_at)
			VALUES (?, 'family', 'hook', 1, 0, '["imported","new_health_item","ping"]', 1, 1, ?, ?, 1, 0)`, conn, `{"url":"`+family+`"}`, token)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	exec(t, old,
		released(1, "grabbed", g["grabbed"].subject, -time.Hour),
		released(2, "grab_failed", g["grab_failed"].subject, -59*time.Minute),
		"INSERT INTO notifications_event (id, kind, at, due_at, subject) VALUES (3, 'imported', "+at(-58*time.Minute)+", "+at(-58*time.Minute)+", "+sqlText(g["imported"].subject)+")",
		released(4, "upgraded", g["upgraded"].subject, -57*time.Minute),
		released(5, "works_added", g["works_added"].subject, -56*time.Minute),
		released(6, "works_added_by_scan", g["scan"].subject, -55*time.Minute),
		released(7, "works_deleted", g["deleted_one"].subject, -54*time.Minute),
		released(8, "works_deleted", g["deleted_many"].subject, -53*time.Minute),
		released(9, "new_health_item", "{}", -52*time.Minute),
		"INSERT INTO notifications_item (event_id, key, level, subject, text, since_at) VALUES (9, 'storage.e', 'warn', 'Disk e is almost full', 'It has 1 GB left.', "+at(-52*time.Minute)+")",
		released(10, "health_item_cleared", "{}", -51*time.Minute),
		"INSERT INTO notifications_item (event_id, key, level, subject, text, since_at, cleared_at) VALUES (10, 'storage.g', 'warn', 'Disk g is almost full', '', "+at(-90*time.Minute)+", "+at(-51*time.Minute)+")",
		"INSERT INTO notifications_event (id, kind, at, due_at, subject) VALUES (11, 'new_health_item', "+at(-2*time.Minute)+", "+at(3*time.Minute)+", '{}')",
		"INSERT INTO notifications_item (event_id, key, level, subject, text, since_at) VALUES (11, 'storage.d', 'fail', 'Disk d is full', 'It has no space left.', "+at(-2*time.Minute)+"), (11, 'storage.f', 'warn', 'Disk f is almost full', 'It has 1 GB left.', "+at(-2*time.Minute)+")",
		released(12, "grabbed", g["grabbed"].subject, -time.Minute),
		"INSERT INTO notifications_shown (seq, event_id, shown_at) VALUES (1, 1, "+at(-time.Hour)+"), (2, 7, "+at(-54*time.Minute)+")",
		"INSERT INTO notifications_shown (seq, event_id, shown_at, items) VALUES (3, 9, "+at(-52*time.Minute)+`, '["storage.e"]')`,
		"INSERT INTO notifications_shown (seq, event_id, shown_at) VALUES (4, 12, "+at(-time.Minute)+")",
		"INSERT INTO notifications_told (target, key, subject, since_at, told_at) VALUES ('bell', 'storage.e', 'Disk e is almost full', "+at(-52*time.Minute)+", "+at(-52*time.Minute)+")",
		"DELETE FROM notifications_event WHERE id = 12",
	)
	err = old.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO identity_account (id, email, display_name, admin, pin_hash, created_at) VALUES (?, 'sam@example.test', 'Sam', 1, '', 0)", admin); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO notifications_delivery (event_id, connection_id, state, attempts, accepted, ended_at, reason)
			VALUES (1, ?, 'given_up', 3, '[]', ?, '{"code":"network"}')`, conn, now.Add(-50*time.Minute).UnixMicro()); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "INSERT INTO notifications_seen (account_id, seq) VALUES (?, 1)", admin)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}
	return seeded{admin: admin, keyFile: keyFile, g: g}

}

func TestTheMoveKeepsTheBellTheQueueAndEveryEntry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		path := filepath.Join(t.TempDir(), storage.LibraryFile)
		s := seedBeforeTheMove(t, path)
		lib, err := storage.OpenLibrary(t.Context(), path, notificationstest.Schemas(), nil, storage.DefaultTimings(), slog.New(slog.DiscardHandler))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = lib.Close() })
		w := notificationstest.Over(t, lib, s.keyFile, notifications.DefaultTimings(),
			notificationstest.Owner{Name: "grabs", Rows: grabs.Entries()},
			notificationstest.Owner{Name: "imports", Rows: imports.Entries()},
			notificationstest.Owner{Name: "library", Rows: library.Entries()},
		)
		sam := w.SignIn(identity.AccountID{UUID: s.admin})

		f := w.Feed(sam)
		want := []string{s.g["imported"].title, "New Health item for Disk e is almost full", s.g["deleted_one"].title, s.g["grabbed"].title}
		if got := titlesOf(f.Rows); !slices.Equal(got, want) || *f.Unread != 3 {
			t.Fatalf("the bell lists %q with %d unread, want %q with 3: the Imported that was still owed, then the old rows", got, *f.Unread, want)
		}
		migrated := historytest.Entries(t, lib)
		wantMigrated(t, migrated)
		wantListed(t, w, migrated)

		w.Record(notificationstest.PingRow, notificationstest.Ping{Name: "new"})
		after := historytest.Entries(t, lib)
		if id := after[len(after)-1].ID; id <= 12 {
			t.Errorf("the new entry took id %d, want one above the deleted 12", id)
		}
		f = w.Feed(sam)
		if top := f.Rows[0]; top.Seq <= 4 || top.Read || *f.Unread != 4 {
			t.Errorf("the new bell row is seq %d, read %v, with %d unread, want above the deleted 4, unread, and 4 unread", top.Seq, top.Read, *f.Unread)
		}

		w.Advance(time.Hour)
		got := w.Courier.Received(family)
		var sent []string
		for _, r := range got {
			sent = append(sent, r.Title)
		}
		if want := []string{s.g["imported"].title, "Ping new", "New Health items for Disk d is full and Disk f is almost full"}; !slices.Equal(sent, want) {
			t.Fatalf("family received %q, want %q: what was owed, each once", sent, want)
		}
		if raise := got[2]; raise.Tone != entry.Fail || !slices.Contains(raise.Lines, "Disk d is full") {
			t.Fatalf("the raise is %+v, want the Worse's words at Error", raise)
		}
		wantNoDanglingRows(t, lib)
	})
}

func titlesOf(rows []notifications.FeedRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.Title
	}
	return out
}

// wantMigrated checks entries are the seeded events, each on its Work
// where it names one.
func wantMigrated(t *testing.T, entries []historytest.Entry) {
	t.Helper()
	var kinds []entry.Kind
	for _, e := range entries {
		kinds = append(kinds, e.Kind)
	}
	want := []entry.Kind{
		"grabbed", "grab_failed", "imported", "upgraded", "works_added", "works_added_by_scan", "works_deleted", "works_deleted",
		"new_health_item", "health_item_cleared", "new_health_item",
	}
	if !slices.Equal(kinds, want) || entries[0].ID != 1 || entries[len(entries)-1].ID != 11 {
		t.Fatalf("History holds %v, want %v as ids 1 to 11", kinds, want)
	}
	one, many := entries[6], entries[7]
	if one.Work.IsZero() || !slices.Equal(one.Works, []works.WorkID{one.Work}) || one.Type == "" {
		t.Errorf("the single-Work delete is about %v of type %q, with Works %v, want its Work", one.Work, one.Type, one.Works)
	}
	if !many.Work.IsZero() || len(many.Works) != 0 || many.Type != "" {
		t.Errorf("the 12-Work delete is about %v of type %q with Works %v, want none", many.Work, many.Type, many.Works)
	}
	if items := entries[10].Items; len(items) != 2 || items[0].Level != "fail" || items[0].Subject != "Disk d is full" {
		t.Errorf("the waiting raise holds %+v, want the Worse's words for storage.d", items)
	}
}

// wantListed is History's list of the migrated entries, newest first, each
// in its owner's words, and a Work's History its entries alone, the
// single-Work delete among them and the 12-Work delete not.
func wantListed(t *testing.T, w *notificationstest.World, migrated []historytest.Entry) {
	t.Helper()
	all, err := w.History.Page(t.Context(), identity.Admin{}, history.Filter{}, 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	var kinds, want []entry.Kind
	for _, e := range all.Entries {
		kinds = append(kinds, e.Spec.Kind)
		if e.Message.Title == "" {
			t.Errorf("History lists entry %d of kind %s with no words", e.ID, e.Spec.Kind)
		}
	}
	for _, e := range slices.Backward(migrated) {
		want = append(want, e.Kind)
	}
	if !slices.Equal(kinds, want) || all.Next != 0 {
		t.Fatalf("History lists %v, want every migrated entry newest first: %v", kinds, want)
	}
	one := migrated[6]
	onWork, err := w.History.Page(t.Context(), identity.Admin{}, history.Filter{Work: one.Work}, 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	var gotIDs, wantIDs []history.EventID
	for _, e := range onWork.Entries {
		gotIDs = append(gotIDs, e.ID)
	}
	for _, e := range slices.Backward(migrated) {
		if e.Work == one.Work {
			wantIDs = append(wantIDs, e.ID)
		}
	}
	if !slices.Equal(gotIDs, wantIDs) || !slices.Contains(gotIDs, one.ID) {
		t.Errorf("the deleted Work's History lists %v, want its own entries newest first: %v", gotIDs, wantIDs)
	}
}

func wantNoDanglingRows(t *testing.T, lib *storage.DB[storage.Library]) {
	t.Helper()
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		rows, err := r.QueryContext(ctx, "PRAGMA foreign_key_check")
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()
		if rows.Next() {
			return errors.New("a row refers to a row that is gone")
		}
		return rows.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
}
