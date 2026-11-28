package converting

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/media/mediatest"
	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/storing"
)

type fakeRun struct {
	mu       sync.Mutex
	progress chan Progress
	exit     Exit
	ended    bool
	killed   bool
	pauses   int
	resumes  int
}

func (r *fakeRun) Progress() <-chan Progress { return r.progress }
func (r *fakeRun) Tail() string              { return "" }

func (r *fakeRun) Pause() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pauses++
	return nil
}

func (r *fakeRun) Resume() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resumes++
	return nil
}

func (r *fakeRun) Kill() {
	r.mu.Lock()
	r.killed = true
	r.mu.Unlock()
	r.finish(Exit{Code: -1})
}

func (r *fakeRun) Wait() Exit {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.exit
}

func (r *fakeRun) send(p Progress) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.ended {
		r.progress <- p
	}
}

func (r *fakeRun) finish(e Exit) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.ended {
		r.ended, r.exit = true, e
		close(r.progress)
	}
}

func (r *fakeRun) wasKilled() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.killed
}

type fakeConverter struct {
	mu   sync.Mutex
	runs []*fakeRun
	args [][]string
}

func (c *fakeConverter) Start(_ context.Context, args []string) (Run, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	r := &fakeRun{progress: make(chan Progress, 16)}
	c.runs, c.args = append(c.runs, r), append(c.args, args)
	return r, nil
}

func (c *fakeConverter) started() []*fakeRun {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.runs)
}

type fakeProofs struct{ Proofs }

func (fakeProofs) Signature(context.Context) (Signature, []Device, error) {
	return "sig", nil, nil
}

type fakeHolder struct {
	intents chan role.Snapshot[*cv1.Intent]
	mu      sync.Mutex
	results []string
}

