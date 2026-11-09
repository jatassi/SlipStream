package series

import (
	"fmt"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/upstream"
	"github.com/jatassi/slipstream/internal/regex"
)

// fixed is a Sonarr pattern compiled once. Sonarr sets no match timeout
// on its parser patterns and neither does SlipStream, so a call errs only
// on a replacement it can't read, which a test would meet.
type fixed struct{ re *regex.Regex }

const compiledOptions = upstream.IgnoreCase | upstream.Compiled

func compile(p upstream.Pattern) fixed {
	if extra := p.Options &^ compiledOptions; extra != 0 {
		panic(fmt.Sprintf("%s has .NET options %d that the parser doesn't run", p.Key, extra))
	}
	return fixed{regex.MustCompile(p.Text, regex.Options{IgnoreCase: p.Options&upstream.IgnoreCase != 0})}
}

func compileAll(ps []upstream.Pattern) []fixed {
	out := make([]fixed, len(ps))
	for i, p := range ps {
		out[i] = compile(p)
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

func (f fixed) replace(t regex.Text, with string) string {
	out, err := f.re.Replace(t, with)
	if err != nil {
		panic(err)
	}
	return out
}
