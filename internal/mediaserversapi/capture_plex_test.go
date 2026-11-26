package mediaserversapi_test

import (
	"encoding/json"
	"maps"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	fakeplex "github.com/jatassi/slipstream/fakes/plex"
	fakeplextv "github.com/jatassi/slipstream/fakes/plextv"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/storage"
)

// shelvesPlex is Living room Plex with a Shelves section.
var shelvesPlex = fakeplex.Server{
	Name: "Living room", Version: "1.43.5.11079", Identifier: plexID, Owner: "plex-admin", Root: "/media",
	Sections: []fakeplex.Section{{Key: "1", Title: "Shelves", Folders: []string{shelves}, Scanned: true}},
}

// plexCaptured is Living room Plex with Clip, its French Sidecar and
// Heat's two versions, which Plex holds as one item, on its Shelves,
// saved with seedbox reaching it.
type plexCaptured struct {
	plexRig
	id string
}

func newPlexCaptured(t *testing.T) plexCaptured {
	t.Helper()
	r := newPlexRigOf(t, shelvesPlex, fakeplextv.Account{Username: "plex-admin", PlexPass: true})
	for _, p := range []string{clip, clipFR, heatUHD, heatHD} {
		place(t, r.plex.Server.Media, strings.TrimPrefix(p, "/media/"))
	}
	r.plex.Server.Cue(t, "plex-scan", fakeplex.Scan{Section: "1"})
	r.plex.Server.Cue(t, "plex-streams", map[string]any{"path": clip, "streams": []map[string]any{
		{"type": "video", "codec": "h264", "height": 1080},
		{"type": "audio", "codec": "aac", "channels": 2, "language": "eng", "default": true},
		{"type": "audio", "codec": "dca", "channels": 6, "language": "fra"},
		{"type": "subtitle", "codec": "pgs", "language": "eng"},
	}})
	settle(time.Second)
	r.signIn(t)
	c := plexCaptured{plexRig: r, id: r.addPlex(t, r.w.Seedbox.String()).ID}
	synctest.Wait()
	return c
}

// play cues a play of Clip by ada on Living room, an Android TV, with
// args over those.
func (c plexCaptured) play(t *testing.T, args map[string]any) fakeplex.Played {
	t.Helper()
	cue := map[string]any{
		"path": clip, "user": "ada", "player": "Living room", "playerKey": "living-room", "platform": "Android TV",
		"video": link.StreamDirect, "audio": link.StreamDirect, "subtitles": link.SubtitlesNone,
	}
	maps.Copy(cue, args)
	raw, err := json.Marshal(c.plex.Server.Cue(t, "plex-play", cue))
	var p fakeplex.Played
	if err == nil {
		err = json.Unmarshal(raw, &p)
	}
	if err != nil {
		t.Fatal(err)
	}
	synctest.Wait()
	return p
}

func (c plexCaptured) stop(t *testing.T, p fakeplex.Played) {
	t.Helper()
	c.plex.Server.Cue(t, "plex-stop", fakeplex.Stop{Session: p.Session})
	wait(every)
	wait(every)
}

func (c plexCaptured) plays(t *testing.T) []string {
	t.Helper()
	u, err := storage.ParseUUID(c.id)
	if err != nil {
		t.Fatal(err)
	}
	return c.w.Plays(t, link.ServerID{UUID: u})
}

