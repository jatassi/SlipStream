package mediaservers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/storage"
)

// startPlay records a play the reaching Node started. Plays upsert by the
// id that Node minted, so applying a report twice changes nothing, and a
// report for a play the Controller already ended, or a start for a play
// it already has, changes nothing either.
func startPlay(ctx context.Context, q *db.Queries, server link.ServerID, m *msv1.PlayStarted) error {
	s := m.GetSession()
	id, err := playIDOf(s.GetPlayId())
	if err != nil {
		return err
	}
	path, err := link.ParsePath(s.GetServerPath())
	if err != nil {
		return fmt.Errorf("a started play's path: %w", err)
	}
	decision, err := link.ParseDecision(s.GetDecision())
	if err != nil {
		return err
	}
	started, seen := timeOf(s.GetStartedUnixMs()), timeOf(s.GetSeenUnixMs())
	if started.IsZero() || seen.Before(started) {
		return errors.New("a started play's times are out of order")
	}
	reasons, err := reasonsOf(s.GetReasons())
	if err != nil {
		return err
	}
	file, err := q.FileAtPath(ctx, db.FileAtPathParams{ServerID: server, PathKey: path.Key()})
	if err != nil && !errors.Is(err, storage.ErrNoRows) {
		return err
	}
	return q.StartPlay(ctx, db.StartPlayParams{
		ServerID: server, ID: id, SessionKey: s.GetSessionKey(), FileID: file, ServerPath: path.String(),
		UserName: s.GetUser(), Player: s.GetPlayer(), Product: s.GetProduct(), Decision: decision,
		Hardware: s.GetHardware(), Reasons: reasons,
		StartedAt: storage.FromTime(started), LastSeenAt: storage.FromTime(seen),
	})
}

func changePlay(ctx context.Context, q *db.Queries, server link.ServerID, m *msv1.PlayChanged) error {
	id, err := playIDOf(m.GetPlayId())
	if err != nil {
		return err
	}
	decision, err := link.ParseDecision(m.GetDecision())
	if err != nil {
		return err
	}
	reasons, err := reasonsOf(m.GetReasons())
	if err != nil {
		return err
	}
	seen, open, err := laterSeen(ctx, q, server, id, timeOf(m.GetSeenUnixMs()))
	if err != nil || !open {
		return err
	}
	return q.ChangePlay(ctx, db.ChangePlayParams{
		ServerID: server, ID: id, Decision: decision, Hardware: m.GetHardware(), Reasons: reasons, LastSeenAt: seen,
	})
}

func endPlay(ctx context.Context, q *db.Queries, server link.ServerID, m *msv1.PlayEnded) error {
	id, err := playIDOf(m.GetPlayId())
	if err != nil {
		return err
	}
	seen, open, err := laterSeen(ctx, q, server, id, timeOf(m.GetLastSeenUnixMs()))
	if err != nil || !open {
		return err
	}
	return q.EndPlay(ctx, db.EndPlayParams{ServerID: server, ID: id, EndedAt: seen})
}

// watching ends every open play of server that m doesn't list, at its
// last look: the plays a holder that stopped left open. The plays it
// lists were seen at m's time.
func watching(ctx context.Context, q *db.Queries, server link.ServerID, m *msv1.Watching) error {
	listed := map[link.PlayID]bool{}
	for _, b := range m.GetPlayIds() {
		id, err := playIDOf(b)
		if err != nil {
			return err
		}
		listed[id] = true
	}
	at := timeOf(m.GetAtUnixMs())
	if at.IsZero() {
		return errors.New("a Watching report has no time")
	}
	open, err := q.ListOpenPlays(ctx, server)
	if err != nil {
		return err
	}
	for _, id := range open {
		if !listed[id] {
			if err := q.EndPlayAsSeen(ctx, db.EndPlayAsSeenParams{ServerID: server, ID: id}); err != nil {
				return err
			}
			continue
		}
		seen, _, err := laterSeen(ctx, q, server, id, at)
		if err != nil {
			return err
		}
		if err := q.SeePlay(ctx, db.SeePlayParams{ServerID: server, ID: id, LastSeenAt: seen}); err != nil {
			return err
		}
	}
	return nil
}

// laterSeen is the later of the open play's last look and at, false when
// the play isn't open.
func laterSeen(ctx context.Context, q *db.Queries, server link.ServerID, id link.PlayID, at time.Time) (storage.Time, bool, error) {
	seen, err := q.GetOpenPlay(ctx, db.GetOpenPlayParams{ServerID: server, ID: id})
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return storage.Time{}, false, nil
	case err != nil:
		return storage.Time{}, false, err
	}
	if t := storage.FromTime(at); t.Compare(seen) > 0 {
		seen = t
	}
	return seen, true, nil
}

// hint marks stale the path index entries of the items a push said
// changed.
func hint(ctx context.Context, q *db.Queries, server link.ServerID, m *msv1.IndexHint) error {
	if len(m.GetItemKeys()) == 0 {
		return nil
	}
	return q.MarkItemsStale(ctx, db.MarkItemsStaleParams{ServerID: server, ItemKeys: m.GetItemKeys()})
}

func playIDOf(b []byte) (link.PlayID, error) {
	u, err := storage.UUIDFromBytes(b)
	if err != nil {
		return link.PlayID{}, fmt.Errorf("read a play id: %w", err)
	}
	return link.PlayID{UUID: u}, nil
}

func timeOf(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

func reasonsOf(rs []string) (string, error) {
	if rs == nil {
		rs = []string{}
	}
	b, err := json.Marshal(rs)
	return string(b), err
}

// Housekeeping is the step that deletes the plays that ended longer ago
// than Keep play history for.
func (m *MediaServers) Housekeeping() housekeeping.Step {
	return housekeeping.Step{Key: "media-server-plays", Label: "Media Server play history", Run: m.prunePlays}
}

func (m *MediaServers) prunePlays(ctx context.Context) error {
	return m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		row, err := q.GetSettings(ctx)
		if err != nil {
			return err
		}
		_, err = q.PrunePlays(ctx, storage.FromTime(time.Now().Add(-KeepPlays.Value(row.KeepPlaysSeconds))))
		return err
	})
}
