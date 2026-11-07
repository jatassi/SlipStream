// Ported from Prowlarr's src/NzbDrone.Core/Parser/DateTimeUtil.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package datetimeutil

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DateTimeStyles is .NET's DateTimeStyles as the sites pass it to
// DateTime.Parse and DateTime.ParseExact. None reads a date without an
// offset as the clock's local time.
type DateTimeStyles int

const (
	None DateTimeStyles = 0
	// AdjustToUniversal returns the time in UTC.
	AdjustToUniversal DateTimeStyles = 1 << iota
	// AssumeUniversal reads a date without an offset as UTC.
	AssumeUniversal
)

// ParseExact is .NET's DateTime.ParseExact with the invariant culture
// over a custom format such as "yyyy-MM-dd HH:mm:ss zzz". A date with an
// offset is that instant; one without is read in the zone style names. A
// part the format omits takes today's date. The result is in the clock's
// zone, or UTC under AdjustToUniversal.
func ParseExact(s, format string, style DateTimeStyles, clock Clock) (time.Time, error) {
	t, ok := TryParseExact(s, format, style, clock)
	if !ok {
		return time.Time{}, fmt.Errorf("%q doesn't match the date format %q", s, format)
	}
	return t, nil
}

// TryParseExact is DateTime.TryParseExact: ParseExact, reporting false
// where ParseExact fails.
func TryParseExact(s, format string, style DateTimeStyles, clock Clock) (time.Time, bool) {
	p := exactParser{in: s, format: format, r: newDateResult()}
	if !p.run() || p.pos != len(p.in) {
		return time.Time{}, false
	}
	return p.r.resolve(style, clock)
}

