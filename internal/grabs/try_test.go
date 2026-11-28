package grabs_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/conversions"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/nodes"
	rel "github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

// titled expects a Work's label as its one title, and a name to cover
// the Work's first Item unless covered is false.
type titled struct {
	w       *works.Works
	covered bool
}

func (l *titled) Expect(ctx context.Context, r storage.Reader[storage.Library], work works.WorkID) (indexers.Expect, error) {
	tree, err := l.w.TreeIn(ctx, r, work)
	if err != nil {
		return indexers.Expect{}, err
	}
	covers := indexers.CoversOnly(tree.Items[0].ID)
	if !l.covered {
		covers = func(rel.Name) []works.ItemID { return nil }
	}
	return indexers.Expect{Claim: rel.Claim{Titles: []string{tree.Label}}, Covers: covers}, nil
}

func (*titled) SearchTerms(context.Context, storage.Reader[storage.Library], works.ItemID) (indexers.Terms, error) {
	return indexers.Terms{}, nil
}

func (*titled) Groups(_ context.Context, _ storage.Reader[storage.Library], _ works.WorkID, items []works.ItemID) ([]indexers.Group, error) {
	return indexers.Singles(items, ""), nil
}

func (*titled) GroupTerms(context.Context, storage.Reader[storage.Library], works.WorkID, indexers.Group) (indexers.Terms, error) {
	return indexers.Terms{}, nil
}

func (*titled) Categories() []newznab.ID { return nil }

type ignored[T any] struct{}

func (ignored[T]) Publish(context.Context, T) {}

// tester is a Grabs over a library holding a Work The Attic for each of
// keys, each of one monitored Item, with the fixture type's starter
// Profile and its one Target, 1080p.
func tester(t *testing.T, keys ...works.Key) (*grabs.Grabs, *titled, []works.WorkID) {
	t.Helper()
	ctx := t.Context()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(),
		downloadclients.Schema(), grabs.Schema(), indexers.Schema(), files.Schema(), conversions.Schema(), transfers.LibrarySchema(), imports.Schema())
	fx := contenttypetest.New()
	w, err := works.Open(ctx, works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx}, Source: fx.Source(), Artwork: fx,
		Changed: ignored[works.Changed]{}, TypesChanged: ignored[works.ContentTypesChanged]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	goals, err := targets.Open(ctx, targets.Config{
		Library: lib, Works: w, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: fx}, Changed: ignored[targets.Changed]{}, Formats: formats.DefaultTimings(),
	})
	if err != nil {
		t.Fatal(err)
	}
	shelf, err := files.Open(ctx, files.Config{Library: lib, Targets: goals, Grabs: imports.Cells{Grabs: grabs.GrabCells{InStep: func(role.NodeID) bool { return true }}}, Today: time.Now})
	if err != nil {
		t.Fatal(err)
	}
	var ids []works.WorkID
	for _, key := range keys {
		fx.Put(key, contenttypetest.Record{Tree: works.Fetched{Label: "The Attic", Children: []works.Entry{
			{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}}},
		}}})
		tree, _, err := w.Add(ctx, identity.Admin{}, contenttypetest.ID, key, works.Adding{
			Monitor: true,
			Also: func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
				return targets.Assign(ctx, tx, w, nil)
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, tree.ID)
	}
	claims := &titled{w: w, covered: true}
	g := grabs.New(grabs.Config{
		Library: lib, Works: w, Targets: goals, Files: shelf,
		Terms: map[works.TypeID]indexers.Facet{contenttypetest.ID: claims}, Names: map[works.TypeID]rel.Names{contenttypetest.ID: grabstest.Dated{}},
		Timings: grabs.DefaultTimings(), Today: time.Now, Log: slog.New(slog.DiscardHandler), History: grabstest.History(),
	})
	return g, claims, ids
}

func tried(t *testing.T, g *grabs.Grabs, trial grabs.Trial) grabs.Tried {
	t.Helper()
	out, err := g.Try(t.Context(), identity.Admin{}, trial)
	if err != nil {
		t.Fatal(err)
	}
	return out[0]
}

func TestTryJudgesEveryTargetWithNoHeldFileWhenANameMatchesSeveralWorks(t *testing.T) {
	g, _, _ := tester(t, "w1", "w2")
	got := tried(t, g, grabs.Trial{Names: []string{"The.Attic.2020.1080p.BluRay.x264-GRP"}})
	naming(t, got.Sentence, "Bluray-1080p", "GRP", "The Attic", "1080p")
	if len(got.Works) != 2 || len(got.Covers) != 0 || len(got.Targets) != 1 || !got.Targets[0].WouldFill {
		t.Errorf("works %v, covers %v, targets %+v", got.Works, got.Covers, got.Targets)
	}
}

func TestTryRefusesANameThatCoversNoneOfTheGivenWorksItems(t *testing.T) {
	g, claims, ids := tester(t, "w1")
	claims.covered = false
	got := tried(t, g, grabs.Trial{Names: []string{"The.Attic.2020.1080p.BluRay.x264-GRP"}, Work: &ids[0]})
	naming(t, got.Sentence, "Bluray-1080p", "GRP", "The Attic")
	if rs := got.Targets[0].Reasons; len(rs) != 1 || rs[0].Code != targets.WrongItem || got.Targets[0].WouldFill {
		t.Errorf("1080p = %+v, want wrong_item alone", got.Targets[0])
	}
}

func TestTryReadsANameWithNoGroupOrQuality(t *testing.T) {
	g, _, _ := tester(t, "w1")
	got := tried(t, g, grabs.Trial{Names: []string{"Something"}})
	if _, ok := got.Read.Facts.Quality.Get(); ok || len(got.Works) != 0 || len(got.Targets) != 1 || got.Targets[0].WouldFill || len(got.Targets[0].Reasons) == 0 {
		t.Errorf("read %+v, works %v, targets %+v, want no quality, no match and 1080p refusing it", got.Read.Facts, got.Works, got.Targets)
	}
	naming(t, got.Sentence, "1080p")
}

func TestTryRefusesADraftThatDoesntCompile(t *testing.T) {
	g, _, _ := tester(t, "w1")
	draft := formats.Format{Name: "Broken", Dialect: formats.SlipStream, Conditions: []formats.Condition{
		{Name: "Open", Kind: "ReleaseTitleSpecification", Fields: []byte(`{"value":"(a"}`)},
	}}
	_, err := g.Try(t.Context(), identity.Admin{}, grabs.Trial{Names: []string{"The.Attic.2020.1080p.BluRay-GRP"}, Draft: &draft})
	if e, ok := errors.AsType[*formats.PatternError](err); !ok || e.Condition != 0 {
		t.Errorf("Try = %v, want the pattern error of condition 0", err)
	}
}

func TestTrySpeaksOfTheGivenWorkWhenANameGivesAnotherTitle(t *testing.T) {
	g, _, ids := tester(t, "w1")
	got := tried(t, g, grabs.Trial{Names: []string{"The.Cellar.2020.1080p.BluRay.x264-GRP"}, Work: &ids[0]})
	naming(t, got.Sentence, "Bluray-1080p", "GRP", "The Cellar (2020)")
	if len(got.Works) != 0 || len(got.Targets) != 1 || got.Targets[0].WouldFill {
		t.Errorf("works %v, targets %+v, want no match and no fill", got.Works, got.Targets)
	}
}

// naming fails t unless sentence names each of facts.
func naming(t *testing.T, sentence string, facts ...string) {
	t.Helper()
	for _, f := range facts {
		if !strings.Contains(sentence, f) {
			t.Errorf("sentence = %q, want it to name %q", sentence, f)
		}
	}
}
