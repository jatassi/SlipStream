package files

import (
	"context"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Scope is the Works a plan or a count covers: one Work, every Work of a
// Content Type, or the whole library. Only WorkScope, TypeScope and
// LibraryScope make one.
type Scope struct {
	kind scopeKind
	work works.WorkID
	typ  works.TypeID
}

type scopeKind uint8

const (
	oneWork scopeKind = iota + 1
	oneType
	library
)

func WorkScope(w works.WorkID) Scope { return Scope{kind: oneWork, work: w} }

func TypeScope(t works.TypeID) Scope { return Scope{kind: oneType, typ: t} }

func LibraryScope() Scope { return Scope{kind: library} }

// Moving counts the files of s the disk queue still moves, for "Renaming
// 128 files…".
func (f *Files) Moving(ctx context.Context, s Scope) (int, error) {
	var n int64
	err := f.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		if s.kind == oneWork {
			var err error
			n, err = q.CountMovingOfWork(ctx, s.work)
			return err
		}
		rows, err := q.CountMovingByWork(ctx)
		if err != nil || len(rows) == 0 {
			return err
		}
		ids := make([]works.WorkID, len(rows))
		for i, row := range rows {
			ids[i] = row.WorkID
		}
		of, err := works.SummariesOf(ctx, r, ids)
		if err != nil {
			return err
		}
		for _, row := range rows {
			if s.kind == library || of[row.WorkID].ContentType == s.typ {
				n += row.Files
			}
		}
		return nil
	})
	return int(n), err
}