// Parse is .NET's DateTime.Parse with the invariant culture over the forms
// the sites send: ISO 8601 with a 'T' or a space, optional seconds,
// fraction and offset (Z, GMT, UTC, +hh, +hhmm or +hh:mm), the invariant
// MM/dd/yyyy with an optional time and AM or PM, and RFC 1123. Other forms
// DateTime.Parse would read are an error.
func Parse(s string, style DateTimeStyles, clock Clock) (time.Time, error) {
	s = strings.TrimSpace(s)
	if m := isoDate.FindStringSubmatch(s); m != nil {
		if t, ok := parsed(m[1], m[2], m[3], m[4], m[5], m[6], m[7], "", m[8], style, clock); ok {
			return t, nil
		}
	}
	if m := usDate.FindStringSubmatch(s); m != nil {
		if t, ok := parsed(m[3], m[1], m[2], m[4], m[5], m[6], "", m[7], "", style, clock); ok {
			return t, nil
		}
	}
	if t, ok := TryParseExact(s, "ddd, dd MMM yyyy HH':'mm':'ss 'GMT'", style|AssumeUniversal, clock); ok {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("%q isn't a date", s)
}

var (
	isoDate = regexp.MustCompile(`^(\d{4})-(\d{1,2})-(\d{1,2})(?:(?:T|\s+)(\d{1,2}):(\d{2})(?::(\d{2})(?:\.(\d{1,7}))?)?)?\s*(Z|GMT|UTC|[+-]\d{1,2}(?::?\d{2})?)?$`)
	usDate  = regexp.MustCompile(`^(\d{1,2})/(\d{1,2})/(\d{4})(?:\s+(\d{1,2}):(\d{2})(?::(\d{2}))?(?:\s*([AaPp][Mm]))?)?$`)
)

func parsed(year, month, day, hour, minute, second, fraction, mark, zone string, style DateTimeStyles, clock Clock) (time.Time, bool) {
	r := newDateResult()
	r.year, r.month, r.day = atoi(year), atoi(month), atoi(day)
	r.hour, r.minute, r.second = atoi(hour), atoi(minute), atoi(second)
	if fraction != "" {
		r.nanos = atoi((fraction + "000000000")[:9])
	}
	if mark != "" {
		r.hour12, r.mark = true, strings.ToUpper(mark)[0]
	}
	if zone != "" {
		off, ok := zoneOffset(zone)
		if !ok {
			return time.Time{}, false
		}
		r.offset = &off
	}
	return r.resolve(style, clock)
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// zoneOffset is seconds east of UTC for Z, GMT, UTC, +hh, +hhmm or +hh:mm.
func zoneOffset(z string) (int, bool) {
	if z == "Z" || z == "GMT" || z == "UTC" {
		return 0, true
	}
	sign := 1
	if z[0] == '-' {
		sign = -1
	}
	digits := strings.ReplaceAll(z[1:], ":", "")
	hours, minutes := digits, ""
	if len(digits) > 2 {
		hours, minutes = digits[:len(digits)-2], digits[len(digits)-2:]
	}
	h, m := atoi(hours), atoi(minutes)
	if h > 14 || m > 59 {
		return 0, false
	}
	return sign * (h*3600 + m*60), true
}

type dateResult struct {
	year, month, day int
	hour, minute     int
	second           int
	nanos            int
	twoDigitYear     bool
	hour12           bool
	mark             byte
	weekday          int
	offset           *int
}

func newDateResult() dateResult {
	return dateResult{year: -1, month: -1, day: -1, weekday: -1}
}

type exactParser struct {
	in     string
	pos    int
	format string
	fpos   int
	r      dateResult
}

func (p *exactParser) run() bool {
	for p.fpos < len(p.format) {
		if !p.step() {
			return false
		}
	}
	return true
}

// step reads the input for the format's next quoted text, escape,
// specifier or literal.
func (p *exactParser) step() bool {
	c := p.format[p.fpos]
	switch {
	case c == '\'' || c == '"':
		return p.quoted(c)
	case c == '\\' && p.fpos+1 < len(p.format):
		p.fpos += 2
		return p.literal(p.format[p.fpos-1])
	case c == '%' && p.fpos+1 < len(p.format):
		p.fpos += 2
		return p.specifier(p.format[p.fpos-1], 1)
	case strings.IndexByte("dfFhHKmMstyzZ", c) >= 0:
		return p.specifier(c, p.repeat(c))
	}
	p.fpos++
	return p.literal(c)
}

func (p *exactParser) repeat(c byte) int {
	n := 0
	for p.fpos < len(p.format) && p.format[p.fpos] == c {
		n++
		p.fpos++
	}
	return n
}

func (p *exactParser) quoted(q byte) bool {
	p.fpos++
	for p.fpos < len(p.format) && p.format[p.fpos] != q {
		c := p.format[p.fpos]
		if c == '\\' && p.fpos+1 < len(p.format) {
			p.fpos++
			c = p.format[p.fpos]
		}
		if !p.literal(c) {
			return false
		}
		p.fpos++
	}
	p.fpos++
	return true
}

func (p *exactParser) literal(c byte) bool {
	if p.pos >= len(p.in) || p.in[p.pos] != c {
		return false
	}
	p.pos++
	return true
}

func (p *exactParser) specifier(c byte, n int) bool {
	switch c {
	case 'y':
		return p.parseYear(n)
	case 'M':
		return p.parseMonth(n)
	case 'd':
		return p.parseDay(n)
	case 'h', 'H':
		p.r.hour12 = p.r.hour12 || c == 'h'
		return p.setPart(&p.r.hour, p.digits(min(n, 2)))
	case 'm':
		return p.setPart(&p.r.minute, p.digits(min(n, 2)))
	case 's':
		return p.setPart(&p.r.second, p.digits(min(n, 2)))
	case 'f', 'F':
		return p.parseFraction(n, c == 'f')
	case 't':
		return p.parseMark(n)
	case 'z':
		return p.parseOffset(n)
	case 'K':
		return p.parseK()
	case 'Z':
		for range n {
			if !p.literal('Z') {
				return false
			}
		}
		zero := 0
		p.r.offset = &zero
		return true
	}
	return false
}

// digits reads n digits, or one or two when n is 1, as .NET's ParseDigits.
func (p *exactParser) digits(n int) int {
	lo, hi := n, n
	if n == 1 {
		lo, hi = 1, 2
	}
	v, read := 0, 0
	for read < hi && p.pos < len(p.in) && isASCIIDigit(rune(p.in[p.pos])) {
		v = v*10 + int(p.in[p.pos]-'0')
		read++
		p.pos++
	}
	if read < lo {
		return -1
	}
	return v
}

func (p *exactParser) setPart(part *int, v int) bool {
	if v < 0 {
		return false
	}
	*part = v
	return true
}

func (p *exactParser) parseYear(n int) bool {
	p.r.twoDigitYear = n <= 2
	return p.setPart(&p.r.year, p.digits(n))
}

func (p *exactParser) parseMonth(n int) bool {
	if n <= 2 {
		return p.setPart(&p.r.month, p.digits(n))
	}
	names := monthAbbr
	if n > 3 {
		names = monthNames
	}
	i := p.matchName(names)
	return p.setPart(&p.r.month, i+1)
}

func (p *exactParser) parseDay(n int) bool {
	if n <= 2 {
		return p.setPart(&p.r.day, p.digits(n))
	}
	names := dayAbbr
	if n > 3 {
		names = dayNames
	}
	return p.setPart(&p.r.weekday, p.matchName(names))
}

// matchName finds the longest name that starts the input, ignoring case,
// and returns its index or -1.
func (p *exactParser) matchName(names []string) int {
	best := -1
	for i, name := range names {
		if len(p.in)-p.pos >= len(name) && strings.EqualFold(p.in[p.pos:p.pos+len(name)], name) &&
			(best < 0 || len(name) > len(names[best])) {
			best = i
		}
	}
	if best >= 0 {
		p.pos += len(names[best])
	}
	return best
}

func (p *exactParser) parseFraction(n int, exact bool) bool {
	start := p.pos
	v := 0
	for p.pos-start < n && p.pos < len(p.in) && isASCIIDigit(rune(p.in[p.pos])) {
		v = v*10 + int(p.in[p.pos]-'0')
		p.pos++
	}
	read := p.pos - start
	if exact && read != n {
		return false
	}
	for range 9 - read {
		v *= 10
	}
	p.r.nanos = v
	return true
}

func (p *exactParser) parseMark(n int) bool {
	for _, m := range []string{"AM", "PM"} {
		want := m
		if n == 1 {
			want = m[:1]
		}
		if len(p.in)-p.pos >= len(want) && strings.EqualFold(p.in[p.pos:p.pos+len(want)], want) {
			p.pos += len(want)
			p.r.mark = m[0]
			return true
		}
	}
	return false
}

// parseOffset is .NET's ParseTimeZoneOffset: a sign and hours for z and zz,
// and for zzz a sign, one or two hour digits, an optional ':' and two
// minute digits.
func (p *exactParser) parseOffset(n int) bool {
	if p.pos >= len(p.in) || (p.in[p.pos] != '+' && p.in[p.pos] != '-') {
		return false
	}
	sign := 1
	if p.in[p.pos] == '-' {
		sign = -1
	}
	p.pos++
	hourDigits := n
	if n > 2 {
		hourDigits = 1
	}
	hours := p.digits(hourDigits)
	if hours < 0 {
		return false
	}
	minutes := 0
	if n > 2 {
		if p.pos < len(p.in) && p.in[p.pos] == ':' {
			p.pos++
		}
		minutes = p.digits(2)
		if minutes < 0 || minutes >= 60 {
			return false
		}
	}
	off := sign * (hours*3600 + minutes*60)
	if p.r.offset != nil && *p.r.offset != off {
		return false
	}
	p.r.offset = &off
	return true
}

func (p *exactParser) parseK() bool {
	if p.pos < len(p.in) && p.in[p.pos] == 'Z' {
		p.pos++
		zero := 0
		p.r.offset = &zero
		return true
	}
	if p.pos < len(p.in) && (p.in[p.pos] == '+' || p.in[p.pos] == '-') {
		return p.parseOffset(3)
	}
	return true
}

// resolve fills the parts the format omitted from today, applies AM and
// PM, checks the date and the day of the week, and places the time.
func (r *dateResult) resolve(style DateTimeStyles, clock Clock) (time.Time, bool) {
	now := clock.Local()
	switch {
	case r.year < 0 && r.month < 0 && r.day < 0:
		r.year, r.month, r.day = now.Year(), int(now.Month()), now.Day()
	case r.month < 0 && r.day < 0:
		r.month, r.day = 1, 1
	default:
		if r.year < 0 {
			r.year = now.Year()
		}
		if r.month < 0 {
			r.month = 1
		}
		if r.day < 0 {
			r.day = 1
		}
	}
	if r.twoDigitYear && r.year < 100 {
		r.year = fourDigitYear(r.year)
	}
	if !r.applyMark() || !validDate(r.year, r.month, r.day) || r.minute > 59 || r.second > 59 {
		return time.Time{}, false
	}
	wall := time.Date(r.year, time.Month(r.month), r.day, r.hour, r.minute, r.second, r.nanos, time.UTC)
	if r.weekday >= 0 && int(wall.Weekday()) != r.weekday {
		return time.Time{}, false
	}
	return place(wall, r.offset, style, clock), true
}

// place reads wall's clock reading at offset seconds east of UTC, or,
// without one, in the zone style names.
func place(wall time.Time, offset *int, style DateTimeStyles, clock Clock) time.Time {
	var t time.Time
	switch {
	case offset != nil:
		t = wall.Add(-time.Duration(*offset) * time.Second)
	case style&AssumeUniversal != 0:
		t = wall
	default:
		t = inZone(wall, clock.Location)
	}
	if style&AdjustToUniversal != 0 {
		return t.UTC()
	}
	return t.In(clock.Location)
}
func (r *dateResult) applyMark() bool {
	if r.hour12 {
		if r.hour > 12 {
			return false
		}
		switch {
		case r.mark != 'P' && r.hour == 12:
			r.hour = 0
		case r.mark == 'P' && r.hour < 12:
			r.hour += 12
		}
		return true
	}
	if (r.mark == 'A' && r.hour >= 12) || (r.mark == 'P' && r.hour < 12) {
		return false
	}
	return r.hour < 24
}

// fourDigitYear is the Gregorian calendar's ToFourDigitYear with
// TwoDigitYearMax 2049.
func fourDigitYear(y int) int {
	if y <= 49 {
		return 2000 + y
	}
	return 1900 + y
}

func validDate(y, m, d int) bool {
	if y < 1 || y > 9999 || m < 1 || m > 12 || d < 1 {
		return false
	}
	return d <= time.Date(y, time.Month(m)+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// inZone reads wall's clock reading as a time in loc.
func inZone(wall time.Time, loc *time.Location) time.Time {
	return time.Date(wall.Year(), wall.Month(), wall.Day(), wall.Hour(), wall.Minute(), wall.Second(), wall.Nanosecond(), loc)
}

var (
	monthNames = []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
	monthAbbr  = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	dayNames   = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	dayAbbr    = []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
)

func isASCIIDigit(r rune) bool { return r >= '0' && r <= '9' }
