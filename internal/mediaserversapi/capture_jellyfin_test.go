package mediaserversapi_test

import (
	"encoding/json"
	"maps"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/jellyfin"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/storage"
)

// product is Jellyfin or Emby, with the route a version is read from.
type product struct {
	kind, version, itemRoute string
	serve                    func(r rig, t *testing.T, srv jellyfin.Server, placed ...string) faketest.Served
}

var products = []product{
	{"jellyfin", floorVer, "item", func(r rig, t *testing.T, srv jellyfin.Server, placed ...string) faketest.Served {
		return r.w.Jellyfin(t, host, srv, placed...)
	}},
	{"emby", "4.10.1.0", "user-item", func(r rig, t *testing.T, srv jellyfin.Server, placed ...string) faketest.Served {
		return r.w.Emby(t, host, srv, placed...)
	}},
}

const (
	clipFR   = shelves + "/Clip.fr.srt"
	heatUHD  = shelves + "/Heat (1995)/Heat (1995) - 2160p.mkv"
	heatHD   = shelves + "/Heat (1995)/Heat (1995) - 1080p.mkv"
	pgsCodec = "PGSSUB"
)

// captured is Den as a product, with Clip, its French Sidecar and Heat's
// two versions on its Shelves, saved with seedbox reaching it.
type captured struct {
	rig
	f   product
	den faketest.Served
	id  string
}

func newCaptured(t *testing.T, f product) captured {
	t.Helper()
	r := newRig(t)
	srv := den(f.version)
	srv.Libraries[0].Scanned = true
	c := captured{rig: r, f: f, den: f.serve(r, t, srv, clip, clipFR, heatUHD, heatHD)}
	c.den.Cue(t, f.kind+"-streams", map[string]any{"path": clip, "streams": []map[string]any{
		{"type": "video", "codec": "h264", "height": 1080},
		{"type": "audio", "codec": "aac", "channels": 2, "language": "eng", "default": true},
		{"type": "audio", "codec": "dts", "channels": 6, "language": "fra"},
		{"type": "subtitle", "codec": pgsCodec, "language": "eng"},
	}})
	draft := `{"kind":"` + f.kind + `","name":"Den","address":"` + address + `","reachingNodeId":"` + r.w.Seedbox.String() + `","apiKey":"` + denKey + `"}`
	status, body := r.o.Call(t, http.MethodPost, servers, draft)
	if status != http.StatusCreated {
		t.Fatalf("add Den = %d %s", status, body)
	}
	c.id = ownertest.FirstID(body)
	synctest.Wait()
	return c
}

func eachProduct(t *testing.T, test func(t *testing.T, c captured)) {
	for _, f := range products {
		t.Run(f.kind, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) { test(t, newCaptured(t, f)) })
		})
	}
}

// play cues a play of Clip by ada on Living room, whose player has
// reported its playback, with args over those.
func (c captured) play(t *testing.T, args map[string]any) played {
	t.Helper()
	cue := map[string]any{
		"path": clip, "user": "ada", "player": "Living room", "playerKey": "living-room", "platform": "Android TV", "unreported": false,
		"video": link.StreamDirect, "audio": link.StreamDirect, "subtitles": link.SubtitlesNone,
	}
	maps.Copy(cue, args)
	raw, err := json.Marshal(c.den.Cue(t, c.f.kind+"-play", cue))
	var p played
	if err == nil {
		err = json.Unmarshal(raw, &p)
	}
	if err != nil {
		t.Fatal(err)
	}
	synctest.Wait()
	return p
}

func (c captured) plays(t *testing.T) []string {
	t.Helper()
	u, err := storage.ParseUUID(c.id)
	if err != nil {
		t.Fatal(err)
	}
	return c.w.Plays(t, link.ServerID{UUID: u})
}

// failLookups has Den answer 500 to every read of a version's item.
func (c captured) failLookups(t *testing.T) {
	t.Helper()
	rule := fault.Rule{ID: "lookup", Target: fault.Target(c.f.kind), Route: c.f.itemRoute, Fault: fault.Status{Code: http.StatusInternalServerError}}
	if _, err := c.den.Faults.Set(rule); err != nil {
		t.Fatal(err)
	}
}