func (f *fakeHolder) Intent(ctx context.Context) iter.Seq[role.Snapshot[*cv1.Intent]] {
	return func(yield func(role.Snapshot[*cv1.Intent]) bool) {
		for {
			select {
			case s := <-f.intents:
				if !yield(s) {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}
}

func (f *fakeHolder) SetFacts(*cv1.Facts) {}

func (f *fakeHolder) Report(_ context.Context, _ *storage.Tx[storage.Local], r *cv1.Result) error {
	var what string
	switch x := r.GetResult().(type) {
	case *cv1.Result_Started:
		what = "started"
	case *cv1.Result_Paused:
		what = "paused"
	case *cv1.Result_Resumed:
		what = "resumed"
	case *cv1.Result_Exited:
		what = fmt.Sprintf("exited %d", x.Exited.GetBytes())
	case *cv1.Result_Fault:
		what = "fault " + x.Fault.GetCode()
	case *cv1.Result_DiskFull:
		what = "disk full"
	case *cv1.Result_Lost:
		what = "lost"
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.results = append(f.results, fmt.Sprintf("%s/%d %s", r.GetConversionId(), r.GetAttempt(), what))
	return nil
}

func (f *fakeHolder) take() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := f.results
	f.results = nil
	return out
}

const stall = 2 * time.Minute

var source = media.Record{
	Container: "matroska,webm", Duration: time.Hour,
	Video: media.Video{Codec: "hevc", Width: 3840, Height: 2160, BitDepth: 10, HDR: []release.HDR{"HDR10"}},
	Audio: []media.Audio{{Codec: "eac3", Channels: 6, Language: "en", Default: true}},
}

// made is an output that matches source converted with settings.
var made = media.Record{
	Container: "matroska,webm", Duration: time.Hour,
	Video: media.Video{Codec: "h264", Width: 1920, Height: 1080, BitDepth: 8, HDR: []release.HDR{"SDR"}},
	Audio: source.Audio,
}

var settings = &cv1.Settings{
	Codec: string(encode.H264), MaxHeight: 1080, Kbps: 8000, Depth: 8, Range: string(encode.SDR),
	Curve: string(encode.BT2390), Audio: string(encode.CopyAudio), DropCommentary: true, Container: string(encode.MKV),
}

type world struct {
	t       *testing.T
	scratch string
	conv    *fakeConverter
	rh      *fakeHolder
	cfg     HolderConfig
	h       *Holder
	stop    func()
}

func newWorld(t *testing.T, probes map[string]media.Record) *world {
	scratch := filepath.Join(t.TempDir(), "converting")
	disk, err := storing.NewOSDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	timings := DefaultTimings()
	timings.Stall = stall
	w := &world{t: t, scratch: scratch, conv: &fakeConverter{}, rh: &fakeHolder{intents: make(chan role.Snapshot[*cv1.Intent])}}
	w.cfg = HolderConfig{
		Converter: w.conv, Proofs: fakeProofs{}, Media: mediatest.Prober{Of: probes},
		Scratch: func() string { return scratch }, Override: func() string { return "" },
		Disk: disk, DB: storagetest.Local(t), Timings: timings, Log: slog.New(slog.DiscardHandler),
	}
	w.run(Hold(w.cfg))
	return w
}

func (w *world) run(h *Holder) {
	w.h = h
	ctx, cancel := context.WithCancel(w.t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := h.Run(ctx, w.rh); err != nil {
			w.t.Error(err)
		}
	}()
	w.stop = func() {
		cancel()
		<-done
	}
	w.t.Cleanup(w.stop)
}

func (w *world) restart(h *Holder) {
	w.stop()
	w.run(h)
}

func (w *world) intent(in *cv1.Intent) {
	w.rh.intents <- role.Snapshot[*cv1.Intent]{Intent: in, Standing: role.Live}
	synctest.Wait()
}

func record(t *testing.T) *cv1.Source {
	b, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	return &cv1.Source{Record: b}
}

// placed is job a's first attempt, reading a source copy and writing to
// scratch, as the Controller places it.
func placed(t *testing.T, stage cv1.Stage) *cv1.Job {
	return &cv1.Job{
		ConversionId: []byte("a"), Attempt: 1, Stage: stage, Start: true,
		Input:    &cv1.Job_Scratch{Scratch: "in/s-f.mkv"},
		Output:   &cv1.Job_ScratchOut{ScratchOut: "out/c-a-1.mkv"},
		Settings: settings, Source: record(t),
		Pipeline: &cv1.Pipeline{Path: string(encode.Processor)},
	}
}

func one(t *testing.T, hold bool, stage cv1.Stage) *cv1.Intent {
	return &cv1.Intent{Jobs: []*cv1.Job{placed(t, stage)}, Hold: hold, AtOnce: 1, Keep: []string{"in/s-f.mkv", "out/c-a-1.mkv"}}
}

func (w *world) started() *fakeRun {
	w.intent(one(w.t, false, cv1.Stage_STAGE_READY))
	runs := w.conv.started()
	if len(runs) != 1 {
		w.t.Fatalf("started %d runs, want 1", len(runs))
	}
	if got := w.rh.take(); !slices.Equal(got, []string{"a/1 started"}) {
		w.t.Fatalf("results = %q, want a/1 started", got)
	}
	return runs[0]
}

func TestStall(t *testing.T) {
	advance := func(r *fakeRun, at int64) {
		r.send(Progress{OutTimeUs: at})
		synctest.Wait()
	}
	cases := []struct {
		name  string
		drive func(w *world, r *fakeRun)
		want  []string
	}{
		{"no progress for the stall time", func(*world, *fakeRun) { time.Sleep(stall + time.Second) }, []string{"a/1 fault stalled"}},
		{"progress that advances", func(_ *world, r *fakeRun) {
			for i := range 5 {
				advance(r, int64(i+1)*1e6)
				time.Sleep(stall - time.Second)
			}
		}, nil},
		{"progress that doesn't advance", func(_ *world, r *fakeRun) {
			for range 3 {
				advance(r, 1e6)
				time.Sleep(stall / 2)
			}
		}, []string{"a/1 fault stalled"}},
		{"paused for longer than the stall time", func(w *world, _ *fakeRun) {
			w.intent(one(w.t, true, cv1.Stage_STAGE_RUNNING))
			time.Sleep(3 * stall)
		}, []string{"a/1 paused"}},
		{"resumed and still", func(w *world, _ *fakeRun) {
			w.intent(one(w.t, true, cv1.Stage_STAGE_RUNNING))
			time.Sleep(3 * stall)
			w.intent(one(w.t, false, cv1.Stage_STAGE_PAUSED))
			time.Sleep(stall - time.Second)
			synctest.Wait()
			if got := w.rh.take(); !slices.Equal(got, []string{"a/1 paused", "a/1 resumed"}) {
				t.Errorf("resumed and still: before the stall time, results = %q", got)
			}
			time.Sleep(2 * time.Second)
		}, []string{"a/1 fault stalled"}},
	}
	for _, c := range cases {
		synctest.Test(t, func(t *testing.T) {
			w := newWorld(t, nil)
			r := w.started()
			c.drive(w, r)
			synctest.Wait()
			if got := w.rh.take(); !slices.Equal(got, c.want) {
				t.Errorf("%s: results = %q, want %q", c.name, got, c.want)
			}
			if stalled := slices.Contains(c.want, "a/1 fault stalled"); r.wasKilled() != stalled {
				t.Errorf("%s: killed = %v, want %v", c.name, r.wasKilled(), stalled)
			}
		})
	}
}

func TestLostAfterARestart(t *testing.T) {
	for _, same := range []bool{true, false} {
		synctest.Test(t, func(t *testing.T) {
			w := newWorld(t, nil)
			r := w.started()
			next := Hold(w.cfg)
			if same {
				next = w.h
			}
			w.restart(next)
			if !r.wasKilled() {
				t.Errorf("same holder %v: the run outlived its holder", same)
			}
			for range 2 {
				w.intent(one(t, false, cv1.Stage_STAGE_RUNNING))
			}
			if got := w.rh.take(); !slices.Equal(got, []string{"a/1 lost"}) {
				t.Errorf("same holder %v: results = %q, want one a/1 lost", same, got)
			}
			if n := len(w.conv.started()); n != 1 {
				t.Errorf("same holder %v: started %d runs, want 1", same, n)
			}
		})
	}
}

func TestExit(t *testing.T) {
	short := made
	short.Duration = 50 * time.Minute
	silent := made
	silent.Audio = nil
	cases := []struct {
		name   string
		exit   Exit
		output media.Record
		want   string
	}{
		{"exit 0 with a matching output", Exit{}, made, "a/1 exited 4"},
		{"exit 0 with a short output", Exit{}, short, "a/1 fault short_output"},
		{"exit 0 missing the audio", Exit{}, silent, "a/1 fault streams"},
		{"no space left", Exit{Code: 228, Tail: "Error writing trailer: No space left on device"}, made, "a/1 disk full"},
		{"an encoder that doesn't open", Exit{Code: 234, Tail: "Error while opening encoder"}, made, "a/1 fault encoder"},
	}
	for _, c := range cases {
		synctest.Test(t, func(t *testing.T) {
			w := newWorld(t, map[string]media.Record{"out/c-a-1.mkv": c.output})
			r := w.started()
			if err := os.WriteFile(filepath.Join(w.scratch, "out", "c-a-1.mkv.part"), []byte("made"), 0o600); err != nil {
				t.Fatal(err)
			}
			r.finish(c.exit)
			synctest.Wait()
			if got := w.rh.take(); !slices.Equal(got, []string{c.want}) {
				t.Errorf("%s: results = %q, want %q", c.name, got, c.want)
			}
		})
	}
}

func TestSweep(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, nil)
		files := []string{
			"in/s-f.mkv", "in/s-gone.mkv", "out/c-a-1.mkv.part", "out/c-a-1.mkv", "out/c-a-0.mkv.part",
			"out/c-b-1.mkv", ".transfers/t/0", "clips/hdr10.mkv",
		}
		for _, f := range files {
			path := filepath.Join(w.scratch, filepath.FromSlash(f))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, nil, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		w.intent(&cv1.Intent{Keep: []string{"in/s-f.mkv", "out/c-a-1.mkv"}})
		var left []string
		for _, f := range files {
			if _, err := os.Stat(filepath.Join(w.scratch, filepath.FromSlash(f))); err == nil {
				left = append(left, f)
			}
		}
		want := []string{"in/s-f.mkv", "out/c-a-1.mkv.part", "out/c-a-1.mkv", ".transfers/t/0", "clips/hdr10.mkv"}
		if !slices.Equal(left, want) {
			t.Errorf("left %q, want %q", left, want)
		}
	})
}
