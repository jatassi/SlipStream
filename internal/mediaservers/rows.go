package mediaservers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/mediaservers/queue"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

// changeOf is a mediaservers_change row as the queue models it.
func changeOf(r db.MediaserversChange) (queue.Change, error) {
	c := queue.Change{
		ID: r.ID, Server: r.ServerID, Kind: r.Kind, Root: r.RootID, File: r.FileID, ReplacedBy: r.ReplacedBy,
		Upgrade: r.Upgrade, State: r.State, Step: r.Step, StepAt: r.StepAt.Time(), Attempts: int(r.Attempts),
		NextAt: r.NextAttemptAt.Time(), Created: r.CreatedAt.Time(), Sent: r.SentAt.Time(),
	}
	var err error
	if r.Path != "" {
		if c.Path, err = storing.ParseRel(r.Path); err != nil {
			return c, fmt.Errorf("read change %s: %w", r.ID, err)
		}
	}
	if r.FromPath != nil {
		if c.From, err = storing.ParseRel(*r.FromPath); err != nil {
			return c, fmt.Errorf("read change %s: %w", r.ID, err)
		}
	}
	if r.Result != nil {
		c.Result = *r.Result
	}
	if err := json.Unmarshal([]byte(r.Lost), &c.Lost); err != nil {
		return c, fmt.Errorf("read change %s: %w", r.ID, err)
	}
	if r.Snapshot != nil {
		c.Snapshot = &link.Snapshot{}
		if err := json.Unmarshal([]byte(*r.Snapshot), c.Snapshot); err != nil {
			return c, fmt.Errorf("read change %s: %w", r.ID, err)
		}
	}
	if r.Problem != nil {
		c.Problem = &link.Problem{}
		if err := json.Unmarshal([]byte(*r.Problem), c.Problem); err != nil {
			return c, fmt.Errorf("read change %s: %w", r.ID, err)
		}
	}
	return c, nil
}

