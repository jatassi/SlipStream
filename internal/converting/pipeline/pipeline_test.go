package pipeline_test

import (
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/converting/pipeline"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
)

var update = flag.Bool("update", false, "rewrite testdata/<row>.golden from the cases table")

func video(codec string, w, h, depth int, hdr []release.HDR, dv *media.DolbyVision) media.Record {
	return media.Record{
		Version: media.Version, Container: "matroska,webm",
		Video: media.Video{Codec: codec, Width: w, Height: h, BitDepth: depth, HDR: hdr, DolbyVision: dv},
		Audio: []media.Audio{
			{Codec: "truehd", Channels: 8, Language: "en", Default: true},
			{Codec: "ac3", Channels: 6, Language: "en", Title: "Commentary"},
		},
		Subtitles: []media.Subtitle{
			{Codec: "hdmv_pgs_subtitle", Image: true, Language: "en"},
			{Codec: "subrip", Language: "en"},
		},
	}
}

var sources = map[string]media.Record{
	"hdr10 2160p":   video("hevc", 3840, 2160, 10, []release.HDR{"HDR10"}, nil),
	"hdr10+ 2160p":  video("hevc", 3840, 2160, 10, []release.HDR{"HDR10+", "HDR10"}, nil),
	"hlg 2160p":     video("hevc", 3840, 2160, 10, []release.HDR{"HLG"}, nil),
	"dv5 1080p":     video("hevc", 1920, 1080, 10, []release.HDR{"DV"}, &media.DolbyVision{Profile: 5, Level: 6}),
	"dv81 2160p":    video("hevc", 3840, 2160, 10, []release.HDR{"DV", "HDR10"}, &media.DolbyVision{Profile: 8, Level: 6, Compatibility: 1}),
	"dv7 2160p":     video("hevc", 3840, 2160, 10, []release.HDR{"DV", "HDR10"}, &media.DolbyVision{Profile: 7, Level: 6, Compatibility: 6}),
	"h264 1080p":    video("h264", 1920, 1080, 8, []release.HDR{"SDR"}, nil),
	"av1 10 hdr10":  video("av1", 3840, 2160, 10, []release.HDR{"HDR10"}, nil),
	"mpeg2 480p":    video("mpeg2video", 720, 480, 8, []release.HDR{"SDR"}, nil),
	"h264 10 hdr10": video("h264", 3840, 2160, 10, []release.HDR{"HDR10"}, nil),
}

func settings(t *testing.T, edit func(r *encode.Raw)) encode.Settings {
	t.Helper()
	r := encode.Raw{
		Codec: "h264", MaxHeight: 1080, Kbps: 8000, Depth: 8, Range: "sdr", Curve: "bt2390",
		Audio: "copy", DropCommentary: true, Container: "mkv",
	}
	edit(&r)
	s, err := encode.Parse(r)
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	return s
}

type golden struct {
	name   string
	source string
	edit   func(r *encode.Raw)
}

