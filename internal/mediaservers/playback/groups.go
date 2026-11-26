package playback

import (
	"cmp"
	"slices"
	"strings"
)

// groupsOf groups a player's judged plays by their set of blocking
// dimensions and values, largest first, ties in dimension table order.
func groupsOf(judged []judgedPlay) []Group {
	out := []Group{}
	at := map[string]int{}
	for _, j := range judged {
		k := keyOf(j.dims)
		i, ok := at[k]
		if !ok {
			i = len(out)
			at[k] = i
			out = append(out, Group{
				Dimensions: j.dims, Cause: causeOf(j.dims), Reasons: []string{}, Targets: []TargetRef{}, Outcomes: []Outcome{},
			})
		}
		g := &out[i]
		g.Plays++
		for _, r := range j.play.Reasons {
			if !slices.Contains(g.Reasons, r) {
				g.Reasons = append(g.Reasons, r)
			}
		}
		for _, t := range j.play.Snapshot.Targets {
			if !slices.Contains(g.Targets, t) {
				g.Targets = append(g.Targets, t)
			}
		}
		g.Outcomes = withOutcome(g.Outcomes, j.outcome)
	}
	for i := range out {
		slices.SortStableFunc(out[i].Targets, func(a, b TargetRef) int { return cmp.Compare(a.Name, b.Name) })
		slices.SortStableFunc(out[i].Outcomes, func(a, b Outcome) int { return cmp.Compare(b.Plays, a.Plays) })
	}
	slices.SortStableFunc(out, func(a, b Group) int {
		return cmp.Or(cmp.Compare(b.Plays, a.Plays), compareDims(a.Dimensions, b.Dimensions))
	})
	return out
}

// keyOf is the set of dims' dimensions and values, which a group shares.
func keyOf(dims []Blocked) string {
	var b strings.Builder
	for _, d := range dims {
		b.WriteString(string(d.Dimension) + "=" + d.Value + "\x00")
	}
	return b.String()
}

// causeOf is the dimensions a player's line names: the first no Target
// fixes, else at most two.
func causeOf(dims []Blocked) []Blocked {
	for _, d := range dims {
		if !d.Choosable {
			return []Blocked{d}
		}
	}
	return slices.Clone(dims[:min(2, len(dims))])
}

// withOutcome is os counting o's play, in the outcome of its kind that
// names the same version, suggestion or Target, with o's Work line when
// its Work is new to it.
func withOutcome(os []Outcome, o Outcome) []Outcome {
	k := outcomeKey(o)
	for i := range os {
		if outcomeKey(os[i]) == k {
			os[i].Plays++
			for _, l := range o.Works {
				if !slices.ContainsFunc(os[i].Works, func(m WorkLine) bool { return m.WorkID == l.WorkID }) {
					os[i].Works = append(os[i].Works, l)
				}
			}
			return os
		}
	}
	o.Plays = 1
	return append(os, o)
}

func outcomeKey(o Outcome) string {
	k := string(o.Kind)
	switch {
	case o.Version != nil:
		k += "\x00" + o.Version.FileID
	case o.Suggestion != nil:
		k += "\x00" + o.Suggestion.Name
	case o.Target != nil:
		k += "\x00" + o.Target.ID
	case o.Missing != nil:
		k += "\x00" + keyOf([]Blocked{*o.Missing})
	}
	return k
}

// compareDims orders two dimension sets by the dimension table,
// then by value.
func compareDims(a, b []Blocked) int {
	for i := range min(len(a), len(b)) {
		if c := cmp.Or(cmp.Compare(order(a[i].Dimension), order(b[i].Dimension)), cmp.Compare(a[i].Value, b[i].Value)); c != 0 {
			return c
		}
	}
	return cmp.Compare(len(a), len(b))
}
