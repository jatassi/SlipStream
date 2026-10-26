package files

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/backoff"
	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

// Disk is the file IO the disk queue does, on the Node that holds each
// root. Move converges on a repeat; Remove of a missing file is an error
// wrapping fs.ErrNotExist, and RemoveAll of a missing folder is none.
// Watch is closed at the next change of a root's report, as a re-home off
// a lost root makes. *roots.Roots is one.
type Disk interface {
	Move(ctx context.Context, root roots.ID, from, to storing.Rel) (int64, error)
	Remove(ctx context.Context, root roots.ID, rel storing.Rel) error
	RemoveAll(ctx context.Context, root roots.ID, rel storing.Rel) error
	ListHeld(ctx context.Context, root roots.ID, folder storing.HeldFolder) ([]string, error)
	Watch() <-chan struct{}
}

type Timings struct {
	// RetryMin and RetryMax bound the backoff before the disk queue runs
	// again after a move or removal failed.
	RetryMin time.Duration `yaml:"retry_min"`
	RetryMax time.Duration `yaml:"retry_max"`
}

func DefaultTimings() Timings { return Timings{RetryMin: time.Second, RetryMax: time.Minute} }

// Poke has the disk queue run soon: a commit that queued disk work calls
// it once the Write returns.
func (f *Files) Poke() {
	select {
	case f.poke <- struct{}{}:
	default:
	}
}

// Run drains the disk queue until ctx ends. Each pass first sweeps the
// held folders of each root not swept since start, then removes every
// doomed file, then moves every queued file to its holding path, then
// from there to its path. Every step is a Move or a Remove that converges
// on a repeat, and a row goes only after its step, so a crash anywhere
// converges on the next pass. A pass that leaves work behind, a root to
// sweep among it, runs again after a backoff. Each step runs on the Node
// that holds the file's root. Work in a lost root waits, without a
// backoff, until a root's report changes.
func (f *Files) Run(ctx context.Context) error {
	unswept, err := f.rootIDs(ctx)
	if err != nil {
		f.logSweep(ctx, err)
	}
	attempts := 0
	timer := time.NewTimer(0)
	defer timer.Stop()
	var watch <-chan struct{}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-f.poke:
		case <-timer.C:
		case <-watch:
		}
		watch = f.cfg.Disk.Watch()
		unanswered, lostRoots := f.sweep(ctx, unswept)
		unswept = slices.Concat(unanswered, lostRoots)
		lostRows, err := f.drain(ctx)
		if len(unanswered) > 0 {
			err = errors.Join(err, fmt.Errorf("%d roots wait for their sweep", len(unanswered)))
		}
		if len(lostRoots) == 0 && !lostRows {
			watch = nil
		}
		switch {
		case ctx.Err() != nil:
			return nil
		case err != nil:
			f.cfg.Log.Warn("the disk queue retries", "err", err.Error())
			timer.Reset(backoff.Delay(f.cfg.Timings.RetryMin, f.cfg.Timings.RetryMax, attempts, f.cfg.Rand))
			attempts++
		default:
			attempts = 0
		}
	}
}

type queue struct {
	doomed []db.FilesDoomed
	moves  []db.ListMovesRow
	lost   map[roots.ID]bool
}

func (f *Files) read(ctx context.Context) (queue, error) {
	var out queue
	err := f.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		var err error
		if out.doomed, err = q.ListDoomed(ctx); err != nil {
			return err
		}
		if out.moves, err = q.ListMoves(ctx); err != nil {
			return err
		}
		out.lost, err = roots.LostIn(ctx, r)
		return err
	})
	return out, err
}

func (q queue) waits() bool {
	for _, d := range q.doomed {
		if q.lost[d.RootID] {
			return true
		}
	}
	for _, m := range q.moves {
		if q.lost[m.RootID] {
			return true
		}
	}
	return false
}

// pending is every place bytes the queue has yet to remove or move on
// sit: no move may take it yet.
func (q queue) pending() map[place]bool {
	out := map[place]bool{}
	for _, d := range q.doomed {
		out[place{d.RootID, d.Path}] = true
		if d.Sits != nil {
			out[place{d.RootID, *d.Sits}] = true
		}
	}
	for _, m := range q.moves {
		if m.Sits != holdingOf(m.FileID, m.Path) {
			out[place{m.RootID, m.Sits}] = true
		}
	}
	return out
}

type place struct {
	root roots.ID
	path string
}

func holdingOf(id ID, p string) string { return HoldingOf(id.UUID, path.Ext(p)) }

