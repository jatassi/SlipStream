package mediaserversapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	fakeplex "github.com/jatassi/slipstream/fakes/plex"
	fakeplextv "github.com/jatassi/slipstream/fakes/plextv"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaservers/mstest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

const (
	plexID      = "fake-plex-living-room"
	homeConn    = "http://" + mstest.HomeIP + ":32400"
	seedboxConn = "http://" + mstest.SeedboxIP + ":32400"
	day         = 24 * time.Hour
)

var livingRoom = fakeplex.Server{Name: "Living room", Version: "1.43.5.11029", Identifier: plexID, Owner: "plex-admin", Root: "/media"}

// serverOut is the fields of a server view the tests read.
type serverOut struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	Address         string             `json:"address"`
	FallbackAddress string             `json:"fallbackAddress"`
	Fallback        string             `json:"fallback"`
	Identity        string             `json:"identity"`
	PlexPass        *bool              `json:"plexPass"`
	State           mediaservers.State `json:"state"`
}

type plexRig struct {
	rig
	plex mstest.Plex
}

func newPlexRig(t *testing.T) plexRig {
	t.Helper()
	r := newRig(t)
	return plexRig{rig: r, plex: r.w.Plex(t, livingRoom, fakeplextv.Account{Username: "plex-admin", PlexPass: true})}
}

func decode[T any](t *testing.T, body string) T {
	t.Helper()
	var v T
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
	return v
}

func (r plexRig) expect(t *testing.T, method, path, body string, want int) string {
	t.Helper()
	status, reply := r.o.Call(t, method, path, body)
	if status != want {
		t.Fatalf("%s %s = %d %s, want %d", method, path, status, reply, want)
	}
	return reply
}

// signIn runs the PIN flow the way the sign-in tab does: start, open the
// page, poll until plex.tv has the Admin's approval.
func (r plexRig) signIn(t *testing.T) {
	t.Helper()
	pin := decode[mediaservers.PlexPin](t, r.expect(t, http.MethodPost, servers+"/plex/pin", `{"forward":"https://slipstream.example.test/settings/system/media-servers"}`, http.StatusCreated))
	state := func() string {
		return decode[mediaservers.PlexPinState](t, r.expect(t, http.MethodGet, servers+"/plex/pin/"+jsonInt(pin.ID), "", http.StatusOK)).State
	}
	if s := state(); s != "waiting" {
		t.Fatalf("before approval the sign-in is %s", s)
	}
	r.plex.Approve(t, pin.AuthURL)
	time.Sleep(time.Second)
	if s := state(); s != "signed_in" {
		t.Fatalf("after approval the sign-in is %s", s)
	}
}

func jsonInt(n int64) string { return strconv.FormatInt(n, 10) }

func (r plexRig) addPlex(t *testing.T, reaching string) serverOut {
	t.Helper()
	draft := `{"kind":"plex","name":"Living room Plex","plexServer":"` + plexID + `","reachingNodeId":"` + reaching + `"}`
	return decode[serverOut](t, r.expect(t, http.MethodPost, servers, draft, http.StatusCreated))
}

func (r plexRig) account(t *testing.T) mediaservers.AccountView {
	t.Helper()
	return decode[mediaservers.AccountView](t, r.expect(t, http.MethodGet, servers+"/plex/account", "", http.StatusOK))
}

// renew runs the renewal task the way the scheduler runs it.
func (r plexRig) renew(t *testing.T) error {
	t.Helper()
	var units []tasks.Unit
	err := r.w.Lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		units, err = r.w.Renewal.Units(ctx, rd)
		return err
	})
	if err != nil || len(units) != 1 {
		t.Fatalf("the renewal task lists %v, %v", units, err)
	}
	return r.w.Renewal.Run(t.Context(), units[0], &tasks.Progress{})
}

func (r plexRig) settingsVersion(t *testing.T) int64 {
	t.Helper()
	in := r.w.Intent(t, r.w.Home)
	if len(in.GetServers()) != 1 {
		t.Fatalf("home's intent has %d servers", len(in.GetServers()))
	}
	return in.GetServers()[0].GetSettingsVersion()
}

