package health

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jatassi/slipstream/internal/health/internal/db"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

type Hub interface {
	Watch(ctx context.Context, events ...live.Event) <-chan struct{}
}

type Config struct {
	Library *storage.DB[file]
	Hub     Hub
	Changed ChangedPublisher
	Storage StorageParts
	// Categories follow Storage, in IDs' order.
	Categories []Category
	// Names is every live Node's name, for the own levels of volumes that
	// no longer report. A Node it lacks is removed, and its volumes' own
	// levels go.
	Names func(ctx context.Context, rd storage.Reader[file]) (map[role.NodeID]string, error)
	// NameEvents are the events after which Names may answer differently.
	NameEvents []live.Event
	// Transitions, when set, takes each evaluation's transitions inside
	// the Write that records them, so each is seen exactly once.
	Transitions func(ctx context.Context, tx *storage.Tx[file], ts []Transition) error
	Zone        *time.Location
	Timings     Timings
	Log         *slog.Logger
}

var (
	ErrNoCategory = errors.New("no Health category has that id")
	ErrNoItem     = errors.New("the category has no item with that key")
	ErrNoCheck    = errors.New("there is no Check now for that")
	errMoved      = errors.New("the health item moved under the evaluator")
)

// Checked is the Report right after a Check now, and what it couldn't
// reach.
type Checked struct {
	Report
	Unreached []Unreached
}

type ask struct {
	ids []ID
	// publish: send health.changed for ids even when nothing shown moved.
	publish bool
	reply   chan Report
}

type Health struct {
	cfg   Config
	all   []Category
	byID  map[ID]int
	asks  chan ask
	wake  chan struct{}
	ready chan struct{}
	// report is the last evaluation's Report, nil before the first.
	report atomic.Pointer[Report]

	mu    sync.Mutex
	dirty map[ID]bool

	// last is each category's last good words, for an unsure key, a
	// failed answer's rows and a cleared item's transition. Only Run
	// touches it.
	last map[ID]map[Key]Says
	// rows is health_item as Run last committed it. Only Run touches it.
	rows map[ID]map[Key]row
	// next is when Run asks each category again with nothing published:
	// its answer's Next, or Timings.Retry after a failure. Only Run
	// touches it.
	next map[ID]time.Time

	// levelsMu holds each levels Write until levels keeps what it
	// committed, so levels is always the stored levels.
	levelsMu sync.Mutex
	levels   atomic.Pointer[Levels]
	// mayPrune: the Node names or the own levels may have changed since
	// the last look for own levels on removed Nodes.
	mayPrune atomic.Bool
}

// Open checks the registry, panicking on a breach, and deletes the rows
// of categories this build doesn't register, publishing nothing.
func Open(ctx context.Context, cfg Config) (*Health, error) {
	h := &Health{
		cfg:   cfg,
		byID:  map[ID]int{},
		asks:  make(chan ask),
		wake:  make(chan struct{}, 1),
		ready: make(chan struct{}),
		dirty: map[ID]bool{},
		last:  map[ID]map[Key]Says{},
		next:  map[ID]time.Time{},
	}
	h.all = append([]Category{h.storage()}, cfg.Categories...)
	registered(h.all)
	for _, s := range []setting.Space{StorageWarnBelow, StorageErrorBelow} {
		if err := s.Check(); err != nil {
			panic(fmt.Sprintf("health: %v", err))
		}
	}
	for i, c := range h.all {
		h.byID[c.ID] = i
	}
	err := cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		rows, err := itemsIn(ctx, tx)
		if err != nil {
			return err
		}
		for id := range rows {
			if _, ok := h.byID[id]; !ok {
				if err := q.DeleteCategory(ctx, string(id)); err != nil {
					return err
				}
				delete(rows, id)
			}
		}
		h.rows = rows
		l, err := LevelsIn(ctx, tx)
		if err != nil {
			return err
		}
		h.levels.Store(&l)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("open health: %w", err)
	}
	h.mayPrune.Store(true)
	return h, nil
}

