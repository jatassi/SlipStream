package identity

import (
	"net/netip"
	"sync"
	"time"
)

// limiter counts failed sign-in attempts per client in memory: an IPv4
// address, or an IPv6 /64. Successes never count. The stored per-Account
// hard stop is the durable bound; this only slows a sweep across Accounts.
type limiter struct {
	mu   sync.Mutex
	keys map[netip.Prefix][]time.Time
}

// limiterCap bounds the clients remembered. Past it, the one whose newest
// failure is oldest is forgotten.
const limiterCap = 65536

func newLimiter() *limiter { return &limiter{keys: map[netip.Prefix][]time.Time{}} }

func keyOf(a netip.Addr) netip.Prefix {
	a = a.Unmap()
	if a.Is4() {
		return netip.PrefixFrom(a, 32)
	}
	p, _ := a.Prefix(64)
	return p
}

// recent is key's failures inside window, oldest first. It runs under mu.
func (l *limiter) recent(key netip.Prefix, window time.Duration, now time.Time) []time.Time {
	times := l.keys[key]
	for len(times) > 0 && !times[0].After(now.Add(-window)) {
		times = times[1:]
	}
	return times
}

// blocked reports when a stops being refused, once it has limit failures
// inside window.
func (l *limiter) blocked(a netip.Addr, limit int, window time.Duration, now time.Time) (time.Time, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	times := l.recent(keyOf(a), window, now)
	if len(times) < limit {
		return time.Time{}, false
	}
	return times[len(times)-limit].Add(window), true
}

func (l *limiter) fail(a netip.Addr, limit int, window time.Duration, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	key := keyOf(a)
	times := l.recent(key, window, now)
	if len(times) >= limit {
		times = times[len(times)-limit+1:]
	}
	if _, known := l.keys[key]; !known && len(l.keys) >= limiterCap {
		l.forgetStalest()
	}
	l.keys[key] = append(times, now)
}

func (l *limiter) forgetStalest() {
	var stalest netip.Prefix
	var at time.Time
	for key, times := range l.keys {
		if newest := times[len(times)-1]; at.IsZero() || newest.Before(at) {
			stalest, at = key, newest
		}
	}
	delete(l.keys, stalest)
}
