package series

import (
	"errors"
	"fmt"
	"math"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"
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
	airDate                      time.Time
	dailyPart                    int
	fullSeason, miniSeries       bool
	part                         int
	tail                         string
}

// errUnread is Sonarr's ParseTitle answering null without trying the
// next pattern: a number or a date it can't read.
var errUnread = errors.New("the name reads as nothing")

// parseTitle is Sonarr's ParseTitle on a prepared name, false where it
// answers null.
func parseTitle(p release.Prepared) (parsed, bool) {
	if p.Obfuscated || seasonFolderName(p.Title) {
		return parsed{}, false
	}
	simple := withAirDateFixed(p.Simple)
	for _, re := range reportTitles {
		ms := re.findAll(simple)
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
// season folder's name, which Sonarr's ParseTitle also answers null for.
func seasonFolderName(t regex.Text) bool {
	return slices.ContainsFunc(seasonFolderNames, func(f fixed) bool { return f.match(t) })
}

// withAirDateFixed is t with a six-digit air date written out, 140301 as
// 2014.03.01.
func withAirDateFixed(t regex.Text) regex.Text {
	m, ok := sixDigitAirDate.find(t)
	if !ok {
		return t
	}
	year, _ := m.Group("airyear")
	month, _ := m.Group("airmonth")
	day, _ := m.Group("airday")
	if month == "00" && day == "00" {
		return t
	}
	date, _ := m.Group("airdate")
	return regex.TextOf(strings.ReplaceAll(t.String(), date, "20"+year+"."+month+"."+day))
}

// errNext is Sonarr's ParseMatchCollection answering null, so ParseTitle
// tries the next pattern.
var errNext = errors.New("try the next pattern")

// parseMatches ports Sonarr's ParseMatchCollection. whole is the prepared
// title the release tokens are cut from, at the index the matches on the
// simple title end at, as Sonarr cuts them.
func parseMatches(ms []regex.Match, whole []rune) (parsed, error) {
	first := ms[0]
	r := parsed{seasons: []int{0}}
	at := end(first.Captures("title"))
	if year, _ := first.Group("airyear"); airYear(year) >= 1900 {
		end, err := r.readAirDate(first, airYear(year))
		if err != nil {
			return parsed{}, err
		}
		at = max(at, end)
	} else {
		for _, m := range ms {
			end, err := r.readNumbers(m, first)
			if err != nil {
				return parsed{}, err
			}
			at = max(at, end)
		}
		at = max(at, r.readSeasons(first))
	}
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

// readAirDate ports ParseMatchCollection's daily branch: the air date
// and its part, and where the date ends. A date it can't read, an
// ambiguous one, one more than two weeks ahead, or one before 1970
// outside Plex DVR's "Title (1955) - 1954-01-23" form answers errUnread,
// Sonarr's InvalidDateException.
func (r *parsed) readAirDate(first regex.Match, year int) (int, error) {
	month, day, err := airMonthDay(first)
	if err != nil {
		return 0, err
	}
	if month > 12 {
		month, day = day, month
	}
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if date.Year() != year || int(date.Month()) != month || date.Day() != day {
		return 0, errUnread
	}
	now := time.Now()
	if date.After(time.Date(now.Year(), now.Month(), now.Day()+14, 0, 0, 0, 0, time.UTC)) {
		return 0, errUnread
	}
	if titleYear, _ := first.Group("titleyear"); date.Year() < 1970 && strings.TrimSpace(titleYear) == "" {
		return 0, errUnread
	}
	if part, ok := first.Group("part"); ok {
		if r.dailyPart, err = strconv.Atoi(part); err != nil {
			return 0, errUnread
		}
	}
	r.airDate = date
	return max(end(first.Captures("airyear")), end(first.Captures("airmonth")), end(first.Captures("airday"))), nil
}

// airMonthDay is the month and day as the match gives them: an ambiguous
// pair only when one of them can't be a month, a short month's name, or
// digits.
func airMonthDay(first regex.Match) (month, day int, err error) {
	ambiguousMonth, okMonth := first.Group("ambiguousairmonth")
	ambiguousDay, okDay := first.Group("ambiguousairday")
	shortMonth, okShort := first.Group("shortairmonth")
	airDay, _ := first.Group("airday")
	switch {
	case okMonth && okDay:
		month, day, err = monthDay(ambiguousMonth, ambiguousDay)
		if err == nil && month <= 12 && day <= 12 {
			err = errUnread
		}
	case okShort:
		month = slices.Index(shortMonths, strings.ToLower(shortMonth)) + 1
		if month == 0 {
			return 0, 0, errUnread
		}
		day, err = strconv.Atoi(airDay)
	default:
		airMonth, _ := first.Group("airmonth")
		month, day, err = monthDay(airMonth, airDay)
	}
	if err != nil {
		return 0, 0, errUnread
	}
	return month, day, nil
}

func monthDay(month, day string) (m, d int, err error) {
	m, monthErr := strconv.Atoi(month)
	d, dayErr := strconv.Atoi(day)
	if monthErr != nil || dayErr != nil {
		return 0, 0, errUnread
	}
	return m, d, nil
}

// shortMonths are Sonarr's ShortMonths, in month order.
var shortMonths = []string{"jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"}

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
// title, its year, and every title of an "A AKA B" or "A / B" name after
// it, the order Sonarr's ParsingService looks a series up in.
func (r *parsed) readTitle(first regex.Match) error {
	given, _ := first.Group("title")
	title := requestInfo.replace(regex.TextOf(spaced(given)), "")
	title = strings.Trim(title, " ")
	withoutYear := title
	m, ok := yearInTitle.find(regex.TextOf(title))
	if ok {
		withoutYear, _ = m.Group("title")
		year, _ := m.Group("year")
		n, err := strconv.Atoi(year)
		if err != nil {
			return errUnread
		}
		r.year = release.Known(n)
	}
	c, ok := titleComponents.find(regex.TextOf(withoutYear))
	var all []string
	switch captures := first.Captures("title"); {
	case ok:
		for _, t := range c.Captures("title") {
			all = append(all, t.Value)
		}
	case len(captures) > 1:
		for _, t := range captures {
			all = append(all, spaced(t.Value))
		}
	}
	if title != "" {
		r.titles = []string{title}
	}
	for _, t := range all {
		if t != title {
			r.titles = append(r.titles, t)
		}
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
	ep, ok := simpleEpisodeNumber.find(regex.TextOf(p.Original))
	if !ok {
		return "", false
	}
	s, ok := seasonFolder.find(regex.TextOf(p.Folder))
	if !ok {
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
		re   fixed
		with string
	}{{percentSign, "percent"}, {normalizeTitle, upstream.SonarrParserNormalizeRegex.With}} {
		t = regex.TextOf(step.re.replace(t, step.with))
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