var cases = []golden{
	{"hdr10 to h264 sdr", "hdr10 2160p", func(*encode.Raw) {}},
	{"hdr10+ to h264 sdr hable", "hdr10+ 2160p", func(r *encode.Raw) { r.Curve = "hable" }},
	{"hlg to h264 sdr", "hlg 2160p", func(*encode.Raw) {}},
	{"hlg to hevc 10 source", "hlg 2160p", func(r *encode.Raw) { r.Codec, r.Depth, r.Range = "hevc", 10, "source" }},
	{"dv5 to h264 sdr", "dv5 1080p", func(*encode.Raw) {}},
	{"dv5 to hevc 10 hdr10", "dv5 1080p", func(r *encode.Raw) { r.Codec, r.Depth, r.Range = "hevc", 10, "hdr10" }},
	{"dv81 to h264 sdr", "dv81 2160p", func(*encode.Raw) {}},
	{"dv81 keeping dv in mp4", "dv81 2160p", func(r *encode.Raw) {
		r.Codec, r.Depth, r.Range, r.KeepDV, r.Container = "hevc", 10, "source", true, "mp4"
	}},
	{"dv7 keeping dv", "dv7 2160p", func(r *encode.Raw) { r.Codec, r.Depth, r.Range, r.KeepDV = "hevc", 10, "hdr10", true }},
	{"hdr10 to hevc 10 hdr10", "hdr10 2160p", func(r *encode.Raw) { r.Codec, r.Depth, r.Range = "hevc", 10, "hdr10" }},
	{"av1 hdr10 to av1 10 hdr10", "av1 10 hdr10", func(r *encode.Raw) { r.Codec, r.Depth, r.Range = "av1", 10, "hdr10" }},
	{"h264 to hevc 720p stereo mp4", "h264 1080p", func(r *encode.Raw) {
		r.Codec, r.MaxHeight, r.Audio, r.Container = "hevc", 720, "stereo_aac", "mp4"
	}},
	{"h264 copied", "h264 1080p", func(r *encode.Raw) { r.Audio = "stereo_aac" }},
	{"mpeg2 decoded on the processor", "mpeg2 480p", func(r *encode.Raw) { r.Audio = "ac3_51_and_stereo_aac" }},
	{"10 bit h264 hdr10 decoded on the processor", "h264 10 hdr10", func(*encode.Raw) {}},
}

var devices = map[encode.API]string{encode.QSV: "/dev/dri/renderD128", encode.VAAPI: "/dev/dri/renderD128", encode.CUDA: "0"}

func job(row pipeline.Row, m media.Record, s encode.Settings) pipeline.Job {
	return pipeline.Job{
		Input: "/scratch/in/s-7.mkv", Output: "/scratch/out/c-9-1.mkv.part", OS: row.OS[0],
		Settings: s, Source: m, Pipeline: row.Pipeline, Device: devices[row.Pipeline.API],
	}
}

func quote(args []string) string {
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = a
		if strings.ContainsAny(a, " ,;\"") {
			out[i] = `"` + a + `"`
		}
	}
	return strings.Join(out, " ")
}

func render(t *testing.T, row pipeline.Row) string {
	t.Helper()
	var b strings.Builder
	for _, c := range cases {
		s := settings(t, c.edit)
		if !slices.ContainsFunc(pipeline.For(row.OS[0], s.Shape()), func(r pipeline.Row) bool { return r.Name == row.Name }) {
			continue
		}
		args, err := pipeline.Build(job(row, sources[c.source], s))
		if err != nil {
			t.Fatalf("%s, %s: %v", row.Name, c.name, err)
		}
		b.WriteString("# " + c.name + "\n" + quote(args) + "\n\n")
	}
	return b.String()
}

