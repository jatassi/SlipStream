package files

import (
	"context"
	"errors"
	"maps"
	"runtime"
	"slices"
	"sync"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// moves is how far a Change moves each file's score at each Target, for
// the files whose score moves.
type moves map[ID]map[targets.TargetID]int

// flips runs edits over the files of ws and answers the Works holding a
// file whose score at some Target moves, and those files' moves. It reads
// the files a batch of Works at a time and hands each batch to GOMAXPROCS
// goroutines as the next one reads, so matching overlaps the read. It
// reads and decodes a file's Title only when an edit tests more than the
// name.
func (f *Files) flips(ctx context.Context, r storage.Reader[file], ws []works.WorkID, edits *targets.Narrow) ([]works.WorkID, moves, error) {
	nameOnly := edits.Match.NameOnly()
	var originals map[works.WorkID]release.Fact[release.Language]
	if !nameOnly {
		var err error
		if originals, err = f.originals(ctx, r, ws); err != nil {
			return nil, nil, err
		}
	}
	batches := make(chan []db.ListFileReadsOfWorksRow, runtime.GOMAXPROCS(0))
	outs := make([]flipped, runtime.GOMAXPROCS(0))
	var wg sync.WaitGroup
	for i := range outs {
		wg.Go(func() {
			outs[i] = flipped{moved: moves{}, works: map[works.WorkID]bool{}}
			for batch := range batches {
				if outs[i].err == nil {
					outs[i].err = outs[i].flip(batch, edits, nameOnly, originals)
				}
			}
		})
	}
	err := readNamed(ctx, db.New(r), ws, nameOnly, func(rows []db.ListFileReadsOfWorksRow) {
		for batch := range slices.Chunk(rows, flipBatch) {
			batches <- batch
		}
	})
	close(batches)
	wg.Wait()
	moved, kept := moves{}, map[works.WorkID]bool{}
	for _, out := range outs {
		err = errors.Join(err, out.err)
		maps.Copy(moved, out.moved)
		maps.Copy(kept, out.works)
	}
	if err != nil {
		return nil, nil, err
	}
	return slices.DeleteFunc(slices.Clone(ws), func(w works.WorkID) bool { return !kept[w] }), moved, nil
}

// flipped is what one goroutine of the flip pass found: the files whose
// score moves and their Works.
type flipped struct {
	moved moves
	works map[works.WorkID]bool
	err   error
}

// flip runs edits over rows, decoding a file's Title only when an edit
// tests more than the name.
func (out *flipped) flip(rows []db.ListFileReadsOfWorksRow, edits *targets.Narrow, nameOnly bool, originals map[works.WorkID]release.Fact[release.Language]) error {
	for _, row := range rows {
		var title db.Reading
		if !nameOnly {
			if err := title.Scan(row.TitleJson); err != nil {
				return err
			}
		}
		if m := edits.Moves(asNamed(title.Read, row.ReleaseName, originals[row.WorkID])); len(m) > 0 {
			out.moved[row.ID], out.works[row.WorkID] = m, true
		}
	}
	return nil
}

// flipBatch is how many files one goroutine of the flip pass takes at a
// time.
const flipBatch = 512

// readNamed reads the files of ws a batch of Works at a time and hands
// each batch's files to each, without their Titles when nameOnly.
func readNamed(ctx context.Context, q *db.Queries, ws []works.WorkID, nameOnly bool, each func([]db.ListFileReadsOfWorksRow)) error {
	for chunk := range slices.Chunk(ws, works.Batch) {
		if !nameOnly {
			rows, err := q.ListFileReadsOfWorks(ctx, chunk)
			if err != nil {
				return err
			}
			each(rows)
			continue
		}
		rows, err := q.ListFileNamesOfWorks(ctx, chunk)
		if err != nil {
			return err
		}
		named := make([]db.ListFileReadsOfWorksRow, len(rows))
		for i, row := range rows {
			named[i] = db.ListFileReadsOfWorksRow{ID: row.ID, WorkID: row.WorkID, ReleaseName: row.ReleaseName}
		}
		each(named)
	}
	return nil
}

// inParallel calls do for each index below n, split in contiguous runs
// across GOMAXPROCS goroutines.
func inParallel(n int, do func(i int)) {
	size := max(1, (n+runtime.GOMAXPROCS(0)-1)/runtime.GOMAXPROCS(0))
	var wg sync.WaitGroup
	for lo := 0; lo < n; lo += size {
		wg.Go(func() {
			for i := lo; i < min(lo+size, n); i++ {
				do(i)
			}
		})
	}
	wg.Wait()
}

// originals is the original language of each of ws its Content Type
// knows.
func (f *Files) originals(ctx context.Context, r storage.Reader[file], ws []works.WorkID) (map[works.WorkID]release.Fact[release.Language], error) {
	byType := map[works.TypeID][]works.WorkID{}
	for chunk := range slices.Chunk(ws, works.Batch) {
		sums, err := works.SummariesOf(ctx, r, chunk)
		if err != nil {
			return nil, err
		}
		for _, s := range sums {
			byType[s.ContentType] = append(byType[s.ContentType], s.ID)
		}
	}
	out := map[works.WorkID]release.Fact[release.Language]{}
	for _, typ := range slices.Sorted(maps.Keys(byType)) {
		of, err := f.cfg.Targets.OriginalsIn(ctx, r, typ, works.Only(byType[typ]))
		if err != nil {
			return nil, err
		}
		maps.Copy(out, of)
	}
	return out, nil
}

// asNamed is a file's Title as a Custom Format reads it: a Release title
// condition tests its release name, and an unknown language reads as the
// Work's original.
func asNamed(title release.Read, name string, original release.Fact[release.Language]) release.Read {
	title.Release = name
	title.Facts.Coverage.Original = original
	return title
}

// onto is the score at its Target of each file filling a cell of fills,
// moved by m, none for a Change with no format edits.
func (m moves) onto(fills map[CellKey]Fill) map[ID]map[targets.TargetID]int {
	if m == nil {
		return nil
	}
	out := map[ID]map[targets.TargetID]int{}
	for key, fill := range fills {
		if out[fill.File] == nil {
			out[fill.File] = map[targets.TargetID]int{}
		}
		out[fill.File][key.Target] = fill.Score + m[fill.File][key.Target]
	}
	return out
}

// after is each of trees' world once a Change applied, given before,
// their world before it. A Change narrowed by format edits moves scores
// alone: it changes only the Goals and the formats they score, so the
// rest of before's world stands.
func (f *Files) after(ctx context.Context, tx *storage.Tx[file], trees []works.Bare, ids []works.WorkID, before []world, moved moves) ([]world, error) {
	if moved == nil {
		return f.worlds(ctx, tx, trees, works.Only(ids), wholeFiles)
	}
	goals, err := targets.GoalsOf(ctx, tx, works.Only(ids))
	if err != nil {
		return nil, err
	}
	catalog, err := f.cfg.Targets.FormatsIn(ctx, tx)
	if err != nil {
		return nil, err
	}
	out := slices.Clone(before)
	for i := range out {
		g := goals[out[i].tree.ID]
		out[i].goals, out[i].scoring = g, newScoring(catalog, g, before[i].scoring.original)
	}
	return out, nil
}
