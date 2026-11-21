package backups

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/jatassi/slipstream/internal/backups/archive"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

const TaskID tasks.TaskID = "backup"

const backupUnit tasks.UnitKey = "backup"

// Cadence is Back up every, the Backup task's cadence.
var Cadence = setting.Duration{Key: "tasks.backup.cadence", Label: "Back up every", Default: day, Min: day, Max: 7 * day, Step: day}

type Config struct {
	Library *storage.DB[storage.Library]
	// Keys is library.key, which seals backup.key.
	Keys   *secret.Keys
	Engine *Engine
	// Start is the tasks Scheduler's, bound once it opens.
	Start func(ctx context.Context, st tasks.Start) (tasks.Started, error)
	// Reconciling is whether the library is Reconciling now, which each
	// Backup records.
	Reconciling func(ctx context.Context, r storage.Reader[storage.Library]) (bool, error)
	History     entry.Recorder
	Log         *slog.Logger
}

type Backups struct {
	cfg   Config
	mu    sync.Mutex
	queue []*Run
}

// Open finishes what a stopped Backup left, takes an update Backup when
// this binary is newer than the last one that started on the library and
// none of this version is listed, and records this version as started.
// It runs before any service starts.
func Open(ctx context.Context, cfg Config) (*Backups, error) {
	b := &Backups{cfg: cfg}
	if _, err := cfg.Engine.Vault(cfg.Keys); err != nil {
		return nil, err
	}
	if err := cfg.Engine.Recover(); err != nil {
		return nil, err
	}
	index := cfg.Engine.Index()
	version := cfg.Engine.cfg.Version
	ledger := index.Read()
	listed := slices.ContainsFunc(ledger.Backups, func(e Entry) bool { return e.Trigger == archive.Update && e.Version == version })
	if newer(version, ledger.StartedVersion) && !listed {
		if _, err := b.take(ctx, request{Trigger: archive.Update}); err != nil {
			return nil, err
		}
	}
	if err := index.Update(func(l *Ledger) error { l.StartedVersion = version; return nil }); err != nil {
		return nil, err
	}
	return b, nil
}

type Request struct {
	Trigger archive.Trigger
	// By is the Admin who pressed Back up now, zero otherwise.
	By storage.UUID
}

// Run answers Wait once its Backup ends.
type Run struct {
	req  Request
	done chan result
}

type result struct {
	entry Entry
	err   error
}

func (r *Run) Wait(ctx context.Context) (Entry, error) {
	select {
	case res := <-r.done:
		return res.entry, res.err
	case <-ctx.Done():
		return Entry{}, fmt.Errorf("wait for the Backup: %w", ctx.Err())
	}
}

// Take starts a run of the backup task's Backup unit ByEvent for req. It
// is the one way to take a Backup outside the schedule. Back up now
// doesn't Wait; slipstream backup and each step that backs up before it
// changes the library do.
func (b *Backups) Take(ctx context.Context, req Request) (*Run, error) {
	run := &Run{req: req, done: make(chan result, 1)}
	b.mu.Lock()
	b.queue = append(b.queue, run)
	b.mu.Unlock()
	if err := b.start(ctx); err != nil {
		b.mu.Lock()
		b.queue = slices.DeleteFunc(b.queue, func(r *Run) bool { return r == run })
		b.mu.Unlock()
		return nil, err
	}
	return run, nil
}

func (b *Backups) start(ctx context.Context) error {
	_, err := b.cfg.Start(ctx, tasks.Start{Task: TaskID, Units: []tasks.UnitKey{backupUnit}, Trigger: tasks.ByEvent})
	return err
}

func (b *Backups) Path(e Entry) string { return b.cfg.Engine.path(e.Name) }

// Task is the backup Scheduled task: first in the Maintenance window, and
// run at once when the window passed while SlipStream was off.
func (b *Backups) Task() tasks.Task {
	return tasks.Task{
		ID:               TaskID,
		Name:             "Backup",
		About:            "Takes a Backup of the library, checks that it opens, and deletes older Backups past their keep time.",
		Group:            tasks.ControllerGroup,
		Cadence:          tasks.InWindow{Days: Cadence},
		Overdue:          tasks.CatchUp,
		WhileReconciling: tasks.Runs,
		Noun:             setting.Noun{One: "Backup", Many: "Backups"},
		Units: func(context.Context, storage.Reader[storage.Library]) ([]tasks.Unit, error) {
			return []tasks.Unit{{Key: backupUnit, Label: "Library", Place: tasks.OnController()}}, nil
		},
		Run: b.run,
	}
}