// Run is the only writer of health_item. It evaluates every category at
// start, a category after its Events or at its answer's Next, one that
// failed again after Timings.Retry, and whatever Evaluate asks, until ctx
// ends.
func (h *Health) Run(ctx context.Context) error {
	var follows sync.WaitGroup
	defer follows.Wait()
	follow := func(events []live.Event, then func()) {
		if len(events) == 0 {
			return
		}
		woken := h.cfg.Hub.Watch(ctx, events...)
		follows.Go(func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-woken:
					then()
				}
			}
		})
	}
	for _, c := range h.all {
		follow(c.Events, func() { h.mark(c.ID) })
	}
	follow(h.cfg.NameEvents, func() {
		h.mayPrune.Store(true)
		h.mark(StorageID)
	})
	h.evaluate(ctx, h.ids(), false)
	close(h.ready)
	due := time.NewTimer(0)
	defer due.Stop()
	for {
		due.Stop()
		if at := h.earliest(); !at.IsZero() {
			due.Reset(time.Until(at))
		}
		select {
		case <-ctx.Done():
			return nil
		case now := <-due.C:
			h.evaluate(ctx, append(h.take(), h.due(now)...), false)
		case <-h.wake:
			h.evaluate(ctx, h.take(), false)
		case a := <-h.asks:
			h.evaluate(ctx, append(a.ids, h.take()...), a.publish)
			a.reply <- *h.report.Load()
		}
	}
}

func (h *Health) ids() []ID {
	ids := make([]ID, len(h.all))
	for i, c := range h.all {
		ids[i] = c.ID
	}
	return ids
}

// earliest is the soonest next, zero when none is set.
func (h *Health) earliest() time.Time {
	var at time.Time
	for _, t := range h.next {
		if !t.IsZero() && (at.IsZero() || t.Before(at)) {
			at = t
		}
	}
	return at
}

// due is the categories whose next has come by now.
func (h *Health) due(now time.Time) []ID {
	var ids []ID
	for id, t := range h.next {
		if !t.IsZero() && !t.After(now) {
			ids = append(ids, id)
		}
	}
	return ids
}

func (h *Health) mark(id ID) {
	h.mu.Lock()
	h.dirty[id] = true
	h.mu.Unlock()
	select {
	case h.wake <- struct{}{}:
	default:
	}
}

func (h *Health) take() []ID {
	h.mu.Lock()
	defer h.mu.Unlock()
	var ids []ID
	for id := range h.dirty {
		ids = append(ids, id)
	}
	clear(h.dirty)
	return ids
}

// Evaluate asks Run to evaluate ids now and answers the Report after it.
func (h *Health) Evaluate(ctx context.Context, ids ...ID) (Report, error) {
	for _, id := range ids {
		if _, ok := h.byID[id]; !ok {
			return Report{}, ErrNoCategory
		}
	}
	return h.ask(ctx, ask{ids: ids})
}

func (h *Health) ask(ctx context.Context, a ask) (Report, error) {
	a.reply = make(chan Report, 1)
	select {
	case h.asks <- a:
	case <-ctx.Done():
		return Report{}, fmt.Errorf("evaluate health: %w", ctx.Err())
	}
	select {
	case r := <-a.reply:
		return r, nil
	case <-ctx.Done():
		return Report{}, fmt.Errorf("evaluate health: %w", ctx.Err())
	}
}

// Report is the last evaluation. It waits for the first.
func (h *Health) Report(ctx context.Context) (Report, error) {
	select {
	case <-h.ready:
		return *h.report.Load(), nil
	case <-ctx.Done():
		return Report{}, fmt.Errorf("read health: %w", ctx.Err())
	}
}

// Worst is the worst level among ids, or among every category when ids is
// empty. It is None before the first evaluation.
func (h *Health) Worst(ids ...ID) Level {
	r := h.report.Load()
	if r == nil {
		return None
	}
	return r.Worst(ids...)
}

// Check runs Check now on the category id, or on its item key when key is
// not "", then evaluates the category, so the Report it answers holds what
// the check found.
func (h *Health) Check(ctx context.Context, id ID, key Key) (Checked, error) {
	i, ok := h.byID[id]
	if !ok {
		return Checked{}, ErrNoCategory
	}
	c := h.all[i]
	if key != "" {
		r, err := h.Report(ctx)
		if err != nil {
			return Checked{}, err
		}
		it, ok := r.item(id, key)
		if !ok {
			return Checked{}, ErrNoItem
		}
		if it.Action.kind != ActionCheck {
			return Checked{}, ErrNoCheck
		}
	}
	if c.Check == nil {
		return Checked{}, ErrNoCheck
	}
	ctx, cancel := context.WithTimeout(ctx, h.cfg.Timings.Check)
	defer cancel()
	unreached, err := c.Check(ctx, key)
	if err != nil {
		return Checked{}, fmt.Errorf("check %s: %w", c.Phrase, err)
	}
	r, err := h.Evaluate(ctx, id)
	return Checked{Report: r, Unreached: unreached}, err
}

