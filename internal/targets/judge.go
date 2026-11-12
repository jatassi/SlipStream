package targets

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/release"
)

type ReasonCode string

// A Release's own refusals, which grabs sets before any Target judges it:
// WrongWork is an id naming another Work, WrongItem a name that covers
// none of the searched Items, and WrongTitle one whose title or year
// doesn't match the Work. CutNotWanted is a cell whose Edition doesn't take the Release's
// cut. The rest are Judge's. RepackOtherGroup's value is the held file's
// release group, empty when the file gives none. Blocklisted is a Release
// whose info hash is on the Blocklist.
const (
	WrongWork          ReasonCode = "wrong_work"
	WrongItem          ReasonCode = "wrong_item"
	WrongTitle         ReasonCode = "wrong_title"
	CutNotWanted       ReasonCode = "cut_not_wanted"
	QualityNotInTarget ReasonCode = "quality_not_in_target"
	FilterFailed       ReasonCode = "filter"
	SizeBelow          ReasonCode = "size_below"
	SizeAbove          ReasonCode = "size_above"
	SizeNotChecked     ReasonCode = "size_not_checked"
	ScoreBelowMinimum  ReasonCode = "score_below_minimum"
	NotAnUpgrade       ReasonCode = "not_an_upgrade"
	RepackOtherGroup   ReasonCode = "repack_other_group"
	Blocklisted        ReasonCode = "blocklisted"
)

type weight uint8

type weighed struct {
	code   ReasonCode
	weight weight
}

// block is neither hard nor soft: judging goes on, so a Blocklisted
// Release's verdicts can say what removing it from the Blocklist and
// grabbing it gets, and nothing but that removal lifts it.
const (
	note weight = iota
	soft
	hard
	block
)

// weights is every ReasonCode, in the order the API lists them, and how
// much it weighs.
var weights = []weighed{
	{WrongWork, hard}, {WrongItem, hard}, {WrongTitle, soft}, {CutNotWanted, soft},
	{QualityNotInTarget, hard}, {FilterFailed, hard}, {SizeBelow, soft}, {SizeAbove, soft}, {SizeNotChecked, note},
	{ScoreBelowMinimum, soft}, {NotAnUpgrade, soft}, {RepackOtherGroup, soft}, {Blocklisted, block},
}

// ReasonCodes is every ReasonCode.
func ReasonCodes() []ReasonCode {
	out := make([]ReasonCode, len(weights))
	for i, w := range weights {
		out[i] = w.code
	}
	return out
}

func (c ReasonCode) weight() weight {
	i := slices.IndexFunc(weights, func(w weighed) bool { return w.code == c })
	return weights[i].weight
}

// Hard is whether the reason refuses a Release outright: a file it names
// never fills the cell, and nothing grabs it. A Release of another Work
// or another Item, a quality the Target doesn't accept and a failed
// filter are hard.
func (c ReasonCode) Hard() bool { return c.weight() == hard }

// Soft is whether the reason keeps only an automatic grab from taking a
// Release: a Manual search may grab past it. size_not_checked is a note,
// neither hard nor soft.
func (c ReasonCode) Soft() bool { return c.weight() == soft }

// Reason is one failure or note. Attribute and Value name the filter
// that failed, Value the quality a Target doesn't accept, and Value the
// score below the minimum.
type Reason struct {
	Code      ReasonCode
	Attribute Attribute
	Value     string
}

// Scored is a release or file with the Custom Formats it matches.
type Scored struct {
	Facts   release.Facts
	Formats formats.Set
}

// Held is the file filling a cell and its score at the cell's Target.
type Held struct {
	Facts release.Facts
	Score int
}

// Verdict is how a release fares against one Target.
type Verdict struct {
	rank    Rank
	ranked  bool
	score   int
	rule    int
	upgrade bool
	reasons []Reason
}

// Meets reports no hard failure: the quality is in a tier and every
// filter passes. A file fills a cell only when it meets the cell's Target.
func (v Verdict) Meets() bool {
	return !slices.ContainsFunc(v.reasons, func(r Reason) bool { return r.Code.Hard() })
}

// Grabbable reports no failure at all, only notes: what an automatic grab
// needs.
func (v Verdict) Grabbable() bool {
	return !slices.ContainsFunc(v.reasons, func(r Reason) bool { return r.Code != SizeNotChecked })
}