func changesOf(rs []db.MediaserversChange) ([]queue.Change, error) {
	out := make([]queue.Change, len(rs))
	for i, r := range rs {
		var err error
		if out[i], err = changeOf(r); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// columns is c as the columns a fold or a step writes.
type columns struct {
	path     string
	from     *string
	snapshot *string
	result   *link.Result
	lost     string
	problem  *string
}

func columnsOf(c queue.Change) (columns, error) {
	col := columns{path: c.Path.String(), lost: "[]"}
	if c.Carry() {
		from := c.From.String()
		col.from = &from
	}
	if c.Result != "" {
		col.result = &c.Result
	}
	if len(c.Lost) > 0 {
		b, err := json.Marshal(c.Lost)
		if err != nil {
			return col, fmt.Errorf("write the losses of change %s: %w", c.ID, err)
		}
		col.lost = string(b)
	}
	var err error
	if col.snapshot, err = jsonOrNil(c.Snapshot); err != nil {
		return col, err
	}
	col.problem, err = jsonOrNil(c.Problem)
	return col, err
}

func jsonOrNil[T any](v *T) (*string, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("write a change's %T: %w", v, err)
	}
	s := string(b)
	return &s, nil
}

// apply writes ops, in their order.
func apply(ctx context.Context, q *db.Queries, ops []queue.Op) error {
	for _, op := range ops {
		if err := write(ctx, q, op); err != nil {
			return err
		}
	}
	return nil
}

func write(ctx context.Context, q *db.Queries, op queue.Op) error {
	c := op.Change
	if op.Do == queue.Delete {
		return q.DeleteChange(ctx, c.ID)
	}
	col, err := columnsOf(c)
	if err != nil {
		return err
	}
	if op.Do == queue.Insert {
		return q.InsertChange(ctx, db.InsertChangeParams{
			ID: c.ID, ServerID: c.Server, Kind: c.Kind, RootID: c.Root, Path: col.path, FromPath: col.from,
			FileID: c.File, ReplacedBy: c.ReplacedBy, Upgrade: c.Upgrade, State: c.State, Step: c.Step,
			StepAt: storage.FromTime(c.StepAt), Snapshot: col.snapshot, Attempts: int64(c.Attempts),
			NextAttemptAt: storage.FromTime(c.NextAt), CreatedAt: storage.FromTime(c.Created),
			SentAt: storage.FromTime(c.Sent), Result: col.result, Lost: col.lost, Problem: col.problem,
		})
	}
	return q.UpdateChange(ctx, db.UpdateChangeParams{
		ID: c.ID, Kind: c.Kind, Path: col.path, FromPath: col.from, FileID: c.File, ReplacedBy: c.ReplacedBy,
		Upgrade: c.Upgrade, State: c.State, Step: c.Step, StepAt: storage.FromTime(c.StepAt), Snapshot: col.snapshot,
		Attempts: int64(c.Attempts), NextAttemptAt: storage.FromTime(c.NextAt), SentAt: storage.FromTime(c.Sent),
		Result: col.result, Lost: col.lost, Problem: col.problem,
	})
}

// openFor reads every foldable row of in's server that in can fold into,
// as queue.Open names them.
func openFor(ctx context.Context, q *db.Queries, in queue.Change) (queue.Open, error) {
	open := queue.Open{AtPath: map[storing.Rel]queue.Change{}}
	carries, err := q.OpenCarry(ctx, db.OpenCarryParams{ServerID: in.Server, FileID: in.File})
	if err != nil {
		return open, err
	}
	paths := in.Paths()
	if len(carries) > 0 {
		c, err := changeOf(carries[0])
		if err != nil {
			return open, err
		}
		open.Carry, paths = &c, append(paths, c.From)
	}
	at := make([]string, len(paths))
	for i, p := range paths {
		at[i] = p.String()
	}
	rows, err := q.OpenAtPaths(ctx, db.OpenAtPathsParams{ServerID: in.Server, RootID: in.Root, Paths: at})
	if err != nil {
		return open, err
	}
	for _, r := range rows {
		c, err := changeOf(r)
		if err != nil {
			return open, err
		}
		open.AtPath[c.Path] = c
	}
	p := db.OpenPartnerParams{ServerID: in.Server}
	if in.Kind == link.ChangeRemoved {
		p.Replacer = in.ReplacedBy
	}
	if in.Kind == link.ChangeAdded {
		p.Added = in.File
	}
	if p.Replacer.IsZero() && p.Added.IsZero() {
		return open, nil
	}
	partners, err := q.OpenPartner(ctx, p)
	if err != nil || len(partners) == 0 {
		return open, err
	}
	c, err := changeOf(partners[0])
	open.Partner = &c
	return open, err
}

// reopen returns back, a row taken to send, to due through queue.Reopen,
// folding into it the row that took its place meanwhile.
func reopen(ctx context.Context, q *db.Queries, back queue.Change) error {
	var newer *queue.Change
	if !back.Step.Started() {
		open, err := openFor(ctx, q, back)
		if err != nil {
			return err
		}
		newer = takenPlace(open, back)
	}
	return apply(ctx, q, queue.Reopen(back, newer))
}

// takenPlace is the row in open that took back's place while it was sent:
// the foldable carry of its file for a carry, else the foldable row at its
// path.
func takenPlace(open queue.Open, back queue.Change) *queue.Change {
	if back.Carry() {
		return open.Carry
	}
	if c, ok := open.AtPath[back.Path]; ok {
		return &c
	}
	return nil
}

// reopenRows returns each of rows to due.
func reopenRows(ctx context.Context, q *db.Queries, rows []db.MediaserversChange) error {
	cs, err := changesOf(rows)
	if err != nil {
		return err
	}
	for _, c := range cs {
		if err := reopen(ctx, q, c); err != nil {
			return err
		}
	}
	return nil
}
