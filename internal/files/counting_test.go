package files_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// stocked adds a Work of key holding boxes as they are listed, under the
// type's default Profile, monitored as monitor says.
func (l *library) stocked(t *testing.T, key works.Key, monitor bool, boxes ...works.Entry) works.Tree {
	t.Helper()
	l.fx.Put(key, contenttypetest.Record{Tree: works.Fetched{Label: "Shelf " + string(key), Children: boxes}})
	tree, _, err := l.works.Add(t.Context(), admin, contenttypetest.ID, key, works.Adding{
		Monitor: monitor,
		Also: func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
			if err := targets.Assign(ctx, tx, w, nil); err != nil {
				return err
			}
			return roots.Assign(ctx, tx, w, nil)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

// restock lists the Work of key as holding boxes and refreshes it.
func (l *library) restock(t *testing.T, tree works.Tree, boxes ...works.Entry) works.Tree {
	t.Helper()
	l.fx.Put(tree.Key, contenttypetest.Record{Tree: works.Fetched{Label: tree.Label, Children: boxes}})
	tree, err := l.works.Refresh(t.Context(), admin, tree.ID)
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func box(key works.Key, aside bool, things ...string) works.Entry {
	b := works.Entry{Key: key, Label: "Box " + string(key), Aside: aside}
	for _, thing := range things {
		b.Children = append(b.Children, works.Entry{Key: works.Key(strings.ToLower(thing)), Label: thing})
	}
	return b
}

func itemOf(t *testing.T, tree works.Tree, label string) works.ItemID {
	t.Helper()
	for _, it := range tree.Items {
		if it.Label == label {
			return it.ID
		}
	}
	t.Fatalf("%s holds no %s", tree.Label, label)
	return works.ItemID{}
}

func tally(c files.Counts) string {
	var parts []string
	for s, n := range c {
		if n > 0 {
			parts = append(parts, fmt.Sprintf("%s:%d", files.State(s), n))
		}
	}
	return "{" + strings.Join(parts, " ") + "}"
}

// summary is what a Rollup counts and the Needs you it lists, each entry
// as its kind, its Items' labels and its Grab.
func summary(tree works.Tree, r files.Rollup, grabs map[storage.UUID]string) string {
	labels := map[works.ItemID]string{}
	for _, it := range tree.Items {
		labels[it.ID] = it.Label
	}
	status := "none"
	if r.Status != 0 {
		status = r.Status.String()
	}
	available, _ := r.Available.Get()
	var b strings.Builder
	fmt.Fprintf(&b, "status=%s items=%d cells=%s byItem=%s failed=%d available=%d-%02d-%02d\n",
		status, r.Items, tally(r.Cells), tally(r.ByItem), r.Failed, available.Year, available.Month, available.Day)
	kinds := map[files.NeedKind]string{files.NeedFailed: "failed", files.NeedUpgrade: "upgrade", files.NeedWaiting: "waiting", files.NeedNoRoot: "no root"}
	for _, n := range r.Needs {
		var its []string
		for _, id := range n.Items {
			its = append(its, labels[id])
		}
		fmt.Fprintf(&b, "%s %s %s", kinds[n.Kind], n.Edition, strings.Join(its, ","))
		if g, ok := grabs[n.Grab]; ok {
			fmt.Fprintf(&b, " grab=%s", g)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func TestRollupCountsMonitoredLiveItems(t *testing.T) {
	l := open(t)
	tree := l.stocked(t, "w1", true, box("b1", false, "Scarf", "Mittens", "Gloves"), box("b9", true, "Button"))
	tree = l.restock(t, tree, box("b1", false, "Scarf", "Mittens"), box("b9", true, "Button"))
	scarf, gloves, button := itemOf(t, tree, "Scarf"), itemOf(t, tree, "Gloves"), itemOf(t, tree, "Button")
	l.fx.SetTiming(button, targets.Timing{Released: release.Known(works.Date{Year: 2000, Month: 6, Day: 1})})
	starter := must[[]targets.Target](t)(l.targets.ListTargets(t.Context(), admin))[0]
	failedScarf, failedGloves, waitsButton := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	svc := l.files(grabbing{
		{Item: scarf, Edition: targets.AnyEdition, Target: starter.ID}:  {State: files.GrabFailed, Grab: failedScarf},
		{Item: gloves, Edition: targets.AnyEdition, Target: starter.ID}: {State: files.GrabFailed, Grab: failedGloves},
		{Item: button, Edition: targets.AnyEdition, Target: starter.ID}: {State: files.GrabInFlight, Grab: waitsButton, Waiting: "node_away"},
	})
	_, cells, rollup, err := svc.WorkCells(t.Context(), admin, tree.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := "Scarf any 1080p failed wanted=true\n" +
		"Mittens any 1080p missing wanted=true\n" +
		"Gloves any 1080p failed wanted=false\n" +
		"Button any 1080p downloading wanted=false\n"
	if got := lines(cells); got != want {
		t.Fatalf("cells =\n%swant every Item's cells, withdrawn Gloves' and unmonitored Button's unwanted\n%s", got, want)
	}
	grabs := map[storage.UUID]string{failedScarf: "scarf", failedGloves: "gloves", waitsButton: "button"}
	want = "status=missing items=2 cells={missing:1 failed:1} byItem={missing:1 failed:1} failed=1 available=2000-01-01\n" +
		"failed any Scarf grab=scarf\n"
	if got := summary(tree, rollup, grabs); got != want {
		t.Fatalf("the rollup is\n%swant Scarf and Mittens alone counted\n%s", got, want)
	}

	quiet := l.stocked(t, "w2", false, box("b1", false, "Scarf"))
	l.fx.SetTiming(quiet.Items[0].ID, targets.Timing{Released: release.Known(works.Date{Year: 2000, Month: 6, Day: 1})})
	_, _, rollup, err = svc.WorkCells(t.Context(), admin, quiet.ID)
	if err != nil {
		t.Fatal(err)
	}
	want = "status=none items=0 cells={} byItem={} failed=0 available=2000-06-01\n"
	if got := summary(quiet, rollup, nil); got != want {
		t.Fatalf("an unmonitored Work's rollup is\n%swant nothing counted, available read from its live Items\n%s", got, want)
	}
}

func TestWithdrawnItemsAreNotWanted(t *testing.T) {
	l := open(t)
	tree := l.stocked(t, "w1", true, box("b1", false, "Scarf", "Mittens"))
	tree = l.restock(t, tree, box("b1", false, "Scarf"))
	_, cells, _, err := l.files(files.NoGrabs{}).WorkCells(t.Context(), admin, tree.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := "Scarf any 1080p missing wanted=true\n" +
		"Mittens any 1080p missing wanted=false\n"
	if got := lines(cells); got != want {
		t.Fatalf("cells once the source stops listing Mittens =\n%swant Mittens kept but unwanted, though monitored\n%s", got, want)
	}
}

func TestNeedsKeepsConsecutiveItemsSideBySide(t *testing.T) {
	l := open(t)
	hd := l.target(t, "HD", []release.Quality{release.WEBDL1080p})
	list := must[targets.Editions](t)(targets.NamedCuts(release.Standard, "extended"))
	both := must[targets.Profile](t)(l.targets.CreateProfile(t.Context(), admin, targets.ProfileSpec{
		Name: "Both", ContentType: contenttypetest.ID, Entries: []targets.Entry{{Target: hd.ID, Sourcing: targets.Download}}, Editions: list,
	})).ID
	tree := l.work(t, "w1", true, &both)
	tree = l.restock(t, tree, box("b1", false, "Scarf", "Mittens", "Gloves"))
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, it := range tree.Items {
			if err := targets.See(ctx, tx, it.ID, "extended", "x"+string(it.Key)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	grabs := map[storage.UUID]string{}
	in := grabbing{}
	fail := func(item works.ItemID, edition targets.Edition, name string, grab storage.UUID) {
		in[files.CellKey{Item: item, Edition: edition, Target: hd.ID}] = files.GrabCell{State: files.GrabFailed, Grab: grab}
		grabs[grab] = name
	}
	pack := storage.NewUUID()
	for _, it := range tree.Items {
		fail(it.ID, targets.Edition(release.Standard), strings.ToLower(it.Label), storage.NewUUID())
	}
	scarf, mittens, gloves := itemOf(t, tree, "Scarf"), itemOf(t, tree, "Mittens"), itemOf(t, tree, "Gloves")
	fail(scarf, "extended", "pack", pack)
	fail(mittens, "extended", "pack", pack)
	fail(gloves, "extended", "gloves extended", storage.NewUUID())
	_, _, rollup, err := l.files(in).WorkCells(t.Context(), admin, tree.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := "status=failed items=3 cells={failed:6} byItem={failed:3} failed=6 available=2000-01-01\n" +
		"failed standard Scarf grab=scarf\n" +
		"failed standard Mittens grab=mittens\n" +
		"failed standard Gloves grab=gloves\n" +
		"failed extended Scarf,Mittens grab=pack\n" +
		"failed extended Gloves grab=gloves extended\n"
	if got := summary(tree, rollup, grabs); got != want {
		t.Fatalf("Needs you is\n%swant each Edition's failures in tree order, side by side, one Grab's joined\n%s", got, want)
	}
}
