// Package pacing is the due rules every paced search shares: one first-due
// spread and one gap rule. It is pure.
package pacing

import (
	"hash/fnv"
	"time"
)

// Spread is a new row's first due offset under interval, fixed by its key,
// so a first Backlog search after an import or a migration doesn't query
// every Indexer at once.
func Spread(work, key string, interval time.Duration) time.Duration {
	h := fnv.New64a()
	_, _ = h.Write([]byte(work + "\x00" + key))
	return time.Duration(int64(h.Sum64()>>1) % int64(interval))
}

// Result is what a finished run tells the pace.
type Result uint8

const (
	// Took is a run that grabbed or started a wait.
	Took Result = iota + 1
	// Answered is a run where at least one asked Indexer answered found or
	// no_releases, and nothing was taken.
	Answered
	// Unheard is a run no asked Indexer answered, or refused at start.
	Unheard
)

// Gap is the next gap after a run: the interval after Took, or while any
// Paced cell is fresh; double, up to the longest gap, after Answered;
// unchanged after Unheard, so an Indexer outage never stretches the gaps.
// The gap is never under interval nor over longest, which is at least
// interval. capped is Config.GapCap's answer for the Items, zero when none,
// and caps the gap below all of these.
func Gap(prev, interval, longest time.Duration, r Result, fresh bool, capped time.Duration) time.Duration {
	g := prev
	switch {
	case r == Took || fresh:
		g = interval
	case r == Answered:
		g = 2 * prev
	}
	g = min(max(g, interval), max(longest, interval))
	if capped > 0 {
		g = min(g, capped)
	}
	return g
}
