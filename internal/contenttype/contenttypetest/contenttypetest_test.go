package contenttypetest_test

import (
	"context"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

func shelf(t *testing.T, f *contenttypetest.Fixture) (tree works.Tree, timings func() map[works.ItemID]targets.Timing) {
	t.Helper()
	lib := storagetest.Library(t, node.LibrarySchemas(contenttype.NewSet(f.Def()))...)
	w, err := works.Open(t.Context(), works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: f}, Timings: works.DefaultTimings(),
		Changed: discard[works.WorkChanged]{}, TypesChanged: discard[works.ContentTypesChanged]{}, Log: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	f.Put("shelf", contenttypetest.Record{Tree: works.Fetched{Label: "Shelf", Children: []works.Entry{
		{Key: "box", Label: "Box", Children: []works.Entry{{Key: "cup", Label: "Cup"}, {Key: "jar", Label: "Jar"}}},
	}}})
	tree, _, err = w.Add(t.Context(), identity.Admin{}, contenttypetest.ID, "shelf", works.Adding{Monitor: true})
	if err != nil {
		t.Fatal(err)
	}
	return tree, func() map[works.ItemID]targets.Timing {
		t.Helper()
		var got map[works.ItemID]targets.Timing
		if err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
			got, err = f.Timings(ctx, r, []works.WorkID{tree.ID})
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return got
	}
}

func TestEveryItemIsReleasedInThePastWithAnUnknownRuntime(t *testing.T) {
	tree, timings := shelf(t, contenttypetest.New())
	past := targets.Timing{Released: release.Known(contenttypetest.Released)}
	want := map[works.ItemID]targets.Timing{tree.Items[0].ID: past, tree.Items[1].ID: past}
	if got := timings(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Timings = %+v, want %+v", got, want)
	}
	if !past.ReleasedBy(time.Now()) {
		t.Fatalf("the default date %s is not in the past", contenttypetest.Released)
	}
}

func TestSetTimingAnswersForThatItemOnly(t *testing.T) {
	f := contenttypetest.New()
	tree, timings := shelf(t, f)
	set := targets.Timing{Released: release.Known(works.Date{Year: 2999}), Runtime: release.Known(90 * time.Minute)}
	f.SetTiming(tree.Items[1].ID, set)
	want := map[works.ItemID]targets.Timing{tree.Items[0].ID: {Released: release.Known(contenttypetest.Released)}, tree.Items[1].ID: set}
	if got := timings(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Timings = %+v, want %+v", got, want)
	}
}

func TestCutsAreTheSetList(t *testing.T) {
	f := contenttypetest.New()
	if cuts := f.Cuts(); len(cuts) != 0 {
		t.Fatalf("Cuts = %+v, want none until set", cuts)
	}
	want := []release.Cut{{Key: "boxed", Name: "Boxed", Aliases: []string{"Boxed", "Boxed Set"}}}
	f.SetCuts(want)
	if got := f.Cuts(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Cuts = %+v, want %+v", got, want)
	}
}
