package files

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// drafted is a Draft's scheme, which names its type's files in place of
// the stored settings.
type drafted struct {
	typ    works.TypeID
	scheme *naming.Scheme
}

// drafted is d's scheme in r, nil for no draft. A draft the save check
// refuses is its *naming.RefusedError.
func (f *Files) drafted(ctx context.Context, r storage.Reader[file], d *Draft) (*drafted, error) {
	if d == nil {
		return nil, nil
	}
	sc, err := f.cfg.Naming.DraftIn(ctx, r, d.Type, d.Patch)
	if err != nil {
		return nil, err
	}
	return &drafted{typ: d.Type, scheme: sc}, nil
}

// Named is a live file and the path the templates give it.
type Named struct {
	File ID
	// From is where the file belongs now, and To where the templates put
	// it, "" when it doesn't fit.
	From string
	To   string
	// Long is why To is "".
	Long *naming.TooLongError
}

// NamesOf is each live file of w, in file id order, with the path the
// templates give it as a plan names it, under the stored settings or
// draft over its type's. It asks no Node, so it never blocks a file for
// what sits at its new path. A draft the save check refuses is its
// *naming.RefusedError.
func (f *Files) NamesOf(ctx context.Context, _ identity.Admin, w works.WorkID, draft *Draft) ([]Named, error) {
	if err := f.known(WorkScope(w), draft); err != nil {
		return nil, err
	}
	var p worldPlan
	err := f.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		trees, among, err := WorkScope(w).trees(ctx, r)
		if err != nil {
			return err
		}
		pl, err := f.plans(ctx, r, trees, among, draft)
		if err != nil {
			return err
		}
		p = pl[0]
		return nil
	})
	if err != nil {
		return nil, err
	}
	long := map[ID]*naming.TooLongError{}
	for _, row := range p.Rows {
		long[row.File] = row.Blocked.Long
	}
	var out []Named
	for _, g := range p.w.files {
		if !g.Gone() {
			out = append(out, Named{File: g.ID, From: g.Path, To: p.names[g.ID].Path, Long: long[g.ID]})
		}
	}
	return out, nil
}

// Imagine is the path b's one candidate, a file that isn't there, would
// take in its Work under the stored settings or draft over its type's,
// named as a Kept file when it fills nothing. A name its judge refuses is
// that *NameTakenError or *naming.TooLongError, and a draft the save
// check refuses is its *naming.RefusedError.
func (f *Files) Imagine(ctx context.Context, r storage.Reader[file], b Batch, draft *Draft) (string, error) {
	d, err := f.drafted(ctx, r, draft)
	if err != nil {
		return "", err
	}
	how := importing
	how.keepFresh, how.draft = true, d
	j, err := f.judge(ctx, r, b, how, nil)
	if err != nil {
		return "", err
	}
	if refused := j.verdicts[0].Refusal; refused != nil {
		return "", refused
	}
	return j.names[j.fresh[0].ID].Path, nil
}
