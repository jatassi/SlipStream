package targets_test

import (
	"bytes"
	"context"
	"log/slog"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type discard struct{}

// described is works.Works' descriptions by Content Type, only their
// names.
type described map[works.TypeID]string

func (d described) Describe(t works.TypeID) (works.Description, bool) {
	name, ok := d[t]
	return works.Description{Name: name}, ok
}

// facets is a facet answering nothing for each of types.
func facets(types ...works.TypeID) map[works.TypeID]targets.Facet {
	out := map[works.TypeID]targets.Facet{}
	for _, t := range types {
		out[t] = noFacet{}
	}
	return out
}

type noFacet struct{}

func (noFacet) Timings(context.Context, storage.Reader[storage.Library], works.Among[works.WorkID]) (map[works.ItemID]targets.Timing, error) {
	return nil, nil
}

func (noFacet) Originals(context.Context, storage.Reader[storage.Library], works.Among[works.WorkID]) (map[works.WorkID]release.Fact[release.Language], error) {
	return nil, nil
}

func (noFacet) Cuts() []release.Cut { return nil }

func (discard) Publish(context.Context, targets.Changed) {}

// opener opens the library at path through a schema of targets, after
// works.
func opener(t *testing.T, path string) func(s storage.Schema[storage.Library]) *storage.DB[storage.Library] {
	return func(s storage.Schema[storage.Library]) *storage.DB[storage.Library] {
		t.Helper()
		lib, err := storage.OpenLibrary(t.Context(), path, []storage.Schema[storage.Library]{works.Schema(), s},
			func(context.Context, *storage.Unmigrated, storage.Plan) error { return nil }, storage.DefaultTimings(), slog.New(slog.DiscardHandler))
		if err != nil {
			t.Fatal(err)
		}
		return lib
	}
}

// checkMigrated checks target's cutoff is its top tier, every quality is
// at its floor, and Standard holds it when held.
func checkMigrated(t *testing.T, target targets.Target, held bool) {
	t.Helper()
	if target.Cutoff != 0 {
		t.Errorf("%s: cutoff tier %d, want the top tier", target.Name, target.Cutoff)
	}
	for _, tier := range target.Tiers {
		for _, a := range tier {
			if a.Size != targets.Floor(a.Quality) {
				t.Errorf("%s: %s migrated at %+v, want its floor %+v", target.Name, a.Quality, a.Size, targets.Floor(a.Quality))
			}
		}
	}
	if (len(target.Profiles) == 1 && target.Profiles[0].Name == "Standard") != held {
		t.Errorf("%s: Profiles %+v", target.Name, target.Profiles)
	}
}

// seedWork adds a Work of type typ as works 00001 holds it.
func seedWork(ctx context.Context, tx *storage.Tx[storage.Library], typ, key string) (works.WorkID, error) {
	id := works.WorkID{UUID: storage.NewUUID()}
	_, err := tx.ExecContext(ctx, `INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, ?, ?, ?, ?, 1, 1)`, id, typ, key, key, key)
	return id, err
}

func TestTargetNamesTwinnedByCaseAreRenamedOnceAtStartUntilEachFoldsApart(t *testing.T) {
	ctx := t.Context()
	open := opener(t, filepath.Join(t.TempDir(), storage.LibraryFile))
	before := open(targets.Schema().Through(3))
	names := []string{"4K", "4k", "4k 2", "Straße"}
	err := before.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, name := range names {
			if _, err := tx.ExecContext(ctx, `INSERT INTO targets_target (id, name, created_at, cutoff_tier) VALUES (?, ?, 1, 0)`, storage.NewUUID(), name); err != nil {
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

	lib := open(targets.Schema())
	t.Cleanup(func() { _ = lib.Close() })
	var logged bytes.Buffer
	cfg := targets.Config{Library: lib, Changed: discard{}, Log: slog.New(slog.NewTextHandler(&logged, nil))}
	listed := func() []string {
		goals := must[*targets.Targets](t)(targets.Open(ctx, cfg))
		var got []string
		for _, target := range must[[]targets.Target](t)(goals.ListTargets(ctx, identity.Admin{})) {
			got = append(got, target.Name)
		}
		slices.Sort(got)
		return got
	}
	want := []string{"1080p", "4K", "4k 2", "4k 3", "Straße"}
	if got := listed(); !reflect.DeepEqual(got, want) {
		t.Fatalf("the Targets after start are %q, want %q", got, want)
	}
	if got := listed(); !reflect.DeepEqual(got, want) || strings.Count(logged.String(), "renamed a Target") != 1 {
		t.Fatalf("a second start leaves %q and logged\n%s\nwant %q and one rename", got, logged.String(), want)
	}
	if !strings.Contains(logged.String(), "from=4k to=\"4k 3\"") {
		t.Fatalf("the log is\n%s\nwant the rename of 4k to 4k 3", logged.String())
	}
	err = lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO targets_target (id, name, name_folded, created_at, cutoff_tier) VALUES (?, 'STRASSE', 'strasse', 1, 0)`, storage.NewUUID())
		return err
	})
	if err == nil {
		t.Fatal("the schema took a second Target folded as strasse")
	}
}
