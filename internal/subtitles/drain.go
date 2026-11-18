package subtitles

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

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
			d.s.cfg.Log.WarnContext(ctx, "The subtitle search drain failed", "error", err)
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
// first queuing fresh imports' and the Admin's due wants whose files'
// audio is now scanned. Nothing runs while Subtitle search is off, the
// library Reconciles or no Node has the Subtitles Role.
func (d *searchDrain) pass(ctx context.Context, wg *sync.WaitGroup) error {
	s := d.s
	if len(s.cfg.Roster.Holding(role.Subtitles)) == 0 {
		return nil
	}
	var on bool
	var ps []Provider
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		halt, err := s.cfg.Reconciling(ctx, tx)
		if err != nil || halt {
			return err
		}
		q := db.New(tx)
		settings, err := q.GetSettings(ctx)
		if err != nil {
			return err
		}
		if on = SearchOn.Value(settings.SearchOn); !on {
			return nil
		}
		if _, err := s.queue(ctx, q, false, SearchUpgrades.Value(settings.SearchUpgrades)); err != nil {
			return err
		}
		ps, err = s.providersIn(ctx, tx)
		return err
	})
	if err != nil || !on {
		return err
	}
	s.recheck(ctx, ps)
	room := s.cfg.Timings.DrainWidth - d.busy()
	if room <= 0 {
		return nil
	}
	next, err := d.next(ctx, readyKinds(ps, s.cfg.Clock()), room)
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

// next is up to room queued wants of kinds in queue order, leaving out
// those an attempt runs for or a pick or an upload waits for.
func (d *searchDrain) next(ctx context.Context, kinds []subtitling.Kind, room int) ([]WantKey, error) {
	var rows []db.SubtitlesWant
	err := d.s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		q := db.New(r)
		for _, k := range kinds {
			got, err := q.NextQueued(ctx, db.NextQueuedParams{Kind: kindNames[k], N: int64(room + d.busy())})
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

func (d *searchDrain) start(ctx context.Context, wg *sync.WaitGroup, k WantKey) {
	d.mu.Lock()
	d.running[k] = true
	d.mu.Unlock()
	wg.Go(func() {
		defer func() {
			d.mu.Lock()
			delete(d.running, k)
			d.mu.Unlock()
			d.wake()
		}()
		err := d.s.run(ctx, k)
		if err != nil && !errors.Is(err, ErrBusy) && !errors.Is(err, errStale) && ctx.Err() == nil {
			d.s.cfg.Log.WarnContext(ctx, "A subtitle search failed", "file", k.File, "want", k.Want, "error", err)
		}
	})
}

// queue stamps up to a chunk of due wants whose files' audio is scanned
// onto the queue at a time, all of them on Subtitle search's run, else
// only fresh imports' and those an Admin or a report asked for. Upgrades
// queue only while searchUpgrades is on.
func (s *Subtitles) queue(ctx context.Context, q *db.Queries, all, upgrades bool) (int, error) {
	now := storage.FromTime(s.cfg.Clock())
	var n int
	for {
		got, err := q.QueueDue(ctx, db.QueueDueParams{Now: now, Upgrades: flag(upgrades), All: flag(all), N: int64(s.cfg.Timings.ReconcileChunk)})
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
