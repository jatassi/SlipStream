package moves

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/moves/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/transfers/source"
	"github.com/jatassi/slipstream/internal/works"
)

// Run drives every unit until ctx ends. It first marks every open unit
// due once. A pass takes the heads of the free lanes and one page of due
// units, takes one Action per unit with next, applies every Action that
// needs no IO in one Write, and starts each IO step in its own goroutine,
// at most one per unit and Timings.Jobs at once. Each IO step ends in a
// Write that re-reads its unit and records the result, so a crash
// anywhere repeats at most one step.
func (m *Moves) Run(ctx context.Context) error {
	m.Wake(ctx)
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).MarkOpenDue(ctx, storage.FromTime(m.now()))
	})
	if err != nil {
		return err
	}
	for {
		began := m.now()
		wait, err := m.pass(ctx)
		if err != nil && ctx.Err() == nil {
			m.cfg.Log.Error("a pass over the Moves failed", "err", err.Error())
			wait = m.cfg.Timings.RetryAfter
		}
		if !m.sleep(ctx, wait) {
			return nil
		}
		if rest := m.cfg.Timings.PassEvery - m.now().Sub(began); rest > 0 && !m.sleep(ctx, rest) {
			return nil
		}
	}
}

// sleep waits for d, a poke or ctx, d < 0 meaning only a poke. It is
// false once ctx ends.
func (m *Moves) sleep(ctx context.Context, d time.Duration) bool {
	var timer <-chan time.Time
	switch {
	case d == 0:
		return ctx.Err() == nil
	case d > 0:
		t := time.NewTimer(d)
		defer t.Stop()
		timer = t.C
	}
	select {
	case <-ctx.Done():
		return false
	case <-m.poke:
	case <-timer:
	}
	return true
}

// idle is a pass's answer when no unit is due: only a poke wakes it.
const idle = time.Duration(-1)

// work is one unit a pass judged, and what it does now.
type work struct {
	u      Unit
	facts  Facts
	action Action
}

func (m *Moves) pass(ctx context.Context) (time.Duration, error) {
	now := m.now()
	var blocked bool
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		blocked, err = m.cfg.Reconciling(ctx, r)
		return err
	})
	if err != nil || blocked {
		return idle, err
	}
	if fired, reconciled := m.wakes(); len(fired) > 0 || reconciled {
		err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
			return applyWakes(ctx, tx, fired, reconciled, storage.FromTime(now))
		})
		if err != nil {
			return 0, err
		}
	}
	var units []Unit
	var facts []Facts
	var set Settings
	err = m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var heads map[UnitID]bool
		var err error
		if units, heads, err = m.dueUnits(ctx, r, now); err != nil {
			return err
		}
		if facts, err = m.factsOf(ctx, r, units, heads); err != nil {
			return err
		}
		set, err = settingsIn(ctx, r)
		return err
	})
	if err != nil {
		return 0, err
	}
	if err := m.copyFacts(ctx, units, facts); err != nil {
		return 0, err
	}
	tm := m.cfg.Timings
	rules := Rules{
		PlexBound: set.PlexBound, CheckBound: set.CheckBound, PlayingEvery: tm.PlayingEvery,
		Guard: tm.Guard, ReplyWait: tm.ReplyWait, StopBound: tm.StopBound, HandoffBound: tm.HandoffBound, DownloadingEvery: tm.DownloadingEvery,
	}
	judged := make([]work, len(units))
	for i, u := range units {
		judged[i] = work{u: u, facts: facts[i], action: next(u, facts[i], now, rules)}
	}
	steps, err := m.apply(ctx, judged, now)
	if err != nil {
		return 0, err
	}
	for _, w := range steps {
		m.start(ctx, w)
	}
	pending, err := m.deleteEmptied(ctx)
	if err != nil {
		return 0, err
	}
	wait, err := m.untilDue(ctx)
	if pending && (wait == idle || wait > tm.PlayingEvery) {
		wait = tm.PlayingEvery
	}
	return wait, err
}

