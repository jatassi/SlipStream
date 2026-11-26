package playback

import (
	"cmp"
	"slices"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/targets"
)

// inside is whether every file t accepts the suggestion would accept too.
// A Quality with no resolution counts as above the envelope.
func (e envelope) inside(t targets.Target) bool {
	f := t.Filters
	return e.qualitiesInside(t.Tiers) &&
		f.Video.Mode == targets.Only && !slices.ContainsFunc(f.Video.Values, func(v release.VideoCodec) bool {
		return slices.Contains(e.outside(VideoCodec), string(v))
	}) &&
		covers(f.HDR, e.outside(HDR)) && covers(f.Audio, e.outside(AudioCodec)) && covers(f.Channels, e.outside(Channels))
}

func (e envelope) qualitiesInside(tiers []targets.Tier) bool {
	highest := e[Resolution][0].Rank
	for _, tier := range tiers {
		for _, a := range tier {
			r := a.Quality.Info().Resolution
			if r == 0 || resolutionRank(r) > highest {
				return false
			}
		}
	}
	return true
}

// covers is whether f passes no file holding a value of outside, on a
// fact a file can hold several values of: Not listing every one of them.
func covers[V ~string](f targets.Filter[V], outside []string) bool {
	if len(outside) == 0 {
		return true
	}
	return f.Mode == targets.Not && !slices.ContainsFunc(outside, func(v string) bool { return !slices.Contains(f.Values, V(v)) })
}

// existingIn is the Target inside e that the played file didn't meet when
// it played: one in the Work's Profile first, then by name.
func existingIn(in Input, p Play, e envelope) (targets.Target, bool) {
	goal := in.Siblings.Goals[p.Snapshot.Work]
	var fit []targets.Target
	for _, t := range in.Targets {
		if e.inside(t) && !p.Snapshot.Met(t.ID) {
			fit = append(fit, t)
		}
	}
	if len(fit) == 0 {
		return targets.Target{}, false
	}
	return slices.MinFunc(fit, func(a, b targets.Target) int {
		return cmp.Or(-compareBool(goal.has(a.ID), goal.has(b.ID)), cmp.Compare(a.Name, b.Name))
	}), true
}

func compareBool(a, b bool) int {
	switch {
	case a == b:
		return 0
	case a:
		return 1
	}
	return -1
}

func (g Goal) has(t targets.TargetID) bool { return slices.Contains(g.Targets, t) }

// workLine is what p's Work lacks for t: a file of its Item meets t but
// isn't a version (not_listed, or not_proven naming the first unproven
// fact); else t is in the Profile and no file meets it (no_file); else
// the Profile lacks t (not_in_profile).
func workLine(in Input, p Play, t targets.Target, pr Proofs) WorkLine {
	s := p.Snapshot
	w := in.Works[s.Work]
	line := WorkLine{WorkID: s.Work.String(), ItemID: s.Item.String(), Title: w.Label, ContentType: string(w.ContentType)}
	for _, sib := range in.Siblings.Of(s.Item) {
		if sib.File == p.File || !slices.ContainsFunc(sib.Met, func(r TargetRef) bool { return r.ID == t.ID.String() }) {
			continue
		}
		if !sib.Listed {
			line.State = WorkNotListed
			return line
		}
		if f, ok := pr.firstUnproven(FactsLike(sib.Media, sib.Sidecars, s.Facts).Tracked()); ok {
			line.State, line.Fact = WorkNotProven, &FactValue{Fact: f.Kind, Value: f.Text}
			return line
		}
	}
	goal := in.Siblings.Goals[s.Work]
	if goal.has(t.ID) {
		line.State = WorkNoFile
		return line
	}
	line.State, line.Profile = WorkNotInProfile, &goal.Profile
	return line
}
