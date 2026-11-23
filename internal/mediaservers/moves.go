package mediaservers

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/mediaservers/queue"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

// MapState is how a server maps a root.
type MapState string

const (
	MapNone        MapState = "none"
	MapHidden      MapState = "hidden"
	MapUnconfirmed MapState = "unconfirmed"
	MapConfirmed   MapState = "confirmed"
)

// Cover is how one server maps one root.
type Cover struct {
	Server link.ServerID
	Kind   link.KindName
	Root   roots.ID
	State  MapState
	// Library is the library the root is in, "" while not shown.
	Library string
	// Fresh is a mapping never scanned.
	Fresh bool
}

// Coverage is, per server and each of rs, how the server maps it.
func (m *MediaServers) Coverage(ctx context.Context, r storage.Reader[storage.Library], rs []roots.ID) ([]Cover, error) {
	q := db.New(r)
	servers, err := q.ListServers(ctx)
	if err != nil {
		return nil, err
	}
	mappings, err := q.ListMappingsIn(ctx, rs)
	if err != nil {
		return nil, err
	}
	var out []Cover
	for _, s := range servers {
		for _, root := range rs {
			c := Cover{Server: s.ID, Kind: s.Kind, Root: root, State: MapNone}
			if i := slices.IndexFunc(mappings, func(mp db.MediaserversMapping) bool { return mp.ServerID == s.ID && mp.RootID == root }); i >= 0 {
				c.State, c.Fresh = stateOf(mappings[i]), mappings[i].ScannedAt.IsZero()
				if mappings[i].Shown {
					c.Library = *mappings[i].LibraryKey
				}
			}
			out = append(out, c)
		}
	}
	return out, nil
}

func stateOf(mp db.MediaserversMapping) MapState {
	switch {
	case !mp.Shown:
		return MapHidden
	case mp.ConfirmedAt.IsZero():
		return MapUnconfirmed
	default:
		return MapConfirmed
	}
}

// Playing is, for each of fs that someone plays, the servers playing it
// whose Reaching Node is up. A play with no file is matched by its path
// through the server's mappings.
func (m *MediaServers) Playing(ctx context.Context, r storage.Reader[storage.Library], fs []files.ID) (map[files.ID][]link.ServerID, error) {
	q := db.New(r)
	servers, err := q.ListServers(ctx)
	if err != nil {
		return nil, err
	}
	up := map[link.ServerID]bool{}
	for _, s := range servers {
		up[s.ID] = m.up(s.ReachingNodeID)
	}
	out := map[files.ID][]link.ServerID{}
	add := func(f files.ID, s link.ServerID) {
		if up[s] && !slices.Contains(out[f], s) {
			out[f] = append(out[f], s)
		}
	}
	filed, err := q.ListPlayingFiles(ctx, fs)
	if err != nil {
		return nil, err
	}
	for _, p := range filed {
		add(p.FileID, p.ServerID)
	}
	return out, m.playedByPath(ctx, r, q, up, fs, add)
}

// playedByPath adds each of fs that a server of up plays with no file id
// on the play, matched by its path.
func (m *MediaServers) playedByPath(ctx context.Context, r storage.Reader[storage.Library], q *db.Queries,
	up map[link.ServerID]bool, fs []files.ID, add func(files.ID, link.ServerID),
) error {
	at, err := unfiledPlays(ctx, q, up)
	if err != nil {
		return err
	}
	byRoot := map[roots.ID][]storing.Rel{}
	for a := range at {
		byRoot[a.Root] = append(byRoot[a.Root], a.Rel)
	}
	for root, rels := range byRoot {
		ids, err := m.cfg.FileIDs(ctx, r, root, rels)
		if err != nil {
			return err
		}
		for rel, id := range ids {
			if slices.Contains(fs, id) {
				for _, s := range at[queue.At{Root: root, Rel: rel}] {
					add(id, s)
				}
			}
		}
	}
	return nil
}

// unfiledPlays is the servers of up playing each root's Rel with no file
// id on the play.
func unfiledPlays(ctx context.Context, q *db.Queries, up map[link.ServerID]bool) (map[queue.At][]link.ServerID, error) {
	plays, err := q.ListUnfiledPlays(ctx)
	if err != nil || len(plays) == 0 {
		return nil, err
	}
	mappings, err := q.ListAllMappings(ctx)
	if err != nil {
		return nil, err
	}
	at := map[queue.At][]link.ServerID{}
	for _, p := range plays {
		if !up[p.ServerID] {
			continue
		}
		path, err := link.ParsePath(p.ServerPath)
		if err != nil {
			return nil, err
		}
		for _, mp := range mappings {
			if mp.ServerID != p.ServerID || !mp.Shown {
				continue
			}
			folder, err := link.ParsePath(*mp.ServerFolder)
			if err != nil {
				return nil, err
			}
			if rel, ok := path.Rel(folder); ok {
				a := queue.At{Root: mp.RootID, Rel: rel}
				at[a] = append(at[a], p.ServerID)
			}
		}
	}
	return at, nil
}

