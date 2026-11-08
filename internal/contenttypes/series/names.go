package series

import (
	"errors"
	"fmt"
	"math"
	"path"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/upstream"
	"github.com/jatassi/slipstream/internal/regex"
	"github.com/jatassi/slipstream/internal/release"
)

var (
	reportTitles        = compileAll(upstream.SonarrParserReportTitleRegex)
	seasonFolderNames   = compileAll(upstream.SonarrParserSeasonFolderRegexes)
	sixDigitAirDate     = compile(upstream.SonarrParserSixDigitAirDateRegex)
	requestInfo         = compile(upstream.SonarrParserRequestInfoRegex)
	yearInTitle         = compile(upstream.SonarrParserYearInTitleRegex)
	titleComponents     = compile(upstream.SonarrParserTitleComponentsRegex)
	seasonFolder        = compile(upstream.SonarrParserSeasonFolderRegex)
	simpleEpisodeNumber = compile(upstream.SonarrParserSimpleEpisodeNumberRegex)
	normalizeTitle      = compile(upstream.SonarrParserNormalizeRegex.Pattern)
	percentSign         = compile(upstream.SonarrParserPercentRegex)
)

const compiledOptions = upstream.IgnoreCase | upstream.Compiled

func compile(p upstream.Pattern) *regex.Regex {
	if extra := p.Options &^ compiledOptions; extra != 0 {
		panic(fmt.Sprintf("%s has .NET options %d that the parser doesn't run", p.Key, extra))
	}
	return regex.MustCompile(p.Text, regex.Options{IgnoreCase: p.Options&upstream.IgnoreCase != 0, Timeout: release.PatternLimit})
}

func compileAll(ps []upstream.Pattern) []*regex.Regex {
	out := make([]*regex.Regex, len(ps))
	for i, p := range ps {
		out[i] = compile(p)
	}
	return out
}

// ReadName ports Sonarr's ParseTitle from its pattern loop, and for a
// file the season folder and number-only steps of its ParsePath.
func (*Series) ReadName(p release.Prepared) release.Name {
	r, ok := parseTitle(p)
	if p.Folder != "" {
		r, ok = parsePath(p, r, ok)
	}
	if !ok {
		return release.Name{}
	}
	marks, span := r.marks()
	return release.Name{Titles: r.titles, Year: r.year, Marks: marks, Span: span, Tail: r.tail}
}

// parsed is the part of Sonarr's ParsedEpisodeInfo SlipStream reads.
type parsed struct {
	titles []string
	year   release.Fact[int]
	// seasons is never empty: Sonarr's SeasonNumbers starts as [0].
	seasons, episodes, absolutes []int
	specialAbsolute              float64
	fullSeason, miniSeries       bool
	part                         int
	tail                         string
}

// errUnread is Sonarr's ParseTitle answering null without trying the
// next pattern: an exception, a timeout or a date it can't read.
var errUnread = errors.New("the name reads as nothing")

// parseTitle is Sonarr's ParseTitle on a prepared name, false where it
// answers null.
func parseTitle(p release.Prepared) (parsed, bool) {
	if p.Obfuscated || seasonFolderName(p.Title) {
		return parsed{}, false
	}
	simple, err := withAirDateFixed(p.Simple)
	if err != nil {
		return parsed{}, false
	}
	for _, re := range reportTitles {
		ms, err := re.FindAll(simple)
		if err != nil {
			return parsed{}, false
		}
		if len(ms) == 0 {
			continue
		}
		r, err := parseMatches(ms, []rune(p.Title.String()))
		if errors.Is(err, errNext) {
			continue
		}
		if err != nil {
			return parsed{}, false
		}
		tokens := r.tail
		if tokens == "" {
			tokens = p.Title.String()
		}
		if r.fullSeason && strings.Contains(strings.ToLower(tokens), "special") {
			r.fullSeason = false
		}
		return r, true
	}
	return parsed{}, false
}

// seasonFolderName is Sonarr's ValidateBeforeParsing rejecting t as a
// season folder's name, or running past its limit, which Sonarr's
// ParseTitle also answers null for.
func seasonFolderName(t regex.Text) bool {
	return slices.ContainsFunc(seasonFolderNames, func(re *regex.Regex) bool {
		ok, err := re.Match(t)
		return err != nil || ok
	})
}

// withAirDateFixed is t with a six-digit air date written out, 140301 as
// 2014.03.01.
func withAirDateFixed(t regex.Text) (regex.Text, error) {
	m, ok, err := sixDigitAirDate.Find(t)
	if err != nil || !ok {
		return t, err
	}
	year, _ := m.Group("airyear")
	month, _ := m.Group("airmonth")
	day, _ := m.Group("airday")
	if month == "00" && day == "00" {
		return t, nil
	}
	date, _ := m.Group("airdate")
	return regex.TextOf(strings.ReplaceAll(t.String(), date, "20"+year+"."+month+"."+day)), nil
}

