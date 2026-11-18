package subtitles

import (
	"time"

	"github.com/jatassi/slipstream/internal/subtitles/timing"
)

// Timings bound the Controller's subtitle work. None is a setting an
// Admin sees.
type Timings struct {
	// AttemptBound is how long one attempt may run before its context
	// ends: the only bound on a hung attempt's lease.
	AttemptBound time.Duration `yaml:"attempt_bound"`
	// CandidatesHold, UploadHold and AttemptHold are how long Manual
	// subtitle search's candidates, a refused upload's bytes and a finished
	// attempt's state stay readable.
	CandidatesHold time.Duration `yaml:"candidates_hold"`
	UploadHold     time.Duration `yaml:"upload_hold"`
	AttemptHold    time.Duration `yaml:"attempt_hold"`
	// PackHold is how long a pack's other members stay for the season's
	// other wants.
	PackHold time.Duration `yaml:"pack_hold"`
	// UploadCap bounds an upload's bytes, which its refusal states.
	UploadCap int `yaml:"upload_cap"`
	// DrainWidth is how many attempts, and how many Keeper settles, run at
	// once.
	DrainWidth int `yaml:"drain_width"`
	// ReconcileChunk is how many files one Read and one Write of the bulk
	// pass cover, and how many keys one Keeper sweep wakes of each kind.
	ReconcileChunk int `yaml:"reconcile_chunk"`
	// SurveyBatch bounds the videos one ListSidecars call lists.
	SurveyBatch int `yaml:"survey_batch"`
	// WakeDelay is how long after a Write the Keeper settles the keys it
	// named, by when the Write has committed.
	WakeDelay time.Duration `yaml:"wake_delay"`
	// KeeperPoll is how often the Keeper sweeps for keys a wake missed.
	KeeperPoll time.Duration `yaml:"keeper_poll"`
	// MapRetryMin and MapRetryMax bound a decode_failed map's backoff, and
	// NodeRetry is how soon a node_fault map is asked for again.
	MapRetryMin time.Duration `yaml:"map_retry_min"`
	MapRetryMax time.Duration `yaml:"map_retry_max"`
	NodeRetry   time.Duration `yaml:"node_retry"`
	// RemoveRetryMin and RemoveRetryMax bound a removal's backoff after a
	// call that didn't reach its Node.
	RemoveRetryMin time.Duration `yaml:"remove_retry_min"`
	RemoveRetryMax time.Duration `yaml:"remove_retry_max"`
	// CueMargin is how far past the file's duration a cue may end.
	CueMargin time.Duration `yaml:"cue_margin"`
	// MaxPause bounds any pause a holder reports, so a Worker can't close
	// a provider for good.
	MaxPause time.Duration `yaml:"max_pause"`
	// Siblings is how many other files' Sidecars a missing want tries.
	Siblings int `yaml:"siblings"`
	// Sync is Timing sync's constants, sent with every Fetch and Sync and
	// read by the Controller's check.
	Sync timing.Constants `yaml:"sync"`
}

func DefaultTimings() Timings {
	return Timings{
		AttemptBound: 10 * time.Minute, CandidatesHold: 15 * time.Minute, UploadHold: 30 * time.Minute, AttemptHold: 15 * time.Minute,
		PackHold: 6 * time.Hour, UploadCap: 5 << 20, DrainWidth: 3, ReconcileChunk: 2000, SurveyBatch: 200,
		WakeDelay: time.Second, KeeperPoll: time.Minute, MapRetryMin: time.Hour, MapRetryMax: 7 * 24 * time.Hour, NodeRetry: time.Hour,
		RemoveRetryMin: time.Minute, RemoveRetryMax: 24 * time.Hour, CueMargin: time.Minute, MaxPause: 24 * time.Hour, Siblings: 3,
		Sync: timing.DefaultConstants(),
	}
}
