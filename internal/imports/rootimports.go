package imports

import (
	"context"

	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// RootImports is roots.Inflight: the unfinished Imports into a root, or
// of ws's Works. A root Import counts only while a file of it
// lands, for one waiting in Pending imports goes with its root.
type RootImports struct{}

func (RootImports) Into(ctx context.Context, r storage.Reader[file], root roots.ID, ws []works.WorkID) (int64, error) {
	return db.New(r).CountUnfinishedInto(ctx, db.CountUnfinishedIntoParams{RootID: root, WorkIds: ws})
}

// Release unties root's finished Imports from it, so it can go and they
// stay.
func (RootImports) Release(ctx context.Context, tx *storage.Tx[file], root roots.ID) error {
	return db.New(tx).ForgetRoot(ctx, root)
}

// EndOf is library.Imports: in a Work delete's Write, before the cascade,
// each unfinished Import of ws declines every file not yet final
// work_deleted, which finishes it. A root Import keeps its waiting files
// for Pending imports, and no longer names the Work.
func (im *Imports) EndOf(ctx context.Context, tx *storage.Tx[file], ws []works.WorkID) error {
	ids, err := db.New(tx).ListUnfinishedOf(ctx, ws)
	if err != nil {
		return err
	}
	for _, id := range ids {
		_, err := im.writeIn(ctx, tx, id, &after{}, func(_ context.Context, _ *storage.Tx[file], cur *Import, _ *after) error {
			for i, f := range cur.Files {
				if f.Outcome.Final() || cur.Source == FromRoot && f.Outcome == Waiting {
					continue
				}
				cur.Files[i].Outcome, cur.Files[i].Reason, cur.Files[i].Hold, cur.Files[i].PlacedBy = Declined, &Reason{Code: WorkDeleted}, nil, ""
			}
			if cur.Source == FromRoot {
				cur.Work = works.WorkID{}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
