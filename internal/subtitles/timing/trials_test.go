//go:build !scale

package timing_test

// falsePassTrials keeps CI to a sample that runs in seconds. The scale
// build measures the 1,000 trials the match default was tuned on.
func falsePassTrials() (trials, honest int) { return 150, 60 }
