// Package mediafix makes real media files for the fakes: a Scenario
// MediaFile encoded by the bundled ffmpeg and dovi_tool, byte for byte the
// same on every run, and the record ffprobe reads from it. Its recipes are
// the one source of the captures internal/media's tests parse.
package mediafix

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
)

// Tools are the bundled tool programs. They come from beside a run's
// binary or from SLIPSTREAM_NATIVE_TOOLS, never from PATH.
type Tools struct{ FFmpeg, FFprobe, DoviTool string }

// ToolsIn are the tools in dir, each named with ext, as a release places
// them beside slipstream.
func ToolsIn(dir, ext string) Tools {
	return Tools{
		FFmpeg:   filepath.Join(dir, "ffmpeg"+ext),
		FFprobe:  filepath.Join(dir, "ffprobe"+ext),
		DoviTool: filepath.Join(dir, "dovi_tool"+ext),
	}
}

// toolsVariable lists the path of every tool program the native bundle
// ships, as go run ./cmd/release deps env sets it.
const toolsVariable = "SLIPSTREAM_NATIVE_TOOLS"

// NativeTools are the tools SLIPSTREAM_NATIVE_TOOLS lists.
func NativeTools() (Tools, error) {
	list, ok := os.LookupEnv(toolsVariable)
	if !ok {
		return Tools{}, fmt.Errorf("%s is unset: run the tests with go run ./cmd/release test, or eval \"$(go run ./cmd/release deps env)\" first", toolsVariable)
	}
	var t Tools
	for _, p := range filepath.SplitList(list) {
		switch strings.TrimSuffix(filepath.Base(p), filepath.Ext(p)) {
		case "ffmpeg":
			t.FFmpeg = p
		case "ffprobe":
			t.FFprobe = p
		case "dovi_tool":
			t.DoviTool = p
		}
	}
	if t.FFmpeg == "" || t.FFprobe == "" || t.DoviTool == "" {
		return Tools{}, fmt.Errorf("%s lists no ffmpeg, ffprobe or dovi_tool: %s", toolsVariable, list)
	}
	return t, nil
}

// Builder makes files into a cache, named by their inputs, so a machine
// encodes each file once.
type Builder struct {
	tools Tools
	cache string
	// ffmpeg and dovi are the tools' version text, which keys the cache.
	ffmpeg, dovi string
}

// NewBuilder is a Builder that runs tools and keeps what it makes in
// cache, the per-machine fixture cache in every caller but a test.
func NewBuilder(ctx context.Context, tools Tools, cache string) (*Builder, error) {
	ffmpeg, err := output(ctx, tools.FFmpeg, "-version")
	if err != nil {
		return nil, err
	}
	dovi, err := output(ctx, tools.DoviTool, "--version")
	if err != nil {
		return nil, err
	}
	return &Builder{tools: tools, cache: cache, ffmpeg: ffmpeg, dovi: dovi}, nil
}

func output(ctx context.Context, tool string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, tool, args...).Output()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w", tool, strings.Join(args, " "), err)
	}
	return string(out), nil
}

// Build returns the cached file f describes, making it first if the cache
// lacks it. Callers copy it out and never write to it.
func (b *Builder) Build(ctx context.Context, f scenario.MediaFile) (string, error) {
	r := recipeOf("media", f)
	key, err := json.Marshal(struct {
		FFmpeg   string     `json:"ffmpeg"`
		DoviTool string     `json:"doviTool"`
		Files    [][]string `json:"files"`
		Steps    [][]string `json:"steps"`
	}{b.ffmpeg, b.dovi, r.fileList(), r.stepList()})
	if err != nil {
		return "", fmt.Errorf("fixture key: %w", err)
	}
	sum := sha256.Sum256(key)
	cached := filepath.Join(b.cache, hex.EncodeToString(sum[:])+path.Ext(f.Path))
	if _, err := os.Stat(cached); err == nil {
		return cached, nil
	}
	if err := os.MkdirAll(b.cache, 0o755); err != nil {
		return "", fmt.Errorf("create the fixture cache: %w", err)
	}
	work, err := os.MkdirTemp(b.cache, ".building-")
	if err != nil {
		return "", fmt.Errorf("create the fixture's work folder: %w", err)
	}
	defer func() { _ = os.RemoveAll(work) }()
	if err := b.run(ctx, r, work); err != nil {
		return "", fmt.Errorf("make %s: %w", f.Path, err)
	}
	// A link never replaces a file another world published, so no open of
	// the cached path races a rename onto it. That file is this one.
	if err := os.Link(filepath.Join(work, r.out), cached); err != nil && !errors.Is(err, fs.ErrExist) {
		return "", fmt.Errorf("cache %s: %w", f.Path, err)
	}
	return cached, nil
}

