package roots

import (
	"context"
	"errors"
	"fmt"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// ErrWorksToType is a root's Works sent to a root of another Content Type.
var ErrWorksToType = errors.New("the root's Works can go only to a root of their Content Type")

// InUseError is a root that can't go while it holds files, doomed ones
// counted, or Grabs are in flight into it.
type InUseError struct{ Files, Grabs int64 }

func (e *InUseError) Error() string {
	return fmt.Sprintf("the root holds %d files and %d Grabs in flight", e.Files, e.Grabs)
}

// Delete takes root id out of the library and leaves its folder and
// everything in it on disk. The type's default can't go while another root
// of its type exists. Its Works go to worksTo, another root of its type,
// which they need while one exists; with the type's last root they become
// rootless. Its finished Grabs keep their history without it.
func (r *Roots) Delete(ctx context.Context, _ identity.Admin, id ID, worksTo *ID) error {
	changed := []ID{id}
	var node role.NodeID
	err := r.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		root, err := Get(ctx, tx, id)
		if err != nil {
			return err
		}
		node = root.Node
		q := db.New(tx)
		homed, err := q.ListWorksOfRoot(ctx, id)
		if err != nil {
			return err
		}
		if err := r.inUse(ctx, tx, id, homed); err != nil {
			return err
		}
		to, err := destination(ctx, tx, root, len(homed) > 0, worksTo)
		if err != nil {
			return err
		}
		if to != nil && len(homed) > 0 {
			if err := r.homeAll(ctx, tx, homed, *to); err != nil {
				return err
			}
			changed = append(changed, *to)
		}
		if err := r.cfg.Inflight.Release(ctx, tx, id); err != nil {
			return err
		}
		return q.DeleteRoot(ctx, id)
	})
	if err != nil {
		return err
	}
	r.cfg.Changed.Publish(ctx, changedOf(changed))
	r.cfg.Storage.Touch(node)
	return nil
}

func (r *Roots) homeAll(ctx context.Context, tx *storage.Tx[file], ws []works.WorkID, root ID) error {
	for _, w := range ws {
		if err := r.Home(ctx, tx, w, root); err != nil {
			return err
		}
	}
	return nil
}

// inUse is an *InUseError while root id holds files or a Grab of homed,
// its Works, or into it is in flight.
func (r *Roots) inUse(ctx context.Context, tx *storage.Tx[file], id ID, homed []works.WorkID) error {
	use, err := r.cfg.Contents.Use(ctx, tx, []ID{id})
	if err != nil {
		return err
	}
	grabs, err := r.cfg.Inflight.Into(ctx, tx, id, homed)
	if err != nil {
		return err
	}
	if files := use[id].Files; files > 0 || grabs > 0 {
		return &InUseError{Files: files, Grabs: grabs}
	}
	return nil
}

// destination is where root's Works go when it's deleted: worksTo, which
// must be another root of its type, or nowhere while it's its type's last.
func destination(ctx context.Context, tx *storage.Tx[file], root Root, hasWorks bool, worksTo *ID) (*ID, error) {
	of, err := db.New(tx).CountRootsOf(ctx, root.ContentType)
	if err != nil {
		return nil, err
	}
	others := of > 1
	switch {
	case root.Default && others:
		return nil, ErrIsDefault
	case worksTo == nil && hasWorks && others:
		return nil, ErrWorksTo
	case worksTo == nil:
		return nil, nil
	}
	to, err := Get(ctx, tx, *worksTo)
	switch {
	case errors.Is(err, ErrNotFound):
		return nil, fmt.Errorf("worksTo: %w", err)
	case err != nil:
		return nil, err
	case to.ID == root.ID:
		return nil, ErrWorksTo
	case to.ContentType != root.ContentType:
		return nil, ErrWorksToType
	}
	return &to.ID, nil
}
