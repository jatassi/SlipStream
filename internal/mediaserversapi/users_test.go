package mediaserversapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/jellyfin"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaservers/access"
	"github.com/jatassi/slipstream/internal/mediaservers/mstest"
)

// denRig is Den, shown to Members, which Sam was offered.
type denRig struct {
	rig
	den    faketest.Served
	server string
	sam    mstest.Member
	row    string
}

// newDenRig adds Den, with each of others as a user besides its
// administrator and no fallback address set, and offers it to Sam.
func newDenRig(t *testing.T, others ...string) denRig {
	t.Helper()
	return newDenRigOn(t, newRig(t), "", others...)
}

// newDenRigOn is newDenRig on r with fallback as Den's fallback address,
// "" for none.
func newDenRigOn(t *testing.T, r rig, fallback string, others ...string) denRig {
	t.Helper()
	srv := den(floorVer)
	for _, name := range others {
		srv.Users = append(srv.Users, jellyfin.User{Name: name, Password: name + "-password"})
	}
	d := denRig{rig: r, den: r.w.Jellyfin(t, host, srv)}
	draft := `{"kind":"jellyfin","name":"Den","address":"` + address + `","reachingNodeId":"` + r.w.Seedbox.String() + `","apiKey":"` + denKey + `"`
	if fallback != "" {
		draft += `,"fallbackAddress":"` + fallback + `"`
	}
	status, body := r.o.Call(t, http.MethodPost, servers, draft+"}")
	if status != http.StatusCreated {
		t.Fatalf("add Den = %d %s", status, body)
	}
	d.server = decode[serverOut](t, body).ID
	r.show(t, d.server)
	d.sam = r.w.Member(t, "sam")
	d.row = d.offerTo(t, d.sam)
	return d
}

// offerTo offers Den to m and answers the row's id.
func (d denRig) offerTo(t *testing.T, m mstest.Member) string {
	t.Helper()
	status, body := d.offer(t, m.ID, d.server, false)
	if status != http.StatusOK {
		t.Fatalf("offer Den = %d %s", status, body)
	}
	return decode[struct{ Row walkRow }](t, body).Row.ID
}

func (d denRig) makeAs(t *testing.T, m mstest.Member, row, username, password string) (status int, body string) {
	t.Helper()
	return d.o.CallAs(t, http.MethodPost, "/api/v1/account/media-servers/"+row+"/account",
		`{"username":"`+username+`","password":"`+password+`"}`, m.Session)
}

// make makes Sam's account on Den, named sam.
func (d denRig) make(t *testing.T) {
	t.Helper()
	if status, body := d.makeAs(t, d.sam, d.row, "sam", "sam-password"); status != http.StatusOK {
		t.Fatalf("make Sam's account = %d %s", status, body)
	}
}

// adminRow is the row as an Admin reads it, false once it is deleted.
func (d denRig) adminRow(t *testing.T, id string) (adminRowOut, bool) {
	t.Helper()
	all := decode[struct{ Rows []adminRowOut }](t, d.expectOK(t, servers+"/access"))
	i := slices.IndexFunc(all.Rows, func(r adminRowOut) bool { return r.ID == id })
	if i < 0 {
		return adminRowOut{}, false
	}
	return all.Rows[i], true
}

type adminRowOut struct {
	ID       string       `json:"id"`
	State    access.State `json:"state"`
	NeedsYou bool         `json:"needsYou"`
	Member   struct {
		AccountID string `json:"accountId"`
		Name      string `json:"name"`
	} `json:"member"`
	Problem *struct {
		Code   string `json:"code"`
		Detail string `json:"detail"`
	} `json:"problem"`
}

// denUserOut is a user as Den lists it.
type denUserOut struct {
	ID     string `json:"Id"`
	Name   string `json:"Name"`
	Policy struct {
		IsDisabled bool `json:"IsDisabled"`
	} `json:"Policy"`
}

// denNow is every user Den lists now.
func (d denRig) denNow(t *testing.T) []denUserOut {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, address+"/Users?ApiKey="+denKey, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := d.den.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var out []denUserOut
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func named(users []denUserOut, name string) (denUserOut, bool) {
	i := slices.IndexFunc(users, func(u denUserOut) bool { return u.Name == name })
	if i < 0 {
		return denUserOut{}, false
	}
	return users[i], true
}

// TestARestartMidMakeLeavesNoUser stops the Controller at each point of
// Sam's make, while Den already has a stranger's user named sam: once it
// starts again, Den has no user the make left, the stranger's user is as
// it was, and Sam may make the account again.
func TestARestartMidMakeLeavesNoUser(t *testing.T) {
	for _, c := range []struct {
		name  string
		rules []fault.Rule
	}{
		{"before the create", []fault.Rule{{Route: "create-user", Fault: fault.Hang{}}}},
		{"after the create, before its id is written", []fault.Rule{
			{Route: "create-user", Fault: fault.Truncate{}, Times: 1}, {Route: "users", Fault: fault.Hang{}},
		}},
		{"after the id is written", []fault.Rule{{Route: "user", Fault: fault.Hang{}}}},
		{"after the libraries", []fault.Rule{{Route: "user-password", Fault: fault.Hang{}}}},
		{"after the password", []fault.Rule{{Route: "update-user", Fault: fault.Hang{}}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				d := newDenRig(t, "sam")
				stranger, _ := named(d.denNow(t), "sam")
				d.stopMidMake(t, c.rules)
				d.w.Restart(t)
				synctest.Wait()

				users := d.denNow(t)
				if len(users) != 2 || users[1] != stranger {
					t.Errorf("Den lists %+v, want its administrator and the stranger %+v", users, stranger)
				}
				if row, _ := d.adminRow(t, d.row); row.State != access.Offered {
					t.Errorf("the row is %+v, want offered", row)
				}
				status, body := d.makeAs(t, d.sam, d.row, "sam2", "sam-password")
				if status != http.StatusOK {
					t.Errorf("make again = %d %s", status, body)
				}
			})
		})
	}
}

