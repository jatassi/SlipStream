package conversions

import (
	"slices"

	"github.com/jatassi/slipstream/internal/role"
)

// RetryWhy is why a run fault fails its Conversion.
type RetryWhy string

const (
	FailedOnTwo RetryWhy = "failed_on_two"
	NoOtherNode RetryWhy = "no_other_node"
)

// RetryPlan is what a run fault does: pin the retry to a Node, exclude
// the Nodes it failed on, or fail.
type RetryPlan struct {
	Pin     *role.NodeID
	Exclude []role.NodeID
	Fail    bool
	Why     RetryWhy
}

// retryOf is decision 8. counted is the run-fault attempts since the last
// Retry, this one included, oldest first. others is whether a Node other
// than theirs can run it now, away Nodes included.
func retryOf(counted []Attempt, others bool) RetryPlan {
	nodes := distinctNodes(counted)
	switch {
	case len(counted) == 1:
		return RetryPlan{Pin: &nodes[0]}
	case len(nodes) >= 2:
		return RetryPlan{Fail: true, Why: FailedOnTwo}
	case !others:
		return RetryPlan{Fail: true, Why: NoOtherNode}
	}
	return RetryPlan{Exclude: nodes}
}

// counted is the attempts the retry budget counts: from budgetFrom on,
// ended by a run fault.
func counted(attempts []Attempt, budgetFrom int) []Attempt {
	var out []Attempt
	for _, a := range attempts {
		if a.N >= budgetFrom && a.Class == RunClass {
			out = append(out, a)
		}
	}
	return out
}

func distinctNodes(as []Attempt) []role.NodeID {
	var out []role.NodeID
	for _, a := range as {
		if !slices.Contains(out, a.Node) {
			out = append(out, a.Node)
		}
	}
	return out
}