func TestGoldens(t *testing.T) {
	names := map[string]bool{}
	for _, row := range pipeline.Rows {
		names[row.Name+".golden"] = true
		want := render(t, row)
		path := filepath.Join("testdata", row.Name+".golden")
		if *update {
			if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Errorf("%s is stale; run go test ./internal/converting/pipeline -update", path)
		}
	}
	files, err := filepath.Glob(filepath.Join("testdata", "*.golden"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if !names[filepath.Base(f)] {
			t.Errorf("%s belongs to no row", f)
		}
	}
}

func adjacent(args []string, name, value string) bool {
	i := slices.Index(args, name)
	return i >= 0 && i+1 < len(args) && args[i+1] == value
}

func toneMappers(args []string) []string {
	i := slices.Index(args, "-vf")
	if i < 0 {
		return nil
	}
	var found []string
	for f := range strings.SplitSeq(args[i+1], ",") {
		if strings.HasPrefix(f, "tonemap") {
			found = append(found, f)
		}
	}
	return found
}

func every[V any](raws []encode.Raw, vs []V, set func(r *encode.Raw, v V)) []encode.Raw {
	var out []encode.Raw
	for _, r := range raws {
		for _, v := range vs {
			set(&r, v)
			out = append(out, r)
		}
	}
	return out
}

// TestEveryList builds every valid settings shape, container and audio
// choice from every source on every row its OS tries.
func TestEveryList(t *testing.T) {
	raws := []encode.Raw{{MaxHeight: 1080, Kbps: 6000, DropCommentary: true}}
	raws = every(raws, []string{"h264", "hevc", "av1"}, func(r *encode.Raw, v string) { r.Codec = v })
	raws = every(raws, []int{8, 10}, func(r *encode.Raw, v int) { r.Depth = v })
	raws = every(raws, []string{"sdr", "hdr10", "source"}, func(r *encode.Raw, v string) { r.Range = v })
	raws = every(raws, []bool{false, true}, func(r *encode.Raw, v bool) { r.KeepDV = v })
	raws = every(raws, []string{"bt2390", "hable", "mobius", "reinhard"}, func(r *encode.Raw, v string) { r.Curve = v })
	raws = every(raws, []string{"copy", "stereo_aac", "ac3_51_and_stereo_aac"}, func(r *encode.Raw, v string) { r.Audio = v })
	raws = every(raws, []string{"mkv", "mp4"}, func(r *encode.Raw, v string) { r.Container = v })
	built := 0
	for _, raw := range raws {
		if s, err := encode.Parse(raw); err == nil {
			for _, goos := range []string{"windows", "linux", "darwin"} {
				built += buildAll(t, goos, s)
			}
		}
	}
	if built == 0 {
		t.Fatal("built no list")
	}
}

func buildAll(t *testing.T, goos string, s encode.Settings) int {
	t.Helper()
	rows := pipeline.For(goos, s.Shape())
	if !slices.ContainsFunc(rows, func(r pipeline.Row) bool { return r.Pipeline.Path == encode.Processor }) {
		t.Fatalf("%s %s: no processor row", goos, s.Shape())
	}
	built := 0
	for _, row := range rows {
		if s.KeepDV && row.Pipeline.Path != encode.Processor {
			t.Fatalf("%s %s: keeping Dolby Vision offers %s", goos, s.Shape(), row.Name)
		}
		for name, m := range sources {
			args, err := pipeline.Build(pipeline.Job{
				Input: "/in.mkv", Output: "/out.part", OS: goos, Settings: s, Source: m,
				Pipeline: row.Pipeline, Device: "0",
			})
			if err != nil {
				t.Fatalf("%s %s from %s: %v", row.Name, s.Shape(), name, err)
			}
			built++
			check(t, row.Name+" "+s.Shape().String()+" from "+name, args, s, m)
		}
	}
	return built
}

func check(t *testing.T, what string, args []string, s encode.Settings, m media.Record) {
	t.Helper()
	if !adjacent(args, "-progress", "pipe:1") || !adjacent(args, "-protocol_whitelist", "file,pipe") {
		t.Fatalf("%s: no progress pipe or protocol whitelist: %q", what, args)
	}
	if slices.Contains(args, "-nostdin") || slices.ContainsFunc(args, func(a string) bool { return strings.Contains(a, "apply_dovi=0") }) {
		t.Fatalf("%s: -nostdin or apply_dovi=0: %q", what, args)
	}
	mappers := toneMappers(args)
	for _, f := range mappers {
		if !strings.Contains(f, ":r=tv") {
			t.Fatalf("%s: tone mapper without r=tv: %s", what, f)
		}
		if strings.Contains(f, "apply_dovi=1") != (m.DolbyVision != nil) {
			t.Fatalf("%s: apply_dovi=1 doesn't follow the Dolby Vision source: %s", what, f)
		}
	}
	hdr := !slices.Equal(m.HDR, []release.HDR{"SDR"})
	if s.Range == encode.SDR && hdr && len(mappers) == 0 {
		t.Fatalf("%s: an HDR source made SDR with no tone mapper", what)
	}
	if !hdr && len(mappers) > 0 {
		t.Fatalf("%s: an SDR source tone mapped", what)
	}
}

// TestRecordedChains holds the Windows Quick Sync row to the chains the
// recorded run proved (conformance/ffmpeg/recorded/2026-11-27-qsv), with
// r=tv on every tone mapper.
func TestRecordedChains(t *testing.T) {
	init := []string{
		"-init_hw_device", "d3d11va=dx11:,vendor=0x8086", "-init_hw_device", "qsv=qs@dx11", "-init_hw_device", "opencl=ocl@dx11",
		"-filter_hw_device", "ocl", "-hwaccel", "d3d11va", "-hwaccel_output_format", "d3d11",
	}
	const dovi = "hwmap=derive_device=opencl,tonemap_opencl=format=nv12:p=bt709:t=bt709:m=bt709:tonemap=bt2390:peak=100:desat=0:apply_dovi=1:r=tv,hwmap=derive_device=qsv:reverse=1"
	cases := []struct {
		name, source string
		edit         func(r *encode.Raw)
		vf           string
		encode       []string
	}{
		{
			"A hdr10 to sdr h264", "hdr10", func(r *encode.Raw) { r.Kbps = 4000 },
			"hwmap=derive_device=opencl,tonemap_opencl=format=nv12:p=bt709:t=bt709:m=bt709:tonemap=bt2390:peak=100:desat=0:r=tv,hwmap=derive_device=qsv:reverse=1",
			[]string{"-c:v", "h264_qsv", "-profile:v", "high", "-preset", "veryfast", "-b:v", "4000k", "-maxrate", "6000k", "-bufsize", "8000k"},
		},
		{
			"B dv5 to sdr h264", "dv5", func(r *encode.Raw) { r.Kbps = 4000 }, dovi,
			[]string{"-c:v", "h264_qsv", "-profile:v", "high", "-preset", "veryfast", "-b:v", "4000k", "-maxrate", "6000k", "-bufsize", "8000k"},
		},
		{
			"C hdr10 to hevc main 10", "hdr10 2160p", func(r *encode.Raw) { r.Codec, r.Depth, r.Range = "hevc", 10, "hdr10" },
			"hwmap=derive_device=qsv,format=qsv",
			[]string{"-c:v", "hevc_qsv", "-profile:v", "main10", "-preset", "veryfast", "-b:v", "8000k", "-maxrate", "12000k", "-bufsize", "16000k"},
		},
		{
			"D dv81 to sdr h264", "dv81", func(r *encode.Raw) { r.Kbps = 4000 }, dovi,
			[]string{"-c:v", "h264_qsv", "-profile:v", "high", "-preset", "veryfast", "-b:v", "4000k", "-maxrate", "6000k", "-bufsize", "8000k"},
		},
	}
	clips := map[string]media.Record{
		"hdr10":       video("hevc", 1920, 1080, 10, []release.HDR{"HDR10"}, nil),
		"hdr10 2160p": video("hevc", 3840, 2160, 10, []release.HDR{"HDR10"}, nil),
		"dv5":         video("hevc", 1920, 1080, 10, []release.HDR{"DV"}, &media.DolbyVision{Profile: 5}),
		"dv81":        video("hevc", 1920, 1080, 10, []release.HDR{"DV", "HDR10"}, &media.DolbyVision{Profile: 8, Compatibility: 1}),
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			args, err := pipeline.Build(pipeline.Job{
				Input: "in.mkv", Output: "out.part", OS: "windows", Settings: settings(t, c.edit), Source: clips[c.source],
				Pipeline: encode.Pipeline{Path: encode.Hardware, API: encode.QSV},
			})
			if err != nil {
				t.Fatal(err)
			}
			if i := slices.Index(args, "-init_hw_device"); i < 0 || !slices.Equal(args[i:i+len(init)], init) || args[i+len(init)] != "-i" {
				t.Fatalf("devices and decode before the input: %q", args)
			}
			vf := slices.Index(args, "-vf")
			if vf < 0 || !strings.HasPrefix(args[vf+1], c.vf) {
				t.Fatalf("video chain %q, want it to start %q", args[vf+1], c.vf)
			}
			if !slices.Equal(args[vf+2:vf+2+len(c.encode)], c.encode) {
				t.Fatalf("encode %q, want %q", args[vf+2:], c.encode)
			}
		})
	}
}

