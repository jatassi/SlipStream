package files_test

import (
	"context"
	"database/sql"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// searching is Grabs in flight and failed as grabbing holds them, and
// facts as each cell's search facts.
type searching struct {
	grabbing
	facts map[files.CellKey]files.SearchFact
}

func (s searching) Searching(context.Context, storage.Reader[storage.Library], works.Among[works.WorkID]) (map[files.CellKey]files.SearchFact, error) {
	return s.facts, nil
}

func searchLines(cells []files.Cell) string {
	var b strings.Builder
	for _, c := range cells {
		fmt.Fprintf(&b, "%s %s %s gone=%t searchable=%t paced=%t\n", c.Item.Label, c.Target.Name, c.State, c.Gone, c.Searchable(), c.Paced())
	}
	return b.String()
}

func TestCellsReadGoneStoppedAndWaitingCells(t *testing.T) {
	l := open(t)
	hd := l.target(t, "HD", []release.Quality{release.WEBDL1080p})
	uhd := l.target(t, "UHD", []release.Quality{release.Remux2160p})
	p := l.profile(t, "Both", hd, uhd)
	tree := l.work(t, "w1", true, &p)
	scarf, mittens := tree.Items[0].ID, tree.Items[1].ID
	key := func(item works.ItemID, target targets.Target) files.CellKey {
		return files.CellKey{Item: item, Edition: targets.AnyEdition, Target: target.ID}
	}
	svc := l.files(searching{facts: map[files.CellKey]files.SearchFact{
		key(mittens, hd):  {Stopped: true},
		key(mittens, uhd): {WaitUntil: today.Add(time.Hour), Candidates: 2},
	}})
	l.place(t, scarf, tree.ID, "Shelf w1/Scarf.mkv", release.WEBDL1080p, hd.ID)
	l.markGone(t, svc, tree.ID, "Shelf w1/Scarf.mkv")

	_, cells, _, err := svc.WorkCells(t.Context(), admin, tree.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := "Scarf HD missing gone=true searchable=false paced=false\n" +
		"Scarf UHD missing gone=false searchable=true paced=true\n" +
		"Mittens HD missing gone=false searchable=false paced=true\n" +
		"Mittens UHD missing gone=false searchable=true paced=true\n"
	if got := searchLines(cells); got != want {
		t.Fatalf("cells =\n%swant the gone file's HD cell gone, the stopped cell paced only and the waiting cell searchable\n%s", got, want)
	}
	if w := cells[3].Search; !w.WaitUntil.Equal(today.Add(time.Hour)) || w.Candidates != 2 {
		t.Fatalf("Mittens UHD's search fact is %+v, want its wait", w)
	}
}

func TestReasonOfIsTheFirstReasonAndCountsEachOtherItemOnce(t *testing.T) {
	item := func(label string) works.Item {
		return works.Item{ID: works.ItemID{UUID: storage.NewUUID()}, Label: label, Monitored: true}
	}
	scarf, mittens, gloves, hat := item("Scarf"), item("Mittens"), item("Gloves"), item("Hat")
	cell := func(it works.Item, s files.State) files.Cell {
		return files.Cell{Item: it, State: s, Released: true, Sourcing: targets.Download}
	}
	gone := func(c files.Cell) files.Cell { c.Gone = true; return c }
	stopped := func(c files.Cell) files.Cell { c.Search.Stopped = true; return c }
	for _, c := range []struct {
		name   string
		cells  []files.Cell
		reason string
		others map[string]int
	}{
		{"an upgrade and a missing cell of one Item, a gone Item and a met one", []files.Cell{
			cell(scarf, files.Upgradable), cell(scarf, files.Missing), cell(mittens, files.Upgradable), gone(cell(gloves, files.Missing)), cell(hat, files.Met),
		}, "missing", map[string]int{"upgrade": 1, "gone": 1}},
		{"a stopped cell", []files.Cell{cell(scarf, files.Missing), stopped(cell(mittens, files.Missing))}, "failed", map[string]int{"missing": 1}},
		{"a failed cell", []files.Cell{cell(scarf, files.Upgradable), cell(mittens, files.Failed)}, "failed", map[string]int{"upgrade": 1}},
		{"gone cells alone", []files.Cell{gone(cell(scarf, files.Missing))}, "gone", map[string]int{}},
		{"nothing wanted", []files.Cell{cell(scarf, files.Met), cell(mittens, files.Downloading), cell(gloves, files.Unreleased)}, "", nil},
	} {
		reason, others, ok := files.ReasonOf(c.cells)
		got := map[string]int{}
		for r, n := range others {
			got[r.String()] = n
		}
		switch {
		case c.reason == "" && ok:
			t.Errorf("%s: ReasonOf = %s %v, want nothing wanted", c.name, reason, got)
		case c.reason != "" && (!ok || reason.String() != c.reason || !maps.Equal(got, c.others)):
			t.Errorf("%s: ReasonOf = %v %v %t, want %s %v", c.name, reason, got, ok, c.reason, c.others)
		}
	}
}

func TestNotFoundNeedsListMissingItemsAndLeaveTheLibraryFlagAlone(t *testing.T) {
	l := open(t)
	tree := l.stocked(t, "w1", true, box("b1", false, "Scarf", "Mittens", "Gloves"))
	starter := must[[]targets.Target](t)(l.targets.ListTargets(t.Context(), admin))[0]
	scarf, mittens, gloves := itemOf(t, tree, "Scarf"), itemOf(t, tree, "Mittens"), itemOf(t, tree, "Gloves")
	key := func(item works.ItemID) files.CellKey {
		return files.CellKey{Item: item, Edition: targets.AnyEdition, Target: starter.ID}
	}
	notFound := files.SearchFact{NotFound: true, Group: "b1"}
	svc := l.files(searching{
		grabbing: grabbing{key(gloves): {State: files.GrabFailed, Grab: storage.NewUUID()}},
		facts:    map[files.CellKey]files.SearchFact{key(scarf): notFound, key(mittens): notFound, key(gloves): notFound},
	})
	_, _, rollup, err := svc.WorkCells(t.Context(), admin, tree.ID)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[files.NeedKind]string{files.NeedFailed: "failed", files.NeedNotFound: "not found"}
	var got []string
	flagged := 0
	for _, n := range rollup.Needs {
		got = append(got, fmt.Sprintf("%s %d %q", kinds[n.Kind], len(n.Items), n.Group))
		if n.Flags() {
			flagged++
		}
	}
	want := []string{`failed 1 ""`, `not found 2 "b1"`}
	if !slices.Equal(got, want) || flagged != 1 {
		t.Fatalf("Needs you = %q, %d flagged; want Scarf and Mittens not found in one run after the failure, the failure alone flagged", got, flagged)
	}
}

// counting is r, counting the queries read through it.
type counting struct {
	storage.Reader[storage.Library]
	queries *int
}

func (c counting) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	*c.queries++
	return c.Reader.QueryContext(ctx, query, args...)
}

func (c counting) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	*c.queries++
	return c.Reader.QueryRowContext(ctx, query, args...)
}

