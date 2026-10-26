package library

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Adding is an add's choices. A nil choice takes its Content Type's
// default. Remember makes each given choice the default in the add's
// Write.
type Adding struct {
	Type     works.TypeID
	Key      works.Key
	Profile  *targets.ProfileID
	Root     *roots.ID
	Monitor  *bool
	Remember bool
}

// Added is the Work an add made or found. Created is false when it was
// already in the library.
type Added struct {
	Tree    works.Tree
	Created bool
}

// remembered is the defaults an add moved.
type remembered struct {
	profile bool
	roots   []roots.ID
}

// Add puts the Work a names in the library, wanting from its Profile and
// homed in its root, or answers the Work already there. Only an add that
// creates the Work remembers, so finding one moves no default.
func (l *Library) Add(ctx context.Context, by identity.Admin, a Adding) (Added, error) {
	monitor, err := l.monitor(ctx, a)
	if err != nil {
		return Added{}, err
	}
	var moved remembered
	tree, created, err := l.cfg.Works.Add(ctx, by, a.Type, a.Key, works.Adding{
		Monitor: monitor,
		Also: func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
			if err := targets.Assign(ctx, tx, w, a.Profile); err != nil {
				return err
			}
			if err := roots.Assign(ctx, tx, w, a.Root); err != nil {
				return err
			}
			if !a.Remember {
				return nil
			}
			var err error
			moved, err = remember(ctx, tx, a)
			return err
		},
	})
	if err != nil {
		return Added{}, err
	}
	if moved.profile {
		l.cfg.Targets.Publish(ctx, nil)
	}
	l.cfg.Roots.Publish(ctx, moved.roots)
	return Added{Tree: tree, Created: created}, nil
}

func (l *Library) monitor(ctx context.Context, a Adding) (bool, error) {
	if a.Monitor != nil {
		return *a.Monitor, nil
	}
	var on bool
	err := l.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		on, err = works.AddMonitorIn(ctx, r, a.Type)
		return err
	})
	return on, err
}

// remember makes each choice a gives its type's default in tx, through
// the paths Make default takes.
func remember(ctx context.Context, tx *storage.Tx[storage.Library], a Adding) (remembered, error) {
	var moved remembered
	var err error
	if a.Profile != nil {
		if moved.profile, err = targets.MakeDefaultIn(ctx, tx, *a.Profile); err != nil {
			return remembered{}, err
		}
	}
	if a.Root != nil {
		if moved.roots, err = roots.MakeDefaultIn(ctx, tx, *a.Root); err != nil {
			return remembered{}, err
		}
	}
	if a.Monitor != nil {
		if _, err = works.SetAddMonitorIn(ctx, tx, a.Type, *a.Monitor); err != nil {
			return remembered{}, err
		}
	}
	return moved, nil
}
