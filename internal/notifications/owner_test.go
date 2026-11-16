package notifications_test

import (
	"context"
	"slices"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/notifications"
	"github.com/jatassi/slipstream/internal/notifications/notificationstest"
	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// every is what a Controller holds after first run.
var every = role.Of(role.Names()...).Without(role.Relaying)

// situation seats den, the Controller, and attic with what the owner
// keeps on each and the targets it tests.
func situation(t *testing.T, w *notificationstest.World, den, attic nodes.Seat) nodes.Situation {
	t.Helper()
	o := w.Service.Owner()
	s := nodes.Situation{Controller: den.ID, Targets: map[role.Name][]nodes.Target{}}
	err := w.Lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		for _, seat := range []nodes.Seat{den, attic} {
			kept, err := o.Keeps(ctx, r, seat.ID)
			if err != nil {
				return err
			}
			seat.Keeps = map[role.Name][]string{role.Notifying: kept}
			s.Seats = append(s.Seats, seat)
		}
		targets, err := o.Reach(ctx, r)
		s.Targets[role.Notifying] = targets
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func seats(w *notificationstest.World, attic role.Set) (den, atticSeat nodes.Seat, id role.NodeID) {
	id = w.AddNode("Attic")
	return nodes.Seat{Named: nodes.Named{ID: w.Self, Name: "Den"}, Assigned: every},
		nodes.Seat{Named: nodes.Named{ID: id, Name: "Attic"}, Assigned: attic}, id
}

func reasons(o nodes.Outcome) []nodes.Reason {
	var out []nodes.Reason
	for _, c := range o.Checks {
		out = append(out, c.Reason)
	}
	return out
}

func TestTakingNotifyingOffAScriptsNodeIsRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := notificationstest.New(t, notifications.DefaultTimings())
		den, attic, id := seats(w, role.Of(role.Notifying))
		w.ConnectScript("Pager", id)
		out := nodes.Plan(situation(t, w, den, attic), nodes.SetRoles{Want: nodes.Roles{id: role.Of()}})
		if !out.Blocked() || !slices.Equal(reasons(out), []nodes.Reason{nodes.KeepsConfig}) {
			t.Fatalf("taking Notifying off Attic checks %+v, want it kept by the script", out.Checks)
		}
		if got, want := out.Checks[0].Words, "Attic still has Custom Script 'Pager'. First delete it, or move it to another Node that sends notifications, including email."; got != want {
			t.Fatalf("the block reads %q, want %q", got, want)
		}
	})
}

func TestAScriptOnTheControllerNeverBlocks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := notificationstest.New(t, notifications.DefaultTimings())
		den, attic, id := seats(w, role.Of())
		w.ConnectScript("Doorbell", w.Self)
		out := nodes.Plan(situation(t, w, den, attic), nodes.SetRoles{Want: nodes.Roles{w.Self: every.Without(role.Notifying), id: role.Of(role.Notifying)}})
		if out.Blocked() || slices.Contains(reasons(out), nodes.KeepsConfig) {
			t.Fatalf("moving Notifying from Den with a script there checks %+v, want it allowed", out.Checks)
		}
	})
}

func TestANodeThatCantReachAConnectionGainsNotifyingWithAWarning(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := notificationstest.New(t, notifications.DefaultTimings())
		den, attic, id := seats(w, role.Of())
		w.Connect("Family")
		w.Connect("Away")
		w.Connect("Off", func(d *notifications.Draft) {
			d.On = false
			d.Settings = providers.Values{"url": notificationstest.Text("https://off.example/hook")}
		})
		s := situation(t, w, den, attic)
		if got := s.Targets[role.Notifying]; len(got) != 1 || got[0].String() != "hook.example:443" || got[0].Block {
			t.Fatalf("the targets are %+v, want hook.example:443 once, warning", got)
		}
		s.Reach = map[nodes.ReachKey]nodes.ReachResult{
			{NodeRole: nodes.NodeRole{Node: id, Role: role.Notifying}, Target: "hook.example:443"}: {Err: "connect: connection refused"},
		}
		out := nodes.Plan(s, nodes.SetRoles{Want: nodes.Roles{id: role.Of(role.Notifying)}})
		if out.Blocked() || !out.Next[id].Has(role.Notifying) || !slices.Equal(reasons(out), []nodes.Reason{nodes.ReachWarned}) {
			t.Fatalf("giving Attic Notifying checks %+v, next %v; want it gained with a warning", out.Checks, out.Next[id])
		}
		if got, want := out.Checks[0].Words, "Attic can't reach hook.example. Messages to Hook 'Away' and Hook 'Family' fail while Attic sends them."; got != want {
			t.Fatalf("the warning reads %q, want %q", got, want)
		}
	})
}
