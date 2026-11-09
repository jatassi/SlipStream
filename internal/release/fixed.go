package release

import (
	"fmt"

	"github.com/jatassi/slipstream/internal/regex"
	"github.com/jatassi/slipstream/internal/release/internal/upstream"
)

// fixed is an upstream pattern compiled once. Sonarr and Radarr set no
// match timeout on these patterns and neither does SlipStream, so a call
// errs only on a replacement it can't read, which a test would meet.
type fixed struct{ re *regex.Regex }

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
	}
	return fixed{re: regex.MustCompile(p.Text, o)}
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

func (f fixed) match(t regex.Text) bool {
	ok, err := f.re.Match(t)
	if err != nil {
		panic(err)
	}
	return ok
}

func (f fixed) find(t regex.Text) (regex.Match, bool) {
	m, ok, err := f.re.Find(t)
	if err != nil {
		panic(err)
	}
	return m, ok
}

func (f fixed) findAll(t regex.Text) []regex.Match {
	all, err := f.re.FindAll(t)
	if err != nil {
		panic(err)
	}
	return all
}

// replace is r applied to t.
func (r replacer) replace(t regex.Text) regex.Text {
	out, err := r.re.Replace(t, r.with)
	if err != nil {
		panic(err)
	}
	if out == t.String() {
		return t
	}
	return regex.TextOf(out)
}

// group is the first of names that f's first match on t captured, or ""
// when f doesn't match.
func (f fixed) group(t regex.Text, names []string) string {
	m, ok := f.find(t)
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
