// Package regex runs patterns written for .NET, on regexp2: Sonarr's and
// Radarr's parser, TRaSH's Custom Formats and Cardigann's definitions.
package regex

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/dlclark/regexp2"
	"github.com/dlclark/regexp2/syntax"
)

// Options are how a pattern compiles. Every caller sets a Timeout.
type Options struct {
	IgnoreCase bool
	Timeout    time.Duration
}

// regexp2 reads match deadlines from a clock that ticks every 100 ms and
// adds one tick to every limit, so a 100 ms limit fired after 200 ms.
// A 10 ms tick keeps a timeout within 20 ms of its limit.
func init() { regexp2.SetTimeoutCheckPeriod(10 * time.Millisecond) }

// Regex is a compiled pattern, safe for concurrent use.
type Regex struct {
	re      *regexp2.Regexp
	pattern string
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
	re.MatchTimeout = o.Timeout
	return &Regex{re: re, pattern: pattern}, nil
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
	if o.IgnoreCase {
		return regexp2.IgnoreCase
	}
	return regexp2.None
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
	ok, err := r.re.MatchRunes(t.r)
	if err != nil {
		return false, r.timeout()
	}
	return ok, nil
}

func (r *Regex) Find(t Text) (Match, bool, error) {
	m, err := r.re.FindRunesMatch(t.r)
	if err != nil {
		return Match{}, false, r.timeout()
	}
	return Match{m}, m != nil, nil
}

// FindAll is .NET's Matches(): every match, left to right.
func (r *Regex) FindAll(t Text) ([]Match, error) {
	var all []Match
	m, err := r.re.FindRunesMatch(t.r)
	for ; m != nil && err == nil; m, err = r.re.FindNextMatch(m) {
		all = append(all, Match{m})
	}
	if err != nil {
		return nil, r.timeout()
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
	m, err := r.re.FindRunesMatch(t.r)
	for ; m != nil && err == nil; m, err = r.re.FindNextMatch(m) {
		b.WriteString(string(t.r[at:m.Index]))
		rules.write(&b, Match{m}, t.r)
		at = m.Index + m.Length
	}
	if err != nil {
		return "", r.timeout()
	}
	b.WriteString(string(t.r[at:]))
	return b.String(), nil
}

func (r *Regex) timeout() error {
	return &TimeoutError{Pattern: r.pattern, Limit: r.re.MatchTimeout}
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
