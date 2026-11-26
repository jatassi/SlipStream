package mediaservers

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/mediaservers/playback"
	"github.com/jatassi/slipstream/internal/storage"
)

// refile gives an open play with no snapshot its file and snapshot, when
// the path index names a file, and reports whether it changed the row. It
// asks the sender to look up a path the index has no row for. A snapshot
// it can't take leaves the play filed without one, so only the database or
// the context fails it.
func (m *MediaServers) refile(ctx context.Context, tx *storage.Tx[storage.Library], server link.ServerID, id link.PlayID) (bool, error) {
	q := db.New(tx)
	p, err := q.GetOpenPlay(ctx, db.GetOpenPlayParams{ServerID: server, ID: id})
	if errors.Is(err, storage.ErrNoRows) || err == nil && p.Snapped {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	file := p.FileID
	if file.IsZero() {
		file, err = q.FileAtPath(ctx, db.FileAtPathParams{ServerID: server, PathKey: p.PathKey})
		switch {
		case errors.Is(err, storage.ErrNoRows):
			path, err := link.ParsePath(p.ServerPath)
			if err != nil {
				return false, err
			}
			m.lookups.ask(server, []link.ServerPath{path}, time.Now())
			m.senders.wake(server)
			return false, nil
		case err != nil:
			return false, err
		case file.IsZero():
			return false, nil
		}
	}
	snap, err := m.snapshotOf(ctx, tx, file, p.AudioTrack, p.SubtitleTrack)
	if err != nil {
		if storage.Failed(err) {
			return false, err
		}
		m.cfg.Log.Warn("take the snapshot of a play", "server", server.String(), "play", id.String(), "err", err.Error())
		if !p.FileID.IsZero() {
			return false, nil
		}
		return true, q.FilePlay(ctx, db.FilePlayParams{ServerID: server, ID: id, FileID: file})
	}
	return true, q.SetSnapshot(ctx, db.SetSnapshotParams{ServerID: server, ID: id, FileID: file, Snapshot: &snap})
}

func (m *MediaServers) snapshotOf(ctx context.Context, r storage.Reader[storage.Library], file files.ID, audio, subtitle *link.Track) (playback.Snapshot, error) {
	f, err := m.cfg.Played(ctx, r, file)
	if err != nil {
		return playback.Snapshot{}, err
	}
	return playback.Snapshot{
		Work: f.Work, Item: f.Item, Label: f.Label, Targets: f.Met,
		Facts: playback.FactsOf(f.Media, audio, subtitle),
	}, nil
}
