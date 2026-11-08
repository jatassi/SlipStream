package external

import "time"

// Timings are downloading.Timings' client fields, and the built-in
// client's LiveSample and TotalsReport.
type Timings struct {
	RetryMin, RetryMax, RefusedRetry, Grace, ResendWait time.Duration
	MissingShare, MissingFloor                          int
	Trackers, Call, LiveFloor, LiveSample, TotalsReport time.Duration
}
