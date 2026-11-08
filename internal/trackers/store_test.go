package trackers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/trackers"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
)

const day = 24 * time.Hour

var admin = identity.Admin{}

func inForce(t *testing.T, s trackers.Summary, want seeding.Rule, from trackers.Source) {
	t.Helper()
	if got, source := s.Rules.InForce(); got != want || source != from {
		t.Fatalf("%s's rule is %#v from %s, want %#v from %s", s.Name, got, source, want, from)
	}
}

func TestAnAdminRuleSticksThroughANewStatedGrabAndClearingReturnsTheReleasesRule(t *testing.T) {
	l := newLibrary(t)
	seedpool := l.tracker("seedpool", indexers.Private)
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	l.grab(seedpool, start, nil, ptr(3*day))
	if err := l.x.SyncReleases(t.Context()); err != nil {
		t.Fatal(err)
	}
	s, err := l.x.Tracker(t.Context(), admin, seedpool)
	if err != nil {
		t.Fatal(err)
	}
	inForce(t, s, seeding.TimeRule{Seed: 3 * day}, trackers.FromReleases)
	l.changed.trackers = nil

	ruled := seeding.EitherRule{Ratio: 100, Seed: 5 * day}
	if s, err = l.x.SetRule(t.Context(), admin, seedpool, ruled); err != nil {
		t.Fatal(err)
	}
	inForce(t, s, ruled, trackers.FromAdmin)
	l.saved(1, &seedpool)

	l.grab(seedpool, start.Add(day), ptr(1.5), ptr(7*day))
	if err := l.x.SyncReleases(t.Context()); err != nil {
		t.Fatal(err)
	}
	if s, err = l.x.Tracker(t.Context(), admin, seedpool); err != nil {
		t.Fatal(err)
	}
	inForce(t, s, ruled, trackers.FromAdmin)
	l.changed.trackers = nil

	l.grab(seedpool, start.Add(2*day), nil, ptr(10*day))
	if s, err = l.x.ClearRule(t.Context(), admin, seedpool); err != nil {
		t.Fatal(err)
	}
	inForce(t, s, seeding.TimeRule{Seed: 10 * day}, trackers.FromReleases)
	l.saved(1, &seedpool)
}

func TestAnOlderStatedGrabNeverReplacesTheReleasesRule(t *testing.T) {
	l := newLibrary(t)
	seedpool := l.tracker("seedpool", indexers.Private)
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	l.grab(seedpool, start, ptr(1.0), nil)
	l.grab(seedpool, start.Add(-day), nil, ptr(3*day))
	l.grab(seedpool, start.Add(day), nil, nil)
	if err := l.x.SyncReleases(t.Context()); err != nil {
		t.Fatal(err)
	}
	s, err := l.x.Tracker(t.Context(), admin, seedpool)
	if err != nil {
		t.Fatal(err)
	}
	inForce(t, s, seeding.RatioRule{Ratio: 100}, trackers.FromReleases)
	if len(l.changed.trackers) != 1 || l.soon != 0 {
		t.Fatalf("a sync that stored one rule published %d events and asked for %d runs, want 1 and none", len(l.changed.trackers), l.soon)
	}
}

func TestAPublicTrackerHasNoRule(t *testing.T) {
	l := newLibrary(t)
	public := l.tracker("Nyaa", indexers.Public)
	l.grab(public, time.Now(), ptr(1.0), nil)
	if err := l.x.SyncReleases(t.Context()); err != nil {
		t.Fatal(err)
	}
	var refused *trackers.PublicError
	if _, err := l.x.SetRule(t.Context(), admin, public, seeding.TimeRule{Seed: day}); !errors.As(err, &refused) || refused.Name != "Nyaa" {
		t.Fatalf("SetRule on a public Tracker = %v, want a PublicError naming Nyaa", err)
	}
	s, err := l.x.Tracker(t.Context(), admin, public)
	if err != nil {
		t.Fatal(err)
	}
	if rule, _ := s.Rules.InForce(); rule != nil {
		t.Fatalf("a public Tracker's rule is %#v, want none", rule)
	}
	if _, err := l.x.SetRule(t.Context(), admin, indexers.TrackerID{}, seeding.TimeRule{Seed: day}); !errors.Is(err, trackers.ErrNoTracker) {
		t.Fatalf("SetRule on no Tracker = %v, want ErrNoTracker", err)
	}
}

