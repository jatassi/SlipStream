package seeding

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestParseRatio(t *testing.T) {
	for _, c := range []struct {
		in   float64
		want Ratio
		err  error
	}{
		{1.2, 120, nil},
		{0.29, 29, nil},
		{0.01, 1, nil},
		{2, 200, nil},
		{0, 0, ErrRatio},
		{-1, 0, ErrRatio},
		{0.005, 0, ErrRatio},
		{1.234, 0, ErrRatio},
		{math.NaN(), 0, ErrRatio},
		{math.Inf(1), 0, ErrRatio},
	} {
		got, err := ParseRatio(c.in)
		if got != c.want || !errors.Is(err, c.err) {
			t.Errorf("ParseRatio(%v) = %d, %v, want %d, %v", c.in, got, err, c.want, c.err)
		}
	}
}

func TestRuleOf(t *testing.T) {
	for _, c := range []struct {
		name  string
		as    Clears
		ratio Ratio
		seed  time.Duration
		want  Rule
	}{
		{"time", ClearsTime, 0, 10 * day, TimeRule{Seed: 10 * day}},
		{"ratio", ClearsRatio, 100, 0, RatioRule{Ratio: 100}},
		{"either", ClearsEither, 100, 10 * day, EitherRule{Ratio: 100, Seed: 10 * day}},
		{"both", ClearsBoth, 100, 10 * day, BothRule{Ratio: 100, Seed: 10 * day}},
		{"a time rule stating a ratio", ClearsTime, 100, 10 * day, nil},
		{"a ratio rule with no ratio", ClearsRatio, 0, 0, nil},
		{"an either rule missing its time", ClearsEither, 100, 0, nil},
		{"a both rule missing its ratio", ClearsBoth, 0, 10 * day, nil},
		{"a negative time", ClearsTime, 0, -day, nil},
		{"an unknown spelling", "any", 100, 10 * day, nil},
	} {
		got, err := RuleOf(c.as, c.ratio, c.seed)
		if got != c.want || (c.want == nil) != errors.Is(err, ErrRule) {
			t.Errorf("%s: RuleOf = %v, %v, want %v", c.name, got, err, c.want)
		}
		if got == nil {
			continue
		}
		if ratio, seed := got.Parts(); got.Clears() != c.as || ratio != c.ratio || seed != c.seed {
			t.Errorf("%s: Clears and Parts = %s %d %s, want what RuleOf took", c.name, got.Clears(), ratio, seed)
		}
	}
}

func TestStated(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	d := func(v time.Duration) *time.Duration { return &v }
	for _, c := range []struct {
		name  string
		ratio *float64
		seed  *time.Duration
		want  Rule
	}{
		{"a ratio and a time either clears", f(1), d(10 * day), EitherRule{Ratio: 100, Seed: 10 * day}},
		{"only a time counts only time", nil, d(10 * day), TimeRule{Seed: 10 * day}},
		{"only a ratio counts only ratio", f(0.29), nil, RatioRule{Ratio: 29}},
		{"a ratio rounds up, so it is never looser than the site's", f(0.999), nil, RatioRule{Ratio: 100}},
		{"a zero ratio states nothing", f(0), d(10 * day), TimeRule{Seed: 10 * day}},
		{"a zero time states nothing", f(1), d(0), RatioRule{Ratio: 100}},
		{"neither states no rule", nil, nil, nil},
	} {
		if got := Stated(c.ratio, c.seed); got != c.want {
			t.Errorf("%s: Stated = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestStrictest(t *testing.T) {
	for _, c := range []struct {
		name string
		in   []Rule
		want Rule
	}{
		{"none", nil, nil},
		{"one that ratio can't clear beats one it can", []Rule{EitherRule{Ratio: 100, Seed: 10 * day}, TimeRule{Seed: 10 * day}}, TimeRule{Seed: 10 * day}},
		{"the longer time", []Rule{TimeRule{Seed: 5 * day}, TimeRule{Seed: 10 * day}}, TimeRule{Seed: 10 * day}},
		{"a time beats a ratio alone", []Rule{RatioRule{Ratio: 100}, EitherRule{Ratio: 100, Seed: day}}, EitherRule{Ratio: 100, Seed: day}},
		{"the higher ratio", []Rule{RatioRule{Ratio: 150}, RatioRule{Ratio: 100}}, RatioRule{Ratio: 150}},
		{"both beats either on the same parts", []Rule{EitherRule{Ratio: 100, Seed: 10 * day}, BothRule{Ratio: 100, Seed: 10 * day}}, BothRule{Ratio: 100, Seed: 10 * day}},
	} {
		if got := Strictest(c.in); got != c.want {
			t.Errorf("%s: Strictest = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestAHighRatioOwesWithoutOverflow(t *testing.T) {
	o, met := owes(RatioRule{Ratio: math.MaxInt64}, Totals{Downloaded: 4 << 40}, 0)
	if met || o.Bytes != math.MaxInt64 {
		t.Errorf("owes = %+v, %v, want the most bytes there are", o, met)
	}
}
