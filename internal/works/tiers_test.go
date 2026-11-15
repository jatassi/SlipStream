package works_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func asides(tree works.Tree) string {
	var b strings.Builder
	for _, l := range tree.Levels {
		if l.Aside {
			b.WriteString("level " + string(l.Key) + " ")
		}
	}
	for _, it := range tree.Items {
		if it.Aside {
			b.WriteString("item " + string(it.Key) + " ")
		}
	}
	return strings.TrimSpace(b.String())
}

func TestAnAsideLevelSetsItsItemsAside(t *testing.T) {
	h := newHarness(t)
	r := shelf()
	r.Tree.Children = append(r.Tree.Children, works.Entry{Key: "b9", Label: "Spares", Aside: true, Children: []works.Entry{{Key: "c9", Label: "Button"}}})
	h.fx.Put("w1", r)
	added, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := asides(added), "level b9 item c9"; got != want {
		t.Fatalf("asides after the add = %q, want %q", got, want)
	}

	r.Tree.Children[2].Aside = false
	h.fx.Put("w1", r)
	refreshed, err := h.works.Refresh(t.Context(), admin, added.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := asides(refreshed); got != "" {
		t.Fatalf("asides after the source moved b9 into the main run = %q, want none", got)
	}
	sameIDs(t, added, refreshed)
}

func TestAnAsideItemIsRefused(t *testing.T) {
	h := newHarness(t)
	r := shelf()
	r.Tree.Children[0].Children[0].Aside = true
	h.fx.Put("w1", r)
	_, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{})
	if !errors.Is(err, works.ErrInvalid) {
		t.Fatalf("add = %v, want ErrInvalid", err)
	}
}

func TestLevelArtworkLandsOnItsLevel(t *testing.T) {
	h := newHarness(t)
	r := shelf()
	label := artwork.Ref{Source: "fixture", Shape: artwork.ShapePoster, Path: "/winter.jpg"}
	r.Tree.Artwork = []works.Art{{Level: "b1", Kind: "label", Ref: label}}
	h.fx.Put("w1", r)
	added, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{})
	if err != nil {
		t.Fatal(err)
	}
	got := map[works.Key]map[string]artwork.Ref{}
	for _, l := range added.Levels {
		if len(l.Artwork) > 0 {
			got[l.Key] = l.Artwork
		}
	}
	want := map[works.Key]map[string]artwork.Ref{"b1": {"label": label}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("level artwork = %v, want %v", got, want)
	}
	if len(added.Artwork) != 0 {
		t.Fatalf("Work artwork = %v, want none", added.Artwork)
	}
}

func TestArtworkOfAnotherTierIsRefused(t *testing.T) {
	for name, art := range map[string]works.Art{
		"a Work kind on a level":   {Level: "b1", Kind: "photo", Ref: artwork.Ref{Source: "fixture", Shape: artwork.ShapePoster, Path: "/a.jpg"}},
		"a level kind on an Item":  {Item: "c1", Kind: "label", Ref: artwork.Ref{Source: "fixture", Shape: artwork.ShapePoster, Path: "/a.jpg"}},
		"a level kind on the Work": {Kind: "label", Ref: artwork.Ref{Source: "fixture", Shape: artwork.ShapePoster, Path: "/a.jpg"}},
	} {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)
			r := shelf()
			r.Tree.Artwork = []works.Art{art}
			h.fx.Put("w1", r)
			if _, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{}); !errors.Is(err, works.ErrInvalid) {
				t.Fatalf("add = %v, want ErrInvalid", err)
			}
		})
	}
}

func monitored(tree works.Tree) string {
	var on []string
	for _, it := range tree.Items {
		if it.Monitored {
			on = append(on, string(it.Key))
		}
	}
	slices.Sort(on)
	return strings.Join(on, " ")
}

