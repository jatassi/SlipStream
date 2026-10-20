package metadata

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/metadata/internal/announced"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// LocalSchema is the metadata local set: the last day the availability
// timer announced.
func LocalSchema() storage.Schema[storage.Local] { return announced.Schema() }

// Released is the hook S34 (Automatic search, Backlog search and RSS
// sync) registers: it gets the monitored Items whose availability date
// arrived, at least once each. A crash before the day is recorded, or an
// error from the hook, hands the same Items over again, so the hook must
// be idempotent.
type Released func(ctx context.Context, items []works.ItemID) error

// NoBacklog is the hook until S34 (Automatic search, Backlog search and
// RSS sync) registers its own.
func NoBacklog(context.Context, []works.ItemID) error { return nil }

const retry = time.Minute

type AvailabilityConfig struct {
	Library *storage.DB[storage.Library]
	Local   *storage.DB[storage.Local]
	Targets *targets.Targets
	// Hub wakes the timer on works.changed and works.type-changed.
	Hub     *live.Hub
	Changed works.Publisher[works.WorkChanged]
	// Now is the time in the Controller's zone, whose calendar days the
	// timer announces.
	Now      func() time.Time
	Released Released
	Log      *slog.Logger
}

// Availability is the availability timer, one on the Controller: on each
// calendar day a monitored Item's availability date falls, it announces
// that Item through works.changed and the Released hook. It is not a
// Scheduled task.
type Availability struct{ cfg AvailabilityConfig }

func NewAvailability(cfg AvailabilityConfig) *Availability { return &Availability{cfg: cfg} }

// Run announces every day after the last one announced through today, a
// missing last day meaning yesterday, then sleeps until the start of the
// next day a monitored Item's availability date falls on, or until a
// works.changed or works.type-changed, and announces again. It returns
// once ctx ends.
func (a *Availability) Run(ctx context.Context) error {
	woken := a.cfg.Hub.Watch(ctx, works.ChangedEvent, works.TypeChangedEvent)
	for {
		if a.sleep(ctx, woken, a.pass(ctx)) {
			return nil
		}
	}
}

func (a *Availability) sleep(ctx context.Context, woken <-chan struct{}, wake time.Time) bool {
	var timer <-chan time.Time
	if !wake.IsZero() {
		t := time.NewTimer(wake.Sub(a.cfg.Now()))
		defer t.Stop()
		timer = t.C
	}
	select {
	case <-ctx.Done():
		return true
	case <-timer:
	case <-woken:
	}
	return false
}

// pass announces through today and answers when to wake: the start of
// the next day an Item is due, zero when none is.
func (a *Availability) pass(ctx context.Context) time.Time {
	now := a.cfg.Now()
	next, err := a.announce(ctx, calendarDay(now))
	if err != nil {
		a.cfg.Log.Error("announce released Items", "err", err)
		return now.Add(retry)
	}
	if next.IsZero() {
		return time.Time{}
	}
	return time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, now.Location())
}

// calendarDay is t's calendar date in its own location, as a UTC midnight
// like targets.Timing.From.
func calendarDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

type arrival struct {
	work works.WorkID
	item works.ItemID
	from time.Time
}

func (a *Availability) announce(ctx context.Context, today time.Time) (next time.Time, err error) {
	var (
		last time.Time
		kept bool
	)
	err = a.cfg.Local.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Local]) error {
		last, kept, err = announced.Day(ctx, r)
		return err
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("read the last day announced: %w", err)
	}
	if !kept {
		last = today.AddDate(0, 0, -1)
	}
	arrivals, err := a.arrivals(ctx)
	if err != nil {
		return time.Time{}, err
	}
	var (
		due     []works.ItemID
		touched []works.WorkID
	)
	for _, ar := range arrivals {
		switch {
		case ar.from.After(today):
			if next.IsZero() || ar.from.Before(next) {
				next = ar.from
			}
		case ar.from.After(last):
			due = append(due, ar.item)
			if !slices.Contains(touched, ar.work) {
				touched = append(touched, ar.work)
			}
		}
	}
	if !last.Before(today) {
		return next, nil
	}
	return next, a.release(ctx, last, today, touched, due)
}

// release hands items to the hook before it publishes works.changed, so a
// failing hook doesn't wake the timer through its own event.
func (a *Availability) release(ctx context.Context, last, today time.Time, ws []works.WorkID, items []works.ItemID) error {
	if len(items) > 0 {
		if err := a.cfg.Released(ctx, items); err != nil {
			return fmt.Errorf("hand %d released Items to the hook: %w", len(items), err)
		}
	}
	for _, w := range ws {
		a.cfg.Changed.Publish(ctx, works.WorkChanged{WorkID: w.String()})
	}
	err := a.cfg.Local.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return announced.SetDay(ctx, tx, today)
	})
	if err != nil {
		return fmt.Errorf("record the day announced: %w", err)
	}
	a.cfg.Log.Info("announced released Items",
		"after", last.Format(time.DateOnly), "through", today.Format(time.DateOnly), "items", len(items), "works", len(ws))
	return nil
}

// arrivals is every monitored Item whose availability date its
// Content Type knows, in the library's Work order.
func (a *Availability) arrivals(ctx context.Context) ([]arrival, error) {
	var out []arrival
	err := a.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		all, err := works.Summaries(ctx, r)
		if err != nil {
			return err
		}
		ids := make([]works.WorkID, len(all))
		for i, w := range all {
			ids[i] = w.ID
		}
		items, err := works.ItemsOf(ctx, r, ids)
		if err != nil {
			return err
		}
		timings, err := a.timings(ctx, r, all, items)
		if err != nil {
			return err
		}
		for _, w := range all {
			for _, it := range items[w.ID] {
				if from, known := timings[it.ID].From(); known && it.Monitored {
					out = append(out, arrival{work: w.ID, item: it.ID, from: from})
				}
			}
		}
		return nil
	})
	return out, err
}

func (a *Availability) timings(ctx context.Context, r storage.Reader[storage.Library], all []works.Summary, items map[works.WorkID][]works.Item) (map[works.ItemID]targets.Timing, error) {
	watched := map[works.TypeID][]works.WorkID{}
	for _, w := range all {
		if slices.ContainsFunc(items[w.ID], monitored) {
			watched[w.ContentType] = append(watched[w.ContentType], w.ID)
		}
	}
	out := map[works.ItemID]targets.Timing{}
	for typ, ws := range watched {
		t, err := a.cfg.Targets.TimingsIn(ctx, r, typ, ws)
		if err != nil {
			return nil, fmt.Errorf("read the %s timings: %w", typ, err)
		}
		maps.Copy(out, t)
	}
	return out, nil
}

func monitored(it works.Item) bool { return it.Monitored }
