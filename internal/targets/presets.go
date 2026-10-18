package targets

import (
	"slices"

	"github.com/jatassi/slipstream/internal/release"
)

// Preset is a way to lay qualities out in tiers. It fills an editor's
// order and is never stored. "No upgrades" is any preset laid out with
// the cutoff at the bottom tier.
type Preset string

const (
	AnythingBetter   Preset = "anything-better"
	DiscOrResolution Preset = "disc-or-resolution"
	ResolutionOnly   Preset = "resolution-only"
)

// Presets is every preset, the default first.
func Presets() []Preset { return []Preset{DiscOrResolution, AnythingBetter, ResolutionOnly} }

func (p Preset) Name() string {
	switch p {
	case AnythingBetter:
		return "Upgrade for anything better"
	case DiscOrResolution:
		return "Upgrade for disc or resolution"
	case ResolutionOnly:
		return "Upgrade for resolution only"
	}
	panic("targets: no preset " + string(p))
}

// Tiers is p over every quality, best first: each resolution from the
// highest, and cinema sources alone at the bottom. Within a resolution,
// anything better ranks each quality alone from the table's highest,
// disc or resolution ties discs above the rest, and resolution only ties
// them all. Qualities within a tier keep the table's order.
func (p Preset) Tiers() [][]release.Quality {
	var tiers [][]release.Quality
	for _, res := range []release.Resolution{release.R2160p, release.R1080p, release.R720p, release.R480p} {
		var discs, rest []release.Quality
		for _, q := range release.Qualities() {
			switch i := q.Info(); {
			case i.Resolution != res:
			case i.Source == release.Bluray || i.Source == release.Remux || i.Source == release.DVD:
				discs = append(discs, q)
			default:
				rest = append(rest, q)
			}
		}
		switch all := append(slices.Clone(rest), discs...); p {
		case AnythingBetter:
			slices.Sort(all)
			for _, q := range slices.Backward(all) {
				tiers = append(tiers, []release.Quality{q})
			}
		case DiscOrResolution:
			tiers = append(tiers, discs, rest)
		case ResolutionOnly:
			slices.Sort(all)
			tiers = append(tiers, all)
		}
	}
	return append(tiers, []release.Quality{release.CAM})
}

// Layout lays chosen out under p: p's tiers kept to chosen, empty tiers
// dropped, every size at its floor. The cutoff is the top tier, or with
// no upgrades the bottom one.
func Layout(p Preset, chosen []release.Quality, upgrades bool) Spec {
	var s Spec
	for _, tier := range p.Tiers() {
		if kept := slices.DeleteFunc(tier, func(q release.Quality) bool { return !slices.Contains(chosen, q) }); len(kept) > 0 {
			s.Tiers = append(s.Tiers, kept)
		}
	}
	switch {
	case len(s.Tiers) == 0:
	case upgrades:
		s.Cutoff = s.Tiers[0][0]
	default:
		s.Cutoff = s.Tiers[len(s.Tiers)-1][0]
	}
	return s
}
