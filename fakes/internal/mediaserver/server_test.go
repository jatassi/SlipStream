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
		MonitorDelay: time.Minute, RestartsOnRepeat: true, ResolvesNearestItem: true, ScanDropsPending: true,
		NewIDOnPathChange: true, Carry: mediaserver.CarryRemovedFirst, PurgesVacated: true, ScanTime: time.Second,
	}
	emby = mediaserver.Behavior{
		MonitorDelay: 90 * time.Second, DeletesAtOnce: true, ItemPerFile: true, NewIDOnPathChange: true,
		Carry: mediaserver.CarryEitherOrder, UserDataIgnoresPlays: true, PurgesVacated: true, ScanTime: time.Second,
	}
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
	w.s.Report([]mediaserver.Update{{Path: path(w.t, "/media/"+rel)}})
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

// twoRoots is a Jellyfin library scanned while its movies folder held
// Metropolis and its fresh folder was empty, as in recorded run
// 2026-11-14.
func twoRoots(t *testing.T) world {
	t.Helper()
	w := world{t: t, media: t.TempDir()}
	w.place(metropolis)
	if err := os.MkdirAll(filepath.Join(w.media, "fresh"), 0o700); err != nil {
		t.Fatal(err)
	}
	w.s = mediaserver.New(jellyfin, mediaserver.Disk{Media: w.media, Root: path(t, "/media")},
		[]mediaserver.Library{{Key: "1", Name: "Films", Type: "movies", Folders: []link.ServerPath{path(t, "/media/movies"), path(t, "/media/fresh")}, Scanned: true}})
	w.s.Libraries()
	return w
}

func (w world) settle() {
	w.t.Helper()
	time.Sleep(w.s.MonitorDelay() + 2*time.Second)
}

func TestAFreshJellyfinRootBesideAScannedOneNeedsAScanWithContent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := twoRoots(t)
		bravo, charlie := "fresh/Bravo (2002)/Bravo (2002).mkv", "fresh/Charlie (2003)/Charlie (2003).mkv"
		w.place(bravo)
		w.report(bravo)
		w.settle()
		if w.has(bravo) {
			t.Fatal("a report into a root scanned only while empty made an item")
		}
		w.remove(bravo)
		w.scan()
		time.Sleep(2 * time.Second)
		w.place(bravo)
		w.report(bravo)
		w.settle()
		if w.has(bravo) {
			t.Fatal("a full scan of the empty root made the root take reports")
		}
		w.scan()
		time.Sleep(2 * time.Second)
		if !w.has(bravo) {
			t.Fatal("the full scan with Bravo on disk didn't pick it up")
		}
		w.place(charlie)
		w.report(charlie)
		w.settle()
		if !w.has(charlie) {
			t.Fatal("a report after the scan with content was ignored")
		}
		w.remove(bravo)
		w.remove(charlie)
		w.report(bravo)
		w.report(charlie)
		w.settle()
		if w.has(bravo) || w.has(charlie) {
			t.Fatal("the deleted files' items outlived their reports")
		}
		delta := "fresh/Delta (2004)/Delta (2004).mkv"
		w.place(delta)
		w.report(delta)
		w.settle()
		if !w.has(delta) {
			t.Fatal("the root emptied again stopped taking reports")
		}
	})
}

func TestJellyfinHonorsTheConfiguredMonitorDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, jellyfin, true, metropolis)
		w.s.SetMonitorDelay(5 * time.Second)
		w.place(heat)
		w.report(heat)
		time.Sleep(4 * time.Second)
		if w.has(heat) {
			t.Fatal("Heat showed up before the 5 s delay")
		}
		time.Sleep(2 * time.Second)
		if !w.has(heat) {
			t.Fatal("Heat didn't show up after the 5 s delay")
		}
	})
}

func TestAFolderAddedToAJellyfinLibraryDropsPendingReports(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, jellyfin, true, metropolis)
		w.place(heat)
		w.report(heat)
		if err := w.s.AddFolder("1", path(t, "/media/more")); err != nil {
			t.Fatal(err)
		}
		w.settle()
		if w.has(heat) {
			t.Fatal("a report sent before the folder was added still scanned")
		}
	})
}

func TestTheRefreshLibraryTaskEndsWithItsLastLibrary(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, jellyfin, true, metropolis)
		asked := w.s.RefreshAll()
		if st := w.s.TaskState(); !st.Running || !st.Started.Equal(asked) {
			t.Fatalf("task %+v right after it was asked for", st)
		}
		if again := w.s.RefreshAll(); !again.Equal(asked) {
			t.Fatal("a second request started another task")
		}
		time.Sleep(2 * time.Second)
		if st := w.s.TaskState(); st.Running || !st.Ended.After(asked) {
			t.Fatalf("task %+v after the scan's time passed", st)
		}
	})
}

func (w world) watchedBy(rel string) mediaserver.Item {
	w.t.Helper()
	it, ok := w.item(rel)
	if !ok {
		w.t.Fatalf("no item at %s", rel)
	}
	w.s.Played("owner", it.Key, time.Now())
	w.s.AddToCollection("Silent", it.Key)
	return it
}

