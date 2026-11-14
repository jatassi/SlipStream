package sessions_test

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/internal/sessions"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/storage"
)

var t0 = time.Date(2026, 11, 14, 20, 0, 0, 0, time.UTC)

// run feeds each look to a fresh Diff and answers every report in order,
// as "started 1 transcode", "changed 1 copy", "ended 1 at 2", with each
// play named by its mint order and each time by its look's index.
func run(t *testing.T, looks ...link.Seen) ([]string, *msv1.ServerFacts) {
	t.Helper()
	d := sessions.New(link.ServerID{UUID: storage.NewUUID()})
	names := map[storage.UUID]int{}
	at := map[int64]int{}
	mint := func() storage.UUID {
		u := storage.NewUUID()
		names[u] = len(names) + 1
		return u
	}
	name := func(b []byte) int {
		u, err := storage.UUIDFromBytes(b)
		if err != nil {
			t.Fatal(err)
		}
		return names[u]
	}
	var got []string
	var facts *msv1.ServerFacts
	for i, s := range looks {
		s.At = t0.Add(time.Duration(i) * time.Minute)
		at[s.At.UnixMilli()] = i
		var reports []*msv1.Report
		d, reports, facts = d.Next(s, mint)
		for _, r := range reports {
			switch {
			case r.GetPlayStarted() != nil:
				got = append(got, fmt.Sprintf("started %d %s", name(r.GetPlayStarted().GetSession().GetPlayId()), r.GetPlayStarted().GetSession().GetDecision()))
			case r.GetPlayChanged() != nil:
				got = append(got, fmt.Sprintf("changed %d %s", name(r.GetPlayChanged().GetPlayId()), r.GetPlayChanged().GetDecision()))
			case r.GetPlayEnded() != nil:
				got = append(got, fmt.Sprintf("ended %d at %d", name(r.GetPlayEnded().GetPlayId()), at[r.GetPlayEnded().GetLastSeenUnixMs()]))
			}
		}
	}
	return got, facts
}

func playing(key, path string, d link.Decision) link.Session {
	p, err := link.ParsePath(path)
	if err != nil {
		panic(err)
	}
	return link.Session{Key: key, Path: p, User: "jackson", Player: "Living room", Decision: d}
}

func poll(ss ...link.Session) link.Seen { return link.Seen{Polled: true, Sessions: ss} }

func push(ss ...link.Session) link.Seen { return link.Seen{Sessions: ss} }

const (
	metropolis = "/media/Movies/Metropolis (1927)/Metropolis (1927).mkv"
	nosferatu  = "/media/Movies/Nosferatu (1922)/Nosferatu (1922).mkv"
)

func TestDiff(t *testing.T) {
	m := playing("1", metropolis, link.DecisionDirectPlay)
	remux := m
	remux.Decision = link.DecisionCopy
	failed := link.Seen{Polled: true, Problem: &link.Problem{Reason: link.ReasonUnreachable}}
	for _, c := range []struct {
		name  string
		looks []link.Seen
		want  []string
	}{
		{"a session missing from one poll plays on", []link.Seen{poll(m), poll(), poll(m)}, []string{"started 1 direct_play"}},
		{"a session missing from two polls ends at its last look", []link.Seen{poll(m), poll(m), poll(), poll()}, []string{"started 1 direct_play", "ended 1 at 1"}},
		{"a push missing a session counts as no poll", []link.Seen{poll(m), push(), push(), poll()}, []string{"started 1 direct_play"}},
		{"a failed poll counts as no poll", []link.Seen{poll(m), failed, failed, poll()}, []string{"started 1 direct_play"}},
		{"a push starts a play", []link.Seen{push(m)}, []string{"started 1 direct_play"}},
		{"a new decision changes the play", []link.Seen{poll(m), poll(remux), poll(remux)}, []string{"started 1 direct_play", "changed 1 copy"}},
		{
			"a session on another file ends one play and starts the next",
			[]link.Seen{poll(m), poll(playing("1", nosferatu, link.DecisionDirectPlay)), poll(playing("1", nosferatu, link.DecisionDirectPlay))},
			[]string{"started 1 direct_play", "started 2 direct_play", "ended 1 at 0"},
		},
		{
			"two sessions are two plays",
			[]link.Seen{poll(m, playing("2", nosferatu, link.DecisionTranscode))},
			[]string{"started 1 direct_play", "started 2 transcode"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got, _ := run(t, c.looks...); !slices.Equal(got, c.want) {
				t.Errorf("reports = %q, want %q", got, c.want)
			}
		})
	}
}

func TestFacts(t *testing.T) {
	m := playing("1", metropolis, link.DecisionDirectPlay)
	_, f := run(t, link.Seen{Polled: true, Sessions: []link.Session{m}, Transcoding: true}, push(m))
	if !f.GetResponding() || !f.GetTranscoding() || len(f.GetSessions()) != 1 || f.GetSessions()[0].GetServerPath() != metropolis {
		t.Errorf("facts after a poll and a push = %v, want responding, transcoding as the poll said, Metropolis playing", f)
	}
	_, f = run(t, poll(m), link.Seen{Polled: true, Problem: &link.Problem{Reason: link.ReasonRefused}})
	if f.GetResponding() || len(f.GetSessions()) != 0 {
		t.Errorf("facts after a refused poll = %v, want not responding with no sessions", f)
	}
}

func TestWatchingListsTheLivePlays(t *testing.T) {
	d := sessions.New(link.ServerID{UUID: storage.NewUUID()})
	d, reports, _ := d.Next(link.Seen{At: t0, Polled: true, Sessions: []link.Session{playing("1", metropolis, link.DecisionDirectPlay)}}, storage.NewUUID)
	w := d.Watching(t0).GetWatching()
	if started := reports[0].GetPlayStarted().GetSession().GetPlayId(); len(w.GetPlayIds()) != 1 || !slices.Equal(w.GetPlayIds()[0], started) {
		t.Errorf("Watching lists %x, want the one play started, %x", w.GetPlayIds(), started)
	}
}
