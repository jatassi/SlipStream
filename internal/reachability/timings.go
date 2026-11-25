package reachability

import "time"

// Timings is timings.reachability: the bounds of one run and the echo.
type Timings struct {
	// Unit bounds one Exposure's run: every read, lookup and test in it.
	Unit time.Duration `yaml:"unit"`
	// ReadFresh is how old a Node's last STUN and router pass may be for a
	// unit to reuse it, so the Exposures of one run share one pass.
	ReadFresh time.Duration `yaml:"read_fresh"`
	// Echo is how long a test's single-use echo answers.
	Echo time.Duration `yaml:"echo"`
	// Suggest is how long a suggestion's lookup of a public address is kept.
	Suggest time.Duration `yaml:"suggest"`
}

func DefaultTimings() Timings {
	return Timings{Unit: 2 * time.Minute, ReadFresh: time.Minute, Echo: 2 * time.Minute, Suggest: 10 * time.Minute}
}
