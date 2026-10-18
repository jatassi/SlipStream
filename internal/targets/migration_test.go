package targets_test

import (
	"context"
	"log/slog"
	"path/filepath"
	"reflect"
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

func (noFacet) Timings(context.Context, storage.Reader[storage.Library], []works.WorkID) (map[works.ItemID]targets.Timing, error) {
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

func TestResolutionTargetsMigrateToTiersOfTheirQualitiesAtTheFloors(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	open := opener(t, path)
	resolutions := map[string][]string{
		"HD":    {"1080p"},
		"All":   {"2160p", "1080p", "720p"},
		"SD":    {"480p", "576p", "720p"},
		"PAL":   {"576p", "480p"},
		"Small": {"720p", "576p"},
	}
	before := open(targets.Schema().Through(1))
	err := before.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for name, rs := range resolutions {
			if err := seedResolutionTarget(ctx, tx, name, rs, name == "HD"); err != nil {
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
	list := must[[]targets.Target](t)(must[*targets.Targets](t)(targets.Open(ctx, targets.Config{Library: lib, Changed: discard{}})).ListTargets(ctx, identity.Admin{}))
	got := map[string][][]string{}
	for _, target := range list {
		got[target.Name] = keys(target.Spec().Tiers)
		checkMigrated(t, target, target.Name == "HD")
	}
	want := map[string][][]string{
		"HD": {{"hdtv-1080p", "webrip-1080p", "webdl-1080p", "bluray-1080p", "remux-1080p"}},
		"All": {
			{"hdtv-2160p", "webrip-2160p", "webdl-2160p", "bluray-2160p", "remux-2160p"},
			{"hdtv-1080p", "webrip-1080p", "webdl-1080p", "bluray-1080p", "remux-1080p"},
			{"hdtv-720p", "webrip-720p", "webdl-720p", "bluray-720p"},
		},
		"SD":    {{"sdtv", "dvd", "webrip-480p"}, {"hdtv-720p", "webrip-720p", "webdl-720p", "bluray-720p"}},
		"PAL":   {{"sdtv", "dvd"}, {"webrip-480p"}},
		"Small": {{"hdtv-720p", "webrip-720p", "webdl-720p", "bluray-720p"}, {"sdtv", "dvd"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("migrated tiers = %v\nwant %v", got, want)
	}
}

// seedResolutionTarget adds a Target of resolutions rs as targets 00001
// held it, and a Profile Standard of it when held.
func seedResolutionTarget(ctx context.Context, tx *storage.Tx[storage.Library], name string, rs []string, held bool) error {
	id := storage.NewUUID()
	if _, err := tx.ExecContext(ctx, `INSERT INTO targets_target VALUES (?, ?, 1)`, id, name); err != nil {
		return err
	}
	for i, r := range rs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO targets_resolution VALUES (?, ?, ?)`, id, i+1, r); err != nil {
			return err
		}
	}
	if !held {
		return nil
	}
	profile := storage.NewUUID()
	if _, err := tx.ExecContext(ctx, `INSERT INTO targets_profile VALUES (?, 'Standard', 'cinema', 1)`, profile); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO targets_profile_target VALUES (?, 1, ?)`, profile, id)
	return err
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

func TestEveryWorkMigratesToItsTypesDefaultProfileAndNoStarterIsMade(t *testing.T) {
	ctx := t.Context()
	open := opener(t, filepath.Join(t.TempDir(), storage.LibraryFile))
	before := open(targets.Schema().Through(1))
	var films, songs works.WorkID
	err := before.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if err := seedResolutionTarget(ctx, tx, "HD", []string{"1080p"}, true); err != nil {
			return err
		}
		var err error
		if films, err = seedWork(ctx, tx, "cinema", "m1"); err != nil {
			return err
		}
		songs, err = seedWork(ctx, tx, "records", "a1")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := before.Close(); err != nil {
		t.Fatal(err)
	}

	lib := open(targets.Schema())
	t.Cleanup(func() { _ = lib.Close() })
	goals := must[*targets.Targets](t)(targets.Open(ctx, targets.Config{
		Library: lib, Works: described{"cinema": "Cinema", "records": "Records"}, Types: facets("cinema", "records"), Changed: discard{},
	}))
	g := must[targets.Goals](t)(goals.Goals(ctx, identity.Admin{}, films))
	if g.Profile.Name != "Standard" || len(g.Targets) != 1 || g.Targets[0].Name != "HD" || g.Profile.Entries[0].Sourcing != targets.Download {
		t.Fatalf("the cinema Work wants %+v, want Standard's HD, downloaded", g)
	}
	if g := must[targets.Goals](t)(goals.Goals(ctx, identity.Admin{}, songs)); g.Profile.Name != "Records" || !g.Profile.Default || len(g.Targets) != 0 {
		t.Fatalf("the records Work, whose type had no Profile, wants %+v, want Open's default Records with no Target", g)
	}
	list := must[[]targets.Target](t)(goals.ListTargets(ctx, identity.Admin{}))
	if len(list) != 1 {
		t.Fatalf("a library that held a Target has %d after Open, want no starter beside HD", len(list))
	}
}

// seedWork adds a Work of type typ as works 00001 holds it.
func seedWork(ctx context.Context, tx *storage.Tx[storage.Library], typ, key string) (works.WorkID, error) {
	id := works.WorkID{UUID: storage.NewUUID()}
	_, err := tx.ExecContext(ctx, `INSERT INTO works_work VALUES (?, ?, ?, ?, ?, 1, 1)`, id, typ, key, key, key)
	return id, err
}
