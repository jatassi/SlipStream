package health

import "time"

type Timings struct {
	// Reevaluate is how often every source is asked again, for what
	// appears with time alone, such as a partial task unit.
	Reevaluate time.Duration `yaml:"reevaluate"`
	// Check bounds one Check now: its sampling and its evaluation.
	Check time.Duration `yaml:"check"`
}

func DefaultTimings() Timings { return Timings{Reevaluate: time.Minute, Check: 30 * time.Second} }
