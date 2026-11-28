package node_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/storage"
)

// conversionVersions is the version each set stood at before S44
// (Conversion)'s migrations. conversions, new with them, has none.
var conversionVersions = map[string]int64{"targets": 6, "files": 9, "imports": 6, "mediaservers": 8, "transfers": 5, "conversions": 0}

var rebuilt = []string{"targets_profile_target", "files_file", "files_imported", "imports_import", "imports_file",
	"transfers_record", "transfers_file", "mediaservers_server"}

const (
	attic   = "x'0000000000000000000000000000000a'"
	work    = "x'0000000000000000000000000000000b'"
	root    = "x'0000000000000000000000000000000c'"
	source  = "x'00000000000000000000000000000001'"
	made    = "x'00000000000000000000000000000002'"
	target  = "x'00000000000000000000000000000003'"
	derived = "x'00000000000000000000000000000004'"
	profile = "x'00000000000000000000000000000005'"
	convID  = "x'00000000000000000000000000000006'"
	server  = "x'00000000000000000000000000000007'"
	record  = "x'00000000000000000000000000000008'"
	imp     = "x'00000000000000000000000000000009'"
)

// seeded is a library as it stood before S44, with a row in every table
// its migrations rebuild or widen.
var seeded = []string{
	"INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (" + attic + ", randomblob(32), 'Attic', 0)",
	"INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (" + work + ", 'fixture', 'w1', 'Orchard', 'Orchard', 0, 0)",
	"INSERT INTO roots_root (id, content_type, node_id, path, is_default) VALUES (" + root + ", 'fixture', " + attic + ", '/media', 1)",
	"INSERT INTO targets_target (id, name, created_at) VALUES (" + target + ", '2160p HDR', 0), (" + derived + ", '720p mobile', 0)",
	"INSERT INTO targets_profile (id, name, content_type, is_default) VALUES (" + profile + ", 'Series', 'fixture', 1)",
	"INSERT INTO targets_profile_target (profile_id, position, target_id) VALUES (" + profile + ", 1, " + target + "), (" + profile + ", 2, " + derived + ")",
	"INSERT INTO files_file (id, work_id, cut, root_id, path, size, probe, quality, release_name, placed_by) VALUES (" +
		source + ", " + work + ", 'standard', " + root + ", 'Orchard.mkv', 10, '{\"version\":3}', 'remux-2160p', 'Orchard.2160p', 'copy')",
	"INSERT INTO files_imported (id, file_id, work_id, placed_by, items, cells, replaced, at, quality) VALUES (" +
		imp + ", " + source + ", " + work + ", 'copy', '[]', '[]', '[]', 5, 'remux-2160p')",
	"INSERT INTO imports_import (id, source, root_id, folder, content_type, hint, created_at) VALUES (" + imp + ", 'root', " + root + ", 'Orchard', 'fixture', 'Orchard', 1)",
	"INSERT INTO imports_file (import_id, position, path, size, cut, release_name, source_name, outcome, reason) VALUES (" +
		imp + ", 0, 'Orchard/Orchard.mkv', 10, 'standard', 'Orchard.2160p', 'Orchard.mkv', 'waiting', 'taken')",
	"INSERT INTO transfers_record (id, purpose, job, source_node, destination_kind, destination_root, created_at) VALUES (" +
		record + ", 'import', 'orchard', " + attic + ", 'root', " + root + ", 1)",
	"INSERT INTO transfers_file (transfer_id, position, group_key, name, source_kind, path, size) VALUES (" + record + ", 0, '0', 'Orchard.mkv', 'path', '/downloads/Orchard.mkv', 10)",
	"INSERT INTO mediaservers_server (id, kind, name, address, identity, version, monitor_delay_ms, reaching_node_id, device_id, secrets_sealed, created_at, updated_at) VALUES (" +
		server + ", 'jellyfin', 'Den Jellyfin', 'http://den:8096', 'f4c1', '12.2.0', 60000, " + attic + ", 'd1', x'00', 1, 1)",
}

