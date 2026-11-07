package library

import (
	"context"
	"errors"

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
	Type    works.TypeID
	Key     works.Key
	Profile *targets.ProfileID
	Root    *roots.ID
	// Monitor is the new Work's monitoring choice.
	Monitor *works.Choice
	// LevelFolders is the new Work's level_folders.
	LevelFolders *bool
	Remember     bool
}

// errNotImplemented answers a monitoring choice the add can't apply yet:
// a preset other than All or None, Include, or a Monitor new override.
var errNotImplemented = errors.New("not implemented")

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
	if a.Remember && a.LevelFolders != nil {
		return Added{}, errNotImplemented
	}
	folders := true
	if a.LevelFolders != nil {
		folders = *a.LevelFolders
	}
	var moved remembered
	tree, created, err := l.cfg.Works.Add(ctx, by, a.Type, a.Key, works.Adding{
		Monitor:      monitor,
		LevelFolders: folders,
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
			moved, err = remember(ctx, tx, a, monitor)
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

// monitor is whether a new Work's Items are all monitored: a's All or
// None, else the type's add default.
func (l *Library) monitor(ctx context.Context, a Adding) (bool, error) {
	if c := a.Monitor; c != nil {
		switch {
		case c.Asides != nil && *c.Asides, c.New != nil:
			return false, errNotImplemented
		case c.Preset == works.PresetAll:
			return true, nil
		case c.Preset == works.PresetNone:
			return false, nil
		}
		return false, errNotImplemented
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
func remember(ctx context.Context, tx *storage.Tx[storage.Library], a Adding, monitor bool) (remembered, error) {
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
		if _, err = works.SetAddMonitorIn(ctx, tx, a.Type, monitor); err != nil {
			return remembered{}, err
		}
	}
	return moved, nil
}
