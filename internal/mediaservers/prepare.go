package mediaservers

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaserverlink"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/mediaservers/queue"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

// missed is what a rename's change loses when Prepare couldn't snapshot
// its item, until a late snapshot at send finds the item still there.
var missed = []link.Loss{link.LostWatchState, link.LostCollections}

// prep is one server's part of a Prepare call: the Job that snapshots
// its renames' items and the prepared rows the snapshots go on. port is
// nil while the server is held.
type prep struct {
	port mediaserverlink.Port
	job  link.Job
	rows []queue.Change
}

// Prepare snapshots, on every server that shows its root, the item of
// each rename whose bytes still sit at its origin, and writes a prepared
// row for it, all within Timings.RenameSnapshot. A server that didn't
// answer in time gets its row at step none with the watch data lost, so
// the next pass doesn't wait for it again and the sender tries the
// snapshot once more at send. A rename that already has an unsent row at
// its origin, or of its file, gets none: that row's send snapshots it.
// Prepared rows of files no pending rename names are dropped.
func (m *MediaServers) Prepare(ctx context.Context, rs []files.Rename) {
	ctx, cancel := context.WithTimeout(ctx, m.cfg.Timings.RenameSnapshot)
	defer cancel()
	var preps []*prep
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		preps, err = m.preps(ctx, db.New(r), rs, time.Now())
		return err
	})
	if err != nil && ctx.Err() == nil {
		m.cfg.Log.Error("read the Media Servers that show a rename", "err", err.Error())
	}
	var wg sync.WaitGroup
	for _, p := range preps {
		wg.Go(func() { m.snapshot(ctx, p) })
	}
	wg.Wait()
	var told []link.ServerID
	err = m.cfg.Library.Write(context.WithoutCancel(ctx), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		told, err = writePrepared(ctx, db.New(tx), rs, preps)
		return err
	})
	if err != nil {
		m.cfg.Log.Error("write the Media Server changes prepared for renames", "err", err.Error())
		return
	}
	m.senders.told(told...)
}

// snapshot takes p's snapshots in one step of a Carry Job. A failed or
// held step leaves every row missed.
func (m *MediaServers) snapshot(ctx context.Context, p *prep) {
	if p.port == nil {
		p.miss()
		return
	}
	call, cancel := context.WithTimeout(ctx, m.cfg.Timings.Call)
	defer cancel()
	out, err := p.port.Advance(call, p.job)
	w, ok := out.Job.Work.(link.Carry)
	if err != nil || !ok || out.Job.Step != link.StepSnapshotted {
		if err != nil {
			m.cfg.Log.Warn("a Media Server's snapshot before a rename failed, and the sender tries it again at send",
				"server", p.job.Server.ID.String(), "err", err.Error())
		}
		p.miss()
		return
	}
	for i, c := range p.rows {
		p.rows[i].Step, p.rows[i].StepAt = link.StepSnapshotted, out.Job.At
		if j := slices.IndexFunc(w.Changes, func(ch link.JobChange) bool { return ch.ID == c.ID }); j >= 0 {
			p.rows[i].Snapshot = w.Changes[j].Snapshot
		}
	}
}

func (p *prep) miss() {
	for i := range p.rows {
		p.rows[i].Lost = missed
	}
}

