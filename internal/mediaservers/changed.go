package mediaservers

import (
	"context"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/mediaservers/queue"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

var _ files.MediaServers = (*MediaServers)(nil)

var changeKinds = map[files.PathChangeKind]link.ChangeKind{
	files.PathAdded:    link.ChangeAdded,
	files.PathReplaced: link.ChangeReplaced,
	files.PathRemoved:  link.ChangeRemoved,
	files.PathMoved:    link.ChangeMoved,
}

// Changed folds each change into the queue of every server whose mapping
// shows its root, in tx, the Write that records the bytes. It wakes those
// servers' senders before tx commits: a sender's next Write starts after
// tx ends, so it always reads these rows.
func (m *MediaServers) Changed(ctx context.Context, tx *storage.Tx[storage.Library], cs []files.PathChange) error {
	q := db.New(tx)
	showing := showings{q: q, of: map[roots.ID][]link.ServerID{}}
	var woken []link.ServerID
	now := time.Now()
	for _, c := range cs {
		if c.Kind == files.PathUnmoved {
			if err := q.DeletePrepared(ctx, c.File); err != nil {
				return err
			}
			continue
		}
		servers, err := showing.servers(ctx, c.Root)
		if err != nil {
			return err
		}
		if len(servers) == 0 {
			m.cfg.Log.Debug("no Media Server shows this root", "root", c.Root.String(), "path", c.Path.String())
		}
		for _, s := range servers {
			if err := fold(ctx, q, incoming(c, s, now)); err != nil {
				return err
			}
			if !slices.Contains(woken, s) {
				woken = append(woken, s)
			}
		}
	}
	m.senders.told(woken...)
	return nil
}

// showings are the servers whose mapping shows each root, read once.
type showings struct {
	q  *db.Queries
	of map[roots.ID][]link.ServerID
}

func (s showings) servers(ctx context.Context, root roots.ID) ([]link.ServerID, error) {
	if ids, ok := s.of[root]; ok {
		return ids, nil
	}
	ids, err := s.q.ListShowing(ctx, root)
	s.of[root] = ids
	return ids, err
}

// incoming is c as a new due row for server s.
func incoming(c files.PathChange, s link.ServerID, now time.Time) queue.Change {
	return queue.Change{
		ID: link.ChangeID{UUID: storage.NewUUID()}, Server: s, Kind: changeKinds[c.Kind], Root: c.Root,
		Path: c.Path, From: c.From, File: c.File, ReplacedBy: c.ReplacedBy, Upgrade: c.Upgrade,
		State: queue.Due, Step: link.StepNone, NextAt: now, Created: now,
	}
}

func fold(ctx context.Context, q *db.Queries, in queue.Change) error {
	open, err := openFor(ctx, q, in)
	if err != nil {
		return err
	}
	return apply(ctx, q, queue.Fold(open, in))
}

// Prepare drops the prepared rows of files no pending rename names: a plan
// called off or replaced since Prepare saw it.
func (m *MediaServers) Prepare(ctx context.Context, rs []files.Rename) {
	pending := make([]files.ID, len(rs))
	for i, r := range rs {
		pending[i] = r.File
	}
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if len(pending) == 0 {
			return db.New(tx).DeleteAllPrepared(ctx)
		}
		return db.New(tx).DeleteStalePrepared(ctx, pending)
	})
	if err != nil && ctx.Err() == nil {
		m.cfg.Log.Error("drop the prepared Media Server changes of renames called off", "err", err.Error())
	}
}
