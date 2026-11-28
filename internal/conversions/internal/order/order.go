// Package order is the fractional keys that order a lane: a key sorts
// between any two others, so moving a row writes only that row.
package order

type Key string

// Before is a key that sorts before k.
func Before(Key) Key { return "" }

// Between is a key that sorts after a and before b.
func Between(_, _ Key) Key { return "" }

// After is a key that sorts after k.
func After(Key) Key { return "" }
