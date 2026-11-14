package mediaserversapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/jellyfin"
	fakeplex "github.com/jatassi/slipstream/fakes/plex"
	fakeplextv "github.com/jatassi/slipstream/fakes/plextv"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaservers/mstest"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

// shelf is Den, which seedbox reaches, with the Shelves root shown in its
// library and already scanned, once the library refresh its mapping
// queued has gone.
type shelf struct {
	rig
	den  faketest.Served
	id   string
	root roots.ID
}

// newShelf adds Den with fallback as its fallback address, "" for none
// set.
func newShelf(t *testing.T, fallback string) shelf {
	t.Helper()
	return newShelfOf(t, den(floorVer), fallback)
}

// newShelfOf is newShelf with srv as Den, and each of placed, a path Den
// names, there before Den scans.
func newShelfOf(t *testing.T, srv jellyfin.Server, fallback string, placed ...string) shelf {
	t.Helper()
	s := addShelfOf(t, srv, fallback, placed...)
	s.w.Scanned(t)
	time.Sleep(15 * time.Second)
	synctest.Wait()
	if scans := records(t, s.den, "jellyfin.scan"); len(scans) != 1 {
		t.Fatalf("saving the mapping refreshed Den's libraries %d times, want once", len(scans))
	}
	return s
}

// addShelf adds Den with the Shelves root shown in its library, which Den
// never scanned with content.
func addShelf(t *testing.T, fallback string) shelf {
	t.Helper()
	return addShelfOf(t, den(floorVer), fallback)
}

func addShelfOf(t *testing.T, srv jellyfin.Server, fallback string, placed ...string) shelf {
	t.Helper()
	r := newRig(t)
	s := shelf{rig: r, den: r.w.Jellyfin(t, host, srv, placed...)}
	draft := `{"kind":"jellyfin","name":"Den","address":"` + address + `","reachingNodeId":"` + r.w.Seedbox.String() + `","apiKey":"` + denKey + `"`
	if fallback != "" {
		draft += `,"fallbackAddress":"` + fallback + `"`
	}
	draft += "}"
	status, body := r.o.Call(t, http.MethodPost, servers, draft)
	if status != http.StatusCreated {
		t.Fatalf("add Den = %d %s", status, body)
	}
	s.id = decode[serverOut](t, body).ID
	s.root = r.w.Root(t, "Shelves").ID
	s.mapTo(t, shelves)
	return s
}

