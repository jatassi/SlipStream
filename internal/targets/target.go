package targets

import (
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets/internal/db"
)

// Tier is qualities a Target ranks as equal, in the Admin's order. A
// Target's tiers are disjoint and none is empty.
type Tier []Accepted

// Accepted is one quality a Target accepts and its size range. A size can
// only name an accepted quality, because it lives on the quality's row.
type Accepted struct {
	Quality release.Quality
	Size    Size
}

// Size is a range in MiB per minute of covered runtime. A nil Max is no
// limit.
type Size struct {
	Min float64
	Max *float64
}

// Floor is q's size for a Target that names none: its floor, no maximum.
func Floor(q release.Quality) Size { return Size{Min: q.Info().Floor} }

type (
	Attribute = db.Attribute
	Mode      = db.Mode
)

const (
	HDR        = db.HDR
	VideoCodec = db.VideoCodec
	AudioCodec = db.AudioCodec
	Channels   = db.Channels
	Language   = db.Language
)

const (
	Any  = db.Any
	Only = db.Only
	Not  = db.Not
)

// Attributes is every attribute a filter rules on, in the order verdicts
// and the API list them.
func Attributes() []Attribute { return []Attribute{HDR, VideoCodec, AudioCodec, Channels, Language} }

// Filter is one attribute's hard rule. For a multi-valued fact, Only
// passes when a present value is in Values and Not when none is. Values
// is empty under Any and set otherwise.
type Filter[V ~string] struct {
	Mode   Mode
	Values []V
}

type Filters struct {
	HDR      Filter[release.HDR]
	Video    Filter[release.VideoCodec]
	Audio    Filter[release.AudioCodec]
	Channels Filter[release.Channels]
	Language Filter[release.Language]
}

var ErrBadFilter = errors.New("a filter is Any with values, Only or Not with none, or names a value twice")

// ParseMode reads a mode by name: any, only or not.
func ParseMode(s string) (Mode, error) {
	switch s {
	case "any":
		return Any, nil
	case string(Only), string(Not):
		return Mode(s), nil
	}
	return "", fmt.Errorf("%w: %q is not a mode", ErrBadFilter, s)
}

// Set sets a's rule from its values by name, as the API and scenarios
// write them.
func (f *Filters) Set(a Attribute, m Mode, values []string) error {
	if (m == Any) != (len(values) == 0) {
		return fmt.Errorf("%w: %s has %d values", ErrBadFilter, a, len(values))
	}
	switch a {
	case HDR:
		return set(&f.HDR, m, values, release.ParseHDR)
	case VideoCodec:
		return set(&f.Video, m, values, release.ParseVideoCodec)
	case AudioCodec:
		return set(&f.Audio, m, values, release.ParseAudioCodec)
	case Channels:
		return set(&f.Channels, m, values, release.ParseChannels)
	case Language:
		return set(&f.Language, m, values, release.ParseLanguage)
	}
	return fmt.Errorf("%w: %q is not an attribute", ErrBadFilter, a)
}

func set[V ~string](f *Filter[V], m Mode, values []string, parse func(string) (V, error)) error {
	out := Filter[V]{Mode: m}
	for _, s := range values {
		v, err := parse(s)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrBadFilter, err)
		}
		if slices.Contains(out.Values, v) {
			return fmt.Errorf("%w: %s twice", ErrBadFilter, v)
		}
		out.Values = append(out.Values, v)
	}
	*f = out
	return nil
}

// Of is a's rule by name.
func (f *Filters) Of(a Attribute) (mode Mode, values []string) {
	switch a {
	case HDR:
		return named(f.HDR)
	case VideoCodec:
		return named(f.Video)
	case AudioCodec:
		return named(f.Audio)
	case Channels:
		return named(f.Channels)
	case Language:
		return named(f.Language)
	}
	panic("targets: no attribute " + string(a))
}

func named[V ~string](f Filter[V]) (mode Mode, values []string) {
	out := make([]string, len(f.Values))
	for i, v := range f.Values {
		out[i] = string(v)
	}
	return f.Mode, out
}

// Target is a named encoding goal. Later specs add Custom Format scores,
// Hold, Conversion settings and download steps to this record.
type Target struct {
	ID      TargetID
	Name    string
	Tiers   []Tier // best first
	Cutoff  int    // index into Tiers; a file at or above it is final
	Filters Filters
	// Profiles is every Profile holding the Target, by name, and Contains
	// every other Target it contains. Both are read only.
	Profiles  []ProfileRef
	Contains  []TargetID
	CreatedAt storage.Time
}

type ProfileRef struct {
	ID   ProfileID
	Name string
}

// Spec is a Target as an Admin writes it: tiers of qualities, a cutoff
// naming a quality of the cutoff tier, and sizes by quality. An accepted
// quality Sizes leaves out takes its Floor.
type Spec struct {
	Name    string
	Tiers   [][]release.Quality
	Cutoff  release.Quality
	Filters Filters
	Sizes   map[release.Quality]Size
}

// ErrQualityNotInTarget is a cutoff or a size naming a quality the Target
// doesn't accept. ErrBadTiers is no tier, an empty tier or a quality in
// two places. ErrBadSize is a negative minimum or a maximum below it.
var (
	ErrQualityNotInTarget = errors.New("the Target doesn't accept that quality")
	ErrBadTiers           = errors.New("tiers are missing, empty or repeat a quality")
	ErrBadSize            = errors.New("a size is negative or its maximum is below its minimum")
)

