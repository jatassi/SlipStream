package files

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/backoff"
	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

// Disk is the file IO the disk queue does. Move converges as files.Move
// does; Remove of a missing file is an error wrapping fs.ErrNotExist.
type Disk interface {
	Move(from, to string) error
	Remove(path string) error
	ReadDir(dir string) ([]fs.DirEntry, error)
}

// OSDisk is Disk on the Node's own filesystem.
type OSDisk struct{}

func (OSDisk) Move(from, to string) error { return Move(from, to) }

func (OSDisk) Remove(name string) error {
	return os.Remove(name) //nolint:wrapcheck // the queue reads fs.ErrNotExist and logs the rest as is
}

func (OSDisk) ReadDir(dir string) ([]fs.DirEntry, error) {
	return os.ReadDir(dir) //nolint:wrapcheck // the sweep reads fs.ErrNotExist and logs the rest as is
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

// Run drains the disk queue until ctx ends, after sweeping each root's
// holding folder. Each pass removes every doomed file, then moves every
// queued file to its holding path, then from there to its path. Every
// step is a Move or a Remove that converges on a repeat, and a row goes
// only after its step, so a crash anywhere converges on the next pass.
// A pass that leaves work behind runs again after a backoff. Every root
// sits on the Controller today; a root on another Storage Node (Moves)
// routes the same Move and Remove there.
func (f *Files) Run(ctx context.Context) error {
	f.sweep(ctx)
	attempts := 0
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-f.poke:
		case <-timer.C:
		}
		err := f.drain(ctx)
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

// queue is the disk queue as one Read found it.
type queue struct {
	doomed []db.FilesDoomed
	moves  []db.ListMovesRow
	roots  map[roots.ID]roots.Root
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
		out.roots = map[roots.ID]roots.Root{}
		for _, id := range out.rootIDs() {
			if out.roots[id], err = roots.Get(ctx, r, id); err != nil {
				return err
			}
		}
		return nil
	})
	return out, err
}

func (q queue) rootIDs() []roots.ID {
	var ids []roots.ID
	seen := map[roots.ID]bool{}
	add := func(id roots.ID) {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, d := range q.doomed {
		add(d.RootID)
	}
	for _, m := range q.moves {
		add(m.RootID)
	}
	return ids
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

func (q queue) local(root roots.ID, p string) string {
	return filepath.Join(q.roots[root].Path, filepath.FromSlash(p))
}

// drain runs one pass of the queue and reports what it left undone.
func (f *Files) drain(ctx context.Context) error {
	q, err := f.read(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, d := range q.doomed {
		errs = append(errs, f.remove(ctx, q, d))
	}
	for _, m := range q.moves {
		if hold := holdingOf(m.FileID, m.Path); m.Sits != hold {
			errs = append(errs, f.step(ctx, q, m, hold))
		}
	}
	if err := ctx.Err(); err != nil {
		return err //nolint:wrapcheck // Run's own cancel
	}
	if q, err = f.read(ctx); err != nil {
		return err
	}
	pending := q.pending()
	for _, m := range q.moves {
		switch {
		case m.Sits != holdingOf(m.FileID, m.Path):
		case pending[place{m.RootID, m.Path}]:
			errs = append(errs, fmt.Errorf("%s waits for the bytes at its path to go", m.Path))
		default:
			errs = append(errs, f.step(ctx, q, m, m.Path))
		}
	}
	return errors.Join(errs...)
}

// step moves m's bytes from m.Sits to to, then records that they reached
// it: the move is done when to is the file's path still, else they sit at
// to until the next pass.
func (f *Files) step(ctx context.Context, q queue, m db.ListMovesRow, to string) error {
	if err := ctx.Err(); err != nil {
		return err //nolint:wrapcheck // Run's own cancel
	}
	if err := f.cfg.Disk.Move(q.local(m.RootID, m.Sits), q.local(m.RootID, to)); err != nil {
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
		switch err := f.cfg.Disk.Remove(q.local(d.RootID, p)); {
		case errors.Is(err, fs.ErrNotExist):
			gone++
		case err != nil:
			return fmt.Errorf("remove %s: %w", p, err)
		}
	}
	if d.Sits != nil && gone == len(places) && !q.holds(d.RootID, d.Path) {
		if err := f.cfg.Disk.Remove(q.local(d.RootID, d.Path)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("remove %s: %w", d.Path, err)
		}
	}
	return f.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).DeleteDoomed(ctx, d.FileID)
	})
}

// sweep removes each file in a root's holding folder whose name is no
// live file's or unfinished Grab's: what a Grab that failed after its
// import placed the bytes, or a crash between two steps, leaves. It lists
// the folders before reading the library, so a holding file placed
// meanwhile belongs to a Grab or file the read sees.
func (f *Files) sweep(ctx context.Context) {
	found, err := f.incoming(ctx)
	if err != nil {
		f.logSweep(ctx, err)
		return
	}
	live, err := f.claimed(ctx)
	if err != nil {
		f.logSweep(ctx, err)
		return
	}
	moving, err := f.read(ctx)
	if err != nil {
		f.logSweep(ctx, err)
		return
	}
	for root, names := range found {
		for _, n := range names {
			stem, _, _ := strings.Cut(n, ".")
			id, err := storage.ParseUUID(stem)
			if err == nil && live[id] || moving.holds(root.ID, incomingDir+"/"+n) {
				continue
			}
			if err := f.cfg.Disk.Remove(filepath.Join(root.Path, incomingDir, n)); err != nil && !errors.Is(err, fs.ErrNotExist) {
				f.logSweep(ctx, err)
			}
		}
	}
}

// incoming is what each root's incoming folder holds.
func (f *Files) incoming(ctx context.Context) (map[roots.Root][]string, error) {
	var rs []roots.Root
	err := f.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		rs, err = roots.All(ctx, r)
		return err
	})
	if err != nil {
		return nil, err
	}
	found := map[roots.Root][]string{}
	for _, root := range rs {
		entries, err := f.cfg.Disk.ReadDir(filepath.Join(root.Path, incomingDir))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			f.logSweep(ctx, err)
		}
		for _, e := range entries {
			found[root] = append(found[root], e.Name())
		}
	}
	return found, nil
}

// claimed is every live file's id and unfinished Grab's.
func (f *Files) claimed(ctx context.Context) (map[storage.UUID]bool, error) {
	live := map[storage.UUID]bool{}
	err := f.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		ids, err := db.New(r).ListFileIDs(ctx)
		if err != nil {
			return err
		}
		for _, id := range ids {
			live[id.UUID] = true
		}
		grabs, err := f.cfg.Grabs.Unfinished(ctx, r)
		for _, id := range grabs {
			live[id] = true
		}
		return err
	})
	return live, err
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
		f.cfg.Log.Error("sweep the holding folders", "err", err.Error())
	}
}
