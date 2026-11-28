package simtest

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simconv"
	"github.com/jatassi/slipstream/internal/converting"
	"github.com/jatassi/slipstream/internal/converting/pipeline"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/role"
)

// Outcome is how one cued run ends.
type Outcome struct {
	kind  outcomeKind
	tail  string
	short time.Duration
}

type outcomeKind uint8

const (
	succeed outcomeKind = iota
	exitFault
	stall
	short
	hold
)

var (
	// Succeed runs as the simulation's runner does.
	Succeed = Outcome{kind: succeed}
	// Stall never advances and never exits until killed.
	Stall = Outcome{kind: stall}
	// DiskFull exits as ffmpeg does when the output's disk fills.
	DiskFull = Outcome{kind: exitFault, tail: "No space left on device"}
	// Hold advances each second until Release.
	Hold = Outcome{kind: hold}
)

// RunFault exits with the error line converting classifies as code, one
// of encoder, filter, device and crash.
func RunFault(code string) Outcome {
	return faultOf(code, map[string]string{
		converting.Encoder:     "Error while opening encoder",
		converting.Filter:      "Error initializing filter",
		converting.DeviceFault: "Device creation failed",
		converting.Crash:       "Conversion failed!",
	})
}

// SourceFault exits with the error line converting classifies as code,
// unreadable.
func SourceFault(code string) Outcome {
	return faultOf(code, map[string]string{converting.Unreadable: "Invalid data found when processing input"})
}

func faultOf(code string, lines map[string]string) Outcome {
	line, ok := lines[code]
	if !ok {
		panic("simtest: no cue for fault " + code)
	}
	return Outcome{kind: exitFault, tail: line}
}

// Short succeeds with an output d shorter than its source.
func Short(d time.Duration) Outcome { return Outcome{kind: short, short: d} }

// Conversions plays cued outcomes on the runs of videos, in order, one per
// run: once a video's cues are played, its runs succeed. A video is the
// stem the walk fills its bytes from, a one-video release's name or
// <release>/<path> for a file of a release of files, so a run matches
// whether it reads the library file or its copy in scratch. Proofs names
// the pipelines a Node's test runs pass. One Conversions serves every Node
// and keeps its state across their restarts.
type Conversions struct {
	mu     sync.Mutex
	proofs map[string][]encode.Pipeline
	cues   map[string][]Outcome
	ran    map[string]int
	held   map[string][]*pending
	shorts map[string]time.Duration
}

func NewConversions() *Conversions {
	return &Conversions{
		proofs: map[string][]encode.Pipeline{}, cues: map[string][]Outcome{}, ran: map[string]int{},
		held: map[string][]*pending{}, shorts: map[string]time.Duration{},
	}
}

// Proofs passes name's test runs on the first of pipelines that a key's
// rows list, and fails them when none is.
func (c *Conversions) Proofs(name string, pipelines ...encode.Pipeline) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.proofs[name] = pipelines
}

// Cue queues outcomes on stem's next runs.
func (c *Conversions) Cue(stem string, outcomes ...Outcome) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cues[stem] = append(c.cues[stem], outcomes...)
}

// Ran is how many runs the Nodes started of stem, a stem Cue named.
func (c *Conversions) Ran(stem string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ran[stem]
}

// Release ends stem's held runs as Succeed does.
func (c *Conversions) Release(stem string) {
	c.mu.Lock()
	held := c.held[stem]
	delete(c.held, stem)
	c.mu.Unlock()
	for _, p := range held {
		p.release()
	}
}

// Option puts c in front of each Node's runner and prober, and in place
// of the proofs of each Node Proofs named.
func (c *Conversions) Option() sim.Option {
	return sim.Edit(func(name string, e *node.Env) {
		base := e.Converting
		e.Media = shortened{c, e.Media}
		e.Converting = func(cfg converting.RealConfig) (converting.Converter, converting.Proofs) {
			conv, proofs := base(cfg)
			c.mu.Lock()
			pipelines, ok := c.proofs[name]
			c.mu.Unlock()
			if ok {
				proofs = proven(pipelines)
			}
			return cued{c, conv}, proofs
		}
	})
}

