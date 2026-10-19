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
		rows, err := q.ListItems(ctx)
		if err != nil {
			return err
		}
		for _, r := range rows {
			if _, ok := h.byID[ID(r.Category)]; !ok {
				if err := q.DeleteCategory(ctx, r.Category); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("open health: %w", err)
	}
	return h, nil
}

// Run is the only writer of health_item. It evaluates every category at
// start and every Timings.Reevaluate, a category after its Events, and
// whatever Evaluate asks, until ctx ends.
func (h *Health) Run(ctx context.Context) error {
	var follows sync.WaitGroup
	defer follows.Wait()
	for _, c := range h.all {
		if len(c.Events) == 0 {
			continue
		}
		woken := h.cfg.Hub.Watch(ctx, c.Events...)
		follows.Go(func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-woken:
					h.mark(c.ID)
				}
			}
		})
	}
	h.evaluate(ctx, h.ids(), false)
	close(h.ready)
	tick := time.NewTicker(h.cfg.Timings.Reevaluate)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
			h.take()
			h.evaluate(ctx, h.ids(), false)
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
	rows, changes, pruned, err := h.record(ctx, ids, answers, failed)
	if err != nil {
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
		cats[at] = shownOf(h.all[at], applied(rows[id], changes[i]), answers[i], failed[i], h.last[id])
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
// with their transitions, pruning the own levels of removed Nodes' volumes
// when Storage is among ids. It answers the rows it read and the changes.
func (h *Health) record(ctx context.Context, ids []ID, answers []Answer, failed []error) (rows map[ID]map[Key]row, changes [][]change, pruned bool, err error) {
	var removed []volumeAt
	err = h.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
		var err error
		if rows, err = itemsIn(ctx, rd); err != nil {
			return err
		}
		if slices.Contains(ids, StorageID) {
			removed, err = h.removed(ctx, rd)
		}
		return err
	})
	if err != nil {
		return nil, nil, false, fmt.Errorf("read the health items: %w", err)
	}
	now := storage.Now().Time()
	changes = make([][]change, len(ids))
	var ts []Transition
	for i, id := range ids {
		changes[i] = diff(rows[id], answers[i], failed[i] != nil, now)
		ts = append(ts, h.transitions(id, changes[i], answers[i], now)...)
	}
	if len(ts) == 0 && len(removed) == 0 {
		return rows, changes, false, nil
	}
	err = h.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		if err := write(ctx, db.New(tx), ids, changes, removed); err != nil {
			return err
		}
		if h.cfg.Transitions == nil || len(ts) == 0 {
			return nil
		}
		return h.cfg.Transitions(ctx, tx, ts)
	})
	if err != nil {
		return nil, nil, false, fmt.Errorf("write the health items: %w", err)
	}
	return rows, changes, len(removed) > 0, nil
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

// removed is the own levels of volumes on Nodes no longer named.
func (h *Health) removed(ctx context.Context, rd storage.Reader[file]) ([]volumeAt, error) {
	l, err := LevelsIn(ctx, rd)
	if err != nil || len(l.own) == 0 {
		return nil, err
	}
	names, err := h.cfg.Names(ctx, rd)
	if err != nil {
		return nil, err
	}
	var out []volumeAt
	for at := range l.own {
		if _, ok := names[at.node]; !ok {
			out = append(out, at)
		}
	}
	return out, nil
}

func write(ctx context.Context, q *db.Queries, ids []ID, changes [][]change, removed []volumeAt) error {
	for i, id := range ids {
		for _, c := range changes[i] {
			var err error
			switch c.Kind {
			case Raised:
				err = q.InsertItem(ctx, db.InsertItemParams{
					Category: string(id), Key: string(c.Key), Level: c.To.String(),
					Since: storage.FromTime(c.Row.Since), ChangedAt: storage.FromTime(c.Row.ChangedAt),
				})
			case Worse, Better:
				err = q.UpdateItemLevel(ctx, db.UpdateItemLevelParams{
					Category: string(id), Key: string(c.Key), Level: c.To.String(), ChangedAt: storage.FromTime(c.Row.ChangedAt),
				})
			case Cleared:
				err = q.DeleteItem(ctx, db.DeleteItemParams{Category: string(id), Key: string(c.Key)})
			}
			if err != nil {
				return err
			}
		}
	}
	for _, at := range removed {
		if err := q.DeleteVolumeLevels(ctx, db.DeleteVolumeLevelsParams{NodeID: at.node, Mount: at.mount}); err != nil {
			return err
		}
	}
	return nil
}
