package roots_test

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func opener(t *testing.T) func(schemas ...storage.Schema[storage.Library]) *storage.DB[storage.Library] {
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	return func(schemas ...storage.Schema[storage.Library]) *storage.DB[storage.Library] {
		t.Helper()
		lib, err := storage.OpenLibrary(t.Context(), path, schemas, func(context.Context, *storage.Unmigrated, storage.Plan) error { return nil },
			storage.DefaultTimings(), slog.New(slog.DiscardHandler))
		if err != nil {
			t.Fatal(err)
		}
		return lib
	}
}

type s09 struct {
	alpha, beta     roots.ID
	one, two, three works.WorkID
}

// seedS09 is a library of one root per Content Type as S09 (Walking
// skeleton) left it: a root of types alpha and beta, a Work of each type
// plus one of a type with no root.
func seedS09(t *testing.T) (*storage.DB[storage.Library], s09) {
	t.Helper()
	open := opener(t)
	before := open(works.Schema(), roots.Schema().Through(1))
	at := s09{
		alpha: roots.ID{UUID: storage.NewUUID()}, beta: roots.ID{UUID: storage.NewUUID()},
		one: works.WorkID{UUID: storage.NewUUID()}, two: works.WorkID{UUID: storage.NewUUID()}, three: works.WorkID{UUID: storage.NewUUID()},
	}
	node := storage.NewUUID()
	err := before.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, stmt := range []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO roots_root VALUES (?, 'alpha', ?, '/media/alpha')`, []any{at.alpha, node}},
			{`INSERT INTO roots_root VALUES (?, 'beta', ?, '/media/beta')`, []any{at.beta, node}},
			{`INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'alpha', 'a1', 'Orchard', 'Orchard', 0, 0)`, []any{at.one}},
			{`INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'beta', 'b1', 'Lanterns', 'Lanterns', 0, 0)`, []any{at.two}},
			{`INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'gamma', 'g1', 'Tide', 'Tide', 0, 0)`, []any{at.three}},
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
	lib := open(works.Schema(), roots.Schema())
	t.Cleanup(func() { _ = lib.Close() })
	return lib, at
}

func TestTheRebuildKeepsIdsAndMakesEachRootItsTypesDefault(t *testing.T) {
	lib, at := seedS09(t)
	all, err := roots.All(t.Context(), lib)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("roots after the rebuild = %+v, want the two S09 roots", all)
	}
	for i, want := range []struct {
		id   roots.ID
		typ  works.TypeID
		path string
	}{{at.alpha, "alpha", "/media/alpha"}, {at.beta, "beta", "/media/beta"}} {
		if r := all[i]; r.ID != want.id || r.ContentType != want.typ || r.Path != want.path || !r.Default {
			t.Errorf("root %d = %+v, want %s's default %s at %s", i, r, want.typ, want.id, want.path)
		}
	}
	err = lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO roots_root (id, content_type, node_id, path, is_default) VALUES (?, 'alpha', ?, '/b', 1)`,
			storage.NewUUID(), storage.NewUUID())
		return err
	})
	if err == nil {
		t.Fatal("a second default alpha root was inserted, want the index to refuse it")
	}
}

func TestEveryWorkTakesItsTypesRoot(t *testing.T) {
	lib, at := seedS09(t)
	for _, want := range []struct {
		work works.WorkID
		root roots.ID
	}{{at.one, at.alpha}, {at.two, at.beta}} {
		r, err := roots.OfWork(t.Context(), lib, want.work)
		if err != nil || r.ID != want.root {
			t.Errorf("OfWork(%s) = %s, %v; want %s", want.work, r.ID, err, want.root)
		}
	}
	if r, err := roots.OfWork(t.Context(), lib, at.three); !errors.Is(err, roots.ErrNoRoot) {
		t.Errorf("OfWork of a Work whose type has no root = %+v, %v; want ErrNoRoot", r, err)
	}
}
