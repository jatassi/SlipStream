package mediafix

import (
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/fakes/scenario"
)

// recipe is the commands that make one file in an empty folder: write
// files, then run steps, which leave out.
type recipe struct {
	files []file
	steps []step
	out   string
}

type file struct{ name, text string }

type step struct {
	tool string // ffmpeg or dovi_tool
	args []string
}

const (
	x265 = "pools=1:frame-threads=1:log-level=error"
	// masterDisplay is HDR10's static metadata: P3 primaries in a BT.2020
	// container, 1000 nits peak.
	masterDisplay = "master-display=G(13250,34500)B(7500,3000)R(34000,16000)WP(15635,16450)L(10000000,1):max-cll=1000,400"
	// vbv is the rate control libx265 needs to write a Dolby Vision
	// configuration record.
	vbv = "vbv-bufsize=20000:vbv-maxrate=20000"
	pq  = "setparams=color_primaries=bt2020:color_trc=smpte2084:colorspace=bt2020nc"
	hlg = "setparams=color_primaries=bt2020:color_trc=arib-std-b67:colorspace=bt2020nc"
)

const fps = 24

var bitexact = []string{"-fflags", "+bitexact", "-flags:v", "+bitexact", "-flags:a", "+bitexact", "-map_metadata", "-1"}

// recipeOf is the recipe that makes f, naming every file it writes after
// stem. The bytes it makes don't depend on stem.
func recipeOf(stem string, f scenario.MediaFile) recipe {
	v := f.Video
	ext := path.Ext(f.Path)
	r := recipe{out: stem + ext}
	frames := strconv.Itoa(v.Seconds * fps)
	source := []string{"-f", "lavfi", "-i", fmt.Sprintf("testsrc2=size=%dx%d:rate=%d:duration=%d", v.Width, v.Height, fps, v.Seconds)}
	var codec []string
	switch v.HDR {
	case "sdr":
		codec = []string{"-c:v", "libx264", "-preset", "ultrafast", "-threads", "1"}
	case "hdr10":
		codec = hevc(pq+",format=yuv420p10le", x265+":hdr10=1:"+masterDisplay)
	case "hdr10plus":
		plus := stem + ".hdr10plus.json"
		r.files = append(r.files, file{plus, hdr10PlusJSON(v.Seconds * fps)})
		codec = hevc(pq+",format=yuv420p10le", x265+":hdr10=1:dhdr10-info="+plus+":"+masterDisplay)
	case "hlg":
		codec = hevc(hlg+",format=yuv420p10le", x265)
	case "dv8":
		source = r.dolbyVision(stem, source,
			hevc(pq+",format=yuv420p10le", x265+":hdr10=1:repeat-headers=1:"+masterDisplay),
			`{"cm_version":"V40","profile":"8.1","length":`+frames+`,"level6":{"max_display_mastering_luminance":1000,"min_display_mastering_luminance":1,"max_content_light_level":1000,"max_frame_average_light_level":400}}`)
		codec = append(hevc(pq, x265+":hdr10=1:"+vbv+":"+masterDisplay), "-dolbyvision", "1")
	case "dv5":
		source = r.dolbyVision(stem, source, hevc("format=yuv420p10le", x265+":repeat-headers=1"), `{"cm_version":"V40","profile":"5","length":`+frames+`}`)
		codec = append(hevc("", x265+":"+vbv), "-dolbyvision", "1")
	}
	args := append([]string{"-v", "error", "-y"}, source...)
	maps := []string{"-map", "0:v"}
	var tracks []string
	subtitles := 1
	if len(f.Audio) > 0 {
		subtitles = 2
		args = append(args, "-f", "lavfi", "-i", fmt.Sprintf("sine=frequency=440:sample_rate=48000:duration=%d", v.Seconds))
		graph, audio := audioGraph(f.Audio)
		args = append(args, "-filter_complex", graph)
		maps = append(maps, audio...)
		tracks = append(tracks, audioTracks(f.Audio)...)
	}
	for i, s := range f.Subtitles {
		srt := fmt.Sprintf("%s.%d.srt", stem, i+1)
		r.files = append(r.files, file{srt, fmt.Sprintf("1\n00:00:00,000 --> 00:00:01,000\nSubtitle %d\n", i+1)})
		args = append(args, "-i", srt)
		maps = append(maps, "-map", strconv.Itoa(subtitles+i))
		tracks = append(tracks, subtitleTrack(i, s)...)
	}
	if len(f.Subtitles) > 0 {
		tracks = append(tracks, "-c:s", "srt")
	}
	args = slices.Concat(args, maps, codec, tracks, bitexact)
	if ext == ".h264" {
		args = append(args, "-f", "h264")
	}
	r.steps = append(r.steps, step{"ffmpeg", append(args, r.out)})
	return r
}

func hevc(filter, params string) []string {
	var args []string
	if filter != "" {
		args = []string{"-vf", filter}
	}
	return append(args, "-c:v", "libx265", "-preset", "ultrafast", "-x265-params", params)
}

