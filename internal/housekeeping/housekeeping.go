package housekeeping

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/nodes"
	housekeepingv1 "github.com/jatassi/slipstream/internal/pb/slipstream/housekeeping/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/housekeeping/v1/housekeepingv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

const day = 24 * time.Hour

var cadence = setting.Duration{Key: "tasks.housekeeping.cadence", Label: "Runs every", Default: day, Min: day, Max: 7 * day, Step: day}

const (
	runHistory tasks.UnitKey = "run-history"
	joinCodes  tasks.UnitKey = "join-codes"
	sessions   tasks.UnitKey = "sessions"
	library    tasks.UnitKey = "library"
)

type Timings struct {
	// PruneBatch is how many rows one prune Write deletes, at most.
	PruneBatch int `yaml:"prune_batch"`
}

func DefaultTimings() Timings { return Timings{PruneBatch: 1000} }

// Retention is "Keep history for": the task's Tuning, and the age past
// which every Aged step deletes.
type Retention interface {
	setting.Panel
	Keep(ctx context.Context) (time.Duration, error)
}

type Config struct {
	Library  *storage.DB[storage.Library]
	Nodes    *nodes.Service
	Identity *identity.Service
	Tidy     func(node role.NodeID) housekeepingv1connect.HousekeepingServiceHandler
	// Keep is nil in a Config that runs no Aged step.
	Keep  Retention
	Batch int
	// Steps are the features' own prunes, each run on the Controller
	// after the sessions and before the databases.
	Steps []Step
	// Reconciling is whether a restored Controller is Reconciling, which
	// pauses each Step declared Pauses.
	Reconciling func(ctx context.Context) (bool, error)
}

// Step is one feature's prune, a unit on the task's page. Only Plain and
// Aged make one.
type Step struct {
	Key   tasks.UnitKey
	Label string
	// WhileReconciling is what the step does while a restored Controller
	// is Reconciling, as a task declares it.
	WhileReconciling tasks.WhileReconciling
	run              func(ctx context.Context, before time.Time, batch int, p *tasks.Progress) error
}

// Plain is a step with its own age rule.
func Plain(key tasks.UnitKey, label string, while tasks.WhileReconciling, run func(ctx context.Context) error) Step {
	return Step{Key: key, Label: label, WhileReconciling: while, run: func(ctx context.Context, _ time.Time, _ int, _ *tasks.Progress) error { return run(ctx) }}
}

// Swept is what one batch Write saw: how many rows it read and deleted,
// and the last row it read, where the next batch starts after.
type Swept[C any] struct {
	Seen, Deleted int
	Last          C
}

// Tally is what a whole prune deleted and kept.
type Tally struct{ Deleted, Kept int }

// Sweep deletes, in one Write, up to limit rows older than before that
// sort after after.
type Sweep[C any] func(ctx context.Context, before time.Time, after C, limit int) (Swept[C], error)

// Batch deletes, in one Write, up to limit rows that sort after after.
type Batch[C any] func(ctx context.Context, after C, limit int) (Swept[C], error)

// Drain runs b from the zero cursor, each batch after the last row the
// previous one saw, until a batch sees fewer than limit rows. Rows a batch
// keeps never stall it.
func Drain[C any](ctx context.Context, limit int, b Batch[C]) (Tally, error) {
	var (
		t     Tally
		after C
	)
	for {
		s, err := b(ctx, after, limit)
		if err != nil {
			return t, err
		}
		t.Deleted += s.Deleted
		t.Kept += s.Seen - s.Deleted
		if s.Seen < limit {
			return t, nil
		}
		after = s.Last
	}
}

// Aged is a step that deletes rows older than Keep history for through
// Drain, and notes its unit. why is what kept rows are, after their count:
// "that are still in use".
func Aged[C any](key tasks.UnitKey, label string, while tasks.WhileReconciling, noun setting.Noun, why string, sweep Sweep[C]) Step {
	return Step{Key: key, Label: label, WhileReconciling: while, run: func(ctx context.Context, before time.Time, batch int, p *tasks.Progress) error {
		t, err := Drain(ctx, batch, func(ctx context.Context, after C, limit int) (Swept[C], error) {
			return sweep(ctx, before, after, limit)
		})
		if err != nil {
			return err
		}
		p.Note(t.words(noun, why))
		return nil
	}}
}

