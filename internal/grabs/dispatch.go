package grabs

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// sendIndexers is the most Indexers one dispatch Write sends lines to,
// past any library's count of Indexers.
const sendIndexers = 50

// fence refuses with ErrReconciling while a restored Controller is
// Reconciling: nothing searches or grabs then (ADR 0019).
func (gr *Grabs) fence(ctx context.Context, r storage.Reader[file]) error {
	on, err := gr.cfg.Reconciling(ctx, r)
	if err != nil {
		return err
	}
	if on {
		return ErrReconciling
	}
	return nil
}

// pass is one dispatch Write: what it read once, what it changed, and the
// earliest time a line that waits may go.
type pass struct {
	tx      *storage.Tx[file]
	q       *db.Queries
	now     storage.Time
	wait    time.Duration
	types   map[works.WorkID]works.TypeID
	lines   map[works.TypeID][]indexers.Line
	ixs     map[indexers.ID]indexers.Indexer
	waiting map[indexers.ID]int
	changed map[works.WorkID]bool
	// runs is the runs the pass started or refused, by Work.
	runs map[SearchID]works.WorkID
	wake time.Time
}

// dispatch is one fenced Write. It sends each Indexer's planned lines
// while its room takes them, starts queued rows while every Indexer a
// row's run would ask has room in its line, then sends the new runs'
// lines. While Reconciling it does nothing.
func (gr *Grabs) dispatch(ctx context.Context) (time.Time, error) {
	var p *pass
	err := gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		if err := gr.fence(ctx, tx); err != nil {
			return err
		}
		wait, err := indexers.SearchWaitIn(ctx, tx)
		if err != nil {
			return err
		}
		p = &pass{
			tx: tx, q: db.New(tx), now: storage.Now(), wait: wait, types: map[works.WorkID]works.TypeID{},
			lines: map[works.TypeID][]indexers.Line{}, ixs: map[indexers.ID]indexers.Indexer{}, changed: map[works.WorkID]bool{},
			runs: map[SearchID]works.WorkID{},
		}
		if err := gr.send(ctx, p); err != nil {
			return err
		}
		started, err := gr.startRuns(ctx, p)
		if err != nil || started == 0 {
			return err
		}
		return gr.send(ctx, p)
	})
	if errors.Is(err, ErrReconciling) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	for run, work := range p.runs {
		gr.cfg.SearchChanged.Publish(ctx, SearchChanged{SearchID: run.String(), WorkID: work.String()})
	}
	gr.queueChanged(ctx, p.changed)
	return p.wake, nil
}

func (gr *Grabs) queueChanged(ctx context.Context, changed map[works.WorkID]bool) {
	if len(changed) == 0 {
		return
	}
	ids := make([]string, 0, len(changed))
	for w := range changed {
		ids = append(ids, w.String())
	}
	gr.cfg.QueueChanged.Publish(ctx, QueueChanged{WorkIDs: ids})
}

// send dispatches the planned lines of each Indexer that has some.
func (gr *Grabs) send(ctx context.Context, p *pass) error {
	ids, err := p.q.ListPlannedIndexers(ctx, sendIndexers)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := gr.sendOn(ctx, p, id); err != nil {
			return err
		}
	}
	return nil
}

// sendOn walks Indexer id's FIFO: each line Reserve takes is sent with its
// durable deadline, each it skips settles with its outcome, and the first
// that waits holds the lines behind it. A line of an Indexer deleted,
// turned off or with Automatic search off leaves its run, as planning
// would leave it out now.
func (gr *Grabs) sendOn(ctx context.Context, p *pass, id indexers.ID) error {
	idx, err := indexers.Get(ctx, p.tx, id)
	if err != nil && !errors.Is(err, indexers.ErrNoIndexer) {
		return err
	}
	off := err != nil || !idx.On || !idx.AutoSearch
	lines, err := p.q.ListPendingLines(ctx, id)
	if err != nil {
		return err
	}
	for _, l := range lines {
		if off {
			if err := p.q.DeletePlannedLine(ctx, db.DeletePlannedLineParams{SearchID: l.SearchID, Position: l.Position}); err != nil {
				return err
			}
			p.changed[l.WorkID] = true
			continue
		}
		held, err := gr.sendLine(ctx, p, id, l)
		if err != nil || held {
			return err
		}
	}
	return nil
}

