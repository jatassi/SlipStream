package tasks

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks/internal/db"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("tasks", schemaFiles) }

type Roster interface {
	Self() role.NodeID
	// Standing is how node stands with the Controller, and since when.
	Standing(node role.NodeID) (nodes.Standing, time.Time)
	// Route is where r work homed on home goes now, or why it waits.
	Route(r role.Name, home role.NodeID) (role.NodeID, error)
	Get(ctx context.Context, id role.NodeID) (nodes.Node, error)
	// Names is every Node's name, removed Nodes included.
	Names(ctx context.Context) (map[role.NodeID]nodes.Named, error)
}

// Changed tells Admins that a task, its runs or the schedule changed. An
// empty TaskID is the schedule settings or the set of tasks shown.
type Changed struct {
	TaskID TaskID `json:"taskId"`
}

const everyTask TaskID = ""

type Progressed struct {
	RunID  RunID  `json:"runId"`
	TaskID TaskID `json:"taskId"`
	Done   int    `json:"done"`
	Total  int    `json:"total"`
}

type Hub interface {
	Watch(ctx context.Context, events ...live.Event) <-chan struct{}
}

type ChangedPublisher interface {
	Publish(ctx context.Context, v Changed)
}

type ProgressPublisher interface {
	Publish(ctx context.Context, v Progressed)
}

type Config struct {
	Library *storage.DB[file]
	// Catalog is every task, in the Tasks page's order within each group
	// and the order window tasks start in.
	Catalog []Task
	Roster  Roster
	// Zone is where the Maintenance window's wall-clock times are kept.
	Zone       *time.Location
	Rand       *rand.Rand
	Hub        Hub
	Changed    ChangedPublisher
	Progressed ProgressPublisher
	Log        *slog.Logger
}

var (
	ErrUnknownTask = errors.New("no task shown has this id")
	ErrNotOpen     = errors.New("no open run has this id")
)

// ErrRunning refuses Run now while the task has a run open.
type ErrRunning struct{ Since time.Time }

func (e *ErrRunning) Error() string { return "the task has run since " + e.Since.Format(time.RFC3339) }

// ErrWaiting refuses Run now while every unit is tied to a Node that
// can't start work.
type ErrWaiting struct {
	Node     string
	Standing nodes.Standing
	Since    time.Time
}

func (e *ErrWaiting) Error() string {
	return "every unit waits for " + e.Node + " since " + e.Since.Format(time.RFC3339)
}

const crashCause = "SlipStream stopped mid-run"

type Scheduler struct {
	cfg  Config
	byID map[TaskID]Task
	self role.NodeID
	wake chan struct{}
	asks chan ask
	runs sync.WaitGroup

	mu      sync.Mutex
	started time.Time
	running map[RunID]*active
	cut     map[TaskID]bool
}

// Open checks the catalog, drops rows of tasks it no longer lists, adds
// rows for new ones and closes every run a previous process left open
// without advancing its slot. Running it twice changes nothing more.
func Open(ctx context.Context, cfg Config) (*Scheduler, error) {
	s := &Scheduler{
		cfg:     cfg,
		byID:    catalogByID(cfg.Catalog),
		self:    cfg.Roster.Self(),
		wake:    make(chan struct{}, 1),
		asks:    make(chan ask),
		running: map[RunID]*active{},
		cut:     map[TaskID]bool{},
	}
	err := cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		if err := s.syncTasks(ctx, q); err != nil {
			return err
		}
		open, err := q.ListOpenRuns(ctx)
		if err != nil {
			return err
		}
		for _, r := range open {
			if err := s.closeCut(ctx, q, r); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("open the tasks: %w", err)
	}
	return s, nil
}

func (s *Scheduler) syncTasks(ctx context.Context, q *db.Queries) error {
	rows, err := q.ListTasks(ctx)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if _, ok := s.byID[row.ID]; !ok {
			if err := q.DeleteTask(ctx, row.ID); err != nil {
				return err
			}
		}
	}
	now := storage.Now()
	for _, t := range s.cfg.Catalog {
		if err := q.InsertTask(ctx, db.InsertTaskParams{ID: t.ID, FirstSeenAt: now}); err != nil {
			return err
		}
	}
	return nil
}

func catalogByID(catalog []Task) map[TaskID]Task {
	byID := map[TaskID]Task{}
	for _, t := range catalog {
		if _, dup := byID[t.ID]; dup {
			panic(fmt.Sprintf("tasks: the catalog lists %q twice", t.ID))
		}
		if err := t.Setting().Check(); err != nil {
			panic(fmt.Sprintf("tasks: %s: %v", t.ID, err))
		}
		if t.Prepare != nil && t.Shown == nil {
			panic(fmt.Sprintf("tasks: %s has Prepare and no Shown", t.ID))
		}
		byID[t.ID] = t
	}
	return byID
}

func (s *Scheduler) closeCut(ctx context.Context, q *db.Queries, r db.TasksRun) error {
	units, err := q.ListUnits(ctx, r.TaskID)
	if err != nil {
		return err
	}
	ended := r.StartedAt
	var lines []line
	var results []result
	for _, u := range units {
		reported := u.LastRunID != nil && *u.LastRunID == r.ID
		if !reported && r.Kind == ByReturn && (u.DueOnReturn == nil || *u.DueOnReturn == 0) {
			continue
		}
		res := result{outcome: Canceled}
		if reported {
			res = result{outcome: *u.LastOutcome, cause: deref(u.LastCause)}
			if u.LastEndedAt.Compare(ended) > 0 {
				ended = u.LastEndedAt
			}
		}
		results = append(results, res)
		lines = append(lines, lineOf(u.UnitKey, u.Label, u.LastNode, res))
	}
	_, summary, _ := rollUp(s.byID[r.TaskID].Noun, results, true)
	if err := closeRun(ctx, q, r.ID, ended, RunCanceled, summary, crashCause, lines); err != nil {
		return err
	}
	if r.Kind == BySchedule || r.Kind == ByCatchUp {
		s.cut[r.TaskID] = true
	}
	return nil
}

type line struct {
	Key     UnitKey `json:"key"`
	Label   string  `json:"label"`
	Node    string  `json:"node,omitempty"`
	Outcome Outcome `json:"outcome"`
	Cause   string  `json:"cause,omitempty"`
}

func lineOf(key UnitKey, label string, node role.NodeID, r result) line {
	l := line{Key: key, Label: label, Outcome: r.outcome, Cause: r.cause}
	if !node.IsZero() {
		l.Node = node.String()
	}
	return l
}

func closeRun(ctx context.Context, q *db.Queries, id RunID, ended storage.Time, r Result, summary, cause string, lines []line) error {
	units, err := json.Marshal(lines)
	if err != nil {
		return fmt.Errorf("run %d lines: %w", id, err)
	}
	return q.CloseRun(ctx, db.CloseRunParams{
		ID: id, EndedAt: ended, Result: &r, Summary: &summary, Cause: optional(cause), Units: new(string(units)),
	})
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Wake makes the clock plan again now: after a Node's link changes, or
// anything else a plan reads.
func (s *Scheduler) Wake() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *Scheduler) changed(ctx context.Context, id TaskID) {
	s.cfg.Changed.Publish(ctx, Changed{TaskID: id})
}
