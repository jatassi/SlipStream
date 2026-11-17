package library

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/searches"
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
	// Preset is the monitoring preset the new Work's Items follow.
	Preset *works.Preset
	// IncludeAsides adds the Items of aside levels to what Preset monitors.
	IncludeAsides *bool
	LevelFolders  *bool
	// Search is what a new Work searches once its preset applies; nil
	// searches nothing. Remember makes it the type's default.
	Search   *works.SearchScope
	Remember bool
	By       works.AddedBy
	// Announce records Works added. Only the add flow sets it, so Pending
	// imports' Add and a Library scan's adopt record nothing.
	Announce bool
	// Attach runs in the add's Write once the Work has its Profile and
	// root and before its preset applies, so a preset that reads files
	// (missing, existing) counts the files it attaches.
	Attach func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error
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

// Add puts the Work a names in the library, wanting from its Profile,
// homed in its root and monitored as its preset says, or answers the Work
// already there. Only an add that creates the Work remembers, so finding
// one moves no default.
func (l *Library) Add(ctx context.Context, by identity.Admin, a Adding) (Added, error) {
	var defaults works.AddDefaults
	err := l.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		defaults, err = works.AddDefaultsIn(ctx, r, a.Type)
		return err
	})
	if err != nil {
		return Added{}, err
	}
	choice, folders := defaults.Monitor, defaults.LevelFolders
	if a.Preset != nil {
		choice.Preset = *a.Preset
	}
	if a.IncludeAsides != nil {
		choice.Asides = a.IncludeAsides
	}
	if a.LevelFolders != nil {
		folders = *a.LevelFolders
	}
	var moved remembered
	tree, created, err := l.cfg.Works.Add(ctx, by, a.Type, a.Key, works.Adding{
		LevelFolders: folders,
		By:           a.By,
		Also: func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
			var err error
			if moved, err = l.settleIn(ctx, tx, w, a, choice); err != nil || !a.Announce {
				return err
			}
			return l.announceAdded(ctx, tx, by, w.ID)
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

// settleIn gives new Work w a's Profile, root and files, applies choice,
// queues what a searches and remembers a's choices, in the add's Write.
func (l *Library) settleIn(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary, a Adding, choice works.Choice) (remembered, error) {
	if err := targets.Assign(ctx, tx, w, a.Profile); err != nil {
		return remembered{}, err
	}
	if err := roots.Assign(ctx, tx, w, a.Root); err != nil {
		return remembered{}, err
	}
	if a.Attach != nil {
		if err := a.Attach(ctx, tx, w); err != nil {
			return remembered{}, err
		}
	}
	if _, err := l.applyIn(ctx, tx, works.Only([]works.WorkID{w.ID}), choice); err != nil {
		return remembered{}, err
	}
	if a.Search != nil && *a.Search != works.SearchNone {
		if err := l.queueAdd(ctx, tx, w.ID, *a.Search); err != nil {
			return remembered{}, err
		}
	}
	if !a.Remember {
		return remembered{}, nil
	}
	return remember(ctx, tx, a)
}

// queueAdd queues the Items of Work id that s searches, in the add's
// Write. A scope that picks nothing queues nothing.
func (l *Library) queueAdd(ctx context.Context, tx *storage.Tx[storage.Library], id works.WorkID, s works.SearchScope) error {
	among := works.Only([]works.WorkID{id})
	trees, err := works.OutlinesIn(ctx, tx, among)
	if err != nil {
		return err
	}
	var facts map[works.WorkID]works.Facts
	if s.NeedsFacts() {
		if facts, err = l.cfg.Files.ItemFacts(ctx, tx, among); err != nil {
			return err
		}
	}
	items := s.ItemsOf(trees[0], facts[id])
	if len(items) == 0 {
		return nil
	}
	return l.cfg.Queue.Queue(ctx, tx, searches.Add, searches.Ask{Work: id, Items: items})
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
	return moved, works.RememberIn(ctx, tx, a.Type, works.Remembering{Preset: a.Preset, Asides: a.IncludeAsides, LevelFolders: a.LevelFolders, Search: a.Search})
}
