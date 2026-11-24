package external

import "time"

// Timings are downloading.Timings' client fields, the built-in client's
// LiveSample and TotalsReport, and downloading.ErrorHeartbeat.
type Timings struct {
	RetryMin, RetryMax, RefusedRetry, Grace, ResendWait time.Duration
	MissingShare, MissingFloor                          int
	Trackers, Call, LiveFloor, LiveSample, TotalsReport time.Duration
	LoadWait, LoadEvery, RunWait                        time.Duration
	ErrorHeartbeat                                      time.Duration
	// ReplyWait is how long a Move's start waits for a tracker's answer,
	// and AnnounceWatch how often the torrents a Move watches are read for
	// their announces.
	ReplyWait, AnnounceWatch time.Duration
}
