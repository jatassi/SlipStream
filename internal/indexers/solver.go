package indexers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

// ChallengeTimeout is how long a Node's Challenge solver may take to clear
// a page, as Prowlarr allows.
var ChallengeTimeout = setting.Duration{
	Key: "searching.challengeTimeout", Label: "Challenge solver timeout",
	Default: time.Minute, Min: 10 * time.Second, Max: 3 * time.Minute, Step: time.Second,
}

// Solver is a Node's Challenge solver, a service that speaks FlareSolverr's
// API. Address is "" when the Node has none.
type Solver struct {
	Address string
	Timeout time.Duration
}

// SolverUnreachableError is a Challenge solver that the Node named Node
// couldn't reach at Address, or that responded to GET /health with
// another status than 200.
type SolverUnreachableError struct {
	Node    string
	Address string
	Err     error
}

func (e *SolverUnreachableError) Error() string {
	return "reach the Challenge solver: " + e.Err.Error()
}

func (e *SolverUnreachableError) Unwrap() error { return e.Err }

// Solver is node's Challenge solver. It fails with ErrNotSearching for a
// Node that doesn't stand in Searching.
func (x *Indexers) Solver(ctx context.Context, _ identity.Admin, node role.NodeID) (Solver, error) {
	if !standsInSearching(x.cfg.Roster, node) {
		return Solver{}, ErrNotSearching
	}
	var s Solver
	err := x.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		s, err = solverOn(ctx, r, node)
		return err
	})
	return s, err
}

// SetSolver saves node's Challenge solver at address, with a timeout of
// timeout seconds or the default for nil, once node reached GET
// {address}/health and the solver responded 200. An empty address removes
// it. It fails with *setting.RangeError for a timeout out of range,
// *SolverUnreachableError, or *UnavailableError when node couldn't take
// the call.
func (x *Indexers) SetSolver(ctx context.Context, _ identity.Admin, node role.NodeID, address string, timeout *int64) (Solver, error) {
	if !standsInSearching(x.cfg.Roster, node) {
		return Solver{}, ErrNotSearching
	}
	stored, err := ChallengeTimeout.Parse(timeout)
	if err != nil {
		return Solver{}, err
	}
	s := Solver{Address: address, Timeout: ChallengeTimeout.Value(stored.Override())}
	if address == "" {
		return Solver{Timeout: ChallengeTimeout.Default}, x.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
			return db.New(tx).DeleteSolver(ctx, node)
		})
	}
	if u, err := url.Parse(address); err != nil || u.Scheme != "http" && u.Scheme != "https" || u.Host == "" {
		return Solver{}, &SolverUnreachableError{Node: x.cfg.Roster.Name(node), Address: address, Err: errors.New("it isn't an http or https address")}
	}
	status, err := x.cfg.Searching(node).ReachSolver(ctx, searching.Solver{Address: s.Address, Timeout: s.Timeout})
	var unreachable *searching.SolverUnreachable
	switch {
	case errors.As(err, &unreachable):
		return Solver{}, &SolverUnreachableError{Node: x.cfg.Roster.Name(node), Address: address, Err: err}
	case err != nil:
		return Solver{}, unavailable(err)
	case status != http.StatusOK:
		return Solver{}, &SolverUnreachableError{Node: x.cfg.Roster.Name(node), Address: address, Err: fmt.Errorf("GET /health responded with status %d", status)}
	}
	return s, x.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).PutSolver(ctx, db.PutSolverParams{NodeID: node, SolverAddress: address, SolverTimeoutSeconds: stored.Override()})
	})
}

// solverOn is node's Challenge solver as r reads it.
func solverOn(ctx context.Context, r storage.Reader[file], node role.NodeID) (Solver, error) {
	row, err := db.New(r).GetSolver(ctx, node)
	if errors.Is(err, storage.ErrNoRows) {
		return Solver{Timeout: ChallengeTimeout.Default}, nil
	}
	if err != nil {
		return Solver{}, fmt.Errorf("read node %s's Challenge solver: %w", node, err)
	}
	return Solver{Address: row.SolverAddress, Timeout: ChallengeTimeout.Value(row.SolverTimeoutSeconds)}, nil
}

// onWire is s as a job carries it, nil for no solver.
func (s Solver) onWire() *searching.Solver {
	if s.Address == "" {
		return nil
	}
	return &searching.Solver{Address: s.Address, Timeout: s.Timeout}
}

// standsInSearching reports whether node stands in Searching: assigned it
// in any state, or covering it.
func standsInSearching(r Roster, node role.NodeID) bool {
	if slices.Contains(r.Holding(role.Searching), node) {
		return true
	}
	picked, ok := r.Pick(role.Searching, node)
	return ok && picked == node
}
