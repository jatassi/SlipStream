package nodes_test

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodesystem"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// paired is c with a Worker box paired and out of session.
func (c *controller) paired(t *testing.T) nodes.ID {
	t.Helper()
	w := c.worker(t, "box", "127.0.0.3")
	_, id := c.pair(t, w, c.mint(t), "box")
	w.Drop(c.node.Key)
	synctest.Wait()
	return id
}

func (c *controller) rolesOf(t *testing.T, id nodes.ID) []role.Name {
	t.Helper()
	n, err := c.svc.Get(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	return roleNames(n.Roles)
}

func TestAStaleBasisIsRefusedWithTheFreshPlan(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		box := c.paired(t)
		before, err := c.svc.RoleBoard(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		c.apply(t, nodes.Roles{box: role.Of(role.Searching)})
		late := nodes.SetRoles{Basis: before.Basis(), Want: nodes.Roles{box: role.Of(role.Downloading)}}
		for name, call := range map[string]func() (nodes.Outcome, error){
			"Preview": func() (nodes.Outcome, error) { return c.svc.Preview(t.Context(), late, false) },
			"Apply":   func() (nodes.Outcome, error) { return c.svc.Apply(t.Context(), late) },
		} {
			_, err := call()
			stale, ok := errors.AsType[*nodes.StaleError](err)
			if !ok {
				t.Fatalf("%s on the old basis = %v, want a StaleError", name, err)
			}
			if fresh := stale.Outcome; fresh.Basis == before.Basis() || fresh.Next[box] != role.Of(role.Downloading) {
				t.Fatalf("%s's fresh plan is on basis %s with box holding %v, want a new basis with box downloading", name, fresh.Basis, fresh.Next[box].Names())
			}
		}
		if got := c.rolesOf(t, box); !slices.Equal(got, []role.Name{role.Searching}) {
			t.Fatalf("after the stale set box holds %v, want only Searching", got)
		}
		unknown := nodes.ID{UUID: storage.NewUUID()}
		if _, err := c.svc.Apply(t.Context(), nodes.SetRoles{Basis: before.Basis(), Want: nodes.Roles{unknown: 0}}); !errors.Is(err, nodes.ErrNotFound) {
			t.Fatalf("a set naming an unknown Node: %v, want ErrNotFound", err)
		}
	})
}

func TestABlockedSetStoresNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		box := c.paired(t)
		c.apply(t, nodes.Roles{box: role.Of(role.Relaying)})
		board, err := c.svc.RoleBoard(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		own := role.Of(role.Names()...).Without(role.Relaying)
		_, err = c.svc.Apply(t.Context(), nodes.SetRoles{Basis: board.Basis(), Want: nodes.Roles{c.svc.Self(): own.With(role.Relaying)}})
		blocked, ok := errors.AsType[*nodes.BlockedError](err)
		if !ok || len(blocked.Outcome.Checks) != 1 || blocked.Outcome.Checks[0].Reason != nodes.RoleTaken {
			t.Fatalf("a second relay = %v, want blocked because box relays", err)
		}
		if want := "box already relays. Drag Relaying to move it."; blocked.Error() != want {
			t.Fatalf("the refusal reads %q, want %q", blocked.Error(), want)
		}
		if got := c.rolesOf(t, c.svc.Self()); slices.Contains(got, role.Relaying) {
			t.Fatalf("after the blocked set home holds %v, want no Relaying", got)
		}
	})
}

func TestTheSameSetOnTheCurrentBasisChangesNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		box := c.paired(t)
		c.apply(t, nodes.Roles{box: role.Of(role.Searching)})
		synctest.Wait()
		changed := c.svc.Changed()
		out := c.apply(t, nodes.Roles{box: role.Of(role.Searching)})
		synctest.Wait()
		select {
		case <-changed:
			t.Fatal("applying the same set changed the roster")
		default:
		}
		if len(out.Effects) != 0 || len(out.Checks) != 0 {
			t.Fatalf("the same set plans %v, want nothing", words(out))
		}
	})
}

// Downloading always has a holder, so the last one letting it go hands it
// back to the Controller in the same Write.
func TestUntickingTheLastHolderHandsTheRoleToTheController(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		box := c.paired(t)
		holder(t, role.InProcess(c.svc.Controls()))
		own := role.Of(role.Names()...).Without(role.Relaying)
		c.apply(t, nodes.Roles{box: role.Of(role.Downloading), c.svc.Self(): own.Without(role.Downloading)})
		if got := c.rolesOf(t, c.svc.Self()); slices.Contains(got, role.Downloading) {
			t.Fatalf("home holds %v, want Downloading on box alone", got)
		}
		out := c.apply(t, nodes.Roles{box: 0})
		wantWords(t, out, "box stops Downloading.", "home takes over Downloading, since SlipStream always needs a Node that downloads.")
		if got := c.rolesOf(t, c.svc.Self()); !slices.Contains(got, role.Downloading) {
			t.Fatalf("home holds %v after box let Downloading go, want Downloading", got)
		}
	})
}

