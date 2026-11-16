package notifications

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/notifications/internal/db"
	"github.com/jatassi/slipstream/internal/notifications/notice"
	"github.com/jatassi/slipstream/internal/storage"
)

// Transitions is Health's sink. It runs inside Health's Write and does row
// work only: a New Health item waits on an unreleased event, a Worse
// inside the wait replaces the waiting item, a clear inside the wait takes
// the item back, and a clear after the wait is recorded when some target
// was told the item or has a due message carrying it. One evaluation makes
// at most one event of each kind.
func (s *Service) Transitions(ctx context.Context, tx *storage.Tx[file], ts []health.Transition) error {
	q := db.New(tx)
	settings, err := q.GetSettings(ctx)
	if err != nil {
		return err
	}
	ev := evaluation{q: q, wait: HealthWait.Value(settings.HealthWaitSeconds)}
	for _, t := range ts {
		key := itemKey(t.Category, t.Key)
		switch t.Kind {
		case health.Raised, health.Worse:
			err = ev.raise(ctx, t, key)
		case health.Cleared:
			err = ev.clear(ctx, t, key)
		case health.Better:
		}
		if err != nil {
			return err
		}
	}
	s.ring()
	return nil
}

// evaluation makes one evaluation's events when it first needs them.
type evaluation struct {
	q                *db.Queries
	wait             time.Duration
	raised, cleared  EventID
}

func (ev *evaluation) raise(ctx context.Context, t health.Transition, key string) error {
	waiting, err := ev.q.WaitingItem(ctx, db.WaitingItemParams{Key: key, Kind: newItemsRow.Kind})
	switch {
	case err == nil:
		return ev.q.ReplaceItem(ctx, db.ReplaceItemParams{EventID: waiting.EventID, Key: key, Level: t.To.String(), Subject: t.Subject, Text: t.Text})
	case !errors.Is(err, storage.ErrNoRows):
		return err
	}
	if ev.raised == 0 {
		if ev.raised, err = ev.event(ctx, newItemsRow.Kind, t.At, t.At.Add(ev.wait)); err != nil {
			return err
		}
	}
	return ev.q.InsertItem(ctx, db.InsertItemParams{
		EventID: ev.raised, Key: key, Level: t.To.String(), Subject: t.Subject, Text: t.Text, SinceAt: storage.FromTime(t.Since),
	})
}

func (ev *evaluation) clear(ctx context.Context, t health.Transition, key string) error {
	waiting, err := ev.q.WaitingItem(ctx, db.WaitingItemParams{Key: key, Kind: newItemsRow.Kind})
	switch {
	case err == nil:
		return ev.unwait(ctx, waiting.EventID, key)
	case !errors.Is(err, storage.ErrNoRows):
		return err
	}
	words, reach, err := ev.clearWords(ctx, t, key)
	if err != nil || !reach {
		return err
	}
	if ev.cleared == 0 {
		if ev.cleared, err = ev.event(ctx, clearedRow.Kind, t.At, t.At); err != nil {
			return err
		}
	}
	return ev.q.InsertItem(ctx, db.InsertItemParams{
		EventID: ev.cleared, Key: key, Level: t.From.String(), Subject: words.Subject, Text: words.Text,
		SinceAt: storage.FromTime(words.Since), ClearedAt: storage.FromTime(t.At),
	})
}

// unwait takes a waiting item back, and its event when it was the last.
func (ev *evaluation) unwait(ctx context.Context, id EventID, key string) error {
	if err := ev.q.DeleteItem(ctx, db.DeleteItemParams{EventID: id, Key: key}); err != nil {
		return err
	}
	left, err := ev.q.CountItems(ctx, id)
	if err != nil || left > 0 {
		return err
	}
	return ev.q.DeleteEvent(ctx, id)
}

// clearWords is what a clear of key says, and whether any target gets it:
// a target was told the item, or has a due message carrying it. The words
// are the newest item's for key, else a told row's, which outlives the
// prune, else the transition's.
func (ev *evaluation) clearWords(ctx context.Context, t health.Transition, key string) (item, bool, error) {
	told, err := ev.q.ListToldFor(ctx, []string{key})
	if err != nil {
		return item{}, false, err
	}
	if len(told) == 0 {
		carrying, err := ev.q.ConnectionsCarrying(ctx, []string{key})
		if err != nil || len(carrying) == 0 {
			return item{}, false, err
		}
	}
	newest, err := ev.q.NewestItem(ctx, key)
	switch {
	case err == nil:
		return itemOf(newest), true, nil
	case !errors.Is(err, storage.ErrNoRows):
		return item{}, false, err
	case len(told) > 0:
		return item{Subject: told[0].Subject, Since: told[0].SinceAt.Time()}, true, nil
	}
	return item{Subject: t.Subject, Text: t.Text, Since: t.Since}, true, nil
}

func (ev *evaluation) event(ctx context.Context, kind notice.Kind, at, due time.Time) (EventID, error) {
	id, err := ev.q.InsertEvent(ctx, db.InsertEventParams{Kind: kind, At: storage.FromTime(at), DueAt: storage.FromTime(due), Subject: "{}"})
	if err != nil {
		return 0, fmt.Errorf("record a %s event: %w", kind, err)
	}
	return id, nil
}

// HealthAbout is the Notifications category: one item per connection that
// is on and isn't sending.
var HealthAbout = health.About{
	ID: "notifications", Label: "Notifications", Phrase: "notification connections",
	Events: []live.Event{ChangedEvent, nodes.ChangedEvent},
}

// HealthSource builds method values only, so a Node can bind it before the
// service runs.
func (s *Service) HealthSource() health.Source {
	return health.Source{Answer: s.answer}
}

// answer words each connection that is paused, or has failed or waited for
// at least Warn when a connection has failed for. Next is when the next
// one crosses that time.
func (s *Service) answer(ctx context.Context) (health.Answer, error) {
	var a health.Answer
	views, warn, err := s.statuses(ctx)
	if err != nil {
		return a, err
	}
	w := s.wording(time.Now(), 0)
	for _, v := range views {
		st := v.Status
		if (st.Shown == ShownFailing || st.Shown == ShownWaiting) && st.Level == health.None {
			if next := st.Since.Add(warn); a.Next.IsZero() || next.Before(a.Next) {
				a.Next = next
			}
		}
		if st.Level == health.None {
			continue
		}
		label, reason := s.label(v.Connection), w.reason(s.provider(v.Provider), st.Reason)
		it := health.Item{Key: health.KeyOf(v.ID.String()), Level: st.Level}
		it.Action = health.LinkAction("/settings/notifications", "Open Notifications")
		switch st.Shown {
		case ShownPaused:
			it.Subject = label + " is paused"
			it.Text = reason + " Edit the connection to fix it, then test it."
		case ShownWaiting:
			it.Subject = "Messages to " + label + " are waiting"
			it.Text = fmt.Sprintf("Messages have waited since %s. %s %s", s.clock(st.Since), reason, waitingWords(st.Waiting))
		default:
			it.Subject = "Messages to " + label + " are failing"
			it.Text = fmt.Sprintf("Messages have failed since %s. %s %s", s.clock(st.Since), reason, waitingWords(st.Waiting))
		}
		a.Items = append(a.Items, it)
	}
	return a, nil
}

func waitingWords(n int) string {
	if n == 1 {
		return "1 message is waiting."
	}
	return fmt.Sprintf("%d messages are waiting.", n)
}

func (s *Service) clock(t time.Time) string { return t.In(s.cfg.Zone).Format("15:04") }
