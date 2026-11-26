package mediaserversapi_test

import (
	"net/http"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/mstest"
	"github.com/jatassi/slipstream/internal/mediaservers/playback"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// hd is a 1080p HEVC file with a 5.1 E-AC-3 track in English and a
// Spanish one, and an English PGS subtitle.
var hd = media.Record{
	Version: 1, Container: "matroska,webm", Bitrate: 8_000_000,
	Video: media.Video{Codec: "hevc", Profile: "Main 10", Width: 1920, Height: 1080, BitDepth: 10, HDR: []release.HDR{"HDR10"}},
	Audio: []media.Audio{
		{Codec: "eac3", Channels: 6, Language: "en", Default: true},
		{Codec: "aac", Channels: 2, Language: "es"},
	},
	Subtitles: []media.Subtitle{{Codec: "hdmv_pgs_subtitle", Image: true, Language: "en"}},
}

// uhd is hd's upgrade.
var uhd = media.Record{
	Version: 1, Container: "matroska,webm", Bitrate: 40_000_000,
	Video: media.Video{Codec: "hevc", Width: 3840, Height: 2160, BitDepth: 10, HDR: []release.HDR{"DV", "HDR10"}},
}

// hdFacts is hd as a play of its default audio track and no subtitle
// keeps it.
var hdFacts = playback.Facts{
	Container: release.MKV, Bitrate: 8_000_000,
	Video: playback.Video{Codec: "x265", Profile: "Main 10", BitDepth: 10, Resolution: release.R1080p, Height: 1080, HDR: []release.HDR{"HDR10"}},
	Audio: &playback.Audio{Codec: "DD+", Channels: 6, Language: "en"},
}

// snapped is Den with Clip indexed as an hd file of Heat, which meets the
// Target HD.
type snapped struct {
	watched
	server link.ServerID
	file   files.ID
	work   works.WorkID
	item   works.ItemID
}

func newSnapped(t *testing.T) snapped {
	t.Helper()
	s := snapped{watched: newWatched(t), work: works.WorkID{UUID: storage.NewUUID()}, item: works.ItemID{UUID: storage.NewUUID()}}
	s.server = serverOf(t, s.id)
	s.w.Target(t, "HD", targets.Target{})
	s.file = s.put(t, hd, "1080p")
	path, err := link.ParsePath(clip)
	if err != nil {
		t.Fatal(err)
	}
	s.w.Index(t, s.server, path, s.file, "heat")
	return s
}

func (s snapped) put(t *testing.T, r media.Record, label string) files.ID {
	t.Helper()
	return s.w.File(t, mstest.File{ID: s.file, Work: s.work, Item: s.item, Label: label, Media: r, Met: []string{"HD"}})
}

func (s snapped) play(audio, subtitle *link.Track) mstest.Play {
	now := time.Now()
	return mstest.Play{
		Server: s.server, Session: "9", Path: clip, Item: "heat", PlayerKey: "lrtv", Player: "Living room", User: "ada",
		Streams: link.Streams{Video: link.StreamTranscode, Audio: link.StreamDirect, Subtitles: link.SubtitlesNone},
		Audio:   audio, Subtitle: subtitle, Started: now, Seen: now,
	}
}

// look has seedbox's facts list play id, as its holder's next look does.
func (s snapped) look(id link.PlayID) {
	now := time.Now().UnixMilli()
	s.w.SetFacts(s.w.Seedbox, factsOf(s.server, &msv1.Session{
		PlayId: id.Bytes(), SessionKey: "9", ServerPath: clip, User: "ada", Player: "Living room",
		StartedUnixMs: now, SeenUnixMs: now, Streams: &msv1.Streams{Video: "transcode", Audio: "direct", Subtitles: "none"},
	}))
}

// changed is how many media-servers.plays-changed named Den.
func (s snapped) changed() int {
	n := 0
	for _, e := range s.w.PlaysChanged() {
		if len(e.ServerIDs) == 1 && e.ServerIDs[0] == s.id {
			n++
		}
	}
	return n
}

// TestAPlayKeepsTheSnapshotOfItsStart plays Clip through each run of
// reports, upgrades, failures and looks, and reads back its file and
// snapshot: taken once and kept, retried while the play is open, and
// never refusing the play.
func TestAPlayKeepsTheSnapshotOfItsStart(t *testing.T) {
	external := &link.Track{External: &link.ExternalTrack{Codec: "SRT", Language: "fr"}}
	withSpanish := hdFacts
	withSpanish.Audio = &playback.Audio{Codec: "AAC", Channels: 2, Language: "es"}
	withSpanish.Subtitle = &playback.Subtitle{Codec: release.PGS, Image: true, Language: "en"}
	withFrench := hdFacts
	withFrench.Subtitle = &playback.Subtitle{Codec: release.SRT, External: true, Language: "fr"}
	noAudio := hdFacts
	noAudio.Audio = nil
	for _, tc := range []struct {
		name            string
		audio, subtitle *link.Track
		// steps run after the start, one of upgrade, gone, fail, recover,
		// change, end and look; a step before "start" runs before it.
		before, steps []string
		want          *playback.Facts
		logged        bool
	}{
		{name: "the snapshot taken at the start survives the file's upgrade and delete", steps: []string{"upgrade", "change", "gone", "change", "look", "end"}, want: &hdFacts},
		{name: "the chosen tracks are the snapshot's", audio: &link.Track{Position: 1}, subtitle: &link.Track{Position: 0}, want: &withSpanish},
		{name: "an external subtitle keeps the track's own codec and language", subtitle: external, want: &withFrench},
		{name: "a track past the probe's count is no track", audio: &link.Track{Position: 2}, subtitle: &link.Track{Position: 1}, want: &noAudio},
		{name: "a snapshot that can't be taken leaves the play filed with none", before: []string{"fail"}, steps: []string{"look"}, logged: true},
		{name: "the next report takes it once the file recovers", before: []string{"fail"}, steps: []string{"recover", "change"}, want: &hdFacts, logged: true},
		{name: "the next look takes it once the file recovers", before: []string{"fail"}, steps: []string{"recover", "look"}, want: &hdFacts, logged: true},
		{name: "an ended play stays as it is", before: []string{"fail"}, steps: []string{"end", "recover", "look", "change"}, logged: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				s := newSnapped(t)
				play := s.play(tc.audio, tc.subtitle)
				var id link.PlayID
				for _, step := range tc.before {
					s.step(t, step, id, play)
				}
				id = s.w.Play(t, s.w.Seedbox, play)
				for _, step := range tc.steps {
					s.step(t, step, id, play)
				}
				s.expectSnapshot(t, id, tc.want)
				if got := strings.Contains(s.w.Logged(), "take the snapshot of a play"); got != tc.logged {
					t.Errorf("logged a failed snapshot = %t, want %t", got, tc.logged)
				}
			})
		})
	}
}