// A Node in session has its new Roles when Apply answers: its facts show
// the node-scope version the rows derive.
func TestApplyAnswersOnceTheNodesInStepHoldTheirRoles(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		w := c.worker(t, "box", "127.0.0.3")
		_, box := c.pair(t, w, c.mint(t), "box")
		c.hold(t, w)
		synctest.Wait()
		c.apply(t, nodes.Roles{box: role.Of(role.Searching)})
		want, err := c.svc.Controls().NodeVersion(t.Context(), box)
		if err != nil {
			t.Fatal(err)
		}
		facts, ok := c.svc.Controls().NodeFacts(box)
		if !ok || !bytes.Equal(facts.GetAppliedVersion(), want) || !slices.Equal(facts.GetAssigned(), []string{"searching"}) {
			t.Fatalf("when Apply answered box's facts were %v, want searching applied at %x", facts, want)
		}
	})
}

// mailTarget owns Notifying with a mail server's target set.
func mailTarget(cfg *nodes.Config) {
	for i, o := range cfg.Owners {
		if o.Role == role.Notifying {
			cfg.Owners[i].Reach = func(context.Context, storage.Reader[storage.Library]) (nodesystem.Target, bool, error) {
				return nodesystem.Target{Host: "smtp.example.com", Port: 587}, true, nil
			}
		}
	}
}

// reacher answers each reach test with its answer, or waits out the test
// while hang is set, and records the Nodes it ran from.
type reacher struct {
	mu     sync.Mutex
	answer nodesystem.Reached
	hang   bool
	from   []nodes.ID
}

func (r *reacher) reach(ctx context.Context, node nodes.ID, _ nodesystem.Target) (nodesystem.Reached, error) {
	r.mu.Lock()
	r.from = append(r.from, node)
	answer, hang := r.answer, r.hang
	r.mu.Unlock()
	if hang {
		<-ctx.Done()
		return nodesystem.Reached{}, errors.New("never answered")
	}
	return answer, nil
}

func (r *reacher) ran() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.from)
}

// notifyingFor is a set giving a paired box in step Notifying, on a
// controller whose reach tests r answers.
func notifyingFor(t *testing.T, r *reacher) (*controller, nodes.ID, nodes.SetRoles) {
	t.Helper()
	c := startController(t, mailTarget, func(cfg *nodes.Config) { cfg.Reach = r.reach })
	w := c.worker(t, "box", "127.0.0.3")
	_, box := c.pair(t, w, c.mint(t), "box")
	c.hold(t, w)
	synctest.Wait()
	board, err := c.svc.RoleBoard(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return c, box, nodes.SetRoles{Basis: board.Basis(), Want: nodes.Roles{box: role.Of(role.Notifying)}}
}

// TestApplyTestsAnAddedNotifyingFromItsNode gives box Notifying while the
// mail server is set. A plan without tests reads no reach test, a plan
// with them and Apply each run one from box, a failed test blocks the set,
// and a passing one applies it.
func TestApplyTestsAnAddedNotifyingFromItsNode(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := &reacher{answer: nodesystem.Reached{Took: 3 * time.Millisecond, Err: "connect: connection refused"}}
		c, box, set := notifyingFor(t, r)

		plain, err := c.svc.Preview(t.Context(), set, false)
		if err != nil || len(plain.Checks) != 0 || r.ran() != 0 {
			t.Fatalf("a plan without tests checks %+v (%v) after %d tests, want none", plain.Checks, err, r.ran())
		}
		refused := "box can't reach smtp.example.com:587 (connect: connection refused)."
		tested, err := c.svc.Preview(t.Context(), set, true)
		if err != nil || len(tested.Checks) != 1 || tested.Checks[0].Words != refused {
			t.Fatalf("a tested plan checks %+v (%v), want %q", tested.Checks, err, refused)
		}
		if _, err := c.svc.Apply(t.Context(), set); !blockedBy(err, nodes.ReachFailed) {
			t.Fatalf("Apply with a failing reach test = %v, want blocked by reach_failed", err)
		}
		if got := c.rolesOf(t, box); len(got) != 0 {
			t.Fatalf("after the blocked set box holds %v, want nothing", got)
		}

		r.mu.Lock()
		r.answer = nodesystem.Reached{OK: true, Took: 42 * time.Millisecond}
		r.mu.Unlock()
		out, err := c.svc.Apply(t.Context(), set)
		if err != nil || len(out.Checks) != 1 || out.Checks[0].Words != "box reached smtp.example.com:587 in 42 ms." {
			t.Fatalf("Apply with a passing reach test = %+v, %v", out.Checks, err)
		}
		if got := c.rolesOf(t, box); !slices.Equal(got, []role.Name{role.Notifying}) {
			t.Fatalf("box holds %v, want Notifying", got)
		}
		if !slices.Equal(r.from, []nodes.ID{box, box, box}) {
			t.Fatalf("reach tests ran from %v, want box for each tested plan and Apply", r.from)
		}
	})
}

// TestAReachTestPastItsTimeFails plans box's Notifying while its reach
// test never answers, so the test fails at timings.nodes.reach.
func TestAReachTestPastItsTimeFails(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c, _, set := notifyingFor(t, &reacher{hang: true})
		late, err := c.svc.Preview(t.Context(), set, true)
		if want := "box can't reach smtp.example.com:587 (timed out after 10s)."; err != nil || len(late.Checks) != 1 || late.Checks[0].Words != want {
			t.Fatalf("a hung reach test checks %+v (%v), want %q", late.Checks, err, want)
		}
	})
}

func blockedBy(err error, why nodes.Reason) bool {
	blocked, ok := errors.AsType[*nodes.BlockedError](err)
	return ok && slices.ContainsFunc(blocked.Outcome.Checks, func(c nodes.Check) bool { return c.Reason == why })
}
