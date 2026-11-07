package cardigann

import (
	"fmt"
	"strconv"
	"time"

	"github.com/dlclark/regexp2"
)

// This file ports the US-date half of CliverSoft's DateTimeRoutines
// (Core/Parser/DateTimeRoutines.cs), which FromFuzzyTime calls. Indexes
// count runes, as regexp2 does.

func mustRe(pattern string) *regexp2.Regexp {
	return regexp2.MustCompile(pattern, regexp2.None)
}

func mustReI(pattern string) *regexp2.Regexp {
	return regexp2.MustCompile(pattern, regexp2.IgnoreCase)
}

const usTimeZone = `(?:\s*(?'time_zone'UTC|GMT|CST|EST))?`

var (
	timeAfterDateOffset = mustRe(`(?<=^\s*,?\s+|^\s*at\s*|^\s*[T\-]\s*)(?'hour'\d{2})\s*:\s*(?'minute'\d{2})\s*:\s*(?'second'\d{2})\s+(?'offset_sign'[\+\-])(?'offset_hh'\d{2}):?(?'offset_mm'\d{2})(?=$|[^\d\w])`)
	timeAfterDate       = mustRe(`(?<=^\s*,?\s+|^\s*at\s*|^\s*[T\-]\s*)(?'hour'\d{1,2})\s*:\s*(?'minute'\d{2})\s*(?::\s*(?'second'\d{2}))?(?:\s*(?'ampm'AM|am|PM|pm))?` + usTimeZone + `(?=$|[^\d\w])`)
	timeAroundDate      = mustRe(`(?<=^|[^\d])(?'hour'\d{1,2})\s*:\s*(?'minute'\d{2})\s*(?::\s*(?'second'\d{2}))?(?:\s*(?'ampm'AM|am|PM|pm))?` + usTimeZone + `(?=$|[\s,]+)`)
	timeAnywhereOffset  = mustRe(`(?<=^|\s+|\s*T\s*)(?'hour'\d{2})\s*:\s*(?'minute'\d{2})\s*:\s*(?'second'\d{2})\s+(?'offset_sign'[\+\-])(?'offset_hh'\d{2}):?(?'offset_mm'\d{2})?(?=$|[^\d\w])`)
	timeAnywhere        = mustRe(`(?<=^|\s+|\s*T\s*)(?'hour'\d{1,2})\s*:\s*(?'minute'\d{2})\s*(?::\s*(?'second'\d{2}))?(?:\s*(?'ampm'AM|am|PM|pm))?` + usTimeZone + `(?=$|[^\d\w])`)

	dateNumeric = mustReI(`(?<=^|[^\d])(?'day'\d{1,2})\s*(?'separator'[\\/\.])+\s*(?'month'\d{1,2})\s*\'separator'+\s*(?'year'\d{2}|\d{4})(?=$|[^\d])`)
	dateISO     = mustReI(`(?<=^|[^\d])(?'year'\d{2}|\d{4})\s*(?'separator'[\-])\s*(?'month'\d{1,2})\s*\'separator'+\s*(?'day'\d{1,2})(?=$|[^\d])`)
	dateNamed   = []*regexp2.Regexp{
		mustReI(`(?:^|[^\d\w])(?'month'Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[uarychilestmbro]*\s+(?'day'\d{1,2})(?:-?st|-?th|-?rd|-?nd)?\s*,?\s*(?'year'\d{4})(?=$|[^\d\w])`),
		mustReI(`(?:^|[^\d\w:])(?'day'\d{1,2})(?:-?st\s+|-?th\s+|-?rd\s+|-?nd\s+|-|\s+)(?'month'Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[uarychilestmbro]*(?:\s*,?\s*|-)'?(?'year'\d{2}|\d{4})(?=$|[^\d\w])`),
		mustReI(`(?:^|[^\d\w])(?'year'\d{4})\s+(?'month'Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[uarychilestmbro]*\s+(?'day'\d{1,2})(?:-?st|-?th|-?rd|-?nd)?(?=$|[^\d\w])`),
		mustReI(`(?:^|[^\d\w])(?'month'Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[uarychilestmbro]*\s+(?'day'\d{1,2})\s+\d{2}\:\d{2}\:\d{2}\s+(?:MDT|UTC)\s+(?'year'\d{4})(?=$|[^\d\w])`),
		mustReI(`(?:^|[^\d\w])(?'month'Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[uarychilestmbro]*\s+(?'day'\d{1,2})(?:-?st|-?th|-?rd|-?nd)?(?:\s*,?\s*(?'year'\d{4}))?(?=$|[^\d\w])`),
	}
)

// monthsByCase is the routine's case-sensitive switch: "Jul" has no
// uppercase spelling there, so "JUL" names no month.
var monthsByCase = map[string]int{
	"Jan": 1, "JAN": 1, "Feb": 2, "FEB": 2, "Mar": 3, "MAR": 3, "Apr": 4, "APR": 4,
	"May": 5, "MAY": 5, "Jun": 6, "JUN": 6, "Jul": 7, "Aug": 8, "AUG": 8,
	"Sep": 9, "SEP": 9, "Oct": 10, "OCT": 10, "Nov": 11, "NOV": 11, "Dec": 12, "DEC": 12,
}

type foundDate struct {
	date          time.Time
	index, length int
}

