package node

import (
	"errors"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/firstrun"
	"github.com/jatassi/slipstream/internal/node/surface"
)

// The steps are built from an empty library and half: only their
// declarations are read.
func TestTheFirstRunStepsMatchTheirIDsAndEvents(t *testing.T) {
	steps := firstRunSteps(&libraryDB{}, &half{})
	ids := make([]firstrun.StepID, len(steps))
	for i, s := range steps {
		ids[i] = s.ID
		for _, e := range s.Events {
			if !slices.Contains(surface.Events, e) {
				t.Errorf("first setup step %s follows %s, which surface.Events doesn't declare", s.ID, e.Name())
			}
		}
	}
	if !slices.Equal(ids, firstrun.IDs()) {
		t.Errorf("the list binds %q, want firstrun.IDs %q in order", ids, firstrun.IDs())
	}
}

func TestFirstRunOpenRefusesBeforeTheControllerHalfStarts(t *testing.T) {
	open, err := surfaceNode{n: &Node{}}.FirstRunOpen(t.Context())
	if open || !errors.Is(err, errControllerNotRunning) {
		t.Fatalf("FirstRunOpen with no half = %t, %v; want false, errControllerNotRunning", open, err)
	}
}
