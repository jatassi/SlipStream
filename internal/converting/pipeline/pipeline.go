// Package pipeline turns a Conversion into ffmpeg's arguments on the Node
// that runs it. Rows are the pipelines a test run tries, in proof order,
// and Build replays the row a proof names. It is pure.
package pipeline

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
)

type Job struct {
	// Input and Output are absolute local paths. Output is the .part name,
	// so the container comes from Settings.
	Input, Output string
	OS            string
	Settings      encode.Settings
	Source        media.Record
	// Keep is the Work's original language plus the Target's filter; nil
	// keeps every track.
	Keep     []language.Code
	Pipeline encode.Pipeline
	// Device is the adapter the proof opened, in its row's terms.
	Device string
}

// Row is one pipeline on the OSes it runs on.
type Row struct {
	Name     string
	OS       []string
	Pipeline encode.Pipeline
	Init     func(device string) []string
	// Decode is nil when the row, or the class, decodes on the processor.
	Decode  func(c encode.Class) []string
	Filter  func(n Need) string
	Encoder map[encode.Codec]string
	Rate    func(c encode.Codec, kbps int) []string
	// DoVi is whether the row's encoders write Dolby Vision metadata, so
	// Build says whether to.
	DoVi bool
}

// Color is a picture's color encoding: what tone mapping reads and writes.
type Color string

const (
	SDR Color = "sdr"
	PQ  Color = "pq"
	HLG Color = "hlg"
	// IPT is a Dolby Vision Profile 5 base layer, which only its metadata
	// can turn into a picture.
	IPT Color = "ipt"
)

// Need is what a row's video chain has to do.
type Need struct {
	W, H int
	// OnDevice is whether the frames arrive decoded on the row's device.
	OnDevice bool
	ToneMap  bool
	To       Color
	Curve    encode.Curve
	// DV is a Dolby Vision source, whose metadata the tone mapper applies.
	DV     bool
	Keep10 bool
}

// For is Rows on os, in proof order, that can encode s. Keep Dolby Vision
// gets the processor only, because no hardware encoder writes it.
func For(os string, s encode.Shape) []Row {
	var rows []Row
	for _, r := range Rows {
		_, encodes := r.Encoder[s.Codec]
		if slices.Contains(r.OS, os) && encodes && (!s.KeepDV || r.Pipeline.Path == encode.Processor) {
			rows = append(rows, r)
		}
	}
	return rows
}

func rowOf(os string, p encode.Pipeline) (Row, bool) {
	for _, r := range Rows {
		if r.Pipeline == p && slices.Contains(r.OS, os) {
			return r, true
		}
	}
	return Row{}, false
}

// Build is the argument list: -hide_banner -nostats -progress pipe:1
// -protocol_whitelist file,pipe -y, never -nostdin (stdin carries p and u),
// then the row, the input, the stream maps (encode.Tracks), the video
// (copy, or the row's chain and encoder), the audio, the subtitles,
// container flags and color tags, the output.
func Build(j Job) ([]string, error) {
	row, ok := rowOf(j.OS, j.Pipeline)
	if !ok {
		return nil, fmt.Errorf("no %s %s pipeline on %s", j.Pipeline.Path, j.Pipeline.API, j.OS)
	}
	s, m := j.Settings, j.Source
	copied := encode.CopiesVideo(m, s)
	encoder, encodes := row.Encoder[s.Codec]
	if !copied && !encodes {
		return nil, fmt.Errorf("the %s pipeline has no %s encoder", row.Name, s.Codec)
	}
	args := []string{"-hide_banner", "-nostats", "-progress", "pipe:1", "-protocol_whitelist", "file,pipe", "-y"}
	var decode []string
	if !copied {
		args = append(args, row.Init(j.Device)...)
		if row.Decode != nil {
			decode = row.Decode(encode.ClassOf(m))
		}
		args = append(args, decode...)
	}
	args = append(args, "-i", j.Input)

	sel := encode.Tracks(m, s, j.Keep)
	args = append(args, "-map", "0:V:0")
	for _, i := range sel.Audio {
		args = append(args, "-map", "0:a:"+strconv.Itoa(i))
		if s.Audio == encode.AC3AndStereoAAC {
			args = append(args, "-map", "0:a:"+strconv.Itoa(i))
		}
	}
	for _, i := range sel.Subtitles {
		args = append(args, "-map", "0:s:"+strconv.Itoa(i))
	}

	from, to := colors(m, s)
	if copied {
		args = append(args, "-c:v", "copy")
	} else {
		w, h := encode.Out(m, s)
		n := Need{
			W: w, H: h, OnDevice: decode != nil, ToneMap: from != to, To: to, Curve: s.Curve,
			DV: m.DolbyVision != nil, Keep10: s.Depth == encode.Bit10,
		}
		args = append(args, "-vf", row.Filter(n), "-c:v", encoder)
		args = append(args, profile(s)...)
		args = append(args, row.Rate(s.Codec, encode.Kbps(m, s))...)
		if row.DoVi && m.DolbyVision != nil && s.Codec != encode.H264 {
			args = append(args, "-dolbyvision", strconv.Itoa(boolInt(keepsDV(m, s))))
		}
		args = append(args, tags[to]...)
	}
	if len(sel.Audio) > 0 {
		args = append(args, audio[s.Audio]...)
	}
	if len(sel.Subtitles) > 0 {
		args = append(args, "-c:s", subtitles[s.Container])
	}
	args = append(args, container(s, m)...)
	return append(args, j.Output), nil
}

