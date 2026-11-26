package mediaserversapi_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/jellyfin"
	fakeplextv "github.com/jatassi/slipstream/fakes/plextv"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaservers/access"
	"github.com/jatassi/slipstream/internal/mediaservers/mstest"
	"github.com/jatassi/slipstream/internal/providers/plextv"
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
	base := newRig(t)
	r := plexRig{rig: base, plex: base.w.Plex(t, sharedRoom, fakeplextv.Account{Username: "plex-admin", PlexPass: true}, samOnPlex)}
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

// samOnPlex is the Plex account Sam signs in with.
var samOnPlex = fakeplextv.Account{Username: "sam", Email: "sam@plex.example"}

// walkRow is the fields of an access row the walk reads.
type walkRow struct {
	ID     string       `json:"id"`
	State  access.State `json:"state"`
	Server struct {
		ID string `json:"id"`
	} `json:"server"`
	Member struct {
		AccountID string `json:"accountId"`
	} `json:"member"`
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
	access.PlexSignIn: func(t *testing.T, r walkRig, _ access.Way) {
		if state := r.memberSignIn(t, r.member, samOnPlex.Username); state.State != mediaservers.PinSignedIn {
			t.Fatalf("Sam's Plex sign-in is %+v", state)
		}
	},
	access.ShareFriend: func(t *testing.T, r walkRig, way access.Way) { r.onRow(t, way, "/share", `{"as":"friend"}`) },
	access.ShareHome:   func(t *testing.T, r walkRig, way access.Way) { r.onRow(t, way, "/share", `{"as":"home"}`) },
	access.MarkAdded:   func(t *testing.T, r walkRig, way access.Way) { r.onRow(t, way, "/mark-added", "") },
	access.Check:       func(t *testing.T, r walkRig, way access.Way) { r.onRow(t, way, "/check", "") },
	access.AnotherPlex: func(t *testing.T, r walkRig, _ access.Way) {
		if status, body := r.o.CallAs(t, http.MethodDelete, "/api/v1/account/media-servers/plex", "", r.member.Session); status != http.StatusNoContent {
			t.Fatalf("use another Plex account = %d %s", status, body)
		}
	},
}

// onRow posts to the Admin's operation at suffix on way's row. Its answer
// is the row the walk reads back, or a refusal the walk expects.
func (r walkRig) onRow(t *testing.T, way access.Way, suffix, body string) {
	t.Helper()
	row, _ := r.row(t, way)
	r.o.CallSignedIn(t, http.MethodPost, servers+"/access/"+row.ID+suffix, body)
}

// sharesSam shares the Plex server with Sam's Plex account in Plex.
func sharesSam(t *testing.T, r walkRig) plextv.ShareID {
	t.Helper()
	return r.w.ShareOnPlexTV(t, r.plex, plexID, samOnPlex.Username)
}

// sharedSam is the Admin's share to Sam left for Sam to accept.
func sharedSam(t *testing.T, r walkRig) {
	t.Helper()
	row, _ := r.row(t, access.WayPlexTV)
	r.w.Shared(t, row.ID, sharesSam(t, r))
}

// step is one press and the Table entries it walks, in order. fail is
// Den's routes whose next call answers 500, plexDown has plex.tv answer
// 503 to every call through the press, and gone deletes Sam's user on Den
// first.
type step struct {
	on       access.Event
	fail     []string
	plexDown bool
	gone     bool
	walks    []access.Transition
	// before runs first, for a state Plex holds.
	before func(t *testing.T, r walkRig)
}

func onUsers(from access.State, on access.Event, when access.Outcome) access.Transition {
	return access.Must(from, on, when, access.WayUsers)
}

func onPlex(from access.State, on access.Event, when access.Outcome) access.Transition {
	return access.Must(from, on, when, access.WayPlexTV)
}

func walk(ts ...access.Transition) step { return step{walks: ts} }

func failing(s step, routes ...string) step { s.fail = routes; return s }

func after(before func(t *testing.T, r walkRig), s step) step { s.before = before; return s }

func plexDown(s step) step { s.plexDown = true; return s }

