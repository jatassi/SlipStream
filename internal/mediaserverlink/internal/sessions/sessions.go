// Package sessions turns each look at a server's sessions into the plays
// it starts, changes and ends, and the server's facts. It does no IO.
package sessions

import (
	"cmp"
	"math"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/storage"
)

// Diff is a server's live plays, by session key and file: a session that
// moves to another file ends one play and starts another.
type Diff struct {
	server      link.ServerID
	missedPolls int
	plays       map[playKey]play
	transcoding bool
}

type playKey struct{ session, path string }

func keyOf(s link.Session) playKey { return playKey{s.Key, s.Version.Path.Key()} }

type play struct {
	id            storage.UUID
	session       link.Session
	started, seen time.Time
	misses        int
}

func New(server link.ServerID, t link.Timings) Diff {
	return Diff{server: server, missedPolls: t.MissedPolls}
}

// Version is the version a live play resolved for the version id, false
// for none. It is the kinds' version cache.
func (d Diff) Version(id string) (link.Version, bool) {
	for _, p := range d.plays {
		if p.session.VersionID == id {
			return p.session.Version, true
		}
	}
	return link.Version{}, false
}

// Next is the Diff after s, the reports s causes in order, and the
// server's facts. A failed look changes no play. A push starts and changes
// the plays it lists, and only a poll counts toward a session missing. A
// pending session's live plays count as listed, as they were.
func (d Diff) Next(s link.Seen, newID func() storage.UUID) (Diff, []*msv1.Report, *msv1.ServerFacts) {
	if s.Problem != nil {
		return d, nil, &msv1.ServerFacts{ServerId: d.server.Bytes()}
	}
	next := Diff{server: d.server, missedPolls: d.missedPolls, plays: make(map[playKey]play, len(d.plays)), transcoding: d.transcoding}
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
			reports = append(reports, &msv1.Report{ServerId: d.server.Bytes(), Report: &msv1.Report_PlayChanged{PlayChanged: playChanged(p.id, x, s.At)}})
		}
		p.session, p.seen, p.misses = x, s.At, 0
		next.plays[k] = p
	}
	for _, p := range d.live() {
		k := keyOf(p.session)
		if listed[k] {
			continue
		}
		if slices.Contains(s.Pending, p.session.Key) {
			p.seen, p.misses = s.At, 0
			next.plays[k] = p
			continue
		}
		if s.Polled {
			p.misses++
		}
		if p.misses < d.missedPolls {
			next.plays[k] = p
			continue
		}
		reports = append(reports, &msv1.Report{ServerId: d.server.Bytes(), Report: &msv1.Report_PlayEnded{
			PlayEnded: &msv1.PlayEnded{PlayId: p.id.Bytes(), LastSeenUnixMs: p.seen.UnixMilli()},
		}})
	}
	return next, reports, next.facts()
}

// changed is whether b reports a PlayChanged over a: streams, hardware,
// reasons, tracks and the received facts but the bitrate.
func changed(a, b link.Session) bool {
	ar, br := a.Received, b.Received
	ar.Bitrate, br.Bitrate = 0, 0
	return a.Streams != b.Streams || a.Hardware != b.Hardware || !slices.Equal(a.Reasons, b.Reasons) ||
		ar != br || !sameTrack(a.Audio, b.Audio) || !sameTrack(a.Subtitle, b.Subtitle)
}

func sameTrack(a, b *link.Track) bool {
	switch {
	case a == nil || b == nil:
		return a == b
	case a.External == nil || b.External == nil:
		return a.External == b.External && a.Position == b.Position
	}
	return *a.External == *b.External
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
	x := p.session
	return &msv1.Session{
		PlayId: p.id.Bytes(), SessionKey: x.Key, ServerPath: x.Version.Path.String(), ItemKeys: x.Version.Items,
		PlayerKey: x.PlayerKey, Player: x.Player, Platform: x.Platform, Product: x.Product, User: x.User,
		Streams: streamsOf(x.Streams), Hardware: x.Hardware, Reasons: x.Reasons, Received: receivedOf(x.Received),
		AudioTrack: trackOf(x.Audio), SubtitleTrack: trackOf(x.Subtitle),
		StartedUnixMs: p.started.UnixMilli(), SeenUnixMs: seen.UnixMilli(),
	}
}

func playChanged(id storage.UUID, x link.Session, seen time.Time) *msv1.PlayChanged {
	return &msv1.PlayChanged{
		PlayId: id.Bytes(), Streams: streamsOf(x.Streams), Hardware: x.Hardware, Reasons: x.Reasons, Received: receivedOf(x.Received),
		AudioTrack: trackOf(x.Audio), SubtitleTrack: trackOf(x.Subtitle), SeenUnixMs: seen.UnixMilli(),
	}
}

func streamsOf(s link.Streams) *msv1.Streams {
	return &msv1.Streams{
		Video: string(s.Video), Audio: string(s.Audio), Subtitles: string(s.Subtitles), ContainerChanged: s.ContainerChanged,
	}
}

func receivedOf(r link.Received) *msv1.Received {
	return &msv1.Received{
		Container: r.Container, VideoCodec: r.VideoCodec, Height: int32Of(r.Height), Bitrate: r.Bitrate,
		AudioCodec: r.AudioCodec, Channels: int32Of(r.Channels),
	}
}

func trackOf(t *link.Track) *msv1.Track {
	switch {
	case t == nil:
		return nil
	case t.External != nil:
		return &msv1.Track{Track: &msv1.Track_External{External: &msv1.ExternalTrack{Codec: t.External.Codec, Language: t.External.Language}}}
	}
	return &msv1.Track{Track: &msv1.Track_Position{Position: int32Of(t.Position)}}
}

func int32Of(n int) int32 {
	return int32(min(max(n, math.MinInt32), math.MaxInt32))
}
