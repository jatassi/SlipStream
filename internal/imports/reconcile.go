package imports

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/backoff"
	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// job is one file's next IO, or with position -1 a converge of its Import
// once the Items its files waited busy for are free.
type job struct {
	id       ImportID
	position int
}

// reconcile runs each unfinished Import's next IO until ctx ends: at
// start, after a Touch or a job, once a hold's backoff passes and at
// least every RetryMax. A job runs once at a time, and what it reads it
// reads again in its Write, so a restart anywhere runs it again.
func (im *Imports) reconcile(ctx context.Context) {
	var (
		mu      sync.Mutex
		running = map[job]bool{}
		wg      sync.WaitGroup
	)
	defer wg.Wait()
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-im.wake:
		case <-timer.C:
		case <-ctx.Done():
			return
		}
		due, next, err := im.pass(ctx)
		if err != nil && ctx.Err() == nil {
			im.cfg.Log.Warn("read the unfinished imports", "err", err)
		}
		for j, run := range due {
			mu.Lock()
			if running[j] {
				mu.Unlock()
				continue
			}
			running[j] = true
			mu.Unlock()
			wg.Go(func() {
				if err := run(ctx); err != nil && !errors.Is(err, ErrNoImport) && ctx.Err() == nil {
					im.cfg.Log.Warn("import a file", "import", j.id, "position", j.position, "err", err)
				}
				mu.Lock()
				delete(running, j)
				mu.Unlock()
				im.Touch()
			})
		}
		timer.Reset(next)
	}
}

// pass reads every unfinished Import and answers the jobs due now and
// how long until the next hold's backoff passes.
func (im *Imports) pass(ctx context.Context) (map[job]func(context.Context) error, time.Duration, error) {
	p := plan{due: map[job]func(context.Context) error{}, next: im.cfg.Timings.RetryMax, now: time.Now()}
	err := im.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		rows, err := db.New(r).ListUnfinished(ctx)
		if err != nil {
			return err
		}
		imps, err := allImports(ctx, r, rows)
		if err != nil {
			return err
		}
		for _, imp := range imps {
			if err := im.plan(ctx, r, imp, &p); err != nil {
				return err
			}
		}
		return nil
	})
	return p.due, p.next, err
}

// plan is one pass's jobs due and its next wake.
type plan struct {
	due  map[job]func(context.Context) error
	next time.Duration
	now  time.Time
}

// plan adds imp's jobs to p: a converge once its busy files are free,
// and each file's next IO whose hold's backoff passed.
func (im *Imports) plan(ctx context.Context, r storage.Reader[file], imp Import, p *plan) error {
	freed, err := busyFreed(ctx, r, imp)
	if err != nil {
		return err
	}
	if freed {
		p.due[job{id: imp.ID, position: -1}] = func(ctx context.Context) error {
			_, err := im.write(ctx, imp.ID, func(context.Context, *storage.Tx[file], *Import, *after) error { return nil })
			return err
		}
	}
	t := im.cfg.Timings
	for _, f := range imp.Files {
		run := im.step(imp, f)
		if run == nil {
			continue
		}
		if f.Hold != nil {
			if wait := f.Hold.Since.Time().Add(backoff.Delay(t.RetryMin, t.RetryMax, f.Hold.Attempts-1, im.cfg.Rand)).Sub(p.now); wait > 0 {
				p.next = min(p.next, wait)
				continue
			}
		}
		p.due[job{id: imp.ID, position: f.Position}] = run
	}
	return nil
}

// step is f's next IO: its probe while pending unread, its place while
// placing, and its source's removal once it is placed by a move. A hold
// the Admin must answer has none.
func (im *Imports) step(imp Import, f File) func(context.Context) error {
	switch {
	case f.Hold != nil && f.Hold.offered():
		return nil
	case f.Outcome == Pending && f.Probe == nil && imp.Source != FromRoot:
		return func(ctx context.Context) error { return im.probeFile(ctx, imp, f) }
	case f.Outcome == Placing:
		return func(ctx context.Context) error { return im.place(ctx, imp, f) }
	case f.sourceDue():
		return func(ctx context.Context) error { return im.removeSource(ctx, imp, f) }
	}
	return nil
}

// busyFreed reports whether a file of imp waits busy for Items no other
// Import holds any more.
func busyFreed(ctx context.Context, r storage.Reader[file], imp Import) (bool, error) {
	var items []works.ItemID
	for _, f := range imp.Files {
		if f.busy() {
			items = append(items, f.Reason.Reading...)
		}
	}
	if len(items) == 0 {
		return false, nil
	}
	held, err := holders(ctx, r, items, imp)
	if err != nil {
		return false, err
	}
	for _, f := range imp.Files {
		if f.busy() && !slices.ContainsFunc(f.Reason.Reading, func(it works.ItemID) bool { _, ok := held[it]; return ok }) {
			return true, nil
		}
	}
	return false, nil
}

// sourceDue reports a file placed by a move whose source is still there.
func (f *File) sourceDue() bool {
	return f.Outcome == Placed && f.PlacedBy == ByMove && f.SourceRemoved.IsZero()
}

func (f *File) busy() bool { return f.Outcome == Waiting && f.Reason != nil && f.Reason.Code == Busy }
