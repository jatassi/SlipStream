package converting

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/converting/clip"
	"github.com/jatassi/slipstream/internal/converting/internal/drivers"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/media"
	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/storing"
)

const asEnv = "converting-test-environ"

func TestMain(m *testing.M) {
	if len(os.Args) == 3 && os.Args[1] == asEnv {
		if err := os.WriteFile(os.Args[2], []byte(strings.Join(os.Environ(), "\n")), 0o600); err != nil {
			os.Exit(9)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestRealRunsFFmpegWithoutSlipStreamVariables(t *testing.T) {
	t.Setenv("SLIPSTREAM_SERVER_PORT", "8080")
	t.Setenv("slipstream_lower", "1")
	t.Setenv("KEPT_FOR_FFMPEG", "1")
	conv, _ := Real(RealConfig{
		Tools: clip.Tools{FFmpeg: os.Args[0], Environ: os.Environ()}, Override: func() string { return "" },
		Timings: DefaultTimings(),
	})
	out := filepath.Join(t.TempDir(), "environ")
	run, err := conv.Start(t.Context(), []string{asEnv, out})
	if err != nil {
		t.Fatal(err)
	}
	if e := run.Wait(); e.Code != 0 {
		t.Fatalf("the program exited %d", e.Code)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	env := strings.Split(string(got), "\n")
	if !slices.Contains(env, "KEPT_FOR_FFMPEG=1") {
		t.Errorf("the environment lost KEPT_FOR_FFMPEG: %q", env)
	}
	for _, kv := range env {
		if strings.HasPrefix(strings.ToUpper(kv), "SLIPSTREAM_") {
			t.Errorf("the environment has %q", kv)
		}
	}
}

func nativeTools(t *testing.T) clip.Tools {
	t.Helper()
	var out clip.Tools
	for _, p := range filepath.SplitList(os.Getenv("SLIPSTREAM_NATIVE_TOOLS")) {
		switch filepath.Base(p) {
		case "ffmpeg", "ffmpeg.exe":
			out.FFmpeg = p
		case "ffprobe", "ffprobe.exe":
			out.FFprobe = p
		case "dovi_tool", "dovi_tool.exe":
			out.DoviTool = p
		}
	}
	if out.FFmpeg == "" || out.FFprobe == "" || out.DoviTool == "" {
		t.Fatal("SLIPSTREAM_NATIVE_TOOLS names no ffmpeg, ffprobe or dovi_tool: run the tests with go run ./cmd/release test")
	}
	out.Environ = os.Environ()
	return out
}

// realProofsWith is Real's Proofs on a Node whose drivers report adapters.
func realProofsWith(t *testing.T, scratch string, adapters []drivers.Adapter) *realProofs {
	tl := nativeTools(t)
	_, proofs := Real(RealConfig{
		Tools: tl, Scratch: func() string { return scratch }, Override: func() string { return "" },
		Media: media.FFprobe{Path: tl.FFprobe}, Timings: DefaultTimings(),
	})
	p, ok := proofs.(*realProofs)
	if !ok {
		t.Fatalf("Real's Proofs is %T", proofs)
	}
	p.adapters = func(context.Context) ([]drivers.Adapter, error) { return adapters, nil }
	return p
}

// TestAHardwareRowFailsItsTestRunWithoutAGPU names adapters this runner
// doesn't have, so each hardware row runs and fails to open its device,
// and the base test proves on the processor.
func TestAHardwareRowFailsItsTestRunWithoutAGPU(t *testing.T) {
	p := realProofsWith(t, t.TempDir(), []drivers.Adapter{
		{Name: "renderD199", Vendor: "Intel", Driver: "Intel iHD driver 24.1.0"},
		{Name: "NVIDIA GeForce RTX 3060", Vendor: "NVIDIA", Driver: "550.54.14"},
	})
	if fit, _ := p.Runs(); fit != role.Weak {
		t.Errorf("before the first answer, Runs = %v, want weak", fit)
	}
	ctx, cancel := context.WithCancel(t.Context())
	changed := make(chan struct{}, 1)
	warmed := make(chan error, 1)
	go func() { warmed <- p.Warm(ctx, func() { changed <- struct{}{} }) }()
	select {
	case <-changed:
	case <-time.After(5 * time.Minute):
		t.Fatal("Warm never answered")
	}
	cancel()
	if err := <-warmed; err != nil {
		t.Fatal(err)
	}
	if fit, _ := p.Runs(); fit != role.Weak {
		t.Errorf("Runs = %v, want weak: only the processor passed", fit)
	}

	proved, err := p.Test(t.Context(), Base)
	if err != nil {
		t.Fatal(err)
	}
	if !proved.Passed || proved.Pipeline.Path != encode.Processor {
		t.Fatalf("the base test = %+v, want a pass on the processor", proved)
	}
	if !proved.Pauses || proved.Speed <= 0 || len(proved.Signature) != 64 {
		t.Errorf("the proof = %+v, want it to pause, a speed and a signature", proved)
	}
	hardware := 0
	for _, try := range proved.Tries {
		if try.Pipeline.Path == encode.Processor {
			continue
		}
		hardware++
		if try.Err == "" {
			t.Errorf("%+v passed on a runner with no GPU", try.Pipeline)
		}
	}
	if hardware == 0 {
		t.Errorf("tries = %+v, want a hardware row before the processor", proved.Tries)
	}
}

// tapConverter keeps every progress its runs report, and lets a test
// write p behind the holder's back.
type tapConverter struct {
	Converter
	mu   sync.Mutex
	runs []*tapRun
}

type tapRun struct {
	Run
	out  chan Progress
	mu   sync.Mutex
	last Progress
}

func (c *tapConverter) Start(ctx context.Context, args []string) (Run, error) {
	r, err := c.Converter.Start(ctx, args)
	if err != nil {
		return nil, err
	}
	tr := &tapRun{Run: r, out: make(chan Progress, 64)}
	go func() {
		defer close(tr.out)
		for p := range r.Progress() {
			tr.mu.Lock()
			tr.last = p
			tr.mu.Unlock()
			select {
			case tr.out <- p:
			default:
			}
		}
	}()
	c.mu.Lock()
	c.runs = append(c.runs, tr)
	c.mu.Unlock()
	return tr, nil
}

func (r *tapRun) Progress() <-chan Progress { return r.out }

func (r *tapRun) outTime() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.last.OutTimeUs
}

type realWorld struct {
	t       *testing.T
	tools   clip.Tools
	scratch string
	conv    *tapConverter
	rh      *fakeHolder
	results []string
}

func newRealWorld(t *testing.T, stall time.Duration) *realWorld {
	tl := nativeTools(t)
	scratch := filepath.Join(t.TempDir(), "converting")
	disk, err := storing.NewOSDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	timings := DefaultTimings()
	timings.Stall = stall
	prober := media.FFprobe{Path: tl.FFprobe}
	cfg := RealConfig{Tools: tl, Scratch: func() string { return scratch }, Override: func() string { return "" }, Media: prober, Timings: timings}
	conv, proofs := Real(cfg)
	w := &realWorld{
		t: t, tools: tl, scratch: scratch, conv: &tapConverter{Converter: conv},
		rh: &fakeHolder{intents: make(chan role.Snapshot[*cv1.Intent])},
	}
	h := Hold(HolderConfig{
		Converter: w.conv, Proofs: proofs, Media: prober, Scratch: cfg.Scratch, Override: cfg.Override,
		Disk: disk, DB: storagetest.Local(t), Timings: timings, Log: slog.New(slog.DiscardHandler),
	})
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = h.Run(ctx, w.rh)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return w
}

// source makes a clip of class c at size s as the job's source copy, and
// answers its record, read as from a source of width by height.
func (w *realWorld) source(c encode.Class, s clip.Size, width, height int) media.Record {
	made, err := clip.Make(w.t.Context(), w.tools, c, s, filepath.Join(w.scratch, "in"))
	if err != nil {
		w.t.Fatal(err)
	}
	if err := os.Rename(made, filepath.Join(w.scratch, "in", "s-f.mkv")); err != nil {
		w.t.Fatal(err)
	}
	rec, err := media.FFprobe{Path: w.tools.FFprobe}.Probe(w.t.Context(), filepath.Join(w.scratch, "in", "s-f.mkv"))
	if err != nil {
		w.t.Fatal(err)
	}
	rec.Width, rec.Height, rec.Bitrate = width, height, 0
	return rec
}

func (w *realWorld) intent(src media.Record, s encode.Settings, hold bool, stage cv1.Stage) {
	b, err := json.Marshal(src)
	if err != nil {
		w.t.Fatal(err)
	}
	job := &cv1.Job{
		ConversionId: []byte("a"), Attempt: 1, Stage: stage, Start: true,
		Input:  &cv1.Job_Scratch{Scratch: "in/s-f.mkv"},
		Output: &cv1.Job_ScratchOut{ScratchOut: "out/c-a-1.mkv"},
		Settings: &cv1.Settings{
			Codec: string(s.Codec), MaxHeight: uint32(s.MaxHeight), Kbps: uint32(s.Kbps), Depth: uint32(s.Depth), //nolint:gosec // the test's own settings
			Range: string(s.Range), KeepDv: s.KeepDV, Curve: string(s.Curve), Audio: string(s.Audio), Container: string(s.Container),
		},
		Source: &cv1.Source{Record: b}, Pipeline: &cv1.Pipeline{Path: string(encode.Processor)},
	}
	in := &cv1.Intent{Jobs: []*cv1.Job{job}, Hold: hold, AtOnce: 1, Keep: []string{"in/s-f.mkv", "out/c-a-1.mkv"}}
	w.rh.intents <- role.Snapshot[*cv1.Intent]{Intent: in, Standing: role.Live}
}

// await waits for the holder to report want, and fails on any other
// result first.
func (w *realWorld) await(want string, within time.Duration) {
	w.t.Helper()
	for deadline := time.Now().Add(within); ; time.Sleep(50 * time.Millisecond) {
		w.results = append(w.results, w.rh.take()...)
		if i := slices.Index(w.results, "a/1 "+want); i >= 0 {
			if i > 0 {
				w.t.Fatalf("results = %q, want %s first", w.results, want)
			}
			w.results = w.results[1:]
			return
		}
		if len(w.results) > 0 || time.Now().After(deadline) {
			w.t.Fatalf("results = %q, want a/1 %s", w.results, want)
		}
	}
}

func (w *realWorld) run() *tapRun {
	w.conv.mu.Lock()
	defer w.conv.mu.Unlock()
	return w.conv.runs[0]
}

// encoding waits until the run's output time has passed zero.
func (w *realWorld) encoding(r *tapRun) {
	w.t.Helper()
	for deadline := time.Now().Add(time.Minute); r.outTime() == 0; time.Sleep(20 * time.Millisecond) {
		if time.Now().After(deadline) {
			w.t.Fatal("the run never reported its output time")
		}
	}
}

func settingsOf(codec encode.Codec, depth encode.Depth, rng encode.Range, keepDV bool, height encode.Height) encode.Settings {
	s := encode.Defaults()
	s.Codec, s.Depth, s.Range, s.KeepDV, s.MaxHeight, s.Kbps = codec, depth, rng, keepDV, height, 2000
	return s
}

// TestTheProcessorConvertsEachClass reads each clip as a 1080p source, so
// a 480p output re-encodes it, and checks the made file's own probe.
func TestTheProcessorConvertsEachClass(t *testing.T) {
	sdr := []release.HDR{"SDR"}
	for _, c := range []struct {
		name     string
		class    encode.Class
		settings encode.Settings
		depth    int
		hdr      []release.HDR
		dv       bool
	}{
		{"HDR10 to SDR H.264", encode.ClassHDR10, settingsOf(encode.H264, encode.Bit8, encode.SDR, false, encode.P480), 8, sdr, false},
		{"Dolby Vision 5 to SDR", encode.ClassDV5, settingsOf(encode.H264, encode.Bit8, encode.SDR, false, encode.P480), 8, sdr, false},
		{"Dolby Vision 8.1 kept", encode.ClassDV81, settingsOf(encode.HEVC, encode.Bit10, encode.SourceRange, true, encode.P480), 10, []release.HDR{"DV", "HDR10"}, true},
		{"HDR10 to HEVC 10-bit", encode.ClassHDR10, settingsOf(encode.HEVC, encode.Bit10, encode.HDR10, false, encode.P480), 10, []release.HDR{"HDR10"}, false},
		{"HLG to SDR", encode.ClassHLG, settingsOf(encode.H264, encode.Bit8, encode.SDR, false, encode.P480), 8, sdr, false},
		{"MPEG-2 decoded on the processor", encode.ClassDecodeSDR, settingsOf(encode.H264, encode.Bit8, encode.SDR, false, encode.P480), 8, sdr, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := newRealWorld(t, 2*time.Minute)
			src := w.source(c.class, clip.Trial, 1920, 1080)
			w.intent(src, c.settings, false, cv1.Stage_STAGE_READY)
			w.await("started", time.Minute)
			w.awaitExit(2 * time.Minute)
			out := filepath.Join(w.scratch, "out", "c-a-1.mkv")
			got, err := media.FFprobe{Path: w.tools.FFprobe}.Probe(t.Context(), out)
			if err != nil {
				t.Fatal(err)
			}
			if got.Codec != string(c.settings.Codec) || got.BitDepth != c.depth || got.Width != 852 || got.Height != 480 ||
				!slices.Equal(got.HDR, c.hdr) || (got.DolbyVision != nil) != c.dv {
				t.Errorf("the made file is %+v, want %s %d-bit 852x480 %v, Dolby Vision %v", got.Video, c.settings.Codec, c.depth, c.hdr, c.dv)
			}
			if r := w.colorRange(out); r != "tv" {
				t.Errorf("the made file's color range is %q, want tv", r)
			}
		})
	}
}

func (w *realWorld) awaitExit(within time.Duration) {
	w.t.Helper()
	for deadline := time.Now().Add(within); ; time.Sleep(50 * time.Millisecond) {
		w.results = append(w.results, w.rh.take()...)
		if len(w.results) > 0 {
			if !strings.HasPrefix(w.results[0], "a/1 exited ") || w.results[0] == "a/1 exited 0" {
				w.t.Fatalf("results = %q, want a/1 exited with its size: %s", w.results, w.run().Tail())
			}
			w.results = w.results[1:]
			return
		}
		if time.Now().After(deadline) {
			w.t.Fatal("the run never ended")
		}
	}
}

func (w *realWorld) colorRange(file string) string {
	cmd := exec.CommandContext(w.t.Context(), w.tools.FFprobe, "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=color_range", "-of", "default=nw=1:nk=1", file)
	out, err := cmd.Output()
	if err != nil {
		w.t.Fatal(err)
	}
	return string(bytes.TrimSpace(out))
}

// TestPauseAndStall runs a 720p H.264 clip through a 720p HEVC encode, long
// enough on the processor to pause. The pause case keeps the default stall
// limit, since TestStall proves under synctest that a paused run never
// stalls, and a resumed encode can take seconds to move again on a loaded
// runner. The stall case shortens the limit and pauses ffmpeg behind the
// holder's back before its first progress line, since the limit runs from
// the start and a loaded runner can take longer than it to report one.
func TestPauseAndStall(t *testing.T) {
	const stall = 3 * time.Second
	s := settingsOf(encode.HEVC, encode.Bit8, encode.SDR, false, encode.P720)
	for _, c := range []struct {
		name  string
		stall time.Duration
		drive func(w *realWorld, src media.Record, r *tapRun)
	}{
		{"p freezes the output time and u resumes to a passing output", DefaultTimings().Stall, func(w *realWorld, src media.Record, r *tapRun) {
			w.encoding(r)
			w.intent(src, s, true, cv1.Stage_STAGE_RUNNING)
			w.await("paused", 10*time.Second)
			time.Sleep(time.Second)
			still := r.outTime()
			time.Sleep(3 * time.Second)
			if now := r.outTime(); now != still {
				w.t.Errorf("the output time moved from %d to %d while paused", still, now)
			}
			w.intent(src, s, false, cv1.Stage_STAGE_PAUSED)
			w.await("resumed", 10*time.Second)
			w.awaitExit(2 * time.Minute)
		}},
		{"a run that stops making progress stalls", stall, func(w *realWorld, _ media.Record, r *tapRun) {
			if err := r.Pause(); err != nil {
				w.t.Fatal(err)
			}
			w.await("fault stalled", stall+30*time.Second)
			done := make(chan Exit, 1)
			go func() { done <- r.Wait() }()
			select {
			case e := <-done:
				if e.Code == 0 {
					w.t.Errorf("the stalled run exited 0")
				}
			case <-time.After(DefaultTimings().KillGrace + 5*time.Second):
				w.t.Error("the stalled run is still running")
			}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := newRealWorld(t, c.stall)
			src := w.source(encode.ClassH264x8, clip.Size{W: 1280, H: 720, Seconds: 10}, 1280, 720)
			w.intent(src, s, false, cv1.Stage_STAGE_READY)
			w.await("started", time.Minute)
			c.drive(w, src, w.run())
			if len(w.results) > 0 {
				t.Errorf("then results = %q", w.results)
			}
		})
	}
}
