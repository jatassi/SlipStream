package mediaservers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// startPlay records a play the reaching Node started, with its file and
// snapshot when the path index names one. Plays upsert by the id that
// Node minted, so applying a report twice changes nothing, and a report
// for a play the Controller already ended, or a start for a play it
// already has, changes nothing either.
func (m *MediaServers) startPlay(ctx context.Context, tx *storage.Tx[storage.Library], node role.NodeID, server link.ServerID, cur db.GetContactRow, s *msv1.PlayStarted) error {
	q := db.New(tx)
	set, err := q.GetSettings(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	w := window(now, KeepPlays.Value(set.KeepPlaysSeconds), cur.PlaysSince.Time())
	p, err := m.caps().started(s.GetSession(), now, w.Since)
	if err != nil {
		return err
	}
	if err := q.StartPlay(ctx, p.startParams(server)); err != nil {
		return err
	}
	m.played.add(byNode(node), server)
	_, err = m.refile(ctx, tx, server, p.id)
	return err
}

func (m *MediaServers) changePlay(ctx context.Context, tx *storage.Tx[storage.Library], node role.NodeID, server link.ServerID, c *msv1.PlayChanged) error {
	p, err := m.caps().changed(c, time.Now())
	if err != nil {
		return err
	}
	q := db.New(tx)
	seen, open, err := laterSeen(ctx, q, server, p.id, p.seen)
	if err != nil || !open {
		return err
	}
	err = q.ChangePlay(ctx, db.ChangePlayParams{
		ServerID: server, ID: p.id, Decision: p.streams.Decision(), Video: p.streams.Video, Audio: p.streams.Audio,
		Subtitles: p.streams.Subtitles, Hardware: p.hardware, Reasons: reasonsJSON(p.reasons), Received: p.received,
		AudioTrack: p.audio, SubtitleTrack: p.subtitle, LastSeenAt: seen,
	})
	if err != nil {
		return err
	}
	m.played.add(byNode(node), server)
	_, err = m.refile(ctx, tx, server, p.id)
	return err
}

// endPlay ends an open play at its last look. The end report is its last
// try at a snapshot, because an ended row stays as it is.
func (m *MediaServers) endPlay(ctx context.Context, tx *storage.Tx[storage.Library], node role.NodeID, server link.ServerID, e *msv1.PlayEnded) error {
	id, err := playIDOf(e.GetPlayId())
	if err != nil {
		return err
	}
	q := db.New(tx)
	seen, open, err := laterSeen(ctx, q, server, id, m.caps().at(timeOf(e.GetLastSeenUnixMs()), time.Now()))
	if err != nil || !open {
		return err
	}
	if _, err := m.refile(ctx, tx, server, id); err != nil {
		return err
	}
	m.played.add(byNode(node), server)
	return q.EndPlay(ctx, db.EndPlayParams{ServerID: server, ID: id, EndedAt: seen})
}

// watching ends every open play of server that w doesn't list, at its
// last look: the plays a holder that stopped left open. The plays it
// lists were seen at w's time.
func (m *MediaServers) watching(ctx context.Context, tx *storage.Tx[storage.Library], node role.NodeID, server link.ServerID, w *msv1.Watching) error {
	listed := map[link.PlayID]bool{}
	for _, b := range w.GetPlayIds() {
		id, err := playIDOf(b)
		if err != nil {
			return err
		}
		listed[id] = true
	}
	at := timeOf(w.GetAtUnixMs())
	if at.IsZero() {
		return errors.New("a Watching report has no time")
	}
	at = m.caps().at(at, time.Now())
	q := db.New(tx)
	open, err := q.ListOpenPlays(ctx, server)
	if err != nil {
		return err
	}
	for _, id := range open {
		if !listed[id] {
			if err := q.EndPlayAsSeen(ctx, db.EndPlayAsSeenParams{ServerID: server, ID: id}); err != nil {
				return err
			}
			m.played.add(byNode(node), server)
			continue
		}
		seen, _, err := laterSeen(ctx, q, server, id, at)
		if err != nil {
			return err
		}
		if err := q.SeePlay(ctx, db.SeePlayParams{ServerID: server, ID: id, LastSeenAt: seen}); err != nil {
			return err
		}
		filed, err := m.refile(ctx, tx, server, id)
		if err != nil {
			return err
		}
		if filed {
			m.played.add(byNode(node), server)
		}
	}
	return nil
}

// laterSeen is the later of the open play's last look and at, false when
// the play isn't open.
func laterSeen(ctx context.Context, q *db.Queries, server link.ServerID, id link.PlayID, at time.Time) (storage.Time, bool, error) {
	p, err := q.GetOpenPlay(ctx, db.GetOpenPlayParams{ServerID: server, ID: id})
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return storage.Time{}, false, nil
	case err != nil:
		return storage.Time{}, false, err
	}
	seen := p.LastSeenAt
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

func reasonsJSON(rs []string) string {
	b, _ := json.Marshal(rs) //nolint:errchkjson // a string slice always marshals
	return string(b)
}

func streamsOf(s *msv1.Streams) (link.Streams, error) {
	video, err := link.ParseStreamDecision(s.GetVideo())
	if err != nil {
		return link.Streams{}, fmt.Errorf("a play's video: %w", err)
	}
	audio, err := link.ParseStreamDecision(s.GetAudio())
	if err != nil {
		return link.Streams{}, fmt.Errorf("a play's audio: %w", err)
	}
	subtitles, err := link.ParseSubtitleDecision(s.GetSubtitles())
	if err != nil {
		return link.Streams{}, fmt.Errorf("a play's subtitles: %w", err)
	}
	return link.Streams{Video: video, Audio: audio, Subtitles: subtitles, ContainerChanged: s.GetContainerChanged()}, nil
}
