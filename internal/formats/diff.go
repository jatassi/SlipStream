package formats

import "github.com/jatassi/slipstream/internal/release"

// Edit is one format before and after a change, compiled together so a
// condition both versions hold runs once per name. The flip pass runs it
// over the held files of the Works whose Targets score the format.
type Edit struct {
	c *Catalog
	// before and after index c.formats, -1 for an absent version.
	before, after int
}

// Diff compiles a format's two versions. A nil version is absent and
// matches nothing, as a deleted format's after.
func Diff(before, after *Format, t Timings) Edit {
	c := newCatalog(t)
	e := Edit{c: c, before: -1, after: -1}
	if before != nil {
		c.add(*before)
		e.before = len(c.formats) - 1
	}
	if after != nil {
		c.add(*after)
		e.after = len(c.formats) - 1
	}
	return e
}

// Flip is one name's match of an Edit's two versions.
type Flip struct {
	Before, After bool
	// Timeouts are the conditions of the after version that ran past the
	// limit, or of the before version when there is no after.
	Timeouts []Timeout
}

func (e Edit) Match(rd release.Read) Flip {
	r := e.c.start(&rd)
	reported := e.after
	if reported < 0 {
		reported = e.before
	}
	var flip Flip
	version := func(i int) bool {
		if i < 0 {
			return false
		}
		f := &e.c.formats[i]
		return r.format(f, false, func(c cond, _, slow bool) {
			if slow && i == reported {
				flip.Timeouts = append(flip.Timeouts, Timeout{Format: f.id, Condition: c.at})
			}
		})
	}
	flip.Before, flip.After = version(e.before), version(e.after)
	return flip
}

// NameOnly reports whether both versions test only a name's text, so a
// Match reads no other fact of the Read.
func (e Edit) NameOnly() bool {
	for _, f := range e.c.formats {
		for _, c := range f.conds {
			if c.Kind != ReleaseTitle {
				return false
			}
		}
	}
	return true
}
