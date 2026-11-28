// Package encode is what a Conversion makes: a Target's Conversion
// settings, the shape and class a proof covers, and the made file's title
// facts, size and tracks before it exists. It is pure, so the Controller
// plans with it and a Worker builds ffmpeg's arguments from it.
package encode

import (
	"fmt"
	"slices"
)

type Codec string

const (
	H264 Codec = "h264"
	HEVC Codec = "hevc"
	AV1  Codec = "av1"
)

// Height is the largest output height. A wider picture fits the 16:9 box
// of that height.
type Height int

const (
	P2160 Height = 2160
	P1080 Height = 1080
	P720  Height = 720
	P480  Height = 480
)

type Depth int

const (
	Bit8  Depth = 8
	Bit10 Depth = 10
)

// Range is the output's dynamic range. SourceRange keeps the source's,
// with HDR10+ written as HDR10.
type Range string

const (
	SDR         Range = "sdr"
	HDR10       Range = "hdr10"
	SourceRange Range = "source"
)

// Curve is the tone mapping curve from HDR to SDR.
type Curve string

const (
	BT2390   Curve = "bt2390"
	Hable    Curve = "hable"
	Mobius   Curve = "mobius"
	Reinhard Curve = "reinhard"
)

type Audio string

const (
	CopyAudio       Audio = "copy"
	StereoAAC       Audio = "stereo_aac"
	AC3AndStereoAAC Audio = "ac3_51_and_stereo_aac"
)

type Container string

const (
	MKV Container = "mkv"
	MP4 Container = "mp4"
)

// Path is how a pipeline runs: decode, tone map and encode on one device,
// a hardware encode with OpenCL tone mapping, or the processor.
type Path string

const (
	Hardware       Path = "hardware"
	HardwareOpenCL Path = "hardware_opencl"
	Processor      Path = "processor"
)

// API is the hardware interface a pipeline uses, "" for the processor.
type API string

const (
	QSV          API = "qsv"
	VAAPI        API = "vaapi"
	CUDA         API = "cuda"
	VideoToolbox API = "videotoolbox"
)

// Pipeline is a proof row. The Node's OS picks among the rows of a Path
// and API.
type Pipeline struct {
	Path Path
	API  API
}

// Settings is a Target's Conversion settings. Parse is the only way to make
// one: it refuses depth 10 with h264, Keep Dolby Vision with sdr or h264,
// and a bitrate outside 100 to 200,000 kbps.
type Settings struct {
	Codec     Codec
	MaxHeight Height
	// Kbps is the video bitrate at MaxHeight.
	Kbps           int
	Depth          Depth
	Range          Range
	KeepDV         bool
	Curve          Curve
	Audio          Audio
	DropCommentary bool
	Container      Container
}

// Raw is Conversion settings as a request or a row gives them.
type Raw struct {
	Codec          string
	MaxHeight      int
	Kbps           int
	Depth          int
	Range          string
	KeepDV         bool
	Curve          string
	Audio          string
	DropCommentary bool
	Container      string
}

const (
	MinKbps = 100
	MaxKbps = 200_000
)

// FieldError is a refused field of Raw, by its name in Raw.
type FieldError struct {
	Field  string
	Reason string
}

func (e *FieldError) Error() string { return e.Reason }

func Parse(raw Raw) (Settings, error) {
	s := Settings{
		Codec: Codec(raw.Codec), MaxHeight: Height(raw.MaxHeight), Kbps: raw.Kbps, Depth: Depth(raw.Depth),
		Range: Range(raw.Range), KeepDV: raw.KeepDV, Curve: Curve(raw.Curve), Audio: Audio(raw.Audio),
		DropCommentary: raw.DropCommentary, Container: Container(raw.Container),
	}
	checks := []struct {
		field  string
		ok     bool
		reason string
	}{
		{"Codec", slices.Contains([]Codec{H264, HEVC, AV1}, s.Codec), fmt.Sprintf("%q isn't a video codec", raw.Codec)},
		{"MaxHeight", slices.Contains([]Height{P2160, P1080, P720, P480}, s.MaxHeight), fmt.Sprintf("%d isn't a largest resolution", raw.MaxHeight)},
		{"Kbps", s.Kbps >= MinKbps && s.Kbps <= MaxKbps, fmt.Sprintf("a bitrate of %d kbps is outside 100 to 200,000 kbps", raw.Kbps)},
		{"Depth", slices.Contains([]Depth{Bit8, Bit10}, s.Depth), fmt.Sprintf("%d isn't a bit depth", raw.Depth)},
		{"Depth", s.Depth != Bit10 || s.Codec != H264, "10 bit needs HEVC or AV1"},
		{"Range", slices.Contains([]Range{SDR, HDR10, SourceRange}, s.Range), fmt.Sprintf("%q isn't a dynamic range", raw.Range)},
		{"KeepDV", !s.KeepDV || (s.Range != SDR && s.Codec != H264), "keeping Dolby Vision needs HEVC or AV1 and an HDR range"},
		{"Curve", slices.Contains([]Curve{BT2390, Hable, Mobius, Reinhard}, s.Curve), fmt.Sprintf("%q isn't a tone mapping curve", raw.Curve)},
		{"Audio", slices.Contains([]Audio{CopyAudio, StereoAAC, AC3AndStereoAAC}, s.Audio), fmt.Sprintf("%q isn't an audio choice", raw.Audio)},
		{"Container", slices.Contains([]Container{MKV, MP4}, s.Container), fmt.Sprintf("%q isn't a container", raw.Container)},
	}
	for _, c := range checks {
		if !c.ok {
			return Settings{}, &FieldError{Field: c.field, Reason: c.reason}
		}
	}
	return s, nil
}

// Defaults is a Target's Conversion settings when they are turned on. Kbps
// is zero: targets.DefaultKbps gives it from the Target's size range.
func Defaults() Settings {
	return Settings{
		Codec: H264, MaxHeight: P1080, Depth: Bit8, Range: SDR, Curve: BT2390,
		Audio: CopyAudio, DropCommentary: true, Container: MKV,
	}
}

// Shape is what a proof covers of Settings. Bitrate, resolution, audio and
// container don't change whether a pipeline works.
type Shape struct {
	Codec  Codec
	Depth  Depth
	Range  Range
	KeepDV bool
	Curve  Curve
}

func (s Settings) Shape() Shape {
	return Shape{Codec: s.Codec, Depth: s.Depth, Range: s.Range, KeepDV: s.KeepDV, Curve: s.Curve}
}

// String is the stored key, "h264.8.sdr.bt2390", with ".dv" when it keeps
// Dolby Vision.
func (s Shape) String() string {
	key := fmt.Sprintf("%s.%d.%s.%s", s.Codec, s.Depth, s.Range, s.Curve)
	if s.KeepDV {
		key += ".dv"
	}
	return key
}

// Key is what one proof covers.
type Key struct {
	Shape Shape
	Class Class
}