// dueUnits is the head of each free lane and one page of due units, none
// an IO step runs for, with their files.
func (m *Moves) dueUnits(ctx context.Context, r storage.Reader[file], now time.Time) ([]Unit, map[UnitID]bool, error) {
	q := db.New(r)
	held, err := q.ListHeldLanes(ctx)
	if err != nil {
		return nil, nil, err
	}
	var rows []db.MovesUnit
	heads := map[UnitID]bool{}
	for after := ""; ; {
		row, err := q.NextHead(ctx, after)
		if errors.Is(err, storage.ErrNoRows) {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		after = row.Lane
		if !slices.Contains(held, row.Lane) {
			rows = append(rows, row)
			heads[row.ID] = true
		}
	}
	due, err := q.ListDue(ctx, db.ListDueParams{Now: storage.FromTime(now), N: int64(m.cfg.Timings.PassPage)})
	if err != nil {
		return nil, nil, err
	}
	for _, row := range due {
		if !heads[row.ID] {
			rows = append(rows, row)
		}
	}
	m.mu.Lock()
	rows = slices.DeleteFunc(rows, func(row db.MovesUnit) bool { return m.running[row.ID] })
	m.mu.Unlock()
	units, err := unitsOf(ctx, q, rows)
	return units, heads, err
}

// unitsOf parses rows, each with its files.
func unitsOf(ctx context.Context, q *db.Queries, rows []db.MovesUnit) ([]Unit, error) {
	ids := make([]UnitID, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	fs, err := q.ListUnitFiles(ctx, ids)
	if err != nil {
		return nil, err
	}
	of := map[UnitID][]File{}
	for _, row := range fs {
		f, err := fileOf(row)
		if err != nil {
			return nil, err
		}
		of[row.UnitID] = append(of[row.UnitID], f)
	}
	out := make([]Unit, len(rows))
	for i, row := range rows {
		if out[i], err = unitOf(row); err != nil {
			return nil, err
		}
		out[i].Files = of[row.ID]
	}
	return out, nil
}

// load reads unit id with its files in r.
func load(ctx context.Context, r storage.Reader[file], id UnitID) (Unit, error) {
	q := db.New(r)
	row, err := q.GetUnit(ctx, id)
	if errors.Is(err, storage.ErrNoRows) {
		return Unit{}, ErrNotFound
	}
	if err != nil {
		return Unit{}, err
	}
	us, err := unitsOf(ctx, q, []db.MovesUnit{row})
	if err != nil {
		return Unit{}, err
	}
	return us[0], nil
}

// save writes u's columns in tx.
func save(ctx context.Context, tx *storage.Tx[file], u Unit) error {
	p, err := columns(u)
	if err != nil {
		return err
	}
	return db.New(tx).UpdateUnit(ctx, p)
}

// factsOf reads what next needs of each of units, with one read per owner
// and kind of fact.
func (m *Moves) factsOf(ctx context.Context, r storage.Reader[file], units []Unit, heads map[UnitID]bool) ([]Facts, error) {
	out := make([]Facts, len(units))
	var switching []int
	for i, u := range units {
		out[i].Head = heads[u.ID]
		from, to := u.nodes()
		out[i].NodesUp = m.cfg.Roster.InStep(from) && m.cfg.Roster.InStep(to)
		if u.Torrent != nil {
			continue
		}
		s := u.Subject
		switch u.Stage.(type) {
		case Switching:
			switching = append(switching, i)
		case Cleaning:
			safe, err := m.cfg.MediaServers.Safe(ctx, r, s.From.Root, libraryFiles(u))
			if err != nil {
				return nil, err
			}
			out[i].Safe = safe
		}
	}
	if err := handoffFacts(ctx, r, units, out); err != nil || len(switching) == 0 {
		return out, err
	}
	return out, m.switchFacts(ctx, r, units, switching, out)
}

// switchFacts reads, into out, the facts of each unit of switching, the
// positions of units at their switch.
func (m *Moves) switchFacts(ctx context.Context, r storage.Reader[file], units []Unit, switching []int, out []Facts) error {
	var ws []works.WorkID
	var fs []files.ID
	var refs []source.Library
	for _, i := range switching {
		u := units[i]
		ws = append(ws, u.Subject.Work)
		fs = append(fs, libraryFiles(u)...)
		for _, f := range u.Files {
			refs = append(refs, sourceOf(u, f))
		}
	}
	importing, err := importingOf(ctx, r, ws)
	if err != nil {
		return err
	}
	sending, err := transfers.Sending(ctx, r, refs)
	if err != nil {
		return err
	}
	writing, err := m.cfg.Writing(ctx, r, fs)
	if err != nil {
		return err
	}
	playing, err := m.cfg.MediaServers.Playing(ctx, r, fs)
	if err != nil {
		return err
	}
	for _, i := range switching {
		u, f := units[i], &out[i]
		f.Importing = importing[u.Subject.Work]
		for _, file := range u.Files {
			f.OpenFiles = f.OpenFiles || sending[sourceOf(u, file)] || writing[file.ID]
			for _, s := range playing[file.ID] {
				f.Playing = appendOnce(f.Playing, s)
			}
		}
		if f.Snapshotted, err = m.cfg.MediaServers.Snapshotted(ctx, r, libraryFiles(u), u.Subject.From.Root); err != nil {
			return err
		}
	}
	return nil
}

// copyFacts reads, into facts, the Transfer's view of each unit of units
// that copies now.
func (m *Moves) copyFacts(ctx context.Context, units []Unit, facts []Facts) error {
	var jobs []string
	for _, u := range units {
		if _, ok := u.Stage.(Copying); ok {
			jobs = append(jobs, u.ID.String())
		}
	}
	if len(jobs) == 0 {
		return nil
	}
	views, err := m.cfg.Transfers.ViewsOf(ctx, transfers.Move, jobs)
	if err != nil {
		return err
	}
	for i, u := range units {
		if v, ok := views[u.ID.String()]; ok {
			facts[i].Transfer = &v
		}
	}
	return nil
}

// sourceOf is f's bytes at u's source, as a Transfer names them.
func sourceOf(u Unit, f File) source.Library {
	return source.Library{Root: [16]byte(u.Subject.From.Root.Bytes()), Rel: f.Sits.String()}
}

// importingOf is which of ws an Import imports into now, read in r.
func importingOf(ctx context.Context, r storage.Reader[file], ws []works.WorkID) (map[works.WorkID]bool, error) {
	items, err := works.ItemsOf(ctx, r, works.Only(ws))
	if err != nil {
		return nil, err
	}
	var ids []works.ItemID
	for _, its := range items {
		for _, it := range its {
			ids = append(ids, it.ID)
		}
	}
	importing, err := imports.Importing(ctx, r, ids)
	if err != nil {
		return nil, err
	}
	out := map[works.WorkID]bool{}
	for w, its := range items {
		for _, it := range its {
			if _, busy := importing[it.ID]; busy {
				out[w] = true
			}
		}
	}
	return out, nil
}

// libraryFiles is the ids of u's library files.
func libraryFiles(u Unit) []files.ID {
	var out []files.ID
	for _, f := range u.Files {
		if f.Kind == KindLibrary {
			out = append(out, f.ID)
		}
	}
	return out
}

// apply writes, in one Write, every Action of judged that needs no IO,
// and a step lease on each unit whose IO step starts, answering those.
// A unit that changed since the pass read it is left for the next pass.
func (m *Moves) apply(ctx context.Context, judged []work, now time.Time) ([]work, error) {
	var steps []work
	var touch []role.NodeID
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		steps, touch = nil, nil
		for _, w := range judged {
			cur, err := load(ctx, tx, w.u.ID)
			if err != nil {
				return err
			}
			if !same(cur, w.u) {
				continue
			}
			step, nodes, err := m.applyOne(ctx, tx, &cur, w.action, now)
			if err != nil {
				return err
			}
			touch = append(touch, nodes...)
			if step {
				steps = append(steps, work{u: cur, facts: w.facts, action: w.action})
			}
			if err := save(ctx, tx, cur); err != nil {
				return err
			}
		}
		return nil
	})
	if err == nil && len(touch) > 0 {
		m.cfg.Clients.Touch(touch...)
	}
	return steps, err
}

