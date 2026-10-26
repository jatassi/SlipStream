package targets

import "github.com/jatassi/slipstream/internal/firstrun"

// TargetsStep is first setup's Targets and Profiles step, done only by
// Continue: the claim already made the starter set it shows.
func (t *Targets) TargetsStep() firstrun.Step { return firstrun.Step{ID: firstrun.TargetsID} }
