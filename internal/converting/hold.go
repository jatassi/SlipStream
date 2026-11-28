package converting

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"runtime"
	"slices"
	"sync"

	"github.com/jatassi/slipstream/internal/converting/pipeline"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/media"
	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

type HolderConfig struct {
	Converter Converter
	Proofs    Proofs
	Media     media.Prober
	// Scratch is the scratch folder now: config.yaml can change it in step.
	Scratch func() string
	// Override is the configured ffmpeg path, empty for the bundled one.
	Override func() string
	// Local is storing.Store.Local.
	Local func(ctx context.Context, root storage.UUID, rel storing.Rel) (string, error)
	Disk  storing.Disk
	// DB holds the result outbox each report is queued in.
	DB      *storage.DB[storage.Local]
	Timings Timings
	Log     *slog.Logger
}

type holderOf = role.Holder[*cv1.Intent, *cv1.Facts, *cv1.Result]

// Holder is the holder half, which Def.Hold runs.
type Holder struct {
	cfg HolderConfig

	mu   sync.Mutex
	runs map[string]*running
	wg   sync.WaitGroup
}

// running is one job this holder took up at one attempt. It stays after
// its run ends until the job leaves the intent, so a run that ended isn't
// started again or reported Lost.
type running struct {
	attempt uint32
	run     Run
	ended   bool
	paused  bool
	killed  bool
	// wake tells the run's watch that paused changed.
	wake     chan struct{}
	progress Progress
	job      pipeline.Job
	output   string
}

func Hold(cfg HolderConfig) *Holder { return &Holder{cfg: cfg, runs: map[string]*running{}} }

func (h *Holder) Run(ctx context.Context, rh holderOf) error {
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		h.stop()
	}()
	f := h.newFacts(ctx)
	h.wg.Go(func() { f.loop(ctx, rh) })
	for snap := range rh.Intent(ctx) {
		p := reconcile(h.have(), snap.Intent, snap.MayStart())
		for _, id := range p.Drop {
			h.drop(id)
		}
		for _, j := range p.Lost {
			h.took(j, &running{ended: true})
			h.report(ctx, rh, j, &cv1.Result{Result: &cv1.Result_Lost{Lost: &cv1.Lost{}}})
		}
		for _, j := range p.Pause {
			h.setPaused(ctx, rh, j, true)
		}
		for _, j := range p.Resume {
			h.setPaused(ctx, rh, j, false)
		}
		for _, j := range p.Start {
			h.start(ctx, rh, j)
		}
		h.sweep(ctx, snap.Intent.GetKeep())
	}
	return nil
}

// stop kills every run, forgets every job and waits for the watches, so a
// restarted Run reports Lost for each job the intent says runs.
func (h *Holder) stop() {
	h.mu.Lock()
	for id := range h.runs {
		h.killLocked(id)
	}
	clear(h.runs)
	h.mu.Unlock()
	h.wg.Wait()
}

// Release kills every tree and deletes the scratch folder's in/, out/ and
// clips/.
func (h *Holder) Release(ctx context.Context) error {
	h.stop()
	var errs []error
	for _, dir := range []string{inDir, outDir, clipsDir} {
		errs = append(errs, h.cfg.Disk.RemoveAll(ctx, h.cfg.Disk.Host().Style.Join(h.cfg.Scratch(), dir)))
	}
	return errors.Join(errs...)
}

func (h *Holder) have() []Have {
	h.mu.Lock()
	defer h.mu.Unlock()
	have := make([]Have, 0, len(h.runs))
	for id, r := range h.runs {
		have = append(have, Have{ID: id, Attempt: r.attempt, Paused: r.paused, Ended: r.ended})
	}
	slices.SortFunc(have, func(a, b Have) int { return cmp.Compare(a.ID, b.ID) })
	return have
}

func (h *Holder) took(j *cv1.Job, r *running) {
	r.attempt = j.GetAttempt()
	h.mu.Lock()
	h.runs[string(j.GetConversionId())] = r
	h.mu.Unlock()
}

func (h *Holder) drop(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.killLocked(id)
	delete(h.runs, id)
}

func (h *Holder) killLocked(id string) {
	r := h.runs[id]
	if r.ended {
		return
	}
	r.killed, r.ended = true, true
	r.run.Kill()
}

func (h *Holder) runOf(j *cv1.Job) *running {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.runs[string(j.GetConversionId())]
}

