package targets_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type ignored[T any] struct{}

func (ignored[T]) Publish(context.Context, T) {}

// filesWorld is targets and files over a fresh library, and the
// fixture's Works.
type filesWorld struct {
	works *works.Works
	fx    *contenttypetest.Fixture
	goals *targets.Targets
	shelf *files.Files
}

func openFilesWorld(t *testing.T) filesWorld {
	t.Helper()
	ctx := t.Context()
	lib := storagetest.Library(t, nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), downloadclients.Schema(), grabs.Schema(), indexers.Schema(), files.Schema())
	fx := contenttypetest.New()
	w := must[*works.Works](t)(works.Open(ctx, works.Config{
		ItemsAdded: works.NoItemsAdded, Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx}, Source: fx.Source(),
		Artwork: fx, Changed: ignored[works.Changed]{}, TypesChanged: ignored[works.ContentTypesChanged]{},
	}))
	goals := must[*targets.Targets](t)(targets.Open(ctx, targets.Config{
		Library: lib, Works: w, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: fx}, Changed: ignored[targets.Changed]{},
	}))
	shelf := must[*files.Files](t)(files.Open(ctx, files.Config{Library: lib, Targets: goals, Grabs: files.NoGrabs{}, Today: time.Now}))
	return filesWorld{works: w, fx: fx, goals: goals, shelf: shelf}
}

// add adds a Work per key wanting from profile.
func (fw filesWorld) add(t *testing.T, profile targets.ProfileID, keys ...works.Key) []works.WorkID {
	t.Helper()
	var out []works.WorkID
	for _, key := range keys {
		fw.fx.Put(key, contenttypetest.Record{Tree: works.Fetched{Label: string(key), Children: []works.Entry{{Key: "c1", Label: "One"}}}})
		tree, _, err := fw.works.Add(t.Context(), identity.Admin{}, contenttypetest.ID, key, works.Adding{
			Also: func(ctx context.Context, tx *storage.Tx[storage.Library], s works.Summary) error {
				return targets.Assign(ctx, tx, s, &profile)
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, tree.ID)
	}
	return out
}

func (fw filesWorld) profileOf(t *testing.T, work works.WorkID) targets.Profile {
	t.Helper()
	return must[targets.Goals](t)(fw.goals.Goals(t.Context(), identity.Admin{}, work)).Profile
}

func TestSplitProfileMovesTheRestToAProfileWithTheOldRowsReusingOneThatHasThem(t *testing.T) {
	ctx := t.Context()
	for _, twin := range []bool{false, true} {
		fw := openFilesWorld(t)
		src := must[targets.Target](t)(fw.goals.CreateTarget(ctx, identity.Admin{}, madeSpec("2160p HDR", nil)))
		made := must[targets.Target](t)(fw.goals.CreateTarget(ctx, identity.Admin{}, madeSpec("720p mobile", ptr(settings()))))
		old := targets.ProfileSpec{Name: "Movies", ContentType: contenttypetest.ID, Entries: []targets.Entry{{Target: src.ID, Sourcing: targets.Download}}, Editions: targets.AnyCut()}
		movies := must[targets.Profile](t)(fw.goals.CreateProfile(ctx, identity.Admin{}, old))
		want := "Movies (without 720p mobile)"
		if twin {
			twinSpec := old
			twinSpec.Name = "Movies as they were"
			want = must[targets.Profile](t)(fw.goals.CreateProfile(ctx, identity.Admin{}, twinSpec)).Name
		}
		ws := fw.add(t, movies.ID, "a", "b", "c")

		next := old
		next.Entries = append(next.Entries, targets.Entry{Target: made.ID, Sourcing: targets.Derive, From: &src.ID})
		split := targets.SplitProfile{Profile: movies.ID, Spec: next, Rest: ws[1:], To: targets.ProfileTarget{NewName: "Movies (without 720p mobile)"}, Default: &movies.ID}
		if _, err := fw.shelf.Change(ctx, identity.Admin{}, split, files.Mode{Unwanted: files.Keep}); err != nil {
			t.Fatalf("twin %v: %v", twin, err)
		}

		if p := fw.profileOf(t, ws[0]); p.ID != movies.ID || len(p.Entries) != 2 || !p.Default {
			t.Errorf("twin %v: the converted Work wants %s with %d rows, default %v, want the default Movies with both", twin, p.Name, len(p.Entries), p.Default)
		}
		rest := fw.profileOf(t, ws[1])
		if fw.profileOf(t, ws[2]).ID != rest.ID || rest.Name != want || !reflect.DeepEqual(rest.Entries, old.Entries) {
			t.Errorf("twin %v: the rest want %s with rows %+v, want %s with Movies' old row", twin, rest.Name, rest.Entries, want)
		}
	}
}
