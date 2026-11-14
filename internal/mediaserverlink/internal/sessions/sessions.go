// Package sessions turns each look at a server's sessions into the plays
// it starts, changes and ends, and the server's facts. It does no IO.
package sessions

import (
	"cmp"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/storage"
)

// missedPolls is how many polls in a row a session is missing from
// before its play ends.
const missedPolls = 2

// Diff is a server's live plays, by session key and file: a session that
// moves to another file ends one play and starts another.
type Diff struct {
	server      link.ServerID
	plays       map[playKey]play
	transcoding bool
}

type playKey struct{ session, path string }

func keyOf(s link.Session) playKey { return playKey{s.Key, s.Path.Key()} }

type play struct {
	id            storage.UUID
	session       link.Session
	started, seen time.Time
	misses        int
}

func New(server link.ServerID) Diff { return Diff{server: server} }

// Next is the Diff after s, the reports s causes in order, and the
// server's facts. A failed look changes no play. A push starts and changes
// the plays it lists, and only a poll counts toward a session missing.
func (d Diff) Next(s link.Seen, newID func() storage.UUID) (Diff, []*msv1.Report, *msv1.ServerFacts) {
	if s.Problem != nil {
		return d, nil, &msv1.ServerFacts{ServerId: d.server.Bytes()}
	}
	next := Diff{server: d.server, plays: make(map[playKey]play, len(d.plays)), transcoding: d.transcoding}
	if s.Polled {
		next.transcoding = s.Transcoding
	}
	var reports []*msv1.Report
	listed := map[playKey]bool{}
	for _, x := range s.Sessions {
		k := keyOf(x)
		if listed[k] {
			continue
		}
		listed[k] = true
		p, live := d.plays[k]
		switch {
		case !live:
			p = play{id: newID(), session: x, started: s.At}
			reports = append(reports, &msv1.Report{ServerId: d.server.Bytes(), Report: &msv1.Report_PlayStarted{
				PlayStarted: &msv1.PlayStarted{Session: sessionOf(p, s.At)},
			}})
		case changed(p.session, x):
			reports = append(reports, &msv1.Report{ServerId: d.server.Bytes(), Report: &msv1.Report_PlayChanged{PlayChanged: &msv1.PlayChanged{
				PlayId: p.id.Bytes(), Decision: string(x.Decision), Hardware: x.Hardware, Reasons: x.Reasons, SeenUnixMs: s.At.UnixMilli(),
			}}})
		}
		p.session, p.seen, p.misses = x, s.At, 0
		next.plays[k] = p
	}
	for _, p := range d.live() {
		k := keyOf(p.session)
		if listed[k] {
			continue
		}
		if s.Polled {
			p.misses++
		}
		if p.misses < missedPolls {
			next.plays[k] = p
			continue
		}
		reports = append(reports, &msv1.Report{ServerId: d.server.Bytes(), Report: &msv1.Report_PlayEnded{
			PlayEnded: &msv1.PlayEnded{PlayId: p.id.Bytes(), LastSeenUnixMs: p.seen.UnixMilli()},
		}})
	}
	return next, reports, next.facts()
}

// changed is whether a PlayChanged reports b over a.
func changed(a, b link.Session) bool {
	return a.Decision != b.Decision || a.Hardware != b.Hardware || !slices.Equal(a.Reasons, b.Reasons)
}

// Watching is the report a watch sends after its first poll: the plays
// open now, which the Controller keeps open while it ends every other.
func (d Diff) Watching(at time.Time) *msv1.Report {
	w := &msv1.Watching{AtUnixMs: at.UnixMilli()}
	for _, p := range d.live() {
		w.PlayIds = append(w.PlayIds, p.id.Bytes())
	}
	return &msv1.Report{ServerId: d.server.Bytes(), Report: &msv1.Report_Watching{Watching: w}}
}

// live is the live plays, by start then id.
func (d Diff) live() []play {
	out := make([]play, 0, len(d.plays))
	for _, p := range d.plays {
		out = append(out, p)
	}
	slices.SortFunc(out, func(a, b play) int {
		return cmp.Or(a.started.Compare(b.started), cmp.Compare(a.id.String(), b.id.String()))
	})
	return out
}

func (d Diff) facts() *msv1.ServerFacts {
	f := &msv1.ServerFacts{ServerId: d.server.Bytes(), Responding: true, Transcoding: d.transcoding}
	for _, p := range d.live() {
		f.Sessions = append(f.Sessions, sessionOf(p, p.seen))
	}
	return f
}

func sessionOf(p play, seen time.Time) *msv1.Session {
	return &msv1.Session{
		PlayId: p.id.Bytes(), SessionKey: p.session.Key, ServerPath: p.session.Path.String(),
		User: p.session.User, Player: p.session.Player, Product: p.session.Product,
		Decision: string(p.session.Decision), Hardware: p.session.Hardware, Reasons: p.session.Reasons,
		StartedUnixMs: p.started.UnixMilli(), SeenUnixMs: seen.UnixMilli(),
	}
}
