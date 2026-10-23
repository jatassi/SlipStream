package health

import (
	"slices"
	"strings"
	"time"
)

// row is one open item as health_item keeps it.
type row struct {
	Level            Level
	Since, ChangedAt time.Time
}

type change struct {
	Key      Key
	Kind     Kind
	From, To Level
	// Row is the row to write; zero for Cleared.
	Row row
}

// diff is one category's changes, in key order. An answer that failed
// changes nothing. An unsure key keeps its row and raises none. A text
// change at one level is no change. A level change keeps since and moves
// changed_at.
func diff(rows map[Key]row, a Answer, failed bool, now time.Time) []change {
	if failed {
		return nil
	}
	var out []change
	answered := map[Key]bool{}
	for _, u := range a.Unsure {
		answered[u.Key] = true
	}
	for _, it := range a.Items {
		answered[it.Key] = true
		r, open := rows[it.Key]
		switch {
		case !open:
			out = append(out, change{Key: it.Key, Kind: Raised, To: it.Level, Row: row{Level: it.Level, Since: now, ChangedAt: now}})
		case it.Level != r.Level:
			kind := Better
			if it.Level > r.Level {
				kind = Worse
			}
			out = append(out, change{Key: it.Key, Kind: kind, From: r.Level, To: it.Level, Row: row{Level: it.Level, Since: r.Since, ChangedAt: now}})
		}
	}
	for k, r := range rows {
		if !answered[k] {
			out = append(out, change{Key: k, Kind: Cleared, From: r.Level})
		}
	}
	slices.SortFunc(out, func(a, b change) int { return strings.Compare(string(a.Key), string(b.Key)) })
	return out
}

// applied is rows after changes.
func applied(rows map[Key]row, changes []change) map[Key]row {
	out := make(map[Key]row, len(rows))
	for k, r := range rows {
		out[k] = r
	}
	for _, c := range changes {
		if c.Kind == Cleared {
			delete(out, c.Key)
		} else {
			out[c.Key] = c.Row
		}
	}
	return out
}