// drain runs one pass of the queue, skipping the rows of lost roots, and
// reports what else it left undone.
func (f *Files) drain(ctx context.Context) (skippedLost bool, err error) {
	q, err := f.read(ctx)
	if err != nil {
		return false, err
	}
	var errs []error
	for _, d := range q.doomed {
		if !q.lost[d.RootID] {
			errs = append(errs, f.remove(ctx, q, d))
		}
	}
	for _, m := range q.moves {
		if hold := holdingOf(m.FileID, m.Path); m.Sits != hold && !q.lost[m.RootID] {
			errs = append(errs, f.step(ctx, m, hold))
		}
	}
	if err := ctx.Err(); err != nil {
		return false, err //nolint:wrapcheck // Run's own cancel
	}
	if q, err = f.read(ctx); err != nil {
		return false, err
	}
	pending := q.pending()
	for _, m := range q.moves {
		switch {
		case m.Sits != holdingOf(m.FileID, m.Path), q.lost[m.RootID]:
		case pending[place{m.RootID, m.Path}]:
			errs = append(errs, fmt.Errorf("%s waits for the bytes at its path to go", m.Path))
		default:
			errs = append(errs, f.step(ctx, m, m.Path))
		}
	}
	return q.waits(), errors.Join(errs...)
}

// step moves m's bytes from m.Sits to to, then records that they reached
// it: the move is done when to is the file's path still, else they sit at
// to until the next pass.
func (f *Files) step(ctx context.Context, m db.ListMovesRow, to string) error {
	if err := ctx.Err(); err != nil {
		return err //nolint:wrapcheck // Run's own cancel
	}
	if err := f.move(ctx, m.RootID, m.Sits, to); err != nil {
		return fmt.Errorf("move %s to %s: %w", m.Sits, to, err)
	}
	return f.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		dq := db.New(tx)
		if to == m.Path {
			done, err := dq.FinishMove(ctx, db.FinishMoveParams{FileID: m.FileID, Sits: m.Sits, Path: to})
			if err != nil || done == 1 {
				return err
			}
		}
		_, err := dq.MoveReached(ctx, db.MoveReachedParams{FileID: m.FileID, Sits: m.Sits, Reached: to})
		return err
	})
}

// remove removes a doomed file's bytes, then its row. Bytes that were
// moving sit at its move's place or its holding path; when neither holds
// them the move had taken them to its path, which goes too unless another
// file's bytes wait there.
func (f *Files) remove(ctx context.Context, q queue, d db.FilesDoomed) error {
	if err := ctx.Err(); err != nil {
		return err //nolint:wrapcheck // Run's own cancel
	}
	places := []string{d.Path}
	if d.Sits != nil {
		places = []string{*d.Sits, holdingOf(d.FileID, d.Path)}
	}
	gone := 0
	for _, p := range places {
		switch err := f.removeAt(ctx, d.RootID, p); {
		case errors.Is(err, fs.ErrNotExist):
			gone++
		case err != nil:
			return fmt.Errorf("remove %s: %w", p, err)
		}
	}
	if d.Sits != nil && gone == len(places) && !q.holds(d.RootID, d.Path) {
		if err := f.removeAt(ctx, d.RootID, d.Path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("remove %s: %w", d.Path, err)
		}
	}
	return f.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).DeleteDoomed(ctx, d.FileID)
	})
}

// sweep removes what the held folders of roots ids hold that nothing
// claims: a holding file whose name is no live file's or unfinished
// Grab's, and a staging folder whose name is no live Transfer's. A Grab
// that failed after its import placed the bytes, a Transfer cancelled
// while its Node was away, or a crash between two steps leaves them. It
// lists the folders, on each root's Node, before reading the library, so
// what is placed or staged meanwhile belongs to a Grab, file or Transfer
// the read sees. It answers the roots left to sweep: those whose Node
// didn't answer or didn't remove an entry, and the lost ones, which it
// doesn't ask.
func (f *Files) sweep(ctx context.Context, ids []roots.ID) (unanswered, lost []roots.ID) {
	if len(ids) == 0 {
		return nil, nil
	}
	lost, asked, err := f.lostOf(ctx, ids)
	if err != nil {
		f.logSweep(ctx, err)
		return ids, nil
	}
	found, unanswered := f.held(ctx, asked)
	live, err := f.claimed(ctx)
	if err != nil {
		f.logSweep(ctx, err)
		return asked, lost
	}
	moving, err := f.read(ctx)
	if err != nil {
		f.logSweep(ctx, err)
		return asked, lost
	}
	for root, entries := range found {
		removed := true
		for _, e := range entries {
			if live.keeps(e) || moving.holds(root, e.path()) {
				continue
			}
			if err := f.removeHeld(ctx, root, e); err != nil && !errors.Is(err, fs.ErrNotExist) {
				f.logSweep(ctx, err)
				removed = false
			}
		}
		if !removed {
			unanswered = append(unanswered, root)
		}
	}
	return unanswered, lost
}

