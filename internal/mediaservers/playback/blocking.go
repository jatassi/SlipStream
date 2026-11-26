package playback

import (
	"slices"
	"strconv"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

// blocking is the dimensions that stopped p playing directly: its reasons
// through the reason table, or with none, what the streams and the
// received facts show.
func blocking(p Play, pr Proofs) []Blocked {
	if len(p.Reasons) > 0 {
		var c []Blocked
		for _, name := range p.Reasons {
			d, stream := reasonOf(name)
			c = add(c, d, valueOn(d, name, stream, p.Snapshot.Facts, pr))
		}
		return inTableOrder(c)
	}
	return fromStreams(p, pr)
}

// fromStreams is the dimensions a play with no reason shows: what the
// server transcoded and what the player received that differs from the
// file, less what the player was proven to receive unchanged.
func fromStreams(p Play, pr Proofs) []Blocked {
	f, r := p.Snapshot.Facts, p.Received
	var c []Blocked
	if p.Streams.Video == link.StreamTranscode {
		c = videoCandidates(f, r, pr)
	}
	if p.Streams.Audio == link.StreamTranscode && f.Audio != nil {
		c = append(c, audioCandidates(*f.Audio, r)...)
	}
	if p.Streams.Subtitles == link.SubtitlesBurned {
		c = slices.DeleteFunc(c, func(b Blocked) bool { return b.Dimension == VideoCodec || b.Dimension == HDR })
		c = add(c, Subtitles, string(f.SubtitleCodec()))
	}
	c = slices.DeleteFunc(c, func(b Blocked) bool { return b.Dimension != HDR && pr.provenFile(b.Dimension, f) })
	if len(c) == 0 {
		if r.Bitrate > 0 && r.Bitrate < f.Bitrate {
			return []Blocked{block(Bitrate, "")}
		}
		return []Blocked{block(Unknown, "")}
	}
	return inTableOrder(c)
}

// videoCandidates compares raw heights, so a cropped 1920x800 file is
// still 1080p. A proven HDR format can't have been the block.
func videoCandidates(f Facts, r link.Received, pr Proofs) []Blocked {
	var c []Blocked
	if r.Height > 0 && r.Height < f.Video.Height {
		c = add(c, Resolution, f.Video.Resolution.String())
	}
	if r.VideoCodec != "" && r.VideoCodec != string(f.Video.Codec) {
		c = add(c, VideoCodec, string(f.Video.Codec))
	}
	if hdr := pr.unprovenHDR(f.Video); len(hdr) > 0 {
		c = add(c, HDR, string(hdr[0]))
	}
	return c
}

func audioCandidates(a Audio, r link.Received) []Blocked {
	var c []Blocked
	if r.AudioCodec != "" && r.AudioCodec != string(a.Codec) {
		c = add(c, AudioCodec, string(a.Codec))
	}
	if r.Channels > 0 && r.Channels < a.Channels {
		c = add(c, Channels, string(a.Layout()))
	}
	return c
}

// valueOn is f's word on d as a reason names it: the reason's own name
// where its line is the lead, the stream an unreadable reason names, and
// "" where no value applies.
func valueOn(d Dimension, reason, stream string, f Facts, pr Proofs) string {
	switch d {
	case Resolution:
		return f.Video.Resolution.String()
	case VideoCodec:
		return string(f.Video.Codec)
	case HDR:
		if hdr := pr.blockedHDR(f.Video); len(hdr) > 0 {
			return string(hdr[0])
		}
	case AudioCodec:
		if f.Audio != nil {
			return string(f.Audio.Codec)
		}
	case Channels:
		if f.Audio != nil {
			return string(f.Audio.Layout())
		}
	case Container:
		return string(f.Container)
	case Subtitles:
		return string(f.SubtitleCodec())
	case VideoProfile:
		return f.Video.Profile
	case BitDepth:
		if f.Video.BitDepth > 0 {
			return strconv.Itoa(f.Video.BitDepth)
		}
	case AudioTrack, VideoShape, AudioFormat:
		return reason
	case Unreadable:
		return stream
	case StreamCount, FrameRate, Bitrate, PlayerError, Unknown:
	}
	return ""
}

// add is c with d at value v, unless c already blocks on d.
func add(c []Blocked, d Dimension, v string) []Blocked {
	if slices.ContainsFunc(c, func(b Blocked) bool { return b.Dimension == d }) {
		return c
	}
	return append(c, block(d, v))
}

func block(d Dimension, v string) Blocked {
	return Blocked{Dimension: d, Value: v, Choosable: choosable(d)}
}

func choosable(d Dimension) bool { return dimensions[order(d)].Choose != "" }

// order is d's place in the dimension table.
func order(d Dimension) int {
	i := 0
	for dimensions[i].Dimension != d {
		i++
	}
	return i
}

func inTableOrder(c []Blocked) []Blocked {
	slices.SortFunc(c, func(a, b Blocked) int { return order(a.Dimension) - order(b.Dimension) })
	return c
}