// words is a prune's unit note: "Deleted 1,204 Grabs. Kept 12 that are
// still in use." why is plural, as most counts are.
func (t Tally) words(noun setting.Noun, why string) string {
	if t.Kept == 1 {
		why = strings.Replace(why, "that are ", "that is ", 1)
	}
	switch {
	case t.Deleted == 0 && t.Kept == 0:
		return "Nothing was old enough to delete."
	case t.Deleted == 0:
		return fmt.Sprintf("Kept %s %s %s.", grouped(t.Kept), noun.Of(int64(t.Kept)), why)
	case t.Kept == 0:
		return fmt.Sprintf("Deleted %s %s.", grouped(t.Deleted), noun.Of(int64(t.Deleted)))
	}
	return fmt.Sprintf("Deleted %s %s. Kept %s %s.", grouped(t.Deleted), noun.Of(int64(t.Deleted)), grouped(t.Kept), why)
}

func grouped(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func Task(cfg Config) tasks.Task {
	return tasks.Task{
		ID:               "housekeeping",
		Name:             "Housekeeping",
		About:            "Deletes old task runs, expired Join Codes, idle sessions, old Indexer queries and old Media Server history, and History, Grabs, Imports and import records older than Keep history for, then optimizes every database.",
		Group:            tasks.ControllerGroup,
		Cadence:          tasks.InWindow{Days: cadence},
		WhileReconciling: tasks.Runs,
		Overdue:          tasks.Wait,
		Noun:             setting.Noun{One: "step", Many: "steps"},
		Units:            cfg.units,
		Run:              cfg.run,
		Tuning:           cfg.Keep,
	}
}

func (c Config) units(ctx context.Context, r storage.Reader[storage.Library]) ([]tasks.Unit, error) {
	named, err := nodes.Names(ctx, r)
	if err != nil {
		return nil, err
	}
	out := []tasks.Unit{
		{Key: runHistory, Label: "Run history", Place: tasks.OnController()},
		{Key: joinCodes, Label: "Join Codes", Place: tasks.OnController()},
		{Key: sessions, Label: "Sessions", Place: tasks.OnController()},
	}
	for _, s := range c.Steps {
		out = append(out, tasks.Unit{Key: s.Key, Label: s.Label, Place: tasks.OnController()})
	}
	for _, n := range named {
		out = append(out, tasks.Unit{Key: tasks.UnitKey("node:" + n.ID.String()), Label: "Local database on " + n.Name, Place: tasks.TiedTo(n.ID)})
	}
	return append(out, tasks.Unit{Key: library, Label: "Library database", Place: tasks.OnController()}), nil
}

func (c Config) run(ctx context.Context, u tasks.Unit, p *tasks.Progress) error {
	switch u.Key {
	case runHistory:
		return tasks.PruneRuns(ctx, c.Library)
	case joinCodes:
		return c.Nodes.PruneJoinCodes(ctx)
	case sessions:
		return c.Identity.PruneSessions(ctx)
	case library:
		return c.Library.Optimize(ctx)
	}
	for _, s := range c.Steps {
		if s.Key == u.Key {
			return c.runStep(ctx, s, p)
		}
	}
	node, _ := u.Place.Node()
	_, err := c.Tidy(node).Tidy(ctx, &housekeepingv1.TidyRequest{})
	return err
}

// pausedStep is the line of a Step paused while Reconciling.
const pausedStep = "Skipped because SlipStream is Reconciling after a restore. The first Housekeeping run after Reconciling finishes deletes them."

func (c Config) runStep(ctx context.Context, s Step, p *tasks.Progress) error {
	if s.WhileReconciling == tasks.Pauses {
		paused, err := c.Reconciling(ctx)
		if err != nil {
			return err
		}
		if paused {
			return tasks.SkipOwned(pausedStep)
		}
	}
	var before time.Time
	if c.Keep != nil {
		keep, err := c.Keep.Keep(ctx)
		if err != nil {
			return err
		}
		before = time.Now().Add(-keep)
	}
	return s.run(ctx, before, c.Batch, p)
}

type Service struct {
	Local *storage.DB[storage.Local]
}

func (s Service) Tidy(ctx context.Context, _ *housekeepingv1.TidyRequest) (*housekeepingv1.TidyResponse, error) {
	if err := s.Local.Optimize(ctx); err != nil {
		return nil, rpc.Errorf(rpc.CodeInternal, "%v", err)
	}
	return &housekeepingv1.TidyResponse{}, nil
}
