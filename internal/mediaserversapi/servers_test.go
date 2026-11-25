package mediaserversapi_test

import (
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/jellyfin"
	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/healthapi"
	"github.com/jatassi/slipstream/internal/mediaservers/mstest"
	"github.com/jatassi/slipstream/internal/mediaserversapi"
)

const (
	host     = "192.168.1.20:8096"
	address  = "http://" + host
	denID    = "ddc0b5b7f2824180aae9f2f245dd00ca"
	denKey   = "den-key"
	servers  = "/api/v1/media-servers"
	shelves  = "/media/Shelves"
	floorVer = "12.2.0"
)

// den is the Jellyfin named Den at version, with a Shelves library.
func den(version string) jellyfin.Server {
	return jellyfin.Server{
		Name: "Den", Version: version, ID: denID, Keys: []string{denKey},
		Users: []jellyfin.User{{Name: "ada", Password: "ada-password", Admin: true}},
		Root:  "/media", Libraries: []jellyfin.Library{{Name: "Shelves", Type: "homevideos", Folders: []string{shelves}}},
	}
}

type rig struct {
	w *mstest.World
	o *ownertest.Owner
	// health is home's Health once a test opens it.
	health *atomic.Pointer[health.Health]
}

func newRig(t *testing.T) rig {
	t.Helper()
	w := mstest.New(t)
	h := &atomic.Pointer[health.Health]{}
	return rig{w: w, health: h, o: ownertest.ServeOver(t, w.Lib, w.Identity, func(r *api.Router) {
		mediaserversapi.Routes(r, w)
		healthapi.Routes(r, watch{h})
	})}
}

type watch struct {
	h *atomic.Pointer[health.Health]
}

func (w watch) Health() (*health.Health, bool) {
	h := w.h.Load()
	return h, h != nil
}

func (r rig) draft(name, reaching, addr string) string {
	return `{"kind":"jellyfin","name":"` + name + `","address":"` + addr + `","reachingNodeId":"` + reaching + `","apiKey":"` + denKey + `"}`
}

// add saves Den, which seedbox reaches, and answers its id.
func (r rig) add(t *testing.T) string {
	t.Helper()
	status, body := r.o.Call(t, http.MethodPost, servers, r.draft("Den", r.w.Seedbox.String(), address))
	if status != http.StatusCreated && status != http.StatusOK {
		t.Fatalf("add Den = %d %s", status, body)
	}
	return ownertest.FirstID(body)
}

func expectProblem(t *testing.T, status int, body string, want int, parts ...string) {
	t.Helper()
	if status != want {
		t.Fatalf("answered %d %s, want %d", status, body, want)
	}
	for _, p := range parts {
		if !strings.Contains(body, p) {
			t.Errorf("answered %s, want it to hold %s", body, p)
		}
	}
}

// TestAddRefusesAServerBelowTheFloor serves Den at Jellyfin 10.10.7:
// the Admin reads which version SlipStream needs, and nothing is saved.
func TestAddRefusesAServerBelowTheFloor(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t)
		r.w.Jellyfin(t, host, den("10.10.7"))
		status, body := r.o.Call(t, http.MethodPost, servers, r.draft("Den", r.w.Seedbox.String(), address))
		expectProblem(t, status, body, http.StatusUnprocessableEntity, `"code":"media_server_too_old"`,
			`Jellyfin 10.10.7 is older than SlipStream supports. Update it to 12.2 or later, then save again.`)
		r.o.Expect(t, http.MethodGet, servers, "", http.StatusOK, `{"servers":[]}`)
	})
}

// TestAddRefusesLocalhostOnAWorker names localhost on seedbox with no
// fallback address, which the Controller can't use.
func TestAddRefusesLocalhostOnAWorker(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t)
		status, body := r.o.Call(t, http.MethodPost, servers, r.draft("Den", r.w.Seedbox.String(), "http://localhost:8096"))
		expectProblem(t, status, body, http.StatusUnprocessableEntity, `"code":"media_server_localhost_on_worker"`,
			`The Controller can't use http://localhost:8096, because localhost on seedbox is seedbox itself.`, `"location":"body.address"`)
	})
}

