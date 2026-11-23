package moves

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/moves/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// View is the one projection of a unit for every reader.
type View struct {
	Unit  UnitID
	Batch BatchID
	// Work is a Work unit's, Torrent a torrent unit's.
	Work        works.WorkID
	Torrent     *downloadclients.ID
	Label       string
	ContentType works.TypeID
	From        Place
	To          Place
	Phase       Phase
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
	// Words is the band's sentence: what the Move does now.
	Words string
}

type FailureView struct {
	Code  FailureCode
	Words string
}

func (m *Moves) viewOf(u Unit, blocked bool, n Named) View {
	w := n.works[u.Subject.Work]
	v := View{
		Unit: u.ID, Batch: u.Batch, Work: u.Subject.Work, Label: w.Label, ContentType: u.Subject.Type,
		From: n.Place(u.Subject.From), To: n.Place(u.Subject.To), Phase: u.Stage.Phase(), Bytes: u.Planned.Bytes, Notes: u.Notes,
	}
	if t := u.Torrent; t != nil {
		src := n.torrents[t.Torrent]
		v.Work, v.Torrent, v.Label, v.ContentType = works.WorkID{}, &t.Torrent, src.Name, ""
		if src.ContentType != nil {
			v.ContentType = *src.ContentType
		}
		v.From = Place{End: End{Node: t.From.Node}, NodeName: n.nodes[t.From.Node], Client: n.sides.client(t.From)}
		v.To = Place{End: End{Node: t.To.Node}, NodeName: n.nodes[t.To.Node], Client: n.sides.client(t.To)}
	}
	wait, since := waitOf(u.Stage)
	if wait != "" {
		v.Waiting, v.Since = wait, &since
	}
	switch s := u.Stage.(type) {
	case Queued:
		if from, to := u.nodes(); !m.cfg.Roster.InStep(from) || !m.cfg.Roster.InStep(to) {
			v.Waiting = WaitNodeOffline
		}
	case Switching:
		v.SwitchNow = s.SwitchNow == nil && (s.Wait == WaitPlaying || s.Wait == WaitMediaServers)
	case Failed:
		v.Failure = &FailureView{Code: s.Code, Words: failureWords(s.Code)}
		if t := u.Torrent; t != nil {
			v.Failure.Words = m.torrentFailureWords(s, v.From.NodeName, v.To.NodeName, n.sides.words(t.To))
		}
		v.Undone = s.Undone != nil
	case Canceled:
		v.Undone = s.Undone != nil
	}
	if blocked && u.open() {
		v.Waiting = WaitReconciling
	}
	v.Words = wordsOf(v)
	return v
}

