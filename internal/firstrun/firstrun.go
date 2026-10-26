// Package firstrun is first setup: the guided steps a new Controller's first
// Admin walks through after the claim, open until that Admin finishes them.
// Each step is exported by the package whose facts it reads, and this
// package imports none of them.
package firstrun

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/jatassi/slipstream/internal/firstrun/internal/db"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/storage"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

// Schema is firstrun's library.db schema set. It reads identity's accounts
// once, to start a Controller claimed before first setup existed finished,
// so it is listed after identity.
func Schema() storage.Schema[file] { return storage.NewSchema[file]("firstrun", schemaFiles) }

// StepID names one first setup step.
type StepID string

// Fact is one of a step's facts, read in the snapshot that every step's
// facts and the stored outcomes are read in together.
type Fact func(ctx context.Context, rd storage.Reader[file]) (bool, error)

// Step is one row of first setup.
type Step struct {
	ID StepID
	// Shows is nil for always.
	Shows Fact
	// Done is nil when only an Admin's Continue marks the step done.
	Done Fact
	// Events are the live events after which Shows or Done may differ.
	Events []live.Event
}

// Outcome is what an Admin's Continue or Skip stores.
type Outcome string

const (
	OutcomeDone    Outcome = "done"
	OutcomeSkipped Outcome = "skipped"
)

// State is a shown step's derived state.
type State string

const (
	StateTodo    State = "todo"
	StateDone    State = "done"
	StateSkipped State = "skipped"
)

// View is first setup as every reader sees it, whoever reads it.
type View struct {
	Open bool
	// Steps are the shown steps, in IDs' order.
	Steps []StepView
}

type StepView struct {
	ID    StepID
	State State
}

// Changed reaches Admins after a step write, after Finish, and after any
// step's Events while first setup is open.
type Changed struct{}

var ChangedEvent = live.AdminEvent[Changed]("firstrun.changed")

var (
	ErrStepUnknown = errors.New("no first setup step has that id")
	ErrFinished    = errors.New("first setup is already finished")
)

type Hub interface {
	Watch(ctx context.Context, events ...live.Event) <-chan struct{}
}

type Publisher interface {
	Publish(ctx context.Context, v Changed)
}

type Config struct {
	Library *storage.DB[file]
	// Steps are every step, in IDs' order.
	Steps   []Step
	Hub     Hub
	Changed Publisher
	Log     *slog.Logger
}

type FirstRun struct {
	cfg Config
	// finished is closed by the one Finish that sets finished_at, so Run
	// returns.
	finished chan struct{}
}

// New panics unless cfg.Steps' ids are IDs, in order: a list mistake is a
// build mistake, and every Controller start in every test hits it. It reads
// nothing.
func New(cfg Config) *FirstRun {
	ids := make([]StepID, len(cfg.Steps))
	for i, s := range cfg.Steps {
		ids[i] = s.ID
	}
	if !slices.Equal(ids, IDs()) {
		panic(fmt.Sprintf("firstrun: the steps are %q but ids.go declares %q, in that order", ids, IDs()))
	}
	return &FirstRun{cfg: cfg, finished: make(chan struct{})}
}

// Run publishes Changed after any step's Events until first setup is
// finished or ctx ends. It returns at once on a finished Controller.
func (f *FirstRun) Run(ctx context.Context) error {
	open, err := f.IsOpen(ctx)
	if err != nil || !open {
		return err
	}
	var events []live.Event
	for _, s := range f.cfg.Steps {
		for _, e := range s.Events {
			if !slices.Contains(events, e) {
				events = append(events, e)
			}
		}
	}
	woken := f.cfg.Hub.Watch(ctx, events...)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-f.finished:
			return nil
		case <-woken:
			f.cfg.Changed.Publish(ctx, Changed{})
		}
	}
}

// View reads every step's facts, the stored outcomes and finished_at in one
// snapshot, then derives the view.
func (f *FirstRun) View(ctx context.Context) (View, error) {
	var v View
	err := f.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
		var err error
		v, err = f.viewIn(ctx, rd)
		return err
	})
	return v, err
}

func (f *FirstRun) viewIn(ctx context.Context, rd storage.Reader[file]) (View, error) {
	reads := make([]read, len(f.cfg.Steps))
	for i, s := range f.cfg.Steps {
		r, err := readStep(ctx, rd, s)
		if err != nil {
			return View{}, fmt.Errorf("read first setup step %s: %w", s.ID, err)
		}
		reads[i] = r
	}
	q := db.New(rd)
	rows, err := q.ListSteps(ctx)
	if err != nil {
		return View{}, err
	}
	stored := make(map[StepID]Outcome, len(rows))
	for _, r := range rows {
		stored[StepID(r.ID)] = Outcome(r.Outcome)
	}
	finishedAt, err := q.GetFinishedAt(ctx)
	if err != nil {
		return View{}, err
	}
	return viewOf(reads, stored, !finishedAt.IsZero()), nil
}

// readStep reads s's facts. A hidden step's Done is never read.
func readStep(ctx context.Context, rd storage.Reader[file], s Step) (read, error) {
	r := read{id: s.ID, shows: true}
	var err error
	if s.Shows != nil {
		if r.shows, err = s.Shows(ctx, rd); err != nil {
			return read{}, err
		}
	}
	if r.shows && s.Done != nil {
		if r.done, err = s.Done(ctx, rd); err != nil {
			return read{}, err
		}
	}
	return r, nil
}

// Set stores an Admin's Continue or Skip on id and returns the new view. It
// refuses an id IDs doesn't list with ErrStepUnknown, and any write once
// first setup is finished with ErrFinished. A hidden step's row is kept,
// and nothing reads it while the step is hidden. Storing the outcome a step
// already has changes nothing and publishes nothing.
func (f *FirstRun) Set(ctx context.Context, id StepID, o Outcome) (View, error) {
	if !slices.Contains(IDs(), id) {
		return View{}, ErrStepUnknown
	}
	var (
		v       View
		changed bool
	)
	err := f.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		finishedAt, err := q.GetFinishedAt(ctx)
		if err != nil {
			return err
		}
		if !finishedAt.IsZero() {
			return ErrFinished
		}
		n, err := q.PutStep(ctx, db.PutStepParams{ID: db.StepID(id), Outcome: db.Outcome(o), At: storage.Now()})
		if err != nil {
			return err
		}
		changed = n > 0
		v, err = f.viewIn(ctx, tx)
		return err
	})
	if err != nil {
		return View{}, err
	}
	if changed {
		f.cfg.Changed.Publish(ctx, Changed{})
	}
	return v, nil
}

// Finish ends first setup for good. A second call changes nothing and
// publishes nothing.
func (f *FirstRun) Finish(ctx context.Context) error {
	var n int64
	err := f.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		n, err = db.New(tx).SetFinishedAt(ctx, storage.Now())
		return err
	})
	if err != nil || n == 0 {
		return err
	}
	f.cfg.Log.Info("first setup finished")
	close(f.finished)
	f.cfg.Changed.Publish(ctx, Changed{})
	return nil
}

// IsOpen is whether first setup is still open.
func (f *FirstRun) IsOpen(ctx context.Context) (bool, error) {
	finishedAt, err := db.New(f.cfg.Library).GetFinishedAt(ctx)
	return finishedAt.IsZero(), err
}