// held is one entry of a root's held folder: a file in holding, named by
// its file's or Grab's id, or a folder in staging, named by its
// Transfer's.
type held struct {
	folder storing.HeldFolder
	name   string
}

func (e held) path() string { return e.folder.String() + "/" + e.name }

// owner is the id e's name starts with, false when it starts with none.
func (e held) owner() (storage.UUID, bool) {
	stem, _, _ := strings.Cut(e.name, ".")
	id, err := storage.ParseUUID(stem)
	return id, err == nil
}

func (f *Files) lostOf(ctx context.Context, ids []roots.ID) (lost, others []roots.ID, err error) {
	err = f.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		of, err := roots.LostIn(ctx, r)
		for _, id := range ids {
			if of[id] {
				lost = append(lost, id)
			} else {
				others = append(others, id)
			}
		}
		return err
	})
	return lost, others, err
}

func (f *Files) rootIDs(ctx context.Context) ([]roots.ID, error) {
	var ids []roots.ID
	err := f.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		rs, err := roots.All(ctx, r)
		for _, root := range rs {
			ids = append(ids, root.ID)
		}
		return err
	})
	return ids, err
}

// held is every entry of the holding and staging folders of each of ids,
// and the roots whose Node didn't list both.
func (f *Files) held(ctx context.Context, ids []roots.ID) (found map[roots.ID][]held, unanswered []roots.ID) {
	found = map[roots.ID][]held{}
	for _, root := range ids {
		entries, err := f.heldIn(ctx, root)
		if err != nil {
			f.cfg.Log.Warn("sweep a root's held folders later", "root", root.String(), "err", err.Error())
			unanswered = append(unanswered, root)
			continue
		}
		found[root] = entries
	}
	return found, unanswered
}

func (f *Files) heldIn(ctx context.Context, root roots.ID) ([]held, error) {
	var entries []held
	for _, folder := range []storing.HeldFolder{storing.Holding, storing.Staging} {
		names, err := f.cfg.Disk.ListHeld(ctx, root, folder)
		if err != nil {
			return nil, err
		}
		for _, n := range names {
			entries = append(entries, held{folder, n})
		}
	}
	return entries, nil
}

func (f *Files) move(ctx context.Context, root roots.ID, from, to string) error {
	src, err := storing.ParseRel(from)
	if err != nil {
		return err
	}
	dst, err := storing.ParseRel(to)
	if err != nil {
		return err
	}
	_, err = f.cfg.Disk.Move(ctx, root, src, dst)
	return err
}

func (f *Files) removeAt(ctx context.Context, root roots.ID, p string) error {
	rel, err := storing.ParseRel(p)
	if err != nil {
		return err
	}
	return f.cfg.Disk.Remove(ctx, root, rel)
}

// removeHeld removes a holding file, or a staging folder and everything
// in it.
func (f *Files) removeHeld(ctx context.Context, root roots.ID, e held) error {
	if e.folder == storing.Holding {
		return f.removeAt(ctx, root, e.path())
	}
	rel, err := storing.ParseRel(e.path())
	if err != nil {
		return err
	}
	return f.cfg.Disk.RemoveAll(ctx, root, rel)
}

// claims is what the held folders may keep: every live file's id and
// unfinished Grab's in holding, and every live Transfer's in staging.
type claims struct {
	holding, staging map[storage.UUID]bool
}

func (c claims) keeps(e held) bool {
	id, ok := e.owner()
	if !ok {
		return false
	}
	if e.folder == storing.Holding {
		return c.holding[id]
	}
	return c.staging[id]
}

func (f *Files) claimed(ctx context.Context) (claims, error) {
	c := claims{holding: map[storage.UUID]bool{}, staging: map[storage.UUID]bool{}}
	err := f.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		ids, err := db.New(r).ListFileIDs(ctx)
		if err != nil {
			return err
		}
		for _, id := range ids {
			c.holding[id.UUID] = true
		}
		grabs, err := f.cfg.Grabs.Unfinished(ctx, r)
		if err != nil {
			return err
		}
		for _, id := range grabs {
			c.holding[id] = true
		}
		live, err := f.cfg.Transfers(ctx, r)
		for id := range live {
			c.staging[id.UUID] = true
		}
		return err
	})
	return c, err
}

// holds reports whether a live file's bytes are recorded at p in root.
func (q queue) holds(root roots.ID, p string) bool {
	for _, m := range q.moves {
		if m.RootID == root && m.Sits == p {
			return true
		}
	}
	return false
}

func (f *Files) logSweep(ctx context.Context, err error) {
	if ctx.Err() == nil {
		f.cfg.Log.Error("sweep the held folders", "err", err.Error())
	}
}
