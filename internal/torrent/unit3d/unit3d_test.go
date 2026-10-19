package unit3d_test

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/unit3d"
)

var t0 = time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)

func seeded() *rand.Rand {
	return rand.New(rand.NewChaCha8([32]byte{'u', '3', 'd'})) //nolint:gosec // seeded on purpose so tests replay
}

type announce struct {
	at      time.Duration
	passkey string
	hash    byte
	peer    byte
	event   string
	ok      bool
}

func run(t *testing.T, steps []announce) {
	t.Helper()
	w := unit3d.NewWindows(30*time.Minute, seeded())
	for i, s := range steps {
		passkey := s.passkey
		if passkey == "" {
			passkey = "pk"
		}
		ok, reason := w.Admit(passkey, bencode.InfoHash{s.hash}, [20]byte{s.peer}, s.event, t0.Add(s.at))
		if ok != s.ok {
			t.Fatalf("step %d: %q at +%v admitted %v (%q), want %v", i, s.event, s.at, ok, reason, s.ok)
		}
		if ok != (reason == "") {
			t.Fatalf("step %d: admitted %v with reason %q", i, ok, reason)
		}
	}
}

func TestWindowsRejectAStartedInsideTheWindow(t *testing.T) {
	for name, steps := range map[string][]announce{
		"a quick pause and resume": {
			{at: 0, event: "started", ok: true},
			{at: time.Minute, event: "stopped", ok: true},
			{at: 2 * time.Minute, event: "started", ok: false},
		},
		"a regular announce": {
			{at: 0, event: "started", ok: true},
			{at: 25*time.Minute + 29*time.Second, event: "", ok: false},
		},
		"a window lasts at most 95% of the min interval": {
			{at: 0, event: "started", ok: true},
			{at: 28*time.Minute + 30*time.Second, event: "", ok: true},
		},
		"a completed opens a window when none is open": {
			{at: 0, event: "completed", ok: true},
			{at: time.Minute, event: "started", ok: false},
		},
		"an unknown event is a regular announce": {
			{at: 0, event: "started", ok: true},
			{at: time.Minute, event: "paused", ok: false},
		},
	} {
		t.Run(name, func(t *testing.T) { run(t, steps) })
	}
}

func TestWindowsAdmitCompletedAndStopped(t *testing.T) {
	run(t, []announce{
		{at: 0, event: "started", ok: true},
		{at: 10 * time.Minute, event: "completed", ok: true},
		{at: 11 * time.Minute, event: "stopped", ok: true},
	})
}

func TestWindowsClearOnAStoppedAfterFiveMinutes(t *testing.T) {
	for name, steps := range map[string][]announce{
		"more than five minutes clears": {
			{at: 0, event: "started", ok: true},
			{at: 5*time.Minute + time.Second, event: "stopped", ok: true},
			{at: 6 * time.Minute, event: "started", ok: true},
		},
		"exactly five minutes does not": {
			{at: 0, event: "started", ok: true},
			{at: 5 * time.Minute, event: "stopped", ok: true},
			{at: 6 * time.Minute, event: "started", ok: false},
		},
		"a stopped with no window opens none": {
			{at: 0, event: "stopped", ok: true},
			{at: time.Minute, event: "started", ok: true},
		},
	} {
		t.Run(name, func(t *testing.T) { run(t, steps) })
	}
}

func TestWindowsRejectTheSameEventWithin30s(t *testing.T) {
	for name, steps := range map[string][]announce{
		"a second stopped": {
			{at: 0, event: "stopped", ok: true},
			{at: 29 * time.Second, event: "stopped", ok: false},
			{at: 30 * time.Second, event: "stopped", ok: true},
		},
		"a completed then a stopped": {
			{at: 0, event: "started", ok: true},
			{at: time.Minute, event: "completed", ok: true},
			{at: time.Minute + time.Second, event: "stopped", ok: true},
		},
		"a rejected duplicate changes nothing": {
			{at: 0, event: "started", ok: true},
			{at: 6 * time.Minute, event: "stopped", ok: true},
			{at: 6*time.Minute + 10*time.Second, event: "stopped", ok: false},
			{at: 6*time.Minute + 20*time.Second, event: "started", ok: true},
		},
	} {
		t.Run(name, func(t *testing.T) { run(t, steps) })
	}
}

func TestWindowsKeyOnPeerID(t *testing.T) {
	run(t, []announce{
		{at: 0, peer: 1, event: "started", ok: true},
		{at: time.Minute, peer: 2, event: "started", ok: true},
		{at: time.Minute, peer: 1, hash: 1, event: "started", ok: true},
		{at: time.Minute, peer: 1, passkey: "other", event: "started", ok: true},
		{at: 2 * time.Minute, peer: 1, event: "started", ok: false},
	})
}

func TestWindowsLastBetween85And95PercentOfTheMinInterval(t *testing.T) {
	w := unit3d.NewWindows(30*time.Minute, seeded())
	var shortest, longest time.Duration
	for p := range 200 {
		peer := [20]byte{byte(p)}
		if ok, _ := w.Admit("pk", bencode.InfoHash{}, peer, "started", t0); !ok {
			t.Fatalf("peer %d: the opening started was rejected", p)
		}
		closes := t0.Add(25*time.Minute + 30*time.Second)
		for {
			if ok, _ := w.Admit("pk", bencode.InfoHash{}, peer, "", closes); ok {
				break
			}
			closes = closes.Add(30 * time.Second)
		}
		d := closes.Sub(t0)
		if d < 25*time.Minute+30*time.Second || d > 28*time.Minute+30*time.Second {
			t.Fatalf("peer %d: window closed after %v", p, d)
		}
		if shortest == 0 || d < shortest {
			shortest = d
		}
		longest = max(longest, d)
	}
	if longest-shortest < 2*time.Minute {
		t.Fatalf("windows closed between %v and %v, want them spread over the 3-minute band", shortest, longest)
	}
}
