package library

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Search answers every Work whose titles match q, best first, and the rows
// of the first limit of them in the same order, from one snapshot. A nil t
// searches every Content Type.
func (l *Library) Search(ctx context.Context, _ identity.Admin, q string, limit int, t *works.TypeID) ([]works.WorkID, []Row, error) {
	var (
		ids  []works.WorkID
		rows []Row
	)
	err := l.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		if ids, err = l.cfg.Works.MatchIn(ctx, r, q, t); err != nil {
			return err
		}
		first := ids[:min(limit, len(ids))]
		if len(first) == 0 {
			return nil
		}
		listed, err := l.rowsIn(ctx, r, works.Scope{Works: first})
		if err != nil {
			return err
		}
		of := make(map[works.WorkID]Row, len(listed))
		for _, row := range listed {
			of[row.Work.ID] = row
		}
		for _, id := range first {
			rows = append(rows, of[id])
		}
		return nil
	})
	return ids, rows, err
}
