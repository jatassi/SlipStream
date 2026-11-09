package metadata

import (
	"strconv"
	"strings"
	"time"
)

// SearchRead is one read a Source's Search makes of its provider: a query,
// and the year to filter it to, 0 for none.
type SearchRead struct {
	Query string
	Year  int
}

// firstFilm is the year of the earliest film a provider dates.
const firstFilm = 1874

// SearchReads are the reads a Search of query makes. A query ending in a
// year is read twice, first as the title before it filtered to that year,
// then as typed, so "Heat 1995" answers that year's film first and "Blade
// Runner 2049" still finds the film with that name. A query that is only
// a year, such as "1917", has no title to filter and is read once, as is
// one whose year falls before the first film or more than ten years after
// now.
func SearchReads(query string, now time.Time) []SearchRead {
	words := strings.Fields(query)
	if len(words) < 2 {
		return []SearchRead{{Query: query}}
	}
	last := words[len(words)-1]
	y, err := strconv.Atoi(last)
	if err != nil || len(last) != 4 || y < firstFilm || y > now.Year()+10 {
		return []SearchRead{{Query: query}}
	}
	return []SearchRead{{Query: strings.Join(words[:len(words)-1], " "), Year: y}, {Query: query}}
}