// TestJellyfinAndEmbyRecordWhatTheyReport plays Clip once per row on each
// product and reads the play back. The decision comes from the streams
// both products report, never from PlayMethod alone: a direct play reads
// DirectStream on Emby, and a remux reads Transcode with both streams
// direct (recorded runs 2026-11-26). The player is its DeviceId, and the
// platform stays empty, since neither product sends DeviceType. Received
// facts are in release words, and each track is a position among the
// file's streams of its type or the external file it is.
func TestJellyfinAndEmbyRecordWhatTheyReport(t *testing.T) {
	for _, row := range []struct {
		name string
		cue  map[string]any
		want string
	}{
		{"direct play", nil, "direct_play (direct direct none) []"},
		{
			"direct stream",
			map[string]any{"containerChanged": true, "reasons": []string{"ContainerNotSupported"}},
			`direct_stream (direct direct none) ["ContainerNotSupported"]`,
		},
		{
			"audio-only transcode",
			map[string]any{
				"audio": link.StreamTranscode, "containerChanged": true, "reasons": []string{"AudioCodecNotSupported"}, "audioTrack": 1,
				"received": map[string]any{"container": "ts", "videoCodec": "h264", "height": 1080, "bitrate": 5742719, "audioCodec": "aac", "channels": 6},
			},
			`transcode (direct transcode none) ["AudioCodecNotSupported"]` + " audio 1 received {Container:TS VideoCodec:x264 Height:1080 Bitrate:5742719 AudioCodec:AAC Channels:6}",
		},
		{
			"video transcode",
			map[string]any{
				"video": link.StreamTranscode, "containerChanged": true, "reasons": []string{"ContainerBitrateExceedsLimit"},
				"received": map[string]any{"container": "mp4", "videoCodec": "h264", "height": 720, "bitrate": 1628002, "audioCodec": "aac", "channels": 2},
			},
			`transcode (transcode direct none) ["ContainerBitrateExceedsLimit"]` + " received {Container:MP4 VideoCodec:x264 Height:720 Bitrate:1628002 AudioCodec:AAC Channels:2}",
		},
		{
			"burned subtitles",
			map[string]any{"video": link.StreamTranscode, "subtitles": link.SubtitlesBurned, "containerChanged": true, "subtitleTrack": 0, "reasons": []string{"SubtitleCodecNotSupported"}},
			`transcode (transcode direct burned) ["SubtitleCodecNotSupported"] subtitle 0`,
		},
		{
			"chosen tracks",
			map[string]any{"subtitles": link.SubtitlesDirect, "audioTrack": 1, "subtitleTrack": -1},
			"direct_play (direct direct direct) [] audio 1 subtitle external SRT fr",
		},
	} {
		t.Run(row.name, func(t *testing.T) {
			eachProduct(t, func(t *testing.T, c captured) {
				p := c.play(t, row.cue)
				head, tail, _ := strings.Cut(row.want, "]")
				want := clip + " " + head + "] by ada on Living room key living-room item " + p.Item + tail + ", open"
				expectPlays(t, c.plays(t), want)
			})
		})
	}
}

// TestAnAlternateVersionIsRecordedAgainstThePlayedFile plays each of
// Heat's versions from the other's item, so one of the two is a Jellyfin
// alternate: each play is the file played.
func TestAnAlternateVersionIsRecordedAgainstThePlayedFile(t *testing.T) {
	for _, pair := range [][2]string{{heatUHD, heatHD}, {heatHD, heatUHD}} {
		eachProduct(t, func(t *testing.T, c captured) {
			c.play(t, map[string]any{"path": pair[0], "version": pair[1]})
			if got := c.plays(t); len(got) != 1 || !strings.HasPrefix(got[0], pair[1]+" direct_play ") {
				t.Errorf("playing %s from %s, the play history is %q", pair[1], pair[0], got)
			}
		})
	}
}

// TestAFailedLookupRecordsNothingUntilItSucceeds plays Clip while Den
// fails every item read: nothing is recorded over two polls, and the poll
// after the reads succeed again records the play.
func TestAFailedLookupRecordsNothingUntilItSucceeds(t *testing.T) {
	eachProduct(t, func(t *testing.T, c captured) {
		c.failLookups(t)
		p := c.play(t, nil)
		wait(every)
		wait(every)
		expectPlays(t, c.plays(t))
		c.den.Faults.Clear("lookup")
		wait(every)
		expectPlays(t, c.plays(t), clip+" direct_play (direct direct none) [] by ada on Living room key living-room item "+p.Item+", open")
	})
}

// TestASwitchedVersionWhoseLookupFailsEndsTheOldPlay plays Heat's 2160p
// file, then has the player switch to its 1080p version while Den fails
// every item read: the 2160p play ends after two polls miss it, and
// nothing is recorded under either file.
func TestASwitchedVersionWhoseLookupFailsEndsTheOldPlay(t *testing.T) {
	eachProduct(t, func(t *testing.T, c captured) {
		old := c.play(t, map[string]any{"path": heatUHD})
		wait(every)
		c.failLookups(t)
		c.den.Cue(t, c.f.kind+"-stop", map[string]string{"session": old.Session})
		c.play(t, map[string]any{"path": heatUHD, "version": heatHD})
		wait(every)
		wait(every)
		if got := c.plays(t); len(got) != 1 || !strings.HasPrefix(got[0], heatUHD+" direct_play ") || !strings.HasSuffix(got[0], ", ended 15s in") {
			t.Errorf("the play history is %q, want only %s ended 15s in", got, heatUHD)
		}
	})
}
