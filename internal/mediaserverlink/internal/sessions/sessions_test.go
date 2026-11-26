package sessions_test

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/internal/mediaserverlink/internal/sessions"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/storage"
)

var t0 = time.Date(2026, 11, 14, 20, 0, 0, 0, time.UTC)

// run feeds each look to a fresh Diff and answers every report in order,
// as "started 1 direct/direct/none", "changed 1 direct/direct/none
// remuxed", "ended 1 at 2", with each
// play named by its mint order and each time by its look's index.
func run(t *testing.T, looks ...link.Seen) ([]string, *msv1.ServerFacts) {
	t.Helper()
	d := sessions.New(link.ServerID{UUID: storage.NewUUID()}, link.DefaultTimings())
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
				got = append(got, fmt.Sprintf("started %d %s", name(r.GetPlayStarted().GetSession().GetPlayId()), streamsWords(r.GetPlayStarted().GetSession().GetStreams())))
			case r.GetPlayChanged() != nil:
				got = append(got, fmt.Sprintf("changed %d %s", name(r.GetPlayChanged().GetPlayId()), streamsWords(r.GetPlayChanged().GetStreams())))
			case r.GetPlayEnded() != nil:
				got = append(got, fmt.Sprintf("ended %d at %d", name(r.GetPlayEnded().GetPlayId()), at[r.GetPlayEnded().GetLastSeenUnixMs()]))
			}
		}
	}
	return got, facts
}

func streamsWords(s *msv1.Streams) string {
	w := s.GetVideo() + "/" + s.GetAudio() + "/" + s.GetSubtitles()
	if s.GetContainerChanged() {
		w += " remuxed"
	}
	return w
}

var direct = link.Streams{Video: link.StreamDirect, Audio: link.StreamDirect, Subtitles: link.SubtitlesNone}

func playing(key, path string, s link.Streams) link.Session {
	p, err := link.ParsePath(path)
	if err != nil {
		panic(err)
	}
	return link.Session{Key: key, Version: link.Version{Path: p}, User: "jackson", Player: "Living room", Streams: s}
}

func with(s link.Session, streams link.Streams) link.Session {
	s.Streams = streams
	return s
}

func poll(ss ...link.Session) link.Seen { return link.Seen{Polled: true, Sessions: ss} }

func push(ss ...link.Session) link.Seen { return link.Seen{Sessions: ss} }

func pending(keys ...string) link.Seen { return link.Seen{Polled: true, Pending: keys} }

func received(s link.Session, r link.Received) link.Session {
	s.Received = r
	return s
}

func tracks(s link.Session, audio, subtitle *link.Track) link.Session {
	s.Audio, s.Subtitle = audio, subtitle
	return s
}

var h264 = link.Received{Container: "TS", VideoCodec: "x264", Height: 720, Bitrate: 4_000_000, AudioCodec: "DD+", Channels: 6}

const (
	metropolis = "/media/Movies/Metropolis (1927)/Metropolis (1927).mkv"
	nosferatu  = "/media/Movies/Nosferatu (1922)/Nosferatu (1922).mkv"
)