func (s shelf) mapTo(t *testing.T, folder string) {
	t.Helper()
	body := `{"mappings":[{"rootId":"` + s.root.String() + `","shown":true,"serverFolder":"` + strings.ReplaceAll(folder, `\`, `\\`) + `"}]}`
	if status, reply := s.o.Call(t, http.MethodPut, servers+"/"+s.id+"/mappings", body); status != http.StatusOK {
		t.Fatalf("save the mapping = %d %s", status, reply)
	}
}

func (s shelf) changes(t *testing.T) []mediaservers.ChangeView {
	t.Helper()
	status, body := s.o.Call(t, http.MethodGet, servers+"/"+s.id+"/changes", "")
	if status != http.StatusOK {
		t.Fatalf("list Den's changes = %d %s", status, body)
	}
	return decode[mediaservers.Changes](t, body).Changes
}

// change is the newest change at path, failing the test when none is.
func change(t *testing.T, cs []mediaservers.ChangeView, path string) mediaservers.ChangeView {
	t.Helper()
	i := slices.IndexFunc(cs, func(c mediaservers.ChangeView) bool { return c.Path == path })
	if i < 0 {
		t.Fatalf("no change at %s among %v", path, cs)
	}
	return cs[i]
}

func records(t *testing.T, f faketest.Served, kind string) []json.RawMessage {
	t.Helper()
	var out []json.RawMessage
	for _, r := range f.Records(t) {
		if r.Kind == kind {
			out = append(out, r.Data)
		}
	}
	return out
}

// reported is every path report Den took, one string per POST, such as
// "Created /media/Shelves/A/A.mkv, Deleted /media/Shelves/B/B.mkv".
func reported(t *testing.T, f faketest.Served) []string {
	t.Helper()
	var out []string
	for _, raw := range records(t, f, "jellyfin.report") {
		var r struct {
			Updates []struct{ Path, UpdateType string }
		}
		if err := json.Unmarshal(raw, &r); err != nil {
			t.Fatal(err)
		}
		var ps []string
		for _, u := range r.Updates {
			ps = append(ps, u.UpdateType+" "+u.Path)
		}
		out = append(out, strings.Join(ps, ", "))
	}
	return out
}

func expectReported(t *testing.T, f faketest.Served, want ...string) {
	t.Helper()
	if got := reported(t, f); !slices.Equal(got, want) {
		t.Errorf("Den took the reports\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// files names the files of one test by name.
type fileIDs map[string]files.ID

func (ids fileIDs) of(name string) files.ID {
	if _, ok := ids[name]; !ok {
		ids[name] = files.ID{UUID: storage.NewUUID()}
	}
	return ids[name]
}

func mustRel(t *testing.T, s string) storing.Rel {
	t.Helper()
	r, err := storing.ParseRel(s)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func (s shelf) tell(t *testing.T, cs ...files.PathChange) {
	t.Helper()
	for i := range cs {
		cs[i].Root = s.root
	}
	s.w.Tell(t, cs...)
}

func at(t *testing.T, kind files.PathChangeKind, file files.ID, path string) files.PathChange {
	t.Helper()
	c := files.PathChange{Kind: kind, File: file}
	from, to, moved := strings.Cut(path, ">")
	if !moved {
		to = from
	} else {
		c.From = mustRel(t, from)
	}
	c.Path = mustRel(t, to)
	return c
}

func settle(d time.Duration) {
	time.Sleep(d)
	synctest.Wait()
}

func (s shelf) wait(t *testing.T, seconds int) {
	t.Helper()
	panel := s.w.Service.Settings()
	if err := panel.Apply(t.Context(), setting.Patch{mediaservers.WaitBeforeSending.Key: json.RawMessage(strconv.Itoa(seconds))}); err != nil {
		t.Fatal(err)
	}
}

const p = "/media/Shelves/"

// TestAChangeGoesThroughTheReachingNode imports one file into Shelves:
// once Wait before sending passed, seedbox reports its path to Den, and
// the Admin reads it sent.
func TestAChangeGoesThroughTheReachingNode(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newShelf(t, "")
		ids := fileIDs{}
		s.tell(t, at(t, files.PathAdded, ids.of("clip"), "Clip/Clip.mkv"))
		settle(5 * time.Second)
		expectReported(t, s.den)
		if c := change(t, s.changes(t), "Clip/Clip.mkv"); c.State != "due" || c.Words != "This change is waiting to be sent." {
			t.Errorf("before Wait before sending passed the change is %s: %s", c.State, c.Words)
		}
		settle(10 * time.Second)
		expectReported(t, s.den, "Created "+p+"Clip/Clip.mkv")
		settle(2 * time.Minute)
		c := change(t, s.changes(t), "Clip/Clip.mkv")
		if c.State != "sent" || c.Result != "accepted" || c.Words != "Sent the new file." {
			t.Errorf("once Den settled the change is %s %s: %s", c.State, c.Result, c.Words)
		}
	})
}

// queued is each of Den's changes but the library refreshes, oldest
// first, as "kind from>path".
func (s shelf) queued(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, c := range slices.Backward(s.changes(t)) {
		if c.Kind == "library" {
			continue
		}
		path := c.Path
		if c.FromPath != "" {
			path = c.FromPath + ">" + path
		}
		out = append(out, c.Kind+" "+path)
	}
	return out
}

// TestTheMergeRulesHoldEndToEnd tells Den's queue each sequence of the
// spec's merge table inside one Wait before sending: the Admin reads the
// merged change, and Den takes it and nothing else. A carry from a path
// where Den holds no item goes as a new file.
func TestTheMergeRulesHoldEndToEnd(t *testing.T) {
	for _, c := range []struct {
		name   string
		told   func(t *testing.T, ids fileIDs) []files.PathChange
		queued []string
		want   []string
	}{
		{"several changes in one folder are one report", func(t *testing.T, ids fileIDs) []files.PathChange {
			return []files.PathChange{
				at(t, files.PathAdded, ids.of("1"), "Trip/1.mkv"), at(t, files.PathAdded, ids.of("2"), "Trip/2.mkv"),
				at(t, files.PathRemoved, ids.of("old"), "Trip/old.mkv"),
			}
		}, []string{"added Trip/1.mkv", "added Trip/2.mkv", "removed Trip/old.mkv"},
			[]string{"Created " + p + "Trip/1.mkv, Created " + p + "Trip/2.mkv, Deleted " + p + "Trip/old.mkv"}},
		{"removed then added on one path is replaced", func(t *testing.T, ids fileIDs) []files.PathChange {
			return []files.PathChange{at(t, files.PathRemoved, ids.of("old"), "Trip/Trip.mkv"), at(t, files.PathAdded, ids.of("new"), "Trip/Trip.mkv")}
		}, []string{"replaced Trip/Trip.mkv"}, []string{"Modified " + p + "Trip/Trip.mkv"}},
		{"added then removed on one path is nothing", func(t *testing.T, ids fileIDs) []files.PathChange {
			return []files.PathChange{at(t, files.PathAdded, ids.of("f"), "Trip/Trip.mkv"), at(t, files.PathRemoved, ids.of("f"), "Trip/Trip.mkv")}
		}, nil, nil},
		{"moved A to B then B to C is moved A to C", func(t *testing.T, ids fileIDs) []files.PathChange {
			return []files.PathChange{at(t, files.PathMoved, ids.of("f"), "A/A.mkv>B/B.mkv"), at(t, files.PathMoved, ids.of("f"), "B/B.mkv>C/C.mkv")}
		}, []string{"moved A/A.mkv>C/C.mkv"}, []string{"Created " + p + "C/C.mkv"}},
		{"moved A to B then removed B is removed A", func(t *testing.T, ids fileIDs) []files.PathChange {
			return []files.PathChange{at(t, files.PathMoved, ids.of("f"), "A/A.mkv>B/B.mkv"), at(t, files.PathRemoved, ids.of("f"), "B/B.mkv")}
		}, []string{"removed A/A.mkv"}, []string{"Deleted " + p + "A/A.mkv"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				s := newShelf(t, "")
				for _, ch := range c.told(t, fileIDs{}) {
					s.tell(t, ch)
					settle(time.Second)
				}
				if got := s.queued(t); !slices.Equal(got, c.queued) {
					t.Errorf("Den's queue holds %v, want %v", got, c.queued)
				}
				settle(10 * time.Minute)
				expectReported(t, s.den, c.want...)
			})
		})
	}
}

// TestAnUpgradeWaitsForItsOtherHalf sets Wait before sending to 0 s and
// lands an upgrade in each order: the half that comes first waits for
// the other, so Den takes one change and never a folder with no version
// in it.
func TestAnUpgradeWaitsForItsOtherHalf(t *testing.T) {
	for _, c := range []struct {
		name          string
		first, second func(t *testing.T, ids fileIDs) files.PathChange
		queued        []string
		want          []string
	}{
		{"removed first, same name", func(t *testing.T, ids fileIDs) files.PathChange {
			c := at(t, files.PathRemoved, ids.of("old"), "Trip/Trip.mkv")
			c.ReplacedBy = ids.of("new")
			return c
		}, func(t *testing.T, ids fileIDs) files.PathChange {
			return at(t, files.PathAdded, ids.of("new"), "Trip/Trip.mkv")
		}, []string{"replaced Trip/Trip.mkv"}, []string{"Modified " + p + "Trip/Trip.mkv"}},
		{"added first, new name", func(t *testing.T, ids fileIDs) files.PathChange {
			c := at(t, files.PathAdded, ids.of("new"), "Trip/Trip - 2160p.mkv")
			c.Upgrade = true
			return c
		}, func(t *testing.T, ids fileIDs) files.PathChange {
			c := at(t, files.PathRemoved, ids.of("old"), "Trip/Trip - 1080p.mkv")
			c.ReplacedBy = ids.of("new")
			return c
		}, []string{"replaced Trip/Trip - 1080p.mkv>Trip/Trip - 2160p.mkv"}, []string{"Created " + p + "Trip/Trip - 2160p.mkv"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				s := newShelf(t, "")
				s.wait(t, 0)
				ids := fileIDs{}
				s.tell(t, c.first(t, ids))
				settle(time.Minute)
				expectReported(t, s.den)
				if cs := s.changes(t); cs[0].Words != "This change is waiting for the other half of its upgrade." {
					t.Errorf("the first half reads %q", cs[0].Words)
				}
				s.tell(t, c.second(t, ids))
				settle(10 * time.Minute)
				if got := s.queued(t); !slices.Equal(got, c.queued) {
					t.Errorf("Den's queue holds %v, want %v", got, c.queued)
				}
				expectReported(t, s.den, c.want...)
			})
		})
	}
}