// errNext is Sonarr's ParseMatchCollection answering null, so ParseTitle
// tries the next pattern.
var errNext = errors.New("try the next pattern")

// parseMatches ports Sonarr's ParseMatchCollection. whole is the prepared
// title the release tokens are cut from, at the index the matches on the
// simple title end at, as Sonarr cuts them.
func parseMatches(ms []regex.Match, whole []rune) (parsed, error) {
	first := ms[0]
	if year, _ := first.Group("airyear"); airYear(year) >= 1900 {
		// SlipStream doesn't read air dates yet.
		return parsed{}, errUnread
	}
	r := parsed{seasons: []int{0}}
	at := end(first.Captures("title"))
	for _, m := range ms {
		end, err := r.readNumbers(m, first)
		if err != nil {
			return parsed{}, err
		}
		at = max(at, end)
	}
	at = max(at, r.readSeasons(first))
	if at < len(whole) {
		r.tail = string(whole[at:])
	}
	if err := r.readTitle(first); err != nil {
		return parsed{}, err
	}
	return r, nil
}

// readNumbers reads one match's episodes, absolute numbers, or season
// part, and answers where the last number ends.
func (r *parsed) readNumbers(m, first regex.Match) (int, error) {
	at := 0
	eps, abs := m.Captures("episode"), m.Captures("absoluteepisode")
	if len(eps) > 0 {
		from, to, err := rangeOf(eps, parseNumber)
		if err != nil {
			return 0, err
		}
		r.episodes, at = between(from, to), end(eps)
	}
	if len(abs) > 0 {
		from, to, err := rangeOf(abs, parseDecimal)
		switch {
		case err != nil:
			return 0, err
		case from == math.Trunc(from) && to == math.Trunc(to):
			r.absolutes = between(int(from), int(to))
		case len(abs) != 1:
			return 0, errNext
		default:
			r.specialAbsolute = from
		}
		at = max(at, end(abs))
	}
	if len(eps) == 0 && len(abs) == 0 {
		part, _ := first.Group("seasonpart")
		_, special := first.Group("special")
		switch {
		case strings.TrimSpace(part) != "":
			n, err := strconv.Atoi(part)
			if err != nil {
				return 0, errUnread
			}
			r.part = n
		case !special:
			r.fullSeason = true
		}
	}
	if count, ok := first.Group("episodecount"); ok && len(eps) == 2 && eps[1].Value == count {
		r.episodes, r.fullSeason = nil, true
	}
	return at, nil
}

// rangeOf is the first and last of cs, errNext when they run backwards.
func rangeOf[N int | float64](cs []regex.Capture, parse func(string) (N, error)) (from, to N, err error) {
	if from, err = parse(cs[0].Value); err != nil {
		return from, to, err
	}
	if to, err = parse(cs[len(cs)-1].Value); err != nil {
		return from, to, err
	}
	if from > to {
		return from, to, errNext
	}
	return from, to, nil
}

// readSeasons reads the first match's seasons, two as the run between
// them, and a name with episodes but no season or absolute number as
// season 1 of a mini-series. It answers where the last season ends.
func (r *parsed) readSeasons(first regex.Match) int {
	at := 0
	var seasons []int
	for _, c := range first.Captures("season") {
		if n, err := strconv.Atoi(c.Value); err == nil {
			seasons = append(seasons, n)
			at = max(at, c.End)
		}
	}
	slices.Sort(seasons)
	seasons = slices.Compact(seasons)
	switch {
	case len(seasons) == 1 || len(seasons) > 2:
		r.seasons = seasons
	case len(seasons) == 2:
		r.seasons = between(seasons[0], seasons[1])
	case len(r.absolutes) == 0 && len(r.episodes) > 0:
		r.seasons, r.miniSeries = []int{1}, true
	}
	return at
}

// readTitle ports Sonarr's series title and GetSeriesTitleInfo: the
// title, its year, and every title of an "A AKA B" or "A / B" name,
// Sonarr's series title first when it is one of them.
func (r *parsed) readTitle(first regex.Match) error {
	given, _ := first.Group("title")
	title, err := requestInfo.Replace(regex.TextOf(spaced(given)), "")
	if err != nil {
		return errUnread
	}
	title = strings.Trim(title, " ")
	withoutYear := title
	m, ok, err := yearInTitle.Find(regex.TextOf(title))
	if err != nil {
		return errUnread
	}
	if ok {
		withoutYear, _ = m.Group("title")
		year, _ := m.Group("year")
		n, err := strconv.Atoi(year)
		if err != nil {
			return errUnread
		}
		r.year = release.Known(n)
	}
	c, ok, err := titleComponents.Find(regex.TextOf(withoutYear))
	if err != nil {
		return errUnread
	}
	switch captures := first.Captures("title"); {
	case ok:
		for _, t := range c.Captures("title") {
			r.titles = append(r.titles, t.Value)
		}
	case len(captures) > 1:
		for _, t := range captures {
			r.titles = append(r.titles, spaced(t.Value))
		}
	case title != "":
		r.titles = []string{title}
	}
	if i := slices.Index(r.titles, title); i > 0 {
		r.titles = slices.Insert(slices.Delete(r.titles, i, i+1), 0, title)
	}
	return nil
}

