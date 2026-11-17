package notifications

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/notifications/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// Queue is history.Outbox.Queue: a pending row for each entry that may be
// sent, in the Write that records it. A quiet entry gets none. A New
// Health item waits for Health wait, and a Worse of a key still waiting
// stays on the earlier entry, keeping its due time, and renders from the
// newer one. A Health item cleared is sent at once for the keys some
// target was told or has a due message carrying; a key still waiting is
// taken back and sends nothing. It does row work only.
func (s *Service) Queue(ctx context.Context, tx *storage.Tx[file], c history.Captured) error {
	if c.Quiet {
		return nil
	}
	q := db.New(tx)
	var (
		keys []string
		due  = c.At
		err  error
	)
	switch c.Kind {
	case history.NewItemsRow.Kind:
		settings, err := q.GetSettings(ctx)
		if err != nil {
			return err
		}
		due = c.At.Add(HealthWait.Value(settings.HealthWaitSeconds))
		if keys, err = raise(ctx, q, c); err != nil {
			return err
		}
	case history.ClearedRow.Kind:
		if keys, err = clears(ctx, q, c); err != nil {
			return err
		}
	default:
		return s.pend(ctx, q, c, due, nil)
	}
	if len(keys) == 0 {
		return nil
	}
	return s.pend(ctx, q, c, due, keys)
}

func (s *Service) pend(ctx context.Context, q *db.Queries, c history.Captured, due time.Time, keys []string) error {
	var items *string
	if keys != nil {
		items = jsonText(keys)
	}
	err := q.InsertPending(ctx, db.InsertPendingParams{
		EventID: c.ID, Kind: c.Kind, At: storage.FromTime(c.At), DueAt: storage.FromTime(due), Items: items,
	})
	if err != nil {
		return err
	}
	s.ring()
	return nil
}

// raise is the keys of a New Health item that wait on its own pending row.
// A key already waiting on an earlier raise stays there and renders from
// this entry.
func raise(ctx context.Context, q *db.Queries, c history.Captured) ([]string, error) {
	var fresh []string
	for _, key := range c.Keys {
		w, ok, err := waiting(ctx, q, key)
		if err != nil {
			return nil, err
		}
		if !ok {
			fresh = append(fresh, key)
			continue
		}
		if err := q.UpsertWorse(ctx, db.UpsertWorseParams{EventID: w.EventID, Key: key, WorseID: c.ID}); err != nil {
			return nil, err
		}
	}
	return fresh, nil
}

// clears is the keys of a Health item cleared that are sent. A key still
// waiting is taken back from its raise, and the raise goes when no key is
// left.
func clears(ctx context.Context, q *db.Queries, c history.Captured) ([]string, error) {
	var carried []string
	for _, key := range c.Keys {
		w, ok, err := waiting(ctx, q, key)
		if err != nil {
			return nil, err
		}
		if ok {
			if err := unwait(ctx, q, w, key); err != nil {
				return nil, err
			}
			continue
		}
		reach, err := reaches(ctx, q, key)
		if err != nil {
			return nil, err
		}
		if reach {
			carried = append(carried, key)
		}
	}
	return carried, nil
}

// waiting is the pending raise key waits on.
func waiting(ctx context.Context, q *db.Queries, key string) (db.WaitingRaiseRow, bool, error) {
	w, err := q.WaitingRaise(ctx, key)
	if errors.Is(err, storage.ErrNoRows) {
		return w, false, nil
	}
	return w, err == nil, err
}

func unwait(ctx context.Context, q *db.Queries, w db.WaitingRaiseRow, key string) error {
	if err := q.DeleteWorse(ctx, db.DeleteWorseParams{EventID: w.EventID, Key: key}); err != nil {
		return err
	}
	keys, err := keysFrom(w.Items)
	if err != nil {
		return err
	}
	keys = slices.DeleteFunc(keys, func(k string) bool { return k == key })
	if len(keys) == 0 {
		return q.DeletePending(ctx, w.EventID)
	}
	return q.SetPendingItems(ctx, db.SetPendingItemsParams{EventID: w.EventID, Items: jsonText(keys)})
}

// reaches is whether some target was told key or has a due message
// carrying it.
func reaches(ctx context.Context, q *db.Queries, key string) (bool, error) {
	told, err := q.ListToldFor(ctx, []string{key})
	if err != nil || len(told) > 0 {
		return len(told) > 0, err
	}
	carrying, err := q.ConnectionsCarrying(ctx, key)
	return len(carrying) > 0, err
}

// Owed is history.Outbox.Owed: the ids with a pending row or a due
// delivery, and the newer entries such a message renders a key from.
func (s *Service) Owed(ctx context.Context, r storage.Reader[file], ids []history.EventID) (map[history.EventID]bool, error) {
	owed, err := db.New(r).ListOwed(ctx, db.ListOwedParams{Ids: ids, DueIds: ids, WorseIds: ids})
	if err != nil {
		return nil, err
	}
	out := make(map[history.EventID]bool, len(owed))
	for _, id := range owed {
		out[id] = true
	}
	return out, nil
}

// Told is history.Outbox.Told: the words a target was last told for key.
func (s *Service) Told(ctx context.Context, r storage.Reader[file], key string) (history.Told, bool, error) {
	told, err := db.New(r).ListToldFor(ctx, []string{key})
	if err != nil || len(told) == 0 {
		return history.Told{}, false, err
	}
	return history.Told{Subject: told[0].Subject, Since: told[0].SinceAt.Time()}, true, nil
}

// head is what notifications keeps of an entry: its id, kind and time.
type head struct {
	ID   EventID
	Kind entry.Kind
	At   time.Time
}

// eventsOf reads each head's subject and, for a Health entry, its items,
// each key from the newer entry a Worse inside the wait named.
func eventsOf(ctx context.Context, r storage.Reader[file], heads []head) (map[EventID]event, error) {
	ids := make([]EventID, len(heads))
	var healthIDs []EventID
	for i, h := range heads {
		ids[i] = h.ID
		if history.IsHealth(h.Kind) {
			healthIDs = append(healthIDs, h.ID)
		}
	}
	stored, err := history.Events(ctx, r, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[EventID]event, len(heads))
	for _, h := range heads {
		out[h.ID] = event{ID: h.ID, Kind: h.Kind, At: h.At, Subject: stored[h.ID].Subject, Work: stored[h.ID].Work}
	}
	if len(healthIDs) == 0 {
		return out, nil
	}
	worse, err := db.New(r).ListWorseOf(ctx, healthIDs)
	if err != nil {
		return nil, err
	}
	from := map[EventID]map[string]EventID{}
	read := slices.Clone(healthIDs)
	for _, w := range worse {
		if from[w.EventID] == nil {
			from[w.EventID] = map[string]EventID{}
		}
		from[w.EventID][w.Key] = w.WorseID
		read = append(read, w.WorseID)
	}
	items, err := history.Items(ctx, r, read)
	if err != nil {
		return nil, err
	}
	for _, id := range healthIDs {
		e := out[id]
		for _, it := range items[id] {
			if newer, ok := from[id][it.Key]; ok {
				it = items[newer][slices.IndexFunc(items[newer], func(n history.Item) bool { return n.Key == it.Key })]
			}
			e.Items = append(e.Items, it)
		}
		out[id] = e
	}
	return out, nil
}
