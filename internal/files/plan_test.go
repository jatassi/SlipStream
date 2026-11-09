package files

import (
	"errors"
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

const planned = "{Shelf} - {Thing} - {Edition Tag} {Target}"

// attic is the world of a Work on the shelf Attic of one Scarf wanting
// Web, holding a file at each path in order: one filling Web when fills
// names it, else a Kept file with N extra.
type attic struct {
	w     world
	web   targets.Target
	scarf works.Item
	ids   map[string]ID
}

func newAttic(t *testing.T) *attic {
	web := newTarget(t, "Web", []release.Quality{release.WEBDL1080p})
	scarf := works.Item{ID: works.ItemID{UUID: storage.NewUUID()}}
	w := shelved(t, planned, targets.Goals{Targets: []targets.Target{web}}, scarf)
	w.tree.ID = works.WorkID{UUID: storage.NewUUID()}
	w.fills = map[CellKey]Fill{}
	return &attic{w: w, web: web, scarf: scarf, ids: map[string]ID{}}
}

// fill adds a file at path that fills Web.
func (a *attic) fill(at string) *attic {
	f := a.add(at, 0)
	a.w.fills[CellKey{Item: a.scarf.ID, Edition: targets.AnyEdition, Target: a.web.ID}] = Fill{File: f.ID}
	return a
}

// kept adds a Kept file at path holding N extra.
func (a *attic) kept(at string, extra int64) *attic {
	a.add(at, extra)
	return a
}

func (a *attic) add(at string, extra int64) File {
	f := File{ID: ID{UUID: storage.NewUUID()}, WorkID: a.w.tree.ID, ItemID: a.scarf.ID, Cut: release.Standard, Quality: release.WEBDL1080p, Label: "Web", Extra: extra, Path: at}
	a.w.files = append(a.w.files, f)
	a.ids[at] = f.ID
	return f
}

// renamingOff names the Attic's files with renaming off.
func (a *attic) renamingOff(t *testing.T) *attic {
	t.Helper()
	off := false
	scheme, err := naming.SchemeOf(shelf(planned).Naming(), naming.Settings{Rename: &off})
	if err != nil {
		t.Fatal(err)
	}
	a.w.scheme = scheme
	return a
}

// rows is a's plan as "from -> to" with each row's Block and Group.
func (a *attic) rows(t *testing.T) (map[string]Row, WorkPlan) {
	t.Helper()
	p, _, err := plan(a.w)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]Row{}
	for _, r := range p.Rows {
		out[r.From] = r
	}
	return out, p
}

func (a *attic) row(from, to string, b Blocked, group int) Row {
	return Row{Work: a.w.tree.ID, File: a.ids[from], From: from, To: to, Blocked: b, Group: group}
}

func TestAPlanListsEachLiveFileTheTemplatesPutElsewhere(t *testing.T) {
	a := newAttic(t).fill("Loft/scarf, as found.mkv").kept("Attic/Attic - Scarf - Web (extra 1).mkv", 1).kept("Den/Attic - Scarf - Web (extra 2).mkv", 2)
	a.w.files[2].GoneAt = storage.Now()

	got, p := a.rows(t)
	want := map[string]Row{"Loft/scarf, as found.mkv": a.row("Loft/scarf, as found.mkv", "Attic/Attic - Scarf - Web.mkv", Blocked{}, 0)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rows =\n%v\nwant the found file moved to the Attic, the matching file and the gone file absent\n%v", got, want)
	}
	if p.Work != a.w.tree.ID || p.Moving != 0 {
		t.Fatalf("plan = %+v, want the Attic's with nothing moving", p)
	}
}

func TestKeptFilesFromTwoFoldersTakeDistinctNWhereTheyMeet(t *testing.T) {
	a := newAttic(t).
		fill("Attic/Attic - Scarf - Web.mkv").
		kept("Loft/Attic - Scarf - Web (extra 1).mkv", 1).
		kept("Den/Attic - Scarf - Web (extra 1).mkv", 1)

	got, _ := a.rows(t)
	want := map[string]Row{
		"Loft/Attic - Scarf - Web (extra 1).mkv": a.row("Loft/Attic - Scarf - Web (extra 1).mkv", "Attic/Attic - Scarf - Web (extra 1).mkv", Blocked{}, 0),
		"Den/Attic - Scarf - Web (extra 1).mkv":  a.row("Den/Attic - Scarf - Web (extra 1).mkv", "Attic/Attic - Scarf - Web (extra 2).mkv", Blocked{}, 0),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rows =\n%v\nwant the older Kept file to keep N 1 in the Attic and the other to take 2\n%v", got, want)
	}
}