// TestALateFileResolvesWhenTheIndexLearnsIt plays Trip, which Den
// scanned once it landed but its path index doesn't hold: the play asks
// a lookup, and once the lookup learns the path the play has Trip's file
// and snapshot, and media-servers.plays-changed names Den.
func TestALateFileResolvesWhenTheIndexLearnsIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newBrowsed(t, "jellyfin", floorVer)
		const rel = "Trip/Trip.mkv"
		place(t, s.den.Media, "Shelves/"+rel)
		file := s.w.Place(s.root, mustRel(t, rel))
		s.tell(t, at(t, files.PathAdded, file, rel))
		settle(3 * time.Minute)
		s.w.Target(t, "HD", targets.Target{})
		work, item := works.WorkID{UUID: storage.NewUUID()}, works.ItemID{UUID: storage.NewUUID()}
		s.w.File(t, mstest.File{ID: file, Work: work, Item: item, Label: "1080p", Media: hd, Met: []string{"HD"}})
		server := serverOf(t, s.id)
		before := len(s.w.PlaysChanged())
		now := time.Now()
		id := s.w.Play(t, s.w.Seedbox, mstest.Play{
			Server: server, Session: "9", Path: p + rel, PlayerKey: "lrtv", Player: "Living room", User: "ada",
			Streams: link.Streams{Video: link.StreamDirect, Audio: link.StreamDirect, Subtitles: link.SubtitlesNone},
			Started: now, Seen: now,
		})
		if f, snap := s.w.Filed(t, server, id); !f.IsZero() || snap != nil {
			t.Fatalf("before the lookup the play has file %s and snapshot %v, want neither", f, snap)
		}
		settle(time.Minute)
		f, snap := s.w.Filed(t, server, id)
		if f != file || snap == nil || snap.Work != work || snap.Label != "1080p" {
			t.Errorf("after the lookup the play has file %s and snapshot %+v, want Trip's %s with its snapshot", f, snap, file)
		}
		named := false
		for _, e := range s.w.PlaysChanged()[before+1:] {
			named = named || len(e.ServerIDs) == 1 && e.ServerIDs[0] == s.id
		}
		if !named {
			t.Errorf("media-servers.plays-changed after the start published %v, want Den", s.w.PlaysChanged()[before+1:])
		}
	})
}

