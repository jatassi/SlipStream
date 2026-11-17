package calendar

import (
	"cmp"
	"slices"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/works"
)

// rowKey is what one row holds: a Work's Items of one level, kind and day,
// monitored or not. An Item without a level is a row of its own.
type rowKey struct {
	day       Day
	work      works.WorkID
	level     works.LevelID
	item      works.ItemID
	kind      works.DatedKey
	monitored bool
}

func keyOf(e entry) rowKey {
	k := rowKey{day: e.Day, work: e.Work, level: e.item.Item.LevelID, kind: e.Kind, monitored: e.item.Item.Monitored}
	if k.level.IsZero() {
		k.item = e.Item
	}
	return k
}

// collapse is es as rows by day, with no threshold: Items of one Work,
// level, kind and day are one row however many there are. es is in day,
// Work and tree order, so each row's Items are in tree order.
func collapse(es []entry) []DayRows {
	at := map[rowKey]int{}
	var groups [][]entry
	for _, e := range es {
		k := keyOf(e)
		i, ok := at[k]
		if !ok {
			i = len(groups)
			at[k] = i
			groups = append(groups, nil)
		}
		groups[i] = append(groups[i], e)
	}
	slices.SortStableFunc(groups, func(a, b []entry) int {
		x, y := a[0], b[0]
		return cmp.Or(x.Day.Compare(y.Day), cmp.Compare(x.work.rank, y.work.rank),
			cmp.Compare(kindIndex(x), kindIndex(y)), cmp.Compare(x.item.index, y.item.index),
			cmp.Compare(monitoredFirst(x), monitoredFirst(y)))
	})
	var out []DayRows
	for _, g := range groups {
		if n := len(out); n == 0 || out[n-1].Day != g[0].Day {
			out = append(out, DayRows{Day: g[0].Day})
		}
		out[len(out)-1].Rows = append(out[len(out)-1].Rows, rowOf(g))
	}
	return out
}

func kindIndex(e entry) int {
	return slices.IndexFunc(e.work.desc.Dated, func(k works.DatedKind) bool { return k.Key == e.Kind })
}

func monitoredFirst(e entry) int {
	if e.item.Item.Monitored {
		return 0
	}
	return 1
}

// rowOf is g's row. Its State is the first of its Items' states in the one
// status order.
func rowOf(g []entry) Row {
	first, w := g[0], g[0].work
	row := Row{
		Work: w.row.ID, Type: w.row.ContentType, Label: w.row.Label, Poster: w.row.Poster,
		Kind: w.desc.Dated[kindIndex(first)], Monitored: first.item.Item.Monitored,
		ItemsWords: itemsWords(w, first.item.Item.LevelID, g),
	}
	for _, e := range g {
		row.Items = append(row.Items, ItemRef{ID: e.Item, Label: itemLabel(w, e.item)})
		s := e.item.Status
		if s == 0 {
			continue
		}
		row.Of++
		if s == files.Met {
			row.Met++
		}
		if row.State == 0 || s < row.State {
			row.State = s
		}
	}
	return row
}
