package nodes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/jatassi/slipstream/internal/nodes/internal/db"
	"github.com/jatassi/slipstream/internal/nodesystem"
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

// Preview plans c on the rows now, with reach tests when test is set. It
// refuses what Apply refuses before the guardrails: a Node with no live
// row and a stale Basis.
func (s *Service) Preview(ctx context.Context, c Change, test bool) (Outcome, error) {
	var reach map[NodeRole]ReachResult
	if test {
		var err error
		if reach, err = s.tests(ctx, c); err != nil {
			return Outcome{}, err
		}
	}
	var out Outcome
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		_, out, err = s.plan(ctx, r, c, reach)
		return err
	})
	return out, err
}

// tests runs, at once, the reach test of each Role c adds to a Node in
// step whose owner has a target, each for at most timings.nodes.reach.
func (s *Service) tests(ctx context.Context, c Change) (map[NodeRole]ReachResult, error) {
	type due struct {
		at     NodeRole
		target nodesystem.Target
	}
	var tests []due
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		_, out, err := s.plan(ctx, r, c, nil)
		if err != nil {
			return err
		}
		for _, at := range out.Untested {
			t, _, err := s.owners[at.Role].Reach(ctx, r)
			if err != nil {
				return fmt.Errorf("read the %s target: %w", at.Role, err)
			}
			tests = append(tests, due{at, t})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	results := make([]ReachResult, len(tests))
	var wg sync.WaitGroup
	for i, d := range tests {
		wg.Go(func() { results[i] = s.reach(ctx, d.at.Node, d.target) })
	}
	wg.Wait()
	out := make(map[NodeRole]ReachResult, len(tests))
	for i, d := range tests {
		out[d.at] = results[i]
	}
	return out, nil
}

// reach is a reach test from node to t, for at most timings.nodes.reach.
// A test the Node never answered fails with why.
func (s *Service) reach(ctx context.Context, node ID, t nodesystem.Target) ReachResult {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.Timings.Reach)
	defer cancel()
	res := ReachResult{Target: t.String()}
	got, err := s.cfg.Reach(ctx, node, t)
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		res.Err = "timed out after " + s.cfg.Timings.Reach.String()
	case err != nil:
		res.Err = err.Error()
	default:
		res.OK, res.Millis, res.Err = got.OK, got.Took.Milliseconds(), got.Err
	}
	return res
}

// Apply runs the reach tests c needs, then plans c again inside one Write
// and stores its Outcome: the Role rows, the forfeits and the portable
// work each owner moves to its heir. A stale Basis or a block refuses c,
// and an Outcome that stores nothing new writes nothing. After the commit
// the roster reads the rows again, each Node whose rows changed is sent
// its new intent, and Apply waits until those in step apply it or ctx
// ends.
func (s *Service) Apply(ctx context.Context, c Change) (Outcome, error) {
	reach, err := s.tests(ctx, c)
	if err != nil {
		return Outcome{}, err
	}
	var (
		out     Outcome
		touched []ID
	)
	err = s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		sit, planned, err := s.plan(ctx, tx, c, reach)
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
	if len(touched) == 0 {
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

// plan is c planned on r with the reach tests run for it. It refuses a
// Node r has no live row for, and a SetRoles planned on another Basis.
func (s *Service) plan(ctx context.Context, r storage.Reader[file], c Change, reach map[NodeRole]ReachResult) (Situation, Outcome, error) {
	sit, err := s.situation(ctx, r, s.Board())
	if err != nil {
		return Situation{}, Outcome{}, err
	}
	sit.Reach = reach
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

// store writes out over sit: each Node's Role rows and forfeits, then the
// work each Moves effect names through its Role's owner. It returns every
// Node whose rows changed.
func (s *Service) store(ctx context.Context, tx *storage.Tx[file], sit Situation, out Outcome) ([]ID, error) {
	q := db.New(tx)
	now := storage.Now()
	var touched []ID
	for _, seat := range sit.Seats {
		next, forfeit := out.Next[seat.ID], out.Forfeit[seat.ID]
		if next == seat.Assigned && forfeit == seat.Forfeit {
			continue
		}
		touched = append(touched, seat.ID)
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
					return nil, err
				}
			}
		}
	}
	for _, e := range out.Effects {
		if m, ok := e.(Moves); ok {
			if err := s.owners[m.Role].Unassigned(ctx, tx, m.From.ID, m.To.ID); err != nil {
				return nil, fmt.Errorf("move %s work from node %s to %s: %w", m.Role, m.From.ID, m.To.ID, err)
			}
		}
	}
	return touched, nil
}
