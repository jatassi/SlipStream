package mediaservers

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storing"
)

// Sessions is the kind's own reading of the server's sessions: the first
// poll of a watch, version lookups included, with an empty Versions.
func Sessions(ctx context.Context, k link.Kind, d link.Dialed) ([]link.Session, error) {
	d.Versions, d.Timings = noVersions{}, watching
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var first *link.Seen
	err := k.Dial(d).Watch(ctx, time.Hour, func(s link.Seen) {
		if s.Polled && first == nil {
			first = &s
			cancel()
		}
	})
	switch {
	case err != nil:
		return nil, err
	case first == nil:
		return nil, errors.New("the watch ended before its first poll")
	case first.Problem != nil:
		return nil, &link.ServerError{Problem: *first.Problem}
	}
	return first.Sessions, nil
}

// watching are the Timings a watch reads until its first poll.
var watching = link.Timings{ReconnectMin: time.Second, ReconnectMax: time.Minute, SessionCap: 100, SessionsPush: 1500 * time.Millisecond}

type noVersions struct{}

func (noVersions) Version(string, string) (link.Version, bool) { return link.Version{}, false }

// player is the player key every case plays on.
const player = "slipstream-conformance-player"

var (
	direct     = link.Streams{Video: link.StreamDirect, Audio: link.StreamDirect, Subtitles: link.SubtitlesNone}
	remuxed    = link.Streams{Video: link.StreamDirect, Audio: link.StreamDirect, Subtitles: link.SubtitlesNone, ContainerChanged: true}
	transcoded = link.Streams{Video: link.StreamTranscode, Audio: link.StreamDirect, Subtitles: link.SubtitlesNone, ContainerChanged: true}
)

// ReadsDecisions plays a file once per decision: the kind reads each
// play's decision, and the session is gone once the play stops.
func ReadsDecisions(t *testing.T, tg Target) {
	if tg.Play == nil {
		Skip(t, "a real server's sessions need a player")
	}
	f := tg.Case(t)
	defer f.Clear()
	r := Rel(t, "Playing", ".mkv")
	f.Place(r)
	f.Tell(r)
	f.Found(r)
	for _, p := range []struct {
		streams link.Streams
		want    link.Decision
	}{
		{direct, link.DecisionDirectPlay},
		{remuxed, link.DecisionDirectStream},
		{transcoded, link.DecisionTranscode},
	} {
		stop := tg.Play(t, Played{Path: f.At(r), PlayerKey: player, Streams: p.streams})
		if s := f.session(r); s.Streams.Decision() != p.want {
			t.Fatalf("playing %s as %s, its session is %+v", f.At(r), p.want, s)
		}
		stop()
		f.Stopped(r)
	}
}

// Stopped waits until no session plays the file at r.
func (f *Files) Stopped(r storing.Rel) {
	f.t.Helper()
	Eventually(f.t, f.tg, "the session's end", func() (bool, error) {
		sessions, err := f.tg.Server.Sessions(f.t.Context())
		return !slices.ContainsFunc(sessions, func(s link.Session) bool { return s.Version.Path.Key() == f.At(r).Key() }), err
	})
}

// ReadsSessions plays one file in turn on one player: direct play, direct
// stream, an audio-only transcode, a video transcode with what the player
// received, burned subtitles, and a chosen embedded audio track with an
// external subtitle. The kind reads every play on one player key, with
// each stream's decision, what the player received in release words, and
// the tracks.
func ReadsSessions(t *testing.T, tg Target) {
	if tg.Play == nil {
		Skip(t, "a real server's sessions need a player")
	}
	f := tg.Case(t)
	defer f.Clear()
	video, srt := Rel(t, "Streamed", ".mkv"), Rel(t, "Streamed", ".en.srt")
	f.Place(video, srt)
	f.Tell(video)
	f.Found(video)
	external := &link.Track{External: &link.ExternalTrack{Codec: string(release.SRT), Language: "en"}}
	received := link.Received{Container: string(release.TS), VideoCodec: "x264", Height: 720, Bitrate: 4_000_000, AudioCodec: "AAC", Channels: 2}
	var key string
	for _, p := range []struct {
		name string
		play Played
	}{
		{"direct play", Played{Streams: direct}},
		{"direct stream", Played{Streams: remuxed}},
		{"audio-only transcode", Played{Streams: link.Streams{Video: link.StreamDirect, Audio: link.StreamTranscode, Subtitles: link.SubtitlesNone, ContainerChanged: true}}},
		{"video transcode", Played{Streams: transcoded, Received: received}},
		{"burned subtitles", Played{Streams: link.Streams{Video: link.StreamTranscode, Audio: link.StreamDirect, Subtitles: link.SubtitlesBurned, ContainerChanged: true}, Subtitle: external}},
		{"chosen tracks", Played{Streams: link.Streams{Video: link.StreamDirect, Audio: link.StreamDirect, Subtitles: link.SubtitlesDirect}, Audio: &link.Track{Position: 0}, Subtitle: external}},
	} {
		p.play.Path, p.play.PlayerKey = f.At(video), player
		stop := tg.Play(t, p.play)
		s := f.session(video)
		if key == "" {
			key = s.PlayerKey
		}
		got := fmt.Sprintf("player %q, %+v, received %+v, audio %s, subtitle %s", s.PlayerKey, s.Streams, s.Received, track(s.Audio), track(s.Subtitle))
		want := fmt.Sprintf("player %q, %+v, received %+v, audio %s, subtitle %s", key, p.play.Streams, p.play.Received, track(p.play.Audio), track(p.play.Subtitle))
		if key == "" || got != want {
			t.Errorf("the %s reads %s, want %s", p.name, got, want)
		}
		stop()
		f.Stopped(video)
	}
}

// AlternateVersion plays the second of two versions of one movie in one
// folder: the kind reads the session playing that version's file.
func AlternateVersion(t *testing.T, tg Target) {
	if tg.Play == nil {
		Skip(t, "a real server's sessions need a player")
	}
	f := tg.Case(t)
	defer f.Clear()
	uhd, hd := Rel(t, "Versioned", " - 2160p.mkv"), Rel(t, "Versioned", " - 1080p.mkv")
	f.Place(uhd, hd)
	f.Tell(uhd, hd)
	f.Found(uhd)
	f.Found(hd)
	stop := tg.Play(t, Played{Path: f.At(uhd), Version: f.At(hd), PlayerKey: player, Streams: direct})
	defer stop()
	f.session(hd)
}

// session is the one session playing the file at r.
func (f *Files) session(r storing.Rel) link.Session {
	f.t.Helper()
	sessions, err := f.tg.Server.Sessions(f.t.Context())
	if err != nil {
		f.t.Fatal(err)
	}
	i := slices.IndexFunc(sessions, func(s link.Session) bool { return s.Version.Path.Key() == f.At(r).Key() })
	if i < 0 {
		f.t.Fatalf("no session plays %s: %+v", f.At(r), sessions)
	}
	return sessions[i]
}

func track(t *link.Track) string {
	switch {
	case t == nil:
		return "none"
	case t.External != nil:
		return fmt.Sprintf("external %s %s", t.External.Codec, t.External.Language)
	}
	return fmt.Sprintf("position %d", t.Position)
}
