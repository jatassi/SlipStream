package cardigann

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dlclark/regexp2"
)

// Clock is the current time and the zone a date without an offset is read
// in: Prowlarr's DateTime.Now and its host's local zone.
type Clock struct {
	Now      time.Time
	Location *time.Location
}

func (c Clock) local() time.Time { return c.Now.In(c.Location) }

// formatRFC1123Z writes t as Prowlarr's Rfc1123ZPattern,
// "ddd, dd MMM yyyy HH':'mm':'ss z", the form dateparse, timeago and
// fuzzytime leave a value in. The offset is whole hours, as .NET's z writes
// it.
func formatRFC1123Z(t time.Time) string {
	_, off := t.Zone()
	sign := "+"
	if off < 0 {
		sign, off = "-", -off
	}
	return t.Format("Mon, 02 Jan 2006 15:04:05") + " " + sign + strconv.Itoa(off/3600)
}

const rfc1123ZFormat = "ddd, dd MMM yyyy HH':'mm':'ss z"

// parseDateLayout is ParseDateTimeGoLang (DateTimeUtil.cs:256-327): the
// layout is first read as a .NET custom format when it holds a y, h or d,
// then rewritten from Go's reference-date tokens to .NET's and read again.
func parseDateLayout(value, layout string, clock Clock) (time.Time, error) {
	value = strings.TrimSpace(value)
	lower := strings.ToLower(layout)
	if strings.ContainsAny(lower, "yhd") {
		if t, ok := parseExact(value, layout, clock); ok {
			return t, nil
		}
	}
	format := goLayoutToDotNet(layout)
	if t, ok := parseExact(value, format, clock); ok {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("%q doesn't match the date layout %q", value, layout)
}

// goLayoutRewrites are Prowlarr's replacements, applied in this order.
var goLayoutRewrites = []string{
	"2006", "yyyy", "06", "yy",
	"January", "MMMM", "Jan", "MMM", "01", "MM",
	"Monday", "dddd", "Mon", "ddd", "02", "dd", "2", "d",
	"05", "ss", "15", "HH", "03", "hh", "3", "h", "04", "mm", "4", "m", "5", "s",
	"1", "M",
	".0000", "ffff", ".000", "fff", ".00", "ff", ".0", "f",
	".9999", "FFFF", ".999", "FFF", ".99", "FF", ".9", "F",
	"PM", "tt", "pm", "tt",
	"Z07:00", "'Z'zzz", "Z07", "'Z'zz", "-07:00", "zzz", "-07", "zz",
}

func goLayoutToDotNet(layout string) string {
	for i := 0; i < len(goLayoutRewrites); i += 2 {
		layout = strings.ReplaceAll(layout, goLayoutRewrites[i], goLayoutRewrites[i+1])
	}
	return layout
}

var timeAgoPart = regexp2.MustCompile(`\s*?([\d\.]+)\s*?([^\d\s\.]+)\s*?`, regexp2.None)

// timeAgoUnits are FromTimeAgo's units, checked in order. A unit matches
// when it contains one of the words or equals the letter.
var timeAgoUnits = []struct {
	words  []string
	letter string
	length time.Duration
}{
	{[]string{"sec"}, "s", time.Second},
	{[]string{"min"}, "m", time.Minute},
	{[]string{"hour", "hr"}, "h", time.Hour},
	{[]string{"day"}, "d", 24 * time.Hour},
	{[]string{"week", "wk"}, "w", 7 * 24 * time.Hour},
	{[]string{"month"}, "mo", 30 * 24 * time.Hour},
	{[]string{"year"}, "y", 365 * 24 * time.Hour},
}

// fromTimeAgo is DateTimeUtil.FromTimeAgo: "2 hours 1 day" ago from now.
func fromTimeAgo(s string, clock Clock) (time.Time, error) {
	s = strings.ToLower(s)
	if strings.Contains(s, "now") {
		return clock.local(), nil
	}
	for _, w := range []string{",", "ago", "and"} {
		s = strings.ReplaceAll(s, w, "")
	}
	var ago time.Duration
	m, err := timeAgoPart.FindStringMatch(s)
	for ; m != nil && err == nil; m, err = timeAgoPart.FindNextMatch(m) {
		d, err := timeAgoLength(m.GroupByNumber(1).String(), m.GroupByNumber(2).String())
		if err != nil {
			return time.Time{}, err
		}
		ago += d
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("read %q as a time ago: %w", s, err)
	}
	return clock.local().Add(-ago), nil
}

func timeAgoLength(number, unit string) (time.Duration, error) {
	val, err := coerceFloat(number)
	if err != nil {
		return 0, err
	}
	for _, u := range timeAgoUnits {
		if unit == u.letter || containsAny(unit, u.words) {
			ticks := int64(float64(float32(val)) * float64(u.length/100))
			return time.Duration(ticks) * 100, nil
		}
	}
	return 0, fmt.Errorf("%q isn't a unit of time", unit)
}

func containsAny(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

var (
	todayRe     = regexp2.MustCompile(`(?i)\btoday(?:[\s,]+(?:at){0,1}\s*|[\s,]*|$)`, regexp2.None)
	tomorrowRe  = regexp2.MustCompile(`(?i)\btomorrow(?:[\s,]+(?:at){0,1}\s*|[\s,]*|$)`, regexp2.None)
	yesterdayRe = regexp2.MustCompile(`(?i)\byesterday(?:[\s,]+(?:at){0,1}\s*|[\s,]*|$)`, regexp2.None)
	weekdayAtRe = regexp2.MustCompile(`(?i)\b(monday|tuesday|wednesday|thursday|friday|saturday|sunday)\s+at\s+`, regexp2.None)
	agoRe       = regexp2.MustCompile(`(?i)\bago`, regexp2.None)
	noYearRe    = regexp2.MustCompile(`^(\d{1,2}-\d{1,2})(\s|$)`, regexp2.None)
	noYearRe2   = regexp2.MustCompile(`^(\d{1,2}\s+\w{3})\s+(\d{1,2}\:\d{1,2}.*)$`, regexp2.None)
)

// parseDate is DateTimeUtil.FromUnknown, which reads a date field: the
// RFC1123Z form the date filters write, a Unix time, "now", a time ago,
// today, yesterday or tomorrow at a time, a weekday at a time, and
// otherwise DateTimeRoutines' fuzzy US-date parser.
func parseDate(s string, clock Clock) (time.Time, error) {
	s = strings.TrimSpace(s)
	if t, ok := parseExact(s, rfc1123ZFormat, clock); ok {
		return t, nil
	}
	if s != "" && strings.IndexFunc(s, func(r rune) bool { return !isASCIIDigit(r) }) < 0 {
		if secs, err := strconv.ParseInt(s, 10, 64); err == nil {
			return time.Unix(secs, 0).In(clock.Location), nil
		}
	}
	if strings.Contains(strings.ToLower(s), "now") {
		return clock.Now.UTC(), nil
	}
	if ok, _ := agoRe.MatchString(s); ok {
		return fromTimeAgo(s, clock)
	}
	for _, rel := range relativeDays {
		if t, ok, err := rel.parse(s, clock); ok || err != nil {
			return t, err
		}
	}
	if t, ok, err := parseWeekdayAt(s, clock); ok || err != nil {
		return t, err
	}
	s = addMissingYear(s, clock)
	t, ok, err := fuzzyDate(s, clock)
	if err != nil {
		return time.Time{}, err
	}
	if !ok {
		return time.Time{}, fmt.Errorf("%q isn't a date", s)
	}
	return t, nil
}

var relativeDays = []relativeDay{{todayRe, 0}, {yesterdayRe, -1}, {tomorrowRe, 1}}

type relativeDay struct {
	re   *regexp2.Regexp
	days int
}

// parse reads "today 14:22" and the like: the UTC date at the given time,
// read as local time, as Prowlarr builds it from DateTime.UtcNow.Date.
func (r relativeDay) parse(s string, clock Clock) (time.Time, bool, error) {
	m, err := r.re.FindStringMatch(s)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("read the date %q: %w", s, err)
	}
	if m == nil {
		return time.Time{}, false, nil
	}
	t, err := utcDateAt(strings.ReplaceAll(s, m.String(), ""), clock)
	if err != nil {
		return time.Time{}, true, err
	}
	return t.AddDate(0, 0, r.days), true, nil
}

func parseWeekdayAt(s string, clock Clock) (time.Time, bool, error) {
	m, err := weekdayAtRe.FindStringMatch(s)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("read the date %q: %w", s, err)
	}
	if m == nil {
		return time.Time{}, false, nil
	}
	t, err := utcDateAt(strings.ReplaceAll(s, m.String(), ""), clock)
	if err != nil {
		return time.Time{}, true, err
	}
	want := strings.ToLower(m.GroupByNumber(1).String())
	for !strings.HasPrefix(want, strings.ToLower(t.Weekday().String())) {
		t = t.AddDate(0, 0, -1)
	}
	return t, true, nil
}