// TestEveryWriteToAPlayPublishesPlaysChanged writes Clip's play each way
// and counts media-servers.plays-changed naming Den after the commit.
func TestEveryWriteToAPlayPublishesPlaysChanged(t *testing.T) {
	for _, tc := range []struct {
		name string
		// before runs before the play starts.
		before func(t *testing.T, s snapped)
		write  func(t *testing.T, s snapped, id link.PlayID)
		want   int
	}{
		{name: "a change", write: func(t *testing.T, s snapped, id link.PlayID) {
			s.w.ChangePlay(t, s.w.Seedbox, id, s.play(nil, nil))
		}, want: 1},
		{name: "an end", write: func(t *testing.T, s snapped, id link.PlayID) {
			s.w.EndPlay(t, s.w.Seedbox, s.server, id, time.Now())
		}, want: 1},
		{name: "a Watching that doesn't list it", write: func(t *testing.T, s snapped, _ link.PlayID) {
			apply(t, s, &msv1.Report{Report: &msv1.Report_Watching{Watching: &msv1.Watching{AtUnixMs: time.Now().UnixMilli()}}})
		}, want: 1},
		{name: "a look that takes its snapshot", before: func(t *testing.T, s snapped) { s.w.FailFile(t, s.file) }, write: func(t *testing.T, s snapped, id link.PlayID) {
			s.put(t, hd, "1080p")
			s.look(id)
		}, want: 1},
		{name: "a look at a play with its snapshot", write: func(_ *testing.T, s snapped, id link.PlayID) { s.look(id) }, want: 0},
		{name: "a refused change", write: func(t *testing.T, s snapped, id link.PlayID) {
			err := s.w.Apply(t.Context(), s.w.Seedbox, &msv1.Report{ServerId: s.server.Bytes(), Report: &msv1.Report_PlayChanged{PlayChanged: &msv1.PlayChanged{
				PlayId: id.Bytes(), SeenUnixMs: time.Now().UnixMilli(), Streams: &msv1.Streams{Video: "copy", Audio: "direct", Subtitles: "none"},
			}}})
			if err == nil {
				t.Fatal("a change with the video copy applied, want a refusal")
			}
			apply(t, s, &msv1.Report{Report: &msv1.Report_Contact{Contact: &msv1.Contact{State: string(link.ContactResponding), AtUnixMs: time.Now().UnixMilli()}}})
		}, want: 0},
		{name: "naming another reaching Node", write: func(t *testing.T, s snapped, _ link.PlayID) {
			if status, body := s.o.Call(t, http.MethodPatch, servers+"/"+s.id, `{"reachingNodeId":"`+s.w.Home.String()+`"}`); status != http.StatusOK {
				t.Fatalf("name home = %d %s", status, body)
			}
		}, want: 1},
		{name: "seedbox away past Mark unhealthy after", write: func(_ *testing.T, s snapped, _ link.PlayID) {
			s.w.Stop(s.w.Seedbox)
			s.w.Offline(s.w.Seedbox)
			wait(5 * time.Minute)
		}, want: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				s := newSnapped(t)
				if tc.before != nil {
					tc.before(t, s)
				}
				id := s.w.Play(t, s.w.Seedbox, s.play(nil, nil))
				synctest.Wait()
				started := s.changed()
				if started != 1 {
					t.Errorf("the start published plays-changed for Den %d times, want once", started)
				}
				tc.write(t, s, id)
				synctest.Wait()
				if got := s.changed() - started; got != tc.want {
					t.Errorf("%s published plays-changed for Den %d times, want %d", tc.name, got, tc.want)
				}
			})
		})
	}
}

func (s snapped) step(t *testing.T, step string, id link.PlayID, p mstest.Play) {
	t.Helper()
	switch step {
	case "upgrade":
		s.put(t, uhd, "2160p")
	case "gone":
		s.w.Gone(t, s.file)
	case "fail":
		s.w.FailFile(t, s.file)
	case "recover":
		s.put(t, hd, "1080p")
	case "change":
		p.Seen = time.Now()
		s.w.ChangePlay(t, s.w.Seedbox, id, p)
	case "end":
		s.w.EndPlay(t, s.w.Seedbox, s.server, id, time.Now())
	case "look":
		s.look(id)
	default:
		t.Fatalf("no step %q", step)
	}
	wait(time.Second)
}

// expectSnapshot fails unless play id is filed under Clip with a snapshot
// of want, which meets HD, or none for nil.
func (s snapped) expectSnapshot(t *testing.T, id link.PlayID, want *playback.Facts) {
	t.Helper()
	file, snap := s.w.Filed(t, s.server, id)
	if file != s.file {
		t.Errorf("the play's file is %s, want Clip's %s", file, s.file)
	}
	switch {
	case want == nil && snap != nil:
		t.Errorf("the play has the snapshot %+v, want none", *snap)
	case want != nil && snap == nil:
		t.Error("the play has no snapshot")
	case want != nil:
		got := *snap
		if len(got.Targets) != 1 || got.Targets[0].Name != "HD" {
			t.Errorf("the snapshot met %+v, want HD", got.Targets)
		}
		got.Targets = nil
		if w := (playback.Snapshot{Work: s.work, Item: s.item, Label: "1080p", Facts: *want}); !reflect.DeepEqual(got, w) {
			t.Errorf("the snapshot is %+v, want %+v", got, w)
		}
	}
}

func apply(t *testing.T, s snapped, r *msv1.Report) {
	t.Helper()
	r.ServerId = s.server.Bytes()
	if err := s.w.Apply(t.Context(), s.w.Seedbox, r); err != nil {
		t.Fatal(err)
	}
}
