package playback

import (
	"cmp"
	"slices"
	"strconv"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/release"
)

// FactsOf is r's facts with the tracks a play chose. A position past r's
// tracks is no track. A play that names no audio track played the
// default one, else the first. An external track takes its codec and
// language from the track itself, because the snapshot's read loads no
// Sidecars for the played file.
func FactsOf(r media.Record, audio, subtitle *link.Track) Facts {
	f := Facts{
		Container: media.ContainerOf(r.Container),
		Bitrate:   r.Bitrate,
		Video: Video{
			Profile: r.Profile, BitDepth: r.BitDepth, Height: r.Height, HDR: r.HDR,
		},
		Audio:    audioOf(r.Audio, audio),
		Subtitle: subtitleOf(r.Subtitles, subtitle),
	}
	f.Video.Codec, _ = release.VideoCodecOf(r.Codec).Get()
	if r.Width > 0 || r.Height > 0 {
		f.Video.Resolution = r.Resolution()
	}
	if r.DolbyVision != nil {
		f.Video.DolbyVisionProfile = r.DolbyVision.Profile
	}
	return f
}

func audioOf(tracks []media.Audio, t *link.Track) *Audio {
	switch {
	case t != nil && t.External != nil:
		return &Audio{Codec: release.AudioCodec(t.External.Codec), Language: t.External.Language}
	case t != nil && t.Position >= len(tracks):
		return nil
	case t != nil:
		return audioFrom(tracks[t.Position])
	case len(tracks) == 0:
		return nil
	}
	i := max(0, slices.IndexFunc(tracks, func(a media.Audio) bool { return a.Default }))
	return audioFrom(tracks[i])
}

func audioFrom(a media.Audio) *Audio {
	return &Audio{Codec: media.AudioCodecOf(a), Channels: a.Channels, Language: string(a.Language)}
}

func subtitleOf(tracks []media.Subtitle, t *link.Track) *Subtitle {
	switch {
	case t == nil:
		return nil
	case t.External != nil:
		c := release.SubtitleCodec(t.External.Codec)
		return &Subtitle{Codec: c, Image: c.Image(), External: true, Language: t.External.Language}
	case t.Position >= len(tracks):
		return nil
	}
	s := tracks[t.Position]
	return &Subtitle{Codec: media.SubtitleCodecOf(s.Codec), Image: s.Image, Language: string(s.Language)}
}

// Fact is one tracked fact of a file: a value a play can prove the player
// received unchanged.
type Fact struct {
	Kind FactKind
	// Text is the vocabulary word, or the number for a count.
	Text string
	// Rank orders resolution, channels and bitrate: a higher one is more
	// demanding.
	Rank int64
}

// Tracked is every fact f holds, one per HDR format.
func (f Facts) Tracked() []Fact {
	var out []Fact
	add := func(k FactKind, text string, rank int64) {
		if text != "" {
			out = append(out, Fact{Kind: k, Text: text, Rank: rank})
		}
	}
	add(FactContainer, string(f.Container), 0)
	if f.Bitrate > 0 {
		add(FactBitrate, strconv.FormatInt(f.Bitrate, 10), f.Bitrate)
	}
	v := f.Video
	if v.Resolution != 0 {
		add(FactResolution, v.Resolution.String(), resolutionRank(v.Resolution))
	}
	add(FactVideoCodec, string(v.Codec), 0)
	add(FactVideoProfile, v.Profile, 0)
	if v.BitDepth > 0 {
		add(FactBitDepth, strconv.Itoa(v.BitDepth), 0)
	}
	for _, h := range v.HDR {
		add(FactHDR, string(h), 0)
	}
	if v.DolbyVisionProfile > 0 {
		add(FactDolbyVision, strconv.Itoa(v.DolbyVisionProfile), 0)
	}
	if a := f.Audio; a != nil {
		add(FactAudioCodec, string(a.Codec), 0)
		if a.Channels > 0 {
			add(FactChannels, cmp.Or(string(a.Layout()), strconv.Itoa(a.Channels)), int64(a.Channels))
		}
	}
	if s := f.Subtitle; s != nil {
		add(FactSubtitleCodec, string(s.Codec), 0)
		add(FactSubtitleImage, strconv.FormatBool(s.Image), 0)
	}
	return out
}

// resolutionRank is r's rank, higher for a higher resolution.
func resolutionRank(r release.Resolution) int64 { return int64(release.R360p - r + 1) }