func utcDateAt(clockTime string, clock Clock) (time.Time, error) {
	u := clock.Now.UTC()
	day := time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, clock.Location)
	if strings.TrimSpace(clockTime) == "" {
		return day, nil
	}
	h, m, sec, err := parseTimeOfDay(clockTime)
	if err != nil {
		return time.Time{}, err
	}
	return day.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(sec)*time.Second), nil
}

var timeOfDayRe = regexp2.MustCompile(`^\s*(\d{1,2})(?::(\d{2}))?(?::(\d{2}))?(?:\.\d+)?\s*(?:([AaPp])\.?[Mm]\.?)?\s*$`, regexp2.None)

// parseTimeOfDay stands in for .NET's DateTime.Parse(time).TimeOfDay over
// the times sites write after "today": H:mm, H:mm:ss and either with AM or
// PM, or an hour with AM or PM.
func parseTimeOfDay(s string) (h, m, sec int, err error) {
	match, err := timeOfDayRe.FindStringMatch(s)
	if err != nil || match == nil || (match.GroupByNumber(2).String() == "" && match.GroupByNumber(4).String() == "") {
		return 0, 0, 0, fmt.Errorf("%q isn't a time of day", s)
	}
	h, _ = strconv.Atoi(match.GroupByNumber(1).String())
	m, _ = strconv.Atoi(match.GroupByNumber(2).String())
	sec, _ = strconv.Atoi(match.GroupByNumber(3).String())
	switch strings.ToUpper(match.GroupByNumber(4).String()) {
	case "P":
		if h < 12 {
			h += 12
		}
	case "A":
		if h == 12 {
			h = 0
		}
	}
	if h > 23 || m > 59 || sec > 59 {
		return 0, 0, 0, fmt.Errorf("%q isn't a time of day", s)
	}
	return h, m, sec, nil
}

func addMissingYear(s string, clock Clock) string {
	year := strconv.Itoa(clock.local().Year())
	if m, _ := noYearRe.FindStringMatch(s); m != nil {
		date := m.GroupByNumber(1).String()
		s = strings.ReplaceAll(s, date, year+"-"+date)
	}
	if m, _ := noYearRe2.FindStringMatch(s); m != nil {
		s = m.GroupByNumber(1).String() + " " + year + " " + m.GroupByNumber(2).String()
	}
	return s
}
