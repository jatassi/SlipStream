package works_test

import (
	"errors"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

var (
	flatShape = works.Description{
		Name: "Films",
		Work: works.Noun{One: "Film", Many: "Films"},
		Item: works.Noun{One: "Cut", Many: "Cuts"},
		Fields: []works.FieldSpec{
			{Key: "year", Label: "Year", Tier: works.TierWork, Kind: works.KindNumber},
			{Key: "runtime", Label: "Runtime", Tier: works.TierItem, Kind: works.KindDuration},
		},
	}
	seasonalShape = works.Description{
		Name:   "Shows",
		Work:   works.Noun{One: "Show", Many: "Shows"},
		Levels: []works.Noun{{One: "Season", Many: "Seasons"}},
		Item:   works.Noun{One: "Episode", Many: "Episodes"},
		Fields: []works.FieldSpec{
			{Key: "number", Label: "Number", Tier: works.TierLevel(1), Kind: works.KindNumber},
			{Key: "aired", Label: "Aired", Tier: works.TierItem, Kind: works.KindDate},
		},
	}
	nestedShape = works.Description{
		Name:   "Serials",
		Work:   works.Noun{One: "Show", Many: "Shows"},
		Levels: []works.Noun{{One: "Arc", Many: "Arcs"}, {One: "Part", Many: "Parts"}},
		Item:   works.Noun{One: "Episode", Many: "Episodes"},
		Fields: []works.FieldSpec{
			{Key: "arc_no", Label: "Arc", Tier: works.TierLevel(1), Kind: works.KindNumber},
			{Key: "part_no", Label: "Part", Tier: works.TierLevel(2), Kind: works.KindNumber},
		},
	}
)

func shaped(t testing.TB, id works.TypeID, d works.Description) *harness {
	t.Helper()
	fx := contenttypetest.New()
	_, h := open(t, storagetest.Library(t, works.Schema()), map[works.TypeID]works.Catalog{id: described{fx, d}})
	h.fx = fx
	return h
}

func leaf(key works.Key, label string) works.Entry { return works.Entry{Key: key, Label: label} }

func branch(key works.Key, label string, children ...works.Entry) works.Entry {
	return works.Entry{Key: key, Label: label, Children: children}
}

func expectTree(t *testing.T, step string, tree works.Tree, err error, want string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", step, err)
	}
	if got := render(tree); got != want {
		t.Fatalf("%s: tree =\n%s\nwant\n%s", step, got, want)
	}
	liveParents(t, step, tree)
}

func liveParents(t *testing.T, step string, tree works.Tree) {
	t.Helper()
	gone := map[works.LevelID]bool{}
	for _, l := range tree.Levels {
		gone[l.ID] = !l.WithdrawnAt.IsZero()
		if l.WithdrawnAt.IsZero() && gone[l.ParentID] {
			t.Errorf("%s: live level %s sits under a withdrawn level", step, l.Key)
		}
	}
	for _, it := range tree.Items {
		if it.WithdrawnAt.IsZero() && gone[it.LevelID] {
			t.Errorf("%s: live item %s sits under a withdrawn level", step, it.Key)
		}
	}
}

func withdrawnAt(tree works.Tree) map[string]storage.Time {
	out := map[string]storage.Time{}
	for _, l := range tree.Levels {
		if !l.WithdrawnAt.IsZero() {
			out["level "+string(l.Key)] = l.WithdrawnAt
		}
	}
	for _, it := range tree.Items {
		if !it.WithdrawnAt.IsZero() {
			out["item "+string(it.Key)] = it.WithdrawnAt
		}
	}
	return out
}

func film(cuts ...works.Entry) contenttypetest.Record {
	return contenttypetest.Record{
		Tree:       works.Fetched{Label: "The Long Night", Children: cuts},
		WorkFields: []works.FieldValue{works.NumberField("year", 1987)},
		ItemFields: map[works.Key][]works.FieldValue{
			"k1": {works.DurationField("runtime", 118*time.Minute)},
			"k2": {works.DurationField("runtime", 131*time.Minute)},
		},
	}
}