func TestJellyfinMakesANewItemForARenameInItsFolder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, jellyfin, true, metropolis)
		old := w.watchedBy(metropolis)
		renamed := "movies/Metropolis (1927)/Metropolis (1927) Bluray-1080p.mkv"
		w.move(renamed)
		w.report(renamed)
		w.report(metropolis)
		w.settle()
		now, ok := w.item(renamed)
		if !ok || now.Key == old.Key {
			t.Fatalf("the renamed file is in %+v, want a new item", now)
		}
		if w.s.UserData("owner", now.Key).PlayCount != 0 || len(w.s.Collections(now.Key)) != 0 {
			t.Fatal("a file with no provider id carried watch data to its new item")
		}
	})
}

func TestTheSamePathGetsTheSameJellyfinID(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, jellyfin, true, metropolis)
		old, _ := w.item(metropolis)
		w.remove(metropolis)
		w.report(metropolis)
		w.settle()
		w.place(metropolis)
		w.report(metropolis)
		w.settle()
		if now, ok := w.item(metropolis); !ok || now.Key != old.Key {
			t.Fatalf("the file back at its path is %+v, want id %s", now, old.Key)
		}
	})
}

func TestJellyfinKeepsVersionsInOneItemAndEmbyAnItemPerFile(t *testing.T) {
	versions := []string{"movies/Heat (1995)/Heat (1995) - 1080p.mkv", "movies/Heat (1995)/Heat (1995) - 2160p.mkv"}
	for _, c := range []struct {
		name  string
		b     mediaserver.Behavior
		items bool
	}{{"jellyfin", jellyfin, false}, {"emby", emby, true}} {
		synctest.Test(t, func(t *testing.T) {
			w := newWorld(t, c.b, true, versions...)
			a, _ := w.item(versions[0])
			b, _ := w.item(versions[1])
			if (a.Key != b.Key) != c.items {
				t.Fatalf("%s: the versions are items %s and %s", c.name, a.Key, b.Key)
			}
		})
	}
}

const (
	tagged      = "movies/Metropolis (1927) [tmdbid-19]/Metropolis (1927) [tmdbid-19].mkv"
	taggedMoved = "movies/Metropolis/Metropolis (1927) [tmdbid-19].mkv"
)

func moveTagged(w world, removedFirst bool) mediaserver.Item {
	w.t.Helper()
	old := w.watchedBy(tagged)
	dst := filepath.Join(w.media, filepath.FromSlash(taggedMoved))
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		w.t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(w.media, filepath.FromSlash(tagged)), dst); err != nil {
		w.t.Fatal(err)
	}
	order := []string{taggedMoved, tagged}
	if removedFirst {
		order = []string{tagged, taggedMoved}
	}
	for _, rel := range order {
		w.s.Report([]mediaserver.Update{{Path: path(w.t, "/media/"+rel), Deleted: rel == tagged}})
		w.settle()
	}
	now, ok := w.item(taggedMoved)
	if !ok || now.Key == old.Key {
		w.t.Fatalf("the moved file is in %+v, want a new item", now)
	}
	if len(w.s.Collections(now.Key)) != 0 {
		w.t.Fatal("the collection carried to the new item")
	}
	return now
}

func TestJellyfinCarriesWatchDataByProviderIDOnlyWhenRemovedFirst(t *testing.T) {
	for _, removedFirst := range []bool{true, false} {
		synctest.Test(t, func(t *testing.T) {
			w := newWorld(t, jellyfin, true, tagged)
			now := moveTagged(w, removedFirst)
			if got := w.s.UserData("owner", now.Key).PlayCount; got != map[bool]int{true: 1, false: 0}[removedFirst] {
				t.Fatalf("removed first %v: the new item's play count is %d", removedFirst, got)
			}
		})
	}
}

func TestEmbyCarriesWatchDataByProviderIDInEitherOrder(t *testing.T) {
	for _, removedFirst := range []bool{true, false} {
		synctest.Test(t, func(t *testing.T) {
			w := newWorld(t, emby, true, tagged)
			now := moveTagged(w, removedFirst)
			if got := w.s.UserData("owner", now.Key).PlayCount; got != 1 {
				t.Fatalf("removed first %v: the new item's play count is %d", removedFirst, got)
			}
		})
	}
}

func TestEmbysUserDataWriteKeepsThePlayCountAndLastPlayed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, emby, true, metropolis)
		it, _ := w.item(metropolis)
		w.s.SetUserData("owner", it.Key, link.UserData{Played: true, PlayCount: 3, LastPlayed: time.Now(), Favorite: true, Position: time.Minute})
		d := w.s.UserData("owner", it.Key)
		if !d.Played || !d.Favorite || d.Position != time.Minute || d.PlayCount != 0 || !d.LastPlayed.IsZero() {
			t.Fatalf("after the write the data is %+v", d)
		}
	})
}

func TestEmbyRemovesADeletedFileAtOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, emby, true, metropolis)
		w.remove(metropolis)
		w.s.Report([]mediaserver.Update{{Path: path(t, "/media/"+metropolis), Deleted: true}})
		if w.has(metropolis) {
			t.Fatal("Emby waited its delay to remove a Deleted file")
		}
	})
}

func TestACollectionHoldsAnItemOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, jellyfin, true, metropolis)
		it, _ := w.item(metropolis)
		w.s.AddToCollection("Silent", it.Key)
		w.s.AddToCollection("Silent", it.Key)
		if got := w.s.Members("Silent"); len(got) != 1 || got[0] != it.Key {
			t.Fatalf("the collection holds %v", got)
		}
	})
}