func TestFilesThatTradePathsAreOneGroup(t *testing.T) {
	a := newAttic(t).
		kept("Attic/Attic - Scarf - Web (EXTRA 2).mkv", 1).
		kept("Attic/Attic - Scarf - Web (extra 1).mkv", 2).
		fill("Loft/scarf.mkv")

	got, _ := a.rows(t)
	want := map[string]Row{
		"Attic/Attic - Scarf - Web (EXTRA 2).mkv": a.row("Attic/Attic - Scarf - Web (EXTRA 2).mkv", "Attic/Attic - Scarf - Web (extra 1).mkv", Blocked{}, 1),
		"Attic/Attic - Scarf - Web (extra 1).mkv": a.row("Attic/Attic - Scarf - Web (extra 1).mkv", "Attic/Attic - Scarf - Web (extra 2).mkv", Blocked{}, 1),
		"Loft/scarf.mkv": a.row("Loft/scarf.mkv", "Attic/Attic - Scarf - Web.mkv", Blocked{}, 0),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rows =\n%v\nwant the two Kept files that swap N one group, compared without case, and the found file alone\n%v", got, want)
	}
}

func TestANameThatDoesntFitIsBlockedAndTheRestRename(t *testing.T) {
	a := newAttic(t).fill("Loft/scarf.mkv").kept("Loft/old scarf.mkv", 1)
	a.w.budgets = map[roots.ID]int{{}: len("Attic/Attic - Scarf - Web.mkv")}

	got, _ := a.rows(t)
	if r := got["Loft/scarf.mkv"]; r.To != "Attic/Attic - Scarf - Web.mkv" || r.Blocked != (Blocked{}) {
		t.Fatalf("the filling file's row = %+v, want it renamed to fit", r)
	}
	r := got["Loft/old scarf.mkv"]
	if r.To != "" || r.Blocked.Reason != TooLong || r.Blocked.Long == nil || r.Blocked.Long.Limit != len("Attic/Attic - Scarf - Web.mkv") {
		t.Fatalf("the Kept file's row = %+v, want it blocked TooLong at the root's budget with no To", r)
	}
}

func TestAFileTheQueueStillMovesIsCountedAndHoldsItsN(t *testing.T) {
	a := newAttic(t).
		kept("Attic/Attic - Scarf - Web (extra 1).mkv", 1).
		kept("Loft/Attic - Scarf - Web (extra 1).mkv", 1)
	a.w.moving = map[ID]bool{a.ids["Attic/Attic - Scarf - Web (extra 1).mkv"]: true}
	a.w.files[0].Label = "Old"

	got, p := a.rows(t)
	want := map[string]Row{"Loft/Attic - Scarf - Web (extra 1).mkv": a.row("Loft/Attic - Scarf - Web (extra 1).mkv", "Attic/Attic - Scarf - Web (extra 2).mkv", Blocked{}, 0)}
	if !reflect.DeepEqual(got, want) || p.Moving != 1 {
		t.Fatalf("rows =\n%v\nmoving %d\nwant the moving file left out and counted, its N held\n%v", got, p.Moving, want)
	}
}

func TestWithRenamingOffAPlanMovesFoldersAndBlocksKeptNamesThatMeet(t *testing.T) {
	a := newAttic(t).renamingOff(t).
		fill("Loft/scarf.mkv").
		kept("Den/Scarf.MKV", 1).
		kept("Attic/glove.mkv", 2).
		kept("Loft/Glove.mkv", 3).
		kept("Den/hat.mkv", 4)

	got, _ := a.rows(t)
	clash := Blocked{Reason: Clash, Other: a.w.tree.ID}
	want := map[string]Row{
		"Loft/scarf.mkv": a.row("Loft/scarf.mkv", "Attic/scarf.mkv", clash, 0),
		"Den/Scarf.MKV":  a.row("Den/Scarf.MKV", "Attic/Scarf.MKV", clash, 0),
		"Loft/Glove.mkv": a.row("Loft/Glove.mkv", "Attic/Glove.mkv", clash, 0),
		"Den/hat.mkv":    a.row("Den/hat.mkv", "Attic/hat.mkv", Blocked{}, 0),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rows =\n%v\nwant every file in the Attic under its own name, the ones meeting another blocked\n%v", got, want)
	}
}

