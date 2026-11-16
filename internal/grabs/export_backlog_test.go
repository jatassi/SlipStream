package grabs

import "context"

// BacklogPlan is one Backlog search's snapshot, as its Writes apply it.
type BacklogPlan = backlogPlan

// PlanBacklog reads a Backlog search's snapshot, false while Backlog
// search is off.
func (gr *Grabs) PlanBacklog(ctx context.Context) (plan BacklogPlan, on bool, err error) {
	return gr.planBacklog(ctx)
}

// ApplyBacklog applies plan's Writes and answers the unit's note.
func (gr *Grabs) ApplyBacklog(ctx context.Context, plan BacklogPlan) (string, error) {
	n, err := gr.applyBacklog(ctx, plan)
	return queuedWords(n), err
}