// TestAnUpgradeHalfGoesAloneOnceThePairingBoundPassed removes an
// upgrade's old file and never lands its new one: the removal waits
// Timings.UpgradePairing, then goes alone.
func TestAnUpgradeHalfGoesAloneOnceThePairingBoundPassed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newShelf(t, "")
		s.wait(t, 0)
		ids := fileIDs{}
		c := at(t, files.PathRemoved, ids.of("old"), "Trip/Trip.mkv")
		c.ReplacedBy = ids.of("new")
		s.tell(t, c)
		settle(110 * time.Second)
		expectReported(t, s.den)
		settle(20 * time.Second)
		expectReported(t, s.den, "Deleted "+p+"Trip/Trip.mkv")
	})
}

// TestAFailingServerKeepsItsChanges stops Den: each send fails, the
// Admin reads why and that SlipStream retries, a later change to the
// file folds into the waiting one, and once Den is back it takes the
// merged change.
func TestAFailingServerKeepsItsChanges(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newShelf(t, "")
		ids := fileIDs{}
		up := s.w.Down(t, host)
		s.tell(t, at(t, files.PathAdded, ids.of("old"), "Trip/Trip.mkv"))
		settle(time.Minute)
		c := change(t, s.changes(t), "Trip/Trip.mkv")
		if c.Attempts == 0 || c.State != "due" || !strings.HasSuffix(c.Words, "SlipStream retries automatically.") {
			t.Errorf("while Den is down the change is %s after %d attempts: %s", c.State, c.Attempts, c.Words)
		}
		s.tell(t, at(t, files.PathRemoved, ids.of("old"), "Trip/Trip.mkv"), at(t, files.PathAdded, ids.of("new"), "Trip/Trip.mkv"))
		if got := s.queued(t); !slices.Equal(got, []string{"added Trip/Trip.mkv"}) {
			t.Errorf("Den's queue holds %v, want the one added", got)
		}
		up()
		settle(20 * time.Minute)
		expectReported(t, s.den, "Created "+p+"Trip/Trip.mkv")
		if c := change(t, s.changes(t), "Trip/Trip.mkv"); c.State != "sent" || c.Words != "Sent the new file." {
			t.Errorf("once Den is back the change is %s: %s", c.State, c.Words)
		}
	})
}

