package files

import (
	"context"
	"path"
	"slices"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

// PutBack is a fresh import the disk queue put back in its root's holding
// folder because another file already had its name.
type PutBack struct {
	File ID
	Work works.WorkID
	Root roots.ID
	// Name is the name its templates give it, which the other file has.
	Name string
}

// PutBacks is every put-back file, read in r. One that is renamed into
// its name or removed is no longer one. A file whose templates no longer
// fit it reads in its Work's rename plan instead.
func (f *Files) PutBacks(ctx context.Context, r storage.Reader[file]) ([]PutBack, error) {
	rows, err := db.New(r).ListPutBack(ctx, storing.HoldingDir+"/")
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	var ids []works.WorkID
	for _, row := range rows {
		if !slices.Contains(ids, row.WorkID) {
			ids = append(ids, row.WorkID)
		}
	}
	among := works.Only(ids)
	outlines, err := works.OutlinesIn(ctx, r, among)
	if err != nil {
		return nil, err
	}
	trees := make([]works.Bare, len(outlines))
	for i, t := range outlines {
		trees[i] = works.BareOf(t)
	}
	plans, err := f.plans(ctx, r, trees, among, nil)
	if err != nil {
		return nil, err
	}
	to := map[ID]string{}
	for _, p := range plans {
		for _, row := range p.Rows {
			to[row.File] = row.To
		}
	}
	var out []PutBack
	for _, row := range rows {
		if name := to[row.ID]; name != "" {
			out = append(out, PutBack{File: row.ID, Work: row.WorkID, Root: row.RootID, Name: path.Base(name)})
		}
	}
	return out, nil
}