// TestPlexSignInThroughPlexTV signs in with a PIN on fake-plex-tv: the
// account, its Plex Pass and its 7-day sign-in read back, and the renewal
// task lists the account.
func TestPlexSignInThroughPlexTV(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newPlexRig(t)
		status, body := r.o.Call(t, http.MethodGet, servers+"/plex/account", "")
		expectProblem(t, status, body, http.StatusConflict, `"code":"plex_not_signed_in"`)
		pin := decode[mediaservers.PlexPin](t, r.expect(t, http.MethodPost, servers+"/plex/pin", `{"forward":"https://slipstream.example.test/m"}`, http.StatusCreated))
		if !strings.HasPrefix(pin.AuthURL, "http://plextv-auth.fake.localhost/auth#?") || !strings.Contains(pin.AuthURL, "forwardUrl=https%3A%2F%2Fslipstream.example.test%2Fm") {
			t.Errorf("the sign-in page is %s", pin.AuthURL)
		}
		r.signIn(t)
		a := r.account(t)
		if a.Name != "plex-admin" || !a.PlexPass || a.Refused || a.ExpiresAt.Sub(a.SignedInAt).Round(time.Hour) != 7*day {
			t.Errorf("the account reads %+v", a)
		}
	})
}

// TestAPlexSignInExpires lets the PIN's 30 minutes pass unapproved.
func TestAPlexSignInExpires(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newPlexRig(t)
		pin := decode[mediaservers.PlexPin](t, r.expect(t, http.MethodPost, servers+"/plex/pin", `{"forward":"https://slipstream.example.test/m"}`, http.StatusCreated))
		time.Sleep(31 * time.Minute)
		s := decode[mediaservers.PlexPinState](t, r.expect(t, http.MethodGet, servers+"/plex/pin/"+jsonInt(pin.ID), "", http.StatusOK))
		if s.State != "expired" {
			t.Errorf("past its life the sign-in is %s", s.State)
		}
		r.expect(t, http.MethodGet, servers+"/plex/pin/7", "", http.StatusNotFound)
	})
}

// TestPlexServersListEachConnectionFromEveryHolder refuses home's
// connection: both Nodes read seedbox's and neither home's, and home's
// address suggests where the server runs.
func TestPlexServersListEachConnectionFromEveryHolder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newPlexRig(t)
		r.signIn(t)
		if _, err := r.plex.Server.Faults.Set(fault.Rule{ID: "home-down", Target: "plex", Origin: "home", Fault: fault.Refuse{}}); err != nil {
			t.Fatal(err)
		}
		out := decode[struct {
			Servers []mediaservers.PlexServerView `json:"servers"`
		}](t, r.expect(t, http.MethodGet, servers+"/plex/servers", "", http.StatusOK))
		if len(out.Servers) != 1 {
			t.Fatalf("the account lists %+v", out.Servers)
		}
		s := out.Servers[0]
		if s.MachineIdentifier != plexID || s.Name != "Living room" || len(s.Connections) != 2 {
			t.Fatalf("the server reads %+v", s)
		}
		expectReached(t, s.Connections[0], homeConn, false)
		expectReached(t, s.Connections[1], seedboxConn, true)
		if s.SuggestedRunsOn == nil || s.SuggestedRunsOn.Name != "home" || s.SuggestedReaching.Name != "home" {
			t.Errorf("the suggestions are %+v and %+v", s.SuggestedRunsOn, s.SuggestedReaching)
		}
	})
}

// expectReached is c at uri, a local plain HTTP connection, which both
// Nodes reached or neither did.
func expectReached(t *testing.T, c mediaservers.PlexConnection, uri string, ok bool) {
	t.Helper()
	if c.URI != uri || !c.Local || c.HTTPS || len(c.ReachedFrom) != 2 {
		t.Fatalf("the connection reads %+v, want %s", c, uri)
	}
	for _, from := range c.ReachedFrom {
		if from.OK != ok {
			t.Errorf("%s from %s responded: %t", c.URI, from.NodeID, from.OK)
		}
	}
}

// TestAddPlexPicksTheFirstConnectionThatResponds saves the local
// connection plex.tv lists first, and with it refused, the next one.
func TestAddPlexPicksTheFirstConnectionThatResponds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newPlexRig(t)
		draft := `{"kind":"plex","name":"Living room Plex","plexServer":"` + plexID + `","reachingNodeId":"` + r.w.Seedbox.String() + `"}`
		status, body := r.o.Call(t, http.MethodPost, servers, draft)
		expectProblem(t, status, body, http.StatusConflict, `"code":"plex_not_signed_in"`)
		r.signIn(t)
		if _, err := r.plex.Server.Faults.Set(fault.Rule{ID: "home-down", Target: "plex", Origin: "home", Fault: fault.Refuse{}}); err != nil {
			t.Fatal(err)
		}
		v := r.addPlex(t, r.w.Seedbox.String())
		if v.Address != seedboxConn || v.FallbackAddress != "" || v.Fallback != "available" || v.Identity != plexID || v.PlexPass == nil || !*v.PlexPass {
			t.Errorf("with home's connection refused the server saved as %+v", v)
		}
		r.plex.Server.Faults.Clear("")
		r.expect(t, http.MethodDelete, servers+"/"+v.ID, "", http.StatusNoContent)
		if v := r.addPlex(t, r.w.Home.String()); v.Address != homeConn {
			t.Errorf("with every connection up the server saved at %s", v.Address)
		}
		in := r.w.Intent(t, r.w.Home)
		if len(in.GetServers()) != 1 || in.GetServers()[0].GetDeviceId() == "" {
			t.Errorf("home's intent carries %v", in.GetServers())
		}
	})
}

