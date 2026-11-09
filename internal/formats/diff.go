package formats

import "github.com/jatassi/slipstream/internal/release"

// Versions is one format before and after a change. A nil version is
// absent and matches nothing, as a deleted format's after.
type Versions struct {
	Before, After *Format
}

// Edits is a change's formats before and after, compiled into one
// catalog, so a name is prepared once for them all and a condition
// several versions hold runs once per name. The flip pass runs it over
// the held files of the Works whose Targets score the formats.
type Edits struct {
	c *Catalog
	// versions index c.formats, before then after, -1 for an absent
	// version.
	versions [][2]int
}

// Diff compiles each format's two versions together.
func Diff(vs []Versions, t Timings) Edits {
	e := Edits{c: newCatalog(t), versions: make([][2]int, len(vs))}
	at := func(f *Format) int {
		if f == nil {
			return -1
		}
		e.c.add(*f)
		return len(e.c.formats) - 1
	}
	for i, v := range vs {
		e.versions[i] = [2]int{at(v.Before), at(v.After)}
	}
	return e
}

// Flip is one name's match of a format's two versions.
type Flip struct {
	Before, After bool
	// Timeouts are the conditions of the after version that ran past the
	// limit, or of the before version when there is no after.
	Timeouts []Timeout
}

// Match is rd's Flip of each format, in Diff's order.
func (e Edits) Match(rd release.Read) []Flip {
	r := e.c.start(&rd)
	out := make([]Flip, len(e.versions))
	for i, v := range e.versions {
		reported := v[1]
		if reported < 0 {
			reported = v[0]
		}
		flip := &out[i]
		version := func(at int) bool {
			if at < 0 {
				return false
			}
			f := &e.c.formats[at]
			return r.format(f, false, func(c cond, _, slow bool) {
				if slow && at == reported {
					flip.Timeouts = append(flip.Timeouts, Timeout{Format: f.id, Condition: c.at})
				}
			})
		}
		flip.Before, flip.After = version(v[0]), version(v[1])
	}
	return out
}

// NameOnly reports whether every version tests only a name's text, so a
// Match reads no other fact of the Read.
func (e Edits) NameOnly() bool {
	for _, f := range e.c.formats {
		for _, c := range f.conds {
			if c.Kind != ReleaseTitle {
				return false
			}
		}
	}
	return true
}
