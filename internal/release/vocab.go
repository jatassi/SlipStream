package release

import (
	"fmt"
	"slices"

	"golang.org/x/text/language"
)

type (
	HDR        string
	VideoCodec string
	AudioCodec string
	Channels   string
	// Language is an ISO 639 language, two letters where ISO 639-1 has
	// one, or Original.
	Language string
)

const Original Language = "original"

// Origin is the fine source: Radarr's and Sonarr's source values meet
// here, and a Source condition tests it.
type Origin uint8

const (
	OriginCAM Origin = iota + 1
	OriginTelesync
	OriginTelecine
	OriginWorkprint
	OriginDVD
	OriginTV
	OriginWEBDL
	OriginWEBRip
	OriginBluray
)

var originKeys = []string{
	OriginCAM: "cam", OriginTelesync: "telesync", OriginTelecine: "telecine", OriginWorkprint: "workprint", OriginDVD: "dvd",
	OriginTV: "tv", OriginWEBDL: "webdl", OriginWEBRip: "webrip", OriginBluray: "bluray",
}

func (o Origin) String() string                { return originKeys[o] }
func (o Origin) MarshalText() ([]byte, error)  { return []byte(o.String()), nil }
func (o *Origin) UnmarshalText(b []byte) error { return parseKey(o, "a source", originKeys, b) }

// Modifier is Radarr's quality modifier.
type Modifier uint8

const (
	ModRemux Modifier = iota + 1
	ModBRDisk
	ModRawHD
	ModRegional
	ModScreener
)

var modifierKeys = []string{ModRemux: "remux", ModBRDisk: "brdisk", ModRawHD: "rawhd", ModRegional: "regional", ModScreener: "screener"}

func (m Modifier) String() string               { return modifierKeys[m] }
func (m Modifier) MarshalText() ([]byte, error) { return []byte(m.String()), nil }
func (m *Modifier) UnmarshalText(b []byte) error {
	return parseKey(m, "a quality modifier", modifierKeys, b)
}

// Flag is one of Radarr's indexer flags, by S27 (Ported site indexers)'
// names.
type Flag string

const (
	Freeleech    Flag = "freeleech"
	Halfleech    Flag = "halfleech"
	Freeleech75  Flag = "freeleech75"
	Freeleech25  Flag = "freeleech25"
	DoubleUpload Flag = "doubleupload"
	Internal     Flag = "internal"
	Scene        Flag = "scene"
	Exclusive    Flag = "exclusive"
	Nuked        Flag = "nuked"
	Golden       Flag = "golden"
	Approved     Flag = "approved"
)

func Flags() []Flag {
	return []Flag{Freeleech, Halfleech, Freeleech75, Freeleech25, DoubleUpload, Internal, Scene, Exclusive, Nuked, Golden, Approved}
}

// ParseFlag is the Flag named s, false for a name it doesn't know.
func ParseFlag(s string) (Flag, bool) {
	f := Flag(s)
	return f, slices.Contains(Flags(), f)
}

// parseKey sets *v to the value whose key in keys is b, the zero value's
// key being unused.
func parseKey[T ~uint8](v *T, kind string, keys []string, b []byte) error {
	i := slices.Index(keys[1:], string(b))
	if i < 0 {
		return fmt.Errorf("%q is not %s", b, kind)
	}
	*v = T(i + 1)
	return nil
}

func HDRs() []HDR { return []HDR{"DV", "HDR10+", "HDR10", "HLG", "SDR"} }

func VideoCodecs() []VideoCodec { return []VideoCodec{"x264", "x265", "AV1", "VP9", "XviD", "MPEG2"} }

func AudioCodecs() []AudioCodec {
	return []AudioCodec{"TrueHD", "DTS-HD MA", "DTS", "DD+", "DD", "AAC", "FLAC", "PCM", "Opus", "MP3"}
}

func ChannelLayouts() []Channels { return []Channels{"7.1", "5.1", "2.0", "1.0"} }

// probedVideo is the video codec of each codec name ffprobe gives that the
// vocabulary has a word for. MPEG-4 Part 2 is the family XviD names.
var probedVideo = map[string]VideoCodec{
	"h264": "x264", "hevc": "x265", "av1": "AV1", "vp9": "VP9", "mpeg4": "XviD", "mpeg2video": "MPEG2",
}

// VideoCodecOf is the video codec of a stream ffprobe names codec,
// unknown when the vocabulary has no word for it.
func VideoCodecOf(codec string) Fact[VideoCodec] {
	if v, ok := probedVideo[codec]; ok {
		return Known(v)
	}
	return Fact[VideoCodec]{}
}

func ParseHDR(s string) (HDR, error) { return parseValue("an HDR format", HDRs(), s) }

func ParseVideoCodec(s string) (VideoCodec, error) {
	return parseValue("a video codec", VideoCodecs(), s)
}

func ParseAudioCodec(s string) (AudioCodec, error) {
	return parseValue("an audio codec", AudioCodecs(), s)
}

func ParseChannels(s string) (Channels, error) {
	return parseValue("a channel layout", ChannelLayouts(), s)
}

func parseValue[T ~string](kind string, all []T, s string) (T, error) {
	if !slices.Contains(all, T(s)) {
		return "", fmt.Errorf("%q is not %s", s, kind)
	}
	return T(s), nil
}

var notLanguages = []string{"und", "mis", "mul", "zxx"}

// ParseLanguage reads an ISO 639 code of any part, such as a probe's
// "eng", as its canonical code ("en"), or "original".
func ParseLanguage(s string) (Language, error) {
	if s == string(Original) {
		return Original, nil
	}
	tag, err := language.All.Parse(s)
	base, _ := tag.Base()
	if err != nil || tag.String() != base.String() || slices.Contains(notLanguages, base.String()) {
		return "", fmt.Errorf("%q is not an ISO 639 language", s)
	}
	return Language(base.String()), nil
}