// failed is whether the step's last entry records a call that failed.
func (s step) failed() bool {
	last := s.walks[len(s.walks)-1]
	return last.Mark&access.KeepProblem != 0 && (len(s.fail) > 0 || s.plexDown || last.When == access.Refused)
}

// checkLater waits out Check again's pacing, then runs each of and.
func checkLater(and ...func(t *testing.T, r walkRig)) func(t *testing.T, r walkRig) {
	return func(t *testing.T, r walkRig) {
		time.Sleep(checkAgain)
		for _, f := range and {
			f(t, r)
		}
	}
}

// cue has plex.tv play cue at the next call it names.
func cue(name string) func(t *testing.T, r walkRig) {
	return func(t *testing.T, r walkRig) { r.plex.TV.Cue(t, name, map[string]string{}) }
}

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
	offerPlex     = walk(onPlex(access.None, access.Offer, access.NoIdentity))
	signInPlex    = walk(onPlex(access.Offered, access.PlexSignIn, access.NotThere))
	// waitingPlex is Sam signed in to Plex, waiting for the Admin's share.
	waitingPlex = []step{offerPlex, signInPlex}
	shareFriend = walk(onPlex(access.Waiting, access.ShareFriend, access.Accepted))
	shareHome   = walk(onPlex(access.Waiting, access.ShareHome, access.Accepted))
	// shareUnaccepted is a share whose accept plex.tv refused.
	shareUnaccepted = after(cue(fakeplextv.RefuseNextAccept.Name), walk(onPlex(access.Waiting, access.ShareFriend, access.NotAccepted)))
	removePlexDown  = plexDown(walk(onPlex(access.Active, access.Remove, access.Failed)))
)

// then is steps followed by more.
func then(steps []step, more ...step) []step { return append(slices.Clone(steps), more...) }

// TestTheTableWalk drives every Table entry at the API seam: each case
// presses its steps in order, and after each reads the row back in the
// To of the last entry the step walks, with the problem recorded only
// after a failed call and no Account once a Leave couldn't finish. An
// offer asks for Plex Home on Plex. Den's user for Sam is then checked as
// Den holds it, and plex.tv keeps no share, invite or Plex Home place of
// Sam's once the Plex row is gone.
func TestTheTableWalk(t *testing.T) {
	walked := map[access.Transition]bool{}
	for _, c := range walkCases {
		for _, s := range c.steps {
			for _, tr := range s.walks {
				walked[tr] = true
			}
		}
		t.Run(c.name, func(t *testing.T) { synctest.Test(t, c.run) })
	}
	for _, tr := range access.Table {
		if !walked[tr] {
			t.Errorf("no case walks %+v", tr)
		}
	}
}

// walkCase presses its steps on way's row, then checks Den's user for Sam.
type walkCase struct {
	name  string
	way   access.Way
	steps []step
	sam   samOnDen
}