// NewTarget checks s and lays it out as a Target with no id yet.
func NewTarget(s Spec) (Target, error) {
	tiers, cutoff, err := layTiers(s.Tiers, s.Cutoff)
	if err != nil {
		return Target{}, err
	}
	for q, size := range s.Sizes {
		if _, _, ok := (Target{Tiers: tiers}).accepted(q); !ok {
			return Target{}, fmt.Errorf("%w: a size for %s", ErrQualityNotInTarget, q)
		}
		if size.Min < 0 || math.IsNaN(size.Min) || size.Max != nil && (math.IsNaN(*size.Max) || *size.Max < size.Min) {
			return Target{}, fmt.Errorf("%w: %s", ErrBadSize, q)
		}
	}
	for _, tier := range tiers {
		for i, a := range tier {
			if size, ok := s.Sizes[a.Quality]; ok {
				tier[i].Size = size
			}
		}
	}
	return Target{Name: s.Name, Tiers: tiers, Cutoff: cutoff, Filters: s.Filters}, nil
}

// layTiers lays qualities out as tiers at their floors, with the index of
// the tier holding cutoff.
func layTiers(qualities [][]release.Quality, cutoff release.Quality) ([]Tier, int, error) {
	if len(qualities) == 0 {
		return nil, 0, ErrBadTiers
	}
	var tiers []Tier
	var seen []release.Quality
	at := -1
	for i, qs := range qualities {
		if len(qs) == 0 {
			return nil, 0, fmt.Errorf("%w: tier %d is empty", ErrBadTiers, i+1)
		}
		if slices.Contains(qs, cutoff) {
			at = i
		}
		tier := make(Tier, len(qs))
		for j, q := range qs {
			if slices.Contains(seen, q) {
				return nil, 0, fmt.Errorf("%w: %s twice", ErrBadTiers, q)
			}
			seen = append(seen, q)
			tier[j] = Accepted{Quality: q, Size: Floor(q)}
		}
		tiers = append(tiers, tier)
	}
	if at < 0 {
		return nil, 0, fmt.Errorf("%w: the cutoff %s", ErrQualityNotInTarget, cutoff)
	}
	return tiers, at, nil
}

// Spec is t as an Admin writes it, every size named.
func (t Target) Spec() Spec {
	s := Spec{Name: t.Name, Filters: t.Filters, Cutoff: t.Tiers[t.Cutoff][0].Quality, Sizes: map[release.Quality]Size{}}
	for _, tier := range t.Tiers {
		var qs []release.Quality
		for _, a := range tier {
			qs = append(qs, a.Quality)
			s.Sizes[a.Quality] = a.Size
		}
		s.Tiers = append(s.Tiers, qs)
	}
	return s
}

// Rank is a tier index. Lower is better.
type Rank int

// accepted is q's place in t, false when t doesn't accept it.
func (t Target) accepted(q release.Quality) (Accepted, Rank, bool) {
	for i, tier := range t.Tiers {
		for _, a := range tier {
			if a.Quality == q {
				return a, Rank(i), true
			}
		}
	}
	return Accepted{}, 0, false
}

// RankOf is q's tier in t, false when t doesn't accept it.
func (t Target) RankOf(q release.Quality) (Rank, bool) {
	_, r, ok := t.accepted(q)
	return r, ok
}

// Final reports whether a file at r is at or past the cutoff, so nothing
// upgrades it.
func (t Target) Final(r Rank) bool { return int(r) <= t.Cutoff }

// Contains reports whether every file that meets inner also meets outer:
// inner's qualities are outer's, each of inner's filters passes only what
// outer's passes, and inner's sizes sit inside outer's. The Profile editor
// flags it; it is never an error.
func Contains(outer, inner Target) bool {
	for _, tier := range inner.Tiers {
		for _, a := range tier {
			o, _, ok := outer.accepted(a.Quality)
			if !ok || !within(a.Size, o.Size) {
				return false
			}
		}
	}
	return stricter(outer.Filters.HDR, inner.Filters.HDR, release.HDRs(), true) &&
		stricter(outer.Filters.Video, inner.Filters.Video, release.VideoCodecs(), false) &&
		stricter(outer.Filters.Audio, inner.Filters.Audio, release.AudioCodecs(), true) &&
		stricter(outer.Filters.Channels, inner.Filters.Channels, release.ChannelLayouts(), true) &&
		stricter(outer.Filters.Language, inner.Filters.Language, nil, true)
}

func within(inner, outer Size) bool {
	return inner.Min >= outer.Min && (outer.Max == nil || inner.Max != nil && *inner.Max <= *outer.Max)
}

// stricter reports whether every fact passing inner passes outer. A
// multi-valued fact can hold values from both an Only and a Not set, so
// across modes only a single-valued fact over a known vocabulary can be
// contained.
func stricter[V ~string](outer, inner Filter[V], all []V, multi bool) bool {
	subset := func(a, b []V) bool {
		for _, v := range a {
			if !slices.Contains(b, v) {
				return false
			}
		}
		return true
	}
	switch {
	case outer.Mode == Any:
		return true
	case inner.Mode == Any:
		return false
	case inner.Mode == Only && outer.Mode == Only:
		return subset(inner.Values, outer.Values)
	case inner.Mode == Not && outer.Mode == Not:
		return subset(outer.Values, inner.Values)
	case multi:
		return false
	case inner.Mode == Only:
		return !slices.ContainsFunc(inner.Values, func(v V) bool { return slices.Contains(outer.Values, v) })
	}
	passing := slices.DeleteFunc(slices.Clone(all), func(v V) bool { return slices.Contains(inner.Values, v) })
	return subset(passing, outer.Values)
}
