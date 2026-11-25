package mediaserversapi_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/jellyfin"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaservers/access"
	"github.com/jatassi/slipstream/internal/mediaservers/mstest"
)

// walkRig is a Plex server and Den, both shown to Members, a Member and
// Kit, an existing user on Den.
type walkRig struct {
	plexRig
	den     faketest.Served
	servers map[access.Way]string
	member  mstest.Member
}

func newWalkRig(t *testing.T) walkRig {
	t.Helper()
	r := newPlexRig(t)
	r.signIn(t)
	plexServer := r.addPlex(t, r.w.Home.String()).ID
	srv := den(floorVer)
	srv.Users = append(srv.Users, jellyfin.User{Name: "kit", Password: "kit-password"})
	f := r.w.Jellyfin(t, host, srv)
	denServer := r.add(t)
	r.show(t, plexServer)
	r.show(t, denServer)
	return walkRig{plexRig: r, den: f, servers: map[access.Way]string{access.WayPlexTV: plexServer, access.WayUsers: denServer}, member: r.w.Member(t, "sam")}
}

// walkRow is the fields of an access row the walk reads.
type walkRow struct {
	ID     string       `json:"id"`
	State  access.State `json:"state"`
	Server struct {
		ID string `json:"id"`
	} `json:"server"`
	Plex *struct {
		PlexHome bool `json:"plexHome"`
	} `json:"plex"`
	Problem *struct {
		Code string `json:"code"`
	} `json:"problem"`
}

// row is the one row on way's server, the Member's or one left by their
// deleted Account, false for none.
func (r walkRig) row(t *testing.T, way access.Way) (walkRow, bool) {
	t.Helper()
	all := decode[struct{ Rows []walkRow }](t, r.expectOK(t, servers+"/access"))
	i := slices.IndexFunc(all.Rows, func(row walkRow) bool { return row.Server.ID == r.servers[way] })
	if i < 0 {
		return walkRow{}, false
	}
	return all.Rows[i], true
}

// denUsers is every user on Den after its last change.
func (r walkRig) denUsers(t *testing.T) []jellyfin.UserRecord {
	t.Helper()
	recs := records(t, r.den, "jellyfin.users")
	if len(recs) == 0 {
		return nil
	}
	var l jellyfin.UserList
	if err := json.Unmarshal(recs[len(recs)-1], &l); err != nil {
		t.Fatal(err)
	}
	return l.Users
}

// denUser is Den's user named name, false for none.
func (r walkRig) denUser(t *testing.T, name string) (jellyfin.UserRecord, bool) {
	t.Helper()
	users := r.denUsers(t)
	i := slices.IndexFunc(users, func(u jellyfin.UserRecord) bool { return u.Name == name })
	if i < 0 {
		return jellyfin.UserRecord{}, false
	}
	return users[i], true
}