// Rank is the tier the quality sits in, false when the Target doesn't
// accept it.
func (v Verdict) Rank() (Rank, bool) { return v.rank, v.ranked }

// Score is the release's score at the Target.
func (v Verdict) Score() int { return v.score }

// Upgrade is the number of the upgrade rule that decided, as Judge lists
// them, and whether the release upgrades the cell. The rule is 0 when the
// Target doesn't accept the release's quality.
func (v Verdict) Upgrade() (rule int, ok bool) { return v.rule, v.upgrade }

// Reasons lists every failure and note: quality, then filters in
// attribute order, then size, then score, then upgrade.
func (v Verdict) Reasons() []Reason { return v.reasons }

// Judge is the one judgment of a release against a Target, for search,
// import and cells. held is the file filling the cell, nil when none
// does. For a release whose quality the Target accepts, the first of
// these upgrade rules that applies decides:
//
//  1. No file, or a file of unknown or unranked quality: an upgrade.
//  2. The same Quality and a higher revision, under PreferAndUpgrade: an
//     upgrade even past the cutoff. A repack from another release group
//     than the file's, or with either group unknown, is
//     repack_other_group.
//  3. A better tier while the file's is below the cutoff: an upgrade.
//  4. A better tier while the file's is at or past the cutoff:
//     not_an_upgrade.
//  5. A worse tier: not_an_upgrade.
//  6. The same Quality and a lower revision: not_an_upgrade unless
//     DontPrefer.
//  7. The file's score is at or above upgrade-until: not_an_upgrade.
//  8. The same tier: an upgrade when the release's score is at least the
//     file's plus the increment, else not_an_upgrade.
func Judge(t Target, s Scored, held *Held) Verdict {
	f := s.Facts
	v := Verdict{score: t.Score(&s)}
	var accepted Accepted
	var ok bool
	q, known := f.Quality.Get()
	if known {
		accepted, v.rank, ok = t.accepted(q)
	}
	v.ranked = ok
	if !ok {
		r := Reason{Code: QualityNotInTarget}
		if known {
			r.Value = q.String()
		}
		v.reasons = append(v.reasons, r)
	}
	v.reasons = append(v.reasons, t.Filters.judge(f)...)
	if r, noted := accepted.Size.judge(f.Size, f.Coverage.Runtime); ok && noted {
		v.reasons = append(v.reasons, r)
	}
	if v.score < t.Scores.Minimum {
		v.reasons = append(v.reasons, Reason{Code: ScoreBelowMinimum, Value: strconv.Itoa(v.score)})
	}
	if !ok {
		return v
	}
	m := moveOf(t, v.rank, s, v.score, held)
	for _, r := range upgradeRules {
		if r.applies(m) {
			v.rule, v.upgrade = r.rule, r.refusal == ""
			if !v.upgrade {
				refused := Reason{Code: r.refusal}
				if r.refusal == RepackOtherGroup {
					refused.Value = m.heldGroup
				}
				v.reasons = append(v.reasons, refused)
			}
			break
		}
	}
	return v
}

// move is a ranked release against the file filling its cell, as the
// upgrade rules read it.
type move struct {
	vacant bool // no file, or one of unknown or unranked quality
	// tier is +1 when the release's tier is better than the file's and
	// -1 when it is worse. revision compares the two within one Quality,
	// and is 0 across Qualities.
	tier, revision int
	heldFinal      bool // the file's tier is at or past the cutoff
	repack         bool
	sameGroup      bool // both groups are known and the same
	heldGroup      string
	propers        Propers
	score, held    int
	upgradeUntil   int
	increment      int
}

func moveOf(t Target, r Rank, s Scored, score int, held *Held) move {
	m := move{vacant: true, propers: t.Propers, score: score, upgradeUntil: t.Scores.UpgradeUntil, increment: t.Scores.Increment}
	if held == nil {
		return m
	}
	hq, known := held.Facts.Quality.Get()
	if !known {
		return m
	}
	h, ranked := t.RankOf(hq)
	if !ranked {
		return m
	}
	m.vacant = false
	m.tier = cmp.Compare(h, r)
	m.heldFinal = int(h) <= t.Cutoff
	if q, _ := s.Facts.Quality.Get(); q == hq {
		m.revision = s.Facts.Revision.Compare(held.Facts.Revision)
	}
	m.repack = s.Facts.Revision.Repack
	group, gok := s.Facts.Group.Get()
	heldGroup, hok := held.Facts.Group.Get()
	m.sameGroup = gok && hok && strings.EqualFold(group, heldGroup)
	m.heldGroup = heldGroup
	m.held = held.Score
	return m
}

