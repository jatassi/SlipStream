package moves

import (
	"context"
	"strings"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/moves/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

// Hooks is everything an owner calls of moves. Each reads or writes only
// moves' rows and does no IO.
type Hooks struct {
	Files   files.Moves
	Library library.Moves
	Busy    func(ctx context.Context, r storage.Reader[file], root roots.ID) (int64, error)
	Copies  imports.OtherCopies
}

func (m *Moves) Hooks() Hooks {
	return Hooks{Files: FilesHooks{}, Library: libraryHooks{m}, Busy: Busy, Copies: OtherCopies}
}

// FilesHooks is files' Moves: they read only moves' rows, so an owner
// built before moves takes them.
type FilesHooks struct{}

func (FilesHooks) Reserved(ctx context.Context, r storage.Reader[file], rs []roots.ID) (map[roots.ID][]storing.Rel, error) {
	q := db.New(r)
	to, err := q.ListReservedTo(ctx, rs)
	if err != nil {
		return nil, err
	}
	out := map[roots.ID][]storing.Rel{}
	for _, row := range to {
		rel, err := storing.ParseRel(row.SitsRel)
		if err != nil {
			return nil, err
		}
		out[row.ToRootID] = append(out[row.ToRootID], rel, holdingOf(row.UnitID, int(row.Position), rel))
	}
	from, err := q.ListReservedFrom(ctx, rs)
	if err != nil {
		return nil, err
	}
	for _, row := range from {
		rel, err := storing.ParseRel(row.SitsRel)
		if err != nil {
			return nil, err
		}
		out[row.FromRootID] = append(out[row.FromRootID], rel)
	}
	return out, nil
}

func (FilesHooks) Holds(ctx context.Context, r storage.Reader[file], ws []works.WorkID) (map[works.WorkID]bool, error) {
	if len(ws) == 0 {
		return nil, nil
	}
	held, err := db.New(r).HeldWorks(ctx, ws)
	out := make(map[works.WorkID]bool, len(held))
	for _, w := range held {
		out[w] = true
	}
	return out, err
}

type libraryHooks struct{ m *Moves }

// CancelOf cancels each open unit of ws before its switch, in tx: the
// Works go in the same Write.
func (h libraryHooks) CancelOf(ctx context.Context, tx *storage.Tx[file], ws []works.WorkID) error {
	if len(ws) == 0 {
		return nil
	}
	rows, err := db.New(tx).ListOpenUnitsBeforeSwitch(ctx, ws)
	if err != nil {
		return err
	}
	us, err := unitsOf(ctx, db.New(tx), rows)
	if err != nil {
		return err
	}
	now := h.m.now()
	for _, u := range us {
		u.Stage, u.Due = Canceled{At: u.Stage.Phase()}, &now
		if err := save(ctx, tx, u); err != nil {
			return err
		}
	}
	return nil
}

func (h libraryHooks) Poke() { h.m.Poke() }

// Of is w's latest unit as the Work page shows it, nil once it moved or
// its cancel was undone.
func (h libraryHooks) Of(ctx context.Context, r storage.Reader[file], w works.WorkID) (*library.Moving, error) {
	of, err := h.m.Of(ctx, r, []works.WorkID{w})
	v, ok := of[w]
	if err != nil || !ok || v.Phase == PhaseDone || v.Phase == PhaseCanceled && v.Undone {
		return nil, err
	}
	return &library.Moving{
		Unit: v.Unit.UUID, Batch: v.Batch.UUID, Phase: string(v.Phase), Waiting: string(v.Waiting),
		ToNode: v.To.Node, ToNodeName: v.To.NodeName, Words: v.Words,
	}, nil
}

// Busy is how many units that haven't ended use root, read in r.
func Busy(ctx context.Context, r storage.Reader[file], root roots.ID) (int64, error) {
	return db.New(r).CountBusyRoot(ctx, root)
}

// OtherCopies is the bytes moves granted to same-Node copies onto volume
// of node: its lanes name the destination volume after ">".
func OtherCopies(ctx context.Context, r storage.Reader[file], node role.NodeID, volume string) (int64, error) {
	rows, err := db.New(r).ListCopyGrants(ctx, node)
	if err != nil {
		return 0, err
	}
	var n int64
	for _, row := range rows {
		if _, to, ok := strings.Cut(row.Lane, ">"); ok && to == volume {
			n += row.Bytes
		}
	}
	return n, nil
}

// Restored marks, in a restore's Write, every unit before its switch for
// Reconcile: the restored rows may be older than both roots.
func Restored(ctx context.Context, tx *storage.Tx[file]) error {
	return db.New(tx).MarkReconcile(ctx)
}