// TestAddRecordsTheReachingNodeAsAHolder saves Den through seedbox: the
// Admin sees seedbox holds its key, as the current reaching Node, and the
// list hears of the change.
func TestAddRecordsTheReachingNodeAsAHolder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t)
		r.w.Jellyfin(t, host, den(floorVer))
		id := r.add(t)
		r.o.Expect(t, http.MethodGet, servers+"/"+id+"/credential-holders", "", http.StatusOK,
			`{"holders":[{"node":{"id":"id1","name":"seedbox"},"fetchedAt":"T","current":true}]}`)
	})
}

// TestEachWriteTellsTheList adds Den, renames it and saves its mappings:
// each write publishes media-servers.changed for Den, so the list reads
// its mappings needed again. Den's first contact tells it too.
func TestEachWriteTellsTheList(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t)
		r.w.Jellyfin(t, host, den(floorVer))
		id := r.add(t)
		synctest.Wait()
		if status, body := r.o.Call(t, http.MethodPatch, servers+"/"+id, `{"name":"Loft"}`); status != http.StatusOK {
			t.Fatalf("rename Den = %d %s", status, body)
		}
		if status, body := r.o.Call(t, http.MethodPut, servers+"/"+id+"/mappings", `{"mappings":[]}`); status != http.StatusOK {
			t.Fatalf("save the mappings = %d %s", status, body)
		}
		got := r.w.Changed()
		if len(got) != 4 {
			t.Fatalf("media-servers.changed published %v, want four times", got)
		}
		for _, c := range got {
			if len(c.ServerIDs) != 1 || c.ServerIDs[0] != id {
				t.Errorf("media-servers.changed named %v, want %s", c.ServerIDs, id)
			}
		}
	})
}

// TestConfirmIdentityAfterAnotherServerAnswers swaps Den for another
// Jellyfin at its address: a test reads the other server, and once the
// Admin confirms it the server tests clean.
func TestConfirmIdentityAfterAnotherServerAnswers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t)
		r.w.Jellyfin(t, host, den(floorVer))
		id := r.add(t)
		other := den(floorVer)
		other.ID = "0f1e2d3c4b5a69788796a5b4c3d2e1f0"
		r.w.Jellyfin(t, host, other)
		status, body := r.o.Call(t, http.MethodPost, servers+"/"+id+"/test", "")
		expectProblem(t, status, body, http.StatusOK,
			`"reaching":{"ok":false,"words":"A different Jellyfin server responds at 192.168.1.20:8096. Check the address, or confirm the new server under Details."}`)
		status, body = r.o.Call(t, http.MethodPost, servers+"/"+id+"/confirm-identity", "")
		expectProblem(t, status, body, http.StatusOK, `"identity":"`+other.ID+`"`)
		status, body = r.o.Call(t, http.MethodPost, servers+"/"+id+"/test", "")
		expectProblem(t, status, body, http.StatusOK, `"reaching":{"ok":true,"words":"seedbox connected to Den."}`)
	})
}

// TestSaveMappingsRefusesAFolderOutsideTheLibraries maps the root to a
// folder Den has no library in: the Admin reads which folders Den has.
func TestSaveMappingsRefusesAFolderOutsideTheLibraries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t)
		r.w.Jellyfin(t, host, den(floorVer))
		root := r.w.Root(t, "Shelves")
		id := r.add(t)
		status, body := r.o.Call(t, http.MethodPut, servers+"/"+id+"/mappings",
			`{"mappings":[{"rootId":"`+root.ID.String()+`","shown":true,"serverFolder":"/srv/elsewhere"}]}`)
		expectProblem(t, status, body, http.StatusUnprocessableEntity, `"code":"mapping_outside_libraries"`,
			`{"location":"body.mappings[0].serverFolder","message":"/srv/elsewhere isn't inside a library folder on Den. Its library folders are /media/Shelves. Add the folder to a library on the server first."}`)
	})
}