// spaced is a title with its dots and underscores read as spaces.
func spaced(title string) string { return strings.NewReplacer(".", " ", "_", " ").Replace(title) }

// parsePath ports Sonarr's ParsePath between reading the file name and
// trying the folder's: a season folder's number with the episode the
// file name starts with, else for a file named by a number alone, that
// episode of the folder's name.
func parsePath(p release.Prepared, r parsed, ok bool) (parsed, bool) {
	if title, found := seasonTitle(p); found && (!ok || r.miniSeries || len(r.absolutes) > 0) {
		return parseTitle(release.Prepare(title))
	}
	if ok {
		return r, true
	}
	n, err := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(p.Original, path.Ext(p.Original))))
	if err != nil {
		return parsed{}, false
	}
	f, ok := parseTitle(release.Prepare(p.Folder))
	switch {
	case ok && slices.Contains(f.absolutes, n):
		f.absolutes = []int{n}
	case ok && slices.Contains(f.episodes, n):
		f.episodes = []int{n}
	default:
		return parsed{}, false
	}
	return f, true
}

// seasonTitle is the name Sonarr reads a file in a season folder as,
// when the file name starts with an episode number: S01E02 or
// S01E02-E03, then what the file name gives after the number.
func seasonTitle(p release.Prepared) (string, bool) {
	ep, ok, err := simpleEpisodeNumber.Find(regex.TextOf(p.Original))
	if err != nil || !ok {
		return "", false
	}
	s, ok, err := seasonFolder.Find(regex.TextOf(p.Folder))
	if err != nil || !ok {
		return "", false
	}
	season, ok := s.Group("season")
	eps := ep.Captures("episode")
	from, err1 := parseNumber(eps[0].Value)
	to, err2 := parseNumber(eps[len(eps)-1].Value)
	if !ok || err1 != nil || err2 != nil {
		return "", false
	}
	title := fmt.Sprintf("S%sE%02d", season, from)
	if from != to {
		title += fmt.Sprintf("-E%02d", to)
	}
	if rest, ok := ep.Group("remaining"); ok {
		title += " " + rest
	}
	return title, true
}

func end(cs []regex.Capture) int {
	if len(cs) == 0 {
		return 0
	}
	return cs[len(cs)-1].End
}

func between(from, to int) []int {
	out := make([]int, 0, to-from+1)
	for n := from; n <= to; n++ {
		out = append(out, n)
	}
	return out
}

// airYear is Sonarr's int.TryParse of the airyear group: 0 when absent.
func airYear(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// numberWords are Sonarr's Numbers, which "Part One" names an episode by.
var numberWords = []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine"}

// parseNumber ports Sonarr's ParseNumber: digits of any script, or a
// number's English word.
func parseNumber(s string) (int, error) {
	if n, err := strconv.Atoi(numerals(s)); err == nil {
		return n, nil
	}
	if n := slices.Index(numberWords, strings.ToLower(s)); n != -1 {
		return n, nil
	}
	return 0, errUnread
}

func parseDecimal(s string) (float64, error) {
	n, err := strconv.ParseFloat(numerals(s), 64)
	if err != nil {
		return 0, errUnread
	}
	return n, nil
}

// numerals is s in compatibility form with every decimal digit written
// in ASCII, Sonarr's ConvertToNumerals.
func numerals(s string) string {
	return strings.Map(func(r rune) rune {
		if !unicode.IsDigit(r) {
			return r
		}
		zero := r
		for unicode.IsDigit(zero - 1) {
			zero--
		}
		return '0' + (r-zero)%10
	}, norm.NFKC.String(s))
}

// CleanTitle ports Sonarr's CleanSeriesTitle: a title of digits alone as
// given, else lower case without punctuation, spaces, diacritics and
// inner articles and conjunctions, with % after a number as "percent".
func (*Series) CleanTitle(title string) string {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return title
	}
	if _, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return title
	}
	t := regex.TextOf(title)
	for _, step := range []struct {
		re   *regex.Regex
		with string
	}{{percentSign, "percent"}, {normalizeTitle, upstream.SonarrParserNormalizeRegex.With}} {
		if out, err := step.re.Replace(t, step.with); err == nil {
			t = regex.TextOf(out)
		}
	}
	return withoutDiacritics(strings.ToLower(t.String()))
}

// moreDiacritics are Sonarr's AdditionalDiacriticsProvider, the letters
// that decompose to no base letter.
var moreDiacritics = strings.NewReplacer("ð", "d", "Ð", "D", "þ", "th")

func withoutDiacritics(s string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(s) {
		if !unicode.Is(unicode.Mn, r) {
			b.WriteRune(r)
		}
	}
	return moreDiacritics.Replace(norm.NFC.String(b.String()))
}
