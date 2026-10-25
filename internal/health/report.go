package health

import (
	"cmp"
	"slices"
	"strings"
	"time"
)

// Report is what is wrong, as the last evaluation of each category left it.
// Every read serves it, so the page, the nav dots and the volume bars
// never disagree.
type Report struct {
	// Level is the worst over every category.
	Level Level
	// First is the item the status sentence leads with: the worst level,
	// then the oldest since, then registry order. Nil when nothing is wrong.
	First      *Pointer
	Categories []CategoryReport
}

type Pointer struct {
	Category ID
	Key      Key
	Text     string
}

type CategoryReport struct {
	About
	CanCheck bool
	Level    Level
	// Unread is why the last answer failed, "" when it didn't. Items then
	// are the open rows with the last words this process saw.
	Unread string
	// Items come errors first, then oldest since, then by key.
	Items []Shown
}

type Shown struct {
	Item
	Since, ChangedAt time.Time
	// Unsure: the owner can't judge it now, so it reads as it last did.
	Unsure bool
}

// Worst is the worst level among ids, or among every category when ids is
// empty.
func (r Report) Worst(ids ...ID) Level {
	if len(ids) == 0 {
		return r.Level
	}
	worst := None
	for _, c := range r.Categories {
		if slices.Contains(ids, c.ID) {
			worst = max(worst, c.Level)
		}
	}
	return worst
}

func (r Report) item(id ID, key Key) (Shown, bool) {
	for _, c := range r.Categories {
		if c.ID != id {
			continue
		}
		for _, s := range c.Items {
			if s.Key == key {
				return s, true
			}
		}
	}
	return Shown{}, false
}

func reportOf(cats []CategoryReport) Report {
	r := Report{Categories: cats}
	var first *Shown
	for _, c := range cats {
		r.Level = max(r.Level, c.Level)
		if len(c.Items) == 0 {
			continue
		}
		lead := c.Items[0]
		if first == nil || lead.Level > first.Level || (lead.Level == first.Level && lead.Since.Before(first.Since)) {
			first = &lead
			r.First = &Pointer{Category: c.ID, Key: lead.Key, Text: lead.Text}
		}
	}
	return r
}

// shownOf is one category as its rows and its answer show it. An answer
// that failed shows the rows with words from last.
func shownOf(c Category, rows map[Key]row, a Answer, failed error, last map[Key]Says) CategoryReport {
	out := CategoryReport{About: c.About, CanCheck: c.Check != nil}
	if failed != nil {
		out.Unread = failed.Error()
	}
	words := map[Key]Says{}
	unsure := map[Key]bool{}
	if failed == nil {
		for _, it := range a.Items {
			words[it.Key] = it.Says
		}
		for _, u := range a.Unsure {
			words[u.Key] = u.Says
			unsure[u.Key] = true
		}
	}
	for k, r := range rows {
		says, ok := words[k]
		if !ok {
			says, ok = last[k]
		}
		if !ok {
			says = Says{Subject: c.Label, Text: "SlipStream couldn't check this item. This shows the last known state. The Controller's log has the cause."}
		}
		out.Items = append(out.Items, Shown{Item: Item{Key: k, Level: r.Level, Says: says}, Since: r.Since, ChangedAt: r.ChangedAt, Unsure: unsure[k]})
		out.Level = max(out.Level, r.Level)
	}
	slices.SortFunc(out.Items, func(a, b Shown) int {
		return cmp.Or(cmp.Compare(b.Level, a.Level), a.Since.Compare(b.Since), strings.Compare(string(a.Key), string(b.Key)))
	})
	return out
}