// TestRehomeUnconfirmsTheMapping confirms the root's mapping, then moves
// the root: Den holds its changes again until the Admin confirms anew.
func TestRehomeUnconfirmsTheMapping(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t)
		r.w.Jellyfin(t, host, den(floorVer))
		root := r.w.Root(t, "Shelves")
		id := r.add(t)
		status, body := r.o.Call(t, http.MethodGet, servers+"/"+id+"/libraries", "")
		expectProblem(t, status, body, http.StatusOK, `"serverFolder":"`+shelves+`"`, `"guess":true`)
		status, body = r.o.Call(t, http.MethodPut, servers+"/"+id+"/mappings",
			`{"mappings":[{"rootId":"`+root.ID.String()+`","shown":true,"serverFolder":"`+shelves+`"}]}`)
		expectProblem(t, status, body, http.StatusOK, `"confirmed":true`)
		status, body = r.o.Call(t, http.MethodGet, servers+"/"+id, "")
		expectProblem(t, status, body, http.StatusOK, `"mappingsNeeded":0`)

		r.w.Rehome(t, root.ID, "Shelves")
		status, body = r.o.Call(t, http.MethodGet, servers+"/"+id, "")
		expectProblem(t, status, body, http.StatusOK, `"mappingsNeeded":1`,
			`"words":"Den won't get changes for files in Shelves library root on home. Add a Path mapping, or mark the root as not in a Den library."`)
		status, body = r.o.Call(t, http.MethodGet, servers+"/"+id+"/libraries", "")
		expectProblem(t, status, body, http.StatusOK, `"confirmed":false`,
			`SlipStream won't send Den changes for files in Shelves library root on home until you confirm its Path mapping.`)
	})
}

// TestSettingsRefuseAMarginNoLongerThanTheCadence sets the renewal
// margin to the renewal cadence's one day.
func TestSettingsRefuseAMarginNoLongerThanTheCadence(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t)
		status, body := r.o.Call(t, http.MethodPatch, servers+"/settings", `{"values":{"mediaServers.renewMargin":86400}}`)
		expectProblem(t, status, body, http.StatusUnprocessableEntity, `"code":"renew_margin_too_short"`,
			`\"Renew when the sign-in expires within\" must be longer than \"Renew Plex sign-in every\", which is 1 day.`)
	})
}

// TestShowToMembersPicksTheShownLibraries turns on Show to Members for
// Den: refused while no Path mapping shows a library, then Shelves, the
// library behind the shown mapping, becomes Den's library for Members.
// An offer doesn't count as a Member who can watch, and the last library
// can't be taken away while Den is shown.
func TestShowToMembersPicksTheShownLibraries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t)
		r.w.Jellyfin(t, host, den(floorVer))
		root := r.w.Root(t, "Shelves")
		id := r.add(t)
		status, body := r.o.Call(t, http.MethodPatch, servers+"/"+id, `{"membersSee":true}`)
		expectProblem(t, status, body, http.StatusUnprocessableEntity, `"code":"no_member_libraries"`)
		status, body = r.o.Call(t, http.MethodPut, servers+"/"+id+"/mappings",
			`{"mappings":[{"rootId":"`+root.ID.String()+`","shown":true,"serverFolder":"`+shelves+`"}]}`)
		expectProblem(t, status, body, http.StatusOK, `"confirmed":true`)
		status, body = r.o.Call(t, http.MethodPatch, servers+"/"+id, `{"membersSee":true}`)
		expectProblem(t, status, body, http.StatusOK, `"membersSee":true`, `"name":"Shelves"`, `"memberCount":0`)

		if status, body := r.offer(t, r.w.Member(t, "sam").ID, id, false); status != http.StatusOK {
			t.Fatalf("offer Den = %d %s", status, body)
		}
		if body := r.expectOK(t, servers+"/"+id); !strings.Contains(body, `"memberCount":0`) {
			t.Errorf("Den with one offer reads %s, want no Member who can watch", body)
		}
		status, body = r.o.Call(t, http.MethodPatch, servers+"/"+id, `{"memberLibraries":[]}`)
		expectProblem(t, status, body, http.StatusUnprocessableEntity, `"code":"no_member_libraries"`)
	})
}

// TestAPlexServerTakesNoAddressForMembers sets an Address for Members on
// the Plex server, whose apps find it through the Member's Plex account.
func TestAPlexServerTakesNoAddressForMembers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newPlexRig(t)
		r.signIn(t)
		id := r.addPlex(t, r.w.Home.String()).ID
		status, body := r.o.Call(t, http.MethodPatch, servers+"/"+id, `{"memberAddress":"https://plex.example.test"}`)
		expectProblem(t, status, body, http.StatusUnprocessableEntity, `"location":"body.memberAddress"`)
	})
}