// values is a row's columns as SQL literals.
type values map[string]string

func (v values) with(changes values) values {
	out := maps.Clone(v)
	for k, c := range changes {
		if c == "" {
			delete(out, k)
		} else {
			out[k] = c
		}
	}
	return out
}

func insert(table string, v values) string {
	cols := slices.Sorted(maps.Keys(v))
	vals := make([]string, len(cols))
	for i, c := range cols {
		vals[i] = v[c]
	}
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(cols, ", "), strings.Join(vals, ", "))
}

var (
	setting = values{"target_id": derived, "video_codec": "'hevc'", "max_height": "1080", "bitrate_kbps": "4000", "bit_depth": "10",
		"range": "'hdr10'", "keep_dv": "1", "curve": "'bt2390'", "audio": "'copy'", "drop_commentary": "1", "container": "'mkv'", "run_on_node_id": attic}
	madeFile = values{"id": made, "work_id": work, "cut": "'standard'", "root_id": root, "path": "'Orchard - 720p mobile.mkv'", "size": "4",
		"probe": "'{}'", "placed_by": "'converted'", "derived": "1", "derived_from_file_id": source, "derived_target_id": derived, "conversion_version": "1"}
	conversion = values{"id": convID, "work_id": work, "type": "'fixture'", "items": "'[]'", "edition": "'standard'", "source_file_id": source,
		"target_id": derived, "version": "1", "lane": "'new'", "position": "'a'", "stage": "'queued'", "not_before": "0", "snapshot": "'{}'", "queued_at": "0"}
	running          = conversion.with(values{"stage": "'running'", "node_id": attic, "attempt": "1", "root_id": root})
	failed           = conversion.with(values{"stage": "'failed'", "failure_code": "'crash'", "failure_class": "'run'", "failure_detail": "''"})
	attempt          = values{"conversion_id": convID, "n": "1", "node_id": attic, "node_name": "'Attic'", "path": "'processor'", "api": "''", "started_at": "0"}
	proof            = values{"node_id": attic, "signature": "'s1'", "shape": "'h264.8.sdr.bt2390'", "class": "'hevc_10'", "path": "'processor'", "api": "''", "speed": "2.5", "proven_at": "1"}
	conversionImport = values{"id": "x'000000000000000000000000000000f1'", "source": "'conversion'", "root_id": root, "conversion_id": convID, "work_id": work,
		"content_type": "'fixture'", "hint": "'Orchard'", "created_at": "2"}
	scratchRecord = values{"id": "x'000000000000000000000000000000f2'", "purpose": "'conversion'", "job": "'src/a/b'", "source_node": attic,
		"destination_kind": "'scratch'", "destination_node": attic, "destination_name": "'in/s-1'", "created_at": "2"}
	scratchFile      = values{"transfer_id": record, "position": "1", "group_key": "''", "name": "'c-6-1.mkv'", "source_kind": "'scratch'", "path": "'out/c-6-1.mkv'", "size": "4"}
	queuedConversion = insert("conversions_conversion", conversion)
)

