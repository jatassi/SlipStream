package subtitles

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
	"github.com/jatassi/slipstream/internal/subtitling"
)

// searchDrain works the queued wants in queue order, at most
// Timings.DrainWidth at once. Each pass reads only the kinds some ready
// provider searches, so a want whose every provider is paused stays
// queued and unread until one resumes.
type searchDrain struct {
	s    *Subtitles
	ring chan struct{}

	mu      sync.Mutex
	running map[WantKey]bool
}

func newSearchDrain(s *Subtitles) *searchDrain {
	return &searchDrain{s: s, ring: make(chan struct{}, 1), running: map[WantKey]bool{}}
}

// soon runs a pass after Timings.WakeDelay, by when the Write that asked
// for it has committed.
func (d *searchDrain) soon() { time.AfterFunc(d.s.cfg.Timings.WakeDelay, d.wake) }

func (d *searchDrain) wake() {
	select {
	case d.ring <- struct{}{}:
	default:
	}
}

// run passes at once, on each wake, and every Timings.KeeperPoll, when a
// pause may have ended.
func (d *searchDrain) run(ctx context.Context) {
	var wg sync.WaitGroup
	defer wg.Wait()
	poll := time.NewTicker(d.s.cfg.Timings.KeeperPoll)
	defer poll.Stop()
	for {
		if err := d.pass(ctx, &wg); err != nil && ctx.Err() == nil {
			d.s.cfg.Log.WarnContext(ctx, "Running the queued subtitle searches failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-d.ring:
		case <-poll.C:
		}
	}
}

// pass starts attempts for the next queued wants of the ready kinds,
// first queuing fresh imports', the Admin's and a pause's due wants whose
// files' audio is now scanned. While Subtitle search is off only the
// searches an Admin or a report asked for run. Nothing runs while the
// library Reconciles or no Node with the Subtitles Role is in step.
func (d *searchDrain) pass(ctx context.Context, wg *sync.WaitGroup) error {
	s := d.s
	if !slices.ContainsFunc(s.cfg.Roster.Holding(role.Subtitles), s.cfg.Roster.InStep) {
		return nil
	}
	var scope queued
	var ps []Provider
	var halt bool
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		if halt, err = s.cfg.Reconciling(ctx, tx); err != nil || halt {
			return err
		}
		q := db.New(tx)
		settings, err := q.GetSettings(ctx)
		if err != nil {
			return err
		}
		scope = soonest
		if !SearchOn.Value(settings.SearchOn) {
			scope = askedFor
		}
		if _, err := s.queue(ctx, q, scope, SearchUpgrades.Value(settings.SearchUpgrades)); err != nil {
			return err
		}
		ps, err = s.providersIn(ctx, tx)
		return err
	})
	if err != nil || halt {
		return err
	}
	s.recheck(ctx, ps)
	room := s.cfg.Timings.DrainWidth - d.busy()
	if room <= 0 {
		return nil
	}
	next, err := d.next(ctx, readyKinds(ps, s.cfg.Clock()), room, scope == askedFor)
	if err != nil {
		return err
	}
	for _, k := range next {
		d.start(ctx, wg, k)
	}
	return nil
}

func (d *searchDrain) busy() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.running)
}