// views is us' views, read in r.
func (m *Moves) views(ctx context.Context, r storage.Reader[file], us []Unit) ([]View, error) {
	blocked, err := m.cfg.Reconciling(ctx, r)
	if err != nil {
		return nil, err
	}
	var ws []works.WorkID
	var ts []downloadclients.ID
	for _, u := range us {
		if u.Torrent != nil {
			ts = append(ts, u.Torrent.Torrent)
		} else {
			ws = append(ws, u.Subject.Work)
		}
	}
	n, err := m.namesOf(ctx, r, ws)
	if err != nil {
		return nil, err
	}
	if len(ts) > 0 {
		if n.torrents, err = downloadclients.MoveSources(ctx, r, ts); err != nil {
			return nil, err
		}
		if n.sides, err = sidesIn(ctx, r, us); err != nil {
			return nil, err
		}
	}
	out := make([]View, len(us))
	for i, u := range us {
		out[i] = m.viewOf(u, blocked, n)
	}
	return out, nil
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

// BatchView is one batch, how many of its units are in each phase, its
// sentence and what the Admin can do to it.
type BatchView struct {
	ID         BatchID
	Origin     Origin
	CreatedAt  time.Time
	FinishedAt *time.Time
	CanceledAt *time.Time
	Counts     map[Phase]int
	// torrents is how many of Counts are torrent units.
	torrents map[Phase]int
	Total    int
	// Words is its sentence, such as "12 of 43 movies switched".
	Words                        string
	CanRetry, CanCancel, CanBack bool
}

// count adds n units of kind in phase to b.
func (b *BatchView) count(kind, phase string, n int64) {
	b.Counts[Phase(phase)] += int(n)
	if kind == "torrent" {
		b.torrents[Phase(phase)] += int(n)
	}
}

// finish fills b's total, sentence and actions from its counts, its Work
// units of types.
func (m *Moves) finish(b *BatchView, types []works.TypeID) {
	c, t := b.Counts, b.torrents
	var torrents int
	for p, n := range c {
		b.Total += n
		torrents += t[p]
	}
	all := kinds{works: b.Total - torrents, torrents: torrents}
	of := func(phases ...Phase) kinds {
		var k kinds
		for _, p := range phases {
			k.works += c[p] - t[p]
			k.torrents += t[p]
		}
		return k
	}
	open := c[PhaseQueued] + c[PhaseTransfer] + c[PhasePlace] + c[PhaseSwitch]
	b.CanRetry, b.CanCancel, b.CanBack = c[PhaseFailed] > 0, open > 0, b.FinishedAt != nil && c[PhaseDone] > 0
	if open+c[PhaseCleanup] > 0 {
		b.Words = m.tally(types, of(PhaseCleanup, PhaseDone), all) + " switched"
		return
	}
	b.Words = m.tally(types, of(PhaseDone), all) + " moved."
	if n := c[PhaseFailed]; n > 0 {
		b.Words += fmt.Sprintf(" %d failed.", n)
	}
	if n := c[PhaseCanceled]; n > 0 {
		b.Words += fmt.Sprintf(" %d canceled.", n)
	}
}

// typesOf is each of batches' units' Content Types, read in q.
func typesOf(ctx context.Context, q *db.Queries, batches []BatchID) (map[BatchID][]works.TypeID, error) {
	rows, err := q.ListBatchTypes(ctx, batches)
	out := map[BatchID][]works.TypeID{}
	for _, row := range rows {
		out[row.BatchID] = append(out[row.BatchID], *row.ContentType)
	}
	return out, err
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
		types, err := typesOf(ctx, q, ids)
		if err != nil {
			return err
		}
		out = make([]BatchView, len(rows))
		for i, row := range rows {
			out[i] = batchViewOf(row)
			for _, c := range counts {
				if c.BatchID == row.ID {
					out[i].count(c.Kind, c.Phase, c.N)
				}
			}
			m.finish(&out[i], types[row.ID])
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
		CanceledAt: timeOf(row.CanceledAt), Counts: map[Phase]int{}, torrents: map[Phase]int{},
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
			b.count(c.Kind, c.Phase, c.N)
		}
		types, err := typesOf(ctx, q, []BatchID{id})
		if err != nil {
			return err
		}
		m.finish(&b, types[id])
		rows, err := q.ListUnitsOfBatch(ctx, db.ListUnitsOfBatchParams{BatchID: id, After: int64(after), N: int64(limit)})
		if err != nil {
			return err
		}
		us, err := unitsOf(ctx, q, rows)
		if err != nil {
			return err
		}
		out, err = m.views(ctx, r, us)
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
		vs, err := m.views(ctx, r, []Unit{u})
		if err == nil {
			v = vs[0]
		}
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
	vs, err := m.views(ctx, r, us)
	out := make(map[works.WorkID]View, len(vs))
	for _, v := range vs {
		out[v.Work] = v
	}
	return out, err
}

// Need is a unit Home's Needs you lists: failed, or waiting at its switch
// longer than "Show a Move waiting to switch in Needs you after". Words is its
// sentence.
type Need struct {
	View
	Words string
}

// NeedsYou is at most limit units that need the Admin, newest first,
// read in r.
func (m *Moves) NeedsYou(ctx context.Context, r storage.Reader[file], limit int) ([]Need, error) {
	now := m.now()
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
	vs, err := m.views(ctx, r, us)
	out := make([]Need, len(vs))
	for i, v := range vs {
		out[i] = Need{View: v, Words: needWords(v, now)}
	}
	return out, err
}
