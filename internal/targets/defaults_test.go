package targets_test

import (
	"context"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type counted struct{ n atomic.Int32 }

func (c *counted) Publish(context.Context, targets.Changed) { c.n.Add(1) }

func TestOpenGivesAFreshLibraryTheStarterAndADefaultProfilePerType(t *testing.T) {
	ctx := t.Context()
	lib := storagetest.Library(t, works.Schema(), targets.Schema())
	var work works.WorkID
	err := lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		work, err = seedWork(ctx, tx, "cinema", "m1")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	changed := &counted{}
	cfg := targets.Config{Library: lib, Works: described{"cinema": "Cinema", "records": "Records"}, Types: facets("cinema", "records"), Changed: changed}
	goals := must[*targets.Targets](t)(targets.Open(ctx, cfg))

	list := must[[]targets.Target](t)(goals.ListTargets(ctx, identity.Admin{}))
	if len(list) != 1 || list[0].Name != "1080p" {
		t.Fatalf("Targets after Open = %+v, want the starter 1080p alone", list)
	}
	starter := list[0]
	wantTiers := [][]string{{"bluray-1080p", "remux-1080p"}, {"hdtv-1080p", "webrip-1080p", "webdl-1080p"}, {"bluray-720p"}, {"hdtv-720p", "webrip-720p", "webdl-720p"}}
	if got := keys(starter.Spec().Tiers); !reflect.DeepEqual(got, wantTiers) || starter.Cutoff != 0 {
		t.Fatalf("the starter's tiers = %v, cutoff tier %d; want %v final at the top tier", got, starter.Cutoff, wantTiers)
	}
	checkMigrated(t, starter, false)
	if len(starter.Filters.HDR.Values)+len(starter.Filters.Language.Values) != 0 {
		t.Fatalf("the starter has filters %+v, want none", starter.Filters)
	}

	profiles := must[[]targets.Profile](t)(goals.ListProfiles(ctx, identity.Admin{}))
	var got []string
	for _, p := range profiles {
		if !p.Default || !reflect.DeepEqual(p.Entries, []targets.Entry{{Target: starter.ID, Sourcing: targets.Download}}) {
			t.Errorf("Profile %s = %+v, want a default holding the starter", p.Name, p)
		}
		got = append(got, string(p.ContentType)+" "+p.Name)
	}
	if want := []string{"cinema Cinema", "records Records"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Profiles after Open = %v, want %v", got, want)
	}
	if g := must[targets.Goals](t)(goals.Goals(ctx, identity.Admin{}, work)); g.Profile.Name != "Cinema" || g.Profile.Works != 1 {
		t.Fatalf("the Work added before Open wants from %+v, want Cinema", g.Profile)
	}
	if n := changed.n.Load(); n != 1 {
		t.Fatalf("Open published targets.changed %d times, want once", n)
	}

	must[*targets.Targets](t)(targets.Open(ctx, cfg))
	if n := changed.n.Load(); n != 1 {
		t.Fatalf("a second Open published, %d in all; it has nothing to write", n)
	}
	if again := must[[]targets.Profile](t)(goals.ListProfiles(ctx, identity.Admin{})); !reflect.DeepEqual(again, profiles) {
		t.Fatalf("a second Open changed the Profiles to %+v", again)
	}
}

func TestADeletedStarterStaysDeleted(t *testing.T) {
	ctx := t.Context()
	lib := storagetest.Library(t, works.Schema(), targets.Schema())
	goals := must[*targets.Targets](t)(targets.Open(ctx, targets.Config{Library: lib, Changed: discard{}}))
	starter := must[[]targets.Target](t)(goals.ListTargets(ctx, identity.Admin{}))[0]
	if err := goals.DeleteTarget(ctx, identity.Admin{}, starter.ID); err != nil {
		t.Fatal(err)
	}

	goals = must[*targets.Targets](t)(targets.Open(ctx, targets.Config{Library: lib, Works: described{"cinema": "Cinema"}, Types: facets("cinema"), Changed: discard{}}))
	if list := must[[]targets.Target](t)(goals.ListTargets(ctx, identity.Admin{})); len(list) != 0 {
		t.Fatalf("Open after the starter was deleted made %+v", list)
	}
	profiles := must[[]targets.Profile](t)(goals.ListProfiles(ctx, identity.Admin{}))
	if len(profiles) != 1 || profiles[0].Name != "Cinema" || !profiles[0].Default || len(profiles[0].Entries) != 0 {
		t.Fatalf("Profiles = %+v, want a default Cinema holding no Target", profiles)
	}
}

func TestADefaultProfileTakesTheFirstFreeName(t *testing.T) {
	ctx := t.Context()
	lib := storagetest.Library(t, works.Schema(), targets.Schema())
	goals := must[*targets.Targets](t)(targets.Open(ctx, targets.Config{Library: lib, Works: described{"cinema": "Cinema"}, Types: facets("cinema"), Changed: discard{}}))
	if _, err := goals.CreateProfile(ctx, identity.Admin{}, targets.ProfileSpec{Name: "Records", ContentType: "cinema"}); err != nil {
		t.Fatal(err)
	}
	goals = must[*targets.Targets](t)(targets.Open(ctx, targets.Config{
		Library: lib, Works: described{"cinema": "Cinema", "records": "Records"}, Types: facets("cinema", "records"), Changed: discard{},
	}))
	var got []string
	for _, p := range must[[]targets.Profile](t)(goals.ListProfiles(ctx, identity.Admin{})) {
		got = append(got, p.Name)
	}
	if want := []string{"Cinema", "Records", "Records 2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Profiles = %v, want %v", got, want)
	}
}
