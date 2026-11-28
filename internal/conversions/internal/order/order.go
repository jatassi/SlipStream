// Package order is the fractional keys that order a lane: a key sorts
// between any two others, so moving a row writes only that row.
package order

import "strings"

type Key string

// digits sort in byte order. A key never ends in the first digit, so a
// key always fits before another.
const digits = "0123456789abcdefghijklmnopqrstuvwxyz"

// Before is a key that sorts before k. An empty k is an empty lane.
func Before(k Key) Key { return Key(mid("", string(k))) }

// Between is a key that sorts after a and before b, a < b.
func Between(a, b Key) Key { return Key(mid(string(a), string(b))) }

// After is a key that sorts after k. An empty k is an empty lane.
func After(k Key) Key { return Key(mid(string(k), "")) }

// mid is a key strictly between a and b, where "" is the start for a and
// the end for b.
func mid(a, b string) string {
	if b != "" {
		n := 0
		for n < len(b) && digitAt(a, n) == b[n] {
			n++
		}
		if n > 0 {
			return b[:n] + mid(tail(a, n), b[n:])
		}
	}
	da, db := 0, len(digits)
	if a != "" {
		da = strings.IndexByte(digits, a[0])
	}
	if b != "" {
		db = strings.IndexByte(digits, b[0])
	}
	if db-da > 1 {
		return string(digits[(da+db)/2])
	}
	if len(b) > 1 {
		return b[:1]
	}
	return string(digits[da]) + mid(tail(a, 1), "")
}

func digitAt(s string, i int) byte {
	if i < len(s) {
		return s[i]
	}
	return digits[0]
}

func tail(s string, n int) string {
	if n < len(s) {
		return s[n:]
	}
	return ""
}