// snapshotsFirst is a kind that carries a Move from a snapshot taken
// before the copy: Jellyfin and Emby, which hold a Move's carry at
// StepSnapshotted.
func snapshotsFirst(k link.Kind) bool { return k.About().MovedHold == link.StepSnapshotted }

// PrepareMove snapshots each of fs on every server that snapshots first
// and shows both from and to, through from's mapping, within
// Timings.RenameSnapshot, and writes a Move's prepared row for each with
// root_id = to and from_root_id = from. A server that didn't answer in time
// gets no row, and a repeat replaces the row.
func (m *MediaServers) PrepareMove(ctx context.Context, fs []files.ID, from, to roots.ID) {
	ctx, cancel := context.WithTimeout(ctx, m.cfg.Timings.RenameSnapshot)
	defer cancel()
	var preps []*prep
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		preps, err = m.movePreps(ctx, r, fs, from, to, time.Now())
		return err
	})
	if err != nil && ctx.Err() == nil {
		m.cfg.Log.Error("read the Media Servers that show a Move", "err", err.Error())
	}
	var wg sync.WaitGroup
	for _, p := range preps {
		wg.Go(func() { m.snapshot(ctx, p) })
	}
	wg.Wait()
	var told []link.ServerID
	err = m.cfg.Library.Write(context.WithoutCancel(ctx), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		told, err = writeMovePrepared(ctx, db.New(tx), preps)
		return err
	})
	if err != nil {
		m.cfg.Log.Error("write the Media Server changes prepared for a Move", "err", err.Error())
		return
	}
	m.senders.told(told...)
}

