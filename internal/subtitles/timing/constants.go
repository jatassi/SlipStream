package timing

import "time"

// Frame is one Speech Map frame and the step of every offset Timing sync
// tries, as in ffsubsync's 100 Hz signal.
const Frame = 10 * time.Millisecond

// Framerates is which ratios of the subtitle's clock to the file's Timing
// sync may try.
type Framerates uint8

const (
	NoRatios Framerates = iota + 1
	StandardRatios
	RangeRatios
)

// standard is 1, 24/23.976, 25/23.976, 25/24 and their inverses, the
// conversions among 23.976, 24 and 25 fps that alass and ffsubsync try.
var standard = [...]float64{1, 24 / 23.976, 25 / 23.976, 25.0 / 24, 23.976 / 24, 23.976 / 25, 24.0 / 25}

// RangeRatios also tries any ratio in [rangeMin, rangeMax], ffsubsync's
// golden-section bounds.
const (
	rangeMin = 0.9
	rangeMax = 1.1
)

// Gate is the Admin's three Timing sync settings.
type Gate struct {
	// Match is the lowest timing match that passes, from 0.10 to 0.95.
	Match float64
	// MaxOffset is the largest shift that passes. The correlation looks
	// twice as far, so a timing just past it is reported as PastOffset.
	MaxOffset  time.Duration
	Framerates Framerates
}

// Constants are Timing sync's config-only constants. The Controller holds
// them in timings.subtitles.sync and sends them with every call, so the
// holder's Align and the Controller's Judge read the same numbers.
type Constants struct {
	// SplitPenalty is the seconds of speech overlap one split must buy,
	// alass's split penalty in ffsubsync's units.
	SplitPenalty float64 `yaml:"split_penalty"`
	// LengthPenalty weighs the speech in each cue's guard bands, alass's
	// standard scoring as ffsubsync ports it.
	LengthPenalty float64 `yaml:"length_penalty"`
	// GuardMax caps a guard band, which is otherwise the cue's own length.
	GuardMax time.Duration `yaml:"guard_max"`
	// Prominence is the share of the best correlation peak that a peak the
	// timing doesn't use may reach before the timing is Ambiguous.
	Prominence float64 `yaml:"prominence"`
	// ProminenceGap is how far from a used offset a peak must lie to count
	// against Prominence.
	ProminenceGap time.Duration `yaml:"prominence_gap"`
	// Coverage is the least share of the map's speech span the cues must
	// span.
	Coverage float64 `yaml:"coverage"`
	// GoldenTolerance is where RangeRatios' golden-section search stops.
	GoldenTolerance float64 `yaml:"golden_tolerance"`
	// MaxSegments bounds the segments of one timing, for Align and Judge.
	MaxSegments int `yaml:"max_segments"`
	// MinSegment is the shortest segment of a timing with more than one.
	MinSegment time.Duration `yaml:"min_segment"`
	// CheckSlack is what Judge allows past the stamp unit on each time.
	CheckSlack time.Duration `yaml:"check_slack"`
}

// DefaultConstants are the values timings.subtitles.sync starts from.
func DefaultConstants() Constants {
	return Constants{
		SplitPenalty: 7, LengthPenalty: 0.25, GuardMax: 2 * time.Second,
		Prominence: 0.9, ProminenceGap: 2 * time.Second, Coverage: 0.6,
		GoldenTolerance: 5e-4, MaxSegments: 8, MinSegment: 3 * time.Minute,
		CheckSlack: 2 * time.Millisecond,
	}
}

// DefaultMatchPercent is subtitles.sync.match's default, which
// TestFalsePassRate measures on 22-minute episodes: random timings score
// at most about 28% and Prominence turns away most of those, while honest
// timings under a harsh VAD score at least about 36%.
const DefaultMatchPercent = 30