func TestWithRenamingOnTwoFilesEndingOnOneNameIsAClash(t *testing.T) {
	a := newAttic(t)
	a.w.scheme = schemeOf(t, "{Shelf} - {Thing}")
	a.fill("Loft/scarf.mkv").kept("Attic/Attic - Scarf.mkv", 1)
	if _, _, err := plan(a.w); !errors.Is(err, ErrNameClash) {
		t.Fatalf("plan of two files the template names alike = %v, want ErrNameClash", err)
	}
}

func schemeOf(t *testing.T, file string) *naming.Scheme {
	t.Helper()
	scheme, err := naming.SchemeOf(shelf(file).Naming(), naming.Settings{})
	if err != nil {
		t.Fatal(err)
	}
	return scheme
}

func TestSettleBlocksRowsThatCantRenameNow(t *testing.T) {
	root := roots.ID{UUID: storage.NewUUID()}
	a, b := works.WorkID{UUID: storage.NewUUID()}, works.WorkID{UUID: storage.NewUUID()}
	row := func(w works.WorkID, from, to string) Row {
		return Row{Work: w, File: ID{UUID: storage.NewUUID()}, From: from, To: to, root: root}
	}
	long := Blocked{Reason: TooLong, Long: &naming.TooLongError{Bytes: 300, Limit: 259}}
	ps := []WorkPlan{
		{Work: a, Rows: []Row{
			row(a, "A/meet.mkv", "Shared/x.mkv"),
			row(a, "A/x.mkv", "A/taken.mkv"),
			row(a, "A/case.mkv", "A/Case.mkv"),
			row(a, "A/y.mkv", "B/held.mkv"),
			row(a, "A/p.mkv", "A/q.mkv"),
			row(a, "A/q.mkv", "A/p.mkv"),
			row(a, "A/long.mkv", ""),
		}},
		{Work: b, Rows: []Row{row(b, "B/meet.mkv", "Shared/X.mkv")}},
	}
	ps[0].Rows[6].Blocked = long
	found := map[ID][]string{
		ps[0].Rows[1].File: {"A/Taken.mkv"},
		ps[0].Rows[2].File: {"A/case.mkv"},
		ps[0].Rows[4].File: {"A/q.mkv"},
		ps[0].Rows[5].File: {"A/p.mkv"},
	}
	tracked := map[place][]works.WorkID{{root, "B/Held.mkv"}: {b}, {root, "A/p.mkv"}: {a}, {root, "A/q.mkv"}: {a}}

	settle(ps, tracked, found)

	want := []Blocked{{Reason: Clash, Other: b}, {Reason: Occupied}, {}, {Reason: Clash, Other: b}, {}, {}, long}
	for i, r := range ps[0].Rows {
		if r.Blocked != want[i] {
			t.Errorf("row %s -> %s blocked %+v, want %+v", r.From, r.To, r.Blocked, want[i])
		}
	}
	if got := ps[1].Rows[0].Blocked; got != (Blocked{Reason: Clash, Other: a}) {
		t.Errorf("the other Work's row meeting it blocked %+v, want a Clash naming the first", got)
	}
	for _, p := range ps {
		if p.Digest != digest(p.Rows) {
			t.Errorf("Work %s digest %s, want its settled rows' %s", p.Work, p.Digest, digest(p.Rows))
		}
	}
}

func TestADigestChangesWithItsRows(t *testing.T) {
	r := Row{File: ID{UUID: storage.NewUUID()}, From: "Loft/a.mkv", To: "Attic/a.mkv"}
	moved, grouped := r, r
	moved.To = "Attic/b.mkv"
	grouped.Group = 1
	d := digest([]Row{r})
	if d != digest([]Row{r}) || d == digest([]Row{moved}) || d == digest([]Row{grouped}) || d == digest(nil) {
		t.Fatalf("digest %s is the same for other rows or differs for the same rows", d)
	}
	back, err := ParseDigest(d.String())
	if err != nil || back != d || len(d.String()) != 16 {
		t.Fatalf("ParseDigest(%q) = %s, %v; want %s", d.String(), back, err, d)
	}
	for _, s := range []string{"", "9f2c41aa07be13", "9f2c41aa07be13d0aa", "9f2c41aa07be13zz"} {
		if _, err := ParseDigest(s); err == nil {
			t.Errorf("ParseDigest(%q) succeeded, want an error", s)
		}
	}
}