// CheckAll runs every category's Check now at once, then evaluates them
// all.
func (h *Health) CheckAll(ctx context.Context) (Checked, error) {
	ctx, cancel := context.WithTimeout(ctx, h.cfg.Timings.Check)
	defer cancel()
	unreached := make([][]Unreached, len(h.all))
	errs := make([]error, len(h.all))
	var wg sync.WaitGroup
	for i, c := range h.all {
		if c.Check == nil {
			continue
		}
		wg.Go(func() {
			unreached[i], errs[i] = c.Check(ctx, "")
			if errs[i] != nil {
				errs[i] = fmt.Errorf("check %s: %w", c.Phrase, errs[i])
			}
		})
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		return Checked{}, err
	}
	r, err := h.Evaluate(ctx, h.ids()...)
	return Checked{Report: r, Unreached: slices.Concat(unreached...)}, err
}

// evaluate asks each of ids for its answer, writes what changed with its
// transitions in one Write, stores the new Report, and then publishes
// health.changed for the categories whose shown items changed, and for
// every one of ids when publish is set.
func (h *Health) evaluate(ctx context.Context, ids []ID, publish bool) {
	ids = h.ordered(ids)
	answers, failed := h.answers(ctx, ids)
	changed := map[ID]bool{}
	if publish {
		for _, id := range ids {
			changed[id] = true
		}
	}
	defer func() { h.publish(ctx, changed) }()
	retry := time.Now().Add(h.cfg.Timings.Retry)
	for i, id := range ids {
		h.next[id] = answers[i].Next
		if failed[i] != nil {
			h.next[id] = retry
		}
	}
	pruned, err := h.record(ctx, ids, answers, failed)
	if err != nil {
		for _, id := range ids {
			h.next[id] = retry
		}
		h.unread(ids, err)
		return
	}
	if pruned {
		changed[StorageID] = true
	}
	prev := h.report.Load()
	cats := make([]CategoryReport, len(h.all))
	if prev != nil {
		copy(cats, prev.Categories)
	}
	for i, id := range ids {
		at := h.byID[id]
		cats[at] = shownOf(h.all[at], h.rows[id], answers[i], failed[i], h.last[id])
		if failed[i] == nil {
			h.last[id] = wordsOf(answers[i])
		}
		if prev != nil && !reflect.DeepEqual(prev.Categories[at], cats[at]) {
			changed[id] = true
		}
	}
	r := reportOf(cats)
	h.report.Store(&r)
}

// answers is each of ids' answer, or why it failed.
func (h *Health) answers(ctx context.Context, ids []ID) ([]Answer, []error) {
	answers := make([]Answer, len(ids))
	failed := make([]error, len(ids))
	for i, id := range ids {
		c := h.all[h.byID[id]]
		answers[i], failed[i] = c.Answer(ctx)
		if failed[i] == nil {
			failed[i] = valid(c, answers[i])
		}
		if failed[i] != nil {
			h.cfg.Log.Error("answer what is wrong", "category", id, "err", failed[i])
		}
	}
	return answers, failed
}

// record diffs the answers against the stored rows and writes the changes
// with their transitions in one Write, after pruning the own levels of
// removed Nodes' volumes when Storage is among ids. An evaluation that
// changes nothing touches no storage.
func (h *Health) record(ctx context.Context, ids []ID, answers []Answer, failed []error) (pruned bool, err error) {
	if slices.Contains(ids, StorageID) {
		if pruned, err = h.prune(ctx); err != nil {
			return false, err
		}
	}
	now := storage.Now().Time()
	changes := make([][]change, len(ids))
	var ts []Transition
	for i, id := range ids {
		changes[i] = diff(h.rows[id], answers[i], failed[i] != nil, now)
		ts = append(ts, h.transitions(id, changes[i], answers[i], now)...)
	}
	if len(ts) == 0 {
		return pruned, nil
	}
	err = h.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		if err := write(ctx, db.New(tx), ids, changes); err != nil {
			return err
		}
		if h.cfg.Transitions == nil {
			return nil
		}
		return h.cfg.Transitions(ctx, tx, ts)
	})
	if err != nil {
		return false, fmt.Errorf("write the health items: %w", err)
	}
	for i, id := range ids {
		h.rows[id] = applied(h.rows[id], changes[i])
	}
	return pruned, nil
}

// unread keeps ids' last reports, or none before the first, with err as
// why they couldn't be read.
func (h *Health) unread(ids []ID, err error) {
	h.cfg.Log.Error("evaluate health", "err", err)
	prev := h.report.Load()
	cats := make([]CategoryReport, len(h.all))
	if prev != nil {
		copy(cats, prev.Categories)
	}
	for _, id := range ids {
		at := h.byID[id]
		if prev == nil {
			cats[at] = CategoryReport{About: h.all[at].About, CanCheck: h.all[at].Check != nil}
		}
		cats[at].Unread = err.Error()
	}
	r := reportOf(cats)
	h.report.Store(&r)
}

