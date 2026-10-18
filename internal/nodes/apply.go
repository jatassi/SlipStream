package nodes

import (
	"context"
	"fmt"
	"strings"

	"github.com/jatassi/slipstream/internal/nodes/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// StaleError refuses a SetRoles planned on rows that changed since.
// Outcome is the same change planned on the rows now.
type StaleError struct{ Outcome Outcome }

func (*StaleError) Error() string { return "the Roles changed since this change was planned" }

// BlockedError refuses a change a guardrail blocks. Outcome's Block checks
// say why.
type BlockedError struct{ Outcome Outcome }

func (e *BlockedError) Error() string {
	var words []string
	for _, c := range e.Outcome.Checks {
		if c.Verdict == Block {
			words = append(words, c.Words)
		}
	}
	return strings.Join(words, " ")
}

// RoleBoard is every live Node's Roles and the plan's verdict on every
// drag, from one Situation and the Board it read.
type RoleBoard struct {
	Situation
	Board    *Board
	Gestures []Gesture
}

func (s *Service) RoleBoard(ctx context.Context) (RoleBoard, error) {
	board := s.Board()
	var sit Situation
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		sit, err = s.situation(ctx, r, board)
		return err
	})
	if err != nil {
		return RoleBoard{}, fmt.Errorf("read the Role board: %w", err)
	}
	return RoleBoard{Situation: sit, Board: board, Gestures: Gestures(sit)}, nil
}

// Preview plans c on the rows now. It refuses what Apply refuses before
// the guardrails: a Node with no live row and a stale Basis.
func (s *Service) Preview(ctx context.Context, c Change) (Outcome, error) {
	var out Outcome
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		_, out, err = s.plan(ctx, r, c)
		return err
	})
	return out, err
}

// Apply plans c again inside one Write and stores its Outcome: the Role
// rows, the forfeits, the portable work each owner moves to its heir and
// the tombstone of a removed Node. A stale Basis or a block refuses c, and
// an Outcome that stores nothing new writes nothing. After the commit a
// removed Node's key is refused and its session closed, the roster reads
// the rows again, each other Node whose rows changed is sent its new
// intent, and Apply waits until those in step apply it or ctx ends.
func (s *Service) Apply(ctx context.Context, c Change) (Outcome, error) {
	var (
		out     Outcome
		touched []ID
	)
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		sit, planned, err := s.plan(ctx, tx, c)
		out = planned
		if err != nil {
			return err
		}
		if out.Blocked() {
			return &BlockedError{Outcome: out}
		}
		touched, err = s.store(ctx, tx, sit, out)
		return err
	})
	if err != nil {
		return out, err
	}
	if gone, ok := out.leaves(); ok {
		s.forget(ctx, gone)
	} else if len(touched) == 0 {
		return out, nil
	}
	if err := s.reload(ctx); err != nil {
		return out, err
	}
	s.controls.Touch(touched...)
	for _, id := range touched {
		if err := s.AwaitDelivery(ctx, id); err != nil {
			return out, err
		}
	}
	return out, nil
}

// plan is c planned on r. It refuses a Node r has no live row for, and a
// SetRoles planned on another Basis.
func (s *Service) plan(ctx context.Context, r storage.Reader[file], c Change) (Situation, Outcome, error) {
	sit, err := s.situation(ctx, r, s.Board())
	if err != nil {
		return Situation{}, Outcome{}, err
	}
	for _, id := range c.nodes() {
		if !sit.has(id) {
			return sit, Outcome{}, ErrNotFound
		}
	}
	out := Plan(sit, c)
	if set, ok := c.(SetRoles); ok && set.Basis != out.Basis {
		return sit, out, &StaleError{Outcome: out}
	}
	return sit, out, nil
}

// store writes out over sit: each Node's Role rows and forfeits, the work
// each Moves effect names through its Role's owner, then the tombstone of
// the Node it removes. It returns every other Node whose rows changed.
func (s *Service) store(ctx context.Context, tx *storage.Tx[file], sit Situation, out Outcome) ([]ID, error) {
	q := db.New(tx)
	now := storage.Now()
	gone, leaves := out.leaves()
	var touched []ID
	for _, seat := range sit.Seats {
		changed, err := storeSeat(ctx, q, now, seat, out.Next[seat.ID], out.Forfeit[seat.ID])
		if err != nil {
			return nil, err
		}
		if changed && (!leaves || seat.ID != gone) {
			touched = append(touched, seat.ID)
		}
	}
	for _, e := range out.Effects {
		if m, ok := e.(Moves); ok {
			if err := s.owners[m.Role].Unassigned(ctx, tx, m.From.ID, m.To.ID); err != nil {
				return nil, fmt.Errorf("move %s work from node %s to %s: %w", m.Role, m.From.ID, m.To.ID, err)
			}
		}
	}
	if leaves {
		if err := entomb(ctx, q, gone); err != nil {
			return nil, err
		}
	}
	return touched, nil
}

// storeSeat writes seat's Role rows and forfeits as next and forfeit, and
// reports whether they changed.
func storeSeat(ctx context.Context, q *db.Queries, now storage.Time, seat Seat, next, forfeit role.Set) (bool, error) {
	if next == seat.Assigned && forfeit == seat.Forfeit {
		return false, nil
	}
	writes := []struct {
		roles role.Set
		write func(role.Name) error
	}{
		{seat.Assigned &^ next, func(r role.Name) error {
			return q.DeleteRole(ctx, db.DeleteRoleParams{NodeID: seat.ID, Role: r})
		}},
		{next &^ seat.Assigned, func(r role.Name) error {
			return q.InsertRole(ctx, db.InsertRoleParams{NodeID: seat.ID, Role: r, AssignedAt: now})
		}},
		{seat.Forfeit &^ forfeit, func(r role.Name) error {
			return q.DeleteForfeit(ctx, db.DeleteForfeitParams{NodeID: seat.ID, Role: r})
		}},
		{forfeit &^ seat.Forfeit, func(r role.Name) error {
			return q.InsertForfeit(ctx, db.InsertForfeitParams{NodeID: seat.ID, Role: r, DeclaredAt: now})
		}},
	}
	for _, w := range writes {
		for _, r := range w.roles.Names() {
			if err := w.write(r); err != nil {
				return false, err
			}
		}
	}
	return true, nil
}
