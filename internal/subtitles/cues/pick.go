package cues

import (
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/language"
)

var (
	nameSE      = regexp.MustCompile(`(?i)\bs(\d{1,2})[ ._-]*e(\d{1,3})`)
	nameMore    = regexp.MustCompile(`(?i)^(?:[ ._]*(-)?[ ._]*e|[ ._]*(-)[ ._]*)(\d{1,3})(?:\D|$)`)
	nameX       = regexp.MustCompile(`(?i)\b(\d{1,2})x(\d{2,3})\b`)
	nameEpisode = regexp.MustCompile(`(?i)\be(?:p|pisode)?[ ._]*(\d{1,3})\b`)
	nameLeading = regexp.MustCompile(`^(\d{1,3})(?:[ ._-]|$)`)
)

// numbering is what a member's name says of its season and episodes.
// season is zero when the name gives none.
type numbering struct {
	season   int
	episodes []int
}

// numbered reads a member's base name, with either path separator, as
// S01E02, S01E01E02, S01E01-02, 1x02, E02 or Episode 2, or a leading
// episode number as in "01 Pilot.srt".
func numbered(name string) numbering {
	base := path.Base(strings.ReplaceAll(name, `\`, "/"))
	base = strings.TrimSuffix(base, path.Ext(base))
	if m := nameSE.FindStringSubmatchIndex(base); m != nil {
		season, _ := strconv.Atoi(base[m[2]:m[3]])
		first, _ := strconv.Atoi(base[m[4]:m[5]])
		return numbering{season, append([]int{first}, moreEpisodes(base[m[1]:], first)...)}
	}
	if m := nameX.FindStringSubmatch(base); m != nil {
		season, _ := strconv.Atoi(m[1])
		e, _ := strconv.Atoi(m[2])
		return numbering{season, []int{e}}
	}
	for _, re := range []*regexp.Regexp{nameEpisode, nameLeading} {
		if m := re.FindStringSubmatch(base); m != nil {
			e, _ := strconv.Atoi(m[1])
			return numbering{0, []int{e}}
		}
	}
	return numbering{}
}

// moreEpisodes reads the episodes after the first: E02 as one more, a
// dash and E04 as a range, and a dash with a bare number only as the
// next episode, so "S01E01 - 720p" stays one episode.
func moreEpisodes(rest string, prev int) []int {
	var es []int
	for {
		m := nameMore.FindStringSubmatchIndex(rest)
		if m == nil {
			return es
		}
		n, _ := strconv.Atoi(rest[m[6]:m[7]])
		withE := m[4] < 0
		dash := m[2] >= 0 || m[4] >= 0
		switch {
		case n <= prev || !withE && n != prev+1:
			return es
		case dash:
			for e := prev + 1; e <= n; e++ {
				es = append(es, e)
			}
		default:
			es = append(es, n)
		}
		prev = n
		rest = rest[m[7]:]
	}
}

// Pick chooses the member for a season and its episodes: every member
// whose name gives exactly those episodes, in that season or none. An
// empty episodes, or an archive whose names give no numbering at all,
// makes every member a candidate. Text beats image, then a member whose
// name tags lang, then the larger. False when none fits.
func Pick(es []Entry, season int, episodes []int, lang language.Code) (Entry, bool) {
	candidates := es
	if len(episodes) > 0 && slices.ContainsFunc(es, func(e Entry) bool { return numbered(e.Name).episodes != nil }) {
		candidates = nil
		for _, e := range es {
			n := numbered(e.Name)
			if (n.season == 0 || n.season == season) && slices.Equal(n.episodes, episodes) {
				candidates = append(candidates, e)
			}
		}
	}
	if len(candidates) == 0 {
		return Entry{}, false
	}
	return slices.MaxFunc(candidates, func(a, b Entry) int {
		return compareRanks(rank(a, lang), rank(b, lang))
	}), true
}

type entryRank struct {
	text, tagged bool
	size         int
}

func rank(e Entry, lang language.Code) entryRank {
	return entryRank{text: image(e.Bytes) == 0, tagged: tags(e.Name, lang), size: len(e.Bytes)}
}

func compareRanks(a, b entryRank) int {
	switch {
	case a.text != b.text:
		return boolRank(a.text)
	case a.tagged != b.tagged:
		return boolRank(a.tagged)
	}
	return a.size - b.size
}

func boolRank(b bool) int {
	if b {
		return 1
	}
	return -1
}

// tags reports whether one of the last three dot tokens of name, before
// its extension, is lang's.
func tags(name string, lang language.Code) bool {
	base := path.Base(strings.ReplaceAll(name, `\`, "/"))
	tokens := strings.Split(strings.TrimSuffix(base, path.Ext(base)), ".")
	for _, t := range tokens[1:][max(0, len(tokens)-4):] {
		if c, ok := language.ParseTag(t); ok && c == lang {
			return true
		}
	}
	return false
}

// Numbered is what a member's name says of its season, zero when it gives
// none, and of its episodes, as Pick reads them.
func Numbered(name string) (season int, episodes []int) {
	n := numbered(name)
	return n.season, n.episodes
}
