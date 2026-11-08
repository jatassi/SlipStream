package grabsapi

import "github.com/jatassi/slipstream/internal/grabs"

// Reason is a reason as the API writes it.
type Reason = reason

type wordedReason struct {
	Reason
	Words string `json:"words" doc:"The reason as a sentence"`
}

// WordedReason is a reason with its words as the API writes it, which
// the Custom Format tester's verdicts and a search's Releases share.
type WordedReason = wordedReason

// WordedOf is ws as the API writes them.
func WordedOf(ws []grabs.Worded) []WordedReason {
	out := []WordedReason{}
	for _, w := range ws {
		out = append(out, wordedReason{Reason: reasonOf(w.Reason), Words: w.Words})
	}
	return out
}
