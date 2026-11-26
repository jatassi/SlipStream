package mediaserver_test

import (
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/internal/mediaserver"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

// TestStreamsSurviveScans sets Metropolis's streams, then deletes and
// restores the file across two full scans: the file's part reports the
// streams before the scans and after them.
func TestStreamsSurviveScans(t *testing.T) {
	streams := []mediaserver.Stream{
		{Type: "video", Codec: "hevc", Profile: "Main 10", Height: 2160},
		{Type: "audio", Codec: "eac3", Channels: 6, Language: "eng", Default: true},
	}
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, jellyfin, true, metropolis)
		w.s.SetStreams(path(t, "/media/"+metropolis), streams)
		reported := func(when string) {
			t.Helper()
			it, ok := w.item(metropolis)
			if !ok || !slices.Equal(it.Parts[0].Streams, streams) {
				t.Errorf("%s, Metropolis is held %v with parts %+v, want streams %+v", when, ok, it.Parts, streams)
			}
		}
		reported("at once")
		w.remove(metropolis)
		w.fullScan()
		if w.has(metropolis) {
			t.Fatal("a full scan kept the deleted Metropolis")
		}
		w.place(metropolis)
		w.fullScan()
		reported("after the file came back")
	})
}

// TestSkipLeavesAPathOut skips Heat before the server's first call: its
// first scan and every later one list Metropolis beside it and not Heat.
func TestSkipLeavesAPathOut(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world{t: t, media: t.TempDir()}
		w.place(metropolis)
		w.place(heat)
		w.s = mediaserver.New(jellyfin, mediaserver.Disk{Media: w.media, Root: path(t, "/media")},
			[]mediaserver.Library{{Key: "1", Name: "Films", Type: "movie", Folders: []link.ServerPath{path(t, "/media/movies")}, Scanned: true}})
		w.s.Skip([]link.ServerPath{path(t, "/media/"+heat)})
		listed := func(when string) {
			t.Helper()
			if !w.has(metropolis) || w.has(heat) {
				t.Errorf("after %s, Metropolis is held %v and Heat %v, want Metropolis alone", when, w.has(metropolis), w.has(heat))
			}
		}
		listed("the first scan")
		w.fullScan()
		listed("a full scan")
	})
}

func (w world) fullScan() {
	w.t.Helper()
	w.scan()
	time.Sleep(2 * time.Second)
	synctest.Wait()
}
