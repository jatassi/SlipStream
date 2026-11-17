package calendar

import (
	"reflect"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

var (
	shippedKind = works.DatedKind{Key: "shipped", Label: "Ship date"}
	shelves     = works.Description{
		Name: "Shelves", Work: works.Noun{One: "Shelf", Many: "Shelves"}, Levels: []works.Noun{{One: "Box", Many: "Boxes"}},
		Item: works.Noun{One: "Thing", Many: "Things"}, Items: works.ItemsMany, Dated: []works.DatedKind{shippedKind},
	}
	single = works.Description{
		Name: "Films", Work: works.Noun{One: "Film", Many: "Films"}, Item: works.Noun{One: "Film", Many: "Films"},
		Items: works.ItemsOne, Dated: []works.DatedKind{{Key: "out", Label: "Out"}, {Key: "home", Label: "Home"}},
	}
	shipped = shippedKind.Key
	nov17   = NewDay(2026, time.November, 17)
)

// shelf is a Work of shelves with one level per box, each holding items
// Things whose states are its next states, Met when they run out.
type shelf struct {
	w     *work
	boxes [][]works.ItemID
}

func newShelf(rank int, label string, boxes ...[]files.State) shelf {
	s := shelf{}
	row := works.Row{Bare: works.Bare{Summary: works.Summary{ID: works.WorkID{UUID: storage.NewUUID()}, ContentType: "shelves", Label: label}}}
	var st files.WorkStates
	for b, states := range boxes {
		level := works.Level{ID: works.LevelID{UUID: storage.NewUUID()}, Label: "Box " + string(rune('1'+b))}
		st.Levels = append(st.Levels, level)
		var ids []works.ItemID
		for _, state := range states {
			it := works.Item{ID: works.ItemID{UUID: storage.NewUUID()}, LevelID: level.ID, Monitored: true}
			st.Items = append(st.Items, files.ItemState{Item: it, Status: state})
			ids = append(ids, it.ID)
		}
		s.boxes = append(s.boxes, ids)
	}
	s.w = newWork(row, rank, shelves, st)
	return s
}

func (s shelf) dated(box int, day Day, places ...int) []Dated {
	var out []Dated
	for _, p := range places {
		out = append(out, Dated{Work: s.w.row.ID, Item: s.boxes[box][p-1], Kind: shipped, Day: day})
	}
	return out
}

func states(n int, s files.State) []files.State {
	out := make([]files.State, n)
	for i := range out {
		out[i] = s
	}
	return out
}

func of(ws ...*work) map[works.WorkID]*work {
	out := map[works.WorkID]*work{}
	for _, w := range ws {
		out[w.row.ID] = w
	}
	return out
}

func rowsOf(t *testing.T, days []DayRows) []Row {
	t.Helper()
	var out []Row
	for _, d := range days {
		out = append(out, d.Rows...)
	}
	return out
}

func TestALevelOnOneDayIsOneRowWhateverItsCount(t *testing.T) {
	s := newShelf(0, "Attic", states(10, files.Met), states(3, files.Met))
	days := collapse(kept(append(s.dated(0, nov17, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10), s.dated(1, nov17, 1, 2)...), of(s.w), false))
	if len(days) != 1 || days[0].Day != nov17 {
		t.Fatalf("days = %+v, want only %s", days, nov17)
	}
	rows := days[0].Rows
	if len(rows) != 2 || rows[0].ItemsWords != "Box 1 · Things 1 to 10" || len(rows[0].Items) != 10 || rows[1].ItemsWords != "Box 2 · Things 1 and 2" {
		t.Fatalf("rows = %+v, want Box 1's ten Things in one row, then Box 2's two", rows)
	}
	if rows[0].Items[0].Label != "Thing 1" || rows[0].Kind != shippedKind || rows[0].State != files.Met || rows[0].Met != 10 || rows[0].Of != 10 {
		t.Fatalf("row = %+v, want Thing 1 first, Ship date, Met 10 of 10", rows[0])
	}
}

func TestAGapReadsByPlaceInTheLevel(t *testing.T) {
	s := newShelf(0, "Attic", states(5, files.Missing))
	s.w.items[s.boxes[0][2]] = placed{ItemState: files.ItemState{Item: works.Item{ID: s.boxes[0][2], LevelID: s.w.items[s.boxes[0][2]].Item.LevelID, WithdrawnAt: storage.Now()}}, index: 2}
	rows := rowsOf(t, collapse(kept(s.dated(0, nov17, 1, 2, 3, 5), of(s.w), false)))
	if len(rows) != 1 || rows[0].ItemsWords != "Box 1 · Things 1, 2 and 5" || len(rows[0].Items) != 3 {
		t.Fatalf("rows = %+v, want one row of Things 1, 2 and 5, withdrawn Thing 3 left out", rows)
	}
}

func TestAMixedRowReadsItsMostUrgentStateAndCountsTheMet(t *testing.T) {
	s := newShelf(0, "Attic", append(append(states(8, files.Met), files.Missing, files.Downloading), 0))
	rows := rowsOf(t, collapse(kept(s.dated(0, nov17, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11), of(s.w), false)))
	if len(rows) != 1 || rows[0].State != files.Missing || rows[0].Met != 8 || rows[0].Of != 10 || len(rows[0].Items) != 11 {
		t.Fatalf("rows = %+v, want one Missing row, 8 of 10 Met, its Thing with no cell uncounted", rows)
	}
	bare := newShelf(0, "Cellar", []files.State{0})
	if rows := rowsOf(t, collapse(kept(bare.dated(0, nov17, 1), of(bare.w), false))); rows[0].State != 0 || rows[0].Of != 0 {
		t.Fatalf("row = %+v, want no state while no Item has a cell", rows[0])
	}
}

func TestUnmonitoredItemsAreHiddenUnlessAskedAndThenARowOfTheirOwn(t *testing.T) {
	s := newShelf(0, "Attic", states(4, files.Missing))
	for _, id := range s.boxes[0][2:] {
		p := s.w.items[id]
		p.Item.Monitored = false
		s.w.items[id] = p
	}
	dated := s.dated(0, nov17, 1, 2, 3, 4)
	if rows := rowsOf(t, collapse(kept(dated, of(s.w), false))); len(rows) != 1 || rows[0].ItemsWords != "Box 1 · Things 1 and 2" || !rows[0].Monitored {
		t.Fatalf("rows = %+v, want Things 1 and 2 only", rows)
	}
	rows := rowsOf(t, collapse(kept(dated, of(s.w), true)))
	if len(rows) != 2 || !rows[0].Monitored || rows[1].Monitored || rows[1].ItemsWords != "Box 1 · Things 3 and 4" {
		t.Fatalf("rows = %+v, want Things 1 and 2, then the unmonitored 3 and 4 in a row of their own", rows)
	}
}

func TestRowsGoByDayThenWorkThenKindAndAnItemOfOneIsARowPerKind(t *testing.T) {
	film := works.Item{ID: works.ItemID{UUID: storage.NewUUID()}, Label: "Heatwave", Monitored: true}
	f := newWork(works.Row{Bare: works.Bare{Summary: works.Summary{ID: works.WorkID{UUID: storage.NewUUID()}, ContentType: "films", Label: "Heatwave"}}}, 0, single,
		files.WorkStates{Items: []files.ItemState{{Item: film, Status: files.Unreleased}}})
	s := newShelf(1, "Attic", states(1, files.Met))
	dated := append(s.dated(0, nov17, 1),
		Dated{Work: f.row.ID, Item: film.ID, Kind: "home", Day: nov17},
		Dated{Work: f.row.ID, Item: film.ID, Kind: "out", Day: nov17},
		Dated{Work: f.row.ID, Item: film.ID, Kind: "home", Day: nov17.Add(-3)},
	)
	days := collapse(kept(dated, of(f, s.w), false))
	var got [][]string
	for _, d := range days {
		var labels []string
		for _, r := range d.Rows {
			labels = append(labels, d.Day.String()+" "+r.Label+" "+r.Kind.Label+" "+r.ItemsWords+r.Items[0].Label)
		}
		got = append(got, labels)
	}
	want := [][]string{{"2026-11-14 Heatwave Home Heatwave"}, {"2026-11-17 Heatwave Out Heatwave", "2026-11-17 Heatwave Home Heatwave", "2026-11-17 Attic Ship date Box 1 · Thing 1Thing 1"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rows = %q, want %q", got, want)
	}
}
