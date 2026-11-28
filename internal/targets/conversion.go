package targets

import (
	"context"
	"errors"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets/internal/db"
)

// Conversion is a Target's Conversion settings: how SlipStream makes its
// files from another Target's. RunOn is the one Node its Conversions run
// on, nil for any. Version names the Settings a made file was made with.
// It rises with each change to Settings and never for RunOn, and is read
// only.
type Conversion struct {
	Settings encode.Settings
	RunOn    *role.NodeID
	Version  int
}

// SourceInUseError is a change that would leave a made-from row of
// Profiles with nothing to make its file from: turning off the
// Conversion settings of a Target those rows make, or a reorder that puts
// a row's source Target below it.
type SourceInUseError struct{ Profiles []string }

func (e *SourceInUseError) Error() string {
	return "a made-from row of " + joinNames(e.Profiles) + " depends on it"
}

func joinNames(names []string) string {
	if len(names) == 1 {
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// ConversionRefusedError is Conversion settings that make files their
// Target refuses, each problem with its code.
type ConversionRefusedError struct{ Problems []Problem }

func (e *ConversionRefusedError) Error() string {
	codes := make([]string, len(e.Problems))
	for i, p := range e.Problems {
		codes[i] = string(p.Code)
	}
	return "the Conversion settings make files the Target refuses: " + strings.Join(codes, ", ")
}

type ProblemCode string

// The ways Conversion settings make files their Target refuses.
// CodecFiltered is a video or audio choice the Target's filters refuse,
// RangeFromSDR an HDR filter that refuses the SDR file every SDR source
// makes, OutsideSize a bitrate outside the size range of every quality
// the Target accepts at the largest resolution, ContainerAudio MP4 with
// copied audio the Target allows MP4 can't hold, and QualityUnranked a
// largest resolution at which the Target accepts no quality.
const (
	CodecFiltered   ProblemCode = "codec_filtered"
	RangeFromSDR    ProblemCode = "range_from_sdr"
	OutsideSize     ProblemCode = "outside_size"
	ContainerAudio  ProblemCode = encode.ContainerAudio
	QualityUnranked ProblemCode = "quality_unranked"
)

func ProblemCodes() []ProblemCode {
	return []ProblemCode{CodecFiltered, RangeFromSDR, OutsideSize, ContainerAudio, QualityUnranked}
}

// Problem is one way settings make refused files. Field names the
// setting by its encode.Raw name, and Facts are what the words name.
type Problem struct {
	Code  ProblemCode
	Field string
	Facts map[string]string
}

// kbpsPerMiBMinute is 1 MiB per minute in kilobits per second.
const kbpsPerMiBMinute = 1 << 20 * 8 / 60.0 / 1000

var resolutions = map[encode.Height]release.Resolution{
	encode.P2160: release.R2160p, encode.P1080: release.R1080p, encode.P720: release.R720p, encode.P480: release.R480p,
}

// audioMade is the audio codecs and layouts each audio choice other than
// copy writes.
var audioMade = map[encode.Audio]struct {
	codecs   []release.AudioCodec
	channels []release.Channels
}{
	encode.StereoAAC:       {[]release.AudioCodec{"AAC"}, []release.Channels{"2.0"}},
	encode.AC3AndStereoAAC: {[]release.AudioCodec{"DD", "AAC"}, []release.Channels{"5.1", "2.0"}},
}

var notInMP4 = []release.AudioCodec{"TrueHD", "DTS-HD MA", "DTS"}

// ConversionProblems is every way s makes files that t refuses, in the
// order of the settings' fields.
func ConversionProblems(t Spec, s encode.Settings) []Problem {
	var out []Problem
	add := func(code ProblemCode, field string, facts map[string]string) {
		out = append(out, Problem{Code: code, Field: field, Facts: facts})
	}
	if !allows(t.Filters.Video, single(release.VideoCodecOf(string(s.Codec)))) {
		add(CodecFiltered, "Codec", map[string]string{"codec": string(s.Codec)})
	}
	if at := acceptedAt(t, s.MaxHeight); at == nil {
		add(QualityUnranked, "MaxHeight", map[string]string{"height": strconv.Itoa(int(s.MaxHeight))})
	} else if !slices.ContainsFunc(at, func(a Accepted) bool { return a.Size.holds(float64(s.Kbps) / kbpsPerMiBMinute) }) {
		add(OutsideSize, "Kbps", sizeFacts(s.Kbps, at))
	}
	if !allows(t.Filters.HDR, release.Known([]release.HDR{"SDR"})) {
		add(RangeFromSDR, "Range", map[string]string{"range": string(s.Range)})
	}
	if made, ok := audioMade[s.Audio]; ok && (!allows(t.Filters.Audio, release.Known(made.codecs)) || !allows(t.Filters.Channels, release.Known(made.channels))) {
		add(CodecFiltered, "Audio", map[string]string{"audio": string(s.Audio)})
	}
	if s.Container == encode.MP4 && s.Audio == encode.CopyAudio && slices.ContainsFunc(notInMP4, func(c release.AudioCodec) bool {
		return allows(t.Filters.Audio, release.Known([]release.AudioCodec{c}))
	}) {
		add(ContainerAudio, "Audio", map[string]string{"container": string(s.Container)})
	}
	return out
}

func checkHoldAndConversion(s Spec) error {
	if err := checkHold(s.Hold); err != nil || s.Conversion == nil {
		return err
	}
	if problems := ConversionProblems(s, s.Conversion.Settings); problems != nil {
		return &ConversionRefusedError{Problems: problems}
	}
	return nil
}

func allows[V ~string](f Filter[V], fact release.Fact[[]V]) bool {
	_, ok := passes(f, fact, listedIn(f))
	return ok
}

func (s Size) holds(rate float64) bool { return rate >= s.Min && (s.Max == nil || rate <= *s.Max) }

// sizeFacts names kbps and the bitrates of the lowest minimum and the
// highest maximum of at, with no maximum when one of at has none.
func sizeFacts(kbps int, at []Accepted) map[string]string {
	low, high := math.Inf(1), 0.0
	for _, a := range at {
		low = min(low, a.Size.Min)
		if a.Size.Max == nil {
			high = math.Inf(1)
		} else {
			high = max(high, *a.Size.Max)
		}
	}
	facts := map[string]string{"kbps": strconv.Itoa(kbps), "minKbps": strconv.Itoa(kbpsOf(low))}
	if !math.IsInf(high, 1) {
		facts["maxKbps"] = strconv.Itoa(kbpsOf(high))
	}
	return facts
}

func kbpsOf(mibPerMinute float64) int { return int(math.Round(mibPerMinute * kbpsPerMiBMinute)) }

// acceptedAt is every quality t accepts at height's resolution, with its
// size, in tier order.
func acceptedAt(t Spec, height encode.Height) []Accepted {
	var out []Accepted
	for _, tier := range t.Tiers {
		for _, q := range tier {
			if q.Info().Resolution != resolutions[height] {
				continue
			}
			size, ok := t.Sizes[q]
			if !ok {
				size = Floor(q)
			}
			out = append(out, Accepted{Quality: q, Size: size})
		}
	}
	return out
}

// DefaultKbps is R15's bitrate for t's files made at height from a
// source of origin tier: the middle of the size range of the quality such
// a file ranks as, else of the widest range among t's qualities at that
// resolution, else of that quality's floor. A range with no maximum uses
// its minimum plus a quarter. The answer is kept within the bitrates
// Conversion settings take.
func DefaultKbps(t Target, at encode.Height, tier release.Origin) int {
	res := resolutions[at]
	q, _ := release.Facts{Source: release.Known(tier)}.WithProbe(release.Probed{Resolution: release.Known(res)}).Quality.Get()
	size := Floor(q)
	if a, _, ok := t.accepted(q); ok {
		size = a.Size
	} else if widest, ok := widestAt(t, res); ok {
		size = widest
	}
	middle := size.Min * 1.25
	if size.Max != nil {
		middle = (size.Min + *size.Max) / 2
	}
	return min(max(kbpsOf(middle), encode.MinKbps), encode.MaxKbps)
}

// widestAt is the widest size range among t's qualities at res, a range
// with no maximum widest of all, the first in tier order on a tie.
func widestAt(t Target, res release.Resolution) (Size, bool) {
	var (
		out   Size
		width = -1.0
	)
	for _, tier := range t.Tiers {
		for _, a := range tier {
			w := math.Inf(1)
			if a.Size.Max != nil {
				w = *a.Size.Max - a.Size.Min
			}
			if a.Quality.Info().Resolution == res && w > width {
				out, width = a.Size, w
			}
		}
	}
	return out, width >= 0
}

// readConversions reads every Target's Conversion settings by Target.
func readConversions(ctx context.Context, q *db.Queries) (map[TargetID]*Conversion, error) {
	rows, err := q.ListConversions(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[TargetID]*Conversion, len(rows))
	for _, row := range rows {
		if out[row.TargetID], err = conversionOf(row); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func conversionOf(row db.TargetsConversion) (*Conversion, error) {
	s, err := encode.Parse(encode.Raw{
		Codec: row.VideoCodec, MaxHeight: int(row.MaxHeight), Kbps: int(row.BitrateKbps), Depth: int(row.BitDepth), Range: row.Range,
		KeepDV: row.KeepDv, Curve: row.Curve, Audio: row.Audio, DropCommentary: row.DropCommentary, Container: row.Container,
	})
	if err != nil {
		return nil, err
	}
	c := &Conversion{Settings: s, Version: int(row.Version)}
	if !row.RunOnNodeID.IsZero() {
		c.RunOn = &row.RunOnNodeID
	}
	return c, nil
}

// writeConversion stores c as Target id's Conversion settings, or turns
// them off when c is nil, refusing with *SourceInUseError while a
// Profile makes id's files. Settings that change take the next version.
// Settings turned on start at the time in milliseconds, so a Target
// turned off and on again never repeats a version its made files carry.
func writeConversion(ctx context.Context, q *db.Queries, id TargetID, c *Conversion) error {
	if c == nil {
		switch making, err := q.ListProfilesMaking(ctx, id); {
		case err != nil:
			return err
		case len(making) > 0:
			return &SourceInUseError{Profiles: making}
		}
		return q.DeleteConversion(ctx, id)
	}
	version := storage.Now().Time().UnixMilli()
	switch was, err := q.GetConversion(ctx, id); {
	case errors.Is(err, storage.ErrNoRows):
	case err != nil:
		return err
	default:
		before, err := conversionOf(was)
		if err != nil {
			return err
		}
		version = was.Version
		if before.Settings != c.Settings {
			version++
		}
	}
	s := c.Settings
	var runOn role.NodeID
	if c.RunOn != nil {
		runOn = *c.RunOn
	}
	return q.PutConversion(ctx, db.PutConversionParams{
		TargetID: id, Version: version, VideoCodec: string(s.Codec), MaxHeight: int64(s.MaxHeight), BitrateKbps: int64(s.Kbps),
		BitDepth: int64(s.Depth), Range: string(s.Range), KeepDv: s.KeepDV, Curve: string(s.Curve), Audio: string(s.Audio),
		DropCommentary: s.DropCommentary, Container: string(s.Container), RunOnNodeID: runOn,
	})
}
