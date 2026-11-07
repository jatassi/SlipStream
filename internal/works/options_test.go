package works_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// stacked is shelf with a stack order that lists Hat and Scarf in one box
// and a Sock the shelf order doesn't, and leaves Gloves out.
func stacked() contenttypetest.Record {
	r := shelf()
	r.Stacked = &works.Fetched{
		Label:     r.Tree.Label,
		SortLabel: r.Tree.SortLabel,
		Children: []works.Entry{
			{Key: "s1", Label: "Top", Children: []works.Entry{{Key: "c3", Label: "Hat"}, {Key: "c1", Label: "Scarf"}}},
			{Key: "s2", Label: "Bottom", Children: []works.Entry{{Key: "c4", Label: "Sock"}}},
		},
	}
	return r
}

func toOrder(o contenttypetest.Order) works.Optioning {
	return works.Optioning{Type: setting.Patch{"order": json.RawMessage(`"` + string(o) + `"`)}}
}

// options is how the Work's options read now, as the Work view shows
// them.
func (h *harness) options(t *testing.T, id works.WorkID) string {
	t.Helper()
	var c works.OptionsChange
	err := h.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		c, err = h.works.OptionsOf(ctx, r, id)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return optionsOf(c)
}

func optionsOf(c works.OptionsChange) string {
	var b strings.Builder
	fmt.Fprintf(&b, "levelFolders %t withdraws %d", c.LevelFolders, c.Withdraws)
	for _, f := range c.Fields {
		choice, ok := f.(setting.ChoiceField)
		if !ok {
			return fmt.Sprintf("an option of kind %T", f)
		}
		fmt.Fprintf(&b, " %s=%s of %d", choice.Key, choice.Value, len(choice.Options))
	}
	return b.String()
}

func monitoredOf(tree works.Tree) string {
	var on []string
	for _, it := range tree.Items {
		if it.Monitored {
			on = append(on, string(it.Key))
		}
	}
	return strings.Join(on, " ")
}

func (h *harness) addStacked(t *testing.T) works.Tree {
	t.Helper()
	h.fx.Put("w1", stacked())
	tree, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{Monitor: true, LevelFolders: true})
	if err != nil {
		t.Fatal(err)
	}
	h.changed.take()
	return tree
}