// TestPlexRecordsWhatItReports plays Clip once per row and reads the last
// play back, as the recorded run of 2026-11-26 read Plex. Each stream's
// decision is the TranscodeSession's copy or transcode, the container
// changed when the Part's decision is transcode, and a burn is its
// subtitleDecision. The player is its machineIdentifier. What the player
// received is the TranscodeSession's names with the selected Media's
// height and bitrate, which Plex gives in kbps: a re-encoded video's
// TranscodeSession has no height. A session's stream ids are strings
// where the item's are numbers, and each track is a position among the
// file's streams of its type or the Sidecar it is. A direct play that
// carries its player's last TranscodeSession is a direct play.
func TestPlexRecordsWhatItReports(t *testing.T) {
	for _, row := range []struct {
		name   string
		before map[string]any
		cue    map[string]any
		want   string
	}{
		{"direct play", nil, nil, "direct_play (direct direct none) []"},
		{"direct stream", nil, map[string]any{"containerChanged": true}, "direct_stream (direct direct none) []"},
		{
			"audio-only transcode", nil,
			map[string]any{
				"audio": link.StreamTranscode, "containerChanged": true, "audioTrack": 1,
				"received": map[string]any{"container": "mp4", "videoCodec": "h264", "height": 1080, "bitrate": 5742, "audioCodec": "aac", "channels": 2},
			},
			"transcode (direct transcode none) [] audio 1 received {Container:MP4 VideoCodec:x264 Height:1080 Bitrate:5742000 AudioCodec:AAC Channels:2}",
		},
		{
			"video transcode", nil,
			map[string]any{
				"video": link.StreamTranscode, "containerChanged": true,
				"received": map[string]any{"container": "mp4", "videoCodec": "h264", "height": 404, "bitrate": 1459, "audioCodec": "aac", "channels": 2},
			},
			"transcode (transcode direct none) [] received {Container:MP4 VideoCodec:x264 Height:404 Bitrate:1459000 AudioCodec:AAC Channels:2}",
		},
		{
			"burned subtitles", nil,
			map[string]any{"video": link.StreamTranscode, "subtitles": link.SubtitlesBurned, "containerChanged": true, "subtitleTrack": 0},
			"transcode (transcode direct burned) [] subtitle 0",
		},
		{
			"chosen tracks", nil,
			map[string]any{"subtitles": link.SubtitlesDirect, "audioTrack": 1, "subtitleTrack": -1},
			"direct_play (direct direct direct) [] audio 1 subtitle external SRT fr",
		},
		{
			"direct play after a transcode",
			map[string]any{"video": link.StreamTranscode, "containerChanged": true, "received": map[string]any{"container": "mp4", "videoCodec": "h264", "height": 404}},
			nil, "direct_play (direct direct none) []",
		},
	} {
		t.Run(row.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				c := newPlexCaptured(t)
				if row.before != nil {
					c.stop(t, c.play(t, row.before))
				}
				p := c.play(t, row.cue)
				head, tail, _ := strings.Cut(row.want, "]")
				want := clip + " " + head + `] by ada on Living room key living-room platform Android TV items ["` + p.Item + `"]` + tail + ", open"
				got := c.plays(t)
				if len(got) == 0 || got[len(got)-1] != want {
					t.Errorf("the play history is %q, want it to end %q", got, want)
				}
			})
		})
	}
}

// TestAPlexSessionWithNoMediaSelectedWaits plays Heat's 1080p file from
// the item that holds both of its versions. While the session lists both
// Media with none selected, a new session starts no play, and a live play
// stays open over many polls until the session ends.
func TestAPlexSessionWithNoMediaSelectedWaits(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := newPlexCaptured(t)
		waiting := c.play(t, map[string]any{"path": heatUHD, "version": heatHD, "unselected": true})
		wait(every)
		wait(every)
		expectPlays(t, c.plays(t))
		c.plex.Server.Cue(t, "plex-stop", fakeplex.Stop{Session: waiting.Session})
		wait(every)
		live := c.play(t, map[string]any{"path": heatUHD, "version": heatHD})
		open := heatHD + ` direct_play (direct direct none) [] by ada on Living room key living-room platform Android TV items ["` + live.Item + `"]`
		expectPlays(t, c.plays(t), open+", open")
		c.plex.Server.Cue(t, "plex-select", fakeplex.Select{Session: live.Session})
		for range 4 {
			wait(every)
		}
		expectPlays(t, c.plays(t), open+", open")
		c.stop(t, live)
		if got := c.plays(t); len(got) != 1 || !strings.HasPrefix(got[0], open+", ended ") {
			t.Errorf("once the session ends the play history is %q, want %s ended", got, open)
		}
	})
}
