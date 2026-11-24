package moves

import (
	"errors"
	"time"
)

// Timings are the Moves protocol's internals, none of them an Admin's
// setting. The torrent handoff's (Guard to StopBound) are read by the
// torrent units that come with S48 (Moves)'s torrent half; Check keeps
// their order from the start.
type Timings struct {
	// Guard is how long after its last announce a source torrent waits
	// before it stops: UNIT3D's clear time.
	Guard time.Duration `yaml:"guard"`
	// ReplyWait is how long the destination waits for its tracker's reply
	// to its start.
	ReplyWait time.Duration `yaml:"reply_wait"`
	// HandoffBound is how long after asking the destination to start the
	// Controller rolls back.
	HandoffBound time.Duration `yaml:"handoff_bound"`
	// BoundMargin keeps the destination's own window, HandoffBound less
	// ReplyWait and BoundMargin, ahead of the Controller's rollback.
	BoundMargin time.Duration `yaml:"bound_margin"`
	StopBound   time.Duration `yaml:"stop_bound"`
	// CompanionsPerFile is the most companions a probe keeps per file.
	CompanionsPerFile int `yaml:"companions_per_file"`
	// PlanLifetime is how long a preview's plan waits for its Start, and
	// Plans how many previews are kept.
	PlanLifetime time.Duration `yaml:"plan_lifetime"`
	Plans        int           `yaml:"plans"`
	// PlanBatch is the Works one Read of the plan reads.
	PlanBatch int `yaml:"plan_batch"`
	// ProbePage is the most planned paths one probe message names, and
	// ProbePageBytes the most bytes of their Rels, so a page of long
	// Windows paths stays under the link's message size too.
	ProbePage      int           `yaml:"probe_page"`
	ProbePageBytes int           `yaml:"probe_page_bytes"`
	ProbesInFlight int           `yaml:"probes_in_flight"`
	ProbeDeadline  time.Duration `yaml:"probe_deadline"`
	// PassEvery is the shortest time between two passes of the runner,
	// and PassPage the most due units one pass reads.
	PassEvery time.Duration `yaml:"pass_every"`
	PassPage  int           `yaml:"pass_page"`
	// StepLease is how long a started IO step keeps its unit from being
	// picked again, should it never write its result.
	StepLease time.Duration `yaml:"step_lease"`
	// Jobs is the most IO steps running at once.
	Jobs int `yaml:"jobs"`
	// RetryAfter is when a step that failed on an error runs again.
	RetryAfter time.Duration `yaml:"retry_after"`
	// PlayingEvery is how often a unit waiting for playback reads the
	// sessions again with no event.
	PlayingEvery     time.Duration `yaml:"playing_every"`
	DownloadingEvery time.Duration `yaml:"downloading_every"`
	// RemoveRetry is the first wait before the Node removes a source file
	// another program holds open again, doubling, and RemoveBound how long
	// it keeps trying: Plex's media analysis holds a file it just scanned
	// for 1 to 2 s on Windows (recorded run 2026-11-23, case 3).
	RemoveRetry time.Duration `yaml:"remove_retry"`
	RemoveBound time.Duration `yaml:"remove_bound"`
}

func DefaultTimings() Timings {
	return Timings{
		Guard: 5 * time.Minute, ReplyWait: 10 * time.Second, HandoffBound: 2 * time.Minute, BoundMargin: 20 * time.Second,
		StopBound: 2 * time.Minute, CompanionsPerFile: 20, PlanLifetime: 30 * time.Minute, Plans: 8, PlanBatch: 500,
		ProbePage: 2000, ProbePageBytes: 4 << 20, ProbesInFlight: 4, ProbeDeadline: time.Minute,
		PassEvery: time.Second, PassPage: 500, StepLease: 10 * time.Minute, Jobs: 16, RetryAfter: 30 * time.Second,
		PlayingEvery: 30 * time.Second, DownloadingEvery: time.Minute, RemoveRetry: 100 * time.Millisecond, RemoveBound: 10 * time.Second,
	}
}

var (
	ErrTimingsOrder = errors.New("moves timings: bound_margin must be above 0 and handoff_bound above reply_wait plus bound_margin")
	ErrRemoveRetry  = errors.New("moves timings: remove_retry must be above 0")
)

// Check refuses Timings whose destination window could end after the
// Controller's rollback, and a remove retry that never waits.
func (t Timings) Check() error {
	switch {
	case t.BoundMargin <= 0 || t.HandoffBound <= t.ReplyWait+t.BoundMargin:
		return ErrTimingsOrder
	case t.RemoveRetry <= 0:
		return ErrRemoveRetry
	}
	return nil
}
