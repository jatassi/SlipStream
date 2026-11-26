package playback

import (
	"cmp"
	"slices"

	"github.com/jatassi/slipstream/internal/release"
)

// versionFor is the sibling of p's Item that would have played: listed by
// this server, every tracked fact proven, and on each blocking dimension a
// value that isn't the played file's. A dimension whose table row names
// no fact admits no version.
func versionFor(in Input, p Play, dims []Blocked, pr Proofs) (Version, []Evidence, bool) {
	type candidate struct {
		sib   Sibling
		facts []Fact
	}
	var ok []candidate
	for _, s := range in.Siblings.Of(p.Snapshot.Item) {
		if s.File == p.File || !s.Listed {
			continue
		}
		f := FactsLike(s.Media, s.Sidecars, p.Snapshot.Facts)
		tracked := f.Tracked()
		if _, unproven := pr.firstUnproven(tracked); unproven || !clears(dims, f, p.Snapshot.Facts, pr) {
			continue
		}
		ok = append(ok, candidate{s, tracked})
	}
	if len(ok) == 0 {
		return Version{}, nil, false
	}
	goal := in.Siblings.Goals[p.Snapshot.Work]
	slices.SortStableFunc(ok, func(a, b candidate) int { return goal.compareSiblings(a.sib, b.sib) })
	best := ok[0]
	v := Version{FileID: best.sib.File.String(), Label: best.sib.Label, SeparateItem: best.sib.ItemKey != p.ItemKey}
	return v, pr.evidenceOf(in, best.facts), true
}

func (pr Proofs) firstUnproven(fs []Fact) (Fact, bool) {
	for _, f := range fs {
		if _, ok := pr.proven(f); !ok {
			return f, true
		}
	}
	return Fact{}, false
}

// clears is whether f, a sibling's facts, differs from played on every
// blocking dimension: lower on an ordered one, and on hdr carrying none
// of the formats that blocked.
func clears(dims []Blocked, f, played Facts, pr Proofs) bool {
	for _, d := range dims {
		row := dimensions[order(d.Dimension)]
		if row.Fact == "" {
			return false
		}
		had := ofKind(played.Tracked(), row.Fact)
		if d.Dimension == HDR {
			had = hdrFacts(pr.blockedHDR(played.Video))
		}
		if slices.ContainsFunc(ofKind(f.Tracked(), row.Fact), func(v Fact) bool {
			return slices.ContainsFunc(had, func(b Fact) bool { return v.Text == b.Text || row.Lower && v.Rank >= b.Rank })
		}) {
			return false
		}
	}
	return true
}

func ofKind(fs []Fact, k FactKind) []Fact {
	return slices.DeleteFunc(slices.Clone(fs), func(f Fact) bool { return f.Kind != k })
}

func hdrFacts(hs []release.HDR) []Fact {
	out := make([]Fact, len(hs))
	for i, h := range hs {
		out[i] = Fact{Kind: FactHDR, Text: string(h)}
	}
	return out
}

// compareSiblings orders versions as the Work's Profile ranks the Targets
// they meet, then by label, with Kept files last by label.
func (g Goal) compareSiblings(a, b Sibling) int {
	switch {
	case a.Kept != b.Kept:
		return compareBool(a.Kept, b.Kept)
	case a.Kept:
		return cmp.Compare(a.Label, b.Label)
	}
	return cmp.Or(cmp.Compare(g.rankOf(a), g.rankOf(b)), cmp.Compare(a.Label, b.Label))
}

// rankOf is the place in the Profile of the first Target s meets, past
// the end when it meets none.
func (g Goal) rankOf(s Sibling) int {
	for i, t := range g.Targets {
		if slices.ContainsFunc(s.Met, func(r TargetRef) bool { return r.ID == t.String() }) {
			return i
		}
	}
	return len(g.Targets)
}
