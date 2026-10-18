package targets

import (
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/release"
)

type ReasonCode string

const (
	QualityNotInTarget ReasonCode = "quality_not_in_target"
	FilterFailed       ReasonCode = "filter"
	SizeBelow          ReasonCode = "size_below"
	SizeAbove          ReasonCode = "size_above"
	SizeNotChecked     ReasonCode = "size_not_checked"
	NotAnUpgrade       ReasonCode = "not_an_upgrade"
)

// Hard is whether the reason keeps a file from filling a cell. Size and
// upgrade reasons only keep an automatic grab from taking a release, and
// size_not_checked is a note.
func (c ReasonCode) Hard() bool { return c == QualityNotInTarget || c == FilterFailed }

// Reason is one failure or note. Attribute and Value name the filter
// that failed, and Value the quality a Target doesn't accept.
type Reason struct {
	Code      ReasonCode
	Attribute Attribute
	Value     string
}

// Verdict is how facts fare against one Target.
type Verdict struct {
	rank    Rank
	ranked  bool
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

// Reasons lists every failure and note: quality, then filters in
// attribute order, then size, then upgrade.
func (v Verdict) Reasons() []Reason { return v.reasons }

// Judge is the one judgment of facts against a Target, for search,
// import and cells. held is the Facts of the file filling the cell, nil
// when none does: a release that doesn't rank above it, or any release
// once it is final, is not_an_upgrade.
func Judge(t Target, f release.Facts, held *release.Facts) Verdict {
	var v Verdict
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
	if ok && held != nil && !upgrades(t, v.rank, *held) {
		v.reasons = append(v.reasons, Reason{Code: NotAnUpgrade})
	}
	return v
}

// upgrades reports whether a release at r upgrades the held file: the file
// is below the cutoff and r ranks above it. A held file the Target doesn't
// rank is below anything it does.
func upgrades(t Target, r Rank, held release.Facts) bool {
	q, known := held.Quality.Get()
	if !known {
		return true
	}
	h, ok := t.RankOf(q)
	return !ok || !t.Final(h) && r < h
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
