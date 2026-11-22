package files

import (
	"context"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

// Moves is what files asks of S48 (Moves). NoMoves when nil.
type Moves interface {
	// Reserved is, per root of rs, every path of a unit a scan or the
	// held-folder sweep must not claim: placed and holding paths in its
	// destination, cleanup paths in its source.
	Reserved(ctx context.Context, r storage.Reader[file], rs []roots.ID) (map[roots.ID][]storing.Rel, error)
	// Holds is which of ws have a unit before its switch. The disk queue
	// starts no rename of their files, and finishes a ring already in
	// holding, so a unit's renames wait always ends.
	Holds(ctx context.Context, r storage.Reader[file], ws []works.WorkID) (map[works.WorkID]bool, error)
}

// NoMoves is the Moves of a library nothing moves.
type NoMoves struct{}

func (NoMoves) Reserved(context.Context, storage.Reader[file], []roots.ID) (map[roots.ID][]storing.Rel, error) {
	return nil, nil
}

func (NoMoves) Holds(context.Context, storage.Reader[file], []works.WorkID) (map[works.WorkID]bool, error) {
	return nil, nil
}

// Homes is the Move permit of roots: *roots.Roots.
type Homes interface {
	HomeMoved(ctx context.Context, tx *storage.Tx[file], work works.WorkID, to roots.ID) error
}

// Sitting is one live library file of a Work and where its bytes sit.
// Ringing is a file whose bytes sit in holding mid-ring.
type Sitting struct {
	Work    works.WorkID
	Root    roots.ID
	Path    storing.Rel
	Sits    storing.Rel
	Ringing bool
	Size    int64
	Seen    *storing.Seen
}

// Sits is where the bytes of each live file of ws sit, read in r.
func Sits(ctx context.Context, r storage.Reader[file], ws []works.WorkID) (map[ID]Sitting, error) {
	rows, err := db.New(r).ListSitsOfWorks(ctx, ws)
	if err != nil {
		return nil, err
	}
	out := make(map[ID]Sitting, len(rows))
	for _, row := range rows {
		if !row.GoneAt.IsZero() {
			continue
		}
		s, err := sittingOf(row)
		if err != nil {
			return nil, err
		}
		out[row.ID] = s
	}
	return out, nil
}

func sittingOf(row db.ListSitsOfWorksRow) (Sitting, error) {
	at, now, _, err := placesOf(row.ID, row.Path, row.Sits, row.Ring)
	if err != nil {
		return Sitting{}, fmt.Errorf("file %s: %w", row.ID, err)
	}
	return Sitting{
		Work: row.WorkID, Root: row.RootID, Path: at, Sits: now, Size: row.Size,
		Ringing: !row.Ring.IsZero() && isHeld(now.String()),
		Seen:    seenOf(row.ModifiedAt, row.NodeFileID, row.NodeVolumeID),
	}, nil
}

// SwitchIn is one Work's switch to another root: the rows a Move placed.
type SwitchIn struct {
	Work     works.WorkID
	From, To roots.ID
	Files    []Switched
	// GoneBefore is the unit's last final pass. A row a scan marked gone
	// before it is forgotten; one marked gone since is a difference.
	GoneBefore storage.Time
	// Restored is a restore's reconcile: rows gone at the source still
	// switch.
	Restored bool
}

// Switched is one placed file: its row as the Move copied it, and the
// destination's Seen.
type Switched struct {
	ID     ID
	Size   int64
	Seen   storing.Seen
	Sits   storing.Rel
	Placed storing.Seen
}

// StaleError lists what differs from a SwitchIn; nothing was written.
type StaleError struct{ Added, Changed, Gone []ID }

func (e *StaleError) Error() string {
	return fmt.Sprintf("the Work's files changed since the Move placed them: %d added, %d changed, %d gone", len(e.Added), len(e.Changed), len(e.Gone))
}

// Switch moves the Work's file rows to root To in tx. It refuses
// *StaleError when the rows differ from in, then has roots home the Work
// in To, sets each row's root and Seen, keeps its path and any queued
// rename, and tells Subtitles and the Media Servers one moved change per
// row.
func (f *Files) Switch(ctx context.Context, tx *storage.Tx[file], in SwitchIn) error {
	q := db.New(tx)
	rows, err := q.ListSitsOfWorks(ctx, []works.WorkID{in.Work})
	if err != nil {
		return err
	}
	if err := stale(rows, in); err != nil {
		return err
	}
	if err := f.cfg.Homes.HomeMoved(ctx, tx, in.Work, in.To); err != nil {
		return err
	}
	facts := make([]fact, 0, len(in.Files))
	for _, s := range in.Files {
		modified, file, volume := seenColumns(&s.Placed)
		if err := q.SwitchFile(ctx, db.SwitchFileParams{RootID: in.To, ModifiedAt: modified, NodeFileID: file, NodeVolumeID: volume, ID: s.ID}); err != nil {
			return err
		}
		facts = append(facts, fact{file: s.ID, root: in.To, fromRoot: in.From, from: s.Sits.String(), to: s.Sits.String()})
	}
	return f.tell(ctx, tx, facts)
}

// stale compares rows, the Work's rows now, with in.
func stale(rows []db.ListSitsOfWorksRow, in SwitchIn) error {
	var e StaleError
	planned := make(map[ID]Switched, len(in.Files))
	for _, s := range in.Files {
		planned[s.ID] = s
	}
	seen := map[ID]bool{}
	for _, row := range rows {
		s, ok := planned[row.ID]
		gone := !row.GoneAt.IsZero()
		switch {
		case gone && ok && !in.Restored:
			e.Gone = append(e.Gone, row.ID)
		case gone && !ok && row.GoneAt.Compare(in.GoneBefore) > 0:
			e.Gone = append(e.Gone, row.ID)
		case gone:
		case !ok:
			e.Added = append(e.Added, row.ID)
		default:
			seen[row.ID] = true
			if sat, err := sittingOf(row); err != nil || !sameBytes(sat, in.From, s) {
				e.Changed = append(e.Changed, row.ID)
			}
		}
	}
	for id := range planned {
		if !seen[id] && !slices.Contains(e.Gone, id) && !slices.Contains(e.Changed, id) {
			e.Gone = append(e.Gone, id)
		}
	}
	if len(e.Added)+len(e.Changed)+len(e.Gone) == 0 {
		return nil
	}
	for _, ids := range [][]ID{e.Added, e.Changed, e.Gone} {
		slices.SortFunc(ids, func(a, b ID) int { return a.Compare(b.UUID) })
	}
	return &e
}

func sameBytes(row Sitting, from roots.ID, s Switched) bool {
	return row.Root == from && row.Size == s.Size && row.Seen != nil && row.Seen.Same(s.Seen) && row.Sits == s.Sits
}