// shapes are what the migrated library takes: each case runs its
// statements in one Write that is rolled back, and refuses only its last
// statement with refusal when refusal is set.
var shapes = []struct {
	name    string
	stmts   []string
	refusal string
}{
	{name: "Conversion settings", stmts: []string{insert("targets_conversion", setting)}},
	{name: "a settings version below 1", stmts: []string{insert("targets_conversion", setting.with(values{"version": "0"}))}, refusal: "CHECK"},
	{name: "a bitrate below 100 kbps", stmts: []string{insert("targets_conversion", setting.with(values{"bitrate_kbps": "99"}))}, refusal: "CHECK"},
	{name: "10 bit H.264", stmts: []string{insert("targets_conversion", setting.with(values{"video_codec": "'h264'", "keep_dv": "0"}))}, refusal: "CHECK"},
	{name: "Dolby Vision kept in SDR", stmts: []string{insert("targets_conversion", setting.with(values{"range": "'sdr'"}))}, refusal: "CHECK"},
	{name: "Dolby Vision kept in H.264", stmts: []string{insert("targets_conversion", setting.with(values{"video_codec": "'h264'", "bit_depth": "8"}))}, refusal: "CHECK"},
	{name: "a row made from the row above", stmts: []string{"UPDATE targets_profile_target SET sourcing = 'derive', from_target_id = " + target + " WHERE position = 2"}},
	{name: "a row downloaded, or made after a day", stmts: []string{"UPDATE targets_profile_target SET sourcing = 'download_then_derive', from_target_id = " + target + ", grace_seconds = 86400 WHERE position = 2"}},
	{name: "made from nothing", stmts: []string{"UPDATE targets_profile_target SET sourcing = 'derive' WHERE position = 2"}, refusal: "CHECK"},
	{name: "downloaded from a Target", stmts: []string{"UPDATE targets_profile_target SET from_target_id = " + target + " WHERE position = 2"}, refusal: "CHECK"},
	{name: "a wait with no grace", stmts: []string{"UPDATE targets_profile_target SET sourcing = 'download_then_derive', from_target_id = " + target + " WHERE position = 2"}, refusal: "CHECK"},
	{name: "a grace without the wait", stmts: []string{"UPDATE targets_profile_target SET sourcing = 'derive', from_target_id = " + target + ", grace_seconds = 86400 WHERE position = 2"}, refusal: "CHECK"},
	{name: "a grace under an hour", stmts: []string{"UPDATE targets_profile_target SET sourcing = 'download_then_derive', from_target_id = " + target + ", grace_seconds = 3599 WHERE position = 2"}, refusal: "CHECK"},
	{name: "a grace over 90 days", stmts: []string{"UPDATE targets_profile_target SET sourcing = 'download_then_derive', from_target_id = " + target + ", grace_seconds = 7776001 WHERE position = 2"}, refusal: "CHECK"},
	{name: "made from itself", stmts: []string{"UPDATE targets_profile_target SET sourcing = 'derive', from_target_id = target_id WHERE position = 2"}, refusal: "CHECK"},
	{name: "a made file", stmts: []string{insert("files_file", madeFile),
		"INSERT INTO files_imported (id, file_id, work_id, placed_by, items, cells, replaced, at) VALUES (" + made + ", " + made + ", " + work + ", 'converted', '[]', '[]', '[]', 6)"}},
	{name: "a made file whose source and Target go", stmts: []string{"INSERT INTO targets_target (id, name, created_at) VALUES (" + server + ", 'Phone', 0)",
		insert("files_file", madeFile.with(values{"derived_target_id": server})), "DELETE FROM files_file WHERE id = " + source, "DELETE FROM targets_target WHERE id = " + server}},
	{name: "provenance on a downloaded file", stmts: []string{insert("files_file", madeFile.with(values{"derived": "0", "derived_target_id": "", "conversion_version": ""}))}, refusal: "CHECK"},
	{name: "a version on a downloaded file", stmts: []string{insert("files_file", madeFile.with(values{"derived": "0", "derived_from_file_id": "", "derived_target_id": ""}))}, refusal: "CHECK"},
	{name: "a Conversion queued", stmts: []string{queuedConversion}},
	{name: "a Conversion running", stmts: []string{insert("conversions_conversion", running)}},
	{name: "a Conversion failed", stmts: []string{insert("conversions_conversion", failed)}},
	{name: "a Conversion done", stmts: []string{insert("conversions_conversion", conversion.with(values{"stage": "'done'", "finished_at": "9", "output_file_id": source}))}},
	{name: "a second unfinished Conversion of a pair", stmts: []string{queuedConversion, insert("conversions_conversion", conversion.with(values{"id": made, "position": "'b'"}))}, refusal: "UNIQUE"},
	{name: "two unfinished Conversions at one position", stmts: []string{queuedConversion, insert("conversions_conversion", conversion.with(values{"id": made, "target_id": target}))}, refusal: "UNIQUE"},
	{name: "a finished Conversion without its time", stmts: []string{insert("conversions_conversion", conversion.with(values{"stage": "'done'"}))}, refusal: "CHECK"},
	{name: "an unfinished Conversion with a finish", stmts: []string{insert("conversions_conversion", conversion.with(values{"finished_at": "9"}))}, refusal: "CHECK"},
	{name: "a running Conversion on no Node", stmts: []string{insert("conversions_conversion", running.with(values{"node_id": ""}))}, refusal: "CHECK"},
	{name: "a queued Conversion on a Node", stmts: []string{insert("conversions_conversion", conversion.with(values{"node_id": attic}))}, refusal: "CHECK"},
	{name: "a failure without its code", stmts: []string{insert("conversions_conversion", conversion.with(values{"stage": "'failed'"}))}, refusal: "CHECK"},
	{name: "a code without its class", stmts: []string{insert("conversions_conversion", failed.with(values{"failure_class": ""}))}, refusal: "CHECK"},
	{name: "a cancel without its reason", stmts: []string{insert("conversions_conversion", conversion.with(values{"stage": "'canceled'", "finished_at": "9"}))}, refusal: "CHECK"},
	{name: "a pin on a started Conversion", stmts: []string{insert("conversions_conversion", running.with(values{"pin_node_id": attic}))}, refusal: "CHECK"},
	{name: "an attempt", stmts: []string{queuedConversion, insert("conversions_attempt", attempt)}},
	{name: "an attempt ended without an outcome", stmts: []string{queuedConversion, insert("conversions_attempt", attempt.with(values{"ended_at": "3"}))}, refusal: "CHECK"},
	{name: "an attempt's code without its class", stmts: []string{queuedConversion, insert("conversions_attempt", attempt.with(values{"ended_at": "3", "outcome": "'failed'", "code": "'crash'"}))}, refusal: "CHECK"},
	{name: "a proof", stmts: []string{insert("conversions_proof", proof)}},
	{name: "a proof neither proven nor failed", stmts: []string{insert("conversions_proof", proof.with(values{"proven_at": ""}))}, refusal: "CHECK"},
	{name: "a proof lost before it was proven", stmts: []string{insert("conversions_proof", proof.with(values{"proven_at": "", "failed_at": "1", "lost_at": "2"}))}, refusal: "CHECK"},
	{name: "a proof proven at no speed", stmts: []string{insert("conversions_proof", proof.with(values{"speed": ""}))}, refusal: "CHECK"},
	{name: "Node settings", stmts: []string{"INSERT INTO conversions_node (node_id, at_once) VALUES (" + attic + ", 16)"}},
	{name: "17 Conversions at once", stmts: []string{"INSERT INTO conversions_node (node_id, at_once) VALUES (" + attic + ", 17)"}, refusal: "CHECK"},
	{name: "hours past midnight", stmts: []string{"INSERT INTO conversions_node (node_id, hours_end) VALUES (" + attic + ", 1440)"}, refusal: "CHECK"},
	{name: "a decline", stmts: []string{"INSERT INTO conversions_declined VALUES (" + source + ", " + derived + ")"}},
	{name: "a nudge", stmts: []string{"INSERT INTO conversions_nudge (work_id, why, since, created_at) VALUES (" + work + ", 'new', 1, 1)"}},
	{name: "a nudge for a made file", stmts: []string{"INSERT INTO conversions_nudge (work_id, why, since, created_at) VALUES (" + work + ", 'made', 1, 1)"}, refusal: "CHECK"},
	{name: "a backfill", stmts: []string{"INSERT INTO conversions_backfill (id, profile_id, target_id, kind, created_at, count) VALUES (" + made + ", " + profile + ", " + derived + ", 'remake', 1, 18)"}},
	{name: "a Conversion's Import", stmts: []string{queuedConversion, insert("imports_import", conversionImport),
		"INSERT INTO imports_file (import_id, position, path, size, cut, release_name, source_name, outcome, placed_by, root_id, dest, quality, title_read) VALUES " +
			"(x'000000000000000000000000000000f1', 0, 'c-6-1.mkv', 4, 'standard', 'Orchard.720p', 'Orchard.mkv', 'placing', 'converted', " + root + ", 'Orchard.mkv', 'bluray-720p', '{}')"}},
	{name: "deleting the Work of a Conversion's Import", stmts: []string{queuedConversion, insert("imports_import", conversionImport), "DELETE FROM works_work WHERE id = " + work}},
	{name: "a second Import of a Conversion", stmts: []string{queuedConversion, insert("imports_import", conversionImport),
		insert("imports_import", conversionImport.with(values{"id": "x'000000000000000000000000000000f3'"}))}, refusal: "UNIQUE"},
	{name: "a Conversion's Import into no root", stmts: []string{queuedConversion, insert("imports_import", conversionImport.with(values{"root_id": ""}))}, refusal: "CHECK"},
	{name: "a Conversion on a root Import", stmts: []string{queuedConversion, insert("imports_import", conversionImport.with(values{"source": "'root'", "folder": "'Elsewhere'"}))}, refusal: "CHECK"},
	{name: "a title read that isn't JSON", stmts: []string{"UPDATE imports_file SET title_read = 'Orchard'"}, refusal: "CHECK"},
	{name: "a scratch destination", stmts: []string{insert("transfers_record", scratchRecord)}},
	{name: "a scratch destination with no name", stmts: []string{insert("transfers_record", scratchRecord.with(values{"destination_name": ""}))}, refusal: "CHECK"},
	{name: "a scratch name on a root", stmts: []string{"UPDATE transfers_record SET destination_name = 'in/s-1'"}, refusal: "CHECK"},
	{name: "a scratch file", stmts: []string{insert("transfers_file", scratchFile)}},
	{name: "a scratch file with no name", stmts: []string{insert("transfers_file", scratchFile.with(values{"path": ""}))}, refusal: "CHECK"},
	{name: "a server that doesn't pause", stmts: []string{"UPDATE mediaservers_server SET pauses_conversions = 0"}},
	{name: "a pause switch neither on nor off", stmts: []string{"UPDATE mediaservers_server SET pauses_conversions = 2"}, refusal: "CHECK"},
}

