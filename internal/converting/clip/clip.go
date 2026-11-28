// Package clip makes the synthetic clips a test run converts: a short
// video of each source class, made on the Node with the bundled ffmpeg and
// dovi_tool, byte for byte the same on every run.
package clip

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/encode"
)

type Size struct{ W, H, Seconds int }

// Trial is a test run's clip.
var Trial = Size{320, 180, 2}

const FPS = 24

// Tools are the bundled programs a clip is made with. Environ is their
// environment: the Node's, less every SLIPSTREAM_ variable.
type Tools struct {
	FFmpeg, FFprobe, DoviTool string
	Environ                   []string
}

type Tool string

const (
	FFmpeg   Tool = "ffmpeg"
	DoviTool Tool = "dovi_tool"
)

type Step struct {
	Tool Tool
	Args []string
}

type File struct{ Name, Text string }

// Video is how a clip's video stream is made in a folder: write Files, run
// Before, then one ffmpeg command reads Input and encodes it with Codec.
// Every name is relative to the folder.
type Video struct {
	Files  []File
	Before []Step
	Input  []string
	Codec  []string
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

var x264 = []string{"-c:v", "libx264", "-preset", "ultrafast", "-threads", "1"}

// VideoOf is the video of class c at size s, its files named after stem.
// A decode class is a codec the pipelines decode only on the processor:
// MPEG-2 for SDR, and 10-bit H.264 for HDR10.
func VideoOf(c encode.Class, s Size, stem string) Video {
	v := Video{Input: testsrc(s)}
	frames := strconv.Itoa(s.Seconds * FPS)
	switch c {
	case encode.ClassH264x8:
		v.Codec = x264
	case encode.ClassHEVCx8:
		v.Codec = hevc("format=yuv420p", x265)
	case encode.ClassHEVCx10:
		v.Codec = hevc("format=yuv420p10le", x265)
	case encode.ClassAV1x8:
		v.Codec = av1("format=yuv420p")
	case encode.ClassAV1x10:
		v.Codec = av1("format=yuv420p10le")
	case encode.ClassHDR10:
		v.Codec = hevc(pq+",format=yuv420p10le", x265+":hdr10=1:"+masterDisplay)
	case encode.ClassHLG:
		v.Codec = hevc(hlg+",format=yuv420p10le", x265)
	case encode.ClassDV81:
		v.dolbyVision(stem,
			hevc(pq+",format=yuv420p10le", x265+":hdr10=1:repeat-headers=1:"+masterDisplay),
			`{"cm_version":"V40","profile":"8.1","length":`+frames+`,"level6":{"max_display_mastering_luminance":1000,"min_display_mastering_luminance":1,"max_content_light_level":1000,"max_frame_average_light_level":400}}`)
		v.Codec = append(hevc(pq, x265+":hdr10=1:"+vbv+":"+masterDisplay), "-dolbyvision", "1")
	case encode.ClassDV5:
		v.dolbyVision(stem, hevc("format=yuv420p10le", x265+":repeat-headers=1"), `{"cm_version":"V40","profile":"5","length":`+frames+`}`)
		v.Codec = append(hevc("", x265+":"+vbv), "-dolbyvision", "1")
	case encode.ClassDecodeSDR:
		v.Codec = []string{"-c:v", "mpeg2video"}
	case encode.ClassDecodeHDR10:
		v.Codec = slices.Concat([]string{"-vf", pq + ",format=yuv420p10le"}, x264, []string{"-profile:v", "high10"})
	}
	return v
}

// HDR10Plus is the HDR10 video at size s with HDR10+ dynamic metadata,
// its files named after stem. The bundled Windows x265 can't write it, so
// no class proves with it.
func HDR10Plus(s Size, stem string) Video {
	plus := stem + ".hdr10plus.json"
	return Video{
		Files: []File{{Name: plus, Text: hdr10PlusJSON(s.Seconds * FPS)}},
		Input: testsrc(s),
		Codec: hevc(pq+",format=yuv420p10le", x265+":hdr10=1:dhdr10-info="+plus+":"+masterDisplay),
	}
}

func testsrc(s Size) []string {
	return []string{"-f", "lavfi", "-i", fmt.Sprintf("testsrc2=size=%dx%d:rate=%d:duration=%d", s.W, s.H, FPS, s.Seconds)}
}

func hevc(filter, params string) []string {
	var args []string
	if filter != "" {
		args = []string{"-vf", filter}
	}
	return append(args, "-c:v", "libx265", "-preset", "ultrafast", "-x265-params", params)
}

func av1(filter string) []string {
	return []string{"-vf", filter, "-c:v", "libsvtav1", "-preset", "12"}
}

// dolbyVision adds the steps that make a stream with the RPU rpu
// describes: an HEVC base layer, the RPU injected into it, then a libx265
// re-encode with -dolbyvision 1, since a remux of the injected stream
// carries no configuration record. The re-encode reads the injected
// stream.
func (v *Video) dolbyVision(stem string, base []string, rpu string) {
	baseFile, rpuJSON, rpuFile, injected := stem+"-base.hevc", stem+".dovi.json", stem+".rpu", stem+"-rpu.hevc"
	v.Files = append(v.Files, File{Name: rpuJSON, Text: rpu + "\n"})
	v.Before = append(v.Before,
		Step{FFmpeg, slices.Concat([]string{"-v", "error", "-y"}, v.Input, base, []string{"-fflags", "+bitexact", "-flags:v", "+bitexact", "-f", "hevc", baseFile})},
		Step{DoviTool, []string{"generate", "-j", rpuJSON, "-o", rpuFile}},
		Step{DoviTool, []string{"inject-rpu", "-i", baseFile, "--rpu-in", rpuFile, "-o", injected}},
	)
	v.Input = []string{"-r", strconv.Itoa(FPS), "-i", injected}
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

var bitexact = []string{"-fflags", "+bitexact", "-flags:v", "+bitexact", "-map_metadata", "-1"}

// Make makes the clip of class c at size s in dir and answers its path,
// <dir>/<c>.mkv.
func Make(ctx context.Context, t Tools, c encode.Class, s Size, dir string) (string, error) {
	stem := string(c)
	v := VideoOf(c, s, stem)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("create the clips folder: %w", err)
	}
	for _, f := range v.Files {
		if err := os.WriteFile(filepath.Join(dir, f.Name), []byte(f.Text), 0o600); err != nil {
			return "", fmt.Errorf("write the %s clip's %s: %w", c, f.Name, err)
		}
	}
	out := stem + ".mkv"
	steps := slices.Concat(v.Before, []Step{{FFmpeg, slices.Concat([]string{"-v", "error", "-y"}, v.Input, v.Codec, bitexact, []string{out})}})
	programs := map[Tool]string{FFmpeg: t.FFmpeg, DoviTool: t.DoviTool}
	for _, step := range steps {
		cmd := exec.CommandContext(ctx, programs[step.Tool], step.Args...) //nolint:gosec // the bundled ffmpeg or dovi_tool on a clip recipe
		cmd.Dir, cmd.Env = dir, t.Environ
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("make the %s clip: %s: %w: %s", c, step.Tool, err, bytes.TrimSpace(stderr.Bytes()))
		}
	}
	return filepath.Join(dir, out), nil
}
