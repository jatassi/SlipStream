package moves

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/moves/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// View is the one projection of a unit for every reader.
type View struct {
	Unit  UnitID
	Batch BatchID
	Work  works.WorkID
	From  End
	To    End
	Phase Phase
	// Waiting is reconciling while a restored Controller is Reconciling,
	// node_offline for a queued unit whose Node is away, else stored.
	Waiting Wait
	Since   *time.Time
	Bytes   int64
	Failure *FailureView
	// SwitchNow is offered: a playing or media_servers wait.
	SwitchNow bool
	// Undone is a failed or canceled unit whose undo finished.
	Undone bool
	Notes  Notes
}

type FailureView struct {
	Code  FailureCode
	Words string
}

func (m *Moves) viewOf(u Unit, blocked bool) View {
	v := View{Unit: u.ID, Batch: u.Batch, Work: u.Subject.Work, From: u.Subject.From, To: u.Subject.To, Phase: u.Stage.Phase(), Bytes: u.Planned.Bytes, Notes: u.Notes}
	wait, since := waitOf(u.Stage)
	if wait != "" {
		v.Waiting, v.Since = wait, &since
	}
	switch s := u.Stage.(type) {
	case Queued:
		if !m.cfg.Roster.InStep(u.Subject.From.Node) || !m.cfg.Roster.InStep(u.Subject.To.Node) {
			v.Waiting = WaitNodeOffline
		}
	case Switching:
		v.SwitchNow = s.SwitchNow == nil && (s.Wait == WaitPlaying || s.Wait == WaitMediaServers)
	case Failed:
		v.Failure = &FailureView{Code: s.Code, Words: failureWords(s.Code)}
		v.Undone = s.Undone != nil
	case Canceled:
		v.Undone = s.Undone != nil
	}
	if blocked && u.open() {
		v.Waiting = WaitReconciling
	}
	return v
}

func waitOf(st Stage) (Wait, time.Time) {
	switch s := st.(type) {
	case Queued:
		return s.Wait, s.Since
	case Copying:
		return s.Wait, s.Since
	case Placing:
		return s.Wait, s.Since
	case Switching:
		return s.Wait, s.Since
	case Cleaning:
		return s.Wait, s.Since
	}
	return "", time.Time{}
}

// BatchView is one batch and how many of its units are in each phase.
type BatchView struct {
	ID         BatchID
	Origin     Origin
	CreatedAt  time.Time
	FinishedAt *time.Time
	CanceledAt *time.Time
	Counts     map[Phase]int
}

func notFound(err error) error {
	if errors.Is(err, storage.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// List is one page of batches, open ones first, newest first, after the
// batch after names, zero for the first page.
func (m *Moves) List(ctx context.Context, after BatchID, limit int) ([]BatchView, error) {
	var out []BatchView
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		p, err := pageAfter(ctx, q, after, limit)
		if err != nil {
			return err
		}
		rows, err := q.ListBatches(ctx, p)
		if err != nil {
			return err
		}
		ids := make([]BatchID, len(rows))
		for i, row := range rows {
			ids[i] = row.ID
		}
		counts, err := q.CountBatchesPhases(ctx, ids)
		if err != nil {
			return err
		}
		out = make([]BatchView, len(rows))
		for i, row := range rows {
			out[i] = batchViewOf(row)
			for _, c := range counts {
				if c.BatchID == row.ID {
					out[i].Counts[Phase(c.Phase)] = int(c.N)
				}
			}
		}
		return nil
	})
	return out, err
}

// pageAfter is the page of limit batches after batch after, zero for the
// first page.
func pageAfter(ctx context.Context, q *db.Queries, after BatchID, limit int) (db.ListBatchesParams, error) {
	if after.IsZero() {
		return db.ListBatchesParams{First: 1, N: int64(limit)}, nil
	}
	b, err := q.GetBatch(ctx, after)
	if err != nil {
		return db.ListBatchesParams{}, notFound(err)
	}
	p := db.ListBatchesParams{AfterCreatedAt: b.CreatedAt, AfterID: b.ID, N: int64(limit)}
	if !b.FinishedAt.IsZero() {
		p.AfterDone = 1
	}
	return p, nil
}

func batchViewOf(row db.MovesBatch) BatchView {
	return BatchView{
		ID: row.ID, Origin: Origin(row.Origin), CreatedAt: row.CreatedAt.Time(), FinishedAt: timeOf(row.FinishedAt),
		CanceledAt: timeOf(row.CanceledAt), Counts: map[Phase]int{},
	}
}

// Get is batch id and one page of its units after position after, -1 for
// the first page.
func (m *Moves) Get(ctx context.Context, id BatchID, after, limit int) (BatchView, []View, error) {
	var b BatchView
	var out []View
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		row, err := q.GetBatch(ctx, id)
		if err != nil {
			return notFound(err)
		}
		b = batchViewOf(row)
		counts, err := q.CountBatchPhases(ctx, id)
		if err != nil {
			return err
		}
		for _, c := range counts {
			b.Counts[Phase(c.Phase)] = int(c.N)
		}
		blocked, err := m.cfg.Reconciling(ctx, r)
		if err != nil {
			return err
		}
		rows, err := q.ListUnitsOfBatch(ctx, db.ListUnitsOfBatchParams{BatchID: id, After: int64(after), N: int64(limit)})
		if err != nil {
			return err
		}
		us, err := unitsOf(ctx, q, rows)
		for _, u := range us {
			out = append(out, m.viewOf(u, blocked))
		}
		return err
	})
	return b, out, err
}

// Unit is unit id's view.
func (m *Moves) Unit(ctx context.Context, id UnitID) (View, error) {
	var v View
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		u, err := load(ctx, r, id)
		if err != nil {
			return err
		}
		blocked, err := m.cfg.Reconciling(ctx, r)
		v = m.viewOf(u, blocked)
		return err
	})
	return v, err
}

// Of is each of ws' latest unit, read in r: the Work view's Move.
func (m *Moves) Of(ctx context.Context, r storage.Reader[file], ws []works.WorkID) (map[works.WorkID]View, error) {
	if len(ws) == 0 {
		return nil, nil
	}
	q := db.New(r)
	rows, err := q.ListLatestUnitsOfWorks(ctx, ws)
	if err != nil {
		return nil, err
	}
	us, err := unitsOf(ctx, q, rows)
	if err != nil {
		return nil, err
	}
	blocked, err := m.cfg.Reconciling(ctx, r)
	out := make(map[works.WorkID]View, len(us))
	for _, u := range us {
		out[u.Subject.Work] = m.viewOf(u, blocked)
	}
	return out, err
}

// Need is a unit Home's Needs you lists: failed, or waiting at its switch
// longer than "Show a waiting switch in Needs you after".
type Need = View

// NeedsYou is at most limit units that need the Admin, newest first,
// read in r.
func (m *Moves) NeedsYou(ctx context.Context, r storage.Reader[file], now time.Time, limit int) ([]Need, error) {
	set, err := settingsIn(ctx, r)
	if err != nil {
		return nil, err
	}
	q := db.New(r)
	rows, err := q.ListNeeds(ctx, db.ListNeedsParams{Before: storage.FromTime(now.Add(-set.NeedsYouAfter)), N: int64(limit)})
	if err != nil {
		return nil, err
	}
	us, err := unitsOf(ctx, q, rows)
	if err != nil {
		return nil, err
	}
	out := make([]Need, len(us))
	for i, u := range us {
		out[i] = m.viewOf(u, false)
	}
	return out, nil
}
