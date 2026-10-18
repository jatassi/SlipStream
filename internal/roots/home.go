package roots

import (
	"context"
	"errors"

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
	err := r.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return r.Home(ctx, tx, work, root)
	})
	if err != nil {
		return err
	}
	r.cfg.Changed.Publish(ctx, Changed{})
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
// that root, and reports whether it homed one.
func homeRootless(ctx context.Context, tx *storage.Tx[file]) (bool, error) {
	all, err := All(ctx, tx)
	if err != nil {
		return false, err
	}
	defaults := map[works.TypeID]ID{}
	for _, root := range all {
		if root.Default {
			defaults[root.ContentType] = root.ID
		}
	}
	if len(defaults) == 0 {
		return false, nil
	}
	q := db.New(tx)
	homed, err := q.ListHomedWorks(ctx)
	if err != nil {
		return false, err
	}
	has := make(map[works.WorkID]bool, len(homed))
	for _, id := range homed {
		has[id] = true
	}
	summaries, err := works.Summaries(ctx, tx)
	if err != nil {
		return false, err
	}
	made := false
	for _, w := range summaries {
		root, ok := defaults[w.ContentType]
		if has[w.ID] || !ok {
			continue
		}
		if err := q.HomeWork(ctx, db.HomeWorkParams{WorkID: w.ID, RootID: root}); err != nil {
			return false, err
		}
		made = true
	}
	return made, nil
}
