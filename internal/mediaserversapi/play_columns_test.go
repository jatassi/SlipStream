package mediaserversapi_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/mstest"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/storage"
)

// TestAPlayRecordsWhatItsNodeReports starts a direct stream of Clip on
// Den and changes it to burn in an external subtitle: each row keeps the
// player, item, tracks and received facts the Node reported, with the
// decision its streams give, and a report naming an unknown stream
// decision is refused.
func TestAPlayRecordsWhatItsNodeReports(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWatched(t)
		u, err := storage.ParseUUID(w.id)
		if err != nil {
			t.Fatal(err)
		}
		server, now := link.ServerID{UUID: u}, time.Now()
		p := mstest.Play{
			Server: server, Session: "9", Path: clip, Items: []string{"i9"}, PlayerKey: "lrtv", Player: "Living room", Platform: "AndroidTv",
			Product: "Jellyfin for Android TV", User: "ada", Audio: &link.Track{Position: 0}, Started: now, Seen: now,
			Streams: link.Streams{Video: link.StreamDirect, Audio: link.StreamDirect, Subtitles: link.SubtitlesNone, ContainerChanged: true},
		}
		id := w.w.Play(t, w.w.Seedbox, p)
		const who = `[] by ada on Living room key lrtv platform AndroidTv items ["i9"] audio 0`
		expectPlays(t, w.plays(t), clip+` direct_stream (direct direct none) `+who+`, open`)

		p.Streams.Subtitles, p.Subtitle, p.Seen = link.SubtitlesBurned, &link.Track{External: &link.ExternalTrack{Codec: "SRT", Language: "en"}}, now.Add(time.Minute)
		p.Received = link.Received{Container: "TS", VideoCodec: "x264", Height: 720, Bitrate: 4_000_000, AudioCodec: "AAC", Channels: 2}
		w.w.ChangePlay(t, w.w.Seedbox, id, p)
		expectPlays(t, w.plays(t), clip+` transcode (direct direct burned) `+who+` subtitle external SRT en`+
			` received {Container:TS VideoCodec:x264 Height:720 Bitrate:4000000 AudioCodec:AAC Channels:2}, open`)

		err = w.w.Apply(t.Context(), w.w.Seedbox, &msv1.Report{ServerId: server.Bytes(), Report: &msv1.Report_PlayStarted{PlayStarted: &msv1.PlayStarted{
			Session: &msv1.Session{
				PlayId: storage.NewUUID().Bytes(), SessionKey: "10", ServerPath: clip, StartedUnixMs: now.UnixMilli(), SeenUnixMs: now.UnixMilli(),
				Streams: &msv1.Streams{Video: "copy", Audio: "direct", Subtitles: "none"},
			},
		}}})
		if err == nil {
			t.Error("a play whose video decision is copy was recorded")
		}
	})
}
