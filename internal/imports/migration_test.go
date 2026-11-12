package imports

import (
	"context"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

func openAt(t *testing.T, path string, schemas ...storage.Schema[storage.Library]) *storage.DB[storage.Library] {
	t.Helper()
	lib, err := storage.OpenLibrary(t.Context(), path, schemas, func(context.Context, *storage.Unmigrated, storage.Plan) error { return nil },
		storage.DefaultTimings(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	return lib
}

func exec(t *testing.T, lib *storage.DB[storage.Library], stmts ...[]any) {
	t.Helper()
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, s := range stmts {
			sql, ok := s[0].(string)
			if !ok {
				t.Fatalf("statement %v starts with no SQL", s)
			}
			if _, err := tx.ExecContext(ctx, sql, s[1:]...); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestTheUpgradeCarriesEveryPendingImportIntoARootImport(t *testing.T) {
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	before := openAt(t, path, works.Schema(), roots.Schema(), Schema().Through(2))
	root, work := storage.NewUUID(), storage.NewUUID()
	first, second := works.ItemID{UUID: storage.NewUUID()}, works.ItemID{UUID: storage.NewUUID()}
	brazil, severance, mixed := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	const pendingFile = `INSERT INTO imports_pending_file (pending_id, path, size, modified_at, node_file_id, node_volume_id, probe, quality, cut, items, reason)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'standard', ?, ?)`
	probe := `{"version":1,"container":"matroska,webm","video":{"codec":"hevc","width":1920,"height":1080}}`
	reasons := []string{
		`{"code":"several","title":"Brazil","year":1985,"count":2}`,
		`{"code":"no_match","title":"Brazil","year":1985}`,
		`{"code":"no_year","title":"Brazil"}`,
		`{"code":"elsewhere","title":"Brazil","year":1985,"root":"` + root.String() + `"}`,
		`{"code":"unlisted","title":"Brazil","year":1985}`,
		`{"code":"no_number"}`,
		`{"code":"no_such_item","names":"S03E14"}`,
		`{"code":"unreadable","line":"moov atom not found"}`,
		`{"code":"left"}`,
		`{"code":"no_rule"}`,
	}
	stmts := [][]any{
		{`INSERT INTO roots_root (id, content_type, node_id, path, is_default, place) VALUES (?, 'movie', ?, '/media/movies', 1, 1)`, root, storage.NewUUID()},
		{`INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'movie', 'w1', 'Brazil', 'Brazil', 1, 1)`, work},
		{`INSERT INTO works_item (id, work_id, key, label, position) VALUES (?, ?, 'i1', 'One', 1)`, first, work},
		{`INSERT INTO works_item (id, work_id, key, label, position) VALUES (?, ?, 'i2', 'Two', 2)`, second, work},
		{`INSERT INTO imports_pending (id, root_id, folder, content_type, hint, suggestion, found_at, ignored_at) VALUES (?, ?, 'Brazil (1985)', 'movie', '{"Title":"Brazil","Year":1985}', '{"key":"w1","label":"Brazil","year":1985,"poster":{}}', 1700000000000, 1700000100000)`, brazil, root},
		{`INSERT INTO imports_pending (id, root_id, folder, content_type, hint, found_at) VALUES (?, ?, 'Severance', 'movie', '{"Title":"Severance"}', 1700000000000)`, severance, root},
		{`INSERT INTO imports_pending (id, root_id, folder, content_type, hint, found_at) VALUES (?, ?, 'Mixed', 'movie', '{"Title":"Mixed"}', 1700000000000)`, mixed, root},
		{pendingFile, severance, "Severance/S01E01E02.mkv", 10, nil, nil, nil, probe, "bluray-1080p", "[]", `{"code":"several_items"}`},
		{pendingFile, mixed, "Mixed/b.mkv", 10, nil, nil, nil, probe, "bluray-1080p", "[]", `{"code":"several_items"}`},
		{pendingFile, mixed, "Mixed/c.mkv", 10, nil, nil, nil, probe, "bluray-1080p", "[]", `{"code":"no_number"}`},
	}
	for i, r := range reasons {
		items := "[]"
		if i == 0 {
			items = `["` + second.String() + `","` + first.String() + `"]`
		}
		var probed, quality any = probe, "bluray-1080p"
		if i == 7 {
			probed, quality = nil, nil
		}
		stmts = append(stmts, []any{pendingFile, brazil, "Brazil (1985)/" + string(rune('a'+i)) + ".mkv", int64(i + 1), int64(1700000000000 + i), "f" + string(rune('a'+i)), "vol", probed, quality, items, r})
	}
	exec(t, before, stmts...)
	if err := before.Close(); err != nil {
		t.Fatal(err)
	}
	lib := openAt(t, path, works.Schema(), roots.Schema(), Schema())
	t.Cleanup(func() { _ = lib.Close() })

	var got []Import
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = unfinishedRootImports(ctx, r)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !slices.ContainsFunc(got, func(im Import) bool { return im.ID.UUID == brazil }) || !slices.ContainsFunc(got, func(im Import) bool { return im.ID.UUID == mixed }) {
		t.Fatalf("after the upgrade the root Imports are %+v, want Brazil's and Mixed's; Severance held only a several_items file", got)
	}
	b := got[slices.IndexFunc(got, func(im Import) bool { return im.ID.UUID == brazil })]
	m := got[slices.IndexFunc(got, func(im Import) bool { return im.ID.UUID == mixed })]
	if b.Source != FromRoot || b.Root != (roots.ID{UUID: root}) || b.Folder != "Brazil (1985)" || b.ContentType != "movie" ||
		b.Hint.Title != "Brazil" || b.Hint.Year != 1985 || b.Suggestion == nil || b.Suggestion.Key != "w1" ||
		!b.Created.Time().Equal(time.UnixMicro(1700000000000)) || !b.Ignored.Time().Equal(time.UnixMicro(1700000100000)) || !b.Finished.IsZero() || b.Stage() != StageWaiting {
		t.Errorf("Brazil's root Import = %+v, want its entry's root, folder, hint, suggestion, times, unfinished and waiting", b)
	}
	checkCarried(t, b.Files, reasons)
	if items := b.Files[0].Items; !slices.Equal(items, []works.ItemID{second, first}) {
		t.Errorf("the first file's Items = %v, want the stored dashed ids as Item ids in their order", items)
	}
	if len(m.Files) != 1 || m.Files[0].Path != "Mixed/c.mkv" || m.Files[0].Position != 0 || m.Files[0].Scan().Code != scanning.NoNumber {
		t.Errorf("Mixed's files = %+v, want only the no_number file, at position 0", m.Files)
	}
	set, err := db.New(lib).GetSettings(t.Context())
	if err != nil || set.VideoExtensions != nil || set.SmallestVideo != nil || set.SampleShorter != nil {
		t.Errorf("the Importing settings after the upgrade = %+v, %v; want each at its default", set, err)
	}
}

// checkCarried checks fs, the files of the entry with a file at each of
// reasons, kept each at its position with its stored record.
func checkCarried(t *testing.T, fs []File, reasons []string) {
	t.Helper()
	if len(fs) != len(reasons) {
		t.Fatalf("the root Import has %d files, want %d", len(fs), len(reasons))
	}
	for i, f := range fs {
		name := string(rune('a'+i)) + ".mkv"
		var want scanning.Reason
		r := reasons[i]
		if err := json.Unmarshal([]byte(r), &want); err != nil {
			t.Fatal(err)
		}
		if f.Position != i || f.Path != "Brazil (1985)/"+name || f.ReleaseName != name || f.SourceName != name || f.Size != int64(i+1) ||
			f.Outcome != Waiting || f.Reason == nil || f.Reason.Code != ReasonCode(want.Code) || f.Scan() != want ||
			f.Cut != release.Standard || f.Seen == nil || !f.Seen.Modified.Equal(time.UnixMicro(int64(1700000000000+i))) || f.Seen.ID != (storing.FileID{Volume: "vol", File: "f" + name[:1]}) {
			t.Errorf("file %d = %+v, want position %d, its path, base name, size and the reason %s waiting", i, f, i, r)
		}
		if (i == 7) != (f.Probe == nil) || (i == 7) != (f.Quality == 0) {
			t.Errorf("file %d probe %+v quality %q, want none only for the unreadable file", i, f.Probe, f.Quality)
		}
	}
}

func TestTheUpgradeLeavesRootsAlreadySeenReadyForTheNightlyScan(t *testing.T) {
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	open := func(schemas ...storage.Schema[storage.Library]) *storage.DB[storage.Library] {
		t.Helper()
		lib, err := storage.OpenLibrary(t.Context(), path, schemas, func(context.Context, *storage.Unmigrated, storage.Plan) error { return nil },
			storage.DefaultTimings(), slog.New(slog.DiscardHandler))
		if err != nil {
			t.Fatal(err)
		}
		return lib
	}
	seen, unseen := roots.ID{UUID: storage.NewUUID()}, roots.ID{UUID: storage.NewUUID()}
	before := open(works.Schema(), roots.Schema())
	err := before.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		node := storage.NewUUID()
		for _, stmt := range []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO roots_root (id, content_type, node_id, path, is_default, place) VALUES (?, 'movie', ?, '/media/movies', 1, 2)`, []any{seen, node}},
			{`INSERT INTO roots_root (id, content_type, node_id, path, is_default, place) VALUES (?, 'series', ?, '/media/series', 1, 1)`, []any{unseen, node}},
			{`INSERT INTO roots_seen (root_id, place, volume_id) VALUES (?, 2, 'vol')`, []any{seen}},
		} {
			if _, err := tx.ExecContext(ctx, stmt.sql, stmt.args...); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := before.Close(); err != nil {
		t.Fatal(err)
	}
	lib := open(works.Schema(), roots.Schema(), Schema())
	t.Cleanup(func() { _ = lib.Close() })

	rows, err := db.New(lib).ListCompared(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].RootID != seen || rows[0].Place != 2 || !rows[0].ComparedAt.IsZero() {
		t.Fatalf("after the upgrade the scan records %+v, want the root seen ready at place 2, never compared, and none for the root never seen", rows)
	}
}