// applyOne changes u in tx by a, an Action that needs no IO, or leases
// u's IO step for a and answers true. It answers the Nodes whose
// downloading intent to Touch once tx commits.
func (m *Moves) applyOne(ctx context.Context, tx *storage.Tx[file], u *Unit, a Action, now time.Time) (bool, []role.NodeID, error) {
	switch a := a.(type) {
	case Arrived, Stop, Resume, AskStart, SwitchTorrent, Rollback, Parted:
		m.beforeStep(ctx, *u, a)
		nodes, err := m.applyTorrent(ctx, tx, u, a, now)
		return false, nodes, err
	case Hold:
		u.Stage, u.Due = waiting(u.Stage, a.Wait, now), nil
		if !a.Until.IsZero() {
			u.Due = &a.Until
		}
	case Nothing:
		u.Due = nil
	case ReleaseServers:
		if err := m.cfg.MediaServers.SourcesGone(ctx, tx, u.Subject.From.Root, libraryFiles(*u)); err != nil {
			return false, nil, err
		}
		u.Notes.Released, u.Due = true, &now
	default:
		u.Due = new(now.Add(m.cfg.Timings.StepLease))
		return true, nil, nil
	}
	return false, nil, nil
}

// same is whether cur is still the unit the pass judged.
func same(cur, judged Unit) bool {
	a, errA := columns(cur)
	b, errB := columns(judged)
	return errA == nil && errB == nil && a.Phase == b.Phase && deref(a.Waiting) == deref(b.Waiting) && a.Pass == b.Pass &&
		a.DueAt == b.DueAt && a.SwitchNowAt == b.SwitchNowAt && a.UndoneAt == b.UndoneAt && a.Notes == b.Notes && len(cur.Files) == len(judged.Files)
}

