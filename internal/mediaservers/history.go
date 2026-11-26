package mediaservers

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/mediaservers/queue"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

// changesPage is how many changes one page of GET changes lists.
const changesPage = 50

// Changes lists the server's changes newest first, the page before the
// change before names, the newest for a zero before.
func (s service) Changes(ctx context.Context, id link.ServerID, before link.ChangeID) (Changes, error) {
	var rows []db.MediaserversChange
	var words map[roots.ID]string
	err := s.m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		q := db.New(r)
		if _, err := q.GetServer(ctx, id); err != nil {
			return err
		}
		var err error
		if words, err = s.m.cfg.RootWords(ctx, r); err != nil {
			return err
		}
		if before.IsZero() {
			rows, err = q.ListChanges(ctx, db.ListChangesParams{ServerID: id, N: changesPage + 1})
		} else {
			rows, err = q.ListChangesBefore(ctx, db.ListChangesBeforeParams{ServerID: id, Before: before, N: changesPage + 1})
		}
		return err
	})
	if errors.Is(err, storage.ErrNoRows) {
		return Changes{}, ErrNotFound
	}
	if err != nil {
		return Changes{}, err
	}
	out := Changes{Changes: []ChangeView{}}
	if len(rows) > changesPage {
		rows = rows[:changesPage]
		out.Next = rows[changesPage-1].ID.String()
	}
	cs, err := changesOf(rows)
	if err != nil {
		return Changes{}, err
	}
	for _, c := range cs {
		out.Changes = append(out.Changes, changeView(c, words[c.Root]))
	}
	return out, nil
}

func changeView(c queue.Change, root string) ChangeView {
	v := ChangeView{
		ID: c.ID.String(), Kind: string(c.Kind), Path: c.Path.String(), State: string(c.State), Step: string(c.Step),
		Result: string(c.Result), Lost: make([]string, len(c.Lost)), CreatedAt: c.Created, Attempts: c.Attempts,
		Words: c.Words(root),
	}
	if c.Carry() {
		v.FromPath = c.From.String()
	}
	for i, l := range c.Lost {
		v.Lost[i] = string(l)
	}
	if !c.Sent.IsZero() {
		sent := c.Sent
		v.SentAt = &sent
	}
	return v
}

// Refresh queues a whole library refresh of each shown, confirmed
// mapping's library.
func (s service) Refresh(ctx context.Context, id link.ServerID) error {
	err := s.m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		if _, err := q.GetServer(ctx, id); err != nil {
			return err
		}
		mappings, err := q.ListMappings(ctx, id)
		if err != nil {
			return err
		}
		now := storage.Now()
		for _, mp := range mappings {
			if !mp.Shown || mp.ConfirmedAt.IsZero() {
				continue
			}
			if err := q.QueueLibrary(ctx, db.QueueLibraryParams{ID: link.ChangeID{UUID: storage.NewUUID()}, ServerID: id, RootID: mp.RootID, At: now}); err != nil {
				return err
			}
		}
		return nil
	})
	if errors.Is(err, storage.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	s.m.changed(ctx, id)
	return nil
}

// Housekeeping is the step that prunes the servers' old history: sent
// changes older than Timings.SentAge and plays that ended longer ago than
// Keep play history for.
func (m *MediaServers) Housekeeping() housekeeping.Step {
	return housekeeping.Plain("media-server-history", "Media Server history", tasks.Runs, m.pruneHistory)
}

// pruneHistory moves plays_since to the cutoff before any batch deletes,
// so no window ever claims days whose plays are gone.
func (m *MediaServers) pruneHistory(ctx context.Context) error {
	now := time.Now()
	var cutoff storage.Time
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		n, err := q.PruneSent(ctx, storage.FromTime(now.Add(-m.cfg.Timings.SentAge)))
		if err != nil {
			return err
		}
		if n > 0 {
			m.cfg.Log.Info("pruned sent Media Server changes", "changes", n)
		}
		row, err := q.GetSettings(ctx)
		if err != nil {
			return err
		}
		cutoff = storage.FromTime(now.Add(-KeepPlays.Value(row.KeepPlaysSeconds)))
		return q.MovePlaysSince(ctx, cutoff)
	})
	if err != nil {
		return err
	}
	t, err := housekeeping.Drain(ctx, m.cfg.Batch, func(ctx context.Context, _ struct{}, limit int) (housekeeping.Swept[struct{}], error) {
		var n int64
		err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			var err error
			n, err = db.New(tx).PrunePlays(ctx, db.PrunePlaysParams{Before: cutoff, N: int64(limit)})
			return err
		})
		return housekeeping.Swept[struct{}]{Seen: int(n), Deleted: int(n)}, err
	})
	if t.Deleted > 0 {
		m.cfg.Log.Info("pruned Media Server plays", "plays", t.Deleted)
	}
	return err
}
