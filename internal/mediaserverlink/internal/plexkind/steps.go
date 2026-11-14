package plexkind

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

// step is one entry of a step list: the Step a Job reaches when run
// finishes it.
type step[W link.Work] struct {
	at  link.Step
	run func(ctx context.Context, c *conn, j link.Job, w *W) (did, error)
}

// did is what one step's run did.
type did struct {
	// waiting is a step that hasn't finished: the Job stays at its step
	// and is called again at wait.
	waiting  bool
	wait     time.Time
	outcomes []link.Outcome
	learned  []link.IndexEntry
}

// run runs the entry of steps after j.Step once. The first entry runs
// from StepNone. The Job is done once its last entry finishes.
func run[W link.Work](ctx context.Context, c *conn, steps []step[W], j link.Job, w W) (link.Advanced, error) {
	i := 0
	if j.Step != link.StepNone {
		i = slices.IndexFunc(steps, func(s step[W]) bool { return s.at == j.Step }) + 1
		if i == 0 || i == len(steps) {
			return link.Advanced{}, fmt.Errorf("plex %T has no step after %s", w, j.Step)
		}
	}
	s := steps[i]
	d, err := s.run(ctx, c, j, &w)
	if err != nil {
		return link.Advanced{}, err
	}
	a := link.Advanced{Job: j, Wait: d.wait, Outcomes: d.outcomes, Learned: d.learned}
	a.Job.Work = w
	if !d.waiting {
		a.Job.Step, a.Job.At = s.at, time.Now()
		a.Done = i == len(steps)-1
	}
	return a, nil
}

func accepted(changes []link.ChangeID) []link.Outcome {
	out := make([]link.Outcome, len(changes))
	for i, id := range changes {
		out[i] = link.Outcome{Change: id, Result: link.ResultAccepted}
	}
	return out
}
