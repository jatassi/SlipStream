package files

import (
	"context"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Scope is the Works a count covers: one Work, or every Work of a Content
// Type. Only WorkScope and TypeScope make one.
type Scope struct {
	work works.WorkID
	typ  works.TypeID
}

func WorkScope(w works.WorkID) Scope { return Scope{work: w} }

func TypeScope(t works.TypeID) Scope { return Scope{typ: t} }

// Moving counts the files of s the disk queue still moves, for "Renaming
// 128 files…".
func (f *Files) Moving(ctx context.Context, s Scope) (int, error) {
	var n int64
	err := f.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		if !s.work.IsZero() {
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
			if of[row.WorkID].ContentType == s.typ {
				n += row.Files
			}
		}
		return nil
	})
	return int(n), err
}