// TestACutLinkIsNoFailure drops seedbox's calls while the roster still
// sees it in step: the change waits at its step with no attempt counted
// and no failure shown, and goes once the link is whole.
func TestACutLinkIsNoFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newShelf(t, "")
		s.w.Cut(s.w.Seedbox)
		s.tell(t, at(t, files.PathAdded, fileIDs{}.of("f"), "Trip/Trip.mkv"))
		settle(time.Minute)
		if c := change(t, s.changes(t), "Trip/Trip.mkv"); c.Attempts != 0 || c.State != "due" || c.Words != "This change is waiting to be sent." {
			t.Errorf("while seedbox's link is cut the change is %s after %d attempts: %s", c.State, c.Attempts, c.Words)
		}
		expectReported(t, s.den)
		s.w.Online(s.w.Seedbox)
		settle(time.Minute)
		expectReported(t, s.den, "Created "+p+"Trip/Trip.mkv")
	})
}

// TestTheControllerSendsWhileTheReachingNodeIsAway takes seedbox offline
// with a fallback available: the Controller reports to Den itself.
func TestTheControllerSendsWhileTheReachingNodeIsAway(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newShelf(t, "")
		s.w.Offline(s.w.Seedbox)
		s.tell(t, at(t, files.PathAdded, fileIDs{}.of("f"), "Trip/Trip.mkv"))
		settle(time.Minute)
		expectReported(t, s.den, "Created "+p+"Trip/Trip.mkv")
	})
}