// sendLine reserves Indexer id's room for l and sends it, or settles it
// with the outcome Reserve skips it with. It reports whether l waits for
// room, holding the lines behind it.
func (gr *Grabs) sendLine(ctx context.Context, p *pass, id indexers.ID, l db.ListPendingLinesRow) (bool, error) {
	typ, err := p.typeOf(ctx, l.WorkID)
	if err != nil {
		return false, err
	}
	query, err := gr.cfg.Indexers.Reserve(ctx, p.tx, id, typ, claimOf(laneOf(l.Lane)))
	var skip *indexers.WaitError
	var room *indexers.RoomError
	switch {
	case err == nil:
		_, err = p.q.DispatchLine(ctx, db.DispatchLineParams{
			QueryID: query, DispatchedAt: p.now, DeadlineAt: storage.FromTime(p.now.Time().Add(p.wait)),
			SearchID: l.SearchID, Position: l.Position,
		})
	case errors.As(err, &skip):
		a := outcomeRow(skip.Outcome)
		a.SearchID, a.Position = l.SearchID, l.Position
		_, err = p.q.AnswerSearchIndexer(ctx, a)
	case errors.As(err, &room):
		if !room.Until.IsZero() && (p.wake.IsZero() || room.Until.Before(p.wake)) {
			p.wake = room.Until
		}
		return true, nil
	}
	if err != nil {
		return false, err
	}
	p.changed[l.WorkID] = true
	return false, nil
}

// claimOf is whose claim a line of lane is.
func claimOf(l pace.Lane) indexers.Claim {
	if l.Press() {
		return indexers.ClaimPress
	}
	return indexers.ClaimBackground
}

// startRuns starts queued rows lane by lane, at most StartsPerPass, each
// once every Indexer its run would ask has room in its line. It reads at
// most that many rows of each lane, so a row whose Indexers have no room
// leaves the rows behind it for a later pass.
func (gr *Grabs) startRuns(ctx context.Context, p *pass) (int, error) {
	counts, err := p.q.CountPlanned(ctx)
	if err != nil {
		return 0, err
	}
	p.waiting = make(map[indexers.ID]int, len(counts))
	for _, c := range counts {
		p.waiting[c.IndexerID] = int(c.Planned)
	}
	started, most := 0, gr.cfg.Timings.StartsPerPass
	for _, lane := range pace.Lanes {
		rows, err := p.q.ListQueued(ctx, db.ListQueuedParams{Lane: new(int64(lane)), N: int64(most - started)})
		if err != nil {
			return 0, err
		}
		for _, row := range rows {
			ok, err := gr.tryStart(ctx, p, row, lane)
			if err != nil {
				return 0, err
			}
			if ok {
				started++
			}
		}
		if started == most {
			break
		}
	}
	return started, nil
}

// tryStart starts row, queued in lane, when its run may start now.
func (gr *Grabs) tryStart(ctx context.Context, p *pass, row db.GrabsDue, lane pace.Lane) (bool, error) {
	typ, err := p.typeOf(ctx, row.WorkID)
	if err != nil {
		return false, err
	}
	lines, err := p.linesOf(ctx, gr.cfg.Indexers, typ)
	if err != nil {
		return false, err
	}
	if ok, err := p.starts(ctx, lines, lane.Press()); err != nil || !ok {
		return false, err
	}
	return true, gr.startRun(ctx, p, row, lines)
}

// starts reports whether a run asking lines may start: every Indexer it
// asks has fewer waiting lines than Searches at once, one more for a
// press.
func (p *pass) starts(ctx context.Context, lines []indexers.Line, press bool) (bool, error) {
	for _, l := range lines {
		if l.Skipped != nil {
			continue
		}
		idx, ok := p.ixs[l.Indexer]
		if !ok {
			var err error
			if idx, err = indexers.Get(ctx, p.tx, l.Indexer); err != nil {
				return false, err
			}
			p.ixs[l.Indexer] = idx
		}
		if !(pace.Room{AtOnce: idx.SearchesAtOnce, Waiting: p.waiting[l.Indexer]}).Starts(press) {
			return false, nil
		}
	}
	return true, nil
}

func (p *pass) typeOf(ctx context.Context, work works.WorkID) (works.TypeID, error) {
	if t, ok := p.types[work]; ok {
		return t, nil
	}
	s, err := works.SummaryOf(ctx, p.tx, work)
	if err != nil {
		return "", err
	}
	p.types[work] = s.ContentType
	return s.ContentType, nil
}

// linesOf is the lines a run of typ plans now, read once a pass.
func (p *pass) linesOf(ctx context.Context, x *indexers.Indexers, typ works.TypeID) ([]indexers.Line, error) {
	if l, ok := p.lines[typ]; ok {
		return l, nil
	}
	l, err := x.Lines(ctx, p.tx, typ, indexers.AutoSearch)
	if err != nil {
		return nil, err
	}
	p.lines[typ] = l
	return l, nil
}
