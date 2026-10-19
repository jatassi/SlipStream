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
	broke   unit3d.Rule
}

func run(t *testing.T, steps []announce) {
	t.Helper()
	w := unit3d.NewWindows(30*time.Minute, seeded())
	for i, s := range steps {
		passkey := s.passkey
		if passkey == "" {
			passkey = "pk"
		}
		broke, reason := w.Admit(passkey, bencode.InfoHash{s.hash}, [20]byte{s.peer}, s.event, t0.Add(s.at))
		if broke != s.broke {
			t.Fatalf("step %d: %q at +%v broke %q (%q), want %q", i, s.event, s.at, broke, reason, s.broke)
		}
		if (broke == "") != (reason == "") {
			t.Fatalf("step %d: broke %q with reason %q", i, broke, reason)
		}
	}
}

func TestWindowsRejectAStartedInsideTheWindow(t *testing.T) {
	for name, steps := range map[string][]announce{
		"a quick pause and resume": {
			{at: 0, event: "started"},
			{at: time.Minute, event: "stopped"},
			{at: 2 * time.Minute, event: "started", broke: unit3d.Window},
		},
		"a regular announce": {
			{at: 0, event: "started"},
			{at: 25*time.Minute + 29*time.Second, event: "", broke: unit3d.Window},
		},
		"a window lasts at most 95% of the min interval": {
			{at: 0, event: "started"},
			{at: 28*time.Minute + 30*time.Second, event: ""},
		},
		"a completed opens a window when none is open": {
			{at: 0, event: "completed"},
			{at: time.Minute, event: "started", broke: unit3d.Window},
		},
		"an unknown event is a regular announce": {
			{at: 0, event: "started"},
			{at: time.Minute, event: "paused", broke: unit3d.Window},
		},
	} {
		t.Run(name, func(t *testing.T) { run(t, steps) })
	}
}

func TestWindowsAdmitCompletedAndStopped(t *testing.T) {
	run(t, []announce{
		{at: 0, event: "started"},
		{at: 10 * time.Minute, event: "completed"},
		{at: 11 * time.Minute, event: "stopped"},
	})
}

func TestWindowsClearOnAStoppedAfterFiveMinutes(t *testing.T) {
	for name, steps := range map[string][]announce{
		"more than five minutes clears": {
			{at: 0, event: "started"},
			{at: 5*time.Minute + time.Second, event: "stopped"},
			{at: 6 * time.Minute, event: "started"},
		},
		"exactly five minutes does not": {
			{at: 0, event: "started"},
			{at: 5 * time.Minute, event: "stopped"},
			{at: 6 * time.Minute, event: "started", broke: unit3d.Window},
		},
		"a stopped with no window opens none": {
			{at: 0, event: "stopped"},
			{at: time.Minute, event: "started"},
		},
	} {
		t.Run(name, func(t *testing.T) { run(t, steps) })
	}
}

func TestWindowsRejectTheSameEventWithin30s(t *testing.T) {
	for name, steps := range map[string][]announce{
		"a second stopped": {
			{at: 0, event: "stopped"},
			{at: 29 * time.Second, event: "stopped", broke: unit3d.Repeat},
			{at: 30 * time.Second, event: "stopped"},
		},
		"a completed then a stopped": {
			{at: 0, event: "started"},
			{at: time.Minute, event: "completed"},
			{at: time.Minute + time.Second, event: "stopped"},
		},
		"a rejected duplicate changes nothing": {
			{at: 0, event: "started"},
			{at: 6 * time.Minute, event: "stopped"},
			{at: 6*time.Minute + 10*time.Second, event: "stopped", broke: unit3d.Repeat},
			{at: 6*time.Minute + 20*time.Second, event: "started"},
		},
	} {
		t.Run(name, func(t *testing.T) { run(t, steps) })
	}
}

func TestWindowsKeyOnPeerID(t *testing.T) {
	run(t, []announce{
		{at: 0, peer: 1, event: "started"},
		{at: time.Minute, peer: 2, event: "started"},
		{at: time.Minute, peer: 1, hash: 1, event: "started"},
		{at: time.Minute, peer: 1, passkey: "other", event: "started"},
		{at: 2 * time.Minute, peer: 1, event: "started", broke: unit3d.Window},
	})
}

func TestWindowsLastBetween85And95PercentOfTheMinInterval(t *testing.T) {
	w := unit3d.NewWindows(30*time.Minute, seeded())
	var shortest, longest time.Duration
	for p := range 200 {
		peer := [20]byte{byte(p)}
		if broke, _ := w.Admit("pk", bencode.InfoHash{}, peer, "started", t0); broke != "" {
			t.Fatalf("peer %d: the opening started was rejected", p)
		}
		closes := t0.Add(25*time.Minute + 30*time.Second)
		for {
			if broke, _ := w.Admit("pk", bencode.InfoHash{}, peer, "", closes); broke == "" {
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
