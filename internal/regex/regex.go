// Package regex runs patterns written for .NET, on regexp2: Sonarr's and
// Radarr's parser, TRaSH's Custom Formats, Cardigann's definitions and
// the ported sites. No other package imports regexp2 (lint rule
// regexp2-confined).
package regex

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/dlclark/regexp2"
	"github.com/dlclark/regexp2/syntax"
)

// Options are how a pattern compiles. Timeout bounds one match. Zero is
// .NET's InfiniteMatchTimeout, for a fixed pattern SlipStream wrote or
// ported whose upstream sets none.
type Options struct {
	IgnoreCase              bool
	IgnorePatternWhitespace bool
	Timeout                 time.Duration
}

// regexp2 reads match deadlines from a clock that ticks every 100 ms and
// adds one tick to every limit, so a 100 ms limit fired after 200 ms.
// A 10 ms tick keeps a timeout within 20 ms of its limit.
func init() { regexp2.SetTimeoutCheckPeriod(10 * time.Millisecond) }

// Regex is a compiled pattern, safe for concurrent use.
type Regex struct {
	re      *regexp2.Regexp
	pattern string
	flags   regexp2.RegexOptions
}

// Compile compiles pattern, rewriting .NET's \p{IsBlock} classes first.
// A pattern that doesn't compile is a *CompileError.
func Compile(pattern string, o Options) (*Regex, error) {
	rewritten, err := rewriteBlocks(pattern)
	if err != nil {
		return nil, err
	}
	re, err := regexp2.Compile(rewritten, o.flags())
	if err != nil {
		return nil, compileError(pattern, o, err)
	}
	if o.Timeout > 0 {
		re.MatchTimeout = o.Timeout
	}
	return &Regex{re: re, pattern: pattern, flags: o.flags()}, nil
}

// MustCompile is Compile for generated tables, whose patterns CI compiles.
func MustCompile(pattern string, o Options) *Regex {
	r, err := Compile(pattern, o)
	if err != nil {
		panic(err)
	}
	return r
}

func (o Options) flags() regexp2.RegexOptions {
	f := regexp2.None
	if o.IgnoreCase {
		f |= regexp2.IgnoreCase
	}
	if o.IgnorePatternWhitespace {
		f |= regexp2.IgnorePatternWhitespace
	}
	return f
}

// Text is a string split into runes once. regexp2 matches runes, so a
// name that 300 parser patterns and 1,300 format patterns read converts
// once.
type Text struct {
	s string
	r []rune
}

func TextOf(s string) Text { return Text{s: s, r: []rune(s)} }

func (t Text) String() string { return t.s }

// String is the pattern as written, before any \p{IsBlock} rewrite.
func (r *Regex) String() string { return r.pattern }

func (r *Regex) Match(t Text) (bool, error) {
	return run(r, func(re *regexp2.Regexp) (bool, error) { return re.MatchRunes(t.r) })
}

func (r *Regex) Find(t Text) (Match, bool, error) {
	m, err := r.first(t)
	if err != nil {
		return Match{}, false, err
	}
	return Match{m}, m != nil, nil
}

// FindAll is .NET's Matches(): every match, left to right.
func (r *Regex) FindAll(t Text) ([]Match, error) {
	var all []Match
	m, err := r.first(t)
	for ; m != nil && err == nil; m, err = r.next(m) {
		all = append(all, Match{m})
	}
	if err != nil {
		return nil, err
	}
	return all, nil
}

// Replace replaces every match, reading $1, ${name}, $$, $&, $`, $', $+
// and $_ in with as .NET does. regexp2's own Replace returns "" and no
// error when a match after the first times out, so the loop is here.
func (r *Regex) Replace(t Text, with string) (string, error) {
	rules, err := r.replacement(with)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	at := 0
	m, err := r.first(t)
	for ; m != nil && err == nil; m, err = r.next(m) {
		b.WriteString(string(t.r[at:m.Index]))
		rules.write(&b, Match{m}, t.r)
		at = m.Index + m.Length
	}
	if err != nil {
		return "", err
	}
	b.WriteString(string(t.r[at:]))
	return b.String(), nil
}

func (r *Regex) first(t Text) (*regexp2.Match, error) {
	return run(r, func(re *regexp2.Regexp) (*regexp2.Match, error) { return re.FindRunesMatch(t.r) })
}

