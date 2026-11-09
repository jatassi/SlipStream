package media

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/release"
)

type sideData struct {
	Type          string `json:"side_data_type"`
	Profile       int    `json:"dv_profile"`
	Level         int    `json:"dv_level"`
	Compatibility int    `json:"dv_bl_signal_compatibility_id"`
}

type disposition struct {
	Default     int `json:"default"`
	Forced      int `json:"forced"`
	AttachedPic int `json:"attached_pic"`
}

type stream struct {
	CodecType     string            `json:"codec_type"`
	CodecName     string            `json:"codec_name"`
	Profile       string            `json:"profile"`
	Width         int               `json:"width"`
	Height        int               `json:"height"`
	PixFmt        string            `json:"pix_fmt"`
	ColorTransfer string            `json:"color_transfer"`
	FrameRate     string            `json:"r_frame_rate"`
	Channels      int               `json:"channels"`
	Layout        string            `json:"channel_layout"`
	Disposition   disposition       `json:"disposition"`
	Tags          map[string]string `json:"tags"`
	SideData      []sideData        `json:"side_data_list"`
}

type streamsOutput struct {
	Format struct {
		FormatName string `json:"format_name"`
		Duration   string `json:"duration"`
		Bitrate    string `json:"bit_rate"`
	} `json:"format"`
	Streams []stream `json:"streams"`
}

type frameOutput struct {
	Frames []struct {
		ColorTransfer string     `json:"color_transfer"`
		SideData      []sideData `json:"side_data_list"`
	} `json:"frames"`
}

const (
	doviRecord = "DOVI configuration record"
	hdr10Plus  = "HDR Dynamic Metadata SMPTE2094-40 (HDR10+)"
)

// imageSubtitles are the bitmap subtitle codecs.
var imageSubtitles = []string{"hdmv_pgs_subtitle", "dvd_subtitle", "dvb_subtitle", "xsub"}

// Parse reads the JSON of FFprobe's two runs: streams is -show_format
// -show_streams, and frame is -select_streams v:0 -show_frames
// -read_intervals "%+#1", the first video frame. HDR10+ is only in a
// frame's side data, and a Matroska file muxed without its color element
// states its transfer only in its frames. A file with no video stream is
// an *UnreadableError.
func Parse(streams, frame []byte) (Record, error) {
	var doc streamsOutput
	if err := json.Unmarshal(streams, &doc); err != nil {
		return Record{}, fmt.Errorf("read ffprobe's streams: %w", err)
	}
	var first frameOutput
	if err := json.Unmarshal(frame, &first); err != nil {
		return Record{}, fmt.Errorf("read ffprobe's first frame: %w", err)
	}
	r := Record{Version: Version, Container: doc.Format.FormatName}
	var err error
	if r.Duration, err = seconds(doc.Format.Duration); err != nil {
		return Record{}, err
	}
	if doc.Format.Bitrate != "" {
		if r.Bitrate, err = strconv.ParseInt(doc.Format.Bitrate, 10, 64); err != nil {
			return Record{}, fmt.Errorf("read ffprobe's bit rate %q: %w", doc.Format.Bitrate, err)
		}
	}
	found := false
	for _, s := range doc.Streams {
		switch {
		case s.CodecType == "video" && !found && s.Disposition.AttachedPic == 0:
			found = true
			if r.Video, err = videoOf(s, first); err != nil {
				return Record{}, err
			}
		case s.CodecType == "audio":
			r.Audio = append(r.Audio, Audio{
				Codec: s.CodecName, Profile: s.Profile, Atmos: strings.Contains(s.Profile, "Atmos"), Channels: s.Channels, Layout: s.Layout,
				Language: languageOf(s.Tags), Default: s.Disposition.Default == 1, Title: s.Tags["title"],
			})
		case s.CodecType == "subtitle":
			r.Subtitles = append(r.Subtitles, Subtitle{
				Codec: s.CodecName, Image: slices.Contains(imageSubtitles, s.CodecName), Language: languageOf(s.Tags),
				Forced: s.Disposition.Forced == 1, Default: s.Disposition.Default == 1,
			})
		}
	}
	if !found {
		return Record{}, &UnreadableError{Line: "no video stream"}
	}
	return r, nil
}

func seconds(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("read ffprobe's duration %q: %w", s, err)
	}
	return time.Duration(f * float64(time.Second)).Round(time.Millisecond), nil
}

func videoOf(s stream, first frameOutput) (Video, error) {
	v := Video{Codec: s.CodecName, Profile: s.Profile, Width: s.Width, Height: s.Height, BitDepth: bitDepthOf(s.PixFmt)}
	if s.FrameRate != "" {
		if _, err := fmt.Sscanf(s.FrameRate, "%d/%d", &v.FrameRate.Num, &v.FrameRate.Den); err != nil {
			return Video{}, fmt.Errorf("read ffprobe's frame rate %q: %w", s.FrameRate, err)
		}
	}
	for _, d := range s.SideData {
		if d.Type == doviRecord {
			v.DolbyVision = &DolbyVision{Profile: d.Profile, Level: d.Level, Compatibility: d.Compatibility}
		}
	}
	transfer, plus := s.ColorTransfer, false
	if len(first.Frames) > 0 {
		f := first.Frames[0]
		if transfer == "" || transfer == "unknown" {
			transfer = f.ColorTransfer
		}
		plus = slices.ContainsFunc(f.SideData, func(d sideData) bool { return d.Type == hdr10Plus })
	}
	v.HDR = hdrOf(transfer, v.DolbyVision, plus)
	return v, nil
}

// hdrOf is the HDR formats of a stream. A Dolby Vision stream's base layer
// is HDR10 or HLG only as its compatibility id says: Profile 5 states the
// PQ transfer of its IPT color space, which is not HDR10.
func hdrOf(transfer string, dv *DolbyVision, plus bool) []release.HDR {
	has := map[release.HDR]bool{"HDR10+": plus}
	if dv != nil {
		has["DV"] = true
		has["HDR10"] = dv.Compatibility == 1 || dv.Compatibility == 6
		has["HLG"] = dv.Compatibility == 4
	} else {
		has["HDR10"] = transfer == "smpte2084"
		has["HLG"] = transfer == "arib-std-b67"
	}
	var out []release.HDR
	for _, h := range release.HDRs() {
		if has[h] {
			out = append(out, h)
		}
	}
	if out == nil {
		return []release.HDR{"SDR"}
	}
	return out
}

var depth = regexp.MustCompile(`p0?(\d+)(le|be)$`)

// bitDepthOf reads a pixel format's bits per component, "yuv420p10le" is
// 10, and 8 for one that names none, "yuv420p".
func bitDepthOf(pixFmt string) int {
	if pixFmt == "" {
		return 0
	}
	if m := depth.FindStringSubmatch(pixFmt); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 8
}

func languageOf(tags map[string]string) release.Language {
	l, err := release.ParseLanguage(tags["language"])
	if err != nil {
		return ""
	}
	return l
}
