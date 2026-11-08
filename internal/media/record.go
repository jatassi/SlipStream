package media

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/release"
)

// Version is the parser's version, which every record Parse reads
// carries. A library scan probes again every file whose record is older,
// so raising it is the only way to reprobe the library: a new bundled
// ffprobe doesn't raise it.
const Version = 1

// Record is what ffprobe read from one file: the one media record a found
// file, an import and a Grab's plan all keep. Its JSON is the
// files_file.probe and grabs_grab.probe columns and crosses the Node link
// as is. Video's fields sit at the top level, so a record from before
// Version 1, which held only the container, duration and the video's
// codec and size, reads as Version 0 with those.
type Record struct {
	Version   int           `json:"version"`
	Container string        `json:"container"` // format_name, "matroska,webm"
	Duration  time.Duration `json:"duration"`  // zero when the container states none
	Bitrate   int64         `json:"bitrate"`   // bits per second, zero when unstated
	Video
	Audio     []Audio    `json:"audio"`     // in stream order
	Subtitles []Subtitle `json:"subtitles"` // in stream order
}

// Video is a file's first video stream that isn't cover art.
type Video struct {
	Codec    string `json:"codec"` // ffprobe's codec_name, "hevc"
	Profile  string `json:"profile"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	BitDepth int    `json:"bitDepth"`
	// FrameRate is the stream's r_frame_rate.
	FrameRate Rate `json:"frameRate"`
	// HDR is every format the stream carries in release.HDRs() order, or
	// SDR alone when it carries none.
	HDR         []release.HDR `json:"hdr"`
	DolbyVision *DolbyVision  `json:"dolbyVision,omitempty"`
}

type Rate struct {
	Num int `json:"num"`
	Den int `json:"den"`
}

// DolbyVision is a stream's DOVI configuration record.
type DolbyVision struct {
	Profile int `json:"profile"`
	Level   int `json:"level"`
	// Compatibility is the base layer's signal: 1 HDR10, 2 SDR, 4 HLG,
	// 6 Blu-ray HDR10, 0 none.
	Compatibility int `json:"compatibility"`
}

type Audio struct {
	Codec    string           `json:"codec"`   // ffprobe's codec_name, "eac3"
	Profile  string           `json:"profile"` // "Dolby Digital Plus + Dolby Atmos"
	Atmos    bool             `json:"atmos"`
	Channels int              `json:"channels"`
	Layout   string           `json:"layout"`   // "5.1(side)"
	Language release.Language `json:"language"` // empty when untagged
	Default  bool             `json:"default"`
	Title    string           `json:"title"`
}

type Subtitle struct {
	Codec    string           `json:"codec"` // "subrip", "hdmv_pgs_subtitle"
	Image    bool             `json:"image"` // a bitmap format
	Language release.Language `json:"language"`
	Forced   bool             `json:"forced"`
	Default  bool             `json:"default"`
}

// Resolution is the video's resolution class, as quality reads it.
func (v Video) Resolution() release.Resolution { return release.ResolutionOf(v.Width, v.Height) }

func (r Record) Value() (driver.Value, error) {
	b, err := json.Marshal(r)
	return string(b), err
}

func (r *Record) Scan(v any) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("scan a media record from %T", v)
	}
	if err := json.Unmarshal([]byte(s), r); err != nil {
		return fmt.Errorf("scan a media record: %w", err)
	}
	return nil
}

// Probed is what a Target judges of a file, in the release vocabularies.
type Probed struct {
	Resolution release.Fact[release.Resolution]
	Video      release.Fact[release.VideoCodec]
	HDR        release.Fact[[]release.HDR]
	Audio      release.Fact[[]release.AudioCodec]
	Channels   release.Fact[[]release.Channels]
	// Languages are the audio tracks' languages, unknown while any track
	// is untagged. Subtitle languages are never judged.
	Languages release.Fact[[]release.Language]
}

// Probed is r in the release vocabularies. A Version 0 record knows only
// its resolution and video codec. Audio codecs and channel layouts the
// vocabularies have no word for are left out.
func (r Record) Probed() Probed {
	var p Probed
	if r.Width > 0 {
		p.Resolution = release.Known(r.Resolution())
	}
	p.Video = release.VideoCodecOf(r.Codec)
	if r.Version == 0 {
		return p
	}
	p.HDR = release.Known(r.HDR)
	var codecs []release.AudioCodec
	var channels []release.Channels
	var languages []release.Language
	untagged := false
	for _, a := range r.Audio {
		if c, ok := audioCodecOf(a); ok && !slices.Contains(codecs, c) {
			codecs = append(codecs, c)
		}
		if c, ok := layouts[a.Channels]; ok && !slices.Contains(channels, c) {
			channels = append(channels, c)
		}
		untagged = untagged || a.Language == ""
		if a.Language != "" && !slices.Contains(languages, a.Language) {
			languages = append(languages, a.Language)
		}
	}
	p.Audio, p.Channels = release.Known(codecs), release.Known(channels)
	if !untagged {
		p.Languages = release.Known(languages)
	}
	return p
}

var layouts = map[int]release.Channels{8: "7.1", 6: "5.1", 2: "2.0", 1: "1.0"}

var audioCodecs = map[string]release.AudioCodec{
	"truehd": "TrueHD", "eac3": "DD+", "ac3": "DD", "aac": "AAC", "flac": "FLAC", "opus": "Opus", "mp3": "MP3",
}

func audioCodecOf(a Audio) (release.AudioCodec, bool) {
	switch {
	case a.Codec == "dts" && strings.HasPrefix(a.Profile, "DTS-HD MA"):
		return "DTS-HD MA", true
	case a.Codec == "dts":
		return "DTS", true
	case strings.HasPrefix(a.Codec, "pcm_"):
		return "PCM", true
	}
	c, ok := audioCodecs[a.Codec]
	return c, ok
}
