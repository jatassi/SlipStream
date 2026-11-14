package mediaservers

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/mediaservers/queue"
	"github.com/jatassi/slipstream/internal/storage"
)

// changesPage is how many changes one page of GET changes lists.
const changesPage = 50

// Changes lists the server's changes newest first, the page before the
// change before names, the newest for a zero before.
func (s service) Changes(ctx context.Context, id link.ServerID, before link.ChangeID) (Changes, error) {
	var rows []db.MediaserversChange
	err := s.m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		q := db.New(r)
		if _, err := q.GetServer(ctx, id); err != nil {
			return err
		}
		var err error
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
		out.Changes = append(out.Changes, changeView(c))
	}
	return out, nil
}

func changeView(c queue.Change) ChangeView {
	v := ChangeView{
		ID: c.ID.String(), Kind: string(c.Kind), Path: c.Path.String(), State: string(c.State), Step: string(c.Step),
		Result: string(c.Result), Lost: make([]string, len(c.Lost)), CreatedAt: c.Created, Attempts: c.Attempts,
		Words: c.Words(),
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

// Housekeeping is the step that prunes the servers' old history.
func (m *MediaServers) Housekeeping() housekeeping.Step {
	return housekeeping.Step{Key: "media-server-history", Label: "Media Server history", Run: m.pruneSent}
}

// pruneSent deletes sent changes older than Timings.SentAge.
func (m *MediaServers) pruneSent(ctx context.Context) error {
	return m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		n, err := db.New(tx).PruneSent(ctx, storage.FromTime(time.Now().Add(-m.cfg.Timings.SentAge)))
		if err == nil && n > 0 {
			m.cfg.Log.Info("pruned sent Media Server changes", "changes", n)
		}
		return err
	})
}