func codes(t *testing.T, s ...string) []language.Code {
	t.Helper()
	var out []language.Code
	for _, c := range s {
		code, err := language.Parse(c)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, code)
	}
	return out
}

func values(args []string, name string) []string {
	var out []string
	for i, a := range args {
		if a == name && i+1 < len(args) {
			out = append(out, args[i+1])
		}
	}
	return out
}

// TestStreams holds the stream maps to stories 11 and 12 and decision 28:
// audio and subtitles in the kept languages, untagged tracks kept,
// commentary dropped on request, a re-encode made from the default kept
// track, and MP4 converting text subtitles and dropping image ones.
func TestStreams(t *testing.T) {
	m := media.Record{
		Version: media.Version,
		Video:   media.Video{Codec: "h264", Width: 1920, Height: 1080, BitDepth: 8, HDR: []release.HDR{"SDR"}},
		Audio: []media.Audio{
			{Codec: "eac3", Channels: 6, Language: "en", Default: true},
			{Codec: "ac3", Channels: 2, Language: "en", Title: "Director's commentary"},
			{Codec: "ac3", Channels: 6, Language: "fr"},
			{Codec: "aac", Channels: 2},
		},
		Subtitles: []media.Subtitle{
			{Codec: "hdmv_pgs_subtitle", Image: true, Language: "en"},
			{Codec: "subrip", Language: "fr"},
			{Codec: "subrip", Language: "en"},
			{Codec: "subrip"},
		},
	}
	cases := []struct {
		name      string
		keep      []string
		edit      func(r *encode.Raw)
		maps      []string
		subtitles string
	}{
		{"no language filter keeps every language", nil, func(*encode.Raw) {},
			[]string{"0:V:0", "0:a:0", "0:a:2", "0:a:3", "0:s:0", "0:s:1", "0:s:2", "0:s:3"}, "copy"},
		{"a filter keeps its languages and untagged tracks", []string{"en"}, func(*encode.Raw) {},
			[]string{"0:V:0", "0:a:0", "0:a:3", "0:s:0", "0:s:2", "0:s:3"}, "copy"},
		{"commentary stays when asked", []string{"en"}, func(r *encode.Raw) { r.DropCommentary = false },
			[]string{"0:V:0", "0:a:0", "0:a:1", "0:a:3", "0:s:0", "0:s:2", "0:s:3"}, "copy"},
		{"stereo in mp4 from the default track, image subtitles dropped", []string{"en"}, func(r *encode.Raw) {
			r.Audio, r.Container = "stereo_aac", "mp4"
		}, []string{"0:V:0", "0:a:0", "0:s:2", "0:s:3"}, "mov_text"},
		{"both encodes from the first kept track", []string{"fr"}, func(r *encode.Raw) { r.Audio = "ac3_51_and_stereo_aac" },
			[]string{"0:V:0", "0:a:2", "0:a:2", "0:s:1", "0:s:3"}, "copy"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var keep []language.Code
			if c.keep != nil {
				keep = codes(t, c.keep...)
			}
			args, err := pipeline.Build(pipeline.Job{
				Input: "in.mkv", Output: "out.part", OS: "linux", Settings: settings(t, c.edit), Source: m, Keep: keep,
				Pipeline: encode.Pipeline{Path: encode.Processor},
			})
			if err != nil {
				t.Fatal(err)
			}
			if got := values(args, "-map"); !slices.Equal(got, c.maps) {
				t.Fatalf("maps %q, want %q", got, c.maps)
			}
			if got := values(args, "-c:s"); !slices.Equal(got, []string{c.subtitles}) {
				t.Fatalf("subtitles %q, want %q", got, c.subtitles)
			}
		})
	}
}