var errRolledBack = errors.New("rolled back")

// TestConversionMigrationsKeepEveryRowAndRefuseWhatTheirChecksForbid
// migrates a library from before S44 (Conversion): every rebuilt table
// keeps its columns, indexes and rows, a server pauses Conversions by
// default, and each new CHECK refuses one row the shape beside it takes.
func TestConversionMigrationsKeepEveryRowAndRefuseWhatTheirChecksForbid(t *testing.T) {
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	kept := seedBeforeConversion(t, path)
	lib := openAt(t, path, node.LibrarySchemas(contenttype.Set{}))
	t.Cleanup(func() { _ = lib.Close() })
	for _, table := range rebuilt {
		kept[table].keptIn(t, table, factsOf(t, lib, table, kept[table].columns))
	}
	if pauses := scalar(t, lib, "SELECT pauses_conversions FROM mediaservers_server"); pauses != "1" {
		t.Errorf("a server from before pauses Conversions = %s, want 1", pauses)
	}
	for _, c := range shapes {
		t.Run(c.name, func(t *testing.T) {
			err := tryRolledBack(t, lib, c.stmts, c.refusal != "")
			switch {
			case c.refusal == "" && err != errRolledBack: //nolint:errorlint // only the bare sentinel means every statement ran
				t.Errorf("refused: %v", err)
			case c.refusal != "" && (err == nil || errors.Is(err, errRolledBack) || !strings.Contains(err.Error(), c.refusal+" constraint failed")):
				t.Errorf("err = %v, want a %s constraint refusal of the last statement", err, c.refusal)
			}
		})
	}
}

