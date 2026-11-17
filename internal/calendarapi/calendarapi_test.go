package calendarapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/calendar"
	"github.com/jatassi/slipstream/internal/calendarapi"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

var admin identity.Admin

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

func must[T any](t *testing.T) func(T, error) T {
	return func(v T, err error) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

const (
	filmsID  works.TypeID = "films"
	cratesID works.TypeID = "crates"
)

// films is the Fixture as a type of one Item with three dated kinds.
type films struct{ *contenttypetest.Fixture }

func (films) Describe() works.Description {
	d := contenttypetest.Description
	d.Name, d.Work, d.Item, d.Items = "Films", works.Noun{One: "Film", Many: "Films"}, works.Noun{One: "Film", Many: "Films"}, works.ItemsOne
	d.Levels, d.Aside = nil, works.Noun{}
	d.Fields = slices.DeleteFunc(slices.Clone(d.Fields), func(f works.FieldSpec) bool { return f.Tier != works.TierWork && f.Tier != works.TierItem })
	d.Artwork = slices.DeleteFunc(slices.Clone(d.Artwork), func(k artwork.Kind) bool { return k.Tier != works.TierWork && k.Tier != works.TierItem })
	d.Dated = []works.DatedKind{{Key: "opening", Label: "In cinemas"}, {Key: "digital", Label: "Digital"}, {Key: "disc", Label: "On disc"}}
	return d
}

// crates is a type the tests turn off, counting each time it is asked.
type crates struct {
	*contenttypetest.Fixture
	asked *atomic.Int32
}

func (crates) Describe() works.Description {
	d := contenttypetest.Description
	d.Name, d.Dated = "Crates", []works.DatedKind{{Key: "crated", Label: "Crate date"}}
	return d
}

func (c crates) Dated(context.Context, storage.Reader[storage.Library], calendar.Span) ([]calendar.Dated, error) {
	c.asked.Add(1)
	return nil, nil
}

type node struct {
	c *calendar.Calendar
	m artwork.Minter
}

func (n node) Calendar() (*calendar.Calendar, bool) { return n.c, true }
func (n node) Minter() (artwork.Minter, bool)       { return n.m, true }

type world struct {
	lib     *storage.DB[storage.Library]
	fx      *contenttypetest.Fixture
	works   *works.Works
	s       *ownertest.Owner
	now     time.Time
	asked   *atomic.Int32
	attic   works.Tree
	lantern works.Tree
}

// The Controller sits 8 hours behind UTC, so its day and UTC's differ in
// its evenings.
var zone = time.FixedZone("Controller", -8*60*60)

func day(t *testing.T, s string) calendar.Day {
	t.Helper()
	return must[calendar.Day](t)(calendar.ParseDay(s))
}

// newWorld holds The Attic, a Shelf whose Box 1 ships its 10 Things on
// 2026-11-20, 8 of them met by one file, and whose Box 2 ships Parts 1,
// 2, 3 (since withdrawn), 4 (unmonitored) and 5 the same day; Lantern Fall, a
// film in cinemas on 2026-11-18, digital on 2026-11-19 and on disc on
// 2026-11-21; and Crates, turned off. The Controller's clock reads
// 2026-11-17 22:00.
func newWorld(t *testing.T) *world {
	ctx := t.Context()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), downloadclients.Schema(), grabs.Schema(), indexers.Schema(), files.Schema(), naming.Schema())
	fx := contenttypetest.New()
	w := &world{lib: lib, fx: fx, now: time.Date(2026, 11, 17, 22, 0, 0, 0, zone), asked: &atomic.Int32{}}
	film, crate := films{fx}, crates{Fixture: fx, asked: w.asked}
	w.works = must[*works.Works](t)(works.Open(ctx, works.Config{
		Library:  lib,
		Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx, filmsID: film, cratesID: crate},
		Source:   fx.SourceAs(contenttypetest.ID, filmsID, cratesID), Artwork: fx,
		Changed: discard[works.Changed]{}, TypesChanged: discard[works.ContentTypesChanged]{},
	}))
	must[works.ContentType](t)(w.works.SetEnabled(ctx, admin, cratesID, false))
	goals := must[*targets.Targets](t)(targets.Open(ctx, targets.Config{
		Library: lib, Works: w.works, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: fx, filmsID: film, cratesID: crate}, Changed: discard[targets.Changed]{},
	}))
	hd := must[targets.Target](t)(goals.CreateTarget(ctx, admin, targets.Spec{
		Scores: targets.Scores{Increment: 1}, Name: "HD", Tiers: [][]release.Quality{{release.WEBDL1080p}}, Cutoff: release.WEBDL1080p,
	}))
	var root roots.Root
	for _, ct := range []works.TypeID{contenttypetest.ID, filmsID} {
		p := must[targets.Profile](t)(goals.CreateProfile(ctx, admin, targets.ProfileSpec{Name: "Standard " + string(ct), ContentType: ct, Entries: []targets.Entry{{Target: hd.ID, Sourcing: targets.Download}}}))
		must[targets.Profile](t)(goals.MakeDefault(ctx, admin, p.ID))
		r := rootstest.Root(t, lib, role.NodeID{UUID: storage.NewUUID()}, ct, t.TempDir())
		if ct == contenttypetest.ID {
			root = r
		}
	}
	attic := contenttypetest.Record{Tree: works.Fetched{Label: "The Attic", Children: []works.Entry{
		{Key: "b1", Label: "Box 1", Children: things("a", 10)},
		{Key: "b2", Label: "Box 2", Children: things("b", 5)},
	}}}
	fx.Put("attic", attic)
	fx.Put("lantern", contenttypetest.Record{Tree: works.Fetched{Label: "Lantern Fall", Children: []works.Entry{{Key: "f1", Label: "Lantern Fall"}}}})
	w.attic = w.add(t, contenttypetest.ID, "attic")
	w.lantern = w.add(t, filmsID, "lantern")

	box1, box2 := w.attic.Items[:10], w.attic.Items[10:]
	for _, it := range slices.Concat(box1, box2) {
		fx.SetDated(w.attic.ID, it.ID, contenttypetest.Shipped.Key, day(t, "2026-11-20"))
	}
	for k, d := range map[works.DatedKey]string{"opening": "2026-11-18", "digital": "2026-11-19", "disc": "2026-11-21"} {
		fx.SetDated(w.lantern.ID, w.lantern.Items[0].ID, k, day(t, d))
	}
	must[int](t)(w.works.SetMonitored(ctx, admin, w.attic.ID, works.Within{Items: []works.ItemID{box2[3].ID}}, false))
	attic.Tree.Children[1].Children = slices.Delete(attic.Tree.Children[1].Children, 2, 3)
	fx.Put("attic", attic)
	must[works.Tree](t)(w.works.Refresh(ctx, admin, w.attic.ID))

	met := make([]works.ItemID, 8)
	for i := range met {
		met[i] = box1[i].ID
	}
	err := lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := files.Add(ctx, tx, files.File{
			WorkID: w.attic.ID, Items: met, Cut: release.Standard, RootID: root.ID,
			Path: "The Attic/Box 1/The Attic - Things 1 to 8.mkv", Size: 4096, Quality: release.WEBDL1080p,
		}, []targets.TargetID{hd.ID})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	clock := func() time.Time { return w.now }
	shelf := must[*files.Files](t)(files.Open(ctx, files.Config{Library: lib, Targets: goals, Grabs: files.NoGrabs{}, Today: clock}))
	dates := calendar.New(calendar.Config{
		Library: lib, Works: w.works, Files: shelf, Today: clock,
		Types: map[works.TypeID]calendar.Facet{contenttypetest.ID: fx, filmsID: film, cratesID: crate},
	})
	n := node{c: dates, m: artwork.NewMinter([]byte("calendar test key 32 bytes long!"))}
	w.s = ownertest.ServeIn(t, lib, zone, func(r *api.Router) { calendarapi.Routes(r, n) })
	return w
}

