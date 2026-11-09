package release

import "strconv"

// Claim is what a Work says about itself for matching a name.
type Claim struct {
	// IDs are the Work's ids, by Torznab's names: tmdbid, imdbid, tvdbid.
	IDs map[string]string
	// Titles are the Work's label, then its other titles.
	Titles []string
	Year   Fact[int]
	// Slack is how many years a name's year may differ from Year: the
	// type's "Release year can differ by".
	Slack int
}

type Matching uint8

const (
	Matched Matching = iota + 1
	// WrongWork is an id that names another Work: a hard refusal.
	WrongWork
	// WrongTitle is no title, or a year, that matches the Work: a soft
	// refusal.
	WrongTitle
)

// Match judges a name against w. ids are the Indexer's and the name's id
// tags. A conflicting id is WrongWork and a shared id matches. Otherwise
// any of the name's titles equal to any of w's under names' clean rule
// matches when the years, both known, differ by at most Slack. It answers
// the title and year as the name gives them, for the reason's value:
// "Heat (1986)".
func Match(n Name, ids map[string]string, w Claim, names Names) (m Matching, given string) {
	shared := false
	for k, v := range ids {
		want, ok := w.IDs[k]
		if ok && want != v {
			return WrongWork, ""
		}
		shared = shared || ok
	}
	if shared {
		return Matched, ""
	}
	given = givenTitle(n)
	if !titled(n, w, names) {
		return WrongTitle, given
	}
	if y, ok := n.Year.Get(); ok {
		if want, ok := w.Year.Get(); ok && abs(y-want) > w.Slack {
			return WrongTitle, given
		}
	}
	return Matched, ""
}

func titled(n Name, w Claim, names Names) bool {
	for _, t := range n.Titles {
		clean := names.CleanTitle(t)
		for _, want := range w.Titles {
			if clean == names.CleanTitle(want) {
				return true
			}
		}
	}
	return false
}

func givenTitle(n Name) string {
	if len(n.Titles) == 0 {
		return ""
	}
	if y, ok := n.Year.Get(); ok {
		return n.Titles[0] + " (" + strconv.Itoa(y) + ")"
	}
	return n.Titles[0]
}

func abs(n int) int { return max(n, -n) }
