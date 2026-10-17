package targets_test

import (
	"context"
	"log/slog"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
)

type discard struct{}

func (discard) Publish(context.Context, targets.Changed) {}

func TestResolutionTargetsMigrateToTiersOfTheirQualitiesAtTheFloors(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	open := func(s storage.Schema[storage.Library]) *storage.DB[storage.Library] {
		t.Helper()
		lib, err := storage.OpenLibrary(ctx, path, []storage.Schema[storage.Library]{s},
			func(context.Context, *storage.Unmigrated, storage.Plan) error { return nil }, storage.DefaultTimings(), slog.New(slog.DiscardHandler))
		if err != nil {
			t.Fatal(err)
		}
		return lib
	}
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
	list := must[[]targets.Target](t)(targets.New(targets.Config{Library: lib, Changed: discard{}}).ListTargets(ctx, identity.Admin{}))
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
	if _, err := tx.ExecContext(ctx, `INSERT INTO targets_profile VALUES (?, 'Standard', 'movie', 1)`, profile); err != nil {
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
