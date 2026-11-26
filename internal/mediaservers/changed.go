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
	"github.com/jatassi/slipstream/internal/storing"
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
// tx ends, so it always reads these rows. A Move's change goes as a carry
// to each server that shows both roots, as an added to one that shows
// only the destination and as a removed to one that shows only the
// source. Each server's path index entry at the change's path goes stale.
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
		folds, err := showing.folds(ctx, c, now)
		if err != nil {
			return err
		}
		if len(folds) == 0 {
			m.cfg.Log.Debug("no Media Server shows this root", "root", c.Root.String(), "path", c.Path.String())
		}
		for _, in := range folds {
			if err := fold(ctx, q, in); err != nil {
				return err
			}
			if !slices.Contains(woken, in.Server) {
				woken = append(woken, in.Server)
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

// folds are c as a new due row on each server that shows one of its roots.
func (s showings) folds(ctx context.Context, c files.PathChange, now time.Time) ([]queue.Change, error) {
	to, err := s.servers(ctx, c.Root)
	if err != nil || c.FromRoot.IsZero() {
		return changesOn(to, c, now), err
	}
	from, err := s.servers(ctx, c.FromRoot)
	if err != nil {
		return nil, err
	}
	var out []queue.Change
	for _, id := range to {
		in := incoming(c, id, now)
		if !slices.Contains(from, id) {
			in.Kind, in.FromRoot, in.From = link.ChangeAdded, roots.ID{}, storing.Rel{}
		}
		out = append(out, in)
	}
	for _, id := range from {
		if !slices.Contains(to, id) {
			in := incoming(c, id, now)
			in.Kind, in.Root, in.Path, in.FromRoot, in.From = link.ChangeRemoved, c.FromRoot, c.From, roots.ID{}, storing.Rel{}
			out = append(out, in)
		}
	}
	return out, nil
}

func changesOn(servers []link.ServerID, c files.PathChange, now time.Time) []queue.Change {
	out := make([]queue.Change, len(servers))
	for i, id := range servers {
		out[i] = incoming(c, id, now)
	}
	return out
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
		FromRoot: c.FromRoot, Path: c.Path, From: c.From, File: c.File, ReplacedBy: c.ReplacedBy, Upgrade: c.Upgrade,
		State: queue.Due, Step: link.StepNone, NextAt: now, Created: now,
	}
}

func fold(ctx context.Context, q *db.Queries, in queue.Change) error {
	open, err := openFor(ctx, q, in)
	if err != nil {
		return err
	}
	if err := apply(ctx, q, queue.Fold(open, in)); err != nil {
		return err
	}
	return relook(ctx, q, in)
}

// relook marks stale in's server's path index entry at in's path, whose
// library file just changed, so the server's next pass looks the path up
// and learn files the open plays there.
func relook(ctx context.Context, q *db.Queries, in queue.Change) error {
	folder, err := q.GetShownFolder(ctx, db.GetShownFolderParams{ServerID: in.Server, RootID: in.Root})
	if err != nil {
		return err
	}
	p, err := link.ParsePath(*folder)
	if err != nil {
		return err
	}
	return q.MarkPathStale(ctx, db.MarkPathStaleParams{ServerID: in.Server, PathKey: p.Join(in.Path).Key()})
}
