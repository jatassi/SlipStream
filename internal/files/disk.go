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
	"github.com/jatassi/slipstream/internal/works"
)

// Disk is the file IO the disk queue does, on the Node that holds each
// root. Move converges on a repeat, and with carry moves from's companions
// after it; Remove of a missing file is an error wrapping fs.ErrNotExist,
// and RemoveAll of a missing folder is none. Survey answers which planned
// targets other files hold, and Prune removes the folders a move emptied.
// Watch is closed at the next change of a root's report, as a re-home off
// a lost root makes. *roots.Roots is one.
type Disk interface {
	Move(ctx context.Context, root roots.ID, from, to storing.Rel, carry bool) (storing.Placed, error)
	Remove(ctx context.Context, root roots.ID, rel storing.Rel) error
	RemoveAll(ctx context.Context, root roots.ID, rel storing.Rel) error
	ListHeld(ctx context.Context, root roots.ID, folder storing.HeldFolder) ([]string, error)
	Survey(ctx context.Context, root roots.ID, planned []storing.Planned, companions bool) ([]storing.Surveyed, error)
	Prune(ctx context.Context, root roots.ID, folder storing.Rel, keep []storing.Rel) ([]storing.Rel, error)
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
// doomed file, then moves every file of a cycle of renames to its holding
// path, then every queued file to its path once no other file's bytes sit
// there, then removes the folders those moves left empty. Every step is a
// Move, a Remove or a Prune that converges on a repeat, and a row goes
// only after its step, so a crash anywhere converges on the next pass. A
// pass that leaves work behind, a root to sweep among it, runs again after
// a backoff. Each step runs on the Node that holds the file's root. Work
// in a lost root waits, without a backoff, until a root's report changes.
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
	// replacers is the path each doomed file's replacer belongs at.
	replacers map[ID]string
	moves     []move
	prunes    []db.FilesPrune
	lost      map[roots.ID]bool
}

func (f *Files) read(ctx context.Context) (queue, error) {
	var out queue
	err := f.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		var err error
		if out.doomed, err = q.ListDoomed(ctx); err != nil {
			return err
		}
		replacers, err := q.ListReplacers(ctx)
		if err != nil {
			return err
		}
		out.replacers = make(map[ID]string, len(replacers))
		for _, row := range replacers {
			out.replacers[row.FileID] = row.Path
		}
		rows, err := q.ListMoves(ctx)
		if err != nil {
			return err
		}
		out.moves = make([]move, len(rows))
		for i, row := range rows {
			out.moves[i] = moveOf(row)
		}
		if out.prunes, err = q.ListPrune(ctx); err != nil {
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
		if q.lost[m.root] {
			return true
		}
	}
	for _, p := range q.prunes {
		if q.lost[p.RootID] {
			return true
		}
	}
	return false
}

// pending is every place, folded, where bytes the queue has yet to remove
// or move on sit, by the file whose bytes they are, zero for a doomed
// file's: no other move may take it yet.
func (q queue) pending() map[place]ID {
	out := map[place]ID{}
	for _, d := range q.doomed {
		out[folded(d.RootID, d.Path)] = ID{}
		if d.Sits != nil {
			out[folded(d.RootID, *d.Sits)] = ID{}
		}
	}
	for _, m := range q.moves {
		out[folded(m.root, m.sits)] = m.file
	}
	return out
}

type place struct {
	root roots.ID
	path string
}

// drain runs one pass of the queue, skipping the rows of lost roots, and
// reports what else it left undone. A pass has the Media Servers prepare
// the pending renames, removes the doomed files, sends each ring member to
// its holding path, then moves every file whose bytes may take its path
// there, and last removes the folders moves left empty.
func (f *Files) drain(ctx context.Context) (skippedLost bool, err error) {
	q, err := f.read(ctx)
	if err != nil {
		return false, err
	}
	renames, err := q.renames()
	if err != nil {
		return false, err
	}
	f.cfg.MediaServers.Prepare(ctx, renames)
	prepared := map[ID]storing.Rel{}
	for _, r := range renames {
		if r.Leaving {
			prepared[r.File] = r.From
		}
	}
	var errs []error
	for _, d := range q.doomed {
		if !q.lost[d.RootID] {
			errs = append(errs, f.remove(ctx, q, d))
		}
	}
	moves, err := f.ring(ctx, q.moves)
	errs = append(errs, err)
	for _, m := range moves {
		if hop := m.next(); hop != m.path && !q.lost[m.root] {
			_, err := f.step(ctx, m, hop)
			errs = append(errs, err)
		}
	}
	if err := ctx.Err(); err != nil {
		return false, err //nolint:wrapcheck // Run's own cancel
	}
	if q, err = f.read(ctx); err != nil {
		return false, err
	}
	errs = append(errs, f.land(ctx, q, prepared))
	if q, err = f.read(ctx); err != nil {
		return false, errors.Join(append(errs, err)...)
	}
	errs = append(errs, f.prune(ctx, q))
	return q.waits(), errors.Join(errs...)
}