// preps is one prep per server that has a leaving rename to snapshot.
func (m *MediaServers) preps(ctx context.Context, q *db.Queries, rs []files.Rename, now time.Time) ([]*prep, error) {
	need, err := needed(ctx, q, rs)
	if err != nil {
		return nil, err
	}
	var out []*prep
	for id, renames := range need {
		p, err := m.prepOf(ctx, q, id, renames, now)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (m *MediaServers) prepOf(ctx context.Context, q *db.Queries, id link.ServerID, rs []files.Rename, now time.Time) (*prep, error) {
	row, err := q.GetServer(ctx, id)
	if err != nil {
		return nil, err
	}
	k, _ := m.cfg.Kinds.Of(row.Kind)
	p := &prep{job: link.Job{
		Server: link.ServerRef{ID: row.ID, Kind: row.Kind, Identity: row.Identity, SettingsVersion: row.SettingsVersion},
		Step:   link.StepNone,
	}}
	if v := routeOf(m.up(row.ReachingNodeID), row.Fallback); v != viaHold {
		if p.port, err = m.port(ctx, q, row, k, v); err != nil {
			return nil, err
		}
	}
	mappings, err := q.ListMappings(ctx, id)
	if err != nil {
		return nil, err
	}
	var carry link.Carry
	for _, r := range rs {
		i := slices.IndexFunc(mappings, func(mp db.MediaserversMapping) bool { return mp.RootID == r.Root })
		folder, err := link.ParsePath(*mappings[i].ServerFolder)
		if err != nil {
			return nil, err
		}
		c := preparedOf(r, id, now)
		p.rows = append(p.rows, c)
		carry.Changes = append(carry.Changes, link.JobChange{
			ID: c.ID, Kind: c.Kind, Library: *mappings[i].LibraryKey, Path: folder.Join(r.To), From: folder.Join(r.From),
		})
	}
	p.job.Work = carry
	return p, nil
}

// preparedOf is r's prepared row on server, at step none: a moved change,
// or for a doomed file whose replacer lands under another name, the
// upgrade's replaced carry, which its removal makes due.
func preparedOf(r files.Rename, server link.ServerID, now time.Time) queue.Change {
	c := queue.Change{
		ID: link.ChangeID{UUID: storage.NewUUID()}, Server: server, Kind: link.ChangeMoved, Root: r.Root,
		Path: r.To, From: r.From, File: r.File, ReplacedBy: r.ReplacedBy,
		State: queue.Prepared, Step: link.StepNone, NextAt: now, Created: now,
	}
	if !r.ReplacedBy.IsZero() {
		c.Kind = link.ChangeReplaced
	}
	return c
}

// writePrepared drops the prepared rows no rename of rs names, and writes
// each row of preps that its server still needs, telling those servers.
func writePrepared(ctx context.Context, q *db.Queries, rs []files.Rename, preps []*prep) ([]link.ServerID, error) {
	pending := make([]files.ID, len(rs))
	for i, r := range rs {
		pending[i] = r.File
	}
	var err error
	if len(pending) == 0 {
		err = q.DeleteAllPrepared(ctx)
	} else {
		err = q.DeleteStalePrepared(ctx, pending)
	}
	if err != nil {
		return nil, err
	}
	need, err := needed(ctx, q, rs)
	if err != nil {
		return nil, err
	}
	var told []link.ServerID
	for _, p := range preps {
		for _, c := range p.rows {
			if !slices.ContainsFunc(need[c.Server], func(r files.Rename) bool { return r.File == c.File && r.From == c.From }) {
				continue
			}
			if err := apply(ctx, q, []queue.Op{{Do: queue.Insert, Change: c}}); err != nil {
				return nil, err
			}
			if !slices.Contains(told, c.Server) {
				told = append(told, c.Server)
			}
		}
	}
	return told, nil
}

// needed is, per server that shows its root, each leaving rename of rs
// that no unsent row of that server meets at its origin or by its file.
func needed(ctx context.Context, q *db.Queries, rs []files.Rename) (map[link.ServerID][]files.Rename, error) {
	byRoot := map[roots.ID][]files.Rename{}
	for _, r := range rs {
		if r.Leaving {
			byRoot[r.Root] = append(byRoot[r.Root], r)
		}
	}
	out := map[link.ServerID][]files.Rename{}
	for root, leaving := range byRoot {
		if err := neededIn(ctx, q, root, leaving, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func neededIn(ctx context.Context, q *db.Queries, root roots.ID, leaving []files.Rename, out map[link.ServerID][]files.Rename) error {
	servers, err := q.ListShowing(ctx, root)
	if err != nil || len(servers) == 0 {
		return err
	}
	p := db.ListUnsentParams{RootID: root}
	for _, r := range leaving {
		from := r.From.String()
		p.Paths, p.Origins, p.Files = append(p.Paths, from), append(p.Origins, &from), append(p.Files, r.File)
	}
	unsent, err := q.ListUnsent(ctx, p)
	if err != nil {
		return err
	}
	for _, r := range leaving {
		for _, s := range servers {
			if !slices.ContainsFunc(unsent, meets(s, r)) {
				out[s] = append(out[s], r)
			}
		}
	}
	return nil
}

func meets(server link.ServerID, r files.Rename) func(db.MediaserversChange) bool {
	from := r.From.String()
	return func(u db.MediaserversChange) bool {
		return u.ServerID == server && (u.FileID == r.File || u.Path == from || u.FromPath != nil && *u.FromPath == from)
	}
}
