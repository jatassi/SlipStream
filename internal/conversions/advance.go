package conversions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/conversions/internal/order"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
)

// move is one transition in flight: the row as read, the row as it will
// be written, and what its effects read once.
type move struct {
	c   *Conversions
	tx  *storage.Tx[storage.Library]
	q   *db.Queries
	was Row
	row Row
	ev  Event
	now time.Time
	// code is the fault the event carries, for every On that runs as a
	// Fault.
	code  Code
	retry RetryPlan
	// failedOn names the Nodes the counted attempts failed on.
	failedOn []string
	src      *files.Located
	conv     *made
}

// made is the Target a Conversion makes a file for, as its effects read it.
type made struct {
	Name string
	targets.Conversion
}

// advance moves Conversion id by ev along the stage table, in tx. An
// event with no transition from the row's stage, or that fails its
// fence, changes nothing. It is the only writer of stage.
func (c *Conversions) advance(ctx context.Context, tx *storage.Tx[storage.Library], id ID, ev Event) error {
	q := db.New(tx)
	raw, err := q.GetConversion(ctx, id)
	if errors.Is(err, storage.ErrNoRows) {
		return ErrNoConversion
	}
	if err != nil {
		return fmt.Errorf("read the Conversion: %w", err)
	}
	was, err := rowOf(raw)
	if err != nil {
		return err
	}
	t, ok := rule(was.Stage, ev.On)
	if !ok || !fenced(t.Fence, was, ev) {
		return nil
	}
	m := &move{c: c, tx: tx, q: q, was: was, row: was, ev: ev, now: c.cfg.Clock(), code: codeOf(ev)}
	next, err := t.To(ctx, m)
	if err != nil || next == "" {
		return err
	}
	m.row.Stage = next
	for _, do := range t.Do {
		if err := do(ctx, m); err != nil {
			return err
		}
	}
	if next == StageDone || next == StageCanceled {
		at := m.now
		m.row.FinishedAt = &at
	}
	return q.WriteConversion(ctx, writeOf(m.row))
}

func fenced(f Fence, r Row, ev Event) bool {
	switch f {
	case FenceNode:
		on := r.Node
		if r.Stage == StageQueued {
			on = r.Pin
		}
		return on != nil && *on == ev.Node
	case FenceAttempt:
		return r.Node != nil && *r.Node == ev.Node && r.Attempt == ev.Attempt
	case FenceBasis:
		return ev.Basis == Basis{Stage: r.Stage, Attempt: r.Attempt}
	case FenceNone:
	}
	return true
}

// codeOf is the fault an event runs as.
func codeOf(ev Event) Code {
	if c, ok := asFault[ev.On]; ok {
		return c
	}
	if ev.On == Checked {
		return ev.Verdict.Code
	}
	return ev.Code
}

// asFault is the code of each event that runs as a run fault of its own.
var asFault = map[On]Code{CopyFailed: CodeCopyFailed, ReturnFailed: CodeReturnFailed}

// source is the Conversion's source file, read once per move.
func (m *move) source(ctx context.Context) (files.Located, error) {
	if m.src == nil {
		if m.row.SourceFile == nil {
			return files.Located{}, errors.New("the Conversion's source file is gone")
		}
		l, err := m.c.cfg.Files.Located(ctx, m.tx, *m.row.SourceFile)
		if err != nil {
			return files.Located{}, fmt.Errorf("read the source file: %w", err)
		}
		m.src = &l
	}
	return *m.src, nil
}

// target is the Target the Conversion makes a file for, read once per
// move.
func (m *move) target(ctx context.Context) (made, error) {
	if m.conv == nil {
		if m.row.Target == nil {
			return made{}, errors.New("the Conversion's Target is gone")
		}
		all, err := targets.All(ctx, m.tx)
		if err != nil {
			return made{}, fmt.Errorf("read the Targets: %w", err)
		}
		i := slices.IndexFunc(all, func(t targets.Target) bool { return t.ID == *m.row.Target })
		if i < 0 || all[i].Conversion == nil {
			return made{}, errors.New("the Conversion's Target makes no files")
		}
		m.conv = &made{Name: all[i].Name, Conversion: *all[i].Conversion}
	}
	return *m.conv, nil
}

func rowOf(d db.ConversionsConversion) (Row, error) {
	r := Row{
		ID: d.ID, Work: d.WorkID, Type: d.Type, Edition: d.Edition, Version: int(d.Version),
		Lane: Lane(d.Lane), Position: order.Key(d.Position), Stage: Stage(d.Stage),
		Attempt: int(d.Attempt), BudgetFrom: int(d.BudgetFrom), NotBefore: d.NotBefore.Time(),
		QueuedAt: d.QueuedAt.Time(), StartedAt: timeOf(d.StartedAt), FinishedAt: timeOf(d.FinishedAt),
		SourceFile: some(d.SourceFileID), Target: some(d.TargetID), Backfill: some(d.BackfillID),
		Node: some(d.NodeID), Root: some(d.RootID), Pin: some(d.PinNodeID), Output: some(d.OutputFileID),
	}
	for _, j := range []struct {
		raw string
		to  any
	}{{d.Items, &r.Items}, {d.Exclude, &r.Exclude}, {d.PausedBy, &r.PausedBy}, {d.Snapshot, &r.Snapshot}} {
		if err := json.Unmarshal([]byte(j.raw), j.to); err != nil {
			return Row{}, fmt.Errorf("read the Conversion %s: %w", d.ID, err)
		}
	}
	if d.FailureCode != nil {
		r.Failure = &Failure{Code: Code(*d.FailureCode), Class: Class(*d.FailureClass), Detail: *d.FailureDetail}
	}
	if d.CancelWhy != nil {
		r.CancelWhy = CancelWhy(*d.CancelWhy)
	}
	return r, nil
}

func writeOf(r Row) db.WriteConversionParams {
	p := db.WriteConversionParams{
		ID: r.ID, Stage: string(r.Stage), Attempt: int64(r.Attempt), BudgetFrom: int64(r.BudgetFrom),
		Lane: string(r.Lane), Position: string(r.Position), Version: int64(r.Version),
		Exclude: jsonOf(nonNil(r.Exclude)), PausedBy: jsonOf(nonNil(r.PausedBy)),
		StartedAt: stored(r.StartedAt), FinishedAt: stored(r.FinishedAt),
		NodeID: value(r.Node), RootID: value(r.Root), PinNodeID: value(r.Pin), OutputFileID: value(r.Output),
	}
	if r.Failure != nil {
		code, class := string(r.Failure.Code), string(r.Failure.Class)
		p.FailureCode, p.FailureClass, p.FailureDetail = &code, &class, &r.Failure.Detail
	}
	if r.CancelWhy != "" {
		why := string(r.CancelWhy)
		p.CancelWhy = &why
	}
	return p
}

func some[T interface{ IsZero() bool }](v T) *T {
	if v.IsZero() {
		return nil
	}
	return &v
}

func value[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

func timeOf(t storage.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	at := t.Time()
	return &at
}

func stored(t *time.Time) storage.Time {
	if t == nil {
		return storage.Time{}
	}
	return storage.FromTime(*t)
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func jsonOf(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("conversions: encode %T: %v", v, err))
	}
	return string(b)
}

func nodeIn(ns []role.NodeID, n role.NodeID) bool { return slices.Contains(ns, n) }