func (h *Holder) setPaused(ctx context.Context, rh holderOf, j *cv1.Job, paused bool) {
	r := h.runOf(j)
	write, result := r.run.Resume, &cv1.Result{Result: &cv1.Result_Resumed{Resumed: &cv1.Resumed{}}}
	if paused {
		write, result = r.run.Pause, &cv1.Result{Result: &cv1.Result_Paused{Paused: &cv1.Paused{}}}
	}
	if err := write(); err != nil {
		h.cfg.Log.Debug("ffmpeg's input is closed, so its exit is reported instead", "err", err.Error())
		return
	}
	h.mu.Lock()
	r.paused = paused
	h.mu.Unlock()
	select {
	case r.wake <- struct{}{}:
	default:
	}
	h.report(ctx, rh, j, result)
}

func (h *Holder) start(ctx context.Context, rh holderOf, j *cv1.Job) {
	job, output, err := h.jobOf(ctx, j)
	var run Run
	if err == nil {
		run, err = h.launch(ctx, job)
	}
	if err != nil {
		h.cfg.Log.Warn("start a Conversion's run", "err", err.Error())
		h.took(j, &running{ended: true})
		h.report(ctx, rh, j, fault(Crash, ""))
		return
	}
	r := &running{run: run, wake: make(chan struct{}, 1), job: job, output: output}
	h.took(j, r)
	h.report(ctx, rh, j, &cv1.Result{Result: &cv1.Result_Started{Started: &cv1.Started{}}})
	h.wg.Go(func() { h.watch(ctx, rh, j, r) })
}

func (h *Holder) launch(ctx context.Context, job pipeline.Job) (Run, error) {
	args, err := pipeline.Build(job)
	if err != nil {
		return nil, err
	}
	if err := h.mkdir(ctx, job.Output); err != nil {
		return nil, err
	}
	return h.cfg.Converter.Start(ctx, args)
}

// mkdir makes the output's folder, and the scratch folder above out/.
func (h *Holder) mkdir(ctx context.Context, output string) error {
	style := h.cfg.Disk.Host().Style
	dir, _ := style.Parent(output)
	var dirs []string
	if dir == style.Join(h.cfg.Scratch(), outDir) {
		dirs = append(dirs, h.cfg.Scratch())
	}
	for _, d := range append(dirs, dir) {
		if err := h.cfg.Disk.Mkdir(ctx, d); err != nil && !errors.Is(err, fs.ErrExist) {
			return err
		}
	}
	return nil
}

// jobOf resolves the Controller-named places under this Node's folders and
// reads the job's settings, source and languages. output is the finished
// name; the run writes output.part.
func (h *Holder) jobOf(ctx context.Context, j *cv1.Job) (job pipeline.Job, output string, err error) {
	input, err := h.place(ctx, j.GetLibrary(), j.GetScratch())
	if err != nil {
		return job, "", err
	}
	if output, err = h.place(ctx, j.GetHolding(), j.GetScratchOut()); err != nil {
		return job, "", err
	}
	s := j.GetSettings()
	settings, err := encode.Parse(encode.Raw{
		Codec: s.GetCodec(), MaxHeight: int(s.GetMaxHeight()), Kbps: int(s.GetKbps()), Depth: int(s.GetDepth()),
		Range: s.GetRange(), KeepDV: s.GetKeepDv(), Curve: s.GetCurve(), Audio: s.GetAudio(),
		DropCommentary: s.GetDropCommentary(), Container: s.GetContainer(),
	})
	if err != nil {
		return job, "", fmt.Errorf("read the settings: %w", err)
	}
	var source media.Record
	if err := json.Unmarshal(j.GetSource().GetRecord(), &source); err != nil {
		return job, "", fmt.Errorf("read the source's record: %w", err)
	}
	var keep []language.Code
	for _, l := range j.GetKeepLanguages() {
		c, err := language.Parse(l)
		if err != nil {
			return job, "", err
		}
		keep = append(keep, c)
	}
	p := j.GetPipeline()
	return pipeline.Job{
		Input: input, Output: output + partSuffix, OS: runtime.GOOS, Settings: settings, Source: source, Keep: keep,
		Pipeline: encode.Pipeline{Path: encode.Path(p.GetPath()), API: encode.API(p.GetApi())}, Device: j.GetDevice(),
	}, output, nil
}

// place is a file in a library root, or a name under the scratch folder.
func (h *Holder) place(ctx context.Context, lib *cv1.LibraryPath, scratch string) (string, error) {
	if lib != nil {
		root, err := storage.UUIDFromBytes(lib.GetRootId())
		if err != nil {
			return "", err
		}
		rel, err := storing.ParseRel(lib.GetRel())
		if err != nil {
			return "", err
		}
		return h.cfg.Local(ctx, root, rel)
	}
	return h.scratchPath(scratch)
}

func (h *Holder) scratchPath(name string) (string, error) {
	rel, err := storing.ParseRel(name)
	if err != nil {
		return "", err
	}
	return h.cfg.Disk.Host().Style.Under(h.cfg.Scratch(), rel), nil
}