// colors is the source's color and the one s writes. An SDR source stays
// SDR, a kept Dolby Vision stream keeps its base layer, an HLG source keeps
// HLG only for the source's range, and every other HDR output is PQ.
func colors(m media.Record, s encode.Settings) (from, to Color) {
	from = colorOf(m)
	switch {
	case s.Range == encode.SDR || from == SDR:
		return from, SDR
	case keepsDV(m, s):
		return from, from
	case s.Range == encode.SourceRange && from == HLG:
		return from, HLG
	}
	return from, PQ
}

func colorOf(m media.Record) Color {
	has := func(h release.HDR) bool { return slices.Contains(m.HDR, h) }
	switch {
	case m.DolbyVision != nil && m.DolbyVision.Profile == 5:
		return IPT
	case has("HLG") && !has("HDR10") && !has("HDR10+"):
		return HLG
	case m.DolbyVision != nil || has("HDR10") || has("HDR10+"):
		return PQ
	}
	return SDR
}

// keepsDV is whether the output carries the source's Dolby Vision. Profile
// 7's enhancement layer is never kept.
func keepsDV(m media.Record, s encode.Settings) bool {
	return s.KeepDV && m.DolbyVision != nil && m.DolbyVision.Profile != 7
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ToneMap is the one tone mapping fragment every row uses: r=tv always,
// and apply_dovi=1 for a Dolby Vision source, never apply_dovi=0 (it
// aborts ffmpeg at vf_tonemap_opencl.c:592 on Profile 5). format is the
// filter's own name for the output pixel format.
func ToneMap(n Need, filter, format string) string {
	t := tonemapTo[n.To]
	f := fmt.Sprintf("%s=format=%s:p=%s:t=%s:m=%s:tonemap=%s", filter, format, t.primaries, t.transfer, t.matrix, n.Curve)
	if n.To == SDR {
		f += ":peak=100"
	}
	f += ":desat=0"
	if n.DV {
		f += ":apply_dovi=1"
	}
	return f + ":r=tv"
}

var tonemapTo = map[Color]struct{ primaries, transfer, matrix string }{
	SDR: {"bt709", "bt709", "bt709"},
	PQ:  {"bt2020", "smpte2084", "bt2020"},
}

var tags = map[Color][]string{
	SDR: {"-color_primaries", "bt709", "-color_trc", "bt709", "-colorspace", "bt709", "-color_range", "tv"},
	PQ:  {"-color_primaries", "bt2020", "-color_trc", "smpte2084", "-colorspace", "bt2020nc", "-color_range", "tv"},
	HLG: {"-color_primaries", "bt2020", "-color_trc", "arib-std-b67", "-colorspace", "bt2020nc", "-color_range", "tv"},
}

func profile(s encode.Settings) []string {
	switch {
	case s.Codec == encode.H264:
		return []string{"-profile:v", "high"}
	case s.Codec == encode.HEVC && s.Depth == encode.Bit10:
		return []string{"-profile:v", "main10"}
	case s.Codec == encode.HEVC:
		return []string{"-profile:v", "main"}
	}
	return nil
}

// audio is the audio encode per choice, at the nominal bitrates
// encode.Bytes estimates with.
var audio = map[encode.Audio][]string{
	encode.CopyAudio: {"-c:a", "copy"},
	encode.StereoAAC: {"-c:a", "aac", "-ac", "2", "-b:a", "192k", "-disposition:a:0", "default"},
	encode.AC3AndStereoAAC: {
		"-c:a:0", "ac3", "-ac:a:0", "6", "-b:a:0", "576k", "-c:a:1", "aac", "-ac:a:1", "2", "-b:a:1", "192k",
		"-disposition:a:0", "default", "-disposition:a:1", "0",
	},
}

// subtitles is the subtitle encode per container. encode.Tracks has
// already dropped the image tracks MP4 can't hold.
var subtitles = map[encode.Container]string{encode.MKV: "copy", encode.MP4: "mov_text"}

func container(s encode.Settings, m media.Record) []string {
	if s.Container == encode.MKV {
		return []string{"-f", "matroska"}
	}
	args := []string{"-f", "mp4", "-movflags", "+faststart"}
	if s.Codec == encode.HEVC {
		args = append(args, "-tag:v", "hvc1")
	}
	if keepsDV(m, s) {
		args = append(args, "-strict", "unofficial")
	}
	return args
}