// land moves every file of q whose next hop is its path there, unless
// another file's bytes still sit there, and puts back each whose path
// another file holds. A rename queued since the pass's Prepare, the origin
// of a file not in prepared, waits for the next pass, which its Write's
// poke runs. It publishes files.moved for the Works whose moves it
// finished or put back.
func (f *Files) land(ctx context.Context, q queue, prepared map[ID]storing.Rel) error {
	var errs []error
	var touched []works.WorkID
	pending := q.pending()
	for _, m := range q.moves {
		if m.next() != m.path || q.lost[m.root] || m.leaves() && prepared[m.file].String() != m.origin {
			continue
		}
		if at, ok := pending[folded(m.root, m.path)]; ok && at != m.file {
			errs = append(errs, fmt.Errorf("%s waits for the bytes at its path to go", m.path))
			continue
		}
		done, err := f.step(ctx, m, m.path)
		switch {
		case errors.Is(err, storing.ErrOccupied):
			put, err := f.putBack(ctx, m)
			touched = append(touched, put...)
			errs = append(errs, err)
		case done:
			touched = append(touched, m.work)
		default:
			errs = append(errs, err)
		}
	}
	if len(touched) > 0 {
		f.cfg.Moved.Publish(ctx, MovedOf(touched))
	}
	return errors.Join(errs...)
}

// ring records the ring of every move that rings finds in a cycle, and
// answers moves with their rings.
func (f *Files) ring(ctx context.Context, moves []move) ([]move, error) {
	found := rings(moves)
	if len(found) == 0 {
		return moves, nil
	}
	out := slices.Clone(moves)
	err := f.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		for i, m := range moves {
			ring, ok := found[m.file]
			if !ok {
				continue
			}
			n, err := q.SetRing(ctx, db.SetRingParams{FileID: m.file, Sits: m.sits, Ring: ring})
			if err != nil {
				return err
			}
			if n == 1 {
				out[i].ring = ring
			}
		}
		return nil
	})
	if err != nil {
		return moves, err
	}
	return out, nil
}

// step moves m's bytes and their companions from m.sits to to, then
// records that they reached it, and that m.sits's folder may be empty
// now. The move is done when to is the file's path still, and a rename
// from m's origin is recorded; else the bytes sit at to until the next
// pass. A target another file holds is storing.ErrOccupied.
func (f *Files) step(ctx context.Context, m move, to string) (done bool, err error) {
	if err := ctx.Err(); err != nil {
		return false, err //nolint:wrapcheck // Run's own cancel
	}
	placed, err := f.move(ctx, m.root, m.sits, to)
	if err != nil {
		return false, fmt.Errorf("move %s to %s: %w", m.sits, to, err)
	}
	for _, left := range placed.Left {
		f.cfg.Log.Warn("another file has a companion's new name", "root", m.root.String(), "file", to, "companion", left.String())
	}
	err = f.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var went []fact
		var err error
		if done, went, err = reached(ctx, db.New(tx), m, to); err != nil {
			return err
		}
		return f.tell(ctx, tx, went)
	})
	return done, err
}