// deleteOnDen deletes Sam's user on Den, as an Admin does in Jellyfin.
func (r walkRig) deleteOnDen(t *testing.T) {
	t.Helper()
	sam, ok := r.denUser(t, "sam")
	if !ok {
		t.Fatal("Den has no user sam")
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, address+"/Users/"+sam.ID+"?ApiKey="+denKey, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := r.den.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
}

// presses starts each event as its caller does: an Admin or the Member
// through the API, identity's Account delete, and the Controller's start
// for Unmake.
var presses = map[access.Event]func(t *testing.T, r walkRig, way access.Way){
	access.Offer: func(t *testing.T, r walkRig, way access.Way) {
		if status, body := r.offer(t, r.member.ID, r.servers[way], way == access.WayPlexTV); status != http.StatusOK {
			t.Fatalf("offer = %d %s", status, body)
		}
	},
	access.Remove: func(t *testing.T, r walkRig, way access.Way) {
		row, _ := r.row(t, way)
		if status, body := r.o.CallSignedIn(t, http.MethodDelete, servers+"/access/"+row.ID, ""); status != http.StatusOK {
			t.Fatalf("remove = %d %s", status, body)
		}
	},
	access.RetryRemove: func(t *testing.T, r walkRig, way access.Way) {
		row, _ := r.row(t, way)
		if status, body := r.o.CallSignedIn(t, http.MethodDelete, servers+"/access/"+row.ID, ""); status != http.StatusOK {
			t.Fatalf("try again = %d %s", status, body)
		}
	},
	access.Leave: func(t *testing.T, r walkRig, _ access.Way) {
		if err := r.w.Identity.DeleteAccount(t.Context(), identity.Admin{}, r.member.ID); err != nil {
			t.Fatal(err)
		}
	},
	access.MakeAccount: func(t *testing.T, r walkRig, way access.Way) {
		row, _ := r.row(t, way)
		r.o.CallAs(t, http.MethodPost, "/api/v1/account/media-servers/"+row.ID+"/account", `{"username":"sam","password":"sam-password"}`, r.member.Session)
	},
	access.Link: func(t *testing.T, r walkRig, way access.Way) {
		row, _ := r.row(t, way)
		kit, _ := r.denUser(t, "kit")
		users := decode[struct{ Users []struct{ ID, Name string } }](t, r.expectOK(t, servers+"/"+r.servers[way]+"/users"))
		if !slices.ContainsFunc(users.Users, func(u struct{ ID, Name string }) bool { return u.Name == "kit" }) {
			t.Fatalf("Den's linkable users are %+v, want kit", users.Users)
		}
		kitID := users.Users[slices.IndexFunc(users.Users, func(u struct{ ID, Name string }) bool { return u.Name == "kit" })].ID
		if kit.ID != "" && kit.ID != kitID {
			t.Fatalf("kit is %s on Den and %s in the list", kit.ID, kitID)
		}
		if status, body := r.o.CallSignedIn(t, http.MethodPost, servers+"/access/"+row.ID+"/link", `{"userId":"`+kitID+`"}`); status != http.StatusOK {
			t.Fatalf("link = %d %s", status, body)
		}
	},
	access.Unmake: func(t *testing.T, r walkRig, _ access.Way) { r.w.Restart(t) },
}

// step is one press and the Table entries it walks, in order. fail is
// Den's routes whose next call answers 500, and gone deletes Sam's user on
// Den first.
type step struct {
	on    access.Event
	fail  []string
	gone  bool
	walks []access.Transition
}

func onUsers(from access.State, on access.Event, when access.Outcome) access.Transition {
	return access.Must(from, on, when, access.WayUsers)
}

func onPlex(from access.State, on access.Event, when access.Outcome) access.Transition {
	return access.Must(from, on, when, access.WayPlexTV)
}

func walk(ts ...access.Transition) step { return step{walks: ts} }

func failing(s step, routes ...string) step { s.fail = routes; return s }

// samOnDen is the user Sam's access leaves on Den.
type samOnDen uint8

const (
	unchecked samOnDen = iota
	noUser
	enabled
	disabled
)

var (
	offerDen  = walk(onUsers(access.None, access.Offer, access.Done))
	makeOnDen = walk(
		onUsers(access.Offered, access.MakeAccount, access.Done), onUsers(access.Making, access.Created, access.Done), onUsers(access.Making, access.Finish, access.Done),
	)
	// leftMaking is a make whose policy and cleanup both failed.
	leftMaking = failing(walk(
		onUsers(access.Offered, access.MakeAccount, access.Done), onUsers(access.Making, access.Created, access.Done), onUsers(access.Making, access.Unmake, access.Failed),
	), "user-policy", "delete-user")
	removeFailing = failing(walk(onUsers(access.Active, access.Remove, access.Failed)), "user")
)

// TestTheTableWalk drives Table entries at the API seam: each case
// presses its steps in order, and after each reads the row back in the
// To of the last entry the step walks, with the problem recorded only
// after a failed call. An offer asks for Plex Home on Plex. Den's user
// for Sam is then checked as Den holds it.
func TestTheTableWalk(t *testing.T) {
	for _, c := range []struct {
		name  string
		way   access.Way
		steps []step
		sam   samOnDen
	}{
		{"offer Plex", access.WayPlexTV, []step{walk(onPlex(access.None, access.Offer, access.NoIdentity))}, unchecked},
		{"offer Plex again", access.WayPlexTV, []step{walk(onPlex(access.None, access.Offer, access.NoIdentity)), walk(onPlex(access.Offered, access.Offer, access.Done))}, unchecked},
		{"remove an offered Plex", access.WayPlexTV, []step{walk(onPlex(access.None, access.Offer, access.NoIdentity)), walk(onPlex(access.Offered, access.Remove, access.Done))}, unchecked},
		{"leave an offered Plex", access.WayPlexTV, []step{walk(onPlex(access.None, access.Offer, access.NoIdentity)), walk(onPlex(access.Offered, access.Leave, access.Done))}, unchecked},

		{"offer Den", access.WayUsers, []step{offerDen}, noUser},
		{"remove an offered Den", access.WayUsers, []step{offerDen, walk(onUsers(access.Offered, access.Remove, access.Done))}, noUser},
		{"leave an offered Den", access.WayUsers, []step{offerDen, walk(onUsers(access.Offered, access.Leave, access.Done))}, noUser},
		{"make", access.WayUsers, []step{offerDen, makeOnDen}, enabled},
		{"a make whose create fails", access.WayUsers, []step{offerDen, failing(walk(
			onUsers(access.Offered, access.MakeAccount, access.Done), onUsers(access.Making, access.Unmake, access.Done),
		), "create-user")}, noUser},
		{"a make whose password fails", access.WayUsers, []step{offerDen, failing(walk(
			onUsers(access.Offered, access.MakeAccount, access.Done), onUsers(access.Making, access.Created, access.Done), onUsers(access.Making, access.Unmake, access.Done),
		), "user-password")}, noUser},
		{"the start unmakes a make left making", access.WayUsers, []step{offerDen, leftMaking, walk(onUsers(access.Making, access.Unmake, access.Done))}, noUser},
		{"a retried make unmakes first", access.WayUsers, []step{offerDen, leftMaking, {on: access.MakeAccount, walks: []access.Transition{
			onUsers(access.Making, access.Unmake, access.Done), onUsers(access.Offered, access.MakeAccount, access.Done),
			onUsers(access.Making, access.Created, access.Done), onUsers(access.Making, access.Finish, access.Done),
		}}}, enabled},
		{"leave a making Den", access.WayUsers, []step{offerDen, leftMaking, walk(onUsers(access.Making, access.Leave, access.Done))}, noUser},
		{"leave a making Den unreached, then start", access.WayUsers, []step{
			offerDen, leftMaking, failing(walk(onUsers(access.Making, access.Leave, access.Failed)), "delete-user"),
			walk(onUsers(access.Making, access.Unmake, access.Left)),
		}, noUser},
		{"link", access.WayUsers, []step{offerDen, walk(onUsers(access.Offered, access.Link, access.Done))}, noUser},
		{"remove", access.WayUsers, []step{offerDen, makeOnDen, walk(onUsers(access.Active, access.Remove, access.Done))}, disabled},
		{"remove failing, then try again", access.WayUsers, []step{offerDen, makeOnDen, removeFailing, walk(onUsers(access.Removing, access.RetryRemove, access.Done))}, disabled},
		{"try again failing", access.WayUsers, []step{offerDen, makeOnDen, removeFailing, failing(walk(onUsers(access.Removing, access.RetryRemove, access.Failed)), "user")}, enabled},
		{"offer again", access.WayUsers, []step{offerDen, makeOnDen, walk(onUsers(access.Active, access.Remove, access.Done)), walk(onUsers(access.Removed, access.Offer, access.Done))}, enabled},
		{"offer again failing", access.WayUsers, []step{offerDen, makeOnDen, walk(onUsers(access.Active, access.Remove, access.Done)), failing(walk(onUsers(access.Removed, access.Offer, access.Failed)), "user")}, disabled},
		{"offer again once Den deleted the user", access.WayUsers, []step{
			offerDen, makeOnDen, walk(onUsers(access.Active, access.Remove, access.Done)), {gone: true, walks: []access.Transition{onUsers(access.Removed, access.Offer, access.Gone)}},
		}, noUser},
		{"leave a removed Den", access.WayUsers, []step{offerDen, makeOnDen, walk(onUsers(access.Active, access.Remove, access.Done)), walk(onUsers(access.Removed, access.Leave, access.Done))}, disabled},
		{"leave an active Den", access.WayUsers, []step{offerDen, makeOnDen, walk(onUsers(access.Active, access.Leave, access.Done))}, disabled},
		{"leave an active Den failing, then try again", access.WayUsers, []step{
			offerDen, makeOnDen, failing(walk(onUsers(access.Active, access.Leave, access.Failed)), "user"), walk(onUsers(access.Removing, access.RetryRemove, access.Left)),
		}, disabled},
		{"leave a removing Den", access.WayUsers, []step{offerDen, makeOnDen, removeFailing, walk(onUsers(access.Removing, access.Leave, access.Done))}, disabled},
		{"leave a removing Den failing", access.WayUsers, []step{offerDen, makeOnDen, removeFailing, failing(walk(onUsers(access.Removing, access.Leave, access.Failed)), "user")}, enabled},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := newWalkRig(t)
				for _, s := range c.steps {
					r.press(t, c.way, s)
				}
				r.expectSam(t, c.sam)
			})
		})
	}
}