// Write copies the file f describes to dst, making its folders.
func (b *Builder) Write(ctx context.Context, f scenario.MediaFile, dst string) error {
	built, err := b.Build(ctx, f)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(built)
	if err != nil {
		return fmt.Errorf("read %s: %w", f.Path, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("write %s: %w", f.Path, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", f.Path, err)
	}
	return nil
}

func (b *Builder) run(ctx context.Context, r recipe, dir string) error {
	for _, f := range r.files {
		if err := os.WriteFile(filepath.Join(dir, f.name), []byte(f.text), 0o600); err != nil {
			return fmt.Errorf("write %s: %w", f.name, err)
		}
	}
	programs := map[string]string{"ffmpeg": b.tools.FFmpeg, "dovi_tool": b.tools.DoviTool}
	for _, s := range r.steps {
		cmd := exec.CommandContext(ctx, programs[s.tool], s.args...)
		cmd.Dir = dir
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %w: %s", s.tool, err, bytes.TrimSpace(stderr.Bytes()))
		}
	}
	return nil
}

func (r *recipe) fileList() [][]string {
	out := make([][]string, len(r.files))
	for i, f := range r.files {
		out[i] = []string{f.name, f.text}
	}
	return out
}

func (r *recipe) stepList() [][]string {
	out := make([][]string, len(r.steps))
	for i, s := range r.steps {
		out[i] = append([]string{s.tool}, s.args...)
	}
	return out
}

// Want is the record the bundled ffprobe reads from the Matroska file
// Build makes of f, but for its bit rate, which only the encode knows, and its
// duration, which ffprobe reads a few milliseconds longer when the file
// has audio. The simulation tier probes a file of f's path as Want.
func Want(f scenario.MediaFile) media.Record {
	v := f.Video
	r := media.Record{Version: media.Version, Container: "matroska,webm", Duration: time.Duration(v.Seconds) * time.Second}
	r.Video = media.Video{
		Codec: "hevc", Profile: "Main 10", Width: v.Width, Height: v.Height, BitDepth: 10, FrameRate: media.Rate{Num: fps, Den: 1},
		HDR: []release.HDR{"HDR10"},
	}
	switch v.HDR {
	case "sdr":
		r.Codec, r.Profile, r.BitDepth, r.HDR = "h264", "Constrained Baseline", 8, []release.HDR{"SDR"}
	case "hdr10plus":
		r.HDR = []release.HDR{"HDR10+", "HDR10"}
	case "hlg":
		r.HDR = []release.HDR{"HLG"}
	case "dv5":
		r.HDR, r.DolbyVision = []release.HDR{"DV"}, &media.DolbyVision{Profile: 5, Level: dvLevel(v), Compatibility: 0}
	case "dv8":
		r.HDR, r.DolbyVision = []release.HDR{"DV", "HDR10"}, &media.DolbyVision{Profile: 8, Level: dvLevel(v), Compatibility: 1}
	}
	for _, a := range f.Audio {
		profile := ""
		if a.Codec == "aac" {
			profile = "LC"
		}
		r.Audio = append(r.Audio, media.Audio{
			Codec: a.Codec, Profile: profile, Channels: a.Channels, Layout: layoutOf(a), Language: languageOf(a.Language), Default: a.Default, Title: a.Title,
		})
	}
	for _, s := range f.Subtitles {
		r.Subtitles = append(r.Subtitles, media.Subtitle{Codec: "subrip", Language: languageOf(s.Language), Forced: s.Forced, Default: s.Default})
	}
	return r
}

func languageOf(s string) release.Language {
	if s == "" {
		return ""
	}
	l, _ := release.ParseLanguage(s)
	return l
}

// dvLevels are the Dolby Vision levels' highest luma sample rates, level
// 1 first.
var dvLevels = []int{
	1280 * 720 * 24, 1280 * 720 * 30, 1920 * 1080 * 24, 1920 * 1080 * 30, 1920 * 1080 * 60,
	3840 * 2160 * 24, 3840 * 2160 * 30, 3840 * 2160 * 48, 3840 * 2160 * 60, 3840 * 2160 * 120,
	7680 * 4320 * 30, 7680 * 4320 * 60,
}

// dvLevel is the level libx265 writes for v, the first whose sample rate
// holds it.
func dvLevel(v scenario.MediaVideo) int {
	rate := v.Width * v.Height * fps
	for i, limit := range dvLevels {
		if rate <= limit {
			return i + 1
		}
	}
	return len(dvLevels) + 1
}
