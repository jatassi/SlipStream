package mediaserversapi_test

import (
	"net/http"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaservers/mstest"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

const browser = "https://den.example.invalid:8920"

// newBrowsed is Den as kind, which seedbox reaches, with a browser
// address and the Shelves root shown in its library, which Den never
// scanned with content.
func newBrowsed(t *testing.T, kind, version string) shelf {
	t.Helper()
	r := newRig(t)
	s := shelf{rig: r}
	if kind == "emby" {
		s.den = r.w.Emby(t, host, den(version))
	} else {
		s.den = r.w.Jellyfin(t, host, den(version))
	}
	draft := `{"kind":"` + kind + `","name":"Den","address":"` + address + `","browserAddress":"` + browser + `/","reachingNodeId":"` +
		r.w.Seedbox.String() + `","apiKey":"` + denKey + `"}`
	status, body := r.o.Call(t, http.MethodPost, servers, draft)
	if status != http.StatusCreated {
		t.Fatalf("add Den = %d %s", status, body)
	}
	s.id = decode[serverOut](t, body).ID
	s.root = r.w.Root(t, "Shelves").ID
	s.mapTo(t, shelves)
	settle(15 * time.Second)
	return s
}

// holds gives a new Work the files at rels of root.
func (s shelf) holds(t *testing.T, ids fileIDs, root roots.ID, rels ...string) works.WorkID {
	t.Helper()
	work := works.WorkID{UUID: storage.NewUUID()}
	fs := make([]files.File, len(rels))
	for i, rel := range rels {
		fs[i] = files.File{ID: ids.of(rel), WorkID: work, RootID: root, Path: rel}
	}
	s.w.Holds(work, fs...)
	return work
}

func (s shelf) openIn(t *testing.T, work works.WorkID) []mediaservers.OpenIn {
	t.Helper()
	got, err := s.w.Service.OpenIn(t.Context(), work)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// expectFound fails unless media-servers.items-found named work.
func (s shelf) expectFound(t *testing.T, work works.WorkID) {
	t.Helper()
	for _, f := range s.w.Found() {
		if slices.Contains(f.WorkIDs, work.String()) {
			return
		}
	}
	t.Errorf("media-servers.items-found published %v, want it to name %s", s.w.Found(), work)
}

// expectLink fails unless links is one link to Den's item at path,
// through want.
func (s shelf) expectLink(t *testing.T, links []mediaservers.OpenIn, kind link.KindName, want func(mstest.Item) string, path string) {
	t.Helper()
	p, err := link.ParsePath(path)
	if err != nil {
		t.Fatal(err)
	}
	it, ok := s.w.Item(t, s.id, p)
	if !ok {
		t.Fatalf("the index holds no item at %s", path)
	}
	if len(links) != 1 || links[0].Server.String() != s.id || links[0].Kind != kind || links[0].URL != want(it) {
		t.Errorf("Open in = %+v, want one %s link to %s", links, kind, want(it))
	}
}

// lookedUp lands rels on the server, gives a Work their files, and opens
// its page: the index holds none of them yet, so the page shows no link
// and the server looks them up, then items-found names the Work and its
// page shows the link.
func (s shelf) lookedUp(t *testing.T, media, prefix string, rels ...string) []mediaservers.OpenIn {
	t.Helper()
	ids := fileIDs{}
	for _, rel := range rels {
		place(t, media, prefix+rel)
		s.tell(t, at(t, files.PathAdded, ids.of(rel), rel))
	}
	settle(3 * time.Minute)
	work := s.holds(t, ids, s.root, rels...)
	if got := s.openIn(t, work); len(got) != 0 {
		t.Fatalf("Open in = %+v before a lookup found the item", got)
	}
	settle(time.Minute)
	s.expectFound(t, work)
	return s.openIn(t, work)
}

// TestOpenInJellyfinLooksUpTheItem opens a Work whose file Den holds but
// the path index doesn't yet: a lookup finds it, and the page links to
// the item in Den's web app at its browser address.
func TestOpenInJellyfinLooksUpTheItem(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newBrowsed(t, "jellyfin", floorVer)
		links := s.lookedUp(t, s.den.Media, "Shelves/", "Trip/Trip.mkv")
		s.expectLink(t, links, "jellyfin", func(it mstest.Item) string {
			return browser + "/web/#/details?id=" + it.Key + "&serverId=" + denID
		}, p+"Trip/Trip.mkv")
	})
}

// TestOpenInEmbyLooksUpTheItem is the Jellyfin case on Emby, which looks
// up by path and has its own item page.
func TestOpenInEmbyLooksUpTheItem(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newBrowsed(t, "emby", "4.10.1.0")
		links := s.lookedUp(t, s.den.Media, "Shelves/", "Trip/Trip.mkv")
		s.expectLink(t, links, "emby", func(it mstest.Item) string {
			return browser + "/web/index.html#!/item?id=" + it.Key + "&serverId=" + denID
		}, p+"Trip/Trip.mkv")
	})
}

// TestOpenInPlexLinksTheTopItem opens a Work of two episodes on Living
// room Plex: the link opens their show in Plex's web app, which the
// plex.tv Endpoints entry names.
func TestOpenInPlexLinksTheTopItem(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s, px := newFresh(t)
		links := s.lookedUp(t, px.Server.Media, "Shows/", "Voyage/Season 01/Voyage - S01E01.mkv", "Voyage/Season 01/Voyage - S01E02.mkv")
		s.expectLink(t, links, "plex", func(it mstest.Item) string {
			if it.Top == it.Key {
				t.Errorf("the episode's top item is the episode itself, %s", it.Key)
			}
			return "https://app.plex.invalid/desktop/#!/server/" + plexID + "/details?key=/library/metadata/" + it.Top
		}, `C:\Media\Shows\Voyage\Season 01\Voyage - S01E01.mkv`)
	})
}

// TestOpenInNeedsABrowserAddress opens a Work on Den with no browser
// address: no link, and no lookup.
func TestOpenInNeedsABrowserAddress(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newShelf(t, "")
		place(t, s.den.Media, "Shelves/Trip/Trip.mkv")
		ids := fileIDs{}
		s.tell(t, at(t, files.PathAdded, ids.of("Trip/Trip.mkv"), "Trip/Trip.mkv"))
		settle(3 * time.Minute)
		work := s.holds(t, ids, s.root, "Trip/Trip.mkv")
		if got := s.openIn(t, work); len(got) != 0 {
			t.Errorf("Open in = %+v with no browser address", got)
		}
		settle(time.Minute)
		if got := s.w.Found(); len(got) != 0 {
			t.Errorf("media-servers.items-found published %v with no browser address", got)
		}
	})
}
