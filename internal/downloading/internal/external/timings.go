package external

import "time"

// Timings are downloading.Timings' client fields, the built-in client's
// LiveSample and TotalsReport, and downloading.ErrorHeartbeat.
type Timings struct {
	RetryMin, RetryMax, RefusedRetry, Grace, ResendWait time.Duration
	MissingShare, MissingFloor                          int
	Trackers, Call, LiveFloor, LiveSample, TotalsReport time.Duration
	ErrorHeartbeat                                      time.Duration
}
