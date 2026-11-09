package naming

import (
	"errors"
	"fmt"
	"slices"
)

// conformBudget is the path budget a type's defaults must name every
// sample within: Windows' 259 without long paths, less a root path of up
// to 59 bytes.
const conformBudget = 200

// Conformance is what's wrong with d as a Content Type declares it, one
// sentence each for the type's author: names and aliases that share a
// key, file variants without a key or sharing one, sample values of a
// kind their token doesn't hold or for a token their tier doesn't have,
// sample variants the type doesn't declare, defaults the check refuses
// and samples that don't name. namingtest.Conform runs it.
func Conformance(d Declaration) []string {
	if _, err := newCatalog(d); err != nil {
		return []string{err.Error()}
	}
	var out []string
	keys := map[string]bool{}
	if len(d.Files) == 0 {
		out = append(out, "the type declares no file template")
	}
	for _, v := range d.Files {
		if v.Key == "" || keys[v.Key] {
			out = append(out, fmt.Sprintf("file variant %q needs a key of its own", v.Label))
		}
		keys[v.Key] = true
	}
	tiered := slices.ContainsFunc(d.Tokens, func(t Token) bool { return t.Scope != WorkScope })
	for _, sample := range d.Samples {
		out = append(out, sampleValues(d, sample.Label, sample.Work, WorkScope)...)
		if tiered && len(sample.Units) == 0 {
			out = append(out, fmt.Sprintf("sample %q has no Unit, and the type's templates read Item and level values", sample.Label))
		}
		for _, u := range sample.Units {
			out = append(out, sampleValues(d, sample.Label, u.Level, LevelScope)...)
			out = append(out, sampleValues(d, sample.Label, u.Item.Set, ItemScope)...)
			if u.Item.Variant != "" && !keys[u.Item.Variant] {
				out = append(out, fmt.Sprintf("sample %q picks file variant %q, which the type doesn't declare", sample.Label, u.Item.Variant))
			}
		}
	}
	s, err := newScheme(d, Settings{})
	var refused *RefusedError
	var problems []Problem
	switch {
	case errors.As(err, &refused):
		problems = refused.Problems
	case err != nil:
		return append(out, err.Error())
	default:
		problems = check(s, conformBudget)
	}
	for _, p := range problems {
		out = append(out, fmt.Sprintf("the default %s: %s", p.Key, p.Sentence))
	}
	return out
}

var kindValues = map[Kind]valueKind{
	KindTitle:  textValue,
	KindText:   textValue,
	KindID:     textValue,
	KindYear:   numberValue,
	KindNumber: numberValue,
	KindDate:   dateValue,
	KindList:   listValue,
}

// sampleValues is what's wrong with set, a sample's values of one tier.
func sampleValues(d Declaration, label string, set Set, scope Scope) []string {
	var out []string
	for name, v := range set {
		i := slices.IndexFunc(d.Tokens, func(t Token) bool { return t.Name == name })
		switch {
		case i < 0 || d.Tokens[i].Scope != scope:
			out = append(out, fmt.Sprintf("sample %q has a value for %q, which isn't a token of its tier", label, name))
		case v.kind != 0 && v.kind != kindValues[d.Tokens[i].Kind]:
			out = append(out, fmt.Sprintf("sample %q has a value for %q of a kind the token doesn't hold", label, name))
		}
	}
	slices.Sort(out)
	return out
}