// fuzzyDate is TryParseDateOrTime with USDate. The UTC offset it may find
// is dropped, as FromFuzzyTime returns the plain DateTime.
func fuzzyDate(s string, clock Clock) (time.Time, bool, error) {
	d, ok, err := findDate(s, clock)
	if err != nil {
		return time.Time{}, false, err
	}
	var at *foundDate
	if ok {
		at = &d
	}
	h, m, sec, timeOK, err := findTime(s, at)
	if err != nil {
		return time.Time{}, false, err
	}
	switch {
	case ok:
		return time.Date(d.date.Year(), d.date.Month(), d.date.Day(), h, m, sec, 0, clock.Location), true, nil
	case timeOK:
		now := clock.local()
		return time.Date(now.Year(), now.Month(), now.Day(), h, m, sec, 0, clock.Location), true, nil
	}
	return time.Time{}, false, nil
}

func findDate(s string, clock Clock) (foundDate, bool, error) {
	m, err := dateNumeric.FindStringMatch(s)
	if err != nil {
		return foundDate{}, false, fmt.Errorf("read %q as a date: %w", s, err)
	}
	if m != nil {
		// In US order the first number is the month.
		return dateFromGroups(m, group(m, "year"), group(m, "day"), group(m, "month"))
	}
	if m, err = dateISO.FindStringMatch(s); err != nil {
		return foundDate{}, false, fmt.Errorf("read %q as a date: %w", s, err)
	}
	if m != nil {
		return dateFromGroups(m, group(m, "year"), group(m, "month"), group(m, "day"))
	}
	for _, re := range dateNamed {
		if m, err = re.FindStringMatch(s); err != nil {
			return foundDate{}, false, fmt.Errorf("read %q as a date: %w", s, err)
		}
		if m != nil {
			break
		}
	}
	if m == nil {
		return foundDate{}, false, nil
	}
	month, ok := monthsByCase[group(m, "month")]
	if !ok {
		month = -1
	}
	year := group(m, "year")
	if year == "" {
		year = strconv.Itoa(clock.local().Year())
	}
	return dateFromGroups(m, year, strconv.Itoa(month), group(m, "day"))
}

func group(m *regexp2.Match, name string) string {
	g := m.GroupByName(name)
	if g == nil {
		return ""
	}
	return g.String()
}

func dateFromGroups(m *regexp2.Match, year, month, day string) (foundDate, bool, error) {
	y, err := strconv.Atoi(year)
	if err != nil {
		return foundDate{}, false, fmt.Errorf("%q isn't a year", year)
	}
	mo, err := strconv.Atoi(month)
	if err != nil {
		return foundDate{}, false, fmt.Errorf("%q isn't a month", month)
	}
	d, err := strconv.Atoi(day)
	if err != nil {
		return foundDate{}, false, fmt.Errorf("%q isn't a day", day)
	}
	date, ok := convertToDate(y, mo, d)
	if !ok {
		return foundDate{}, false, nil
	}
	return foundDate{date: date, index: m.Index, length: m.Length}, true, nil
}

// convertToDate reads a two-digit year as 19xx above 30 and 20xx
// otherwise, and refuses a three-digit year.
func convertToDate(year, month, day int) (time.Time, bool) {
	switch {
	case year >= 100 && year < 1000:
		return time.Time{}, false
	case year < 100 && year > 30:
		year += 1900
	case year < 100:
		year += 2000
	}
	if !validDate(year, month, day) {
		return time.Time{}, false
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), true
}

// findTime is TryParseTime: with a date found it looks after the date, then
// before it, then inside it; without one it looks anywhere.
func findTime(s string, d *foundDate) (h, m, sec int, ok bool, err error) {
	var match *regexp2.Match
	if d != nil {
		runes := []rune(s)
		after := string(runes[d.index+d.length:])
		candidates := []struct {
			re   *regexp2.Regexp
			text string
		}{
			{timeAfterDateOffset, after},
			{timeAfterDate, after},
			{timeAroundDate, string(runes[:d.index])},
			{timeAroundDate, string(runes[d.index : d.index+d.length])},
		}
		for _, c := range candidates {
			if match, err = c.re.FindStringMatch(c.text); err != nil || match != nil {
				break
			}
		}
	} else if match, err = timeAnywhereOffset.FindStringMatch(s); err == nil && match == nil {
		match, err = timeAnywhere.FindStringMatch(s)
	}
	if err != nil {
		return 0, 0, 0, false, fmt.Errorf("read %q as a time: %w", s, err)
	}
	if match == nil {
		return 0, 0, 0, false, nil
	}
	return timeFromMatch(match)
}

func timeFromMatch(match *regexp2.Match) (h, m, sec int, ok bool, err error) {
	if h, err = strconv.Atoi(group(match, "hour")); err != nil {
		return 0, 0, 0, false, fmt.Errorf("%q isn't an hour", group(match, "hour"))
	}
	if m, err = strconv.Atoi(group(match, "minute")); err != nil {
		return 0, 0, 0, false, fmt.Errorf("%q isn't a minute", group(match, "minute"))
	}
	if s := group(match, "second"); s != "" {
		if sec, err = strconv.Atoi(s); err != nil {
			return 0, 0, 0, false, fmt.Errorf("%q isn't a second", s)
		}
	}
	if h > 23 || m > 59 || sec > 59 {
		return 0, 0, 0, false, nil
	}
	switch ampm := group(match, "ampm"); {
	case (ampm == "PM" || ampm == "pm") && h < 12:
		h += 12
	case (ampm == "AM" || ampm == "am") && h == 12:
		h = 0
	}
	return h, m, sec, true, nil
}
