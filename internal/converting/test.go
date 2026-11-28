package converting

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/converting/clip"
	"github.com/jatassi/slipstream/internal/converting/internal/drivers"
	"github.com/jatassi/slipstream/internal/converting/pipeline"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
)

type realProofs struct {
	cfg      RealConfig
	conv     *realConverter
	adapters drivers.Reader
	probe    probe
}

func (p *realProofs) Signature(ctx context.Context) (Signature, []Device, error) {
	return readSignature(ctx, p.conv.ffmpeg(), p.cfg.Tools.Environ, p.adapters)
}

// trialSource is the size a test run's clip stands for. Out never scales
// up, so a clip read at its own size would copy its video and prove no
// encoder; read as this, it re-encodes to the 852x480 of trialHeight.
var trialSource = struct{ W, H int }{1920, 1080}

const (
	trialHeight = encode.P480
	trialKbps   = 2000
	// pauseSettle is how long after p a paused run's progress has to stop,
	// and pauseHeld how long it then stays still.
	pauseSettle = 1500 * time.Millisecond
	pauseHeld   = time.Second
)

func (p *realProofs) Test(ctx context.Context, k encode.Key) (Proved, error) {
	proved, _, err := p.test(ctx, k)
	return proved, err
}

// test is Test, and whether a hardware row opened its encoder.
func (p *realProofs) test(ctx context.Context, k encode.Key) (Proved, bool, error) {
	sig, devices, err := p.Signature(ctx)
	if err != nil {
		return Proved{}, false, err
	}
	dir := filepath.Join(p.cfg.Scratch(), clipsDir, k.Shape.String()+"-"+string(k.Class))
	defer func() { _ = os.RemoveAll(dir) }()
	made, err := clip.Make(ctx, p.cfg.Tools, k.Class, clip.Trial, dir)
	if err != nil {
		return Proved{}, false, err
	}
	src, err := p.cfg.Media.Probe(ctx, made)
	if err != nil {
		return Proved{}, false, fmt.Errorf("probe the %s clip: %w", k.Class, err)
	}
	src.Width, src.Height, src.Bitrate = trialSource.W, trialSource.H, 0
	s := encode.Settings{
		Codec: k.Shape.Codec, MaxHeight: trialHeight, Kbps: trialKbps, Depth: k.Shape.Depth, Range: k.Shape.Range,
		KeepDV: k.Shape.KeepDV, Curve: k.Shape.Curve, Audio: encode.CopyAudio, Container: encode.MKV,
	}
	proved, opened := Proved{Signature: sig}, false
	for _, row := range pipeline.For(runtime.GOOS, k.Shape) {
		hardware := row.Pipeline.Path != encode.Processor
		candidates := devicesOf(row.Pipeline, devices)
		if len(candidates) == 0 {
			proved.Tries = append(proved.Tries, Try{Pipeline: row.Pipeline, Err: "This Node has no adapter for it."})
			continue
		}
		for i, device := range candidates {
			job := pipeline.Job{
				Input: made, Output: filepath.Join(dir, fmt.Sprintf("%s-%d.mkv", row.Name, i)), OS: runtime.GOOS,
				Settings: s, Source: src, Pipeline: row.Pipeline, Device: device,
			}
			t := p.try(ctx, job)
			opened = opened || (hardware && t.opened)
			proved.Tries = append(proved.Tries, Try{Pipeline: row.Pipeline, Err: t.err})
			if t.err != "" {
				continue
			}
			w, h := encode.Out(src, s)
			proved.Passed, proved.Pipeline, proved.Device = true, row.Pipeline, device
			proved.Speed = t.speed * float64(w*h) / float64(trialSource.W*trialSource.H)
			job.Output = filepath.Join(dir, fmt.Sprintf("%s-%d-pause.mkv", row.Name, i))
			proved.Pauses = p.pauses(ctx, job)
			return proved, opened, nil
		}
	}
	return proved, opened, nil
}

// devicesOf is each adapter a pipeline can open, as its row's Init names
// it: a render node's path for VA-API and Linux Quick Sync, an NVIDIA
// GPU's index for CUDA, and one empty device for Windows Quick Sync,
// VideoToolbox and the processor.
func devicesOf(p encode.Pipeline, devices []Device) []string {
	if p.Path == encode.Processor {
		return []string{""}
	}
	var out []string
	vendor := func(d Device, name string) bool { return strings.Contains(strings.ToLower(d.Vendor), name) }
	gpus := 0
	for _, d := range devices {
		switch p.API {
		case encode.CUDA:
			if vendor(d, "nvidia") {
				out = append(out, strconv.Itoa(gpus))
				gpus++
			}
		case encode.VAAPI, encode.QSV:
			switch {
			case runtime.GOOS == "windows" && p.API == encode.QSV && vendor(d, "intel"):
				return []string{""}
			case strings.HasPrefix(d.Name, "renderD") && (p.API == encode.VAAPI || vendor(d, "intel")):
				out = append(out, "/dev/dri/"+d.Name)
			}
		case encode.VideoToolbox:
			if vendor(d, "apple") {
				return []string{""}
			}
		}
	}
	return out
}

