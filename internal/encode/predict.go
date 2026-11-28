package encode

import (
	"math"
	"slices"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
)

// box is the 16:9 frame of the largest output height, which a wider or
// taller picture fits inside.
func box(h Height) (w, hh float64) { return float64(h) * 16 / 9, float64(h) }

// Out is the output picture: m's, fitted inside the 16:9 box of
// s.MaxHeight, never upscaled, with even width and height.
func Out(m media.Record, s Settings) (w, h int) {
	bw, bh := box(s.MaxHeight)
	f := min(1, bw/float64(m.Width), bh/float64(m.Height))
	return int(float64(m.Width)*f) &^ 1, int(float64(m.Height)*f) &^ 1
}

// Kbps is the video bitrate a re-encode targets: s.Kbps scaled by the
// output's pixel count against the full box, and never more than the
// source's own bitrate.
func Kbps(m media.Record, s Settings) int {
	w, h := Out(m, s)
	bw, bh := box(s.MaxHeight)
	kbps := int(math.Round(float64(s.Kbps) * float64(w*h) / (bw * bh)))
	if m.Bitrate > 0 {
		kbps = min(kbps, int(m.Bitrate/1000))
	}
	return kbps
}

// Bytes is the made file's estimated size: its video bitrate times its
// length plus the chosen audio's. A copied video keeps the source's
// bitrate less its audio.
func Bytes(m media.Record, s Settings) int64 {
	video := Kbps(m, s)
	if CopiesVideo(m, s) {
		video = max(0, int(m.Bitrate/1000)-audioKbps(m.Audio))
	}
	kbps := video + audioKbps(audioOut(m, s, Tracks(m, s, nil).Audio))
	return int64(float64(kbps) * 1000 / 8 * m.Duration.Seconds())
}

var lossless = []release.AudioCodec{"TrueHD", "DTS-HD MA", "FLAC", "PCM"}

// audioKbps is a nominal bitrate for tracks, which a media record doesn't
// state per track.
func audioKbps(tracks []media.Audio) int {
	kbps := 0
	for _, a := range tracks {
		perChannel := 96
		if slices.Contains(lossless, media.AudioCodecOf(a)) {
			perChannel = 576
		}
		kbps += perChannel * a.Channels
	}
	return kbps
}

// CopiesVideo reports whether m's video already matches s, so the
// Conversion copies the stream and keeps its modifier.
func CopiesVideo(m media.Record, s Settings) bool {
	w, h := Out(m, s)
	return Codec(m.Codec) == s.Codec && m.BitDepth == int(s.Depth) && w == m.Width && h == m.Height && keepsRange(m, s)
}

// keepsRange reports whether m's dynamic range is what s writes. A Dolby
// Vision stream stays only when s keeps it, and Profile 7 never does.
func keepsRange(m media.Record, s Settings) bool {
	if dv := m.DolbyVision; dv != nil && (!s.KeepDV || dv.Profile == 7) {
		return false
	}
	switch s.Range {
	case SDR:
		return !hdr(m.HDR)
	case HDR10:
		return slices.Contains(m.HDR, "HDR10")
	case SourceRange:
		return true
	}
	return false
}

// outHDR is the HDR formats a re-encode of m with s writes. An SDR source
// stays SDR, HDR10+'s dynamic metadata is dropped, and Dolby Vision is
// written only when s keeps it, never from Profile 7.
func outHDR(m media.Record, s Settings) []release.HDR {
	if s.Range == SDR || !hdr(m.HDR) {
		return []release.HDR{"SDR"}
	}
	if s.KeepDV && m.DolbyVision != nil && m.DolbyVision.Profile != 7 {
		return []release.HDR{"DV", "HDR10"}
	}
	if s.Range == SourceRange && slices.Contains(m.HDR, "HLG") {
		return []release.HDR{"HLG"}
	}
	return []release.HDR{"HDR10"}
}

// audioOut is the audio tracks s writes from m's selected tracks.
func audioOut(m media.Record, s Settings, selected []int) []media.Audio {
	if len(selected) == 0 {
		return nil
	}
	language := m.Audio[selected[0]].Language
	switch s.Audio {
	case CopyAudio:
		out := make([]media.Audio, len(selected))
		for i, n := range selected {
			out[i] = m.Audio[n]
		}
		return out
	case StereoAAC:
		return []media.Audio{{Codec: "aac", Channels: 2, Language: language, Default: true}}
	case AC3AndStereoAAC:
		return []media.Audio{{Codec: "ac3", Channels: 6, Language: language, Default: true}, {Codec: "aac", Channels: 2, Language: language}}
	}
	return nil
}

// Predict is the made file's title facts before it exists: the source's,
// with the resolution, codec, range and audio s writes, and the Remux,
// BR-DISK and Raw-HD modifiers cleared unless the video is copied. Its
// size is Bytes. A made file has no release name.
func Predict(src release.Read, m media.Record, s Settings) release.Read {
	out, copied := m, CopiesVideo(m, s)
	if !copied {
		w, h := Out(m, s)
		out.Video = media.Video{Codec: string(s.Codec), Width: w, Height: h, BitDepth: int(s.Depth), FrameRate: m.FrameRate, HDR: outHDR(m, s)}
	}
	out.Audio = audioOut(m, s, Tracks(m, s, nil).Audio)
	r := rewrite(src, out, copied)
	r.Facts.Size = release.Known(Bytes(m, s))
	return r
}

// Rewrite is Predict with the made file's probe winning. copied says
// whether the Conversion copied the source's video stream, which only the
// run knows: a re-encode at the same codec and resolution still clears Remux.
func Rewrite(src release.Read, out media.Record, copied bool) release.Read {
	r := rewrite(src, out, copied)
	r.Facts.Size = release.Fact[int64]{}
	if out.Bitrate > 0 && out.Duration > 0 {
		r.Facts.Size = release.Known(int64(float64(out.Bitrate) / 8 * out.Duration.Seconds()))
	}
	return r
}

var reencodedAway = []release.Modifier{release.ModRemux, release.ModBRDisk, release.ModRawHD}

func rewrite(src release.Read, out media.Record, copied bool) release.Read {
	f := src.Facts
	if mod, ok := f.Modifier.Get(); ok && !copied && slices.Contains(reencodedAway, mod) {
		f.Modifier = release.Fact[release.Modifier]{}
	}
	return release.Read{Name: src.Name, Facts: f.WithProbe(out.Probed())}
}