func (h *Health) ordered(ids []ID) []ID {
	var out []ID
	for _, id := range h.ids() {
		if slices.Contains(ids, id) {
			out = append(out, id)
		}
	}
	return out
}

func (h *Health) publish(ctx context.Context, changed map[ID]bool) {
	if len(changed) == 0 {
		return
	}
	var cs Changed
	for _, id := range h.ids() {
		if changed[id] {
			cs.Categories = append(cs.Categories, string(id))
		}
	}
	h.cfg.Changed.Publish(ctx, cs)
}

func (h *Health) transitions(id ID, changes []change, a Answer, now time.Time) []Transition {
	words := map[Key]Says{}
	for _, it := range a.Items {
		words[it.Key] = it.Says
	}
	var out []Transition
	for _, c := range changes {
		says, ok := words[c.Key]
		if c.Kind == Cleared {
			says, ok = h.last[id][c.Key]
		}
		if !ok {
			says = Says{Subject: h.all[h.byID[id]].Label + ": a problem that cleared while SlipStream was stopped"}
		}
		out = append(out, Transition{Category: id, Key: c.Key, Kind: c.Kind, From: c.From, To: c.To, Subject: says.Subject, Text: says.Text, At: now})
	}
	return out
}

func wordsOf(a Answer) map[Key]Says {
	out := make(map[Key]Says, len(a.Items)+len(a.Unsure))
	for _, it := range a.Items {
		out[it.Key] = it.Says
	}
	for _, u := range a.Unsure {
		out[u.Key] = u.Says
	}
	return out
}

func itemsIn(ctx context.Context, rd storage.Reader[file]) (map[ID]map[Key]row, error) {
	all, err := db.New(rd).ListItems(ctx)
	if err != nil {
		return nil, err
	}
	out := map[ID]map[Key]row{}
	for _, r := range all {
		id := ID(r.Category)
		if out[id] == nil {
			out[id] = map[Key]row{}
		}
		out[id][Key(r.Key)] = row{Level: levelOf(r.Level), Since: r.Since.Time(), ChangedAt: r.ChangedAt.Time()}
	}
	return out, nil
}

// prune deletes the own levels of volumes on Nodes no longer named, when
// the names or the levels may have changed since it last looked.
func (h *Health) prune(ctx context.Context) (bool, error) {
	if len(h.Levels().own) == 0 || !h.mayPrune.Swap(false) {
		return false, nil
	}
	var pruned bool
	err := h.writeLevels(ctx, func(ctx context.Context, tx *storage.Tx[file], l *Levels) error {
		names, err := h.cfg.Names(ctx, tx)
		if err != nil {
			return err
		}
		for at := range l.own {
			if _, ok := names[at.node]; ok {
				continue
			}
			if err := db.New(tx).DeleteVolumeLevels(ctx, db.DeleteVolumeLevelsParams{NodeID: at.node, Mount: at.mount}); err != nil {
				return err
			}
			delete(l.own, at)
			pruned = true
		}
		return nil
	})
	if err != nil {
		h.mayPrune.Store(true)
		return false, fmt.Errorf("prune the levels of removed Nodes' volumes: %w", err)
	}
	return pruned, nil
}

// write writes changes. Run alone writes health_item, so each row it
// updates or deletes is there; one that isn't fails the Write.
func write(ctx context.Context, q *db.Queries, ids []ID, changes [][]change) error {
	for i, id := range ids {
		for _, c := range changes[i] {
			if err := writeItem(ctx, q, id, c); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeItem(ctx context.Context, q *db.Queries, id ID, c change) error {
	var n int64
	var err error
	switch c.Kind {
	case Raised:
		return q.InsertItem(ctx, db.InsertItemParams{
			Category: string(id), Key: string(c.Key), Level: c.To.String(),
			Since: storage.FromTime(c.Row.Since), ChangedAt: storage.FromTime(c.Row.ChangedAt),
		})
	case Worse, Better:
		n, err = q.UpdateItemLevel(ctx, db.UpdateItemLevelParams{
			Category: string(id), Key: string(c.Key), Level: c.To.String(), ChangedAt: storage.FromTime(c.Row.ChangedAt),
		})
	case Cleared:
		n, err = q.DeleteItem(ctx, db.DeleteItemParams{Category: string(id), Key: string(c.Key)})
	}
	if err == nil && n != 1 {
		err = fmt.Errorf("%s %s: %w", id, c.Key, errMoved)
	}
	return err
}
