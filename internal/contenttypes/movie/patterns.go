package movie

import (
	"fmt"

	"github.com/jatassi/slipstream/internal/contenttypes/movie/internal/upstream"
	"github.com/jatassi/slipstream/internal/regex"
	"github.com/jatassi/slipstream/internal/release"
)

// fixed is a Radarr pattern compiled once.
type fixed struct{ re *regex.Regex }

const compiledOptions = upstream.IgnoreCase | upstream.Compiled

func compile(p upstream.Pattern) fixed {
	if extra := p.Options &^ compiledOptions; extra != 0 {
		panic(fmt.Sprintf("%s has .NET options %d that the parser doesn't run", p.Key, extra))
	}
	o := regex.Options{IgnoreCase: p.Options&upstream.IgnoreCase != 0, Timeout: release.PatternLimit}
	return fixed{regex.MustCompile(p.Text, o)}
}

func compileAll(ps []upstream.Pattern) []fixed {
	out := make([]fixed, len(ps))
	for i, p := range ps {
		out[i] = compile(p)
	}
	return out
}

func (f fixed) find(t regex.Text) (regex.Match, bool) {
	m, ok, err := f.re.Find(t)
	return m, err == nil && ok
}

func (f fixed) replace(s, with string) string {
	out, err := f.re.Replace(regex.TextOf(s), with)
	if err != nil {
		return s
	}
	return out
}

// split is .NET's Regex.Split for a pattern with no capturing group.
func (f fixed) split(s string) []string {
	t := regex.TextOf(s)
	all, err := f.re.FindAll(t)
	if err != nil {
		return []string{s}
	}
	r := []rune(s)
	out := make([]string, 0, len(all)+1)
	at := 0
	for _, m := range all {
		out = append(out, string(r[at:m.Index()]))
		at = m.Index() + m.Length()
	}
	return append(out, string(r[at:]))
}
