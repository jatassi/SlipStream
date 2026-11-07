package indexers_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

// solverNode is a Node whose solver /health responds status, or that
// can't reach it when unreachable. It passes every Test and keeps the
// solver each job carried.
type solverNode struct {
	mu          sync.Mutex
	status      int
	unreachable bool
	carried     []*searching.Solver
}

func (n *solverNode) ReachSolver(context.Context, searching.Solver) (int, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.unreachable {
		return 0, &searching.SolverUnreachable{Cause: "dial tcp 127.0.0.9:8191: connect: connection refused"}
	}
	return n.status, nil
}

func (n *solverNode) Run(_ context.Context, j searching.Job) (searching.Poll, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.carried = append(n.carried, j.Solver)
	return searching.Poll{Answer: &searching.Answer{Report: &searching.Report{Capabilities: searching.Capabilities{Modes: map[string][]string{"search": {"q"}}}}}}, nil
}

func (n *solverNode) last() *searching.Solver {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.carried[len(n.carried)-1]
}

type solverWorld struct {
	x    *indexers.Indexers
	home role.NodeID
	id   indexers.ID
	node *solverNode
}

func newSolverWorld(t *testing.T) solverWorld {
	t.Helper()
	sealer := secrettest.Sealer(t)
	lib := storagetest.Library(t, secret.Schema(), nodes.Schema(), indexers.Schema())
	w := solverWorld{home: role.NodeID{UUID: storage.NewUUID()}, id: indexers.ID{UUID: storage.NewUUID()}, node: &solverNode{status: http.StatusOK}}
	if err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if err := nodestest.Insert(ctx, tx, w.home, "seedbox"); err != nil {
			return err
		}
		return indexerstest.Torznab(ctx, tx, sealer, w.id, "Cinder", w.home, "https://site.test/torznab", "k")
	}); err != nil {
		t.Fatal(err)
	}
	roster := indexerstest.Alone(w.home)
	roster.Names = map[role.NodeID]string{w.home: "seedbox"}
	w.x = indexers.New(indexers.Config{
		Library: lib, Sealer: sealer, Roster: roster,
		Searching: func(role.NodeID) searching.Remote { return w.node },
		Types:     map[works.TypeID]indexers.Facet{indexerstest.Type: indexerstest.Facet{}},
		Changed:   discard[indexers.Changed]{}, Status: discard[indexers.StatusChanged]{}, Trackers: discard[indexers.TrackerChanged]{},
		Timings: indexers.Timings{Job: time.Minute, Poll: time.Second, Limited: time.Minute, Ladder: []time.Duration{time.Minute}},
		Log:     slog.New(slog.DiscardHandler),
	})
	return w
}

func (w solverWorld) solver(t *testing.T) indexers.Solver {
	t.Helper()
	s, err := w.x.Solver(t.Context(), identity.Admin{}, w.home)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// tested is the solver the job of a Test carried.
func (w solverWorld) tested(t *testing.T) *searching.Solver {
	t.Helper()
	if _, err := w.x.Test(t.Context(), identity.Admin{}, w.id, nil); err != nil {
		t.Fatal(err)
	}
	return w.node.last()
}

const solverAddress = "http://127.0.0.9:8191"

func TestAChallengeSolverSavesOnlyOnceItsNodeReachesIt(t *testing.T) {
	w := newSolverWorld(t)
	if s := w.solver(t); s != (indexers.Solver{Timeout: time.Minute}) {
		t.Fatalf("a new Node's solver = %+v, want none with the 60 s default", s)
	}
	const sentence = "seedbox can't reach a Challenge solver at " + solverAddress + ". Check the address and that the solver is running beside seedbox."
	for name, set := range map[string]func(){
		"a solver that responds 404": func() { w.node.status = http.StatusNotFound },
		"no solver":                  func() { w.node.unreachable = true },
	} {
		set()
		_, err := w.x.SetSolver(t.Context(), identity.Admin{}, w.home, solverAddress, nil)
		var unreachable *indexers.SolverUnreachableError
		if !errors.As(err, &unreachable) || unreachable.Sentence() != sentence {
			t.Errorf("SetSolver at %s = %v, want %q", name, err, sentence)
		}
	}
	if _, err := w.x.SetSolver(t.Context(), identity.Admin{}, w.home, "localhost:8191", nil); !errors.As(err, new(*indexers.SolverUnreachableError)) {
		t.Errorf("SetSolver of an address with no scheme = %v, want SolverUnreachableError", err)
	}
	if s, carried := w.solver(t), w.tested(t); s.Address != "" || carried != nil {
		t.Fatalf("after refused saves the solver is %+v and a job carries %+v, want none", s, carried)
	}
	if _, err := w.x.SetSolver(t.Context(), identity.Admin{}, role.NodeID{UUID: storage.NewUUID()}, solverAddress, nil); !errors.Is(err, indexers.ErrNotSearching) {
		t.Fatalf("SetSolver on a Node outside Searching = %v, want ErrNotSearching", err)
	}
}

func TestASavedChallengeSolverRidesEveryJobUntilRemoved(t *testing.T) {
	w := newSolverWorld(t)
	admin, ctx := identity.Admin{}, t.Context()
	short := int64(5)
	if _, err := w.x.SetSolver(ctx, admin, w.home, solverAddress, &short); !errors.As(err, new(*setting.RangeError)) {
		t.Errorf("SetSolver with a 5 s timeout = %v, want a RangeError", err)
	}
	timeout := int64(45)
	if s, err := w.x.SetSolver(ctx, admin, w.home, solverAddress, &timeout); err != nil || s != (indexers.Solver{Address: solverAddress, Timeout: 45 * time.Second}) {
		t.Fatalf("SetSolver of a reachable solver = %+v, %v; want it saved", s, err)
	}
	if s, carried := w.solver(t), w.tested(t); s.Address != solverAddress || carried == nil || *carried != (searching.Solver{Address: solverAddress, Timeout: 45 * time.Second}) {
		t.Fatalf("the saved solver is %+v and a job carries %+v, want the solver with its 45 s timeout", s, carried)
	}
	if _, err := w.x.SetSolver(ctx, admin, w.home, "", nil); err != nil {
		t.Fatal(err)
	}
	if s, carried := w.solver(t), w.tested(t); s.Address != "" || carried != nil {
		t.Fatalf("after removing it the solver is %+v and a job carries %+v, want none", s, carried)
	}
}