func TestDiff(t *testing.T) {
	m := playing("1", metropolis, direct)
	slower := h264
	slower.Bitrate = 2_000_000
	smaller := h264
	smaller.Height = 480
	english, french := &link.Track{Position: 0}, &link.Track{Position: 1}
	srt := &link.Track{External: &link.ExternalTrack{Codec: "SRT", Language: "fra"}}
	failed := link.Seen{Polled: true, Problem: &link.Problem{Reason: link.ReasonUnreachable}}
	for _, c := range []struct {
		name  string
		looks []link.Seen
		want  []string
	}{
		{"a session missing from one poll plays on", []link.Seen{poll(m), poll(), poll(m)}, []string{"started 1 direct/direct/none"}},
		{"a session missing from two polls ends at its last look", []link.Seen{poll(m), poll(m), poll(), poll()}, []string{"started 1 direct/direct/none", "ended 1 at 1"}},
		{"a push missing a session counts as no poll", []link.Seen{poll(m), push(), push(), poll()}, []string{"started 1 direct/direct/none"}},
		{"a failed poll counts as no poll", []link.Seen{poll(m), failed, failed, poll()}, []string{"started 1 direct/direct/none"}},
		{"a push starts a play", []link.Seen{push(m)}, []string{"started 1 direct/direct/none"}},
		{
			"each stream change changes the play once",
			[]link.Seen{
				poll(m),
				poll(with(m, link.Streams{Video: link.StreamDirect, Audio: link.StreamDirect, Subtitles: link.SubtitlesNone, ContainerChanged: true})),
				poll(with(m, link.Streams{Video: link.StreamDirect, Audio: link.StreamTranscode, Subtitles: link.SubtitlesNone, ContainerChanged: true})),
				poll(with(m, link.Streams{Video: link.StreamTranscode, Audio: link.StreamTranscode, Subtitles: link.SubtitlesNone, ContainerChanged: true})),
				poll(with(m, link.Streams{Video: link.StreamTranscode, Audio: link.StreamTranscode, Subtitles: link.SubtitlesBurned, ContainerChanged: true})),
				poll(with(m, link.Streams{Video: link.StreamTranscode, Audio: link.StreamTranscode, Subtitles: link.SubtitlesBurned, ContainerChanged: true})),
			},
			[]string{
				"started 1 direct/direct/none", "changed 1 direct/direct/none remuxed", "changed 1 direct/transcode/none remuxed",
				"changed 1 transcode/transcode/none remuxed", "changed 1 transcode/transcode/burned remuxed",
			},
		},
		{
			"a session on another file ends one play and starts the next",
			[]link.Seen{poll(m), poll(playing("1", nosferatu, direct)), poll(playing("1", nosferatu, direct))},
			[]string{"started 1 direct/direct/none", "started 2 direct/direct/none", "ended 1 at 0"},
		},
		{
			"two sessions are two plays",
			[]link.Seen{poll(m, playing("2", nosferatu, link.Streams{Video: link.StreamTranscode, Audio: link.StreamDirect, Subtitles: link.SubtitlesNone}))},
			[]string{"started 1 direct/direct/none", "started 2 transcode/direct/none"},
		},
		{"a bitrate-only change changes nothing", []link.Seen{poll(received(m, h264)), poll(received(m, slower))}, []string{"started 1 direct/direct/none"}},
		{
			"a received fact but the bitrate changes the play",
			[]link.Seen{poll(received(m, h264)), poll(received(m, smaller))},
			[]string{"started 1 direct/direct/none", "changed 1 direct/direct/none"},
		},
		{
			"each track change changes the play once",
			[]link.Seen{
				poll(tracks(m, english, nil)), poll(tracks(m, &link.Track{Position: 0}, nil)), poll(tracks(m, french, nil)),
				poll(tracks(m, french, srt)), poll(tracks(m, french, &link.Track{Position: 1})), poll(m),
			},
			[]string{
				"started 1 direct/direct/none", "changed 1 direct/direct/none", "changed 1 direct/direct/none",
				"changed 1 direct/direct/none", "changed 1 direct/direct/none",
			},
		},
		{
			"a pending session keeps its live play, seen at that look",
			[]link.Seen{poll(m), pending("1"), pending("1"), poll(), poll()},
			[]string{"started 1 direct/direct/none", "ended 1 at 2"},
		},
		{"a pending session with no live play starts nothing", []link.Seen{pending("1"), pending("1")}, nil},
		{
			"a pending session keeps only its own plays",
			[]link.Seen{poll(m, playing("2", nosferatu, direct)), pending("1"), pending("1")},
			[]string{"started 1 direct/direct/none", "started 2 direct/direct/none", "ended 2 at 0"},
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
	m := playing("1", metropolis, direct)
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
	d := sessions.New(link.ServerID{UUID: storage.NewUUID()}, link.DefaultTimings())
	d, reports, _ := d.Next(link.Seen{At: t0, Polled: true, Sessions: []link.Session{playing("1", metropolis, direct)}}, storage.NewUUID)
	w := d.Watching(t0).GetWatching()
	if started := reports[0].GetPlayStarted().GetSession().GetPlayId(); len(w.GetPlayIds()) != 1 || !slices.Equal(w.GetPlayIds()[0], started) {
		t.Errorf("Watching lists %x, want the one play started, %x", w.GetPlayIds(), started)
	}
}

func TestTheDiffAnswersTheVersionsOfItsLivePlays(t *testing.T) {
	p, err := link.ParsePath(metropolis)
	if err != nil {
		t.Fatal(err)
	}
	v := link.Version{Path: p, Audio: []link.Stream{{ID: "11", Codec: "DTS", Language: "deu"}}, External: []link.Stream{{ID: "12", Codec: "SRT", Language: "eng"}}}
	m := link.Session{Key: "1", Item: "42", VersionID: "7", Version: v, Streams: direct}
	d := sessions.New(link.ServerID{UUID: storage.NewUUID()}, link.DefaultTimings())
	d, _, _ = d.Next(link.Seen{At: t0, Polled: true, Sessions: []link.Session{m}}, storage.NewUUID)
	for _, c := range []struct {
		item, version string
		want          bool
	}{{"42", "7", true}, {"42", "8", false}, {"43", "7", false}} {
		got, ok := d.Version(c.item, c.version)
		if ok != c.want || ok && !reflect.DeepEqual(got, v) {
			t.Errorf("Version(%s, %s) = %+v, %v, want the live play's version: %v", c.item, c.version, got, ok, c.want)
		}
	}
	for range link.DefaultTimings().MissedPolls {
		d, _, _ = d.Next(link.Seen{At: t0, Polled: true}, storage.NewUUID)
	}
	if got, ok := d.Version("42", "7"); ok {
		t.Errorf("Version(42, 7) once the play ended = %+v, want none", got)
	}
}

// TestTheWireCarriesTheWholeSession starts a play and changes its tracks:
// the PlayStarted and the PlayChanged carry every fact the session holds.
func TestTheWireCarriesTheWholeSession(t *testing.T) {
	p, err := link.ParsePath(metropolis)
	if err != nil {
		t.Fatal(err)
	}
	transcode := link.Streams{Video: link.StreamTranscode, Audio: link.StreamDirect, Subtitles: link.SubtitlesNone, ContainerChanged: true}
	m := link.Session{
		Key: "1", Item: "42", VersionID: "7", Version: link.Version{Path: p}, PlayerKey: "office", Player: "Office PC", Platform: "Windows",
		Product: "Plex Web", User: "bob", Streams: transcode, Hardware: true, Reasons: []string{"VideoCodecNotSupported"}, Received: h264,
		Audio: &link.Track{Position: 1}, Subtitle: &link.Track{External: &link.ExternalTrack{Codec: "SRT", Language: "eng"}},
	}
	later := tracks(m, &link.Track{Position: 0}, nil)
	d := sessions.New(link.ServerID{UUID: storage.NewUUID()}, link.DefaultTimings())
	d, started, _ := d.Next(link.Seen{At: t0, Polled: true, Sessions: []link.Session{m}}, storage.NewUUID)
	_, changed, _ := d.Next(link.Seen{At: t0.Add(time.Minute), Polled: true, Sessions: []link.Session{later}}, storage.NewUUID)
	if len(started) != 1 || len(changed) != 1 {
		t.Fatalf("reports = %v then %v, want one start and one change", started, changed)
	}
	id := started[0].GetPlayStarted().GetSession().GetPlayId()
	streams := &msv1.Streams{Video: "transcode", Audio: "direct", Subtitles: "none", ContainerChanged: true}
	rec := &msv1.Received{Container: "TS", VideoCodec: "x264", Height: 720, Bitrate: 4_000_000, AudioCodec: "DD+", Channels: 6}
	for _, c := range []struct{ got, want proto.Message }{
		{started[0].GetPlayStarted().GetSession(), &msv1.Session{
			PlayId: id, SessionKey: "1", ServerPath: metropolis, ItemKey: "42", PlayerKey: "office", Player: "Office PC", Platform: "Windows",
			Product: "Plex Web", User: "bob", Streams: streams, Hardware: true, Reasons: []string{"VideoCodecNotSupported"}, Received: rec,
			AudioTrack:    &msv1.Track{Track: &msv1.Track_Position{Position: 1}},
			SubtitleTrack: &msv1.Track{Track: &msv1.Track_External{External: &msv1.ExternalTrack{Codec: "SRT", Language: "eng"}}},
			StartedUnixMs: t0.UnixMilli(), SeenUnixMs: t0.UnixMilli(),
		}},
		{changed[0].GetPlayChanged(), &msv1.PlayChanged{
			PlayId: id, Streams: streams, Hardware: true, Reasons: []string{"VideoCodecNotSupported"}, Received: rec,
			AudioTrack: &msv1.Track{Track: &msv1.Track_Position{Position: 0}}, SeenUnixMs: t0.Add(time.Minute).UnixMilli(),
		}},
	} {
		if !proto.Equal(c.got, c.want) {
			t.Errorf("sent %v, want %v", c.got, c.want)
		}
	}
}