// TestChangesWaitForTheReachingNodeWithNoFallback names a fallback
// address the Controller can't reach and takes seedbox offline: nothing
// goes, and once seedbox is back Den takes every change in one report.
func TestChangesWaitForTheReachingNodeWithNoFallback(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newShelf(t, "http://192.168.1.99:8096")
		s.w.Offline(s.w.Seedbox)
		ids := fileIDs{}
		s.tell(t, at(t, files.PathAdded, ids.of("1"), "Trip/1.mkv"))
		settle(time.Minute)
		s.tell(t, at(t, files.PathAdded, ids.of("2"), "Trip/2.mkv"))
		settle(time.Hour)
		expectReported(t, s.den)
		s.w.Online(s.w.Seedbox)
		settle(time.Minute)
		expectReported(t, s.den, "Created "+p+"Trip/1.mkv, Created "+p+"Trip/2.mkv")
	})
}

// TestRefreshWholeLibraryCoversWaitingChanges presses Refresh whole
// library while a change waits: Den refreshes its libraries, and the
// Admin reads that the refresh covered the change.
func TestRefreshWholeLibraryCoversWaitingChanges(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newShelf(t, "http://192.168.1.99:8096")
		s.w.Offline(s.w.Seedbox)
		s.tell(t, at(t, files.PathAdded, fileIDs{}.of("f"), "Trip/Trip.mkv"))
		if status, body := s.o.Call(t, http.MethodPost, servers+"/"+s.id+"/refresh", ""); status != http.StatusAccepted {
			t.Fatalf("refresh Den = %d %s", status, body)
		}
		settle(time.Minute)
		s.w.Online(s.w.Seedbox)
		settle(time.Minute)
		if scans := records(t, s.den, "jellyfin.scan"); len(scans) != 2 {
			t.Errorf("Den refreshed its libraries %d times, want twice", len(scans))
		}
		expectReported(t, s.den)
		c := change(t, s.changes(t), "Trip/Trip.mkv")
		if c.Result != "covered" || c.Words != "A refresh of the whole library covered this change." {
			t.Errorf("the change is %s %s: %s", c.State, c.Result, c.Words)
		}
	})
}

// windowsPlex is Living room Plex on Windows, with a Shows section at
// C:\Media\Shows.
var windowsPlex = fakeplex.Server{
	Name: "Living room", Version: "1.43.5.11029", Identifier: plexID, Owner: "plex-admin", Root: `C:\Media`, Tokens: []string{plexToken},
	Sections: []fakeplex.Section{{Key: "2", Title: "Shows", Type: "show", Folders: []string{`C:\Media\Shows`}, Scanned: true}},
}