// stopMidMake starts Sam's make with rules on Den, waits for it to stop
// at a hung call and ends it as a Controller that stops does: its calls
// and Writes fail. Den then serves without the rules.
func (d denRig) stopMidMake(t *testing.T, rules []fault.Rule) {
	t.Helper()
	for i, rule := range rules {
		rule.ID, rule.Target = "stop "+strconv.Itoa(i), fault.Target(jellyfin.Fake.Name)
		if _, err := d.den.Faults.Set(rule); err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		for i := range rules {
			d.den.Faults.Clear("stop " + strconv.Itoa(i))
		}
	}()
	ctx, cancel := context.WithCancel(t.Context())
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/v1/account/media-servers/"+d.row+"/account",
		strings.NewReader(`{"username":"sam","password":"sam-password"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", identity.CookieName+"="+d.sam.Session)
	done := make(chan struct{})
	go func() {
		defer close(done)
		d.o.Send(req)
	}()
	synctest.Wait()
	select {
	case <-done:
		t.Fatal("the make ended before its hung call")
	default:
	}
	cancel()
	<-done
}

// TestAMakeTakesTheRouteTheControllerTested makes Sam's account on Den
// with the fallback address in each state the Controller's own test left
// it: a fallback it couldn't use sends the make through seedbox, and one
// that stopped responding since its test answers media_server_unreached
// with nothing made through seedbox, and the row left making for the
// sweep, as the Controller couldn't check for a user to delete.
func TestAMakeTakesTheRouteTheControllerTested(t *testing.T) {
	const fallback = "192.168.1.21:8096"
	for _, c := range []struct {
		name     string
		fallback string
		stale    bool
		status   int
		detail   string
	}{
		{name: "an unusable fallback", fallback: "http://192.168.1.99:8096", status: http.StatusOK},
		{name: "a fallback that stopped responding", fallback: "http://" + fallback, stale: true, status: http.StatusBadGateway,
			detail: "The Controller couldn't use the fallback address. Test the server again."},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := newRig(t)
				r.w.Jellyfin(t, fallback, den(floorVer))
				d := newDenRigOn(t, r, c.fallback)
				if c.stale {
					d.w.Down(t, fallback)
				}
				status, body := d.makeAs(t, d.sam, d.row, "sam", "sam-password")
				if status != c.status {
					t.Fatalf("make = %d %s, want %d", status, body, c.status)
				}
				_, made := named(d.denNow(t), "sam")
				row, _ := d.adminRow(t, d.row)
				count := decode[struct {
					MemberCount int `json:"memberCount"`
				}](t, d.expectOK(t, servers+"/"+d.server)).MemberCount
				switch {
				case c.stale && (made || row.State != access.Making || row.Problem == nil || row.Problem.Detail != c.detail || count != 0):
					t.Errorf("after the make Den has sam %t, the row is %+v and Den counts %d Members", made, row, count)
				case !c.stale && (!made || row.State != access.Active || count != 1):
					t.Errorf("after the make Den has sam %t, the row is %+v and Den counts %d Members", made, row, count)
				}
			})
		})
	}
}

// TestAMakeRefusedByNameKeepsTheOffer makes Sam's account with a name
// Den's user Kit has: Sam reads that the name is taken, the Admin reads
// Den's reply, and Den has no new user.
func TestAMakeRefusedByNameKeepsTheOffer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDenRig(t, "kit")
		status, body := d.makeAs(t, d.sam, d.row, "KIT", "sam-password")
		expectProblem(t, status, body, http.StatusConflict, `"code":"username_taken"`)
		if row, _ := d.adminRow(t, d.row); row.State != access.Offered || row.Problem == nil || row.Problem.Code != "username_taken" ||
			!strings.Contains(row.Problem.Detail, "already exists") {
			t.Errorf("the row is %+v, want offered with Den's reply", row)
		}
		if users := d.denNow(t); len(users) != 2 {
			t.Errorf("Den lists %+v, want only its administrator and kit", users)
		}
	})
}

// TestLinkableUsersLeaveOutWhatCantBeLinked lists Den's users with an
// administrator, Sam's user disabled once their Account is gone, Lee's
// user linked to Pat, and Kit: only Kit can be linked.
func TestLinkableUsersLeaveOutWhatCantBeLinked(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDenRig(t, "kit", "lee")
		d.make(t)
		if status, body := d.o.CallSignedIn(t, http.MethodDelete, servers+"/access/"+d.row, ""); status != http.StatusOK {
			t.Fatalf("remove Sam's access = %d %s", status, body)
		}
		if err := d.w.Identity.DeleteAccount(t.Context(), identity.Admin{}, d.sam.ID); err != nil {
			t.Fatal(err)
		}
		pat := d.w.Member(t, "pat")
		patRow := d.offerTo(t, pat)
		lee, _ := named(d.denNow(t), "lee")
		if status, body := d.o.CallSignedIn(t, http.MethodPost, servers+"/access/"+patRow+"/link", `{"userId":"`+lee.ID+`"}`); status != http.StatusOK {
			t.Fatalf("link Lee = %d %s", status, body)
		}
		got := decode[struct{ Users []struct{ Name string } }](t, d.expectOK(t, servers+"/"+d.server+"/users")).Users
		if len(got) != 1 || got[0].Name != "kit" {
			t.Errorf("Den's linkable users are %+v, want kit alone", got)
		}
	})
}

// TestADeleteWithDenDownLeavesItsRemovalOnNeedsYou deletes Sam's Account
// while Den is down: the delete goes through, the row waits under Needs
// you with no Account and Health warns of it. Once Den is back, Try again
// disables Sam's user and deletes the row and the warning.
func TestADeleteWithDenDownLeavesItsRemovalOnNeedsYou(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDenRig(t)
		d.make(t)
		d.openHealth(t)
		up := d.w.Down(t, host)
		if err := d.w.Identity.DeleteAccount(t.Context(), identity.Admin{}, d.sam.ID); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		row, ok := d.adminRow(t, d.row)
		if !ok || row.State != access.Removing || !row.NeedsYou || row.Member.AccountID != "" || row.Member.Name == "" || row.Problem == nil {
			t.Fatalf("after the delete the row is %+v, want removing under Needs you with no Account", row)
		}
		if warned := d.accessWarnings(t); len(warned) != 1 || warned[0].Level != "warn" ||
			!strings.Contains(warned[0].Subject, row.Member.Name) || !strings.Contains(warned[0].Subject, "Den") {
			t.Fatalf("Health warns %+v, want one warning naming %s and Den", warned, row.Member.Name)
		}
		up()
		if status, body := d.o.CallSignedIn(t, http.MethodDelete, servers+"/access/"+d.row, ""); status != http.StatusOK {
			t.Fatalf("try again = %d %s", status, body)
		}
		synctest.Wait()
		if row, ok := d.adminRow(t, d.row); ok {
			t.Errorf("after Try again the row is %+v, want it deleted", row)
		}
		if sam, _ := named(d.denNow(t), "sam"); !sam.Policy.IsDisabled {
			t.Errorf("Den's user sam is %+v, want it disabled", sam)
		}
		if warned := d.accessWarnings(t); len(warned) != 0 {
			t.Errorf("Health still warns %+v", warned)
		}
	})
}

// accessWarnings is the Health items of Members' access.
func (d denRig) accessWarnings(t *testing.T) []healthItem {
	t.Helper()
	return slices.DeleteFunc(d.mediaServersHealth(t), func(it healthItem) bool { return !strings.HasPrefix(it.Key, "access") })
}

// TestAnOfferRacingAnAccountDeleteLeavesNothing deletes Sam's Account
// through identity while Den is slow to disable Sam's user: an Admin's
// offer started meanwhile waits for the delete, then finds the Account
// gone, and no row is left.
func TestAnOfferRacingAnAccountDeleteLeavesNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDenRig(t)
		d.make(t)
		if _, err := d.den.Faults.Set(fault.Rule{ID: "slow", Target: fault.Target(jellyfin.Fake.Name), Route: "user", Fault: fault.Delay(time.Minute)}); err != nil {
			t.Fatal(err)
		}
		deleted := make(chan error, 1)
		go func() { deleted <- d.w.Identity.DeleteAccount(context.Background(), identity.Admin{}, d.sam.ID) }()
		synctest.Wait()
		type answer struct {
			status int
			body   string
		}
		offered := make(chan answer, 1)
		go func() {
			status, body := d.offer(t, d.sam.ID, d.server, false)
			offered <- answer{status, body}
		}()
		synctest.Wait()
		select {
		case a := <-offered:
			t.Fatalf("the offer answered %d %s during the delete", a.status, a.body)
		default:
		}
		if err := <-deleted; err != nil {
			t.Fatal(err)
		}
		a := <-offered
		expectProblem(t, a.status, a.body, http.StatusNotFound, `"code":"account_not_found"`)
		if all := decode[struct{ Rows []adminRowOut }](t, d.expectOK(t, servers+"/access")); len(all.Rows) != 0 {
			t.Errorf("the delete left rows %+v", all.Rows)
		}
	})
}
