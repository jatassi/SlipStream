package mediaserversapi_test

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/mediaservers/access"
	"github.com/jatassi/slipstream/internal/storage"
)

// accessOut is the fields of an access row the tests read.
type accessOut struct {
	ID       string         `json:"id"`
	State    access.State   `json:"state"`
	NeedsYou bool           `json:"needsYou"`
	Actions  []access.Event `json:"actions"`
	Server   struct {
		ID string `json:"id"`
	} `json:"server"`
	Plex *struct {
		PlexHome bool `json:"plexHome"`
	} `json:"plex"`
}

type accessListOut struct {
	Rows      []accessOut `json:"rows"`
	Offerable []struct {
		ID string `json:"id"`
	} `json:"offerable"`
}

// show turns on Show to Members for server, with one library for
// Members.
func (r rig) show(t *testing.T, server string) {
	t.Helper()
	status, body := r.o.Call(t, http.MethodPatch, servers+"/"+server, `{"membersSee":true,"memberLibraries":[{"key":"1","name":"Shelves"}]}`)
	if status != http.StatusOK {
		t.Fatalf("show %s to Members = %d %s", server, status, body)
	}
}

func (r rig) offer(t *testing.T, account identity.AccountID, server string, plexHome bool) (status int, body string) {
	t.Helper()
	return r.o.CallSignedIn(t, http.MethodPut, "/api/v1/accounts/"+account.String()+"/media-servers/"+server,
		`{"plexHome":`+strconv.FormatBool(plexHome)+`}`)
}

// accessOf is account's rows and the servers it may be offered, as an
// Admin lists them.
func (r rig) accessOf(t *testing.T, account identity.AccountID) accessListOut {
	t.Helper()
	status, body := r.o.Call(t, http.MethodGet, servers+"/access?account="+account.String(), "")
	if status != http.StatusOK {
		t.Fatalf("list %s's access = %d %s", account, status, body)
	}
	return decode[accessListOut](t, body)
}

// TestOfferRefusesWhatCantBeOffered offers Den, shown to Members or not,
// to each Account an Admin could pick.
func TestOfferRefusesWhatCantBeOffered(t *testing.T) {
	for _, c := range []struct {
		name     string
		shown    bool
		admin    bool
		gone     bool
		plexHome bool
		status   int
		code     string
	}{
		{name: "an Admin", shown: true, admin: true, status: http.StatusConflict, code: "access_state"},
		{name: "a server not shown to Members", status: http.StatusConflict, code: "not_shown_to_members"},
		{name: "Plex Home on Jellyfin", shown: true, plexHome: true, status: http.StatusUnprocessableEntity, code: "plex_home_not_plex"},
		{name: "an Account that is gone", shown: true, gone: true, status: http.StatusNotFound, code: "account_not_found"},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := newRig(t)
				r.w.Jellyfin(t, host, den(floorVer))
				server := r.add(t)
				if c.shown {
					r.show(t, server)
				}
				account := r.w.Member(t, "sam").ID
				switch {
				case c.admin:
					account = identitytest.Admin(t, r.w.Lib, "bea@example.test")
				case c.gone:
					account = identity.AccountID{UUID: storage.NewUUID()}
				}
				status, body := r.offer(t, account, server, c.plexHome)
				expectProblem(t, status, body, c.status, `"code":"`+c.code+`"`)
				if all := decode[accessListOut](t, r.expectOK(t, servers+"/access")); len(all.Rows) != 0 {
					t.Errorf("a refused offer left rows %+v", all.Rows)
				}
			})
		})
	}
}

func (r rig) expectOK(t *testing.T, path string) string {
	t.Helper()
	status, body := r.o.Call(t, http.MethodGet, path, "")
	if status != http.StatusOK {
		t.Fatalf("GET %s = %d %s", path, status, body)
	}
	return body
}

var errRolledBack = errors.New("the redemption Write rolled back")

// TestARedeemedSignupLinkOffersItsServers redeems a link offering Den, the
// Plex server as Plex Home and a server removed since: the Account gets an
// offered row for each server still there, in the redemption Write, and
// none when that Write rolls back.
func TestARedeemedSignupLinkOffersItsServers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newPlexRig(t)
		r.signIn(t)
		plexServer := r.addPlex(t, r.w.Home.String()).ID
		r.w.Jellyfin(t, host, den(floorVer))
		denServer := r.add(t)
		r.show(t, denServer)
		r.show(t, plexServer)
		member := r.w.Member(t, "sam")
		grants := identity.Grants{MediaServers: []identity.ServerGrant{
			{Server: serverUUID(t, denServer)},
			{Server: serverUUID(t, plexServer), PlexHome: true},
			{Server: storage.NewUUID()},
		}}
		redemption := identity.Redemption{Account: member.ID, Grants: grants}

		if err := r.redeem(t, redemption, false); !errors.Is(err, errRolledBack) {
			t.Fatalf("the rolled back redemption answered %v", err)
		}
		if rows := r.accessOf(t, member.ID).Rows; len(rows) != 0 {
			t.Fatalf("a rolled back redemption left rows %+v", rows)
		}
		if err := r.redeem(t, redemption, true); err != nil {
			t.Fatal(err)
		}
		got := map[string]accessOut{}
		for _, row := range r.accessOf(t, member.ID).Rows {
			got[row.Server.ID] = row
		}
		if len(got) != 2 || got[denServer].State != access.Offered || got[plexServer].State != access.Offered {
			t.Fatalf("the redeemed link offered %+v, want Den and the Plex server offered", got)
		}
		if p := got[plexServer].Plex; p == nil || !p.PlexHome {
			t.Errorf("the Plex row is %+v, want it offered as Plex Home", got[plexServer])
		}
	})
}

// redeem runs the Granter inside a redemption Write, as identity does,
// which rolls back unless commit.
func (r rig) redeem(t *testing.T, red identity.Redemption, commit bool) error {
	t.Helper()
	var after func(context.Context)
	err := r.w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		after, err = r.w.Moves().Granter().Redeemed(ctx, tx, red)
		if err == nil && !commit {
			err = errRolledBack
		}
		return err
	})
	if err == nil {
		after(t.Context())
	}
	return err
}

func serverUUID(t *testing.T, id string) storage.UUID {
	t.Helper()
	u, err := storage.ParseUUID(id)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
