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

// flips runs each of edits over the files of ws, across GOMAXPROCS
// goroutines on one prepared name per file, and answers the Works holding
// a file whose score at some Target moves, and those files' moves. It
// decodes a file's Title, in those goroutines, only when an edit tests
// more than the name.
func (f *Files) flips(ctx context.Context, r storage.Reader[file], ws []works.WorkID, edits []targets.FormatEdit) ([]works.WorkID, moves, error) {
	q := db.New(r)
	var rows []db.ListFileReadsOfWorksRow
	for chunk := range slices.Chunk(ws, works.Batch) {
		of, err := q.ListFileReadsOfWorks(ctx, chunk)
		if err != nil {
			return nil, nil, err
		}
		rows = append(rows, of...)
	}
	originals, err := f.originals(ctx, r, ws)
	if err != nil {
		return nil, nil, err
	}
	nameOnly := !slices.ContainsFunc(edits, func(e targets.FormatEdit) bool { return !e.Match.NameOnly() })
	moved := make([]map[targets.TargetID]int, len(rows))
	failed := make([]error, len(rows))
	inParallel(len(rows), func(i int) {
		var title db.Reading
		if !nameOnly {
			if failed[i] = title.Scan(rows[i].TitleJson); failed[i] != nil {
				return
			}
		}
		moved[i] = movesOf(asNamed(title.Read, rows[i].ReleaseName, originals[rows[i].WorkID]), edits)
	})
	if err := errors.Join(failed...); err != nil {
		return nil, nil, err
	}
	out := moves{}
	kept := map[works.WorkID]bool{}
	for i, m := range moved {
		if len(m) > 0 {
			out[rows[i].ID] = m
			kept[rows[i].WorkID] = true
		}
	}
	return slices.DeleteFunc(slices.Clone(ws), func(w works.WorkID) bool { return !kept[w] }), out, nil
}

// movesOf is how far edits move read's score at each Target, leaving out
// the Targets it doesn't move.
func movesOf(read release.Read, edits []targets.FormatEdit) map[targets.TargetID]int {
	var out map[targets.TargetID]int
	for _, e := range edits {
		flip := e.Match.Match(read)
		for t := range e.Scores {
			if m := e.Move(flip, t); m != 0 {
				if out == nil {
					out = map[targets.TargetID]int{}
				}
				out[t] += m
			}
		}
	}
	maps.DeleteFunc(out, func(_ targets.TargetID, n int) bool { return n == 0 })
	return out
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
	title.Given = name
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

// reread reads the files of before's Works once a Change applied: for
// one narrowed by format edits, which writes no file, the files before
// read.
func (m moves) reread(before []world) fileRead {
	if m == nil {
		return wholeFiles
	}
	return func(context.Context, *db.Queries, works.Among[works.WorkID]) ([]File, error) {
		var out []File
		for _, w := range before {
			out = append(out, w.files...)
		}
		return out, nil
	}
}
