package cardigann

import (
	"time"

	netregex "github.com/jatassi/slipstream/internal/regex"
)

// regex is a .NET regular expression, with .NET's replacement syntax.
type regex struct {
	re *netregex.Regex
}

// compileRegex bounds one match to a second, so a pattern that backtracks
// without end fails its field instead of holding the search.
func compileRegex(pattern string) (*regex, error) {
	re, err := netregex.Compile(pattern, netregex.Options{Timeout: time.Second})
	if err != nil {
		return nil, err
	}
	return &regex{re: re}, nil
}

// Group1 is the first group of the first match, empty when nothing
// matches: .NET's Match(s).Groups[1].Value.
func (r *regex) Group1(s string) (string, error) {
	m, ok, err := r.re.Find(netregex.TextOf(s))
	if err != nil || !ok {
		return "", err
	}
	g, _ := m.Group("1")
	return g, nil
}

// Replace replaces every match, reading $1, ${name} and $$ in replacement
// as .NET does.
func (r *regex) Replace(s, replacement string) (string, error) {
	return r.re.Replace(netregex.TextOf(s), replacement)
}

func (r *regex) MatchString(s string) (bool, error) {
	return r.re.Match(netregex.TextOf(s))
}