// reached records that m's bytes reached to, and whether that finished m:
// then the bytes went from m's origin to its path.
func reached(ctx context.Context, q *db.Queries, m move, to string) (bool, []fact, error) {
	if from := path.Dir(m.sits); from != "." && !isHeld(m.sits) && from != path.Dir(to) {
		if err := q.InsertPrune(ctx, db.InsertPruneParams{RootID: m.root, Folder: from}); err != nil {
			return false, nil, err
		}
	}
	if to == m.path {
		n, err := q.FinishMove(ctx, db.FinishMoveParams{FileID: m.file, Sits: m.sits, Path: to})
		went := []fact{{file: m.file, root: m.root, from: m.origin, to: to}}
		switch {
		case err != nil:
			return false, nil, err
		case n == 1 && (isHeld(m.origin) || m.origin == to):
			return true, went, nil
		case n == 1:
			return true, went, q.InsertRenamed(ctx, db.InsertRenamedParams{FileID: m.file, RootID: m.root, FromPath: m.origin, ToPath: to, RenamedAt: storage.Now()})
		}
	}
	_, err := q.MoveReached(ctx, db.MoveReachedParams{FileID: m.file, Sits: m.sits, Reached: to})
	return false, nil, err
}

// prune removes each folder a finished move left, with its empty parents,
// unless a queued move's path, sits or next hop lies under it.
func (f *Files) prune(ctx context.Context, q queue) error {
	keep := map[place][]storing.Rel{}
	for _, m := range q.moves {
		for _, p := range []string{m.path, m.sits, m.next()} {
			dir := path.Dir(p)
			if dir == "." || isHeld(p) {
				continue
			}
			rel, err := storing.ParseRel(dir)
			if err != nil {
				return err
			}
			top := folded(m.root, topOf(dir))
			keep[top] = append(keep[top], rel)
		}
	}
	var errs []error
	for _, p := range q.prunes {
		if q.lost[p.RootID] {
			continue
		}
		if err := f.pruneOne(ctx, p, keep[folded(p.RootID, topOf(p.Folder))]); err != nil {
			errs = append(errs, fmt.Errorf("remove the emptied folder %s: %w", p.Folder, err))
		}
	}
	return errors.Join(errs...)
}

func (f *Files) pruneOne(ctx context.Context, p db.FilesPrune, keep []storing.Rel) error {
	if err := ctx.Err(); err != nil {
		return err //nolint:wrapcheck // Run's own cancel
	}
	folder, err := storing.ParseRel(p.Folder)
	if err != nil {
		return err
	}
	if _, err := f.cfg.Disk.Prune(ctx, p.RootID, folder, keep); err != nil {
		return err
	}
	return f.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).DeletePrune(ctx, db.DeletePruneParams(p))
	})
}

// topOf is p's first segment.
func topOf(p string) string {
	top, _, _ := strings.Cut(p, "/")
	return top
}

// remove removes a doomed file's bytes, then its row. Bytes that were
// moving sit at its move's place or its holding path; when neither holds
// them the move had taken them to its path, which goes too unless another
// file's bytes wait there. Servers saw them leave the path the doom names.
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
		if err := db.New(tx).DeleteDoomed(ctx, d.FileID); err != nil {
			return err
		}
		return f.tell(ctx, tx, []fact{{file: d.FileID, root: d.RootID, from: named(d), replacedBy: d.ReplacedBy}})
	})
}

// sweep removes what the held folders of roots ids hold that nothing
// claims: a holding file whose name is no live file's or unfinished
// Grab's, and a staging folder whose name is no live Transfer's. A Grab
// that failed after its import placed the bytes, a Transfer canceled
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
			if live.keeps(root, e) || moving.holds(root, e.path()) {
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

func (f *Files) move(ctx context.Context, root roots.ID, from, to string) (storing.Placed, error) {
	src, err := storing.ParseRel(from)
	if err != nil {
		return storing.Placed{}, err
	}
	dst, err := storing.ParseRel(to)
	if err != nil {
		return storing.Placed{}, err
	}
	return f.cfg.Disk.Move(ctx, root, src, dst, true)
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
// every path an unfinished import claims in holding, with the .part a
// copy to it writes first, and every live Transfer's id in staging.
type claims struct {
	holding, staging map[storage.UUID]bool
	paths            map[roots.ID]map[string]bool
}

func (c claims) keeps(root roots.ID, e held) bool {
	if e.folder == storing.Holding && (c.paths[root][e.path()] || c.paths[root][strings.TrimSuffix(e.path(), ".part")]) {
		return true
	}
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
		if c.paths, err = f.cfg.Grabs.Holding(ctx, r); err != nil {
			return err
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
		if m.root == root && m.sits == p {
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
