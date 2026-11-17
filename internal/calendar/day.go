package calendar

import (
	"errors"
	"fmt"
	"time"
)

// Day is a full civil date. Only its constructors make one, so a partial
// date never reaches the grid.
type Day struct {
	year  int
	month time.Month
	day   int
}

// ParseDay reads YYYY-MM-DD, a date the calendar has.
func ParseDay(s string) (Day, error) {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return Day{}, fmt.Errorf("%q isn't a day: %w", s, err)
	}
	return DayOf(t), nil
}

// NewDay is the civil date y-m-d, normalized as time.Date normalizes one.
func NewDay(y int, m time.Month, d int) Day { return DayOf(time.Date(y, m, d, 0, 0, 0, 0, time.UTC)) }

// DayOf is the civil date of t in its own zone.
func DayOf(t time.Time) Day {
	y, m, d := t.Date()
	return Day{year: y, month: m, day: d}
}

func (d Day) String() string { return fmt.Sprintf("%04d-%02d-%02d", d.year, d.month, d.day) }

// Add is the day n days after d, before it when n is negative.
func (d Day) Add(n int) Day { return DayOf(d.midnight(time.UTC).AddDate(0, 0, n)) }

func (d Day) Compare(o Day) int { return d.midnight(time.UTC).Compare(o.midnight(time.UTC)) }

// midnight is when d starts in loc. time.Date normalizes a day past the
// month's end, which is how Add and the next day's start step over it.
func (d Day) midnight(loc *time.Location) time.Time {
	return time.Date(d.year, d.month, d.day, 0, 0, 0, 0, loc)
}

// MaxSpan is a request bound: the six weeks a month grid shows.
const MaxSpan = 42

// Span is the days from From up to To, exclusive. Only NewSpan makes one,
// so it holds 1 to MaxSpan days.
type Span struct{ from, to Day }

var (
	ErrSpanTooLong = errors.New("the span is over 42 days")
	ErrSpanEmpty   = errors.New("the span ends on or before its first day")
)

func NewSpan(from, to Day) (Span, error) {
	switch {
	case to.Compare(from) <= 0:
		return Span{}, ErrSpanEmpty
	case to.Compare(from.Add(MaxSpan)) > 0:
		return Span{}, ErrSpanTooLong
	}
	return Span{from: from, to: to}, nil
}

func (s Span) From() Day { return s.from }

// To is the day after the span's last.
func (s Span) To() Day { return s.to }

func (s Span) Contains(d Day) bool { return d.Compare(s.from) >= 0 && d.Compare(s.to) < 0 }

func (s Span) covers(o Span) bool { return o.from.Compare(s.from) >= 0 && o.to.Compare(s.to) <= 0 }
