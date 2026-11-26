package emby_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/emby"
	"github.com/jatassi/slipstream/fakes/emby"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/internal/mediabrowser"
	"github.com/jatassi/slipstream/fakes/internal/mediaserver"
	"github.com/jatassi/slipstream/fakes/internal/serverfake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

func TestFakeEmbyConforms(t *testing.T) {
	for _, version := range emby.Versions {
		for _, root := range []string{"/media", `D:\Media`} {
			name := version + "/" + map[bool]string{false: "posix", true: "windows"}[root[1] == ':']
			t.Run(name+"/loopback", func(t *testing.T) {
				f := faketest.Loopback(t, emby.Playing(server(version, root, 2*time.Second)), scenario.Scenario{Seed: "emby"}, faketest.WithMedia(seeded(t)), faketest.As("Den"))
				conformance.Run(t, target(t, f, root, 3*time.Second, 30*time.Second))
			})
			t.Run(name+"/in memory", func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					f := faketest.InMemory(t, emby.Playing(server(version, root, 0)), scenario.Scenario{Seed: "emby"}, faketest.WithMedia(seeded(t)), faketest.As("Den"))
					conformance.Run(t, target(t, f, root, 2*time.Minute, 5*time.Minute))
				})
			})
		}
	}
}

func join(root, rel string) string {
	sep := "/"
	if root[1] == ':' {
		sep = `\`
	}
	return root + sep + rel
}

func server(version, root string, delay time.Duration) emby.Server {
	return emby.Server{
		Name: "Den", Version: version, ID: "5f1b0c3e2a7d4e6f8a9b0c1d2e3f4a5b", Keys: []string{"conformance-key"},
		Users:        []emby.User{{Name: "jackson", Password: "conformance-password", Admin: true}},
		MonitorDelay: delay, Root: root,
		Libraries: []emby.Library{
			{Name: "Movies", Type: "movies", Folders: []string{join(root, "Movies")}, Scanned: true},
			{Name: "Fresh", Type: "movies", Folders: []string{join(root, "Fresh")}},
			{Name: "Movies 4K", Type: "movies", Folders: []string{join(root, "Movies 4K")}, Scanned: true},
		},
	}
}

// seeded is a media dir whose Movies and Movies 4K folders each hold one
// film and whose Fresh folder is empty.
func seeded(t *testing.T) string {
	t.Helper()
	media := t.TempDir()
	if err := os.MkdirAll(filepath.Join(media, "Fresh"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, lib := range []string{"Movies", "Movies 4K"} {
		file := filepath.Join(media, lib, "Metropolis (1927)", "Metropolis (1927).mkv")
		if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte("Metropolis "+lib), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return media
}

func path(t *testing.T, s string) link.ServerPath {
	t.Helper()
	p, err := link.ParsePath(s)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func target(t *testing.T, f faketest.Served, root string, settle, bound time.Duration) conformance.Target {
	t.Helper()
	return conformance.Target{
		Address: f.Address.URL(""), Key: "conformance-key", HTTP: f.Client,
		Library: "Movies", Folder: path(t, join(root, "Movies")),
		Disk:   conformance.DirDisk{Dir: filepath.Join(f.Media, "Movies")},
		Fresh:  &conformance.Library{Library: "Fresh", Folder: path(t, join(root, "Fresh")), Disk: conformance.DirDisk{Dir: filepath.Join(f.Media, "Fresh")}},
		Second: &conformance.Library{Library: "Movies 4K", Folder: path(t, join(root, "Movies 4K")), Disk: conformance.DirDisk{Dir: filepath.Join(f.Media, "Movies 4K")}},
		Admin:  &conformance.Admin{User: "jackson", Password: "conformance-password"},
		Play: func(t *testing.T, p conformance.Play) (conformance.Played, func()) {
			if p.Audio != nil {
				f.Cue(t, "emby-streams", serverfake.SetStreams{Path: p.Path.String(), Streams: audioTracks(p.Audio.Position)})
			}
			played, _ := f.Cue(t, "emby-play", cued(p)).(emby.Played)
			return conformance.Played{Session: played.Session, Item: played.Item}, func() { f.Cue(t, "emby-stop", emby.Stop{Session: played.Session}) }
		},
		Fake: true, Settle: settle, Bound: bound,
		Log: slog.New(slog.NewTextHandler(t.Output(), nil)),
	}
}

// cued is the play cue for p, by jackson on Living Room.
func cued(p conformance.Play) emby.Play {
	return emby.Play{
		Path: p.Path.String(), Version: p.Version.String(), User: "jackson", Player: "Living Room", PlayerKey: p.PlayerKey, Platform: p.Platform,
		Video: p.Streams.Video, Audio: p.Streams.Audio, Subtitles: p.Streams.Subtitles, ContainerChanged: p.Streams.ContainerChanged,
		Received: mediabrowser.ReceivedOf(p.Received), AudioTrack: serverfake.Track(p.Audio), SubtitleTrack: serverfake.Track(p.Subtitle), Reasons: p.Reasons,
	}
}

// audioTracks are a file's streams with an audio track at position.
func audioTracks(position int) []mediaserver.Stream {
	out := []mediaserver.Stream{{Type: "video", Codec: "h264", Height: 1080}}
	for range position + 1 {
		out = append(out, mediaserver.Stream{Type: "audio", Codec: "aac", Channels: 2, Language: "eng"})
	}
	return out
}
