package lint

import (
	"cmp"
	"fmt"
	"io"
	"slices"
)

type Finding struct {
	Path    string
	Line    int
	Rule    string
	Message string
}

func (f Finding) String() string {
	return fmt.Sprintf("%s:%d: [%s] %s", f.Path, f.Line, f.Rule, f.Message)
}

type Rule struct {
	Name  string
	Doc   string
	Check func(*Repo) ([]Finding, error)
}

var registry = map[string]Rule{}

func Register(r Rule) {
	if _, dup := registry[r.Name]; dup {
		panic("lint: duplicate rule " + r.Name)
	}
	registry[r.Name] = r
}

func Rules() []Rule {
	rules := make([]Rule, 0, len(registry))
	for _, r := range registry {
		rules = append(rules, r)
	}
	slices.SortFunc(rules, func(a, b Rule) int { return cmp.Compare(a.Name, b.Name) })
	return rules
}

func Lookup(name string) (Rule, bool) {
	r, ok := registry[name]
	return r, ok
}

func Run(w io.Writer, repo *Repo, rules []Rule) error {
	var findings []Finding
	for _, r := range rules {
		found, err := r.Check(repo)
		if err != nil {
			return fmt.Errorf("rule %s: %w", r.Name, err)
		}
		for _, f := range found {
			f.Rule = r.Name
			findings = append(findings, f)
		}
	}
	slices.SortFunc(findings, func(a, b Finding) int {
		return cmp.Or(
			cmp.Compare(a.Path, b.Path),
			cmp.Compare(a.Line, b.Line),
			cmp.Compare(a.Rule, b.Rule),
			cmp.Compare(a.Message, b.Message),
		)
	})
	for _, f := range findings {
		fmt.Fprintln(w, f)
	}
	if len(findings) > 0 {
		return fmt.Errorf("%d findings", len(findings))
	}
	return nil
}