func (r *Regex) next(m *regexp2.Match) (*regexp2.Match, error) {
	return run(r, func(re *regexp2.Regexp) (*regexp2.Match, error) { return re.FindNextMatch(m) })
}

// run makes one regexp2 call, on r's compile first. regexp2 reads its
// deadline from a clock that a goroutine advances, and under load the
// goroutine runs late, so the clock leaps and fires a deadline
// microseconds after the call began. run checks each timeout against the
// monotonic clock and runs a false one again on a fresh compile limited
// to the time left, so a *TimeoutError means the limit passed. A false
// timeout needs the clock to advance by the time left plus a tick, so
// the reruns end once the limit passes.
func run[T any](r *Regex, call func(*regexp2.Regexp) (T, error)) (T, error) {
	start := time.Now()
	re := r.re
	for {
		v, err := call(re)
		if err == nil {
			return v, nil
		}
		left := r.re.MatchTimeout - time.Since(start)
		if left <= 0 {
			return v, &TimeoutError{Pattern: r.pattern, Limit: r.re.MatchTimeout}
		}
		re = regexp2.MustCompile(r.re.String(), r.flags)
		re.MatchTimeout = left
	}
}

// replacement is a parsed replacement string: literal runs and group
// numbers, with regexp2's specials for $`, $', $+ and $_ below zero.
type replacement struct {
	data *syntax.ReplacerData
	last int
}

const (
	leftPortion  = -1
	rightPortion = -2
	lastGroup    = -3
	wholeString  = -4
	specials     = 4
)

func (r *Regex) replacement(with string) (replacement, error) {
	numbers := r.re.GetGroupNumbers()
	slots := make(map[int]int, len(numbers))
	for _, n := range numbers {
		slots[n] = n
	}
	names := make(map[string]int, len(numbers))
	for _, name := range r.re.GetGroupNames() {
		names[name] = r.re.GroupNumberFromName(name)
	}
	data, err := syntax.NewReplacerData(with, slots, len(numbers), names, syntax.RegexOptions(0))
	if err != nil {
		return replacement{}, fmt.Errorf("read the replacement %q: %w", with, err)
	}
	return replacement{data: data, last: numbers[len(numbers)-1]}, nil
}

func (p replacement) write(b *bytes.Buffer, m Match, text []rune) {
	for _, rule := range p.data.Rules {
		switch {
		case rule >= 0:
			b.WriteString(p.data.Strings[rule])
		case rule < -specials:
			b.WriteString(m.number(-specials - 1 - rule))
		default:
			switch -specials - 1 - rule {
			case leftPortion:
				b.WriteString(string(text[:m.m.Index]))
			case rightPortion:
				b.WriteString(string(text[m.m.Index+m.m.Length:]))
			case lastGroup:
				b.WriteString(m.number(p.last))
			case wholeString:
				b.WriteString(string(text))
			}
		}
	}
}

// Match is one match. Group answers a named or numbered group's last
// capture, Captures every capture of a repeated group, which Sonarr's
// episode lists read (Groups["episode"].Captures).
type Match struct{ m *regexp2.Match }

func (m Match) Index() int  { return m.m.Index }
func (m Match) Length() int { return m.m.Length }

func (m Match) Value() string { return m.m.String() }

// Group is the group's last capture, and false when the pattern has no
// such group or it captured nothing.
func (m Match) Group(name string) (string, bool) {
	g := m.group(name)
	if g == nil || len(g.Captures) == 0 {
		return "", false
	}
	return g.String(), true
}

func (m Match) Captures(name string) []string {
	g := m.group(name)
	if g == nil {
		return nil
	}
	out := make([]string, len(g.Captures))
	for i := range g.Captures {
		out[i] = g.Captures[i].String()
	}
	return out
}

// group is .NET's Groups[name]: a group by name, else by number. regexp2
// reads a number only when the pattern names no group.
func (m Match) group(name string) *regexp2.Group {
	if g := m.m.GroupByName(name); g != nil {
		return g
	}
	n, err := strconv.ParseUint(name, 10, 31)
	if err != nil {
		return nil
	}
	return m.m.GroupByNumber(int(n))
}

func (m Match) number(n int) string { return m.m.GroupByNumber(n).String() }
