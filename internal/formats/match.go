package formats

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/release"
)

// Catalog is a compiled set of formats, immutable and safe to share. It
// holds each distinct test once, so a condition several formats share
// runs once per name.
type Catalog struct {
	formats []compiled
	tests   []test
	index   map[test]int
	timings Timings
}

type compiled struct {
	id   ID
	name string
	// err is why the format doesn't compile. Its groups are then nil.
	err error
	// groups are its conditions grouped by kind, cheapest kind first,
	// each group's Required conditions first.
	groups [][]cond
	conds  []Condition
}

type cond struct {
	at       int
	test     int
	negate   bool
	required bool
}

// Check compiles a format with the Match limit, and names the first
// condition that doesn't compile: a *PatternError, a *KindError, a
// *FieldError for a field its kind lacks, or a *ValueError for a value
// its kind can't read in the format's dialect. A format with no
// conditions is an *EmptyError.
func Check(f Format, t Timings) error {
	if len(f.Conditions) == 0 {
		return &EmptyError{Name: f.Name}
	}
	c := newCatalog(t)
	c.add(f)
	return c.formats[0].err
}

// Compile compiles formats with the Match limit, in name order. A format
// that no longer compiles matches nothing and is listed by Broken.
func Compile(fs []Format, t Timings) *Catalog {
	c := newCatalog(t)
	for _, f := range fs {
		c.add(f)
	}
	slices.SortStableFunc(c.formats, func(a, b compiled) int { return strings.Compare(a.name, b.name) })
	return c
}

func newCatalog(t Timings) *Catalog {
	return &Catalog{index: map[test]int{}, timings: t}
}

func (c *Catalog) add(f Format) {
	groups, err := c.compile(f)
	c.formats = append(c.formats, compiled{id: f.ID, name: f.Name, err: err, groups: groups, conds: f.Conditions})
}

func (c *Catalog) compile(f Format) ([][]cond, error) {
	d := dialects[f.Dialect]
	byKind := map[Kind][]cond{}
	for i := range f.Conditions {
		fc := &f.Conditions[i]
		t, err := c.compileTest(i, fc, d)
		if err != nil {
			return nil, err
		}
		byKind[fc.Kind] = append(byKind[fc.Kind], cond{at: i, test: c.intern(t), negate: fc.Negate, required: fc.Required})
	}
	groups := make([][]cond, 0, len(byKind))
	for _, k := range slices.SortedFunc(maps.Keys(byKind), byCost) {
		g := byKind[k]
		slices.SortStableFunc(g, func(a, b cond) int { return boolOrder(b.required) - boolOrder(a.required) })
		groups = append(groups, g)
	}
	return groups, nil
}

func (c *Catalog) compileTest(at int, fc *Condition, d *dialect) (test, error) {
	k, ok := kindOf(fc.Kind)
	if !ok {
		return nil, &KindError{Name: fc.Name, Kind: string(fc.Kind)}
	}
	by, err := k.fieldsOf(fc)
	if err != nil {
		return nil, err
	}
	return k.compile(fields{at: at, c: fc, by: by, dialect: d, timeout: c.timings.Match})
}

// fieldsOf is a condition's fields by name, refusing a name its kind
// lacks.
func (k *kind) fieldsOf(fc *Condition) (map[string]json.RawMessage, error) {
	by := map[string]json.RawMessage{}
	if len(fc.Fields) > 0 {
		if err := json.Unmarshal(fc.Fields, &by); err != nil {
			return nil, &ValueError{Name: fc.Name, Kind: fc.Kind, Field: "fields", Value: string(fc.Fields), Want: "an object"}
		}
	}
	for _, name := range slices.Sorted(maps.Keys(by)) {
		if !slices.Contains(k.Fields, name) {
			return nil, &FieldError{Name: fc.Name, Kind: fc.Kind, Field: name}
		}
	}
	return by, nil
}

func (c *Catalog) intern(t test) int {
	if i, ok := c.index[t]; ok {
		return i
	}
	c.tests = append(c.tests, t)
	c.index[t] = len(c.tests) - 1
	return len(c.tests) - 1
}

func byCost(a, b Kind) int {
	ka, _ := kindOf(a)
	kb, _ := kindOf(b)
	if ka.cost != kb.cost {
		return ka.cost - kb.cost
	}
	return strings.Compare(string(a), string(b))
}

