package role_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	nodev1 "github.com/jatassi/slipstream/internal/pb/slipstream/node/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/role/roletest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

var timings = role.Timings{RestartMin: time.Second, RestartMax: time.Minute, Beat: 15 * time.Second}

// partitioned is how long a stalled path takes to end its session:
// keepalive, then write_timeout, plus a margin.
const partitioned = 36 * time.Second

// redialed is long enough for the Worker to re-dial and catch up.
const redialed = 2 * time.Minute

var discard = slog.New(slog.DiscardHandler)

// cluster is a Controller holding the probe's control half and a Worker,
// over linktest.
type cluster struct {
	pair                   *linktest.Pair
	lib                    *storage.DB[storage.Library]
	controls               *role.Controls
	probe                  *roletest.Control
	controllerID, workerID role.NodeID

	mu       sync.Mutex
	assigned map[role.NodeID][]role.Name
}

func (c *cluster) Self() role.NodeID { return c.controllerID }

func (c *cluster) NodeOf(key link.Key) (role.NodeID, bool) {
	switch key {
	case c.pair.Controller.Key:
		return c.controllerID, true
	case c.pair.Worker.Key:
		return c.workerID, true
	}
	return role.NodeID{}, false
}

func (c *cluster) scopeOf(_ context.Context, _ storage.Reader[storage.Library], node role.NodeID) (role.NodeScope, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return role.NodeScope{Holds: slices.Clone(c.assigned[node])}, nil
}

// assign sets node's Roles and touches its node scope, as setRoles does.
func (c *cluster) assign(node role.NodeID, roles ...role.Name) {
	c.mu.Lock()
	c.assigned[node] = roles
	c.mu.Unlock()
	c.controls.Touch(node)
}

func (c *cluster) setIntent(t *testing.T, node role.NodeID, value string) {
	t.Helper()
	if err := c.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return roletest.SetIntent(ctx, tx, node, value)
	}); err != nil {
		t.Fatal(err)
	}
}

func (c *cluster) path() *fault.Plan {
	return c.pair.Net.Path(linktest.ControllerAddr, linktest.WorkerAddr)
}

func newCluster(t *testing.T, controllerVersion, workerVersion string) *cluster {
	t.Helper()
	lockstep := http.NewServeMux()
	c := &cluster{
		pair:         linktest.New(t, linktest.Side{Version: controllerVersion, Lockstep: lockstep}, linktest.Side{Version: workerVersion}),
		lib:          storagetest.Library(t, role.LibrarySchema(), roletest.Schema()),
		probe:        roletest.Serve(),
		controllerID: role.NodeID{UUID: storage.NewUUID()},
		workerID:     role.NodeID{UUID: storage.NewUUID()},
		assigned:     map[role.NodeID][]role.Name{},
	}
	c.controls = role.NewControls(role.ControlsConfig{Library: c.lib, Nodes: c, Scope: c.scopeOf, Log: discard}, c.probe)
	lockstep.Handle(c.controls.Handler())
	return c
}

// holder is one running holder half with the probe.
type holder struct {
	*role.Holding
	probe *roletest.Probe
	stop  func()
}

func (c *cluster) startWorker(t *testing.T, local *storage.DB[storage.Local], held ...role.Held) *holder {
	t.Helper()
	probe := roletest.NewProbe(local)
	h, err := role.NewHolding(t.Context(), role.HoldingConfig{
		Local:    local,
		Upstream: role.OverLink(c.pair.Worker.Link, c.pair.Controller.Key),
		Held:     append([]role.Held{probe.Held()}, held...),
		Timings:  timings,
		Rand:     rand.New(rand.NewPCG(1, 2)), //nolint:gosec // seeded on purpose so tests replay
		Log:      discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = h.Run(ctx)
	}()
	stop := sync.OnceFunc(func() {
		cancel()
		<-done
	})
	t.Cleanup(stop)
	return &holder{Holding: h, probe: probe, stop: stop}
}

func check(t *testing.T, what, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: %s, want %s", what, got, want)
	}
}

func snapshot(t *testing.T, p *roletest.Probe) string {
	t.Helper()
	s, ok := p.Snapshot()
	if !ok {
		return "no snapshot"
	}
	return fmt.Sprintf("%q %v may start %v", s.Intent.GetValue(), s.Standing, s.MayStart())
}

func nodeFacts(c *cluster, node role.NodeID) string {
	f, ok := c.controls.NodeFacts(node)
	if !ok {
		return "no facts"
	}
	return fmt.Sprintf("assigned %v running %v restarting %v", f.GetAssigned(), f.GetRunning(), f.GetRestarting())
}