// run takes the Backup the oldest waiting Take asked for, or a scheduled
// one when none waits, and starts the unit again while another waits.
func (b *Backups) run(ctx context.Context, _ tasks.Unit, p *tasks.Progress) error {
	b.mu.Lock()
	var run *Run
	if len(b.queue) > 0 {
		run, b.queue = b.queue[0], b.queue[1:]
	}
	more := len(b.queue) > 0
	b.mu.Unlock()
	req := request{Trigger: archive.Scheduled}
	if run != nil {
		req.Trigger = run.req.Trigger
		if !run.req.By.IsZero() {
			req.By = run.req.By.String()
		}
	}
	taken, err := b.take(ctx, req)
	if err != nil {
		if rerr := b.recordFailed(ctx, req.Trigger); rerr != nil {
			b.cfg.Log.Error("record a failed Backup", "err", rerr)
		}
	} else {
		p.Note(fmt.Sprintf("Took a %s Backup and checked it.", Size(taken.Size)))
	}
	if run != nil {
		run.done <- result{entry: taken, err: err}
	}
	if more {
		if serr := b.start(context.WithoutCancel(ctx)); serr != nil {
			b.cfg.Log.Error("start the next Backup", "err", serr)
		}
	}
	return err
}

func (b *Backups) take(ctx context.Context, req request) (Entry, error) {
	err := b.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		retention, err := ReadRetention(ctx, r)
		if err != nil {
			return err
		}
		req.Retention = &retention
		req.WhileReconciling, err = b.cfg.Reconciling(ctx, r)
		return err
	})
	if err != nil {
		return Entry{}, err
	}
	return b.cfg.Engine.take(ctx, b.cfg.Library, req)
}

type FailedFacts struct {
	Trigger archive.Trigger `json:"trigger"`
}

func (FailedFacts) About() entry.About { return entry.About{} }

// FailedRow is Backup failed, recorded when a run of the backup task fails.
var FailedRow = entry.Row[FailedFacts]{
	Spec:   entry.Spec{Kind: "backup_failed", Label: "Backup failed", Group: entry.Backups, OnByDefault: true, BellByDefault: true},
	Render: renderFailed,
}

func Entries() []entry.Entry { return []entry.Entry{FailedRow} }

func renderFailed(f FailedFacts, _ entry.Context) entry.Message {
	what, ok := failedWords[f.Trigger]
	if !ok {
		what = "The Backup failed."
	}
	return entry.Message{
		Title: "Backup failed",
		Lines: []string{what + " Open the Backup task for the cause."},
		Link:  entry.Link{Path: "/tasks/" + string(TaskID)},
		Tone:  entry.Fail,
	}
}

// failedWords names the failed Backup by why it was taken. The kinds whose
// features haven't shipped read as a plain Backup.
var failedWords = map[archive.Trigger]string{
	archive.Scheduled:     "The scheduled Backup failed.",
	archive.Manual:        "The Backup you started failed.",
	archive.Update:        "The Backup before the update failed.",
	archive.BeforeRestore: "The Backup before the restore failed.",
	archive.Superseded:    "The Backup before the wipe failed.",
}

func (b *Backups) recordFailed(ctx context.Context, trigger archive.Trigger) error {
	return b.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return FailedRow.Record(ctx, tx, b.cfg.History, FailedFacts{Trigger: trigger})
	})
}

// newer is whether this version is above started, both semver. A build
// that isn't semver, such as dev, is never newer.
func newer(this, started string) bool {
	a, okA := parseSemver(this)
	b, okB := parseSemver(started)
	return okA && okB && compareSemver(a, b) > 0
}

type semver struct {
	core [3]int
	pre  string
}

func parseSemver(s string) (semver, bool) {
	s = strings.TrimPrefix(s, "v")
	s, _, _ = strings.Cut(s, "+")
	core, pre, _ := strings.Cut(s, "-")
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return semver{}, false
	}
	var v semver
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return semver{}, false
		}
		v.core[i] = n
	}
	v.pre = pre
	return v, true
}

func compareSemver(a, b semver) int {
	if c := slices.Compare(a.core[:], b.core[:]); c != 0 {
		return c
	}
	switch {
	case a.pre == b.pre:
		return 0
	case a.pre == "":
		return 1
	case b.pre == "":
		return -1
	}
	return strings.Compare(a.pre, b.pre)
}