// seedBeforeConversion writes seeded into a library at path migrated to
// just before S44, and answers the rebuilt tables' facts there.
func seedBeforeConversion(t *testing.T, path string) map[string]tableFacts {
	t.Helper()
	var before []storage.Schema[storage.Library]
	for _, s := range node.LibrarySchemas(contenttype.Set{}) {
		switch v, ok := conversionVersions[s.Set()]; {
		case !ok:
			before = append(before, s)
		case v > 0:
			before = append(before, s.Through(v))
		}
	}
	old := openAt(t, path, before)
	if err := old.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, stmt := range seeded {
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("%s: %w", stmt, err)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	kept := map[string]tableFacts{}
	for _, table := range rebuilt {
		kept[table] = factsOf(t, old, table, nil)
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}
	return kept
}

// tryRolledBack runs stmts in one Write and rolls it back. It answers the
// bare errRolledBack when every statement ran with no foreign key broken,
// and the last statement's own error when lastMayFail.
func tryRolledBack(t *testing.T, lib *storage.DB[storage.Library], stmts []string, lastMayFail bool) error {
	t.Helper()
	return lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for i, stmt := range stmts {
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				if lastMayFail && i == len(stmts)-1 {
					return err
				}
				return fmt.Errorf("%s: %w", stmt, errors.Join(errRolledBack, err))
			}
		}
		var broken int
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM pragma_foreign_key_check").Scan(&broken); err != nil || broken > 0 {
			return fmt.Errorf("%d foreign keys broken (%w)", broken, errors.Join(errRolledBack, err))
		}
		return errRolledBack
	})
}