// next counts a run of input and answers its outcome and stem, false for
// a run no cue names.
func (c *Conversions) next(input string) (Outcome, string, bool) {
	head := make([]byte, 32)
	f, err := os.Open(input)
	if err != nil {
		return Outcome{}, "", false
	}
	_, err = io.ReadFull(f, head)
	_ = f.Close()
	if err != nil {
		return Outcome{}, "", false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for stem, cues := range c.cues {
		if !bytes.Equal(head, releases.Filler(stem, int64(len(head)))) {
			continue
		}
		c.ran[stem]++
		if len(cues) == 0 {
			return Succeed, stem, true
		}
		c.cues[stem] = cues[1:]
		return cues[0], stem, true
	}
	return Outcome{}, "", false
}

type cued struct {
	c    *Conversions
	base converting.Converter
}

func (r cued) Start(ctx context.Context, args []string) (converting.Run, error) {
	o, stem, ok := r.c.next(args[slices.Index(args, "-i")+1])
	if !ok {
		return r.base.Start(ctx, args)
	}
	switch o.kind {
	case exitFault:
		return exited(converting.Exit{Code: 1, Tail: o.tail}), nil
	case stall:
		return start(nil), nil
	case hold:
		p := start(func(ctx context.Context) converting.Exit {
			run, err := r.base.Start(ctx, args)
			if err != nil {
				return converting.Exit{Code: 1, Tail: err.Error()}
			}
			return run.Wait()
		})
		r.c.mu.Lock()
		r.c.held[stem] = append(r.c.held[stem], p)
		r.c.mu.Unlock()
		return p, nil
	case short:
		r.c.mu.Lock()
		r.c.shorts[strings.TrimSuffix(filepath.Base(args[len(args)-1]), ".part")] = o.short
		r.c.mu.Unlock()
	case succeed:
	}
	return r.base.Start(ctx, args)
}

var errExited = errors.New("ffmpeg has exited")

// exited is a run that has already exited with e.
func exited(e converting.Exit) *pending {
	p := start(nil)
	p.end(e)
	return p
}

// pending is a run until Kill, or until release when finish is set, which
// advances its output time each second while it isn't paused.
type pending struct {
	finish   func(context.Context) converting.Exit
	progress chan converting.Progress
	stop     chan struct{}
	once     sync.Once
	mu       sync.Mutex
	paused   bool
	exit     converting.Exit
}

func start(finish func(context.Context) converting.Exit) *pending {
	p := &pending{finish: finish, progress: make(chan converting.Progress), stop: make(chan struct{})}
	go p.advance()
	return p
}

func (p *pending) advance() {
	defer close(p.progress)
	if p.finish == nil {
		<-p.stop
		return
	}
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	var at time.Duration
	for {
		select {
		case <-p.stop:
			return
		case <-tick.C:
		}
		p.mu.Lock()
		paused := p.paused
		p.mu.Unlock()
		if paused {
			continue
		}
		at += time.Second
		select {
		case p.progress <- converting.Progress{OutTimeUs: at.Microseconds(), Speed: 1}:
		case <-p.stop:
			return
		}
	}
}

func (p *pending) end(e converting.Exit) {
	p.once.Do(func() {
		p.exit = e
		close(p.stop)
	})
}

func (p *pending) release() {
	select {
	case <-p.stop:
	default:
		p.end(p.finish(context.Background()))
	}
}

func (p *pending) setPaused(paused bool) error {
	select {
	case <-p.stop:
		return errExited
	default:
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.paused = paused
	return nil
}

func (p *pending) Progress() <-chan converting.Progress { return p.progress }
func (p *pending) Pause() error                         { return p.setPaused(true) }
func (p *pending) Resume() error                        { return p.setPaused(false) }
func (p *pending) Kill()                                { p.end(converting.Exit{Code: -1}) }

func (p *pending) Tail() string {
	select {
	case <-p.stop:
		return p.exit.Tail
	default:
		return ""
	}
}

func (p *pending) Wait() converting.Exit {
	<-p.stop
	return p.exit
}

// shortened answers each output a Short cue made as its probe less the
// cued length.
type shortened struct {
	c    *Conversions
	base media.Prober
}

func (s shortened) Probe(ctx context.Context, path string) (media.Record, error) {
	r, err := s.base.Probe(ctx, path)
	if err != nil {
		return r, err
	}
	s.c.mu.Lock()
	r.Duration -= s.c.shorts[filepath.Base(path)]
	s.c.mu.Unlock()
	return r, nil
}

// proven passes a test run on the first of its pipelines that the key's
// rows list, at four times real time on hardware and real time on the
// processor.
type proven []encode.Pipeline

// adapters are the device each API's test run opens, as Init takes it.
var adapters = map[encode.API]struct {
	device string
	converting.Device
}{
	encode.QSV:          {"/dev/dri/renderD128", converting.Device{Name: "renderD128", Vendor: "Intel", Driver: "i915"}},
	encode.VAAPI:        {"/dev/dri/renderD129", converting.Device{Name: "renderD129", Vendor: "AMD", Driver: "amdgpu"}},
	encode.CUDA:         {"0", converting.Device{Name: "NVIDIA GeForce RTX 3060", Vendor: "NVIDIA", Driver: "550.54"}},
	encode.VideoToolbox: {"", converting.Device{Name: "Apple M2", Vendor: "Apple", Driver: "Metal"}},
}

func (p proven) Signature(context.Context) (converting.Signature, []converting.Device, error) {
	var devices []converting.Device
	for _, pl := range p {
		if a, ok := adapters[pl.API]; ok && !slices.Contains(devices, a.Device) {
			devices = append(devices, a.Device)
		}
	}
	return simconv.Signature, devices, nil
}

func (p proven) Test(_ context.Context, k encode.Key) (converting.Proved, error) {
	proved := converting.Proved{Signature: simconv.Signature}
	for _, row := range pipeline.For(runtime.GOOS, k.Shape) {
		if !slices.Contains(p, row.Pipeline) {
			proved.Tries = append(proved.Tries, converting.Try{Pipeline: row.Pipeline, Err: "This Node has no adapter for it."})
			continue
		}
		proved.Passed, proved.Pipeline, proved.Device, proved.Speed, proved.Pauses = true, row.Pipeline, adapters[row.Pipeline.API].device, 4, true
		if row.Pipeline.Path == encode.Processor {
			proved.Speed = 1
		}
		return proved, nil
	}
	return proved, nil
}

func (p proven) Runs() (role.Fit, role.ProbeReason) {
	switch {
	case slices.ContainsFunc(p, func(pl encode.Pipeline) bool { return pl.Path != encode.Processor }):
		return role.Fits, ""
	case slices.Contains(p, encode.Pipeline{Path: encode.Processor}):
		return role.Weak, role.NoGPU
	}
	return role.Fails, role.FFmpegFails
}

func (proven) Warm(context.Context, func()) error { return nil }
