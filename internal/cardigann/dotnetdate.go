package cardigann

import (
	"strings"
	"time"
)

// parseExact is .NET's DateTime.TryParseExact with the invariant culture
// and DateTimeStyles.None over a custom format such as
// "yyyy-MM-dd HH:mm:ss zzz". A date with an offset is moved into the
// clock's zone; one without is read as the clock's local time. A part the
// format omits takes today's date.
func parseExact(s, format string, clock Clock) (time.Time, bool) {
	p := exactParser{in: s, format: format, r: newDateResult()}
	if !p.run() || p.pos != len(p.in) {
		return time.Time{}, false
	}
	return p.r.resolve(clock)
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
func (r *dateResult) resolve(clock Clock) (time.Time, bool) {
	now := clock.local()
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
	if r.offset != nil {
		return wall.Add(-time.Duration(*r.offset) * time.Second).In(clock.Location), true
	}
	return inZone(wall, clock.Location), true
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