func boolOrder(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Only is the catalog's formats the Set names.
func (c *Catalog) Only(s Set) *Catalog {
	out := *c
	out.formats = slices.DeleteFunc(slices.Clone(c.formats), func(f compiled) bool {
		_, found := slices.BinarySearchFunc(s, f.id, compareID)
		return !found
	})
	return &out
}

// With is the catalog with a format added, replacing the one of its ID:
// the tester's draft.
func (c *Catalog) With(f Format) *Catalog {
	out := &Catalog{
		formats: slices.DeleteFunc(slices.Clone(c.formats), func(have compiled) bool { return have.id == f.ID }),
		tests:   slices.Clone(c.tests),
		index:   maps.Clone(c.index),
		timings: c.timings,
	}
	out.add(f)
	slices.SortStableFunc(out.formats, func(a, b compiled) int { return strings.Compare(a.name, b.name) })
	return out
}

func (c *Catalog) Broken() []ID {
	var ids []ID
	for _, f := range c.formats {
		if f.err != nil {
			ids = append(ids, f.id)
		}
	}
	return ids
}

func (c *Catalog) Len() int { return len(c.formats) }

func compareID(a, b ID) int { return bytes.Compare(a[:], b[:]) }

type outcome uint8

const (
	unrun outcome = iota
	failed
	matched
	timedOut
)

type run struct {
	c    *Catalog
	in   *input
	memo []outcome
}

func (c *Catalog) start(r *release.Read) *run {
	return &run{c: c, in: newInput(r), memo: make([]outcome, len(c.tests))}
}

func (r *run) test(i int) outcome {
	if r.memo[i] == unrun {
		ok, slow := r.c.tests[i].run(r.in)
		switch {
		case slow:
			r.memo[i] = timedOut
		case ok:
			r.memo[i] = matched
		default:
			r.memo[i] = failed
		}
	}
	return r.memo[i]
}

// cond is one condition after Negate. A condition that timed out is not
// matched, negated or not.
func (r *run) cond(c cond) (ok, slow bool) {
	o := r.test(c.test)
	if o == timedOut {
		return false, true
	}
	return (o == matched) != c.negate, false
}

// format is upstream's rule: each group matches when no Required
// condition in it failed and one matched, and the format when every group
// does. With all unset it stops at the first group that fails, and a
// group once its outcome is settled. seen hears every condition it ran.
func (r *run) format(f *compiled, all bool, seen func(c cond, ok, slow bool)) bool {
	if f.err != nil {
		return false
	}
	matches := true
	for _, g := range f.groups {
		some, failedRequired := false, false
		for _, c := range g {
			if !all && (failedRequired || some && !c.required) {
				break
			}
			ok, slow := r.cond(c)
			seen(c, ok, slow)
			some = some || ok
			failedRequired = failedRequired || c.required && !ok
		}
		if !some || failedRequired {
			matches = false
			if !all {
				return false
			}
		}
	}
	return matches
}

// Match is every format a Read matches, by upstream's rule: each condition
// matches, then Negate flips it; conditions group by kind; a group matches
// when no Required condition in it failed and one matched; a format
// matches when every group does. An unknown language reads as the Read's
// Facts.Coverage.Original. A timed-out condition counts as not matched.
// Match stops a format at its first failed group, cheap kinds first.
func (c *Catalog) Match(rd release.Read) Result {
	r := c.start(&rd)
	var res Result
	for i := range c.formats {
		f := &c.formats[i]
		ok := r.format(f, false, func(cd cond, _, slow bool) {
			if slow {
				res.Timeouts = append(res.Timeouts, Timeout{Format: f.id, Condition: cd.at})
			}
		})
		if ok {
			res.Matched = append(res.Matched, f.id)
		}
	}
	slices.SortFunc(res.Matched, compareID)
	return res
}

// Explain is Match with every condition evaluated, for the tester.
func (c *Catalog) Explain(rd release.Read) []Explained {
	r := c.start(&rd)
	out := make([]Explained, len(c.formats))
	for i := range c.formats {
		f := &c.formats[i]
		e := Explained{Format: f.id, Name: f.name, Conditions: make([]Evaluated, len(f.conds))}
		for j, fc := range f.conds {
			e.Conditions[j] = Evaluated{Name: fc.Name, Kind: fc.Kind, Negate: fc.Negate, Required: fc.Required}
		}
		e.Matched = r.format(f, true, func(cd cond, ok, slow bool) {
			e.Conditions[cd.at].Matched, e.Conditions[cd.at].TimedOut = ok, slow
			e.TimedOut = e.TimedOut || slow
		})
		out[i] = e
	}
	return out
}
