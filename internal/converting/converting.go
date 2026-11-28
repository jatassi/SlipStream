// Package converting is the Converting Role's holder: it runs ffmpeg for
// the Conversions the Controller places on this Node, reports their
// progress and results, and answers the Controller's test runs.
package converting

import (
	"time"

	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/role"
)

var Def = role.Def[*cv1.Intent, *cv1.Facts, *cv1.Result]{Name: role.Converting, Notable: notable}

// notable is a change of signature, devices or scratch volume. Progress
// never is.
func notable(_, _ *cv1.Facts) bool { return false }

// Settings is the Node's config.yaml section converting.
type Settings struct {
	// Scratch is where this Node copies sources and writes outputs. A
	// relative folder is in the data dir.
	Scratch string `yaml:"scratch"`
	// FFmpeg overrides the bundled ffmpeg, unsupported. Empty runs the
	// bundled one.
	FFmpeg string `yaml:"ffmpeg"`
}

func DefaultSettings() Settings { return Settings{Scratch: "converting"} }

type Timings struct {
	// Trial is a test run's deadline.
	Trial time.Duration `yaml:"trial"`
	// Stall is how long a run may go without progress before it is killed.
	Stall time.Duration `yaml:"stall"`
	// Progress is how often the holder reports a run's progress.
	Progress time.Duration `yaml:"progress"`
	// KillGrace is how long a killed run's tree has to exit.
	KillGrace       time.Duration `yaml:"kill_grace"`
	LengthTolerance Tolerance     `yaml:"length_tolerance"`
	// SpeedWeight is how far one run moves its proof's speed toward its own.
	SpeedWeight float64 `yaml:"speed_weight"`
}

func DefaultTimings() Timings {
	return Timings{
		Trial: time.Minute, Stall: 2 * time.Minute, Progress: 2 * time.Second, KillGrace: 5 * time.Second,
		LengthTolerance: Tolerance{Floor: time.Second, Share: 0.005}, SpeedWeight: 0.3,
	}
}

// Tolerance is how far an output's length may differ from its source's:
// the larger of Floor and Share of the source's length.
type Tolerance struct {
	Floor time.Duration `yaml:"floor"`
	Share float64       `yaml:"share"`
}

func (t Tolerance) Of(length time.Duration) time.Duration {
	return max(t.Floor, time.Duration(float64(length)*t.Share))
}