func things(prefix string, n int) []works.Entry {
	out := make([]works.Entry, n)
	for i := range out {
		out[i] = works.Entry{Key: works.Key(fmt.Sprintf("%s%d", prefix, i+1)), Label: fmt.Sprintf("Part %d", i+1)}
	}
	return out
}

func (w *world) add(t *testing.T, ct works.TypeID, key works.Key) works.Tree {
	t.Helper()
	tree, _, err := w.works.Add(t.Context(), admin, ct, key, works.Adding{
		Monitor: true,
		Also: func(ctx context.Context, tx *storage.Tx[storage.Library], s works.Summary) error {
			if err := targets.Assign(ctx, tx, s, nil); err != nil {
				return err
			}
			return roots.Assign(ctx, tx, s, nil)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

type answer struct {
	Today     string `json:"today"`
	NextDayAt string `json:"nextDayAt"`
	Sentence  *struct {
		NeedsYou bool   `json:"needsYou"`
		Words    string `json:"words"`
	} `json:"sentence"`
	Days []struct {
		Day  string `json:"day"`
		Rows []struct {
			Work struct {
				Label string `json:"label"`
			} `json:"work"`
			Kind       string `json:"kind"`
			ItemsWords string `json:"itemsWords"`
			Items      []struct {
				Label string `json:"label"`
			} `json:"items"`
			State     *string `json:"state"`
			Met       int     `json:"met"`
			Of        int     `json:"of"`
			Monitored bool    `json:"monitored"`
		} `json:"rows"`
	} `json:"days"`
}

func (w *world) read(t *testing.T, query string) answer {
	t.Helper()
	code, body := w.s.Call(t, http.MethodGet, "/api/v1/calendar?"+query, "")
	if code != http.StatusOK {
		t.Fatalf("getCalendar?%s = %d %s, want 200", query, code, body)
	}
	var a answer
	if err := json.Unmarshal([]byte(body), &a); err != nil {
		t.Fatal(err)
	}
	return a
}

// rows is each row of a as "day label kind items state met/of", with
// "unmonitored" after a row of unmonitored Items.
func rows(a answer) string {
	var lines []string
	for _, d := range a.Days {
		for _, r := range d.Rows {
			state := "none"
			if r.State != nil {
				state = *r.State
			}
			items := r.ItemsWords
			if items == "" {
				items = r.Items[0].Label
			}
			line := fmt.Sprintf("%s %s %s %s %s %d/%d", d.Day, r.Work.Label, r.Kind, items, state, r.Met, r.Of)
			if !r.Monitored {
				line += " unmonitored"
			}
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

func sentence(a answer) string {
	switch {
	case a.Sentence == nil:
		return "none"
	case a.Sentence.NeedsYou:
		return "needs you: " + a.Sentence.Words
	}
	return a.Sentence.Words
}

const month = "from=2026-11-16&to=2026-12-16"

func TestGetCalendarCollapsesEachLevelsItemsOfADayIntoARow(t *testing.T) {
	w := newWorld(t)
	got := w.read(t, month)
	want := "2026-11-18 Lantern Fall opening Lantern Fall missing 0/1\n" +
		"2026-11-19 Lantern Fall digital Lantern Fall missing 0/1\n" +
		"2026-11-20 The Attic shipped Box 1 · Things 1 to 10 missing 8/10\n" +
		"2026-11-20 The Attic shipped Box 2 · Things 1, 2 and 4 missing 0/3\n" +
		"2026-11-21 Lantern Fall disc Lantern Fall missing 0/1"
	if rows(got) != want {
		t.Fatalf("a month's rows =\n%s\nwant\n%s", rows(got), want)
	}
	if s, want := sentence(got), "13 things and 1 film come out this week."; s != want {
		t.Errorf("the sentence = %q, want %q", s, want)
	}
	shown := w.read(t, month+"&unmonitored=true")
	if want := "2026-11-20 The Attic shipped Box 2 · Thing 3 missing 0/1 unmonitored"; !strings.Contains(rows(shown), want) {
		t.Errorf("rows with unmonitored Items =\n%s\nwant a row of their own\n%s", rows(shown), want)
	}
	if n := len(shown.Days[2].Rows); n != 3 {
		t.Errorf("2026-11-20 with unmonitored Items has %d rows, want 3", n)
	}
	if labels := shown.Days[2].Rows[1].Items; labels[0].Label != "Thing 1 · Part 1" || labels[2].Label != "Thing 4 · Part 5" {
		t.Errorf("Box 2's Items are labeled %v, want them numbered by their place in the box now that Part 3 is withdrawn", labels)
	}
}

func TestGetCalendarAnswersOneTypesKindsAndNeverAsksAnOffType(t *testing.T) {
	w := newWorld(t)
	w.s.Expect(t, http.MethodGet, "/api/v1/calendar?from=2026-11-16&to=2026-11-20&contentType=films&kinds=opening,disc", "", http.StatusOK,
		`{"today":"2026-11-17","nextDayAt":"T","sentence":{"needsYou":false,"words":"1 film comes out this week."},"days":[`+
			`{"day":"2026-11-18","rows":[{"work":{"id":"id1","contentType":"films","label":"Lantern Fall","posterUrl":null},"kind":"opening","kindLabel":"In cinemas",`+
			`"items":[{"id":"id2","label":"Lantern Fall"}],"itemsWords":"","state":"missing","met":0,"of":1,"monitored":true}]}]}`)
	if got := rows(w.read(t, month+"&kinds=shipped")); strings.Contains(got, "Lantern Fall") || !strings.Contains(got, "The Attic") {
		t.Errorf("kinds=shipped answers\n%s\nwant only The Attic's rows", got)
	}
	if n := w.asked.Load(); n != 0 {
		t.Errorf("the Calendar asked Crates, which is off, %d times, want never", n)
	}
	for _, c := range []struct{ query, want string }{
		{"contentType=crates", `{"location":"query.contentType","message":"Crates is turned off. Turn it on under Kinds of media first."}`},
		{"contentType=boats", `{"location":"query.contentType","message":"There's no kind of media called boats."}`},
		{"kinds=shipped,boxed", `{"location":"query.kinds","message":"There's no release date kind called boxed. Choose one from the Calendar's filter."}`},
	} {
		w.s.Expect(t, http.MethodGet, "/api/v1/calendar?"+month+"&"+c.query, "", http.StatusUnprocessableEntity,
			`{"status":422,"code":"invalid","title":"Some fields have values SlipStream can't accept","errors":[`+c.want+`]}`)
	}
}

func TestGetCalendarRefusesASpanOfMoreThan42DaysOrNoDays(t *testing.T) {
	w := newWorld(t)
	for _, c := range []struct{ query, want string }{
		{"from=2026-11-16&to=2026-12-29", "A calendar request can span at most 42 days. Ask for a shorter span."},
		{"from=2026-11-16&to=2026-11-16", "The span has no days. Ask for an end day after the start day."},
	} {
		w.s.Expect(t, http.MethodGet, "/api/v1/calendar?"+c.query, "", http.StatusUnprocessableEntity,
			`{"status":422,"code":"invalid","title":"Some fields have values SlipStream can't accept","errors":[{"location":"query.to","message":"`+c.want+`"}]}`)
	}
	if code, _ := w.s.Call(t, http.MethodGet, "/api/v1/calendar?from=2026-11-16&to=2026-12-28", ""); code != http.StatusOK {
		t.Errorf("a 42-day span = %d, want 200", code)
	}
	if code, body := w.s.Call(t, http.MethodGet, "/api/v1/calendar?from=2026-11-31&to=2026-12-01", ""); code != http.StatusUnprocessableEntity || !strings.Contains(body, `"location":"query.from"`) {
		t.Errorf("a day that doesn't exist = %d %s, want 422 at query.from", code, body)
	}
}

func TestGetCalendarFollowsTheControllersMidnight(t *testing.T) {
	w := newWorld(t)
	got := w.read(t, month)
	if got.Today != "2026-11-17" || got.NextDayAt != "2026-11-18T08:00:00Z" {
		t.Fatalf("at 22:00 on 2026-11-17, 06:00 UTC the next day, today = %s and nextDayAt = %s, want 2026-11-17 and 2026-11-18T08:00:00Z", got.Today, got.NextDayAt)
	}
	w.now = time.Date(2026, 11, 21, 0, 0, 0, 0, zone)
	got = w.read(t, month)
	if got.Today != "2026-11-21" || got.NextDayAt != "2026-11-22T08:00:00Z" {
		t.Errorf("at midnight on 2026-11-21 today = %s and nextDayAt = %s, want 2026-11-21 and 2026-11-22T08:00:00Z", got.Today, got.NextDayAt)
	}
	if s, want := sentence(got), "needs you: 5 things and 1 film are out and still missing."; s != want {
		t.Errorf("once the Items are out the sentence = %q, want %q, counting Lantern Fall once across its 3 rows", s, want)
	}
}

func TestGetCalendarMovesARowsStateAndNotItsDayWhenATimingChanges(t *testing.T) {
	w := newWorld(t)
	later := targets.Timing{Released: release.Known(works.Date{Year: 2026, Month: 12, Day: 1})}
	for _, it := range w.attic.Items[8:10] {
		w.fx.SetTiming(it.ID, later)
	}
	got := rows(w.read(t, month))
	if want := "2026-11-20 The Attic shipped Box 1 · Things 1 to 10 unreleased 8/10"; !strings.Contains(got, want) {
		t.Fatalf("once Things 9 and 10 come out later, rows =\n%s\nwant\n%s", got, want)
	}
}
