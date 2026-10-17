package files_test

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// corpus is release titles a library may hold Grabs of, v1's bare ts
// false matches among them.
var corpus = []string{
	"The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW",
	"The.Quiet.Orchard.2021.720p.WEB-DL.H264-FKW",
	"The Quiet Orchard 2021 1080p WEBDL DDP5.1",
	"The.Quiet.Orchard.2021.2160p.WEB.H265",
	"The.Quiet.Orchard.2021.1080p.WEBRip.x264",
	"The Quiet Orchard (2021) [720p] [WEB-Rip]",
	"The.Quiet.Orchard.2021.1080p.BluRay.x264",
	"The.Quiet.Orchard.2021.720p.Blu-Ray.DTS",
	"The.Quiet.Orchard.2021.2160p.UHD.BluRay.REMUX",
	"The.Quiet.Orchard.2021.1080p.BDREMUX",
	"The.Quiet.Orchard.2021.720p.HDTV.x264",
	"The.Quiet.Orchard.S01E01.SDTV.XviD",
	"The.Quiet.Orchard.2021.DVDRip.XviD",
	"The.Quiet.Orchard.2021.576p.DVD",
	"The.Quiet.Orchard.2021.HDCAM.x264",
	"The.Quiet.Orchard.2021.TS.XviD",
	"The.Quiet.Orchard.2021.1080p.TELESYNC",
	"Pirates.of.the.Orchard.2021.1080p.WEB-DL",
	"Tsunami.Orchard.2021.1080p.WEBRip",
	"T.S.Orchard.2021.1080p.BluRay",
	"Its.Always.Orchard.S01E01.720p",
	"The.Orchard.Tapes.2021.2160p.WEB-DL",
	"The.Cambridge.Orchard.2021.1080p.WEBRip",
	"The.Quiet.Orchard.2021.1080p.WEB-DL.TrueHD.Atmos-TSG",
	"Orchard’s.TS",
	"Orchard/ts\\cam",
	"Örchard.WEB-DL",
	"The Quiet Orchard 2021",
}

var frames = []files.Probe{{Width: 3840, Height: 2160}, {Width: 3840, Height: 1600}, {Width: 1920, Height: 1080}, {Width: 1280, Height: 720}, {Width: 720, Height: 480}}

