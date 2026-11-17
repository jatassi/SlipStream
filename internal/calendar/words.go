package calendar

import (
	"strconv"

	"github.com/jatassi/slipstream/internal/works"
)

// itemLabel is "Episode 1 · Hello, Ms. Cobel" by the Item's place in its
// level, "Episode 1" when it has no title of its own, and a lone Item's
// label for a type of one Item.
func itemLabel(w *work, it placed) string {
	if w.desc.Items == works.ItemsOne {
		return it.Item.Label
	}
	words := w.desc.Item.One + " " + strconv.Itoa(w.places[it.Item.ID])
	if it.Item.Label == "" || it.Item.Label == words {
		return words
	}
	return words + " · " + it.Item.Label
}

// itemsWords is a row's Items with their level, "Season 2 · Episodes 1 to
// 10", by each Item's place in its level, as Imported entries number them.
// A row without a level holds one Item and has no words.
func itemsWords(w *work, level works.LevelID, g []entry) string {
	if level.IsZero() {
		return ""
	}
	places := make([]int, len(g))
	for i, e := range g {
		places[i] = w.places[e.Item]
	}
	return w.levels[level] + " · " + works.NumbersWords(w.desc.Item, places)
}