// plexToken is a token Living room Plex takes beside plex.tv's, for a
// test that seeds watch data as a person does in Plex.
const plexToken = "living-room-test-token"

// shows is Living room Plex, which seedbox reaches, with the Shows root
// mapped and scanned, once the section refresh its mapping queued has
// gone. Shows holds an episode of Harbor, so Plex scanned it with
// content, and each of placed, a path under Shows, is there before that
// refresh.
func newShows(t *testing.T, placed ...string) (shelf, mstest.Plex) {
	t.Helper()
	r := plexRig{rig: newRig(t)}
	r.plex = r.w.Plex(t, windowsPlex, fakeplextv.Account{Username: "plex-admin", PlexPass: true})
	for _, rel := range append([]string{"Harbor/Season 01/Harbor - S01E01.mkv"}, placed...) {
		place(t, r.plex.Server.Media, "Shows/"+rel)
	}
	r.signIn(t)
	s := shelf{rig: r.rig, id: r.addPlex(t, r.w.Seedbox.String()).ID, root: r.w.Root(t, "Shows").ID}
	s.mapTo(t, `C:\Media\Shows`)
	s.w.Scanned(t)
	settle(15 * time.Second)
	if got := refreshed(t, r.plex); !slices.Equal(got, []string{"section 2"}) {
		t.Fatalf("saving the mapping refreshed %v", got)
	}
	return s, r.plex
}

// refreshed is every refresh Plex took, as "section key" or "section key
// at folder".
func refreshed(t *testing.T, px mstest.Plex) []string {
	t.Helper()
	var out []string
	for _, r := range px.Server.Records(t) {
		if r.Kind != fakeplex.Refreshes.Kind {
			continue
		}
		var f fakeplex.Refresh
		if err := json.Unmarshal(r.Data, &f); err != nil {
			t.Fatal(err)
		}
		s := "section " + f.Section
		if f.Path != "" {
			s += " at " + f.Path
		}
		out = append(out, s)
	}
	return out
}

// TestASeasonPackIsOnePlexRefresh lands 40 episodes in one season folder
// of a Windows Plex: Plex refreshes that one folder once, named with the
// server's backslashes, and holds the last episode as an item.
func TestASeasonPackIsOnePlexRefresh(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s, px := newShows(t)
		ids := fileIDs{}
		var cs []files.PathChange
		for e := 1; e <= 40; e++ {
			name := fmt.Sprintf("Voyage/Season 01/Voyage - S01E%02d.mkv", e)
			place(t, px.Server.Media, "Shows/"+name)
			cs = append(cs, at(t, files.PathAdded, ids.of(name), name))
		}
		s.tell(t, cs...)
		settle(time.Minute)
		want := []string{"section 2", `section 2 at C:\Media\Shows\Voyage\Season 01`}
		if got := refreshed(t, px); !slices.Equal(got, want) {
			t.Errorf("Plex took the refreshes %v, want %v", got, want)
		}
		plexKey(t, px, "Voyage/Season 01/Voyage - S01E40.mkv")
	})
}

// TestElevenFoldersAreOneSectionRefresh lands files in eleven folders of
// one section: past Timings.SectionFolders Plex refreshes the section
// once instead.
func TestElevenFoldersAreOneSectionRefresh(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s, px := newShows(t)
		ids := fileIDs{}
		var cs []files.PathChange
		for f := 1; f <= 11; f++ {
			name := fmt.Sprintf("Voyage %d/Voyage %d.mkv", f, f)
			cs = append(cs, at(t, files.PathAdded, ids.of(name), name))
		}
		s.tell(t, cs...)
		settle(time.Minute)
		if got := refreshed(t, px); !slices.Equal(got, []string{"section 2", "section 2"}) {
			t.Errorf("Plex took the refreshes %v, want one section refresh after the mapping's", got)
		}
	})
}
