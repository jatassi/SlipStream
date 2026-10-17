package nodes_test

import (
	"bytes"
	"errors"
	"slices"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/nodes"
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
			"Preview": func() (nodes.Outcome, error) { return c.svc.Preview(t.Context(), late) },
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
