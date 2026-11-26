package mediaserversapi_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/mstest"
	"github.com/jatassi/slipstream/internal/storage"
)

// TestAPlayRecordsWhatItsNodeReports starts a direct stream of Clip on
// Den and changes it to burn in an external subtitle: each row keeps the
// player, item, tracks and received facts the Node reported, with the
// decision its streams give.
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
	})
}
