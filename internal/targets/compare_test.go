package targets_test

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/targets"
)

func TestStepsLeaveOutRevisionOnlyUnderDontPrefer(t *testing.T) {
	for p, want := range map[targets.Propers][]targets.Step{
		targets.PreferAndUpgrade: {targets.ByTier, targets.ByRevision, targets.ByScore},
		targets.PreferNoUpgrade:  {targets.ByTier, targets.ByRevision, targets.ByScore},
		targets.DontPrefer:       {targets.ByTier, targets.ByScore},
	} {
		target := must[targets.Target](t)(targets.NewTarget(scoredSpec(discsOverWeb, release.Remux1080p, 0, 0, 1, p)))
		if got := target.Steps(); !slices.Equal(got, want) {
			t.Errorf("%s: Steps = %v, want %v", p, got, want)
		}
	}
}

func TestCompareFindsTheFirstStepTwoReleasesDifferOn(t *testing.T) {
	scored := func(q release.Quality, r release.Revision, fs ...formats.ID) targets.Scored {
		return targets.Scored{Facts: release.Facts{Quality: release.Known(q), Revision: r}, Formats: fs}
	}
	v1 := release.Revision{Version: 1}
	proper := release.Revision{Version: 2}
	for name, c := range map[string]struct {
		propers targets.Propers
		a, b    targets.Scored
		step    targets.Step
		better  int
	}{
		"a better tier wins over a better score": {
			targets.PreferAndUpgrade, scored(release.Bluray1080p, v1), scored(release.WEBDL1080p, proper, fa, fb), targets.ByTier, 1,
		},
		"a proper wins across the Qualities of a tier": {
			targets.PreferAndUpgrade, scored(release.WEBRip1080p, v1, fb), scored(release.WEBDL1080p, proper), targets.ByRevision, -1,
		},
		"under don't prefer, the score decides": {
			targets.DontPrefer, scored(release.WEBRip1080p, v1, fb), scored(release.WEBDL1080p, proper), targets.ByScore, 1,
		},
		"a quality the Target doesn't take ranks last": {
			targets.PreferAndUpgrade, scored(release.HDTV720p, proper, fb), scored(release.WEBDL1080p, v1), targets.ByTier, -1,
		},
		"equal at every step": {
			targets.PreferAndUpgrade, scored(release.WEBRip1080p, v1, fa), scored(release.WEBDL1080p, v1, fa), 0, 0,
		},
	} {
		target := must[targets.Target](t)(targets.NewTarget(scoredSpec(discsOverWeb, release.Remux1080p, 0, 0, 1, c.propers)))
		step, better := target.Compare(c.a, c.b)
		if step != c.step || better != c.better {
			t.Errorf("%s: Compare = %d, %d; want %d, %d", name, step, better, c.step, c.better)
		}
		if back, worse := target.Compare(c.b, c.a); back != c.step || worse != -c.better {
			t.Errorf("%s: Compare the other way = %d, %d", name, back, worse)
		}
	}
}

func TestValueSaysEachStepInWords(t *testing.T) {
	target := must[targets.Target](t)(targets.NewTarget(scoredSpec(discsOverWeb, release.Remux1080p, 0, 0, 1, targets.PreferAndUpgrade)))
	a := targets.Scored{
		Facts:   release.Facts{Quality: release.Known(release.Bluray1080p), Revision: release.Revision{Version: 2, Repack: true}},
		Formats: formats.Set{fa, fb},
	}
	reals := targets.Scored{Facts: release.Facts{Quality: release.Known(release.WEBDL1080p), Revision: release.Revision{Version: 1, Real: 2}}}
	for _, c := range []struct {
		step targets.Step
		of   targets.Scored
		want string
	}{
		{targets.ByTier, a, "Bluray-1080p"},
		{targets.ByRevision, a, "v2 REPACK"},
		{targets.ByRevision, reals, "REAL REAL v1"},
		{targets.ByScore, a, "35"},
	} {
		if got := target.Value(c.step, c.of); got != c.want {
			t.Errorf("Value(%d) = %q, want %q", c.step, got, c.want)
		}
	}
}
