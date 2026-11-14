package files

import (
	"context"
	"path"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

// move is one queued move as the disk queue and a scan read it.
type move struct {
	file ID
	work works.WorkID
	root roots.ID
	// path is where the bytes belong, sits where they are, and origin
	// where they sat when the move was queued.
	path, sits, origin string
	// ring is the cycle of renames the move is in, zero for a straight
	// move.
	ring storage.UUID
}

func moveOf(row db.ListMovesRow) move {
	return move{file: row.FileID, work: row.WorkID, root: row.RootID, path: row.Path, sits: row.Sits, origin: row.Origin, ring: row.Ring}
}

func (m move) holding() string { return holdingOf(m.file, m.path) }

// next is where the bytes go next: the holding path for a ring member not
// there yet, else path. The queue moves them there, and a scan looks for
// them there too, so it never calls a file gone or found mid-rename.
func (m move) next() string {
	if !m.ring.IsZero() && m.sits != m.holding() {
		return m.holding()
	}
	return m.path
}

func holdingOf(id ID, p string) string { return HoldingOf(id.UUID, path.Ext(p)) }

// isHeld reports whether p lies in a root's holding folder.
func isHeld(p string) bool { return strings.HasPrefix(p, storing.HoldingDir+"/") }

func folded(root roots.ID, p string) place { return place{root, storing.Fold(p)} }

// rings answers, for each cycle of straight moves whose paths fold to each
// other's bytes, the ring its moves pass through holding in: the cycle's
// lowest file id. Such moves would each wait for another forever.
func rings(moves []move) map[ID]storage.UUID {
	var straight []move
	at := map[place][]int{}
	for _, m := range moves {
		if m.ring.IsZero() && m.sits != m.holding() {
			at[folded(m.root, m.sits)] = append(at[folded(m.root, m.sits)], len(straight))
			straight = append(straight, m)
		}
	}
	waits := func(i int) []int {
		return slices.DeleteFunc(slices.Clone(at[folded(straight[i].root, straight[i].path)]), func(j int) bool { return j == i })
	}
	out := map[ID]storage.UUID{}
	for _, cycle := range cycles(len(straight), waits) {
		ring := slices.MinFunc(cycle, func(a, b int) int { return straight[a].file.Compare(straight[b].file.UUID) })
		for _, i := range cycle {
			out[straight[i].file] = straight[ring].file.UUID
		}
	}
	return out
}

// cycles answers the strongly connected components of more than one node
// of the graph of n nodes whose edges next answers (Tarjan's algorithm).
func cycles(n int, next func(int) []int) [][]int {
	index, low := make([]int, n), make([]int, n)
	on := make([]bool, n)
	var stack []int
	var out [][]int
	at := 1
	var visit func(int)
	visit = func(v int) {
		index[v], low[v] = at, at
		at++
		stack = append(stack, v)
		on[v] = true
		for _, w := range next(v) {
			switch {
			case index[w] == 0:
				visit(w)
				low[v] = min(low[v], low[w])
			case on[w]:
				low[v] = min(low[v], index[w])
			}
		}
		if low[v] != index[v] {
			return
		}
		var c []int
		for {
			w := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			on[w] = false
			c = append(c, w)
			if w == v {
				break
			}
		}
		if len(c) > 1 {
			out = append(out, c)
		}
	}
	for v := range n {
		if index[v] == 0 {
			visit(v)
		}
	}
	return out
}

// putBack answers m's path taken at disk time by a file the library
// doesn't know of. A straight move took nothing, so the row's path becomes
// where the bytes sit and the move goes; bytes at another owner's holding
// name, a fresh import's, move to this file's own holding name, which the
// sweep keeps. A ring member in holding returns to its origin when no
// settled file has it, and stays at its holding name otherwise. Every
// queued move whose path folds to a place a put-back file returns to is
// put back too, transitively, in the same Write. A move put back for good
// tells servers its bytes went from its origin to where they stay. It
// answers the Works whose moves it put back, none when m's row changed
// since the queue read it.
func (f *Files) putBack(ctx context.Context, m move) ([]works.WorkID, error) {
	var put []works.WorkID
	err := f.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		put = nil
		q := db.New(tx)
		moves, err := movesIn(ctx, q, m.root)
		if err != nil || !slices.Contains(moves, m) {
			return err
		}
		backs, err := returns(moves, m, func(p string) (bool, error) {
			settled, err := q.PathSettled(ctx, db.PathSettledParams{RootID: m.root, Path: p})
			return !settled, err
		})
		if err != nil {
			return err
		}
		if err := putAll(ctx, q, backs); err != nil {
			return err
		}
		var went []fact
		for _, b := range backs {
			put = append(put, b.m.work)
			if !b.keep {
				went = append(went, fact{file: b.m.file, root: b.m.root, from: b.m.origin, to: b.to})
			}
		}
		f.cfg.Log.Warn("another file is at a file's new path, and the file stays where it was", "root", m.root.String(), "path", m.path, "files", len(backs))
		return f.tell(ctx, tx, went)
	})
	return put, err
}

func movesIn(ctx context.Context, q *db.Queries, root roots.ID) ([]move, error) {
	rows, err := q.ListMoves(ctx)
	if err != nil {
		return nil, err
	}
	var out []move
	for _, row := range rows {
		if row.RootID == root {
			out = append(out, moveOf(row))
		}
	}
	return out, nil
}

// putAll writes each of backs: every row through its holding path first,
// so rows may trade paths without a clash.
func putAll(ctx context.Context, q *db.Queries, backs []back) error {
	for _, b := range backs {
		if err := q.SetPath(ctx, db.SetPathParams{ID: b.m.file, Path: b.m.holding()}); err != nil {
			return err
		}
	}
	for _, b := range backs {
		if err := q.SetPath(ctx, db.SetPathParams{ID: b.m.file, Path: b.to}); err != nil {
			return err
		}
		if b.keep {
			continue
		}
		if _, err := q.DeleteMove(ctx, db.DeleteMoveParams{FileID: b.m.file, Sits: b.m.sits}); err != nil {
			return err
		}
	}
	return nil
}

// back is where a move put back leaves its file's row: to, and whether
// the queue still takes the bytes there.
type back struct {
	m    move
	to   string
	keep bool
}

// returns is m put back, and every move of moves whose path folds to a
// place one put back returns to, transitively.
func returns(moves []move, m move, free func(string) (bool, error)) ([]back, error) {
	first, err := returnOf(m, free)
	if err != nil {
		return nil, err
	}
	out := []back{first}
	seen := map[ID]bool{m.file: true}
	for i := 0; i < len(out); i++ {
		to := folded(out[i].m.root, out[i].to)
		for _, n := range moves {
			if seen[n.file] || folded(n.root, n.path) != to {
				continue
			}
			seen[n.file] = true
			b, err := returnOf(n, free)
			if err != nil {
				return nil, err
			}
			out = append(out, b)
		}
	}
	return out, nil
}

func returnOf(m move, free func(string) (bool, error)) (back, error) {
	own := m.holding()
	switch {
	case m.sits == own && !isHeld(m.origin) && m.origin != m.path:
		ok, err := free(m.origin)
		if err != nil || !ok {
			return back{m: m, to: own}, err
		}
		return back{m: m, to: m.origin, keep: true}, nil
	case m.sits != own && isHeld(m.sits):
		return back{m: m, to: own, keep: true}, nil
	}
	return back{m: m, to: m.sits}, nil
}