func TestAFlatWorkKeepsItsItemsThroughRefreshes(t *testing.T) {
	h := shaped(t, "flat", flatShape)
	h.fx.Put("f1", film(leaf("k1", "Theatrical"), leaf("k2", "Director's Cut")))
	first, _, err := h.works.Add(t.Context(), admin, "flat", "f1", false)
	expectTree(t, "add", first, err, `f1 "The Long Night" sort "The Long Night" year=1987
  item k1 in  #1 "Theatrical" runtime=1h58m0s
  item k2 in  #2 "Director's Cut" runtime=2h11m0s
`)
	if first.Items[0].LevelID != (works.LevelID{}) {
		t.Fatalf("a Cut of a type with no levels sits in level %s", first.Items[0].LevelID)
	}

	h.fx.Put("f1", film(leaf("k1", "Theatrical Cut")))
	second, err := h.works.Refresh(t.Context(), admin, first.ID)
	expectTree(t, "rename and withdraw", second, err, `f1 "The Long Night" sort "The Long Night" year=1987
  item k1 in  #1 "Theatrical Cut" runtime=1h58m0s
  item k2 in  #2 "Director's Cut" withdrawn runtime=2h11m0s
`)
	sameIDs(t, first, second)

	h.fx.Put("f1", film(leaf("k1", "Theatrical Cut"), leaf("k3", "Extended"), leaf("k2", "Director's Cut")))
	third, err := h.works.Refresh(t.Context(), admin, first.ID)
	expectTree(t, "return", third, err, `f1 "The Long Night" sort "The Long Night" year=1987
  item k1 in  #1 "Theatrical Cut" runtime=1h58m0s
  item k3 in  #2 "Extended"
  item k2 in  #3 "Director's Cut" runtime=2h11m0s
`)
	sameIDs(t, first, third)
	got, err := h.works.Get(t.Context(), admin, first.ID)
	if err != nil || encode(t, got) != encode(t, third) {
		t.Fatalf("get = %s, %v; want the refreshed tree %s", encode(t, got), err, encode(t, third))
	}
}

func TestAFlatWorkAnnouncedWithNoItemsGainsOne(t *testing.T) {
	h := shaped(t, "flat", flatShape)
	h.fx.Put("f2", film())
	first, _, err := h.works.Add(t.Context(), admin, "flat", "f2", false)
	expectTree(t, "announced", first, err, `f2 "The Long Night" sort "The Long Night" year=1987
`)
	if first.Items != nil || first.Levels != nil {
		t.Fatalf("an announced Work holds levels %v and items %v", first.Levels, first.Items)
	}
	h.fx.Put("f2", film(leaf("k1", "Theatrical")))
	second, err := h.works.Refresh(t.Context(), admin, first.ID)
	expectTree(t, "released", second, err, `f2 "The Long Night" sort "The Long Night" year=1987
  item k1 in  #1 "Theatrical" runtime=1h58m0s
`)
	sameIDs(t, first, second)
}

func TestAWithdrawnItemFollowsItsLiveSiblings(t *testing.T) {
	h := shaped(t, "flat", flatShape)
	h.fx.Put("f1", film(leaf("k1", "Theatrical"), leaf("k2", "Director's Cut")))
	first, _, err := h.works.Add(t.Context(), admin, "flat", "f1", false)
	if err != nil {
		t.Fatal(err)
	}
	h.fx.Put("f1", film(leaf("k1", "Theatrical"), leaf("k3", "Extended")))
	second, err := h.works.Refresh(t.Context(), admin, first.ID)
	expectTree(t, "replace", second, err, `f1 "The Long Night" sort "The Long Night" year=1987
  item k1 in  #1 "Theatrical" runtime=1h58m0s
  item k3 in  #2 "Extended"
  item k2 in  #2 "Director's Cut" withdrawn runtime=2h11m0s
`)
}

func show() contenttypetest.Record {
	return contenttypetest.Record{
		Tree: works.Fetched{
			Label: "Lamplight Bay",
			Children: []works.Entry{
				branch("sp", "Specials", leaf("x9001", "Christmas Special")),
				branch("sn1", "Season 1", leaf("x1101", "Pilot"), leaf("x1102", "Tides")),
				branch("sn2", "Season 2", leaf("x1201", "Return"), leaf("x1202", "Storm")),
				branch("sn3", "Season 3", leaf("x1301", "Lighthouse")),
			},
		},
		LevelFields: map[works.Key][]works.FieldValue{
			"sp": {works.NumberField("number", 0)}, "sn1": {works.NumberField("number", 1)},
			"sn2": {works.NumberField("number", 2)}, "sn3": {works.NumberField("number", 3)},
		},
		ItemFields: map[works.Key][]works.FieldValue{
			"x9001": {works.DateField("aired", works.Date{Year: 2019, Month: 12, Day: 24})},
			"x1301": {works.DateField("aired", works.Date{Year: 2021, Month: 3})},
		},
	}
}

