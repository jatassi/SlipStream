package seeding_test

import (
	"testing"

	dc "github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
)

func TestSeenOf(t *testing.T) {
	reached := map[dc.State]bool{}
	for _, c := range []struct {
		name   string
		state  dc.State
		hold   dc.Hold
		inStep bool
		want   seeding.Seen
	}{
		{"seeding", dc.StateSeeding, dc.HoldNone, true, seeding.Announcing},
		{"downloading", dc.StateDownloading, dc.HoldNone, true, seeding.Announcing},
		{"stalled still announces", dc.StateStalled, dc.HoldNone, true, seeding.Announcing},
		{"held for a tracker window or the start spread", dc.StatePaused, dc.HoldWindow, true, seeding.Announcing},
		{"paused by the Admin or a goal", dc.StatePaused, dc.HoldWanted, true, seeding.Quiet},
		{"paused for disk", dc.StatePaused, dc.HoldDiskFull, true, seeding.Quiet},
		{"waiting for a check slot", dc.StateQueued, dc.HoldCheckSlot, true, seeding.Quiet},
		{"held while the seedbox drains", dc.StateQueued, dc.HoldDraining, true, seeding.Quiet},
		{"held until the Controller is reached", dc.StateQueued, dc.HoldController, true, seeding.Quiet},
		{"checking", dc.StateChecking, dc.HoldNone, true, seeding.Quiet},
		{"a tracker error", dc.StateTrackerErr, dc.HoldNone, true, seeding.Quiet},
		{"failed", dc.StateFailed, dc.HoldNone, true, seeding.Quiet},
		{"being removed", dc.StateRemoving, dc.HoldNone, true, seeding.Quiet},
		{"seeding on a Node out of step", dc.StateSeeding, dc.HoldNone, false, seeding.Unseen},
		{"failed on a Node out of step", dc.StateFailed, dc.HoldNone, false, seeding.Unseen},
	} {
		if got := seeding.SeenOf(string(c.state), string(c.hold), c.inStep); got != c.want {
			t.Errorf("%s: SeenOf = %s, want %s", c.name, got, c.want)
		}
		reached[c.state] = true
	}
	for _, s := range dc.States {
		if !reached[s] {
			t.Errorf("no row reads %s", s)
		}
	}
}