type tried struct {
	// opened is whether the run got as far as encoding a frame.
	opened bool
	speed  float64
	// err is empty when the run exited 0 inside Timings.Trial with an
	// output that has the settings' codec, depth and range in limited
	// range.
	err string
}

func (p *realProofs) try(ctx context.Context, job pipeline.Job) tried {
	args, err := pipeline.Build(job)
	if err != nil {
		return tried{err: "SlipStream can't build this pipeline's arguments."}
	}
	ctx, cancel := context.WithTimeout(ctx, p.cfg.Timings.Trial)
	defer cancel()
	run, err := p.conv.Start(ctx, args)
	if err != nil {
		return tried{err: "ffmpeg didn't start."}
	}
	var last Progress
	for pr := range run.Progress() {
		last = pr
	}
	e := run.Wait()
	t := tried{opened: last.OutTimeUs > 0 || e.Code == 0, speed: last.Speed}
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		t.err = fmt.Sprintf("It took longer than the test run's %s.", p.cfg.Timings.Trial)
	case e.Code != 0:
		t.err = fmt.Sprintf("ffmpeg exited with code %d. %s", e.Code, lastLine(e.Tail))
	default:
		t.err = p.check(ctx, job)
	}
	return t
}

// check is why job's output isn't what its settings make, empty when it
// is.
func (p *realProofs) check(ctx context.Context, job pipeline.Job) string {
	out, err := p.cfg.Media.Probe(ctx, job.Output)
	if err != nil {
		return "Its output can't be probed."
	}
	s := job.Settings
	switch {
	case encode.Codec(out.Codec) != s.Codec:
		return fmt.Sprintf("Its output is %s, not %s.", out.Codec, s.Codec)
	case out.BitDepth != int(s.Depth):
		return fmt.Sprintf("Its output is %d-bit, not %d-bit.", out.BitDepth, s.Depth)
	case !ranged(job.Source, s, out):
		return fmt.Sprintf("Its output's dynamic range is %v.", out.HDR)
	}
	cmd := exec.CommandContext(ctx, p.cfg.Tools.FFprobe, "-v", "error", "-select_streams", "v:0", //nolint:gosec // the bundled ffprobe on a test run's output
		"-show_entries", "stream=color_range", "-of", "default=nw=1:nk=1", job.Output)
	cmd.Env = p.cfg.Tools.Environ
	colorRange, err := cmd.Output()
	if err != nil {
		return "Its output can't be probed."
	}
	if r := string(bytes.TrimSpace(colorRange)); r != "tv" {
		return fmt.Sprintf("Its output is in %s range, not limited range.", r)
	}
	return ""
}

// ranged is whether out has the dynamic range s writes from src.
func ranged(src media.Record, s encode.Settings, out media.Record) bool {
	has := func(h release.HDR) bool { return slices.Contains(out.HDR, h) }
	srcHLG := slices.Contains(src.HDR, "HLG") && !slices.Contains(src.HDR, "HDR10")
	switch {
	case s.Range == encode.SDR:
		return slices.Equal(out.HDR, []release.HDR{"SDR"})
	case s.KeepDV && src.DolbyVision != nil && src.DolbyVision.Profile != 7:
		return out.DolbyVision != nil
	case s.Range == encode.SourceRange && srcHLG:
		return has("HLG")
	}
	return has("HDR10")
}

func lastLine(tail string) string {
	lines := strings.Split(strings.TrimSpace(tail), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// pauses runs job at its clip's own pace, writes p at once, and reports
// whether progress then stayed still for pauseHeld and, after u, the run
// exited 0.
func (p *realProofs) pauses(ctx context.Context, job pipeline.Job) bool {
	args, err := pipeline.Build(job)
	if err != nil {
		return false
	}
	at := slices.Index(args, "-i")
	args = slices.Insert(args, at, "-re")
	ctx, cancel := context.WithTimeout(ctx, p.cfg.Timings.Trial)
	defer cancel()
	run, err := p.conv.Start(ctx, args)
	if err != nil {
		return false
	}
	defer run.Kill()
	if run.Pause() != nil {
		return false
	}
	var last Progress
	follow := func(d time.Duration) bool {
		wait := time.NewTimer(d)
		defer wait.Stop()
		for {
			select {
			case pr, ok := <-run.Progress():
				if !ok {
					return false
				}
				last = pr
			case <-wait.C:
				return true
			}
		}
	}
	if !follow(pauseSettle) {
		return false
	}
	still := last.OutTimeUs
	if !follow(pauseHeld) || last.OutTimeUs != still || run.Resume() != nil {
		return false
	}
	for pr := range run.Progress() {
		last = pr
	}
	return run.Wait().Code == 0 && last.OutTimeUs > still
}