// dolbyVision adds the steps that make a stream with the RPU rpu
// describes: an HEVC base layer, the RPU injected into it, then a libx265
// re-encode with -dolbyvision 1, since a remux of the injected stream
// carries no configuration record. It answers the re-encode's input.
func (r *recipe) dolbyVision(stem string, source, base []string, rpu string) []string {
	baseFile, rpuJSON, rpuFile, injected := stem+"-base.hevc", stem+".dovi.json", stem+".rpu", stem+"-rpu.hevc"
	r.files = append(r.files, file{rpuJSON, rpu + "\n"})
	r.steps = append(r.steps,
		step{"ffmpeg", slices.Concat([]string{"-v", "error", "-y"}, source, base, []string{"-fflags", "+bitexact", "-flags:v", "+bitexact", "-f", "hevc", baseFile})},
		step{"dovi_tool", []string{"generate", "-j", rpuJSON, "-o", rpuFile}},
		step{"dovi_tool", []string{"inject-rpu", "-i", baseFile, "--rpu-in", rpuFile, "-o", injected}},
	)
	return []string{"-r", strconv.Itoa(fps), "-i", injected}
}

// hdr10PlusJSON is x265's dhdr10-info for n frames: one scene, and the
// same SceneInfo for each frame, as x265 reads one per frame.
func hdr10PlusJSON(n int) string {
	scenes := make([]string, n)
	for i := range n {
		scenes[i] = fmt.Sprintf(`{"BezierCurveData":{"Anchors":[256,512,768],"KneePointX":0,"KneePointY":0},"LuminanceParameters":{"AverageRGB":1000,"LuminanceDistributions":{"DistributionIndex":[1,5,10,25,50,75,90,95,99],"DistributionValues":[0,100,200,300,400,500,600,700,800]},"MaxScl":[1000,1000,1000]},"NumberOfWindows":1,"TargetedSystemDisplayMaximumLuminance":400,"SceneFrameIndex":%d,"SceneId":0,"SequenceFrameIndex":%d}`, i, i)
	}
	return fmt.Sprintf(`{"JSONInfo":{"HDR10plusProfile":"A","Version":"1.0"},"SceneInfo":[%s],"SceneInfoSummary":{"SceneFirstFrameIndex":[0],"SceneFrameNumbers":[%d]},"ToolInfo":{"Tool":"slipstream","Version":"1"}}`+"\n", strings.Join(scenes, ","), n)
}

// layouts are the channel layouts a track of each channel count gets.
// FFmpeg's AAC encoder writes 5.1(side) as an unknown layout, so AAC's
// six channels are 5.1.
var layouts = map[int]string{1: "mono", 2: "stereo", 6: "5.1(side)", 8: "7.1"}

func layoutOf(a scenario.MediaAudio) string {
	if a.Codec == "aac" && a.Channels == 6 {
		return "5.1"
	}
	return layouts[a.Channels]
}

// audioGraph splits input 1's tone into one pan per track, and answers
// the graph and each track's -map.
func audioGraph(audio []scenario.MediaAudio) (graph string, maps []string) {
	var split, pans strings.Builder
	split.WriteString("[1:a]asplit=" + strconv.Itoa(len(audio)))
	for i, a := range audio {
		fmt.Fprintf(&split, "[s%d]", i)
		fmt.Fprintf(&pans, ";[s%d]pan=%s", i, layoutOf(a))
		for c := range a.Channels {
			fmt.Fprintf(&pans, "|c%d=c0", c)
		}
		fmt.Fprintf(&pans, "[a%d]", i)
		maps = append(maps, "-map", fmt.Sprintf("[a%d]", i))
	}
	return split.String() + pans.String(), maps
}

func audioTracks(audio []scenario.MediaAudio) []string {
	var args []string
	experimental := false
	for i, a := range audio {
		n := strconv.Itoa(i)
		args = append(args, "-c:a:"+n, a.Codec)
		if a.Language != "" {
			args = append(args, "-metadata:s:a:"+n, "language="+a.Language)
		}
		if a.Title != "" {
			args = append(args, "-metadata:s:a:"+n, "title="+a.Title)
		}
		args = append(args, "-disposition:a:"+n, disposition(a.Default, false))
		experimental = experimental || a.Codec == "truehd"
	}
	if experimental {
		args = append(args, "-strict", "-2")
	}
	return args
}

func subtitleTrack(i int, s scenario.MediaSubtitle) []string {
	n := strconv.Itoa(i)
	var args []string
	if s.Language != "" {
		args = append(args, "-metadata:s:s:"+n, "language="+s.Language)
	}
	return append(args, "-disposition:s:"+n, disposition(s.Default, s.Forced))
}

func disposition(def, forced bool) string {
	switch {
	case def && forced:
		return "default+forced"
	case def:
		return "default"
	case forced:
		return "forced"
	}
	return "0"
}