type tableFacts struct {
	columns, indexes, rows []string
}

func (was tableFacts) keptIn(t *testing.T, table string, now tableFacts) {
	t.Helper()
	for _, c := range was.columns {
		if !slices.Contains(now.columns, c) {
			t.Errorf("%s lost its column %s", table, c)
		}
	}
	for _, i := range was.indexes {
		if !slices.Contains(now.indexes, i) {
			t.Errorf("%s lost its index %s", table, i)
		}
	}
	if !slices.Equal(now.rows, was.rows) {
		t.Errorf("%s rows = %q after the migrations, want %q", table, now.rows, was.rows)
	}
}

// factsOf reads table's columns, its indexes and its rows over columns,
// or over every column when columns is nil.
func factsOf(t *testing.T, lib *storage.DB[storage.Library], table string, columns []string) tableFacts {
	t.Helper()
	ctx := t.Context()
	var f tableFacts
	f.columns = strings.Split(scalar(t, lib, "SELECT group_concat(name, ',') FROM (SELECT name FROM pragma_table_xinfo(?) ORDER BY cid)", table), ",")
	f.indexes = strings.Split(scalar(t, lib, `SELECT group_concat(d, '|') FROM (SELECT CASE l.origin WHEN 'c' THEN s.sql
		ELSE l.origin || ' ' || (SELECT group_concat(name, ',') FROM pragma_index_info(l.name)) END AS d
		FROM pragma_index_list(?) l LEFT JOIN sqlite_schema s ON s.name = l.name ORDER BY d)`, table), "|")
	if columns == nil {
		columns = f.columns
	}
	quoted := make([]string, len(columns))
	for i, c := range columns {
		quoted[i] = "quote(" + c + ")"
	}
	rows, err := lib.QueryContext(ctx, "SELECT "+strings.Join(quoted, " || ' ' || ")+" FROM "+table+" ORDER BY 1")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			t.Fatal(err)
		}
		f.rows = append(f.rows, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return f
}

func scalar(t *testing.T, lib *storage.DB[storage.Library], query string, args ...any) string {
	t.Helper()
	var s string
	if err := lib.QueryRowContext(t.Context(), query, args...).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

func openAt(t *testing.T, path string, schemas []storage.Schema[storage.Library]) *storage.DB[storage.Library] {
	t.Helper()
	lib, err := storage.OpenLibrary(t.Context(), path, schemas, func(context.Context, *storage.Unmigrated, storage.Plan) error { return nil },
		storage.DefaultTimings(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	return lib
}
