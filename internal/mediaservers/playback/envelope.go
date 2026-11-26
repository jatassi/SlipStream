package playback

import (
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/release"
)

// vocabulary is the values a Target can name on each dimension it can
// choose, most demanding first.
var vocabulary = map[Dimension][]Fact{
	Resolution: resolutionWords(release.R2160p, release.R1080p, release.R720p, release.R480p),
	VideoCodec: words(FactVideoCodec, release.VideoCodecs()),
	HDR:        words(FactHDR, release.HDRs()),
	AudioCodec: words(FactAudioCodec, release.AudioCodecs()),
	Channels:   channelWords(release.ChannelLayouts()),
}

func resolutionWords(rs ...release.Resolution) []Fact {
	out := make([]Fact, len(rs))
	for i, r := range rs {
		out[i] = Fact{Kind: FactResolution, Text: r.String(), Rank: resolutionRank(r)}
	}
	return out
}

func words[T ~string](k FactKind, vs []T) []Fact {
	out := make([]Fact, len(vs))
	for i, v := range vs {
		out[i] = Fact{Kind: k, Text: string(v)}
	}
	return out
}

// channelWords ranks each layout by its channel count: 5.1 is six.
func channelWords(ls []release.Channels) []Fact {
	out := make([]Fact, len(ls))
	for i, l := range ls {
		main, lfe, _ := strings.Cut(string(l), ".")
		n, _ := strconv.Atoi(main)
		m, _ := strconv.Atoi(lfe)
		out[i] = Fact{Kind: FactChannels, Text: string(l), Rank: int64(n + m)}
	}
	return out
}

// envelope is the proven vocabulary values on each dimension a Target can
// choose, most demanding first.
type envelope map[Dimension][]Fact

// vocabulary is every vocabulary value the player proved.
func (pr Proofs) vocabulary() envelope {
	e := envelope{}
	for d, vs := range vocabulary {
		for _, v := range vs {
			if _, ok := pr.proven(v); ok {
				e[d] = append(e[d], v)
			}
		}
	}
	return e
}

// envelopeOf is the proven values with, on each blocking dimension, the
// played file's value dropped, and for resolution and channels every
// value at or above it.
func envelopeOf(p Play, dims []Blocked, pr Proofs) envelope {
	e, f := pr.vocabulary(), p.Snapshot.Facts
	for _, b := range dims {
		e[b.Dimension] = slices.DeleteFunc(e[b.Dimension], drops(b.Dimension, f, pr))
	}
	return e
}

// drops is whether blocking on d rules a value out: one of the file's
// values, and on resolution and channels any value at or above it.
func drops(d Dimension, f Facts, pr Proofs) func(Fact) bool {
	vs, lower := fileValues(d, f, pr), dimensions[order(d)].Lower
	return func(v Fact) bool {
		return slices.ContainsFunc(vs, func(b Fact) bool { return v.Text == b.Text || lower && v.Rank >= b.Rank })
	}
}

// targetClass is the resolution a Target names r by: 576p, 540p and 360p
// read as 480p.
func targetClass(r release.Resolution) release.Resolution {
	if r > release.R720p {
		return release.R480p
	}
	return r
}

// fileValues is f's values on d, ranked as a Target names them, none on a
// dimension no Target chooses.
func fileValues(d Dimension, f Facts, pr Proofs) []Fact {
	switch d {
	case Resolution:
		r := f.Video.Resolution
		return []Fact{{Kind: FactResolution, Text: r.String(), Rank: resolutionRank(targetClass(r))}}
	case VideoCodec:
		return []Fact{{Kind: FactVideoCodec, Text: string(f.Video.Codec)}}
	case HDR:
		var out []Fact
		for _, h := range pr.blockedHDR(f.Video) {
			out = append(out, Fact{Kind: FactHDR, Text: string(h)})
		}
		return out
	case AudioCodec:
		if f.Audio != nil {
			return []Fact{{Kind: FactAudioCodec, Text: string(f.Audio.Codec)}}
		}
	case Channels:
		if f.Audio == nil {
			return []Fact{{Kind: FactChannels}}
		}
		return []Fact{{Kind: FactChannels, Text: string(f.Audio.Layout()), Rank: int64(f.Audio.Channels)}}
	case Container, Subtitles, AudioTrack, StreamCount, VideoProfile, BitDepth, FrameRate, VideoShape, AudioFormat, Bitrate, Unreadable, PlayerError, Unknown:
	}
	return nil
}

// firstMissing is the first dimension a Target can choose that holds no
// value, at the played file's value.
func (e envelope) firstMissing(f Facts, pr Proofs) (Blocked, bool) {
	for _, row := range dimensions {
		if row.Choose != "" && len(e[row.Dimension]) == 0 {
			return block(row.Dimension, valueOn(row.Dimension, "", "", f, pr)), true
		}
	}
	return Blocked{}, false
}

// facts is the values the envelope holds that evidence names: on
// resolution and channels only the highest, which proves the rest.
func (e envelope) facts() []Fact {
	var out []Fact
	for _, row := range dimensions {
		vs := e[row.Dimension]
		if row.Lower && len(vs) > 0 {
			vs = vs[:1]
		}
		out = append(out, vs...)
	}
	return out
}
