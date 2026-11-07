package library

import (
	"errors"

	"github.com/jatassi/slipstream/internal/works"
)

type Result uint8

const (
	Done Result = iota + 1
	Skipped
	Failed
)

// Outcome is what an action did to one Work. Why is nil for Done, a
// sentinel for Skipped (ErrAlready, roots.ErrHasFiles,
// works.ErrGoneUpstream, or works.ErrNoWork on a delete) and the owner's
// error for Failed. Outcomes are in request order, one per distinct id.
type Outcome struct {
	Work   works.WorkID
	Result Result
	Why    error
}

// ErrAlready skips a Work that already reads the way an action asks.
var ErrAlready = errors.New("the Work already has that setting")

func done(w works.WorkID) Outcome { return Outcome{Work: w, Result: Done} }

func skipped(w works.WorkID, why error) Outcome { return Outcome{Work: w, Result: Skipped, Why: why} }

func failed(w works.WorkID, why error) Outcome { return Outcome{Work: w, Result: Failed, Why: why} }

// distinct is ws in request order with each repeat after the first left
// out.
func distinct(ws []works.WorkID) []works.WorkID {
	seen := make(map[works.WorkID]bool, len(ws))
	out := make([]works.WorkID, 0, len(ws))
	for _, w := range ws {
		if !seen[w] {
			seen[w] = true
			out = append(out, w)
		}
	}
	return out
}

// presentOf is each of ids a Work has, in request order.
func presentOf(ids []works.WorkID, summaries map[works.WorkID]works.Summary) []works.WorkID {
	out := make([]works.WorkID, 0, len(summaries))
	for _, id := range ids {
		if _, ok := summaries[id]; ok {
			out = append(out, id)
		}
	}
	return out
}
