package files_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

func TestItemStatesReadEveryItemOfEachWorkInTreeOrder(t *testing.T) {
	l := open(t)
	hd := l.target(t, "HD", []release.Quality{release.WEBDL1080p})
	profile := l.profile(t, "HD", hd)
	met, thinned := l.work(t, "w1", true, &profile), l.work(t, "w2", true, &profile)
	l.place(t, met.Items[0].ID, met.ID, "Shelf w1/Scarf - HD.mkv", release.WEBDL1080p, hd.ID)
	if _, err := l.works.SetMonitored(t.Context(), admin, met.ID, works.Within{Items: []works.ItemID{met.Items[1].ID}}, false); err != nil {
		t.Fatal(err)
	}
	l.fx.Put("w2", contenttypetest.Record{Tree: works.Fetched{Label: "Shelf w2", Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c2", Label: "Mittens"}}},
	}}})
	if _, err := l.works.Refresh(t.Context(), admin, thinned.ID); err != nil {
		t.Fatal(err)
	}

	var got map[works.WorkID]files.WorkStates
	err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = l.files(files.NoGrabs{}).ItemStates(ctx, r, works.Only([]works.WorkID{met.ID, thinned.ID, {UUID: storage.NewUUID()}}))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, w := range []works.Tree{met, thinned} {
		s := got[w.ID]
		var line strings.Builder
		fmt.Fprintf(&line, "%s levels=%d:", w.Label, len(s.Levels))
		for _, it := range s.Items {
			fmt.Fprintf(&line, " %s %v monitored=%t live=%t released=%t;", it.Item.Label, it.Status, it.Item.Monitored, it.Item.Live(), it.Released)
		}
		lines = append(lines, line.String())
	}
	want := "Shelf w1 levels=1: Scarf met monitored=true live=true released=true; Mittens missing monitored=false live=true released=true;\n" +
		"Shelf w2 levels=1: Mittens missing monitored=true live=true released=true; Scarf missing monitored=true live=false released=true;"
	if g := strings.Join(lines, "\n"); len(got) != 2 || g != want {
		t.Fatalf("ItemStates answers %d Works:\n%s\nwant 2:\n%s", len(got), g, want)
	}
}

func TestACellsSourcingDecidesWhetherItIsSearched(t *testing.T) {
	counted := works.Item{Monitored: true}
	for _, c := range []struct {
		sourcing                  targets.Sourcing
		state                     files.State
		gone, stopped             bool
		wanted, paced, searchable bool
	}{
		{sourcing: targets.Download, state: files.Missing, wanted: true, paced: true, searchable: true},
		{sourcing: targets.Download, state: files.Failed, wanted: true, paced: true, searchable: true},
		{sourcing: targets.Download, state: files.Upgradable, wanted: true, paced: true, searchable: true},
		{sourcing: targets.Download, state: files.Downloading},
		{sourcing: targets.Download, state: files.Met},
		{sourcing: targets.Download, state: files.Missing, gone: true, wanted: true},
		{sourcing: targets.Download, state: files.Missing, stopped: true, wanted: true, paced: true},
		{sourcing: targets.Derive, state: files.Missing},
		{sourcing: targets.Derive, state: files.Failed},
		{sourcing: targets.Derive, state: files.Upgradable},
		{sourcing: targets.Derive, state: files.Converting},
		{sourcing: targets.DownloadThenDerive, state: files.Missing, wanted: true, paced: true, searchable: true},
		{sourcing: targets.DownloadThenDerive, state: files.Failed, wanted: true, paced: true, searchable: true},
		{sourcing: targets.DownloadThenDerive, state: files.Upgradable, wanted: true, paced: true, searchable: true},
		{sourcing: targets.DownloadThenDerive, state: files.Converting, wanted: true, paced: true, searchable: true},
		{sourcing: targets.DownloadThenDerive, state: files.Downloading},
		{sourcing: targets.DownloadThenDerive, state: files.Met},
		{sourcing: targets.DownloadThenDerive, state: files.Converting, gone: true, wanted: true},
		{sourcing: targets.DownloadThenDerive, state: files.Converting, stopped: true, wanted: true, paced: true},
	} {
		cell := files.Cell{Item: counted, Released: true, Sourcing: c.sourcing, State: c.state, Gone: c.gone, Search: files.SearchFact{Stopped: c.stopped}}
		if w, p, s := cell.Wanted(), cell.Paced(), cell.Searchable(); w != c.wanted || p != c.paced || s != c.searchable {
			t.Errorf("a %s %s cell, gone %t, stopped %t: wanted %t, paced %t, searchable %t; want %t, %t, %t",
				c.sourcing, c.state, c.gone, c.stopped, w, p, s, c.wanted, c.paced, c.searchable)
		}
	}
}

