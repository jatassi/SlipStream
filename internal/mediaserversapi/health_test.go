package mediaserversapi_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	fakeplextv "github.com/jatassi/slipstream/fakes/plextv"
	"github.com/jatassi/slipstream/internal/files"
)

// healthItem is the fields of a Health item the tests read.
type healthItem struct {
	Key     string `json:"key"`
	Level   string `json:"level"`
	Subject string `json:"subject"`
	Text    string `json:"text"`
	Unsure  bool   `json:"unsure"`
	Detail  []struct {
		Label string `json:"label"`
	} `json:"detail"`
}

func (it healthItem) has(label string) bool {
	for _, f := range it.Detail {
		if f.Label == label {
			return true
		}
	}
	return false
}

// openHealth starts home's Health, which the rig's Health API then reads.
func (r rig) openHealth(t *testing.T) {
	t.Helper()
	r.health.Store(r.w.Health(t))
	synctest.Wait()
}

// mediaServersHealth is the Media Servers category's items, as the Health
// page reads them.
func (r rig) mediaServersHealth(t *testing.T) []healthItem {
	t.Helper()
	status, body := r.o.Call(t, http.MethodGet, "/api/v1/system/health", "")
	if status != http.StatusOK {
		t.Fatalf("read Health = %d %s", status, body)
	}
	type category struct {
		ID    string       `json:"id"`
		Label string       `json:"label"`
		Items []healthItem `json:"items"`
	}
	cs := decode[struct{ Categories []category }](t, body).Categories
	i := slices.IndexFunc(cs, func(c category) bool { return c.ID == "media-servers" })
	if i < 0 || cs[i].Label != "Media Servers" || cs[i-1].ID != "download-clients" {
		t.Fatalf("Health lists %+v, want Media Servers after Download Clients", cs)
	}
	return cs[i].Items
}

// expectHealth fails unless Health holds exactly one Media Servers item,
// at level, whose text has each of parts, and answers it.
func (r rig) expectHealth(t *testing.T, level string, parts ...string) healthItem {
	t.Helper()
	items := r.mediaServersHealth(t)
	if len(items) != 1 || items[0].Level != level {
		t.Fatalf("Media Servers Health holds %+v, want one %s item", items, level)
	}
	for _, p := range parts {
		if !strings.Contains(items[0].Text, p) {
			t.Errorf("the item reads %q, want it to hold %q", items[0].Text, p)
		}
	}
	return items[0]
}

func (r rig) expectHealthy(t *testing.T) {
	t.Helper()
	if items := r.mediaServersHealth(t); len(items) != 0 {
		t.Errorf("Media Servers Health holds %+v, want none", items)
	}
}

// expectTransitions fails unless the events published are want, each
// "unhealthy <reason>" or "recovered <reason>", for server id of kind.
func (r rig) expectTransitions(t *testing.T, id, kind string, want ...string) {
	t.Helper()
	var got []string
	for _, tr := range r.w.Transitions() {
		word := "unhealthy"
		if tr.Recovered {
			word = "recovered"
		}
		if tr.ServerID != id || tr.Kind != kind {
			t.Errorf("an event names %s %s, want %s %s", tr.Kind, tr.ServerID, kind, id)
		}
		got = append(got, word+" "+tr.Reason)
	}
	if !slices.Equal(got, want) {
		t.Errorf("the events published are %q, want %q", got, want)
	}
}

// stop stops Den, as a server that stopped does, until start.
func (s shelf) stop(t *testing.T) (start func()) {
	t.Helper()
	return s.w.Down(t, host)
}

// TestAServerNotRespondingIsUnhealthyAfterMarkUnhealthyAfter stops Den:
// Health raises it only once it hasn't responded for Mark unhealthy
// after, 5 minutes, and clears it once Den responds again.
func TestAServerNotRespondingIsUnhealthyAfterMarkUnhealthyAfter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newShelf(t, "")
		s.openHealth(t)
		s.expectHealthy(t)
		start := s.stop(t)
		settle(4 * time.Minute)
		s.expectHealthy(t)
		s.expectTransitions(t, s.id, "jellyfin")
		settle(2 * time.Minute)
		it := s.expectHealth(t, "fail", "Den hasn't responded to seedbox since 00:", "Check that Jellyfin is running and that seedbox can reach "+host+".")
		if it.Subject != "Den" || !it.has("Last response") {
			t.Errorf("the item is %+v, want Den with its last response under Details", it)
		}
		s.expectTransitions(t, s.id, "jellyfin", "unhealthy not_responding")
		start()
		settle(2 * time.Minute)
		s.expectHealthy(t)
		s.expectTransitions(t, s.id, "jellyfin", "unhealthy not_responding", "recovered not_responding")
	})
}

// TestFailingSendsAreUnhealthyAfterMarkUnhealthyAfter has Den refuse
// every path report while it still responds: Health raises the failing
// sends after 5 minutes and clears them once a report goes.
func TestFailingSendsAreUnhealthyAfterMarkUnhealthyAfter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newShelf(t, "")
		s.openHealth(t)
		if _, err := s.den.Faults.Set(fault.Rule{ID: "reports", Target: "jellyfin", Route: "updated", Fault: fault.Status{Code: http.StatusInternalServerError}}); err != nil {
			t.Fatal(err)
		}
		s.tell(t, at(t, files.PathAdded, fileIDs{}.of("f"), "Trip/Trip.mkv"))
		settle(4 * time.Minute)
		s.expectHealthy(t)
		settle(2 * time.Minute)
		s.expectHealth(t, "fail", "Den hasn't accepted changes since 00:", "SlipStream retries automatically. Check Jellyfin's log for the cause.")
		s.expectTransitions(t, s.id, "jellyfin", "unhealthy sends_failing")
		s.den.Faults.Clear("reports")
		settle(20 * time.Minute)
		expectReported(t, s.den, "Created "+p+"Trip/Trip.mkv")
		s.expectHealthy(t)
		s.expectTransitions(t, s.id, "jellyfin", "unhealthy sends_failing", "recovered sends_failing")
	})
}