func TestAWorkerFollowsItsIntentAcrossAPartition(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := newCluster(t, "", "")
		c.setIntent(t, c.workerID, "first")
		c.assign(c.workerID, roletest.Name)
		w := c.startWorker(t, storagetest.Local(t, role.LocalSchema()))
		c.pair.Connect(t)
		synctest.Wait()
		if got := snapshot(t, w.probe); got != `"first" live may start true` {
			t.Fatalf("after connecting the probe sees %s", got)
		}
		check(t, "the Worker's facts", nodeFacts(c, c.workerID), "assigned [probe] running [probe] restarting []")
		f, _ := c.controls.NodeFacts(c.workerID)
		if v, err := c.controls.NodeVersion(t.Context(), c.workerID); err != nil || !slices.Equal(f.GetAppliedVersion(), v) {
			t.Errorf("applied version %x, want %x (%v)", f.GetAppliedVersion(), v, err)
		}
		if !w.Holds(roletest.Name) || w.Holds(role.Searching) {
			t.Errorf("Holds(probe) = %v, Holds(searching) = %v", w.Holds(roletest.Name), w.Holds(role.Searching))
		}

		c.setIntent(t, c.workerID, "second")
		c.probe.Touch(c.workerID)
		synctest.Wait()
		if got := snapshot(t, w.probe); got != `"second" live may start true` {
			t.Fatalf("after Touch the probe sees %s", got)
		}

		if _, err := c.path().Set(fault.Rule{Fault: fault.Stall{}}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(partitioned)
		synctest.Wait()
		check(t, "partitioned, the probe", snapshot(t, w.probe), `"second" unreachable may start false`)
		check(t, "partitioned, the Controller has the Worker's facts", nodeFacts(c, c.workerID), "no facts")

		c.setIntent(t, c.workerID, "third")
		c.probe.Touch(c.workerID)
		c.path().Clear("")
		time.Sleep(redialed)
		synctest.Wait()
		check(t, "back in step, the probe", snapshot(t, w.probe), `"third" live may start true`)

		c.assign(c.workerID)
		synctest.Wait()
		check(t, "unassigned, the Worker's facts", nodeFacts(c, c.workerID), "assigned [] running [] restarting []")
		if w.Holds(roletest.Name) {
			t.Error("the Worker still holds the probe once unassigned")
		}
	})
}

// Serving has no intent, so its run starts as soon as the node scope
// assigns it and stops once the scope drops it.
func TestARoleWithNoIntentRunsWhileAssigned(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := newCluster(t, "", "")
		var runs, stops int
		serving := role.Hold(role.Serving, func(ctx context.Context) error {
			runs++
			<-ctx.Done()
			stops++
			return nil
		})
		c.startWorker(t, storagetest.Local(t, role.LocalSchema()), serving)
		c.pair.Connect(t)
		c.assign(c.workerID, role.Serving)
		synctest.Wait()
		check(t, "assigned, the Worker's facts", nodeFacts(c, c.workerID), "assigned [serving] running [serving] restarting []")
		if runs != 1 || stops != 0 {
			t.Fatalf("assigned Serving ran %d times and stopped %d", runs, stops)
		}
		c.assign(c.workerID)
		synctest.Wait()
		check(t, "unassigned, the Worker's facts", nodeFacts(c, c.workerID), "assigned [] running [] restarting []")
		if runs != 1 || stops != 1 {
			t.Fatalf("unassigned Serving ran %d times and stopped %d", runs, stops)
		}
	})
}

func TestARestartedWorkerStartsFromItsCachedIntent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := newCluster(t, "", "")
		c.setIntent(t, c.workerID, "cached")
		c.assign(c.workerID, roletest.Name)
		local := storagetest.Local(t, role.LocalSchema())
		w := c.startWorker(t, local)
		c.pair.Connect(t)
		synctest.Wait()

		if _, err := c.path().Set(fault.Rule{Fault: fault.Stall{}}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(partitioned)
		w.stop()
		again := c.startWorker(t, local)
		synctest.Wait()
		check(t, "restarted while partitioned, the probe", snapshot(t, again.probe), `"cached" unreachable may start false`)
		if err := again.probe.Report(t.Context(), 7); err != nil {
			t.Errorf("report while partitioned: %v", err)
		}
	})
}

func TestAMismatchedWorkerKeepsItsLastIntent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		local := storagetest.Local(t, role.LocalSchema())
		inStep := newCluster(t, "", "")
		inStep.setIntent(t, inStep.workerID, "last")
		inStep.assign(inStep.workerID, roletest.Name)
		w := inStep.startWorker(t, local)
		inStep.pair.Connect(t)
		synctest.Wait()
		w.stop()

		mismatched := newCluster(t, "0.0.0-a", "0.0.0-b")
		again := mismatched.startWorker(t, local)
		mismatched.pair.Connect(t)
		synctest.Wait()
		check(t, "between versions, the probe", snapshot(t, again.probe), `"last" mismatched may start false`)
		check(t, "a Sync stream opened between versions", nodeFacts(mismatched, mismatched.workerID), "no facts")
	})
}

func TestScopeVersionsAreContentHashes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := newCluster(t, "", "")
		c.mu.Lock()
		c.assigned[c.workerID] = []role.Name{role.Searching, role.Downloading}
		c.mu.Unlock()
		got, err := c.controls.NodeVersion(t.Context(), c.workerID)
		if err != nil {
			t.Fatal(err)
		}
		b, err := proto.MarshalOptions{Deterministic: true}.Marshal(&nodev1.NodeIntent{Roles: []string{"searching", "downloading"}})
		if err != nil {
			t.Fatal(err)
		}
		if want := "\n\tsearching\n\x0bdownloading"; string(b) != want {
			t.Fatalf("encoding %q, want %q", b, want)
		}
		check(t, "the node-scope version", hex.EncodeToString(got), "72ea4f7debbb0aeb53da114bfe3ffa4a")
	})
}

func TestTheRoleSetsPassTheAudit(t *testing.T) {
	storagetest.AuditLibrary(t, []storage.Schema[storage.Library]{role.LibrarySchema(), roletest.Schema()})
	storagetest.AuditLocal(t, []storage.Schema[storage.Local]{role.LocalSchema()})
}