func TestASeasonalWorkFollowsItsSource(t *testing.T) {
	h := shaped(t, "seasonal", seasonalShape)
	h.fx.Put("s1", show())
	first, _, err := h.works.Add(t.Context(), admin, "seasonal", "s1", false)
	expectTree(t, "add", first, err, `s1 "Lamplight Bay" sort "Lamplight Bay"
  level sp depth 1 under "" #1 "Specials" number=0
  level sn1 depth 1 under "" #2 "Season 1" number=1
  level sn2 depth 1 under "" #3 "Season 2" number=2
  level sn3 depth 1 under "" #4 "Season 3" number=3
  item x9001 in sp #1 "Christmas Special" aired=2019-12-24
  item x1101 in sn1 #1 "Pilot"
  item x1102 in sn1 #2 "Tides"
  item x1201 in sn2 #1 "Return"
  item x1202 in sn2 #2 "Storm"
  item x1301 in sn3 #1 "Lighthouse" aired=2021-03
`)

	next := show()
	next.Tree.Children = []works.Entry{
		branch("sp", "Specials"),
		branch("sn2", "Season 2", leaf("x1201", "Return"), leaf("x1202", "Storm")),
		branch("sn1", "Season 1", leaf("x1101", "Pilot"), leaf("x1102", "Tides")),
		branch("sn3", "Season 3", leaf("x1301", "Lighthouse"), leaf("x1302", "TBA"), leaf("x9001", "Christmas Special"), leaf("x1303", "TBA")),
	}
	h.fx.Put("s1", next)
	second, err := h.works.Refresh(t.Context(), admin, first.ID)
	expectTree(t, "move, reorder, unaired", second, err, `s1 "Lamplight Bay" sort "Lamplight Bay"
  level sp depth 1 under "" #1 "Specials" number=0
  level sn2 depth 1 under "" #2 "Season 2" number=2
  level sn1 depth 1 under "" #3 "Season 1" number=1
  level sn3 depth 1 under "" #4 "Season 3" number=3
  item x1201 in sn2 #1 "Return"
  item x1202 in sn2 #2 "Storm"
  item x1101 in sn1 #1 "Pilot"
  item x1102 in sn1 #2 "Tides"
  item x1301 in sn3 #1 "Lighthouse" aired=2021-03
  item x1302 in sn3 #2 "TBA"
  item x9001 in sn3 #3 "Christmas Special" aired=2019-12-24
  item x1303 in sn3 #4 "TBA"
`)
	sameIDs(t, first, second)

	next.Tree.Children = []works.Entry{
		branch("sp", "Specials"),
		branch("sn1", "Season 1", leaf("x1101", "Pilot"), leaf("x1102", "Tides")),
		branch("sn3", "Season 3", leaf("x1301", "Lighthouse"), leaf("x1302", "Undertow"), leaf("x9001", "Christmas Special"), leaf("x1303", "Ashore")),
	}
	h.fx.Put("s1", next)
	third, err := h.works.Refresh(t.Context(), admin, first.ID)
	expectTree(t, "season withdrawn", third, err, `s1 "Lamplight Bay" sort "Lamplight Bay"
  level sp depth 1 under "" #1 "Specials" number=0
  level sn1 depth 1 under "" #2 "Season 1" number=1
  level sn3 depth 1 under "" #3 "Season 3" number=3
  level sn2 depth 1 under "" #2 "Season 2" withdrawn number=2
  item x1101 in sn1 #1 "Pilot"
  item x1102 in sn1 #2 "Tides"
  item x1301 in sn3 #1 "Lighthouse" aired=2021-03
  item x1302 in sn3 #2 "Undertow"
  item x9001 in sn3 #3 "Christmas Special" aired=2019-12-24
  item x1303 in sn3 #4 "Ashore"
  item x1201 in sn2 #1 "Return" withdrawn
  item x1202 in sn2 #2 "Storm" withdrawn
`)
	sameIDs(t, first, third)
	at := third.RefreshedAt
	want := map[string]storage.Time{"level sn2": at, "item x1201": at, "item x1202": at}
	if got := withdrawnAt(third); encode(t, got) != encode(t, want) {
		t.Fatalf("withdrawn = %v, want %v", got, want)
	}
}

func TestAnEpisodeKeyRepeatedUnderTwoSeasonsIsRefused(t *testing.T) {
	h := shaped(t, "seasonal", seasonalShape)
	r := show()
	r.Tree.Children = []works.Entry{
		branch("sn1", "Season 1", leaf("e1", "Pilot"), leaf("e2", "Tides")),
		branch("sn2", "Season 2", leaf("e1", "Return")),
	}
	h.fx.Put("s1", r)
	_, _, err := h.works.Add(t.Context(), admin, "seasonal", "s1", false)
	if want := "the metadata source sent a record SlipStream can't use: season sn2: two Items have key e1"; err == nil || err.Error() != want || !errors.Is(err, works.ErrInvalid) {
		t.Fatalf("add = %v, want %q", err, want)
	}
}

