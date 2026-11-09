package movie

import (
	"fmt"

	"github.com/jatassi/slipstream/internal/contenttypes/movie/internal/upstream"
	"github.com/jatassi/slipstream/internal/regex"
)

// fixed is a Radarr pattern compiled once. Radarr sets no match timeout
// on its parser patterns and neither does SlipStream, so a call errs only
// on a replacement it can't read, which a test would meet.
type fixed struct{ re *regex.Regex }

const compiledOptions = upstream.IgnoreCase | upstream.Compiled

func compile(p upstream.Pattern) fixed {
	if extra := p.Options &^ compiledOptions; extra != 0 {
		panic(fmt.Sprintf("%s has .NET options %d that the parser doesn't run", p.Key, extra))
	}
	o := regex.Options{IgnoreCase: p.Options&upstream.IgnoreCase != 0}
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
	if err != nil {
		panic(err)
	}
	return m, ok
}

func (f fixed) replace(s, with string) string {
	out, err := f.re.Replace(regex.TextOf(s), with)
	if err != nil {
		panic(err)
	}
	return out
}

// split is .NET's Regex.Split for a pattern with no capturing group.
func (f fixed) split(s string) []string {
	t := regex.TextOf(s)
	all, err := f.re.FindAll(t)
	if err != nil {
		panic(err)
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
