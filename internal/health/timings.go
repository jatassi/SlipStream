package health

import "time"

type Timings struct {
	// Retry is how soon a category whose answer or write failed is asked
	// again.
	Retry time.Duration `yaml:"retry"`
	// Check bounds one Check now: its sampling and its evaluation.
	Check time.Duration `yaml:"check"`
}

func DefaultTimings() Timings { return Timings{Retry: time.Minute, Check: 30 * time.Second} }
