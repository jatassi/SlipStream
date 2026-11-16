package grabs

import (
	"context"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/backoff"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

type report struct {
	id     storage.UUID
	result result
}

// schedule is the reconciler's memory: what runs, and when a waiting job
// may run again. Only Run's goroutine touches it.
type schedule struct {
	inflight map[storage.UUID]bool
	attempts map[storage.UUID]int
	retryAt  map[storage.UUID]time.Time
	shown    map[GrabID]Progress
	ticked   time.Time
	// limited is when the search limit next frees a slot for a line that
	// waits, zero when none waits on one, and waitEnds when the next wait
	// for a better Release ends, zero when none waits.
	limited  time.Time
	waitEnds time.Time
}

// Run reconciles until ctx ends. Each pass settles what the Admin wrote
// off and what a client removed, lands nudges on the search queue,
// dispatches it, then starts what is due: each line in flight asks its
// Indexer, each queue run whose lines all answered decides, each Work
// whose wait for a better Release ended grabs, each Manual run listens,
// and each unfinished Grab runs its step, unless it runs or backs off. A
// pass runs
// at each change of the roster too. A restart needs no recovery: the
// first pass is the recovery.
func (gr *Grabs) Run(ctx context.Context) error {
	s := schedule{
		inflight: map[storage.UUID]bool{}, attempts: map[storage.UUID]int{}, retryAt: map[storage.UUID]time.Time{},
		shown: map[GrabID]Progress{},
	}
	done := make(chan report)
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		changed, moved := gr.cfg.Roster.Changed(), gr.cfg.Downloads.Moved()
		searches, grabs, err := gr.settled(ctx)
		var wake time.Time
		var q queueJobs
		if err == nil {
			err = gr.drain(ctx)
		}
		if err == nil {
			wake, err = gr.dispatch(ctx)
		}
		if err == nil {
			q, err = gr.queueJobs(ctx)
		}
		now := time.Now()
		if err == nil {
			gr.startDue(ctx, &s, now, done, searches, grabs, q)
		} else if ctx.Err() == nil {
			gr.cfg.Log.Error("read, settle, drain or dispatch the open runs and Grabs", "err", err.Error())
		}
		s.limited, s.waitEnds = wake, q.waits.next
		timer.Reset(s.next(now, grabs, err != nil, gr.cfg.Timings))
		select {
		case r := <-done:
			delete(s.inflight, r.id)
			s.record(r, time.Now(), gr.cfg.Timings, gr.cfg.Rand)
		case <-gr.poke:
		case <-changed:
		case <-moved:
		case <-timer.C:
		case <-ctx.Done():
			for range s.inflight {
				<-done
			}
			return nil
		}
	}
}

// startDue starts every job due now: each line in flight asks, each
// queue run whose lines all answered decides, each Work whose wait ended
// grabs, each Manual run listens and each unfinished Grab runs its step.
func (gr *Grabs) startDue(ctx context.Context, s *schedule, now time.Time, done chan<- report, searches []Search, grabs []Grab, q queueJobs) {
	s.keep(searches, grabs, q)
	for _, l := range q.lines {
		gr.start(ctx, s, now, l.line.Query.UUID, done, func() result { return gr.ask(ctx, l) })
	}
	for _, run := range q.ready {
		gr.start(ctx, s, now, run.ID.UUID, done, func() result { return gr.settle(ctx, run) })
	}
	for _, work := range q.waits.works {
		gr.start(ctx, s, now, work.UUID, done, func() result { return gr.endDue(ctx, work) })
	}
	for _, run := range searches {
		if !run.Manual.IsZero() {
			gr.start(ctx, s, now, run.ID.UUID, done, func() result { return gr.listen(ctx, run) })
		}
	}
	for _, g := range grabs {
		if g.due() {
			gr.start(ctx, s, now, g.ID.UUID, done, func() result { return gr.step(ctx, g) })
		}
	}
	gr.publishProgress(ctx, s, now, grabs)
}

// queueJobs is the search queue's work for a pass: every line in flight,
// every queue run whose lines all answered, were skipped or went late,
// and the waits that end.
type queueJobs struct {
	lines []sent
	ready []Search
	waits waitEnds
}

func (gr *Grabs) queueJobs(ctx context.Context) (queueJobs, error) {
	var out queueJobs
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if out.lines, err = inFlight(ctx, r); err != nil {
			return err
		}
		q := db.New(r)
		rows, err := q.ListReadyRuns(ctx)
		if err != nil {
			return err
		}
		if out.ready, err = runsOf(ctx, q, rows, nil); err != nil {
			return err
		}
		out.waits, err = waitEndsIn(ctx, q, storage.Now())
		return err
	})
	return out, err
}

// settled is the open search runs and unfinished Grabs once the release
// faults, what the Admin wrote off and what a client removed are settled.
func (gr *Grabs) settled(ctx context.Context) ([]Search, []Grab, error) {
	if err := gr.settleFaults(ctx); err != nil {
		return nil, nil, err
	}
	searches, grabs, err := gr.open(ctx)
	if err != nil {
		return nil, nil, err
	}
	changed, err := gr.settleGone(ctx, grabs)
	if err != nil || !changed {
		return searches, grabs, err
	}
	return gr.open(ctx)
}

