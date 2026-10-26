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
// sentinel for Skipped and the owner's error for Failed. Outcomes are in
// request order, one per distinct id.
type Outcome struct {
	Work   works.WorkID
	Result Result
	Why    error
}

// ErrAlready skips a Work that already reads the way an action asks.
var ErrAlready = errors.New("the Work already reads that way")