func TestAMadeFromCellReadsItsConversionAndAMadeFileNeverMeetsADownloadedOrMadeCell(t *testing.T) {
	waiting := files.ConversionCell{ID: storage.NewUUID(), Unfinished: true}
	due := files.ConversionCell{ID: storage.NewUUID(), Unfinished: true, Due: true}
	failed := files.ConversionCell{ID: storage.NewUUID(), Unfinished: true, Failed: true, Due: true, Code: "encoder_failed"}
	for _, c := range []struct {
		name     string
		sourcing targets.Sourcing
		mittens  *files.ConversionCell
		want     string
		needs    []files.NeedKind
	}{
		{"a made file and no Conversion", targets.DownloadThenDerive, nil,
			"Scarf Mobile upgradable wanted=true\nMittens Mobile missing wanted=true\n", []files.NeedKind{files.NeedUpgrade}},
		{"a Conversion in its download wait", targets.DownloadThenDerive, &waiting,
			"Scarf Mobile upgradable wanted=true\nMittens Mobile missing wanted=true\n", []files.NeedKind{files.NeedUpgrade}},
		{"a due Conversion", targets.DownloadThenDerive, &due,
			"Scarf Mobile upgradable wanted=true\nMittens Mobile converting wanted=true\n", []files.NeedKind{files.NeedUpgrade}},
		{"a failed Conversion", targets.DownloadThenDerive, &failed,
			"Scarf Mobile upgradable wanted=true\nMittens Mobile missing wanted=true\n", []files.NeedKind{files.NeedUpgrade}},
		{"a made file and no Conversion", targets.Derive, nil,
			"Scarf Mobile met wanted=false\nMittens Mobile missing wanted=false\n", nil},
		{"a due Conversion", targets.Derive, &due,
			"Scarf Mobile met wanted=false\nMittens Mobile converting wanted=false\n", nil},
		{"a failed Conversion", targets.Derive, &failed,
			"Scarf Mobile met wanted=false\nMittens Mobile failed wanted=false\n", []files.NeedKind{files.NeedConversionFailed}},
	} {
		t.Run(string(c.sourcing)+" "+c.name, func(t *testing.T) {
			m := newMade(t)
			p := m.madeProfile(t, "Movies", c.sourcing)
			tree := m.work(t, "w1", true, &p)
			scarf := m.add(t, tree, 0, "Shelf w1/a.mkv", release.WEBDL2160p, nil, m.uhd.ID)
			m.add(t, tree, 1, "Shelf w1/b.mkv", release.WEBDL2160p, nil, m.uhd.ID)
			m.add(t, tree, 0, "Shelf w1/a - Mobile.mkv", release.Bluray1080p, &files.Derived{From: scarf.ID, Target: m.mobile.ID, Version: m.version()}, m.mobile.ID)
			if c.mittens != nil {
				m.conv.cells = map[files.CellKey]files.ConversionCell{{Item: tree.Items[1].ID, Edition: targets.AnyEdition, Target: m.mobile.ID}: *c.mittens}
			}
			_, cells, rollup, err := m.files(files.NoGrabs{}).WorkCells(t.Context(), admin, tree.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got := cellsAt(cells, m.mobile.ID); got != c.want {
				t.Fatalf("Mobile's cells =\n%swant\n%s", got, c.want)
			}
			checkNeeds(t, rollup.Needs, c.needs, failed, m.mobile.ID)
		})
	}
}

// cellsAt is a line for each of cells at Target t.
func cellsAt(cells []files.Cell, t targets.TargetID) string {
	var b strings.Builder
	for _, c := range cells {
		if c.Target.ID == t {
			fmt.Fprintf(&b, "%s %s %s wanted=%t\n", c.Item.Label, c.Target.Name, c.State, c.Wanted())
		}
	}
	return b.String()
}

// checkNeeds checks needs are of kinds, and that a failed Conversion's
// names failed, its fault and its Target t.
func checkNeeds(t *testing.T, needs []files.Need, kinds []files.NeedKind, failed files.ConversionCell, target targets.TargetID) {
	t.Helper()
	got := make([]files.NeedKind, len(needs))
	for i, n := range needs {
		got[i] = n.Kind
		if n.Kind == files.NeedConversionFailed && (n.Conversion != failed.ID || n.Code != failed.Code || n.Target != target) {
			t.Fatalf("the failed Conversion's need is %+v, want its Conversion, fault and Target", n)
		}
	}
	if !slices.Equal(got, kinds) {
		t.Fatalf("Needs you = %v, want %v", got, kinds)
	}
}
