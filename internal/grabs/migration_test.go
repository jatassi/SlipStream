package grabs_test

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type fixtureNamed struct{}

func (fixtureNamed) Describe(works.TypeID) (works.Description, bool) {
	return works.Description{Name: "Fixtures"}, true
}

func TestAGrabsTargetsSurviveTheMigrationAndGoWithTheirTarget(t *testing.T) {
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
	before := open(nodes.Schema(), works.Schema(), targets.Schema().Through(1), roots.Schema().Through(1), indexers.Schema().Through(2), grabs.Schema().Through(1), files.Schema().Through(1))
	target, grab := storage.NewUUID(), storage.NewUUID()
	err := before.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		node, indexer, work, item := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
		for _, stmt := range []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO targets_target VALUES (?, '1080p', 0)`, []any{target}},
			{`INSERT INTO targets_resolution VALUES (?, 1, '1080p')`, []any{target}},
			{`INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'home', 0)`, []any{node}},
			{`INSERT INTO indexers_indexer VALUES (?, 'Fixture Site', 'torznab', 'https://site.test/api', NULL, ?, '{}', 0)`, []any{indexer, node}},
			{`INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'fixture', 'w1', 'Orchard', 'Orchard', 0, 0)`, []any{work}},
			{`INSERT INTO works_item (id, work_id, key, label, position) VALUES (?, ?, 'i1', 'Orchard', 1)`, []any{item, work}},
			{`INSERT INTO grabs_grab (id, work_id, item_id, edition, source, indexer_id, node_id, release_title, release_size, release_link, stage,
				created_at, updated_at) VALUES (?, ?, ?, 'standard', 'search', ?, ?, 'Orchard.1080p.WEB-DL', 1, 'https://site.test/dl', 'fetching', 0, 0)`,
				[]any{grab, work, item, indexer, node}},
			{`INSERT INTO grabs_target VALUES (?, 0, ?)`, []any{grab, target}},
			{`INSERT INTO grabs_fill VALUES (?, 0, ?)`, []any{grab, target}},
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

	lib := open(nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), downloadclients.Schema(), grabs.Schema(), indexers.Schema(), files.Schema())
	t.Cleanup(func() { _ = lib.Close() })
	g := grabs.New(grabs.Config{Library: lib})
	held := func() []targets.TargetID {
		t.Helper()
		v, err := g.Get(ctx, identity.Admin{}, grabs.GrabID{UUID: grab})
		if err != nil {
			t.Fatal(err)
		}
		return v.Targets
	}
	if got := held(); len(got) != 1 || got[0].UUID != target {
		t.Fatalf("after the migration the Grab is for %v, want the Target %s", got, target)
	}
	goals, err := targets.Open(ctx, targets.Config{
		Library: lib, Works: fixtureNamed{}, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: contenttypetest.New()}, Changed: discard[targets.Changed]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := goals.DeleteTarget(ctx, identity.Admin{}, targets.TargetID{UUID: target}); err != nil {
		t.Fatalf("delete a Target only a Grab's history names: %v", err)
	}
	if got := held(); len(got) != 0 {
		t.Fatalf("after the Target's delete the Grab is for %v, want none", got)
	}
}