func TestAGoalEqualToTheDefaultFollowsTheDefault(t *testing.T) {
	l := newLibrary(t)
	seedpool := l.tracker("seedpool", indexers.Private)
	own := seeding.MustFinite(seeding.Both, 300, 30*day, seeding.Pause)
	s, err := l.x.SetGoal(t.Context(), admin, seedpool, own)
	if err != nil {
		t.Fatal(err)
	}
	if s.Goal != own {
		t.Fatalf("seedpool's goal is %#v, want %#v", s.Goal, own)
	}
	if s, err = l.x.SetGoal(t.Context(), admin, seedpool, trackers.InitialGoal); err != nil {
		t.Fatal(err)
	}
	if s.Goal != nil {
		t.Fatalf("seedpool's goal set to the default is %#v, want it to follow the default", s.Goal)
	}
	l.saved(2, &seedpool)

	if err := l.x.SetDefaultGoal(t.Context(), admin, seeding.Forever{}); err != nil {
		t.Fatal(err)
	}
	l.saved(1, nil)
	if s, err = l.x.SetGoal(t.Context(), admin, seedpool, seeding.Forever{}); err != nil {
		t.Fatal(err)
	}
	if s.Goal != nil {
		t.Fatalf("seedpool's goal set to the default Forever is %#v, want it to follow the default", s.Goal)
	}
	if _, err = l.x.SetGoal(t.Context(), admin, seedpool, own); err != nil {
		t.Fatal(err)
	}
	if s, err = l.x.ClearGoal(t.Context(), admin, seedpool); err != nil || s.Goal != nil {
		t.Fatalf("ClearGoal = %#v, %v, want seedpool following the default", s.Goal, err)
	}
}

func TestATrackersGoalWithOnePartRoundTrips(t *testing.T) {
	l := newLibrary(t)
	seedpool := l.tracker("seedpool", indexers.Private)
	for _, g := range []seeding.Goal{
		seeding.MustFinite(seeding.Both, 300, 0, seeding.Pause),
		seeding.MustFinite(seeding.First, 0, 30*day, seeding.Remove),
	} {
		if _, err := l.x.SetGoal(t.Context(), admin, seedpool, g); err != nil {
			t.Fatal(err)
		}
		s, err := l.x.Tracker(t.Context(), admin, seedpool)
		if err != nil {
			t.Fatal(err)
		}
		if s.Goal != g {
			t.Fatalf("seedpool's goal saved as %#v reads %#v", g, s.Goal)
		}
	}
}

func TestTheDefaultGoalRoundTrips(t *testing.T) {
	l := newLibrary(t)
	for _, g := range []seeding.Goal{
		trackers.InitialGoal,
		seeding.MustFinite(seeding.Both, 125, 36*time.Hour, seeding.Remove),
		seeding.MustFinite(seeding.Both, 125, 0, seeding.Pause),
		seeding.MustFinite(seeding.First, 0, 36*time.Hour, seeding.RemoveDelete),
		seeding.Forever{},
		trackers.InitialGoal,
	} {
		if err := l.x.SetDefaultGoal(t.Context(), admin, g); err != nil {
			t.Fatal(err)
		}
		got, err := l.x.DefaultGoal(t.Context(), admin)
		if err != nil {
			t.Fatal(err)
		}
		if got != g {
			t.Fatalf("the default goal saved as %#v reads %#v", g, got)
		}
	}
	l.saved(6, nil)
}

func TestTheSeedingSettingsChangeAllOrNone(t *testing.T) {
	l := newLibrary(t)
	p := l.x.Settings()
	if err := p.Apply(t.Context(), setting.Patch{"trackers.margin": json.RawMessage(`600`), "trackers.warnAfter": json.RawMessage(`1`)}); err == nil {
		t.Fatal("a warning delay under a minute was saved")
	}
	if err := p.Apply(t.Context(), setting.Patch{"trackers.margin": json.RawMessage(`600`), "trackers.goal": json.RawMessage(`1`)}); err == nil {
		t.Fatal("an unknown key was saved")
	}
	if err := p.Apply(t.Context(), setting.Patch{"trackers.margin": json.RawMessage(`0`), "trackers.warnAfter": json.RawMessage(`3600`)}); err != nil {
		t.Fatal(err)
	}
	l.saved(1, nil)
	fields, err := p.Fields(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	want, err := json.Marshal([]setting.Field{trackers.Margin.Field(ptr(int64(0))), trackers.WarnAfter.Field(ptr(int64(3600)))})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("the seeding settings read %s, want %s", got, want)
	}
}
