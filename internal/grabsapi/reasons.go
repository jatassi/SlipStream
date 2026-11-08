package grabsapi

import "github.com/jatassi/slipstream/internal/grabs"

// Reason is a reason as the API writes it, which the Custom Format
// tester's verdicts share.
type Reason = reason

// ReasonsOf is rs as the API writes them.
func ReasonsOf(rs []grabs.Reason) []Reason { return reasonsOf(rs) }
