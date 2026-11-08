package scanning

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	// tag is an id in the Plex and Jellyfin forms: {tmdb-603},
	// {tmdbid-603}, [tmdbid-603], {imdb-tt0133093}, [imdbid-tt0133093],
	// {tvdb-81189}, [tvdbid=81189].
	tag = regexp.MustCompile(`(?i)[{\[](tmdb|imdb|tvdb)(?:id)?[-=](tt\d+|\d+)[}\]]`)
	// group is any {…} or […] group, such as {edition-Final Cut}: never
	// part of a title.
	group       = regexp.MustCompile(`\{[^}]*\}|\[[^\]]*\]`)
	inBrackets  = regexp.MustCompile(`\((\d{4})\)`)
	yearAlone   = regexp.MustCompile(`^\d{4}$`)
	idProviders = map[string]Provider{"tmdb": TMDB, "imdb": IMDb, "tvdb": TVDB}
)

const (
	firstYear = 1880
	lastYear  = 2099
)

// ReadName reads a title, a year and id tags from a Work folder's or a
// loose file's name, as Plex, Jellyfin and release names write them:
// "Heat (1995) {tmdb-949}", "Heat.1995.1080p.BluRay". The year is the last
// one in brackets, else the last word that is a year and not the first
// word, and the title is what comes before it, so "1917 (2019)" and
// "2001 A Space Odyssey" keep their numbers. Movie and series call it from
// Hint; it is deleted when Hint moves onto S23's parser.
func ReadName(name string) Hint {
	var h Hint
	for _, m := range tag.FindAllStringSubmatch(name, -1) {
		if id, ok := idOf(idProviders[strings.ToLower(m[1])], strings.ToLower(m[2])); ok {
			h.IDs = append(h.IDs, id)
		}
	}
	rest := group.ReplaceAllString(name, " ")
	if !strings.Contains(rest, " ") {
		rest = strings.NewReplacer(".", " ", "_", " ").Replace(rest)
	}
	if loc := inBrackets.FindAllStringSubmatchIndex(rest, -1); loc != nil {
		last := loc[len(loc)-1]
		if y := year(rest[last[2]:last[3]]); y != 0 {
			h.Title, h.Year = clean(rest[:last[0]]), y
			return h
		}
	}
	words := strings.Fields(rest)
	for i := len(words) - 1; i > 0; i-- {
		if y := year(words[i]); y != 0 {
			h.Title, h.Year = clean(strings.Join(words[:i], " ")), y
			return h
		}
	}
	h.Title = clean(rest)
	return h
}

// idOf is an imdb id as "tt" and its digits, and a tmdb or tvdb id as its
// number, so "0603" and "603" name one Work.
func idOf(p Provider, v string) (ID, bool) {
	if p == IMDb {
		return ID{Provider: p, Value: v}, strings.HasPrefix(v, "tt")
	}
	n, err := strconv.ParseUint(v, 10, 63)
	if err != nil || n == 0 {
		return ID{}, false
	}
	return ID{Provider: p, Value: strconv.FormatUint(n, 10)}, true
}

func year(s string) int {
	if !yearAlone.MatchString(s) {
		return 0
	}
	y, _ := strconv.Atoi(s)
	if y < firstYear || y > lastYear {
		return 0
	}
	return y
}

func clean(s string) string {
	return strings.Trim(strings.Join(strings.Fields(s), " "), " -_,")
}
