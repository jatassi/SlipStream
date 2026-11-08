package release

import (
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/regex"
	"github.com/jatassi/slipstream/internal/release/internal/upstream"
)

// fixedLimit is the compiled-in limit on one fixed pattern's match, the
// formats Timings default, which the parser doesn't read from settings.
const fixedLimit = 100 * time.Millisecond

// fixed is an upstream pattern compiled once, named by its upstream key.
type fixed struct {
	key string
	re  *regex.Regex
}

// compiledOptions are the .NET options that change nothing on regexp2 or
// that regex.Options carries.
const compiledOptions = upstream.IgnoreCase | upstream.IgnorePatternWhitespace | upstream.Compiled | upstream.CultureInvariant

func compile(p upstream.Pattern) fixed {
	if extra := p.Options &^ compiledOptions; extra != 0 {
		panic(fmt.Sprintf("%s has .NET options %d that the parser doesn't run", p.Key, extra))
	}
	o := regex.Options{
		IgnoreCase:              p.Options&upstream.IgnoreCase != 0,
		IgnorePatternWhitespace: p.Options&upstream.IgnorePatternWhitespace != 0,
		Timeout:                 fixedLimit,
	}
	return fixed{key: p.Key, re: regex.MustCompile(p.Text, o)}
}

func compileAll(ps []upstream.Pattern) []fixed {
	out := make([]fixed, len(ps))
	for i, p := range ps {
		out[i] = compile(p)
	}
	return out
}

// replacer is an upstream RegexReplace compiled once.
type replacer struct {
	fixed
	with string
}

func compileReplace(r upstream.Replace) replacer {
	return replacer{compile(r.Pattern), r.With}
}

func compileReplaces(rs []upstream.Replace) []replacer {
	out := make([]replacer, len(rs))
	for i, r := range rs {
		out[i] = compileReplace(r)
	}
	return out
}

// scan runs fixed patterns over one name and keeps the key of each that
// ran past fixedLimit. A pattern that did reads as no match, so the fact
// it reads stays unknown.
type scan struct{ slow []string }

func (s *scan) timedOut(f fixed, err error) bool {
	if err != nil {
		s.slow = append(s.slow, f.key)
	}
	return err != nil
}

func (s *scan) match(f fixed, t regex.Text) bool {
	ok, err := f.re.Match(t)
	return !s.timedOut(f, err) && ok
}

func (s *scan) find(f fixed, t regex.Text) (regex.Match, bool) {
	m, ok, err := f.re.Find(t)
	return m, !s.timedOut(f, err) && ok
}

func (s *scan) findAll(f fixed, t regex.Text) []regex.Match {
	all, err := f.re.FindAll(t)
	if s.timedOut(f, err) {
		return nil
	}
	return all
}

// replace is r applied to t, or t when r ran past the limit.
func (s *scan) replace(r replacer, t regex.Text) regex.Text {
	out, err := r.re.Replace(t, r.with)
	if s.timedOut(r.fixed, err) || out == t.String() {
		return t
	}
	return regex.TextOf(out)
}

// group is the first of names that f's first match on t captured, or ""
// when f doesn't match.
func (s *scan) group(f fixed, t regex.Text, names []string) string {
	m, ok := s.find(f, t)
	if !ok {
		return ""
	}
	return firstGroup(m, names)
}

// firstGroup is the first of names m captured, or "".
func firstGroup(m regex.Match, names []string) string {
	for _, n := range names {
		if _, ok := m.Group(n); ok {
			return n
		}
	}
	return ""
}