// TestAnotherServerIsUnhealthyAtOnce serves another Jellyfin at Den's
// address and restarts seedbox's watch: Health raises it at the watch's
// first check, without waiting for Mark unhealthy after, and while seedbox is
// offline the item stays an Error that SlipStream can't check.
func TestAnotherServerIsUnhealthyAtOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newShelf(t, "")
		s.openHealth(t)
		other := den(floorVer)
		other.ID = "0f6f1c9e3b2a4d5e8f7a6b5c4d3e2f10"
		s.w.Jellyfin(t, host, other)
		s.w.Stop(s.w.Seedbox)
		s.w.Start(s.w.Seedbox)
		settle(30 * time.Second)
		s.expectHealth(t, "fail", "A different Jellyfin server responds at "+host+". Check the address, or confirm the new server under Details.")
		s.expectTransitions(t, s.id, "jellyfin", "unhealthy other_server")
		s.w.Offline(s.w.Seedbox)
		settle(time.Minute)
		if it := s.expectHealth(t, "fail", "Den can't be checked while seedbox is offline."); !it.Unsure {
			t.Errorf("with seedbox offline the item is %+v, want it unsure", it)
		}
		s.expectTransitions(t, s.id, "jellyfin", "unhealthy other_server")
	})
}

// TestAnOfflineReachingNodeIsAWarning takes seedbox offline: Health warns
// that Den can't be checked and that the Controller sends its changes,
// raises no unhealthy event, and clears once seedbox is back.
func TestAnOfflineReachingNodeIsAWarning(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newShelf(t, "")
		s.openHealth(t)
		s.w.Offline(s.w.Seedbox)
		synctest.Wait()
		s.expectHealth(t, "warn", "Den can't be checked while seedbox is offline. SlipStream sends changes through the Controller until seedbox is back.")
		settle(10 * time.Minute)
		s.expectTransitions(t, s.id, "jellyfin")
		s.w.Online(s.w.Seedbox)
		settle(time.Minute)
		s.expectHealthy(t)
	})
}

// TestARootWithNoMappingIsAWarning adds Den beside a root it has no
// mapping for.
func TestARootWithNoMappingIsAWarning(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t)
		r.w.Jellyfin(t, host, den(floorVer))
		r.add(t)
		r.w.Root(t, "Shelves")
		r.openHealth(t)
		it := r.expectHealth(t, "warn", "Den won't get changes for files in ", ". Add a Path mapping, or mark the root as not shown on this server.")
		if !strings.HasSuffix(it.Subject, " on Den") {
			t.Errorf("the item's subject is %q", it.Subject)
		}
	})
}

// TestARefusedPlexSignInIsUnhealthyAtOnce signs the device out on
// plex.tv: the refused renewal raises Living room Plex at once with the
// spec's words, and a new sign-in clears it.
func TestARefusedPlexSignInIsUnhealthyAtOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newPlexRig(t)
		r.signIn(t)
		v := r.addPlex(t, r.w.Home.String())
		r.openHealth(t)
		r.expectHealthy(t)
		r.plex.TV.Cue(t, fakeplextv.SignOut.Name, map[string]string{})
		time.Sleep(5 * day)
		_ = r.renew(t)
		synctest.Wait()
		r.expectHealth(t, "fail", "SlipStream's sign-in to Living room Plex has expired or was revoked. Sign in with Plex again.")
		r.expectTransitions(t, v.ID, "plex", "unhealthy sign_in_refused")
		r.signIn(t)
		settle(time.Minute)
		r.expectHealthy(t)
		r.expectTransitions(t, v.ID, "plex", "unhealthy sign_in_refused", "recovered sign_in_refused")
	})
}

// TestAContactChangeTellsTheList stops Den: once seedbox reports it isn't
// responding, media-servers.changed names Den, and the list then reads
// its new state, so the state line refreshes before Health raises it.
func TestAContactChangeTellsTheList(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newShelf(t, "")
		before := len(s.w.Changed())
		s.stop(t)
		settle(time.Minute)
		var told bool
		for _, c := range s.w.Changed()[before:] {
			told = told || slices.Equal(c.ServerIDs, []string{s.id})
		}
		if !told {
			t.Fatalf("media-servers.changed published %v once Den stopped, want one naming %s", s.w.Changed()[before:], s.id)
		}
		status, body := s.o.Call(t, http.MethodGet, servers, "")
		if status != http.StatusOK {
			t.Fatalf("list the servers = %d %s", status, body)
		}
		type view struct {
			Contact string `json:"contact"`
		}
		got := decode[struct{ Servers []view }](t, body).Servers
		if len(got) != 1 || got[0].Contact != "not_responding" {
			t.Errorf("the list reads %+v, want Den not responding", got)
		}
	})
}

// TestTheTelevisionLightsWhileAServerResponds stops Den: seedbox's
// television is lit while Den responds and dark once it doesn't, and
// home's, which reaches no server, stays dark.
func TestTheTelevisionLightsWhileAServerResponds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newShelf(t, "")
		if !s.w.Television(s.w.Seedbox) || s.w.Television(s.w.Home) {
			t.Fatalf("seedbox's television lit %t and home's %t, want seedbox's alone", s.w.Television(s.w.Seedbox), s.w.Television(s.w.Home))
		}
		s.stop(t)
		settle(time.Minute)
		if s.w.Television(s.w.Seedbox) {
			t.Error("seedbox's television is lit once Den stopped, want it dark")
		}
	})
}
