package mediaserver_test

import (
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/internal/mediaserver"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

var (
	plex = mediaserver.Behavior{
		FreshSectionIgnores: true, StableKeyInFolder: true, PurgesVacated: true, ScanTime: 250 * time.Millisecond,
	}
	jellyfin = mediaserver.Behavior{
		MonitorDelay: time.Minute, RestartsOnRepeat: true, FreshRootIgnores: true, ScanDropsPending: true, ScanTime: time.Second,
	}
	emby = mediaserver.Behavior{MonitorDelay: 90 * time.Second, ScanTime: time.Second}
)

type world struct {
	t     *testing.T
	media string
	s     *mediaserver.Server
}

func newWorld(t *testing.T, b mediaserver.Behavior, scanned bool, seed ...string) world {
	t.Helper()
	w := world{t: t, media: t.TempDir()}
	for _, rel := range seed {
		w.place(rel)
	}
	w.s = mediaserver.New(b, mediaserver.Disk{Media: w.media, Root: path(t, "/media")},
		[]mediaserver.Library{{Key: "1", Name: "Films", Type: "movie", Folders: []link.ServerPath{path(t, "/media/movies")}, Scanned: scanned}})
	w.s.Libraries()
	return w
}

func path(t *testing.T, s string) link.ServerPath {
	t.Helper()
	p, err := link.ParsePath(s)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func (w world) place(rel string) {
	w.t.Helper()
	file := filepath.Join(w.media, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		w.t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(rel), 0o600); err != nil {
		w.t.Fatal(err)
	}
}

func (w world) move(to string) {
	w.t.Helper()
	dst := filepath.Join(w.media, filepath.FromSlash(to))
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		w.t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(w.media, filepath.FromSlash(metropolis)), dst); err != nil {
		w.t.Fatal(err)
	}
}

func (w world) remove(rel string) {
	w.t.Helper()
	if err := os.Remove(filepath.Join(w.media, filepath.FromSlash(rel))); err != nil {
		w.t.Fatal(err)
	}
}

func (w world) item(rel string) (mediaserver.Item, bool) {
	w.t.Helper()
	return w.s.ItemAt(path(w.t, "/media/"+rel))
}

func (w world) has(rel string) bool {
	w.t.Helper()
	_, ok := w.item(rel)
	return ok
}

func (w world) refresh(folder string) {
	w.t.Helper()
	if err := w.s.RefreshFolder("1", path(w.t, "/media/"+folder)); err != nil {
		w.t.Fatal(err)
	}
}

func (w world) report(rel string) {
	w.t.Helper()
	w.s.Report([]link.ServerPath{path(w.t, "/media/"+rel)})
}

func (w world) scan() {
	w.t.Helper()
	if _, err := w.s.RefreshLibrary("1"); err != nil {
		w.t.Fatal(err)
	}
}

const (
	metropolis = "movies/Metropolis (1927)/Metropolis (1927).mkv"
	heat       = "movies/Heat (1995)/Heat (1995).mkv"
)

func TestARepeatReportRestartsJellyfinsTimer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, jellyfin, true, metropolis)
		w.place(heat)
		w.report(heat)
		time.Sleep(40 * time.Second)
		w.report(heat)
		time.Sleep(30 * time.Second)
		if w.has(heat) {
			t.Fatal("Heat showed up 70 s after its first report, though the repeat at 40 s restarted its timer")
		}
		time.Sleep(31 * time.Second)
		if !w.has(heat) {
			t.Fatal("Heat didn't show up a minute after its repeat report")
		}
	})
}

func TestEmbysOneTimerIsntRestarted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, emby, true, metropolis)
		w.place(heat)
		w.report(heat)
		time.Sleep(60 * time.Second)
		w.report(heat)
		time.Sleep(31 * time.Second)
		if !w.has(heat) {
			t.Fatal("Heat didn't show up 90 s after its first report")
		}
	})
}

func TestAReportIntoAFreshJellyfinRootIsIgnoredUntilAScan(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, jellyfin, false)
		w.place(metropolis)
		w.report(metropolis)
		time.Sleep(2 * time.Minute)
		if w.has(metropolis) {
			t.Fatal("a report into a root never scanned with content made an item")
		}
		w.scan()
		time.Sleep(2 * time.Second)
		if !w.has(metropolis) {
			t.Fatal("the full scan didn't pick up Metropolis")
		}
		w.place(heat)
		w.report(heat)
		time.Sleep(61 * time.Second)
		if !w.has(heat) {
			t.Fatal("a report after a scan with content was ignored")
		}
	})
}

func TestAFullScanDropsJellyfinsPendingReports(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, jellyfin, true, metropolis)
		w.report(heat)
		w.scan()
		time.Sleep(30 * time.Second)
		w.place(heat)
		time.Sleep(31 * time.Second)
		if w.has(heat) {
			t.Fatal("a report sent before the full scan still scanned its folder after the delay")
		}
	})
}

func TestAWholeSectionScanEndsAfterItStarted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, plex, true, metropolis)
		asked := time.Now()
		w.scan()
		if st, _ := w.s.ScanState("1"); !st.Running {
			t.Fatal("the section isn't refreshing right after the scan started")
		}
		again, err := w.s.RefreshLibrary("1")
		if err != nil || !again.Equal(asked) {
			t.Fatalf("a second request started another scan at %v", again)
		}
		time.Sleep(300 * time.Millisecond)
		st, _ := w.s.ScanState("1")
		if st.Running || !st.Ended.After(asked) {
			t.Fatalf("scan state %+v 0.3 s after a 0.25 s scan started at %v", st, asked)
		}
	})
}