var walkCases = []walkCase{
	{"offer Plex", access.WayPlexTV, []step{walk(onPlex(access.None, access.Offer, access.NoIdentity))}, unchecked},
	{"offer Plex again", access.WayPlexTV, []step{walk(onPlex(access.None, access.Offer, access.NoIdentity)), walk(onPlex(access.Offered, access.Offer, access.Done))}, unchecked},
	{"remove an offered Plex", access.WayPlexTV, []step{walk(onPlex(access.None, access.Offer, access.NoIdentity)), walk(onPlex(access.Offered, access.Remove, access.Done))}, unchecked},
	{"leave an offered Plex", access.WayPlexTV, []step{walk(onPlex(access.None, access.Offer, access.NoIdentity)), walk(onPlex(access.Offered, access.Leave, access.Done))}, unchecked},

	{"sign in to an offered Plex", access.WayPlexTV, []step{offerPlex, signInPlex}, unchecked},
	{"sign in where Plex already shares", access.WayPlexTV, []step{offerPlex, after(func(t *testing.T, r walkRig) { sharesSam(t, r) },
		walk(onPlex(access.Offered, access.PlexSignIn, access.AlreadyThere)))}, unchecked},
	{"sign in again", access.WayPlexTV, []step{offerPlex, signInPlex, walk(onPlex(access.Waiting, access.PlexSignIn, access.NotThere))}, unchecked},
	{"sign in again once Plex shares", access.WayPlexTV, []step{offerPlex, signInPlex, after(func(t *testing.T, r walkRig) { sharesSam(t, r) },
		walk(onPlex(access.Waiting, access.PlexSignIn, access.AlreadyThere)))}, unchecked},
	{"sign in again accepts the share", access.WayPlexTV, []step{offerPlex, signInPlex, after(sharedSam,
		walk(onPlex(access.Shared, access.PlexSignIn, access.Accepted)))}, unchecked},
	{"sign in again with the accept refused", access.WayPlexTV, []step{offerPlex, signInPlex, after(func(t *testing.T, r walkRig) {
		sharedSam(t, r)
		r.plex.TV.Cue(t, fakeplextv.RefuseNextAccept.Name, map[string]string{})
	}, walk(onPlex(access.Shared, access.PlexSignIn, access.NotAccepted)))}, unchecked},
	{"use another Plex account", access.WayPlexTV, []step{offerPlex, signInPlex, walk(onPlex(access.Waiting, access.AnotherPlex, access.Done))}, unchecked},
	{"offer a waiting Plex again", access.WayPlexTV, then(waitingPlex, walk(onPlex(access.Waiting, access.Offer, access.Done))), unchecked},
	{"offer Plex once Sam signed in", access.WayPlexTV, then(waitingPlex, walk(onPlex(access.Waiting, access.Remove, access.Done)),
		walk(onPlex(access.None, access.Offer, access.IdentityHeld))), unchecked},

	{"share", access.WayPlexTV, then(waitingPlex, shareFriend), unchecked},
	{"share, accept refused", access.WayPlexTV, then(waitingPlex, shareUnaccepted), unchecked},
	{"share refused", access.WayPlexTV, then(waitingPlex, after(cue(fakeplextv.RefuseNextShare.Name),
		walk(onPlex(access.Waiting, access.ShareFriend, access.Refused)))), unchecked},
	{"add to Plex Home", access.WayPlexTV, then(waitingPlex, shareHome), unchecked},
	{"add to Plex Home, accept refused", access.WayPlexTV, then(waitingPlex, after(cue(fakeplextv.RefuseNextAccept.Name),
		walk(onPlex(access.Waiting, access.ShareHome, access.NotAccepted)))), unchecked},
	{"add to a full Plex Home", access.WayPlexTV, then(waitingPlex, after(cue(fakeplextv.HomeFull.Name),
		walk(onPlex(access.Waiting, access.ShareHome, access.Refused)))), unchecked},
	{"mark as added", access.WayPlexTV, then(waitingPlex, walk(onPlex(access.Waiting, access.MarkAdded, access.Done))), unchecked},
	{"check accepts the share", access.WayPlexTV, then(waitingPlex, shareUnaccepted, after(checkLater(),
		walk(onPlex(access.Shared, access.Check, access.Accepted)))), unchecked},
	{"check with the accept refused", access.WayPlexTV, then(waitingPlex, shareUnaccepted, after(checkLater(cue(fakeplextv.RefuseNextAccept.Name)),
		walk(onPlex(access.Shared, access.Check, access.NotAccepted)))), unchecked},

	{"remove a waiting Plex failing, then try again", access.WayPlexTV, then(waitingPlex, plexDown(walk(onPlex(access.Waiting, access.Remove, access.Failed))),
		walk(onPlex(access.Removing, access.RetryRemove, access.Done))), unchecked},
	{"remove a shared Plex", access.WayPlexTV, then(waitingPlex, shareUnaccepted, walk(onPlex(access.Shared, access.Remove, access.Done))), unchecked},
	{"remove a shared Plex failing", access.WayPlexTV, then(waitingPlex, shareUnaccepted, plexDown(walk(onPlex(access.Shared, access.Remove, access.Failed)))), unchecked},
	{"remove an active Plex", access.WayPlexTV, then(waitingPlex, shareFriend, walk(onPlex(access.Active, access.Remove, access.Done))), unchecked},
	{"remove a Plex Home share", access.WayPlexTV, then(waitingPlex, shareHome, walk(onPlex(access.Active, access.Remove, access.Done))), unchecked},
	{"try again failing on Plex", access.WayPlexTV, then(waitingPlex, shareFriend, removePlexDown,
		plexDown(walk(onPlex(access.Removing, access.RetryRemove, access.Failed)))), unchecked},

	{"leave a waiting Plex", access.WayPlexTV, then(waitingPlex, walk(onPlex(access.Waiting, access.Leave, access.Done))), unchecked},
	{"leave a waiting Plex failing", access.WayPlexTV, then(waitingPlex, plexDown(walk(onPlex(access.Waiting, access.Leave, access.Failed)))), unchecked},
	{"leave a shared Plex", access.WayPlexTV, then(waitingPlex, shareUnaccepted, walk(onPlex(access.Shared, access.Leave, access.Done))), unchecked},
	{"leave a shared Plex failing", access.WayPlexTV, then(waitingPlex, shareUnaccepted, plexDown(walk(onPlex(access.Shared, access.Leave, access.Failed)))), unchecked},
	{"leave an active Plex", access.WayPlexTV, then(waitingPlex, shareFriend, walk(onPlex(access.Active, access.Leave, access.Done))), unchecked},
	{"leave an active Plex failing, then try again", access.WayPlexTV, then(waitingPlex, shareFriend,
		plexDown(walk(onPlex(access.Active, access.Leave, access.Failed))), walk(onPlex(access.Removing, access.RetryRemove, access.Done))), unchecked},
	{"leave a removing Plex", access.WayPlexTV, then(waitingPlex, shareFriend, removePlexDown, walk(onPlex(access.Removing, access.Leave, access.Done))), unchecked},
	{"leave a removing Plex failing", access.WayPlexTV, then(waitingPlex, shareFriend, removePlexDown,
		plexDown(walk(onPlex(access.Removing, access.Leave, access.Failed)))), unchecked},

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
}