func TestExistingFilesMigrateToTheQualityTheirGrabAndProbeGive(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	open := func(schemas ...storage.Schema[storage.Library]) *storage.DB[storage.Library] {
		t.Helper()
		lib, err := storage.OpenLibrary(ctx, path, schemas, func(context.Context, *storage.Unmigrated, storage.Plan) error { return nil },
			storage.DefaultTimings(), slog.New(slog.DiscardHandler))
		if err != nil {
			t.Fatal(err)
		}
		return lib
	}
	want := map[string]seeded{}
	before := open(nodes.Schema(), works.Schema(), targets.Schema().Through(1), roots.Schema(), indexers.Schema(), grabs.Schema().Through(1), files.Schema().Through(1))
	err := before.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		node, indexer, work, root := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
		for _, stmt := range []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO nodes_node VALUES (?, randomblob(32), 'home', 0)`, []any{node}},
			{`INSERT INTO indexers_indexer VALUES (?, 'Fixture Site', 'torznab', 'https://site.test/api', NULL, ?, '{}', 0)`, []any{indexer, node}},
			{`INSERT INTO works_work VALUES (?, 'fixture', 'w1', 'Orchard', 'Orchard', 0, 0)`, []any{work}},
			{`INSERT INTO roots_root VALUES (?, 'fixture', ?, '/media')`, []any{root, node}},
		} {
			if _, err := tx.ExecContext(ctx, stmt.sql, stmt.args...); err != nil {
				return fmt.Errorf("%s: %w", stmt.sql, err)
			}
		}
		n := 0
		for _, title := range append(corpus, "") {
			for _, f := range frames {
				n++
				file, err := seedFile(ctx, tx, n, title, f, seeds{node, indexer, work, root})
				if err != nil {
					return err
				}
				want[file] = seeded{title, f}
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

	lib := open(nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), indexers.Schema(), grabs.Schema(), files.Schema())
	t.Cleanup(func() { _ = lib.Close() })
	checked := checkQualities(t, lib, want)
	if checked != len(want) {
		t.Fatalf("checked %d files, want %d", checked, len(want))
	}
	distinct(t, lib, `SELECT DISTINCT edition FROM grabs_grab`, "any")
}

type seeded struct {
	title string // empty for a file no Grab brought
	probe files.Probe
}

type seeds struct {
	node, indexer, work, root storage.UUID
}

// seedFile adds the nth Item of the Work and a file of it probed at probe,
// brought by a Grab of title unless title is empty, and returns the
// file's path.
func seedFile(ctx context.Context, tx *storage.Tx[storage.Library], n int, title string, probe files.Probe, at seeds) (string, error) {
	item, file := storage.NewUUID(), storage.NewUUID()
	if _, err := tx.ExecContext(ctx, `INSERT INTO works_item (id, work_id, key, label, position) VALUES (?, ?, ?, 'Item', ?)`, item, at.work, strconv.Itoa(n), n); err != nil {
		return "", err
	}
	var grab any
	if title != "" {
		id := storage.NewUUID()
		_, err := tx.ExecContext(ctx, `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, indexer_id, node_id, release_title, release_size,
			release_link, stage, created_at, updated_at) VALUES (?, ?, ?, 'standard', 'search', ?, ?, ?, 1, 'https://site.test/dl', 'fetching', 0, 0)`,
			id, at.work, item, at.indexer, at.node, title)
		if err != nil {
			return "", err
		}
		grab = id
	}
	probeJSON := fmt.Sprintf(`{"width":%d,"height":%d}`, probe.Width, probe.Height)
	if _, err := tx.ExecContext(ctx, `INSERT INTO files_file VALUES (?, ?, ?, 'standard', ?, ?, 1, ?, ?)`, file, at.work, item, at.root, file.String(), probeJSON, grab); err != nil {
		return "", err
	}
	return file.String(), nil
}

// checkQualities checks each migrated file's quality is what release.File
// reads from its Grab's title and its probe, and returns how many it read.
func checkQualities(t *testing.T, lib *storage.DB[storage.Library], want map[string]seeded) int {
	t.Helper()
	checked := 0
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		rows, err := r.QueryContext(ctx, `SELECT path, quality FROM files_file ORDER BY path`)
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var path, got string
			if err := rows.Scan(&path, &got); err != nil {
				return fmt.Errorf("read a file: %w", err)
			}
			s := want[path]
			q, _ := release.File(s.title, nil, release.ResolutionOf(s.probe.Width, s.probe.Height)).Quality.Get()
			if got != q.String() {
				t.Errorf("a %dx%d file of %q migrated as %s, release.File says %s", s.probe.Width, s.probe.Height, s.title, got, q)
			}
			checked++
		}
		return rows.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	return checked
}

func TestExistingFilesTakeTheFirstTargetTheyFillInProfileOrderAsTheirLabel(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	open := func(schemas ...storage.Schema[storage.Library]) *storage.DB[storage.Library] {
		t.Helper()
		lib, err := storage.OpenLibrary(ctx, path, schemas, func(context.Context, *storage.Unmigrated, storage.Plan) error { return nil },
			storage.DefaultTimings(), slog.New(slog.DiscardHandler))
		if err != nil {
			t.Fatal(err)
		}
		return lib
	}
	before := open(nodes.Schema(), works.Schema(), targets.Schema().Through(1), roots.Schema(), indexers.Schema(), grabs.Schema().Through(1), files.Schema().Through(1))
	work, root, profile := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	small, large := storage.NewUUID(), storage.NewUUID()
	both, second, none := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	err := before.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		type stmt struct {
			sql  string
			args []any
		}
		stmts := []stmt{
			{`INSERT INTO nodes_node VALUES (?, randomblob(32), 'home', 0)`, []any{root}},
			{`INSERT INTO works_work VALUES (?, 'fixture', 'w1', 'Orchard', 'Orchard', 0, 0)`, []any{work}},
			{`INSERT INTO roots_root VALUES (?, 'fixture', ?, '/media')`, []any{root, root}},
			{`INSERT INTO targets_target VALUES (?, 'Small', 1)`, []any{small}},
			{`INSERT INTO targets_resolution VALUES (?, 1, '720p')`, []any{small}},
			{`INSERT INTO targets_target VALUES (?, 'Large', 1)`, []any{large}},
			{`INSERT INTO targets_resolution VALUES (?, 1, '1080p')`, []any{large}},
			{`INSERT INTO targets_profile VALUES (?, 'Standard', 'fixture', 1)`, []any{profile}},
			{`INSERT INTO targets_profile_target VALUES (?, 1, ?)`, []any{profile, large}},
			{`INSERT INTO targets_profile_target VALUES (?, 2, ?)`, []any{profile, small}},
		}
		for i, f := range []storage.UUID{both, second, none} {
			item := storage.NewUUID()
			stmts = append(stmts,
				stmt{`INSERT INTO works_item (id, work_id, key, label, position) VALUES (?, ?, ?, 'Item', ?)`, []any{item, work, strconv.Itoa(i), i + 1}},
				stmt{`INSERT INTO files_file VALUES (?, ?, ?, 'standard', ?, ?, 1, '{"width":1920,"height":1080}', NULL)`, []any{f, work, item, root, f.String()}},
			)
			fills := map[storage.UUID][]storage.UUID{both: {small, large}, second: {small}}[f]
			for _, target := range fills {
				stmts = append(stmts, stmt{`INSERT INTO files_fill VALUES (?, ?, ?, 'standard')`, []any{f, target, item}})
			}
		}
		for _, stmt := range stmts {
			if _, err := tx.ExecContext(ctx, stmt.sql, stmt.args...); err != nil {
				return fmt.Errorf("%s: %w", stmt.sql, err)
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

	lib := open(nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), indexers.Schema(), grabs.Schema(), files.Schema())
	t.Cleanup(func() { _ = lib.Close() })
	got := map[string]string{}
	rows, err := lib.QueryContext(ctx, `SELECT path, label, extra FROM files_file ORDER BY path`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var path, label string
		var extra int
		if err := rows.Scan(&path, &label, &extra); err != nil {
			t.Fatal(err)
		}
		got[path] = fmt.Sprintf("%s %d", label, extra)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{both.String(): "Large 0", second.String(): "Small 0", none.String(): " 0"}
	if !maps.Equal(got, want) {
		t.Fatalf("migrated labels and extras = %v, want %v: the first filled Target in Profile order, and no Kept file", got, want)
	}
	distinct(t, lib, `SELECT DISTINCT cut FROM files_file`, "standard")
	distinct(t, lib, `SELECT DISTINCT edition FROM files_fill`, "any")
}

// distinct checks the one value query reads in lib is want.
func distinct(t *testing.T, lib *storage.DB[storage.Library], query, want string) {
	t.Helper()
	var got []string
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		rows, err := r.QueryContext(ctx, query)
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var v string
			if err := rows.Scan(&v); err != nil {
				return fmt.Errorf("read a value: %w", err)
			}
			got = append(got, v)
		}
		return rows.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("%s = %v after the migration, want only %s", query, got, want)
	}
}
