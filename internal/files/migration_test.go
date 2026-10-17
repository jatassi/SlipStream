package files_test

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
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
	type seeded struct {
		title string // empty for a file no Grab brought
		probe files.Probe
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
		add := func(title string, probe files.Probe) error {
			n++
			item, file := storage.NewUUID(), storage.NewUUID()
			if _, err := tx.ExecContext(ctx, `INSERT INTO works_item (id, work_id, key, label, position) VALUES (?, ?, ?, 'Item', ?)`, item, work, fmt.Sprint(n), n); err != nil {
				return err
			}
			var grab any
			if title != "" {
				id := storage.NewUUID()
				_, err := tx.ExecContext(ctx, `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, indexer_id, node_id, release_title, release_size,
					release_link, stage, created_at, updated_at) VALUES (?, ?, ?, 'standard', 'search', ?, ?, ?, 1, 'https://site.test/dl', 'fetching', 0, 0)`,
					id, work, item, indexer, node, title)
				if err != nil {
					return err
				}
				grab = id
			}
			probeJSON := fmt.Sprintf(`{"width":%d,"height":%d}`, probe.Width, probe.Height)
			if _, err := tx.ExecContext(ctx, `INSERT INTO files_file VALUES (?, ?, ?, 'standard', ?, ?, 1, ?, ?)`, file, work, item, root, file.String(), probeJSON, grab); err != nil {
				return err
			}
			want[file.String()] = seeded{title, probe}
			return nil
		}
		for _, title := range append(corpus, "") {
			for _, f := range frames {
				if err := add(title, f); err != nil {
					return err
				}
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
	checked := 0
	err = lib.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		rows, err := r.QueryContext(ctx, `SELECT path, quality FROM files_file ORDER BY path`)
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var path, got string
			if err := rows.Scan(&path, &got); err != nil {
				return err
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
	if checked != len(want) {
		t.Fatalf("checked %d files, want %d", checked, len(want))
	}
}