func (gr *Grabs) start(ctx context.Context, s *schedule, now time.Time, id storage.UUID, done chan<- report, job func() result) {
	if s.inflight[id] || now.Before(s.retryAt[id]) || ctx.Err() != nil {
		return
	}
	s.inflight[id] = true
	go func() { done <- report{id, job()} }()
}

// keep forgets each job neither open nor running, as one whose Work was
// deleted: a vanished job ends as a finished one does, and its past retry
// never wakes the reconciler again.
func (s *schedule) keep(searches []Search, grabs []Grab, q queueJobs) {
	open := make(map[storage.UUID]bool, len(searches)+len(grabs)+len(q.lines)+len(q.waits.works))
	for _, run := range searches {
		open[run.ID.UUID] = true
	}
	for _, g := range grabs {
		open[g.ID.UUID] = true
	}
	for _, l := range q.lines {
		open[l.line.Query.UUID] = true
	}
	for _, w := range q.waits.works {
		open[w.UUID] = true
	}
	for id := range s.retryAt {
		if !open[id] && !s.inflight[id] {
			delete(s.retryAt, id)
			delete(s.attempts, id)
		}
	}
	for id := range s.shown {
		if !open[id.UUID] {
			delete(s.shown, id)
		}
	}
}

func (s *schedule) record(r report, now time.Time, t Timings, rnd *rand.Rand) {
	switch r.result {
	case committed, rerun:
		delete(s.attempts, r.id)
		delete(s.retryAt, r.id)
	case waited:
		s.retryAt[r.id] = now.Add(backoff.Delay(t.RetryMin, t.RetryMax, s.attempts[r.id], rnd))
		s.attempts[r.id]++
	case halted:
	}
}

// next is how long the reconciler sleeps: until the earliest retry of a
// job not running, as a running one may be waiting on the roster, until
// the search limit frees a slot for a waiting line or until the next wait
// for a better Release ends, and no longer than one progress tick while a
// Grab downloads or a read failed.
func (s *schedule) next(now time.Time, grabs []Grab, readFailed bool, t Timings) time.Duration {
	wake := now.Add(time.Hour)
	for _, at := range []time.Time{s.limited, s.waitEnds} {
		if !at.IsZero() && at.Before(wake) {
			wake = at
		}
	}
	if readFailed || slices.ContainsFunc(grabs, func(g Grab) bool { return g.waitsOnTorrent() }) {
		wake = now.Add(t.Progress)
	}
	for id, at := range s.retryAt {
		if !s.inflight[id] && at.Before(wake) {
			wake = at
		}
	}
	return max(wake.Sub(now), 0)
}

// publishProgress sends grabs.changed for each downloading Grab whose
// progress moved, at most once per progress tick.
func (gr *Grabs) publishProgress(ctx context.Context, s *schedule, now time.Time, grabs []Grab) {
	if now.Sub(s.ticked) < gr.cfg.Timings.Progress {
		return
	}
	s.ticked = now
	progress := gr.progress(grabs)
	for _, g := range grabs {
		p := progress[g.ID]
		if p == nil || s.shown[g.ID] == *p {
			continue
		}
		s.shown[g.ID] = *p
		gr.cfg.Changed.Publish(ctx, Changed{GrabID: g.ID.String(), WorkID: g.WorkID.String(), Progress: true})
	}
}

// Progress is a download's bytes done of its total.
type Progress struct {
	Done  int64
	Total int64
}

// progress is each of grabs' progress, when it has one: a downloading
// Grab's live while its Torrent is sampled, else as its Node last reported
// it.
func (gr *Grabs) progress(grabs []Grab) map[GrabID]*Progress {
	out := map[GrabID]*Progress{}
	for _, g := range grabs {
		if g.Stage != Downloading {
			continue
		}
		if p := gr.downloadProgress(g); p != nil {
			out[g.ID] = p
		}
	}
	return out
}

func (gr *Grabs) downloadProgress(g Grab) *Progress {
	if s, ok := gr.cfg.Downloads.Sample(g.Torrent.ID); ok {
		return &Progress{Done: s.Done, Total: s.Wanted}
	}
	p := g.Torrent.Progress
	if p == nil {
		return nil
	}
	return &Progress{Done: p.Done, Total: p.Size}
}

// open reads, in one snapshot, every open search run and unfinished Grab,
// oldest first.
func (gr *Grabs) open(ctx context.Context) ([]Search, []Grab, error) {
	var searches []Search
	var grabs []Grab
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		runs, err := q.ListOpenSearches(ctx)
		if err != nil {
			return err
		}
		if searches, err = runsOf(ctx, q, runs, nil); err != nil {
			return err
		}
		rows, err := q.ListOpenGrabs(ctx)
		if err != nil {
			return err
		}
		grabs, err = loadAll(ctx, r, rows)
		return err
	})
	return searches, grabs, err
}

func (gr *Grabs) wake() {
	select {
	case gr.poke <- struct{}{}:
	default:
	}
}
