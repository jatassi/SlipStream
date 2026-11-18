package subtitles

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/subtitling"
	"github.com/jatassi/slipstream/internal/targets"
)

func TestRankingRunsItsStepsInOrderAndAvoidedHearingImpairedIsNeverFinal(t *testing.T) {
	en, err := language.Parse("en")
	if err != nil {
		t.Fatal(err)
	}
	fr, err := language.Parse("fr")
	if err != nil {
		t.Fatal(err)
	}
	made := []string{"Film.2024.1080p.WEB-DL-GRP"}
	o := func(key string, edit func(*subtitling.Offer)) subtitling.Offer {
		x := subtitling.Offer{Provider: "subdl", Key: key, Language: en, Releases: made, Downloads: 10}
		if edit != nil {
			edit(&x)
		}
		return x
	}
	offers := []subtitling.Offer{
		o("machine", func(x *subtitling.Offer) { x.AI, x.Downloads = true, 900 }),
		o("hi", func(x *subtitling.Offer) { x.HI, x.Downloads = true, 800 }),
		o("other-group", func(x *subtitling.Offer) { x.Releases, x.Downloads = []string{"Film.2024.1080p.WEB-DL-OTHER"}, 700 }),
		o("other-source", func(x *subtitling.Offer) { x.Releases, x.Downloads = []string{"Film.2024.720p.HDTV-GRP"}, 600 }),
		o("second-provider", func(x *subtitling.Offer) { x.Provider, x.Downloads = "gestdown", 500 }),
		o("fewer-downloads", func(x *subtitling.Offer) { x.Downloads, x.Rating = 5, 9 }),
		o("lower-rating", func(x *subtitling.Offer) { x.Downloads, x.Rating = 5, 1 }),
		o("best", nil),
		o("french", func(x *subtitling.Offer) { x.Language = fr }),
		o("forced", func(x *subtitling.Offer) { x.Forced = true }),
		o("image", func(x *subtitling.Offer) { x.Image = true }),
	}
	want := targets.WantedLanguage{Language: en, HI: targets.Avoid}
	r := rankingOf(want, subtitling.Subject{Kind: subtitling.Feature}, "Film/Film.2024.1080p.WEB-DL-GRP.mkv", "GRP", []subtitling.ProviderID{"subdl", "gestdown"})
	keys := func(rs []ranked) []string {
		var out []string
		for _, x := range rs {
			out = append(out, x.offer.Key)
		}
		return out
	}
	order := []string{"best", "fewer-downloads", "lower-rating", "second-provider", "other-source", "other-group", "hi", "machine"}
	if got := keys(r.rank(offers, false)); !slices.Equal(got, order) {
		t.Fatalf("the ranking is %q, want %q", got, order)
	}
	if got := keys(r.rank(offers, true)); !slices.Equal(got, append(slices.Clone(order), "image")) {
		t.Fatalf("with images the ranking is %q, want the image one last", got)
	}
	for _, c := range []struct {
		key   string
		hi    targets.HI
		final bool
	}{
		{"best", targets.Avoid, true},
		{"hi", targets.Avoid, false},
		{"hi", targets.Prefer, true},
		{"machine", targets.Allow, false},
		{"other-group", targets.Allow, false},
	} {
		i := slices.IndexFunc(offers, func(x subtitling.Offer) bool { return x.Key == c.key })
		if got := finalOf(offers[i], "GRP", c.hi); got != c.final {
			t.Errorf("%s under %v is final %v, want %v", c.key, c.hi, got, c.final)
		}
	}
	best, other := r.of(offers[7]), r.of(offers[5])
	if outranks(other, best) || outranks(best, other) {
		t.Fatal("downloads and rating made an upgrade")
	}
	if !outranks(best, r.of(offers[4])) {
		t.Fatal("a higher provider made no upgrade")
	}
}

func TestOnlyReadyKindsAreRead(t *testing.T) {
	subdl := Provider{Spec: subtitling.Spec{ID: "subdl", Kinds: []subtitling.Kind{subtitling.Feature, subtitling.Episode}, Keyed: true}, On: true, HasKey: true}
	gestdown := Provider{Spec: subtitling.Spec{ID: "gestdown", Kinds: []subtitling.Kind{subtitling.Episode}}, On: true}
	now := subdl.KeyCheckedAt
	if got := readyKinds([]Provider{subdl, gestdown}, now); !slices.Equal(got, []subtitling.Kind{subtitling.Feature, subtitling.Episode}) {
		t.Fatalf("with both ready the kinds are %v", got)
	}
	subdl.Paused = &Pause{Until: now.Add(1), Reason: PausedQuota}
	if got := readyKinds([]Provider{subdl, gestdown}, now); !slices.Equal(got, []subtitling.Kind{subtitling.Episode}) {
		t.Fatalf("with SubDL out of downloads the kinds are %v, want episodes alone", got)
	}
}
