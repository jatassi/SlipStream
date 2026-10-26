package downloading_test

import (
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec // the v1 info hash
	"os"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/magnet"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

type resolved struct {
	res *dlv1.ResolveResponse
	err error
}

func resolve(c *client, link magnet.Link, endsAt time.Time) <-chan resolved {
	out := make(chan resolved, 1)
	go func() {
		res, err := c.Resolve(context.Background(), &dlv1.ResolveRequest{
			InfoHash: link.InfoHash[:], Trackers: link.Trackers, Name: link.Name, EndsAtUnixMs: endsAt.UnixMilli(),
		})
		out <- resolved{res, err}
	}()
	return out
}

func await(t *testing.T, ch <-chan resolved) resolved {
	t.Helper()
	synctest.Wait()
	select {
	case r := <-ch:
		return r
	default:
		t.Fatal("Resolve hasn't answered")
		return resolved{}
	}
}

func pending(t *testing.T, ch <-chan resolved, why string) {
	t.Helper()
	synctest.Wait()
	select {
	case r := <-ch:
		t.Fatalf("Resolve answered %v, %v; want it to wait %s", r.res, r.err, why)
	default:
	}
}

func linkOf(f file, trackers ...string) magnet.Link {
	return magnet.Link{InfoHash: f.hash, Name: f.name, Trackers: trackers}
}

func (n *node) serve(f file) {
	n.t.Helper()
	m, err := bencode.DecodeMetainfo(f.meta)
	if err != nil {
		n.t.Fatal(err)
	}
	n.world.Serve(m)
}

func infoOf(t *testing.T, r resolved) []byte {
	t.Helper()
	if r.err != nil || r.res.GetInfo() == nil {
		t.Fatalf("Resolve answered %v, %v; want the info dictionary", r.res, r.err)
	}
	return r.res.GetInfo()
}

func searchingOf(t *testing.T, r resolved) *dlv1.Searching {
	t.Helper()
	if r.err != nil || r.res.GetSearching() == nil {
		t.Fatalf("Resolve answered %v, %v; want how the search stands", r.res, r.err)
	}
	return r.res.GetSearching()
}

func unresolvedOf(t *testing.T, r resolved) *dlv1.Unresolved {
	t.Helper()
	if r.err != nil || r.res.GetUnresolved() == nil {
		t.Fatalf("Resolve answered %v, %v; want the search's end", r.res, r.err)
	}
	return r.res.GetUnresolved()
}

const wait = 30 * time.Minute

func TestTwoCallsJoinOneSearchAndGetTheInfoDictionaryOnceItsTorrentIsGone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		f := newFile(t, "Orchard.mkv")
		c := n.start()
		e := c.engine
		c.send(live())

		link := linkOf(f, trackerURL)
		first := resolve(c, link, time.Now().Add(wait))
		second := resolve(c, link, time.Now().Add(wait))
		pending(t, first, "while no peer sent the metadata")
		time.Sleep(timings.ResolveHold)
		if s := searchingOf(t, await(t, first)); s.GetSinceUnixMs() == 0 {
			t.Errorf("the search stands as %v, want its start", s)
		}
		searchingOf(t, await(t, second))

		third := resolve(c, link, time.Now().Add(wait))
		n.serve(f)
		info := infoOf(t, await(t, third))
		if sha1.Sum(info) != f.hash { //nolint:gosec // the v1 info hash
			t.Fatalf("the info dictionary hashes to %x, want %s", sha1.Sum(info), f.hash) //nolint:gosec // the v1 info hash
		}
		if e.searches() != 1 {
			t.Errorf("the engine was asked for %d metadata searches, want one for all three calls", e.searches())
		}
		if again := infoOf(t, await(t, resolve(c, link, time.Now().Add(wait)))); !bytes.Equal(again, info) || e.searches() != 1 {
			t.Errorf("a call whose answer was lost got %d bytes after %d searches, want the kept info dictionary and no new search", len(again), e.searches())
		}

		n.world.Payload(f.hash, f.payload)
		id := storage.NewUUID()
		n.ctl.answer(id, &dlv1.TorrentResponse{Metainfo: f.meta, ContentType: "Movies"})
		c.send(live(f.started(id, 1)))
		path, err := c.Path(f.hash, 0)
		if err != nil {
			t.Fatalf("the download added after the search = %v", err)
		}
		if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, f.payload) {
			t.Fatalf("the download holds %d bytes, %v; want its %d", len(got), err, len(f.payload))
		}
		if r := await(t, resolve(c, link, time.Now().Add(wait))); r.res.GetHeld() == nil {
			t.Errorf("Resolve of a held torrent's hash = %v, %v; want held", r.res, r.err)
		}
	})
}

