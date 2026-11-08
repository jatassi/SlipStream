package release

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
// tags. A shared id matches, a conflicting id is WrongWork, and otherwise
// any title equal to any of w's under names' clean rule matches when the
// years, both known, differ by at most Slack. It answers the title and
// year as the name gives them, for the reason's value: "Heat (1986)".
// Until titles are compared, every name without a conflicting id matches.
func Match(_ Name, ids map[string]string, w Claim, _ Names) (m Matching, given string) {
	for k, v := range ids {
		if want, ok := w.IDs[k]; ok && want != v {
			return WrongWork, ""
		}
	}
	return Matched, ""
}
