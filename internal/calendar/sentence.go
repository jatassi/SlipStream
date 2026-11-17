package calendar

import (
	"cmp"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/works"
)

// sentenceOf leads with what needs attention: es's Items dated on or
// before today that are Missing or Failed. Else it names soon's Items, the
// next 7 days'. Else it is nil. Each counts an Item once, whatever its
// rows. It counts only monitored Items, whatever the rows show, since
// Wanted, where it links, lists only those.
func sentenceOf(today Day, es, soon []entry) *Sentence {
	out := slices.DeleteFunc(slices.Clone(es), func(e entry) bool {
		return !e.item.Item.Monitored || e.Day.Compare(today) > 0 || (e.item.Status != files.Missing && e.item.Status != files.Failed)
	})
	if words, one := tally(out); words != "" {
		return &Sentence{NeedsYou: true, Words: words + verb(one, " is out and still missing.", " are out and still missing.")}
	}
	ahead := slices.DeleteFunc(slices.Clone(soon), func(e entry) bool { return !e.item.Item.Monitored })
	if words, one := tally(ahead); words != "" {
		return &Sentence{Words: words + verb(one, " comes out this week.", " come out this week.")}
	}
	return nil
}

// tally is es's Items counted once each per type with the type's Item
// noun, largest count first: "2 episodes and 1 movie". one is whether it
// names a single Item.
func tally(es []entry) (words string, one bool) {
	items := map[works.TypeID]map[works.ItemID]bool{}
	nouns := map[works.TypeID]works.Noun{}
	for _, e := range es {
		t := e.work.row.ContentType
		if items[t] == nil {
			items[t] = map[works.ItemID]bool{}
		}
		items[t][e.Item] = true
		nouns[t] = e.work.desc.Item
	}
	types := slices.SortedFunc(maps.Keys(items), func(a, b works.TypeID) int {
		return cmp.Or(cmp.Compare(len(items[b]), len(items[a])), cmp.Compare(a, b))
	})
	parts := make([]string, len(types))
	for i, t := range types {
		n, noun := len(items[t]), nouns[t].Many
		if n == 1 {
			noun = nouns[t].One
		}
		parts[i] = strconv.Itoa(n) + " " + strings.ToLower(noun)
	}
	return works.JoinWords(parts), len(types) == 1 && len(items[types[0]]) == 1
}

func verb(one bool, singular, plural string) string {
	if one {
		return singular
	}
	return plural
}