func TestASearchEndsAtItsDeadlineWithTheTrackersLastWord(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const failing = "https://failing.example/announce"
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		n.world.Tracker(failing, enginetest.TrackerRules{Failure: "torrent not registered with this tracker"})
		f := newPrivateFile(t, "Orchard.mkv")
		n.serve(f)
		c := n.start()
		c.send(live())

		endsAt := time.Now().Add(timings.ResolveHold / 2)
		call := resolve(c, linkOf(f, failing, trackerURL), endsAt)
		time.Sleep(timings.ResolveHold / 2)
		r := await(t, call)
		u := unresolvedOf(t, r)
		if u.GetReason() != dlv1.UnresolvedReason_UNRESOLVED_REASON_TIMED_OUT || u.GetTrackerMessage() != "torrent not registered with this tracker" {
			t.Errorf("a private magnet link's search ended with %v, want timed out with the tracker's failure reason", u)
		}
		if late := time.Since(endsAt); late < 0 || late > 2*time.Second {
			t.Errorf("the search ended %v after its deadline, want at it", late)
		}
	})
}

func resolveAfterHold(c *client, link magnet.Link, endsAt time.Time) <-chan resolved {
	ch := resolve(c, link, endsAt)
	synctest.Wait()
	if len(ch) == 0 {
		time.Sleep(timings.ResolveHold)
	}
	return ch
}

func TestASearchNobodyAsksAboutEnds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		f := newFile(t, "Orchard.mkv")
		c := n.start()
		c.send(live())

		since := time.Now()
		searchingOf(t, await(t, resolveAfterHold(c, linkOf(f, trackerURL), time.Now().Add(wait))))
		time.Sleep(timings.ResolveLease)
		synctest.Wait()
		if stopped := announcesOf(n.world, f, "stopped", since); stopped != 1 {
			t.Errorf("the tracker heard %d stopped for a search nobody asked about, want one", stopped)
		}
	})
}

func TestATrackerlessMagnetLinkSearchesTheDHTOnlyWhileItsSwitchIsOn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		f := newFile(t, "Orchard.mkv")
		n.serve(f)
		c := n.start()

		c.send(withDHT(false))
		if u := unresolvedOf(t, await(t, resolve(c, linkOf(f), time.Now().Add(wait)))); u.GetReason() != dlv1.UnresolvedReason_UNRESOLVED_REASON_NO_PEER_SOURCE {
			t.Errorf("a trackerless magnet link with the DHT off ended with %v, want no peer source at once", u)
		}

		c.send(withDHT(true))
		info := infoOf(t, await(t, resolve(c, linkOf(f), time.Now().Add(wait))))
		if sha1.Sum(info) != f.hash || n.world.Lookups(f.hash) == 0 { //nolint:gosec // the v1 info hash
			t.Errorf("with the DHT on the search got %d bytes after %d lookups, want the info dictionary from the DHT", len(info), n.world.Lookups(f.hash))
		}
		if c.engine.running().DHT {
			t.Error("the DHT still runs once the only search that may use it ended")
		}
	})
}

func TestAMagnetLinkThatNamesATrackerNeverUsesTheDHT(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		public, f := newFile(t, "Meadow.mkv"), newFile(t, "Orchard.mkv")
		n.world.Payload(public.hash, public.payload)
		c := n.start()
		c.send(withDHT(true, public.started(n.hold(public), 1)))
		searchingOf(t, await(t, resolveAfterHold(c, linkOf(f, trackerURL), time.Now().Add(wait))))
		if !c.engine.running().DHT || n.world.Lookups(f.hash) != 0 {
			t.Errorf("the DHT runs = %v and looked up a magnet link naming a tracker %d times, want it running for the public torrent and no lookup", c.engine.running().DHT, n.world.Lookups(f.hash))
		}
	})
}

