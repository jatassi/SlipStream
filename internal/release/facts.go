// Package release is what SlipStream knows about a release or a file: the
// fixed quality table, the vocabularies hard filters name, the cuts, and
// the reading of a name, Read, whose Facts are the one typed record the
// title readers and the probe fill.
package release

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Facts is everything known about one release or file.
type Facts struct {
	// Quality is one of the 18, derived from Source, Resolution and
	// Modifier.
	Quality    Fact[Quality]
	Source     Fact[Origin]
	Resolution Fact[Resolution]
	Modifier   Fact[Modifier]
	Revision   Revision
	Group      Fact[string]
	Cut        CutKey // Standard when nothing names one
	// Edition is the raw edition text, which an Edition condition tests.
	Edition   Fact[string]
	HDR       Fact[[]HDR]
	Video     Fact[VideoCodec]
	Audio     Fact[[]AudioCodec]
	Channels  Fact[[]Channels]
	Languages Fact[[]Language]
	// IDs are the id tags the name carries, by Torznab's names: tmdbid,
	// imdbid, tvdbid.
	IDs      map[string]string
	Size     Fact[int64] // bytes
	Flags    []Flag
	Coverage Coverage
}

// Coverage is what the Items a release or file covers add up to, which
// the caller fills from the Content Type's answer.
type Coverage struct {
	// Runtime is the sum over the covered Items, unknown when any covered
	// Item's is.
	Runtime Fact[time.Duration]
	// Original is the Work's original language, unknown until a Content
	// Type facet answers it.
	Original Fact[Language]
}

// Revision is how often a release was re-released to fix it.
type Revision struct {
	// Version is 1 unless PROPER, REPACK, RERIP or vN raise it.
	Version int
	// Real is how many times the whole word REAL appears, in capitals.
	Real int
	// Repack is REPACK or RERIP, which the same-group rule applies to.
	Repack bool
}

// String is r in upstream's words: "v1", "v2", "REAL v1" or "v2 REPACK".
func (r Revision) String() string {
	words := slices.Repeat([]string{"REAL"}, r.Real)
	words = append(words, "v"+strconv.Itoa(max(r.Version, 1)))
	if r.Repack {
		words = append(words, "REPACK")
	}
	return strings.Join(words, " ")
}

// Original reports that r is the first release: v1 and never REAL.
func (r Revision) Original() bool { return r.Real == 0 && r.Version <= 1 }

// Compare orders Real first, then Version. A zero Version reads as 1.
// Repack takes no part.
func (r Revision) Compare(o Revision) int {
	return cmp.Or(cmp.Compare(r.Real, o.Real), cmp.Compare(max(r.Version, 1), max(o.Version, 1)))
}

// Probed is what a file's probe measured.
type Probed struct {
	Resolution Fact[Resolution]
	Video      Fact[VideoCodec]
	HDR        Fact[[]HDR]
	Audio      Fact[[]AudioCodec]
	Channels   Fact[[]Channels]
	Languages  Fact[[]Language]
}

// WithProbe is f as Judge reads a file. Each fact the probe measured wins
// over the name's, and Quality is derived again from f's Source and
// Modifier at the probed Resolution.
func (f Facts) WithProbe(p Probed) Facts {
	if _, ok := p.Resolution.Get(); ok {
		f.Resolution = p.Resolution
		f.Quality = qualityOfFacts(f.Source, f.Modifier, f.Resolution)
	}
	f.Video = probedOr(p.Video, f.Video)
	f.HDR = probedOr(p.HDR, f.HDR)
	f.Audio = probedOr(p.Audio, f.Audio)
	f.Channels = probedOr(p.Channels, f.Channels)
	f.Languages = probedOr(p.Languages, f.Languages)
	return f
}

func probedOr[T any](probed, named Fact[T]) Fact[T] {
	if _, ok := probed.Get(); ok {
		return probed
	}
	return named
}
