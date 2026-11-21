package subtitles

import (
	"cmp"
	"path"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/subtitling"
	"github.com/jatassi/slipstream/internal/targets"
)

// floor is why a candidate can't meet a want at all, or floorOK.
type floor uint8

const (
	floorOK floor = iota
	wrongLanguage
	wrongKind
	wrongEpisode
	imageOnly
)

// ranking is what ranking reads of a want and its file: the Wanted
// Language, what the file is searched as, the Release group, source and
// resolution its name reads to, and the Admin's provider order.
type ranking struct {
	want    targets.WantedLanguage
	subject subtitling.Subject
	group   string
	file    release.Facts
	order   []subtitling.ProviderID
}

// ranked is one offer with the floor it fails and the ranking steps that
// hold for it.
type ranked struct {
	offer subtitling.Offer
	floor floor
	// person: made by a person, not machine or AI translated. hi: the
	// hearing-impaired preference holds. group and source: a release it
	// was made for has the file's Release group, or its source and
	// resolution.
	person, hi, group, source bool
	// place is its provider's place in the order, 0 first.
	place int
}

func rankingOf(w targets.WantedLanguage, subject subtitling.Subject, video, group string, order []subtitling.ProviderID) ranking {
	return ranking{want: w, subject: subject, group: group, file: release.Title(path.Base(video), release.TitleOnly{}, nil).Facts, order: order}
}

// of is o's floor and steps.
func (r ranking) of(o subtitling.Offer) ranked {
	x := ranked{offer: o, person: !o.AI, place: slices.Index(r.order, o.Provider)}
	if x.place < 0 {
		x.place = len(r.order)
	}
	switch r.want.HI {
	case targets.Avoid:
		x.hi = !o.HI
	case targets.Prefer:
		x.hi = o.HI
	case targets.Allow:
		x.hi = true
	}
	for _, name := range o.Releases {
		f := release.Title(name, release.TitleOnly{}, nil).Facts
		if g, ok := f.Group.Get(); ok && r.group != "" && strings.EqualFold(g, r.group) {
			x.group = true
		}
		if sameSource(f, r.file) {
			x.source = true
		}
	}
	switch {
	case o.Language != r.want.Language:
		x.floor = wrongLanguage
	case o.Forced != r.want.Forced:
		x.floor = wrongKind
	case !r.covers(o):
		x.floor = wrongEpisode
	case o.Image:
		x.floor = imageOnly
	}
	return x
}

func sameSource(a, b release.Facts) bool {
	as, aok := a.Source.Get()
	bs, bok := b.Source.Get()
	ar, rok := a.Resolution.Get()
	br, brok := b.Resolution.Get()
	return aok && bok && rok && brok && as == bs && ar == br
}

// covers is whether o can hold the file's episodes: a feature, an offer
// that names no season, or one of the file's season that names every
// episode or is a pack.
func (r ranking) covers(o subtitling.Offer) bool {
	if r.subject.Kind != subtitling.Episode || o.Season == 0 {
		return true
	}
	if o.Season != r.subject.Season {
		return false
	}
	if o.Pack || len(o.Episodes) == 0 {
		return true
	}
	for _, e := range r.subject.Episodes {
		if !slices.Contains(o.Episodes, e) {
			return false
		}
	}
	return true
}

// rank is offers ranked for the want, best first: each step breaks the
// last one's ties. Floored offers are left out, except image ones when
// images is set, which the Manual subtitle search sheet shows last.
func (r ranking) rank(offers []subtitling.Offer, images bool) []ranked {
	out := make([]ranked, 0, len(offers))
	for _, o := range offers {
		x := r.of(o)
		if x.floor == floorOK || images && x.floor == imageOnly {
			out = append(out, x)
		}
	}
	slices.SortStableFunc(out, func(a, b ranked) int {
		return cmp.Or(cmp.Compare(a.floor, b.floor), compareSteps(a, b), cmp.Compare(b.offer.Downloads, a.offer.Downloads), cmp.Compare(b.offer.Rating, a.offer.Rating))
	})
	return out
}

// compareSteps orders a and b by steps 1 to 6: made by a person, the
// hearing-impaired preference, Release group, source and resolution,
// then provider order. Moviehash matching has no source yet.
func compareSteps(a, b ranked) int {
	return cmp.Or(ahead(a.person, b.person), ahead(a.hi, b.hi), ahead(a.group, b.group), ahead(a.source, b.source), cmp.Compare(a.place, b.place))
}

func ahead(a, b bool) int {
	switch {
	case a == b:
		return 0
	case a:
		return -1
	}
	return 1
}

// outranks is whether a outranks b strictly, which an upgrade needs. A
// candidate's downloads and rating never make one: they would replace a
// passing subtitle with an equal one.
func outranks(a, b ranked) bool { return compareSteps(a, b) < 0 }

// current is sc ranked as the candidate it was. A Sidecar with no offer
// ranks under every candidate.
func (r ranking) current(sc *Sidecar) (ranked, bool) {
	if sc == nil || sc.Offer == nil {
		return ranked{}, false
	}
	o := subtitling.Offer{Provider: sc.Offer.Provider, Key: sc.Offer.Key, Language: sc.Tail.Language, Forced: sc.Tail.Forced, HI: sc.Tail.HI, AI: sc.Offer.AI}
	if sc.Offer.Release != "" {
		o.Releases = []string{sc.Offer.Release}
	}
	return r.of(o), true
}

// reasons is the ranking steps x meets, as the sheet checks them. Its
// provider order holds when no candidate's provider is higher.
func reasons(x ranked, top int) []Reason {
	var out []Reason
	for _, r := range []struct {
		holds bool
		is    Reason
	}{{x.person, MadeByPerson}, {x.group, SameGroup}, {x.source, SameSource}, {x.place == top, ProviderOrder}} {
		if r.holds {
			out = append(out, r.is)
		}
	}
	return out
}