// writeMovePrepared writes each snapshotted row of preps in place of the
// server's Move prepared row of its file, and answers those servers.
func writeMovePrepared(ctx context.Context, q *db.Queries, preps []*prep) ([]link.ServerID, error) {
	var told []link.ServerID
	for _, p := range preps {
		for _, c := range p.rows {
			if c.Step != link.StepSnapshotted {
				continue
			}
			if err := q.DeleteMovePreparedOf(ctx, db.DeleteMovePreparedOfParams{ServerID: c.Server, FileID: c.File}); err != nil {
				return nil, err
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

func (m *MediaServers) movePreps(ctx context.Context, r storage.Reader[storage.Library], fs []files.ID, from, to roots.ID, now time.Time) ([]*prep, error) {
	q := db.New(r)
	servers, err := m.showingKind(ctx, q, from, snapshotsFirst)
	if err != nil || len(servers) == 0 {
		return nil, err
	}
	rels, err := m.cfg.FilePlaces(ctx, r, from, fs)
	if err != nil {
		return nil, err
	}
	var out []*prep
	for _, row := range servers {
		k, _ := m.cfg.Kinds.Of(row.Kind)
		mappings, err := q.ListMappings(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		src, dst, ok, err := shownPair(mappings, from, to)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		p, err := m.prepFor(ctx, q, row, k)
		if err != nil {
			return nil, err
		}
		var carry link.Carry
		for _, f := range fs {
			rel, ok := rels[f]
			if !ok {
				continue
			}
			c := queue.Change{
				ID: link.ChangeID{UUID: storage.NewUUID()}, Server: row.ID, Kind: link.ChangeMoved, Root: to, FromRoot: from,
				Path: rel, From: rel, File: f, State: queue.Prepared, Step: link.StepNone, NextAt: now, Created: now,
			}
			p.rows = append(p.rows, c)
			carry.Changes = append(carry.Changes, link.JobChange{
				ID: c.ID, Kind: c.Kind, Library: dst.library, FromLibrary: src.library, Path: dst.folder.Join(rel), From: src.folder.Join(rel),
			})
		}
		p.job.Work = carry
		out = append(out, p)
	}
	return out, nil
}

type shown struct {
	library string
	folder  link.ServerPath
}

// shownPair is the server's shown mappings of from and to, false unless
// it shows both.
func shownPair(mappings []db.MediaserversMapping, from, to roots.ID) (src, dst shown, ok bool, err error) {
	i := slices.IndexFunc(mappings, func(mp db.MediaserversMapping) bool { return mp.RootID == from && mp.Shown })
	j := slices.IndexFunc(mappings, func(mp db.MediaserversMapping) bool { return mp.RootID == to && mp.Shown })
	if i < 0 || j < 0 {
		return src, dst, false, nil
	}
	src, err = shownOf(mappings[i])
	if err == nil {
		dst, err = shownOf(mappings[j])
	}
	return src, dst, err == nil, err
}

func shownOf(mp db.MediaserversMapping) (shown, error) {
	folder, err := link.ParsePath(*mp.ServerFolder)
	return shown{*mp.LibraryKey, folder}, err
}

// showingKind is the servers that show root and whose kind want answers
// true for.
func (m *MediaServers) showingKind(ctx context.Context, q *db.Queries, root roots.ID, want func(link.Kind) bool) ([]db.MediaserversServer, error) {
	ids, err := q.ListShowing(ctx, root)
	if err != nil {
		return nil, err
	}
	var out []db.MediaserversServer
	for _, id := range ids {
		row, err := q.GetServer(ctx, id)
		if err != nil {
			return nil, err
		}
		if k, ok := m.cfg.Kinds.Of(row.Kind); ok && want(k) {
			out = append(out, row)
		}
	}
	return out, nil
}

// Snapshotted is each server that snapshots first and shows from, and
// whether it holds a Move snapshot of every file of fs that no play since
// made stale: one seen after the snapshot, or one still playing, matched
// by its file or its path.
func (m *MediaServers) Snapshotted(ctx context.Context, r storage.Reader[storage.Library], fs []files.ID, from roots.ID) (map[link.ServerID]bool, error) {
	q := db.New(r)
	stale, err := q.ListPlayedSinceSnapshot(ctx, db.ListPlayedSinceSnapshotParams{Files: fs, FromRootID: from})
	if err != nil {
		return nil, err
	}
	playing, err := m.Playing(ctx, r, fs)
	if err != nil {
		return nil, err
	}
	return m.everyMoveRow(ctx, q, fs, from, snapshotsFirst, func(c db.MediaserversChange) bool {
		return c.Step == link.StepSnapshotted && !slices.Contains(playing[c.FileID], c.ServerID) &&
			!slices.ContainsFunc(stale, func(s db.ListPlayedSinceSnapshotRow) bool { return s.ServerID == c.ServerID && s.FileID == c.FileID })
	})
}

// Safe is each server that showed from and confirms before it holds, and
// whether it found an item at the new path of every file of fs. A carry
// still waiting on its confirm, however long, is not safe.
func (m *MediaServers) Safe(ctx context.Context, r storage.Reader[storage.Library], from roots.ID, fs []files.ID) (map[link.ServerID]bool, error) {
	confirmsFirst := func(k link.Kind) bool { return !snapshotsFirst(k) }
	return m.everyMoveRow(ctx, db.New(r), fs, from, confirmsFirst, func(c db.MediaserversChange) bool {
		return slices.Contains([]link.Step{link.StepConfirmed, link.StepRestored, link.StepSourceScanned, link.StepSettled}, c.Step)
	})
}

// everyMoveRow is each server showing from whose kind want answers true
// for, and whether every file of fs has a Move row from there that ok
// answers true for.
func (m *MediaServers) everyMoveRow(ctx context.Context, q *db.Queries, fs []files.ID, from roots.ID,
	want func(link.Kind) bool, ok func(db.MediaserversChange) bool,
) (map[link.ServerID]bool, error) {
	servers, err := m.showingKind(ctx, q, from, want)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListMoveRows(ctx, db.ListMoveRowsParams{Files: fs, FromRootID: from})
	if err != nil {
		return nil, err
	}
	out := make(map[link.ServerID]bool, len(servers))
	for _, s := range servers {
		out[s.ID] = !slices.ContainsFunc(fs, func(f files.ID) bool {
			return !slices.ContainsFunc(rows, func(c db.MediaserversChange) bool { return c.ServerID == s.ID && c.FileID == f && ok(c) })
		})
	}
	return out, nil
}

// DropMovePrepared deletes the Move's prepared rows of fs.
func DropMovePrepared(ctx context.Context, tx *storage.Tx[storage.Library], fs []files.ID) error {
	return db.New(tx).DeleteMovePrepared(ctx, fs)
}

// SourcesGone stamps source_gone_at on fs's unsent Move carries from root
// from, and wakes their senders before tx commits. A stamped row keeps its
// time.
func (m *MediaServers) SourcesGone(ctx context.Context, tx *storage.Tx[storage.Library], from roots.ID, fs []files.ID) error {
	q := db.New(tx)
	if err := q.StampSourcesGone(ctx, db.StampSourcesGoneParams{At: storage.FromTime(time.Now()), Files: fs, FromRootID: from}); err != nil {
		return err
	}
	servers, err := q.ListServersOfMoveFiles(ctx, db.ListServersOfMoveFilesParams{Files: fs, FromRootID: from})
	if err != nil {
		return err
	}
	m.senders.told(servers...)
	return nil
}

// OpenMovesFrom counts the Move carries from root that aren't sent: a
// root is deleted only after they finish, because its mapping cascades
// with it.
func OpenMovesFrom(ctx context.Context, r storage.Reader[storage.Library], root roots.ID) (int, error) {
	n, err := db.New(r).CountOpenMovesFrom(ctx, root)
	return int(n), err
}

// ShowingRoots is the roots any of servers shows.
func ShowingRoots(ctx context.Context, r storage.Reader[storage.Library], servers []link.ServerID) ([]roots.ID, error) {
	return db.New(r).ListShowingRoots(ctx, servers)
}