// upgradeRules are Judge's upgrade rules in order. A rule that refuses
// names its reason, and the first rule that applies decides.
var upgradeRules = []struct {
	rule    int
	applies func(move) bool
	refusal ReasonCode
}{
	{1, func(m move) bool { return m.vacant }, ""},
	{2, func(m move) bool {
		return m.revision > 0 && m.propers == PreferAndUpgrade && m.repack && !m.sameGroup
	}, RepackOtherGroup},
	{2, func(m move) bool { return m.revision > 0 && m.propers == PreferAndUpgrade }, ""},
	{3, func(m move) bool { return m.tier > 0 && !m.heldFinal }, ""},
	{4, func(m move) bool { return m.tier > 0 }, NotAnUpgrade},
	{5, func(m move) bool { return m.tier < 0 }, NotAnUpgrade},
	{6, func(m move) bool { return m.revision < 0 && m.propers != DontPrefer }, NotAnUpgrade},
	{7, func(m move) bool { return m.held >= m.upgradeUntil }, NotAnUpgrade},
	{8, func(m move) bool { return m.score >= m.held+m.increment }, ""},
	{8, func(move) bool { return true }, NotAnUpgrade},
}

func (s Size) judge(size release.Fact[int64], runtime release.Fact[time.Duration]) (Reason, bool) {
	bytes, sized := size.Get()
	length, timed := runtime.Get()
	if !sized || !timed || length <= 0 {
		return Reason{Code: SizeNotChecked}, true
	}
	rate := float64(bytes) / (1 << 20) / length.Minutes()
	switch {
	case rate < s.Min:
		return Reason{Code: SizeBelow}, true
	case s.Max != nil && rate > *s.Max:
		return Reason{Code: SizeAbove}, true
	}
	return Reason{}, false
}

func (f *Filters) judge(facts release.Facts) []Reason {
	checks := map[Attribute]func() (string, bool){
		HDR:        func() (string, bool) { return passes(f.HDR, facts.HDR, listedIn(f.HDR)) },
		VideoCodec: func() (string, bool) { return passes(f.Video, single(facts.Video), listedIn(f.Video)) },
		AudioCodec: func() (string, bool) { return passes(f.Audio, facts.Audio, listedIn(f.Audio)) },
		Channels:   func() (string, bool) { return passes(f.Channels, facts.Channels, listedIn(f.Channels)) },
		Language:   func() (string, bool) { return languages(f.Language, facts) },
	}
	var out []Reason
	for _, a := range Attributes() {
		if failed, ok := checks[a](); !ok {
			out = append(out, Reason{Code: FilterFailed, Attribute: a, Value: failed})
		}
	}
	return out
}

// languages judges the language filter, where original names the Work's
// original language. While that is unknown, a filter naming original
// passes.
func languages(f Filter[release.Language], facts release.Facts) (string, bool) {
	listed := listedIn(f)
	if slices.Contains(f.Values, release.Original) {
		original, known := facts.Coverage.Original.Get()
		if !known {
			return "", true
		}
		listed = func(l release.Language) bool { return l == original || slices.Contains(f.Values, l) }
	}
	return passes(f, facts.Languages, listed)
}

func listedIn[V ~string](f Filter[V]) func(V) bool {
	return func(v V) bool { return slices.Contains(f.Values, v) }
}

func single[V any](f release.Fact[V]) release.Fact[[]V] {
	if v, known := f.Get(); known {
		return release.Known([]V{v})
	}
	return release.Fact[[]V]{}
}

// passes judges a fact against a filter whose set listed answers, and
// names the value that failed it. An unknown fact passes.
func passes[V ~string](f Filter[V], fact release.Fact[[]V], listed func(V) bool) (string, bool) {
	present, known := fact.Get()
	if !known {
		return "", true
	}
	switch f.Mode {
	case Any:
	case Only:
		if slices.ContainsFunc(present, listed) {
			return "", true
		}
		if len(present) == 0 {
			return "", false
		}
		return string(present[0]), false
	case Not:
		if i := slices.IndexFunc(present, listed); i >= 0 {
			return string(present[i]), false
		}
	}
	return "", true
}