func (h *harness) get(t *testing.T, id works.WorkID) works.Tree {
	t.Helper()
	tree, err := h.works.Get(t.Context(), admin, id)
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func (h *harness) expectChanged(t *testing.T, step string, ids ...works.WorkID) {
	t.Helper()
	var want []string
	for _, id := range ids {
		want = append(want, id.String())
	}
	got := h.changed.take()
	if len(want) == 0 && len(got) == 0 {
		return
	}
	if len(got) != 1 || !slices.Equal(got[0].WorkIDs, want) {
		t.Fatalf("%s published %v, want works.changed for %v", step, got, want)
	}
}

func TestOrderChangeKeepsItemsAndWithdrawsTheRest(t *testing.T) {
	h := newHarness(t)
	first := h.addStacked(t)

	c, err := h.works.SetOptions(t.Context(), admin, first.ID, toOrder(contenttypetest.OrderStack), false)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := optionsOf(c), "levelFolders true withdraws 1 order=stack of 2"; got != want {
		t.Fatalf("the change answered %s, want %s", got, want)
	}
	if got := h.fx.Calls(); string(got[len(got)-1].Options) != `{"order":"stack"}` {
		t.Fatalf("the refetch asked with options %s, want the stack order", got[len(got)-1].Options)
	}
	stackedTree := h.get(t, first.ID)
	want := `w1 "The Attic" sort "Attic, The" origin="Grandma"
  level s1 depth 1 under "" #1 "Top"
  level s2 depth 1 under "" #2 "Bottom"
  level b1 depth 1 under "" #1 "Winter" withdrawn packed=2024-11
  level b2 depth 1 under "" #2 "Summer" withdrawn
  item c3 in s1 #1 "Hat" count=1
  item c1 in s1 #2 "Scarf" count=2 runtime=1m30s
  item c4 in s2 #1 "Sock"
  item c2 in b1 #2 "Gloves" withdrawn
`
	if got := render(stackedTree); got != want {
		t.Fatalf("the tree in stack order =\n%s\nwant\n%s", got, want)
	}
	sameIDs(t, first, stackedTree)
	if got, want := monitoredOf(stackedTree), "c3 c1 c4 c2"; got != want {
		t.Fatalf("monitored after the change: %s, want every Item kept and the new one monitored as the Work's rule says: %s", got, want)
	}
	if got, want := h.options(t, first.ID), "levelFolders true withdraws 0 order=stack of 2"; got != want {
		t.Fatalf("the Work's options read %s, want %s", got, want)
	}
	h.expectChanged(t, "the change", first.ID)

	c, err = h.works.SetOptions(t.Context(), admin, first.ID, toOrder(contenttypetest.OrderShelf), false)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := optionsOf(c), "levelFolders true withdraws 1 order=shelf of 2"; got != want {
		t.Fatalf("the change back answered %s, want %s: Sock withdrawn", got, want)
	}
	back := h.get(t, first.ID)
	sameIDs(t, stackedTree, back)
	if got, want := render(back), `w1 "The Attic" sort "Attic, The" origin="Grandma"
  level b1 depth 1 under "" #1 "Winter" packed=2024-11
  level b2 depth 1 under "" #2 "Summer"
  level s1 depth 1 under "" #1 "Top" withdrawn
  level s2 depth 1 under "" #2 "Bottom" withdrawn
  item c1 in b1 #1 "Scarf" count=2 runtime=1m30s
  item c2 in b1 #2 "Gloves"
  item c3 in b2 #1 "Hat" count=1
  item c4 in s2 #1 "Sock" withdrawn
`; got != want {
		t.Fatalf("the tree back in shelf order =\n%s\nwant\n%s", got, want)
	}
	h.expectChanged(t, "the change back", first.ID)
}

func TestAFailedRefetchChangesNothing(t *testing.T) {
	h := newHarness(t)
	first := h.addStacked(t)
	h.fx.Fail("w1", works.ErrUnavailable)
	if _, err := h.works.SetOptions(t.Context(), admin, first.ID, works.Optioning{LevelFolders: new(false), Type: toOrder(contenttypetest.OrderStack).Type}, false); !errors.Is(err, works.ErrUnavailable) {
		t.Fatalf("an order change whose fetch fails = %v, want ErrUnavailable", err)
	}
	if got, want := render(h.get(t, first.ID)), render(first); got != want {
		t.Fatalf("after a failed fetch the tree is\n%s\nwant it unchanged\n%s", got, want)
	}
	if got, want := h.options(t, first.ID), "levelFolders true withdraws 0 order=shelf of 2"; got != want {
		t.Fatalf("after a failed fetch the options read %s, want %s", got, want)
	}
	h.expectChanged(t, "a failed change")
}

func TestOptionsPreviewWritesNothing(t *testing.T) {
	h := newHarness(t)
	first := h.addStacked(t)
	fetched := h.fx.Fetches("w1")

	c, err := h.works.SetOptions(t.Context(), admin, first.ID, works.Optioning{LevelFolders: new(false), Type: toOrder(contenttypetest.OrderStack).Type}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := optionsOf(c), "levelFolders false withdraws 1 order=stack of 2"; got != want {
		t.Fatalf("the preview answered %s, want what the change would make: %s", got, want)
	}
	if n := h.fx.Fetches("w1") - fetched; n != 1 {
		t.Fatalf("the preview fetched %d times, want once, to count what it withdraws", n)
	}
	if got, want := render(h.get(t, first.ID)), render(first); got != want {
		t.Fatalf("after a preview the tree is\n%s\nwant it unchanged\n%s", got, want)
	}
	if got, want := h.options(t, first.ID), "levelFolders true withdraws 0 order=shelf of 2"; got != want {
		t.Fatalf("after a preview the options read %s, want %s", got, want)
	}
	h.expectChanged(t, "a preview")

	c, err = h.works.SetOptions(t.Context(), admin, first.ID, toOrder(contenttypetest.OrderStack), false)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := optionsOf(c), "levelFolders true withdraws 1 order=stack of 2"; got != want {
		t.Fatalf("the change after a preview answered %s, want %s", got, want)
	}
	h.expectChanged(t, "the change after a preview", first.ID)
}

func TestSameOptionsFetchNothing(t *testing.T) {
	h := newHarness(t)
	first := h.addStacked(t)
	fetched := h.fx.Fetches("w1")
	for _, o := range []works.Optioning{{}, toOrder(contenttypetest.OrderShelf), {LevelFolders: new(true), Type: setting.Patch{}}} {
		c, err := h.works.SetOptions(t.Context(), admin, first.ID, o, false)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := optionsOf(c), "levelFolders true withdraws 0 order=shelf of 2"; got != want {
			t.Fatalf("options %+v answered %s, want the Work's own: %s", o, got, want)
		}
	}
	if n := h.fx.Fetches("w1") - fetched; n != 0 {
		t.Fatalf("unchanged options fetched %d times, want none", n)
	}
	h.expectChanged(t, "unchanged options")

	c, err := h.works.SetOptions(t.Context(), admin, first.ID, works.Optioning{LevelFolders: new(false)}, false)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := optionsOf(c), "levelFolders false withdraws 0 order=shelf of 2"; got != want {
		t.Fatalf("turning level folders off answered %s, want %s", got, want)
	}
	if n := h.fx.Fetches("w1") - fetched; n != 0 {
		t.Fatalf("a level folder switch fetched %d times, want none", n)
	}
	h.expectChanged(t, "a level folder switch", first.ID)
}

func TestAnOrderTheSourceDoesNotListIsRefused(t *testing.T) {
	h := newHarness(t)
	h.fx.Put("w1", shelf())
	tree, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "w1", works.Adding{})
	if err != nil {
		t.Fatal(err)
	}
	fetched := h.fx.Fetches("w1")
	if _, err := h.works.SetOptions(t.Context(), admin, tree.ID, toOrder(contenttypetest.OrderStack), false); !errors.Is(err, works.ErrOrderUnavailable) {
		t.Fatalf("an order the source doesn't list = %v, want ErrOrderUnavailable", err)
	}
	var unknown *setting.UnknownKeyError
	if _, err := h.works.SetOptions(t.Context(), admin, tree.ID, works.Optioning{Type: setting.Patch{"shape": json.RawMessage(`"round"`)}}, false); !errors.As(err, &unknown) {
		t.Fatalf("an option the type doesn't have = %v, want an UnknownKeyError", err)
	}
	if _, err := h.works.SetOptions(t.Context(), admin, works.WorkID{UUID: storage.NewUUID()}, works.Optioning{}, false); !errors.Is(err, works.ErrNoWork) {
		t.Fatalf("options of an unknown Work = %v, want ErrNoWork", err)
	}
	if n := h.fx.Fetches("w1") - fetched; n != 0 {
		t.Fatalf("refused options fetched %d times, want none", n)
	}
}
