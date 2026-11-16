package notifications

import "time"

type Timings struct {
	// Send bounds one request or one script start.
	Send time.Duration `yaml:"send"`
	// Ladder is the wait before each retry of a failing connection; the
	// last step repeats.
	Ladder []time.Duration `yaml:"ladder"`
}

func DefaultTimings() Timings {
	return Timings{
		Send:   30 * time.Second,
		Ladder: []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, 30 * time.Minute, time.Hour, 3 * time.Hour, 6 * time.Hour, 12 * time.Hour},
	}
}
