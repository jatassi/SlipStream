package mediaserversapi_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	fakeplex "github.com/jatassi/slipstream/fakes/plex"
	fakeplextv "github.com/jatassi/slipstream/fakes/plextv"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaservers"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
)

// place writes rel under media, as the disk queue leaves a file's bytes
// before it tells the Media Servers.
func place(t *testing.T, media, rel string) {
	t.Helper()
	at := filepath.Join(media, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(at), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(at, []byte(rel), 0o600); err != nil {
		t.Fatal(err)
	}
}

// scanned is whether the server's mapping of root reads as scanned.
func (s shelf) scanned(t *testing.T) bool {
	t.Helper()
	status, body := s.o.Call(t, http.MethodGet, servers+"/"+s.id+"/libraries", "")
	if status != http.StatusOK {
		t.Fatalf("list the libraries = %d %s", status, body)
	}
	for _, m := range decode[mediaservers.Libraries](t, body).Mappings {
		if m.RootID == s.root.String() {
			return m.Scanned
		}
	}
	t.Fatalf("no mapping of %s in %s", s.root, body)
	return false
}

// items are the files of every item Den added.
func (s shelf) items(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, raw := range records(t, s.den, "jellyfin.item") {
		var it struct {
			Event string
			Files []string
		}
		if err := json.Unmarshal(raw, &it); err != nil {
			t.Fatal(err)
		}
		if it.Event == "added" {
			out = append(out, it.Files...)
		}
	}
	return out
}

// TestAFreshFolderIsScannedBeforeJellyfinTakesReports lands the first
// file in Shelves, which Den never scanned with content: Den would ignore
// its report (#101), so it scans every library first, takes the report
// once that scan ends, and holds the item. The mapping then reads scanned,
// and the next file is only reported.
func TestAFreshFolderIsScannedBeforeJellyfinTakesReports(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := addShelf(t, "")
		settle(15 * time.Second)
		ids := fileIDs{}
		place(t, s.den.Media, "Shelves/Trip/Trip.mkv")
		s.tell(t, at(t, files.PathAdded, ids.of("trip"), "Trip/Trip.mkv"))
		settle(time.Minute)
		if n := len(records(t, s.den, "jellyfin.scan")); n != 2 {
			t.Errorf("Den scanned %d times, want the mapping's refresh and the fresh folder's scan", n)
		}
		expectReported(t, s.den, "Created "+p+"Trip/Trip.mkv")
		if !s.scanned(t) {
			t.Error("the mapping doesn't read scanned once the scan ended")
		}
		settle(2 * time.Minute)
		if c := change(t, s.changes(t), "Trip/Trip.mkv"); c.State != "sent" {
			t.Errorf("the change is %s: %s", c.State, c.Words)
		}
		place(t, s.den.Media, "Shelves/Walk/Walk.mkv")
		s.tell(t, at(t, files.PathAdded, ids.of("walk"), "Walk/Walk.mkv"))
		settle(3 * time.Minute)
		if n := len(records(t, s.den, "jellyfin.scan")); n != 2 {
			t.Errorf("Den scanned %d times once the folder was scanned, want still 2", n)
		}
		if got := s.items(t); !slices.Equal(got, []string{p + "Trip/Trip.mkv", p + "Walk/Walk.mkv"}) {
			t.Errorf("Den holds items at %v, want the reported file among them", got)
		}
	})
}

// TestARetryNeverStartsASecondScan loses the answer of the call that
// started the fresh folder's scan, as a link that broke while it ran:
// the retry finds the scan ended and starts none.
func TestARetryNeverStartsASecondScan(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := addShelf(t, "")
		settle(15 * time.Second)
		s.w.LoseAnswer(s.w.Seedbox, func(j *msv1.Job) bool { return j.GetFresh().GetRequestedUnixMs() != 0 })
		place(t, s.den.Media, "Shelves/Trip/Trip.mkv")
		s.tell(t, at(t, files.PathAdded, fileIDs{}.of("trip"), "Trip/Trip.mkv"))
		settle(2 * time.Minute)
		if n := len(records(t, s.den, "jellyfin.scan")); n != 2 {
			t.Errorf("Den scanned %d times, want the mapping's refresh and one fresh folder scan", n)
		}
		expectReported(t, s.den, "Created "+p+"Trip/Trip.mkv")
		if c := change(t, s.changes(t), "Trip/Trip.mkv"); c.Attempts != 0 {
			t.Errorf("the change counts %d attempts after a lost answer, want none", c.Attempts)
		}
	})
}

// freshPlex is Living room Plex with a Films section it never scanned.
var freshPlex = fakeplex.Server{
	Name: "Living room", Version: "1.43.5.11029", Identifier: plexID, Owner: "plex-admin", Root: `C:\Media`,
	Sections: []fakeplex.Section{{Key: "5", Title: "Films", Type: "movie", Folders: []string{`C:\Media\Films`}}},
}

// TestAFreshFolderIsScannedBeforePlexRefreshesIt lands the first file in
// a section Plex never scanned with content, which ignores a folder
// refresh (#101): Plex refreshes the section, and the folder once that
// scan ends.
func TestAFreshFolderIsScannedBeforePlexRefreshesIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := plexRig{rig: newRig(t)}
		r.plex = r.w.Plex(t, freshPlex, fakeplextv.Account{Username: "plex-admin", PlexPass: true})
		r.signIn(t)
		s := shelf{rig: r.rig, id: r.addPlex(t, r.w.Seedbox.String()).ID, root: r.w.Root(t, "Films").ID}
		s.mapTo(t, `C:\Media\Films`)
		settle(15 * time.Second)
		place(t, r.plex.Server.Media, "Films/Heat (1995)/Heat (1995).mkv")
		s.tell(t, at(t, files.PathAdded, fileIDs{}.of("heat"), "Heat (1995)/Heat (1995).mkv"))
		settle(time.Minute)
		want := []string{"section 5", "section 5", `section 5 at C:\Media\Films\Heat (1995)`}
		if got := refreshed(t, r.plex); !slices.Equal(got, want) {
			t.Errorf("Plex took the refreshes %v, want %v", got, want)
		}
		if !s.scanned(t) {
			t.Error("the mapping doesn't read scanned once the scan ended")
		}
	})
}

// TestAReportDroppedByJellyfinsOwnScanIsSentAgain starts Den's own
// scheduled scan inside the monitor delay of a report, which drops it:
// with nothing else queued, the report goes again once the scan ends, and
// the change reads sent only after Den settled the second one.
func TestAReportDroppedByJellyfinsOwnScanIsSentAgain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newShelf(t, "")
		s.tell(t, at(t, files.PathAdded, fileIDs{}.of("trip"), "Trip/Trip.mkv"))
		settle(15 * time.Second)
		expectReported(t, s.den, "Created "+p+"Trip/Trip.mkv")
		s.den.Cue(t, "jellyfin-scan", struct{}{})
		settle(2 * time.Minute)
		expectReported(t, s.den, "Created "+p+"Trip/Trip.mkv", "Created "+p+"Trip/Trip.mkv")
		settle(2 * time.Minute)
		if c := change(t, s.changes(t), "Trip/Trip.mkv"); c.State != "sent" {
			t.Errorf("the change is %s: %s", c.State, c.Words)
		}
		expectReported(t, s.den, "Created "+p+"Trip/Trip.mkv", "Created "+p+"Trip/Trip.mkv")
	})
}
