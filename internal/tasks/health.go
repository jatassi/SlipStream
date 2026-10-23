package tasks

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/live"
)

// HealthAbout declares the Scheduled tasks category. A task waits while
// its Node is away, and the clock publishes ChangedEvent at each change of
// the roster, so ChangedEvent covers the Nodes' changes too.
var HealthAbout = health.About{
	ID:     health.ScheduledTasksID,
	Label:  "Scheduled tasks",
	Phrase: "scheduled tasks",
	Events: []live.Event{ChangedEvent},
}

// HealthSource answers category's items from the tasks whose Health names
// it: one Warn per task whose latest run failed or whose unit went without
// success past tasks.partialWarnIntervals, with Run now as its action, and
// as Next the instant the next unit goes overdue. A task waiting for its
// Node is unsure, so its item stands until the Node returns. No Check now:
// Run now is the task's own retry.
func (s *Scheduler) HealthSource(category health.ID) health.Source {
	return health.Source{Answer: func(ctx context.Context) (health.Answer, error) {
		l, err := s.look(ctx, everyTask)
		if err != nil {
			return health.Answer{}, err
		}
		now := time.Now()
		var a health.Answer
		for _, v := range s.views(l, now) {
			if categoryOf(v.Task) != string(category) {
				continue
			}
			key := health.KeyOf(string(v.Task.ID))
			if wait, ok := v.Status.(Waiting); ok {
				a.Unsure = append(a.Unsure, health.Unsure{Key: key, Says: health.Says{
					Subject: v.Task.Name,
					Text:    v.Task.Name + " is waiting for " + wait.Node + " and runs again once it is back.",
					Detail:  []health.Fact{{Label: "Waiting since", At: wait.Since}},
				}})
				continue
			}
			if w, ok := s.warning(l, v, now); ok {
				a.Items = append(a.Items, health.Item{Key: key, Level: health.Warn, Says: saysOf(w)})
			}
			if next := s.nextWarning(l, v, now); !next.IsZero() && (a.Next.IsZero() || next.Before(a.Next)) {
				a.Next = next
			}
		}
		return a, nil
	}}
}

func saysOf(w Warning) health.Says {
	says := health.Says{Subject: w.Name, Action: health.RunAction(string(w.Task))}
	if w.Failed {
		says.Text = w.Text + ". Run it again once the cause is fixed."
		says.Detail = []health.Fact{{Label: "Failed at", At: w.At}}
	} else {
		says.Text = w.Text + ". Run it now to try " + w.Unit + " again."
		says.Detail = []health.Fact{{Label: "Not finished", Text: w.Unit}, {Label: "Unfinished since", At: w.At}}
	}
	if w.Cause != "" {
		says.Detail = append(says.Detail, health.Fact{Label: "Cause", Text: w.Cause})
	}
	return says
}