func TestTheIntentListingASearchedHashEndsTheSearchBeforeTheDownloadIsAdded(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		f := newFile(t, "Orchard.mkv")
		n.world.Payload(f.hash, f.payload)
		id := n.hold(f)
		c := n.start()
		c.send(live())

		waiting := resolve(c, linkOf(f, trackerURL), time.Now().Add(wait))
		pending(t, waiting, "while no peer sent the metadata")
		c.send(live(f.started(id, 1)))
		if r := await(t, waiting); r.res.GetHeld() == nil {
			t.Errorf("a call waiting on a search the intent took over got %v, %v; want held", r.res, r.err)
		}
		path, err := c.Path(f.hash, 0)
		if err != nil {
			t.Fatalf("the download added once the search ended = %v", err)
		}
		if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, f.payload) {
			t.Fatalf("the download holds %d bytes, %v; want its %d", len(got), err, len(f.payload))
		}
	})
}

func TestAClientThatIsntLiveTakesNoSearch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		f := newFile(t, "Orchard.mkv")
		c := n.start()
		s := live()
		s.Draining = true
		c.send(s)
		if r := await(t, resolve(c, linkOf(f, trackerURL), time.Now().Add(wait))); r.res.GetNotTaking() == nil {
			t.Errorf("Resolve on a draining client = %v, %v; want not taking", r.res, r.err)
		}
	})
}

func TestACallAfterASearchTimedOutGetsASearchOfItsOwn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute, HoldStarted: true})
		f := newFile(t, "Orchard.mkv")
		c := n.start()
		c.send(live())
		link := linkOf(f, trackerURL)

		early := resolve(c, link, time.Now().Add(timings.ResolveHold/2))
		time.Sleep(timings.ResolveHold/2 + 2*time.Second)
		late := resolve(c, link, time.Now().Add(wait))
		pending(t, late, "while the search that timed out is removed")
		n.world.Answer(trackerURL)
		if u := unresolvedOf(t, await(t, early)); u.GetReason() != dlv1.UnresolvedReason_UNRESOLVED_REASON_TIMED_OUT {
			t.Errorf("the call whose deadline passed got %v, want timed out", u)
		}
		pending(t, late, "for a search of its own, not the end of one that timed out at another call's deadline")
		n.serve(f)
		if info := infoOf(t, await(t, late)); sha1.Sum(info) != f.hash { //nolint:gosec // the v1 info hash
			t.Errorf("the late call got an info dictionary hashing to %x, want %s", sha1.Sum(info), f.hash) //nolint:gosec // the v1 info hash
		}
		if s := c.engine.searches(); s != 2 {
			t.Errorf("the engine was asked for %d metadata searches, want one for each deadline", s)
		}
	})
}

func TestACallWhoseLinkNamesATrackerNeverRidesATrackerlessSearch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		f := newFile(t, "Orchard.mkv")
		c := n.start()
		c.send(withDHT(true))

		since := time.Now()
		trackerless := resolve(c, linkOf(f), time.Now().Add(timings.ResolveHold/2))
		pending(t, trackerless, "while no peer sent the metadata")
		tracked := resolve(c, linkOf(f, trackerURL), time.Now().Add(wait))
		pending(t, tracked, "while the trackerless search runs")
		if started := announcesOf(n.world, f, "started", since); started != 0 {
			t.Errorf("the tracker heard started %d times while the trackerless search ran, want none", started)
		}
		time.Sleep(timings.ResolveHold/2 + 2*time.Second)
		if u := unresolvedOf(t, await(t, trackerless)); u.GetReason() != dlv1.UnresolvedReason_UNRESOLVED_REASON_TIMED_OUT {
			t.Errorf("the trackerless call got %v at its deadline, want timed out", u)
		}
		if started := announcesOf(n.world, f, "started", since); started != 1 {
			t.Errorf("the tracker the link names heard started %d times once the trackerless search ended, want once", started)
		}
		lookups := n.world.Lookups(f.hash)
		n.serve(f)
		infoOf(t, await(t, tracked))
		if got := n.world.Lookups(f.hash); got != lookups {
			t.Errorf("the DHT looked up the hash %d times for a link that names a tracker, want none", got-lookups)
		}
	})
}