// next is up to room queued wants of kinds in queue order, only those
// someone asked for when asked is set, leaving out those an attempt runs
// for, a pick or an upload waits for, or whose file's Node isn't in step.
// Those stay queued until it is again.
func (d *searchDrain) next(ctx context.Context, kinds []subtitling.Kind, room int, asked bool) ([]WantKey, error) {
	var rows []db.SubtitlesWant
	err := d.s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		q := db.New(r)
		for _, k := range kinds {
			got, err := d.reachable(ctx, q, kindNames[k], room+d.busy(), asked)
			if err != nil {
				return err
			}
			rows = append(rows, got...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.SortFunc(rows, func(a, b db.SubtitlesWant) int {
		return cmp.Or(cmp.Compare(deref(a.QueueRank), deref(b.QueueRank)), cmp.Compare(deref(a.QueueOrder), deref(b.QueueOrder)), a.FileID.Compare(b.FileID.UUID))
	})
	d.mu.Lock()
	defer d.mu.Unlock()
	var out []WantKey
	for _, row := range rows {
		l, err := language.Parse(row.Language)
		if err != nil {
			return nil, err
		}
		k := WantKey{File: row.FileID, Want: Want{Language: l, Forced: row.Forced}}
		if d.running[k] || d.s.leases.waited(k) {
			continue
		}
		if out = append(out, k); len(out) == room {
			break
		}
	}
	return out, nil
}

// reachable is the first n queued wants of kind whose files' Nodes are
// in step, only those someone asked for when asked is set. It reads past
// the queue's head while a Node that is away holds it, so that Node's
// wants never stall the others'.
func (d *searchDrain) reachable(ctx context.Context, q *db.Queries, kind string, n int, asked bool) ([]db.SubtitlesWant, error) {
	for read := n; ; read *= 2 {
		got, err := q.NextQueued(ctx, db.NextQueuedParams{Kind: kind, N: int64(read)})
		if err != nil {
			return nil, err
		}
		end := len(got) < read
		if i := slices.IndexFunc(got, func(row db.SubtitlesWant) bool { return asked && deref(row.QueueRank) > askedRank }); i >= 0 {
			got, end = got[:i], true
		}
		ids := make([]files.ID, len(got))
		for i, row := range got {
			ids[i] = row.FileID
		}
		fs, err := q.ListFilesIn(ctx, ids)
		if err != nil {
			return nil, err
		}
		away := map[files.ID]bool{}
		for _, f := range fs {
			away[f.FileID] = !f.MapNodeID.IsZero() && !d.s.cfg.Roster.InStep(f.MapNodeID)
		}
		out := slices.DeleteFunc(got, func(row db.SubtitlesWant) bool { return away[row.FileID] })
		if len(out) >= n || end {
			return out[:min(len(out), n)], nil
		}
	}
}

func (d *searchDrain) start(ctx context.Context, wg *sync.WaitGroup, k WantKey) {
	d.mu.Lock()
	d.running[k] = true
	d.mu.Unlock()
	wg.Go(func() {
		err := d.s.run(ctx, k)
		d.mu.Lock()
		delete(d.running, k)
		d.mu.Unlock()
		// An attempt that ended its want frees its room at once. One that
		// failed leaves the want queued, so the drain tries it again at its
		// next wake or poll, never at once: a Node that doesn't answer
		// would be asked in a loop.
		if err == nil {
			d.wake()
		}
		if err != nil && !errors.Is(err, ErrBusy) && !errors.Is(err, errStale) && ctx.Err() == nil {
			d.s.cfg.Log.WarnContext(ctx, "A subtitle search failed", "file", k.File, "want", k.Want, "error", err)
		}
	})
}

// queued is which due wants a queue pass stamps onto the queue.
type queued uint8

const (
	// askedFor is the searches an Admin or a report asked for, which run
	// while Subtitle search is off.
	askedFor queued = iota
	// soonest adds fresh imports' wants and those a provider's pause held.
	soonest
	// everyDue is every due want, on Subtitle search's run.
	everyDue
)

// askedRank is the last queue rank of a search someone asked for.
var askedRank = max(priorities[ByAdmin].rank, priorities[ByReport].rank)

// queue stamps up to a chunk of the due wants scope names whose files'
// audio is scanned onto the queue at a time. Upgrades queue only while
// searchUpgrades is on.
func (s *Subtitles) queue(ctx context.Context, q *db.Queries, scope queued, upgrades bool) (int, error) {
	now := storage.FromTime(s.cfg.Clock())
	var n int
	for {
		got, err := q.QueueDue(ctx, db.QueueDueParams{
			Now: now, Upgrades: flag(upgrades), All: flag(scope == everyDue), Soonest: flag(scope >= soonest), N: int64(s.cfg.Timings.ReconcileChunk),
		})
		if err != nil {
			return n, err
		}
		n += int(got)
		if got < int64(s.cfg.Timings.ReconcileChunk) {
			return n, nil
		}
	}
}

// slots lets one call at a time reach each provider.
type slots struct {
	mu sync.Mutex
	of map[subtitling.ProviderID]chan struct{}
}

// take holds each of ids' slots, in id order so two takers never wait on
// each other, and answers the func that frees them.
func (sl *slots) take(ctx context.Context, ids ...subtitling.ProviderID) (func(), error) {
	ids = slices.Sorted(slices.Values(ids))
	var held []chan struct{}
	done := func() {
		for _, c := range held {
			<-c
		}
	}
	for _, id := range ids {
		c := sl.slot(id)
		select {
		case c <- struct{}{}:
			held = append(held, c)
		case <-ctx.Done():
			done()
			return nil, fmt.Errorf("wait for a provider's turn: %w", context.Cause(ctx))
		}
	}
	return done, nil
}

func (sl *slots) slot(id subtitling.ProviderID) chan struct{} {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	if sl.of == nil {
		sl.of = map[subtitling.ProviderID]chan struct{}{}
	}
	if sl.of[id] == nil {
		sl.of[id] = make(chan struct{}, 1)
	}
	return sl.of[id]
}