func TestAFirstPartialScanIntoANeverScannedSectionIsIgnored(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, plex, false)
		w.place(metropolis)
		w.refresh("movies/Metropolis (1927)")
		if w.has(metropolis) {
			t.Fatal("a partial scan into a never-scanned section made an item")
		}
		w.scan()
		time.Sleep(time.Second)
		w.place(heat)
		w.refresh("movies/Heat (1995)")
		if !w.has(metropolis) || !w.has(heat) {
			t.Fatal("the section scan and the partial scan after it didn't pick both up")
		}
	})
}

func watched(t *testing.T, w world) mediaserver.Item {
	t.Helper()
	it, ok := w.item(metropolis)
	if !ok {
		t.Fatal("no item at Metropolis")
	}
	w.s.Played("owner", it.Key, time.Now())
	w.s.AddToCollection("Silent", it.Key)
	return it
}

func TestPlexKeepsTheKeyOfARenameInOneFolder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, plex, true, metropolis)
		old := watched(t, w)
		renamed := "movies/Metropolis (1927)/Metropolis (1927) Bluray-1080p.mkv"
		w.move(renamed)
		w.refresh("movies/Metropolis (1927)")
		now, ok := w.item(renamed)
		if !ok || now.Key != old.Key || len(now.Parts) != 1 {
			t.Fatalf("after the rename the item is %+v, want key %s with one part", now, old.Key)
		}
		if w.s.UserData("owner", now.Key).PlayCount != 1 || len(w.s.Collections(now.Key)) != 1 {
			t.Fatal("the rename lost watch data or the collection")
		}
	})
}

func TestPlexCarriesADestinationFirstMoveToANewItem(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, plex, true, metropolis)
		old := watched(t, w)
		moved := "movies/Metropolis (1927) {edition-Test}/Metropolis (1927) {edition-Test}.mkv"
		w.move(moved)
		w.refresh("movies/Metropolis (1927) {edition-Test}")
		now, ok := w.item(moved)
		if !ok || now.Key == old.Key {
			t.Fatalf("the moved file is in %+v, want a new item", now)
		}
		if _, ok := w.s.Item(old.Key); ok {
			t.Fatal("the old item outlived the destination refresh")
		}
		if w.s.UserData("owner", now.Key).PlayCount != 1 || len(w.s.Collections(now.Key)) != 1 {
			t.Fatal("the new item didn't carry the play count and the collection")
		}
		w.refresh("movies/Metropolis (1927)")
		if after, _ := w.item(moved); after.Key != now.Key || w.s.UserData("owner", now.Key).PlayCount != 1 {
			t.Fatal("the source refresh after the destination changed the new item")
		}
	})
}

func TestPlexPurgesAVacatedFolderScannedFirst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, plex, true, metropolis)
		old := watched(t, w)
		moved := "movies/Metropolis/Metropolis (1927).mkv"
		w.move(moved)
		w.refresh("movies/Metropolis (1927)")
		if _, ok := w.s.Item(old.Key); ok {
			t.Fatal("a partial scan of the vacated folder kept the item")
		}
		w.refresh("movies/Metropolis")
		now, ok := w.item(moved)
		if !ok {
			t.Fatal("the destination refresh made no item")
		}
		if w.s.UserData("owner", now.Key).PlayCount != 0 || len(w.s.Collections(now.Key)) != 0 {
			t.Fatal("watch data survived the purge")
		}
	})
}

func TestWithoutAutoEmptyTrashASourceFirstMoveStillCarries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		b := plex
		b.PurgesVacated = false
		w := newWorld(t, b, true, metropolis)
		watched(t, w)
		moved := "movies/Metropolis/Metropolis (1927).mkv"
		w.move(moved)
		w.refresh("movies/Metropolis (1927)")
		w.refresh("movies/Metropolis")
		now, ok := w.item(moved)
		if !ok || w.s.UserData("owner", now.Key).PlayCount != 1 {
			t.Fatal("with the trash kept, the move lost its play count")
		}
	})
}

func TestEditionsMakeTheirOwnItemOnlyUnderPlexPass(t *testing.T) {
	cut := "movies/Heat (1995)/Heat (1995) {edition-Director's Cut}.mkv"
	for _, pass := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			b := plex
			b.Editions = pass
			w := newWorld(t, b, true, heat, cut)
			a, _ := w.item(heat)
			c, _ := w.item(cut)
			if (a.Key != c.Key) != pass {
				t.Fatalf("Plex Pass %v: the edition is item %s beside %s", pass, c.Key, a.Key)
			}
		})
	}
}

func TestAnItemRefreshDropsARemovedSidecar(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		srt := "movies/Heat (1995)/Heat (1995).en.srt"
		w := newWorld(t, plex, true, heat, srt)
		it, _ := w.item(heat)
		if len(it.Parts[0].Sidecars) != 1 {
			t.Fatalf("the part has Sidecars %v, want the .srt", it.Parts[0].Sidecars)
		}
		w.remove(srt)
		if !w.s.RefreshItem(it.Key) {
			t.Fatal("no item to refresh")
		}
		if it, _ = w.item(heat); len(it.Parts[0].Sidecars) != 0 {
			t.Fatalf("the part still has Sidecars %v", it.Parts[0].Sidecars)
		}
	})
}

func TestARestartDropsSocketsAndStopsAScan(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, plex, true, metropolis)
		sub := w.s.Subscribe()
		w.scan()
		w.s.Restart()
		select {
		case <-sub.Done:
		default:
			t.Fatal("the socket outlived the restart")
		}
		time.Sleep(time.Second)
		if st, _ := w.s.ScanState("1"); st.Running || !st.Ended.IsZero() {
			t.Fatalf("scan state %+v after a restart mid-scan", st)
		}
	})
}
