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
	to, err := Get(ctx, tx, root)
	if err != nil {
		return err
	}
	_, _, err = r.home(ctx, tx, work, to)
	return err
}

// home is Home into to. It answers whether work moved and the root it
// left, zero when it was rootless.
func (r *Roots) home(ctx context.Context, tx *storage.Tx[file], work works.WorkID, to Root) (moved bool, left ID, err error) {
	w, err := works.SummaryOf(ctx, tx, work)
	if err != nil {
		return false, ID{}, err
	}
	if to.ContentType != w.ContentType {
		return false, ID{}, ErrWrongType
	}
	switch now, err := OfWork(ctx, tx, work); {
	case err == nil && now.ID == to.ID:
		return false, ID{}, nil
	case err == nil:
		left = now.ID
	case !errors.Is(err, ErrNoRoot):
		return false, ID{}, err
	}
	switch has, err := r.cfg.Contents.HasFiles(ctx, tx, work); {
	case err != nil:
		return false, ID{}, err
	case has:
		return false, ID{}, ErrHasFiles
	}
	return true, left, db.New(tx).HomeWork(ctx, db.HomeWorkParams{WorkID: work, RootID: to.ID})
}

// HomeAll homes each of ws in root in one Write, ErrNotFound when no root
// has id root, and publishes one roots.changed naming root and each root
// a Work left. moved is each Work that changed root. refused holds each
// Work Home refuses, which stays where it is: works.ErrNoWork,
// ErrHasFiles or ErrWrongType. A Work already in root is in neither.
func (r *Roots) HomeAll(ctx context.Context, _ identity.Admin, ws []works.WorkID, root ID) (moved []works.WorkID, refused map[works.WorkID]error, err error) {
	var changed []ID
	err = r.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		moved, refused, changed = nil, map[works.WorkID]error{}, []ID{root}
		to, err := Get(ctx, tx, root)
		if err != nil {
			return err
		}
		for _, w := range ws {
			went, left, err := r.home(ctx, tx, w, to)
			switch {
			case errors.Is(err, works.ErrNoWork), errors.Is(err, ErrHasFiles), errors.Is(err, ErrWrongType):
				refused[w] = err
			case err != nil:
				return err
			case went:
				moved = append(moved, w)
				if !left.IsZero() && !slices.Contains(changed, left) {
					changed = append(changed, left)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	if len(moved) > 0 {
		r.Publish(ctx, changed)
	}
	return moved, refused, nil
}

// Publish sends one roots.changed naming ids, none when ids is empty: for
// a composer that changed what the roots hold, such as a delete of their
// Works.
func (r *Roots) Publish(ctx context.Context, ids []ID) {
	if len(ids) > 0 {
		r.cfg.Changed.Publish(ctx, changedOf(ids))
	}
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