func TestWantedListsEachWantingWorkReadingEachKindOfRowOnce(t *testing.T) {
	l := open(t)
	hd := l.target(t, "HD", []release.Quality{release.WEBDL1080p})
	list := must[targets.Editions](t)(targets.NamedCuts(release.Standard, "extended"))
	both := must[targets.Profile](t)(l.targets.CreateProfile(t.Context(), admin, targets.ProfileSpec{
		Name: "Both", ContentType: contenttypetest.ID, Entries: []targets.Entry{{Target: hd.ID, Sourcing: targets.Download}}, Editions: list,
	})).ID
	w1 := l.work(t, "w1", true, &both)
	w2 := l.work(t, "w2", true, &both)
	w3 := l.work(t, "w3", false, &both)
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, w := range []works.Tree{w1, w2} {
			if err := targets.See(ctx, tx, w.Items[0].ID, "extended", "x"+string(w.Key)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	svc := l.files(files.NoGrabs{})
	l.place(t, w2.Items[0].ID, w2.ID, "Shelf w2/Scarf.mkv", release.WEBDL1080p, hd.ID)
	l.settle(t, svc)

	wanted := func(ids ...works.WorkID) ([]files.WorkWanted, int) {
		var out []files.WorkWanted
		n := 0
		err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
			var err error
			out, err = svc.Wanted(ctx, counting{r, &n}, works.Only(ids))
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return out, n
	}
	wanted(w2.ID)
	_, one := wanted(w2.ID)
	all, many := wanted(w1.ID, w2.ID, w3.ID)
	if many != one {
		t.Errorf("Wanted of three Works ran %d queries and of one %d once its formats were cached, want the same", many, one)
	}
	var got []string
	for _, w := range all {
		var cells, unseen []string
		for _, c := range w.Cells {
			cells = append(cells, c.Item.Label+" "+string(c.Edition)+" "+c.State.String())
		}
		for _, it := range w.Unseen {
			unseen = append(unseen, it.String())
		}
		got = append(got, fmt.Sprintf("%s: %s; unseen %d", w.Title, strings.Join(cells, ", "), len(unseen)))
	}
	want := []string{
		"Shelf w1: Scarf standard missing, Scarf extended missing, Mittens standard missing; unseen 1",
		"Shelf w2: Scarf extended missing, Mittens standard missing; unseen 1",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("Wanted =\n%s\nwant the monitored Works' wanted cells and Mittens unseen in each\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// queueing is a Queue that records each call in the caller's Write, as
// grabs writes its nudges there.
type queueing struct{ lib *storage.DB[storage.Library] }

func record(t *testing.T, lib *storage.DB[storage.Library]) queueing {
	t.Helper()
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `CREATE TABLE queued (call INTEGER PRIMARY KEY, trigger TEXT NOT NULL, works TEXT NOT NULL)`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return queueing{lib}
}

func (queueing) Queue(ctx context.Context, tx *storage.Tx[storage.Library], t searches.Trigger, asks ...searches.Ask) error {
	var ws []string
	for _, a := range asks {
		if a.Items != nil {
			return fmt.Errorf("asked for Items %v of %s, want the whole Work", a.Items, a.Work)
		}
		ws = append(ws, a.Work.String())
	}
	slices.Sort(ws)
	_, err := tx.ExecContext(ctx, `INSERT INTO queued (trigger, works) VALUES (?, ?)`, string(t), strings.Join(ws, ","))
	return err
}

func (r queueing) calls(t *testing.T) []string {
	t.Helper()
	var joined string
	err := r.lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		row := rd.QueryRowContext(ctx, `SELECT coalesce(group_concat(trigger || ' ' || works, ';'), '') FROM (SELECT trigger, works FROM queued ORDER BY call)`)
		if err := row.Scan(&joined); err != nil {
			return fmt.Errorf("read the queued calls: %w", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if joined == "" {
		return nil
	}
	return strings.Split(joined, ";")
}

func TestASavedChangeQueuesItsWorksOnceAndAPreviewRollsItBack(t *testing.T) {
	l := open(t)
	queue := record(t, l.lib)
	l.queue = queue
	hd := l.target(t, "HD", []release.Quality{release.WEBDL1080p})
	both := l.profile(t, "Both", hd)
	other := l.profile(t, "Other", hd)
	w1, w2 := l.work(t, "w1", true, &both), l.work(t, "w2", true, &both)
	svc := l.files(files.NoGrabs{})
	pair := strings.Join(slices.Sorted(slices.Values([]string{w1.ID.String(), w2.ID.String()})), ",")

	move := targets.SetProfiles{Works: []works.WorkID{w1.ID, w2.ID}, Profile: other}
	l.change(t, svc, move, files.Mode{Unwanted: files.Delete, Preview: true})
	if got := queue.calls(t); len(got) != 0 {
		t.Fatalf("a Preview left %q queued, want its rollback to take them", got)
	}
	l.change(t, svc, move, deleting)
	l.change(t, svc, targets.SetHold{ID: hd.ID, Hold: time.Hour}, deleting)
	renamed := hd.Spec()
	renamed.Name = "Full HD"
	l.change(t, svc, targets.UpdateTarget{ID: hd.ID, Spec: renamed}, deleting)
	if got, want := queue.calls(t), []string{"profile " + pair, "retarget " + pair}; !slices.Equal(got, want) {
		t.Fatalf("queued %q, want one call per saved Change with the Works it touched, none for the Hold\n%q", got, want)
	}
}

func TestSearchesOfEachChange(t *testing.T) {
	for _, c := range []struct {
		change targets.Change
		want   searches.Trigger
	}{
		{targets.SetProfiles{}, searches.Profile},
		{targets.DeleteProfile{}, searches.Profile},
		{targets.SetWants{}, searches.Editions},
		{targets.UpdateTarget{}, searches.Retarget},
		{targets.UpdateProfile{}, searches.Retarget},
		{targets.UpdateCut{}, searches.Retarget},
		{targets.UpdateFormat{}, searches.Retarget},
		{targets.DeleteFormat{ID: formats.ID{}}, searches.Retarget},
		{targets.AddFormats{}, searches.Retarget},
		{targets.SetHold{}, ""},
	} {
		got, ok := files.SearchesOf(c.change)
		if got != c.want || ok != (c.want != "") {
			t.Errorf("searchesOf(%T) = %q %t, want %q", c.change, got, ok, c.want)
		}
	}
}