func (r walkRig) press(t *testing.T, way access.Way, s step) {
	t.Helper()
	on := s.on
	if on == "" {
		on = s.walks[0].On
	}
	if s.gone {
		r.deleteOnDen(t)
	}
	for _, route := range s.fail {
		r.den.Cue(t, "jellyfin-fail-next", jellyfin.FailNext{Route: route})
	}
	presses[on](t, r, way)
	synctest.Wait()
	last := s.walks[len(s.walks)-1]
	row, ok := r.row(t, way)
	switch {
	case last.To == access.None && ok:
		t.Fatalf("after %s the row is %s, want it deleted", on, row.State)
	case last.To == access.None:
	case !ok || row.State != last.To:
		t.Fatalf("after %s the row is %+v, want %s", on, row, last.To)
	case (row.Problem != nil) != (len(s.fail) > 0 && last.Mark&access.KeepProblem != 0):
		t.Errorf("after %s the row's problem is %+v", on, row.Problem)
	case way == access.WayPlexTV && (row.Plex == nil || !row.Plex.PlexHome):
		t.Errorf("the Plex row is %+v, want it offered as Plex Home", row.Plex)
	}
}

// expectSam checks Den's user for Sam, and that no user of an unfinished
// make is left.
func (r walkRig) expectSam(t *testing.T, want samOnDen) {
	t.Helper()
	if want == unchecked {
		return
	}
	for _, u := range r.denUsers(t) {
		if strings.HasPrefix(u.Name, "slipstream-") {
			t.Errorf("Den kept the user %s of an unfinished make", u.Name)
		}
	}
	sam, ok := r.denUser(t, "sam")
	switch {
	case want == noUser && ok:
		t.Errorf("Den has the user sam %+v, want none", sam)
	case want == noUser:
	case !ok:
		t.Fatalf("Den has no user sam")
	case sam.Disabled != (want == disabled) || sam.Administrator || sam.AllFolders || !slices.Equal(sam.Folders, []string{"1"}):
		t.Errorf("Den's user sam is %+v, want disabled %t with only the library for Members", sam, want == disabled)
	}
}