func (c walkCase) run(t *testing.T) {
	r := newWalkRig(t)
	home := true
	for _, s := range c.steps {
		for _, tr := range s.walks {
			home = (home || tr.Mark&access.SetPlexHome != 0) && tr.Mark&access.ClearPlexHome == 0
		}
		r.press(t, c.way, s, home)
	}
	r.expectSam(t, c.sam)
	if _, kept := r.row(t, c.way); c.way == access.WayPlexTV && !kept {
		if holds := r.holdsFor(t, samOnPlex.Username); holds != (plexHolds{}) {
			t.Errorf("plex.tv holds %+v for Sam once the row is gone", holds)
		}
	}
}

func (r walkRig) press(t *testing.T, way access.Way, s step, home bool) {
	t.Helper()
	on := s.on
	if on == "" {
		on = s.walks[0].On
	}
	if s.gone {
		r.deleteOnDen(t)
	}
	if s.before != nil {
		s.before(t, r)
	}
	for _, route := range s.fail {
		r.den.Cue(t, "jellyfin-fail-next", jellyfin.FailNext{Route: route})
	}
	if s.plexDown {
		if _, err := r.plex.TV.Faults.Set(fault.Rule{ID: "plex-down", Target: fault.Target(fakeplextv.Fake.Name), Fault: fault.Status{Code: http.StatusServiceUnavailable}}); err != nil {
			t.Fatal(err)
		}
	}
	presses[on](t, r, way)
	synctest.Wait()
	r.plex.TV.Faults.Clear("plex-down")
	last := s.walks[len(s.walks)-1]
	row, ok := r.row(t, way)
	switch {
	case last.To == access.None && ok:
		t.Fatalf("after %s the row is %s, want it deleted", on, row.State)
	case last.To == access.None:
	case !ok || row.State != last.To:
		t.Fatalf("after %s the row is %+v, want %s", on, row, last.To)
	case (row.Problem != nil) != s.failed():
		t.Errorf("after %s the row's problem is %+v", on, row.Problem)
	case (row.Member.AccountID == "") != (last.Mark&access.ClearAccount != 0):
		t.Errorf("after %s the row's Account is %q", on, row.Member.AccountID)
	case way == access.WayPlexTV && (row.Plex == nil || row.Plex.PlexHome != home):
		t.Errorf("the Plex row is %+v, want Plex Home %t", row.Plex, home)
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
