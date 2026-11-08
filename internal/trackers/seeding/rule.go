// Package seeding decides an info hash's hit and run window and seeding
// goal. It reads no clock and no store: trackers gathers a Hash in one
// snapshot, calls Evaluate and writes what the Verdict says, and every
// reader shows the same Verdict. Its imports are the standard library.
package seeding

import (
	"cmp"
	"errors"
	"math"
	"slices"
	"time"
)

// Ratio is a share ratio in hundredths: 150 is 1.50. Hundredths keep every
// comparison exact, so a goal equal to the default reads as equal.
type Ratio int64

var ErrRatio = errors.New("a ratio is 0.01 or more with at most two decimals")

// ParseRatio reads a ratio an Admin typed.
func ParseRatio(v float64) (Ratio, error) {
	h := math.Round(v * 100)
	if !(h >= 1 && h <= 1<<53) || math.Abs(v*100-h) > 1e-6 {
		return 0, ErrRatio
	}
	return Ratio(h), nil
}

func (r Ratio) Float() float64 { return float64(r) / 100 }

// RatioOf is S28's Ratio on real bytes in hundredths, rounded down:
// uploaded over the larger of downloaded and size.
func RatioOf(t Totals) Ratio {
	base := max(t.Downloaded, t.Size)
	if base == 0 {
		return 0
	}
	return Ratio(t.Uploaded/base*100 + t.Uploaded%base*100/base)
}

// Rule is what a Tracker requires before a torrent may stop seeding. It is
// one of TimeRule, RatioRule, EitherRule and BothRule.
type Rule interface {
	Clears() Clears
	// Parts are its ratio and seeding time, zero for one it doesn't count.
	Parts() (Ratio, time.Duration)
	// valid is every part it counts above zero.
	valid() bool
}

type (
	TimeRule   struct{ Seed time.Duration }
	RatioRule  struct{ Ratio Ratio }
	EitherRule struct {
		Ratio Ratio
		Seed  time.Duration
	}
	BothRule struct {
		Ratio Ratio
		Seed  time.Duration
	}
)

func (TimeRule) Clears() Clears   { return ClearsTime }
func (RatioRule) Clears() Clears  { return ClearsRatio }
func (EitherRule) Clears() Clears { return ClearsEither }
func (BothRule) Clears() Clears   { return ClearsBoth }

func (r TimeRule) Parts() (Ratio, time.Duration)   { return 0, r.Seed }
func (r RatioRule) Parts() (Ratio, time.Duration)  { return r.Ratio, 0 }
func (r EitherRule) Parts() (Ratio, time.Duration) { return r.Ratio, r.Seed }
func (r BothRule) Parts() (Ratio, time.Duration)   { return r.Ratio, r.Seed }

func (r TimeRule) valid() bool   { return r.Seed > 0 }
func (r RatioRule) valid() bool  { return r.Ratio > 0 }
func (r EitherRule) valid() bool { return r.Ratio > 0 && r.Seed > 0 }
func (r BothRule) valid() bool   { return r.Ratio > 0 && r.Seed > 0 }

// Clears is what meets a rule, as the API and trackers_rule spell it.
type Clears string

const (
	ClearsTime   Clears = "time"
	ClearsRatio  Clears = "ratio"
	ClearsEither Clears = "either"
	ClearsBoth   Clears = "both"
)

var ErrRule = errors.New("a rule states exactly the ratio and time it counts")

// RuleOf builds the rule c names from its parts, zero for a part it
// doesn't count. It is the inverse of Parts.
func RuleOf(c Clears, ratio Ratio, seed time.Duration) (Rule, error) {
	var r Rule
	switch c {
	case ClearsTime:
		r = TimeRule{Seed: seed}
	case ClearsRatio:
		r = RatioRule{Ratio: ratio}
	case ClearsEither:
		r = EitherRule{Ratio: ratio, Seed: seed}
	case ClearsBoth:
		r = BothRule{Ratio: ratio, Seed: seed}
	default:
		return nil, ErrRule
	}
	if gotRatio, gotSeed := r.Parts(); gotRatio != ratio || gotSeed != seed || !r.valid() {
		return nil, ErrRule
	}
	return r, nil
}

// Stated is the rule one Release states: either when it states both, the
// one it states otherwise, nil when it states neither. A ratio rounds up
// to hundredths, so the rule is never looser than the site's.
func Stated(ratio *float64, seed *time.Duration) Rule {
	var r Ratio
	if ratio != nil {
		if h := math.Ceil(*ratio*100 - 1e-6); h >= 1 && h <= 1<<53 {
			r = Ratio(h)
		}
	}
	var s time.Duration
	if seed != nil {
		s = max(*seed, 0)
	}
	switch {
	case r > 0 && s > 0:
		return EitherRule{Ratio: r, Seed: s}
	case r > 0:
		return RatioRule{Ratio: r}
	case s > 0:
		return TimeRule{Seed: s}
	}
	return nil
}

// Strictest is the rule among several Grabs' floors that clears last: one
// that ratio alone can't clear, then the longer time, then one that time
// alone can't clear, then the higher ratio. Nil for none.
func Strictest(rs []Rule) Rule {
	if len(rs) == 0 {
		return nil
	}
	return slices.MaxFunc(rs, func(a, b Rule) int {
		ar, as := a.Parts()
		br, bs := b.Parts()
		return cmp.Or(
			compareBool(needsTime(a), needsTime(b)),
			cmp.Compare(as, bs),
			compareBool(needsRatio(a), needsRatio(b)),
			cmp.Compare(ar, br),
		)
	})
}

func compareBool(a, b bool) int {
	switch {
	case a == b:
		return 0
	case a:
		return 1
	}
	return -1
}

func needsTime(r Rule) bool  { return r.Clears() == ClearsTime || r.Clears() == ClearsBoth }
func needsRatio(r Rule) bool { return r.Clears() == ClearsRatio || r.Clears() == ClearsBoth }

// Owes is what an owing window still needs to meet its rule. Time includes
// the margin. A part is zero where the rule doesn't count it or it is
// reached, and Clears says whether either part or both still meet it.
type Owes struct {
	Time   time.Duration
	Bytes  int64
	Clears Clears
}

// owes is what t still lacks of r, and whether r is met.
func owes(r Rule, t Totals, margin time.Duration) (Owes, bool) {
	ratio, seed := r.Parts()
	o := Owes{Clears: r.Clears()}
	if seed > 0 {
		o.Time = max(seed+margin-t.Announced, 0)
	}
	if ratio > 0 {
		o.Bytes = uploadShort(ratio, t)
	}
	if o.Clears == ClearsEither {
		return o, o.Time == 0 || o.Bytes == 0
	}
	return o, o.Time == 0 && o.Bytes == 0
}

// uploadShort is how many more bytes t must upload to reach ratio r:
// uploaded*100 >= r*max(downloaded, size). It splits r into whole and
// hundredths so a high ratio saturates instead of overflowing.
func uploadShort(r Ratio, t Totals) int64 {
	base := max(t.Downloaded, t.Size)
	whole, part := int64(r)/100, int64(r)%100
	if base > 0 && whole >= math.MaxInt64/base {
		return math.MaxInt64
	}
	return max(base*whole+(base*part+99)/100-t.Uploaded, 0)
}
