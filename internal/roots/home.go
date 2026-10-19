package roots

import (
	"context"
	"errors"
	"slices"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/roots/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Home is the one way a Work changes root: ErrHasFiles while work has a
// file, ErrWrongType when root is another Content Type's. A Work already
// in root stays.
func (r *Roots) Home(ctx context.Context, tx *storage.Tx[file], work works.WorkID, root ID) error {
	w, err := works.SummaryOf(ctx, tx, work)
	if err != nil {
		return err
	}
	to, err := Get(ctx, tx, root)
	if err != nil {
		return err
	}
	if to.ContentType != w.ContentType {
		return ErrWrongType
	}
	if now, err := OfWork(ctx, tx, work); err == nil && now.ID == root {
		return nil
	} else if err != nil && !errors.Is(err, ErrNoRoot) {
		return err
	}
	switch has, err := r.cfg.Contents.HasFiles(ctx, tx, work); {
	case err != nil:
		return err
	case has:
		return ErrHasFiles
	}
	return db.New(tx).HomeWork(ctx, db.HomeWorkParams{WorkID: work, RootID: root})
}

// SetWorkRoot moves work, which has no files, into root.
func (r *Roots) SetWorkRoot(ctx context.Context, _ identity.Admin, work works.WorkID, root ID) error {
	changed := []ID{root}
	err := r.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		changed = changed[:1]
		switch was, err := OfWork(ctx, tx, work); {
		case err == nil && was.ID != root:
			changed = append(changed, was.ID)
		case err != nil && !errors.Is(err, ErrNoRoot):
			return err
		}
		return r.Home(ctx, tx, work, root)
	})
	if err != nil {
		return err
	}
	r.cfg.Changed.Publish(ctx, changedOf(changed))
	return nil
}

// Assign homes w, a Work being added, in root, or in its type's default
// when root is nil, or nowhere while its type has no root.
func Assign(ctx context.Context, tx *storage.Tx[file], w works.Summary, root *ID) error {
	q := db.New(tx)
	var to Root
	var err error
	if root == nil {
		to, err = found(q.GetDefaultRoot(ctx, w.ContentType))
		if errors.Is(err, ErrNotFound) {
			return nil
		}
	} else {
		to, err = Get(ctx, tx, *root)
	}
	switch {
	case err != nil:
		return err
	case to.ContentType != w.ContentType:
		return ErrWrongType
	}
	return q.HomeWork(ctx, db.HomeWorkParams{WorkID: w.ID, RootID: to.ID})
}

// homeRootless homes every rootless Work whose type has a default root in
// that root, and answers the roots it homed a Work in.
func homeRootless(ctx context.Context, tx *storage.Tx[file]) ([]ID, error) {
	all, err := All(ctx, tx)
	if err != nil {
		return nil, err
	}
	defaults := map[works.TypeID]ID{}
	for _, root := range all {
		if root.Default {
			defaults[root.ContentType] = root.ID
		}
	}
	if len(defaults) == 0 {
		return nil, nil
	}
	q := db.New(tx)
	homed, err := q.ListHomedWorks(ctx)
	if err != nil {
		return nil, err
	}
	has := make(map[works.WorkID]bool, len(homed))
	for _, id := range homed {
		has[id] = true
	}
	summaries, err := works.Summaries(ctx, tx)
	if err != nil {
		return nil, err
	}
	var into []ID
	for _, w := range summaries {
		root, ok := defaults[w.ContentType]
		if has[w.ID] || !ok {
			continue
		}
		if err := q.HomeWork(ctx, db.HomeWorkParams{WorkID: w.ID, RootID: root}); err != nil {
			return nil, err
		}
		if !slices.Contains(into, root) {
			into = append(into, root)
		}
	}
	return into, nil
}
