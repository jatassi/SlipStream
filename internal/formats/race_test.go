//go:build race

package formats_test

// raceSlowdown scales the snapshot's limit by the race detector's worst
// slowdown, which Go's documentation puts at 20 times.
const raceSlowdown = 20