// waiting is st with wait w, its Since kept while w doesn't change.
func waiting(st Stage, w Wait, now time.Time) Stage {
	since := func(was Wait, at time.Time) time.Time {
		if was == w {
			return at
		}
		return now
	}
	switch s := st.(type) {
	case Queued:
		s.Wait, s.Since = w, since(s.Wait, s.Since)
		return s
	case Copying:
		s.Wait, s.Since = w, since(s.Wait, s.Since)
		return s
	case Placing:
		s.Wait, s.Since = w, since(s.Wait, s.Since)
		return s
	case Switching:
		s.Wait, s.Since = w, since(s.Wait, s.Since)
		return s
	case Cleaning:
		s.Wait, s.Since = w, since(s.Wait, s.Since)
		return s
	}
	return st
}

// start runs w's IO step in its own goroutine, at most Timings.Jobs at
// once. A step that fails is due again after retry_after.
func (m *Moves) start(ctx context.Context, w work) {
	m.mu.Lock()
	m.running[w.u.ID] = true
	m.mu.Unlock()
	go func() {
		defer func() {
			m.mu.Lock()
			delete(m.running, w.u.ID)
			m.mu.Unlock()
			m.Poke()
		}()
		select {
		case m.jobs <- struct{}{}:
		case <-ctx.Done():
			return
		}
		defer func() { <-m.jobs }()
		m.beforeStep(ctx, w.u, w.action)
		if err := m.step(ctx, w.u, w.action); err != nil && ctx.Err() == nil {
			m.cfg.Log.Warn("a Move's step failed", "unit", w.u.ID.String(), "step", actionName(w.action), "err", err.Error())
			m.retry(ctx, w.u.ID)
		}
	}()
}

func (m *Moves) beforeStep(ctx context.Context, u Unit, a Action) {
	if m.cfg.BeforeStep != nil {
		m.cfg.BeforeStep(ctx, Step{Unit: u.ID, Work: u.Subject.Work, Action: actionName(a)})
	}
}

// retry has unit id due again after retry_after.
func (m *Moves) retry(ctx context.Context, id UnitID) {
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).SetDue(ctx, db.SetDueParams{ID: id, DueAt: storage.FromTime(m.now().Add(m.cfg.Timings.RetryAfter))})
	})
	if err != nil && ctx.Err() == nil {
		m.cfg.Log.Error("couldn't mark a Move's step to run again", "unit", id.String(), "err", err.Error())
	}
}

// untilDue is the time until the earliest due unit, idle when none is.
func (m *Moves) untilDue(ctx context.Context) (time.Duration, error) {
	var at int64
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		at, err = db.New(r).NextDue(ctx)
		return err
	})
	if err != nil || at == 0 {
		return idle, err
	}
	return max(time.UnixMicro(at).Sub(m.now()), time.Millisecond), nil
}
