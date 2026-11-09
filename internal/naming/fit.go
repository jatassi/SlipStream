package naming

import (
	"slices"
	"strings"
)

// maxSegment is the bytes one folder or file name may take on NTFS,
// APFS, ext4 and the rest.
const maxSegment = 255

// fit names p's file from b after folders, cutting b's Item-scope titles,
// the longest first, at a whole character with "...", until every segment
// is at most maxSegment bytes and the path at most p.Budget. Nothing else
// shrinks: not {Target}, the Edition tag, the episode block or the
// extension. What still doesn't fit is a *TooLongError.
func (s *Scheme) fit(b bound, p Placing, folders []string, anchor string) (Named, error) {
	st := s.style()
	var cuts []shrink
	for at, part := range b.parts {
		if r, ok := part.(*boundRef); ok && slices.Contains(b.shrinkable, at) {
			cuts = append(cuts, shrink{at: at, ref: r, bytes: len(form(r, valueOf(r.entry, p.Subject, 0, p.File), p.File))})
		}
	}
	cut := b
	for {
		name := segment(st.fileText(cut, p.Subject, p.File, anchor) + p.Ext)
		named, err := fits(folders, name, p.Budget)
		if err == nil {
			return named, nil
		}
		longest := 0
		for i := range cuts {
			if cuts[i].bytes > cuts[longest].bytes {
				longest = i
			}
		}
		over := max(len(name)-maxSegment, len(named.Rel)-p.Budget)
		if over <= 0 || len(cuts) == 0 || cuts[longest].bytes == 0 {
			return Named{}, err
		}
		cuts[longest].bytes = max(0, cuts[longest].bytes-over)
		cut = withCuts(b, cuts)
	}
}

// shrink is a shrinkable ref at part at and the bytes fit cuts it to.
type shrink struct {
	at    int
	ref   *boundRef
	bytes int
}

// withCuts is b with each ref of cuts cut to its bytes, from the end when
// its own cut is.
func withCuts(b bound, cuts []shrink) bound {
	b.parts = slices.Clone(b.parts)
	for _, sh := range cuts {
		r := *sh.ref
		c := cut{bytes: sh.bytes}
		if own, ok := r.mod.(cut); ok {
			c = cut{bytes: min(sh.bytes, own.bytes), fromEnd: own.fromEnd}
		}
		r.mod = c
		b.parts[sh.at] = &r
	}
	return b
}

// fits joins folders and file into a Named, or a *TooLongError for the
// first segment over maxSegment bytes or a path over budget.
func fits(folders []string, file string, budget int) (Named, error) {
	segs := append(append([]string(nil), folders...), file)
	named := Named{Rel: strings.Join(segs, "/"), Segments: segs}
	for _, seg := range segs {
		if len(seg) > maxSegment {
			return named, &TooLongError{Segment: seg, Bytes: len(seg), Limit: maxSegment}
		}
	}
	if len(named.Rel) > budget {
		return named, &TooLongError{Bytes: len(named.Rel), Limit: budget}
	}
	return named, nil
}