func TestRefreshInsertsUnderMonitorNew(t *testing.T) {
	for _, c := range []struct {
		name             string
		monitor          bool
		added, refreshed string
		told             []works.Key
	}{
		{"a Work that monitors new Items", true, "c1 c2 c3 c4", "c1 c2 c3 c4 c5", []works.Key{"c5"}},
		{"a Work that doesn't", false, "", "", nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			h := newHarness(t)
			r := shelf()
			r.Tree.Children[1].Children = append(r.Tree.Children[1].Children, works.Entry{Key: "c4", Label: "Sandals"})
			r.Tree.Children = append(r.Tree.Children, works.Entry{Key: "b9", Label: "Spares", Aside: true, Children: []works.Entry{{Key: "c9", Label: "Button"}}})
			h.fx.Put("w1", r)
			added, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{Monitor: c.monitor})
			if err != nil {
				t.Fatal(err)
			}
			if got := monitored(added); got != c.added {
				t.Fatalf("monitored after the add = %q, want %q", got, c.added)
			}

			r.Tree.Children[1].Children = append(r.Tree.Children[1].Children, works.Entry{Key: "c5", Label: "Towel"})
			r.Tree.Children[2].Children = append(r.Tree.Children[2].Children, works.Entry{Key: "c10", Label: "Thread"})
			h.fx.Put("w1", r)
			refreshed, err := h.works.Refresh(t.Context(), admin, added.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got := monitored(refreshed); got != c.refreshed {
				t.Fatalf("monitored after a refresh added c5 and the aside c10 = %q, want %q", got, c.refreshed)
			}
			if got := keysOfIDs(refreshed, h.added.items[added.ID]); !slices.Equal(got, c.told) {
				t.Fatalf("ItemsAdded heard %v of the refresh that added c5 and the aside c10, want %v", got, c.told)
			}
		})
	}
}

func keysOfIDs(t works.Tree, ids []works.ItemID) []works.Key {
	var keys []works.Key
	for _, id := range ids {
		i := slices.IndexFunc(t.Items, func(it works.Item) bool { return it.ID == id })
		keys = append(keys, t.Items[i].Key)
	}
	return keys
}

func TestARefreshWhoseItemsAddedFailsInsertsNothing(t *testing.T) {
	h := newHarness(t)
	r := shelf()
	h.fx.Put("w1", r)
	added, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{Monitor: true})
	if err != nil {
		t.Fatal(err)
	}
	r.Tree.Children[1].Children = append(r.Tree.Children[1].Children, works.Entry{Key: "c4", Label: "Sandals"})
	h.fx.Put("w1", r)
	h.added.refuse = errors.New("the queue is closed")
	if _, err := h.works.Refresh(t.Context(), admin, added.ID); !errors.Is(err, h.added.refuse) {
		t.Fatalf("a refresh whose ItemsAdded fails = %v, want its error", err)
	}
	got, err := h.works.Get(t.Context(), admin, added.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := monitored(got); got != "c1 c2 c3" {
		t.Fatalf("after a refresh whose ItemsAdded failed the Work monitors %q, want c1 c2 c3 and no c4", got)
	}
}

func TestWorksOfItemsGroupsItemsByTheirWork(t *testing.T) {
	h := newHarness(t)
	h.fx.Put("w1", shelf())
	h.fx.Put("w2", shelf())
	one, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{})
	if err != nil {
		t.Fatal(err)
	}
	two, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w2", works.Adding{})
	if err != nil {
		t.Fatal(err)
	}
	unknown := works.ItemID{UUID: storage.NewUUID()}
	var got map[works.WorkID][]works.ItemID
	err = h.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		got, err = works.WorksOfItems(ctx, r, []works.ItemID{two.Items[0].ID, one.Items[2].ID, unknown, one.Items[0].ID})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[works.WorkID][]works.ItemID{one.ID: {one.Items[0].ID, one.Items[2].ID}, two.ID: {two.Items[0].ID}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("WorksOfItems = %v, want %v with the unknown id left out", got, want)
	}
}