func serial() contenttypetest.Record {
	return contenttypetest.Record{
		Tree: works.Fetched{
			Label: "The Archive",
			Children: []works.Entry{
				branch("a1", "First Arc",
					branch("p1", "Part One", leaf("e1", "Opening"), leaf("e2", "Descent")),
					branch("p2", "Part Two", leaf("e3", "Reckoning"))),
				branch("a2", "Second Arc",
					branch("p3", "Part Three", leaf("e4", "Aftermath"))),
			},
		},
		LevelFields: map[works.Key][]works.FieldValue{
			"a1": {works.NumberField("arc_no", 1)}, "a2": {works.NumberField("arc_no", 2)},
			"p1": {works.NumberField("part_no", 1)}, "p3": {works.NumberField("part_no", 3)},
		},
	}
}

func TestATwoLevelWorkMovesNodesAcrossParents(t *testing.T) {
	h := shaped(t, "nested", nestedShape)
	h.fx.Put("n1", serial())
	first, _, err := h.works.Add(t.Context(), admin, "nested", "n1", false)
	expectTree(t, "add", first, err, `n1 "The Archive" sort "The Archive"
  level a1 depth 1 under "" #1 "First Arc" arc_no=1
  level a2 depth 1 under "" #2 "Second Arc" arc_no=2
  level p1 depth 2 under "a1" #1 "Part One" part_no=1
  level p2 depth 2 under "a1" #2 "Part Two"
  level p3 depth 2 under "a2" #1 "Part Three" part_no=3
  item e1 in p1 #1 "Opening"
  item e2 in p1 #2 "Descent"
  item e3 in p2 #1 "Reckoning"
  item e4 in p3 #1 "Aftermath"
`)

	moved := serial()
	moved.Tree.Children = []works.Entry{
		branch("a2", "Second Arc",
			branch("p3", "Part Three", leaf("e4", "Aftermath"), leaf("e2", "Descent")),
			branch("p2", "Part Two", leaf("e3", "Reckoning"))),
		branch("a1", "First Arc",
			branch("p1", "Part One", leaf("e1", "Opening"))),
	}
	h.fx.Put("n1", moved)
	second, err := h.works.Refresh(t.Context(), admin, first.ID)
	expectTree(t, "moves", second, err, `n1 "The Archive" sort "The Archive"
  level a2 depth 1 under "" #1 "Second Arc" arc_no=2
  level a1 depth 1 under "" #2 "First Arc" arc_no=1
  level p3 depth 2 under "a2" #1 "Part Three" part_no=3
  level p2 depth 2 under "a2" #2 "Part Two"
  level p1 depth 2 under "a1" #1 "Part One" part_no=1
  item e4 in p3 #1 "Aftermath"
  item e2 in p3 #2 "Descent"
  item e3 in p2 #1 "Reckoning"
  item e1 in p1 #1 "Opening"
`)
	sameIDs(t, first, second)

	moved.Tree.Children = []works.Entry{
		branch("a2", "Second Arc",
			branch("p3", "Part Three", leaf("e4", "Aftermath"), leaf("e2", "Descent")),
			branch("p1", "Part One", leaf("e1", "Opening"))),
	}
	h.fx.Put("n1", moved)
	third, err := h.works.Refresh(t.Context(), admin, first.ID)
	expectTree(t, "arc withdrawn", third, err, `n1 "The Archive" sort "The Archive"
  level a2 depth 1 under "" #1 "Second Arc" arc_no=2
  level a1 depth 1 under "" #2 "First Arc" withdrawn arc_no=1
  level p3 depth 2 under "a2" #1 "Part Three" part_no=3
  level p1 depth 2 under "a2" #2 "Part One" part_no=1
  level p2 depth 2 under "a2" #2 "Part Two" withdrawn
  item e4 in p3 #1 "Aftermath"
  item e2 in p3 #2 "Descent"
  item e1 in p1 #1 "Opening"
  item e3 in p2 #1 "Reckoning" withdrawn
`)
	sameIDs(t, first, third)

	h.fx.Put("n1", serial())
	fourth, err := h.works.Refresh(t.Context(), admin, first.ID)
	expectTree(t, "return", fourth, err, render(first))
	sameIDs(t, first, fourth)
}