// TestRenewalWaitsForTheMarginAndKeepsAnUnchangedServer renews only once
// less than the 3-day margin is left, and a renewal that finds the same
// token leaves the settings version alone.
func TestRenewalWaitsForTheMarginAndKeepsAnUnchangedServer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newPlexRig(t)
		r.signIn(t)
		r.addPlex(t, r.w.Home.String())
		first := r.account(t)
		time.Sleep(3 * day)
		if err := r.renew(t); err != nil {
			t.Fatal(err)
		}
		if a := r.account(t); a.RenewedAt != nil || !a.ExpiresAt.Equal(first.ExpiresAt) {
			t.Errorf("with 4 days left the renewal ran: %+v", a)
		}
		time.Sleep(day + time.Minute)
		if err := r.renew(t); err != nil {
			t.Fatal(err)
		}
		a := r.account(t)
		if a.RenewedAt == nil || a.ExpiresAt.Sub(first.ExpiresAt).Round(time.Hour) != 4*day {
			t.Errorf("with under 3 days left the account reads %+v", a)
		}
		if v := r.settingsVersion(t); v != 1 {
			t.Errorf("a renewal with the same token moved the settings version to %d", v)
		}
	})
}

// TestANewTokenMovesTheSettingsVersion signs in again, which gets a new
// server token from plex.tv: the server's settings version moves, so its
// reaching Node fetches the new token.
func TestANewTokenMovesTheSettingsVersion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newPlexRig(t)
		r.signIn(t)
		v := r.addPlex(t, r.w.Home.String())
		r.signIn(t)
		if got := r.settingsVersion(t); got != 2 {
			t.Errorf("after a new token the settings version is %d", got)
		}
		if tested := decode[mediaservers.Tested](t, r.expect(t, http.MethodPost, servers+"/"+v.ID+"/test", "", http.StatusOK)); !tested.Reaching.OK {
			t.Errorf("the new token tests as %+v", tested)
		}
		if err := r.renew(t); err != nil {
			t.Fatal(err)
		}
		if got := r.settingsVersion(t); got != 2 {
			t.Errorf("a renewal with no new token moved the settings version to %d", got)
		}
	})
}

// TestARefusedRenewalRaisesSignInRefused signs the device out on plex.tv:
// the renewal gets 422, the account reads refused and the Plex server's
// state asks for a new sign-in, which clears it.
func TestARefusedRenewalRaisesSignInRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newPlexRig(t)
		r.signIn(t)
		v := r.addPlex(t, r.w.Home.String())
		r.plex.TV.Cue(t, fakeplextv.SignOut.Name, map[string]string{})
		time.Sleep(5 * day)
		err := r.renew(t)
		const words = "SlipStream's sign-in to Plex has expired or was revoked. Sign in with Plex again."
		if skip, ok := errors.AsType[*tasks.SkipError](err); !ok || !skip.Owned || skip.Cause != words {
			t.Errorf("the refused renewal ended %v", err)
		}
		if a := r.account(t); !a.Refused {
			t.Errorf("after the refusal the account reads %+v", a)
		}
		got := decode[serverOut](t, r.expect(t, http.MethodGet, servers+"/"+v.ID, "", http.StatusOK))
		if got.State.Level != "fail" || got.State.Words != "SlipStream's sign-in to Living room Plex has expired or was revoked. Sign in with Plex again." {
			t.Errorf("the server's state is %+v", got.State)
		}
		r.signIn(t)
		if a := r.account(t); a.Refused {
			t.Errorf("a new sign-in left the account refused")
		}
	})
}

// TestSignOutOfPlexRemovesEveryPlexServer keeps the Jellyfin server.
func TestSignOutOfPlexRemovesEveryPlexServer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newPlexRig(t)
		r.w.Jellyfin(t, host, den(floorVer))
		r.add(t)
		r.signIn(t)
		r.addPlex(t, r.w.Home.String())
		r.expect(t, http.MethodDelete, servers+"/plex/account", "", http.StatusNoContent)
		list := decode[struct {
			Servers []serverOut `json:"servers"`
		}](t, r.expect(t, http.MethodGet, servers, "", http.StatusOK))
		if len(list.Servers) != 1 || list.Servers[0].Name != "Den" {
			t.Errorf("after the sign-out the servers are %+v", list.Servers)
		}
		r.expect(t, http.MethodGet, servers+"/plex/account", "", http.StatusConflict)
	})
}
