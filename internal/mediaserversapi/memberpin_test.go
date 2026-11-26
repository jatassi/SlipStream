package mediaserversapi_test

import (
	"net/http"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	fakeplex "github.com/jatassi/slipstream/fakes/plex"
	fakeplextv "github.com/jatassi/slipstream/fakes/plextv"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaservers/mstest"
)

const mine = "/api/v1/account/media-servers"

// sharedRoom is the Living room with a library to share.
var sharedRoom = func() fakeplex.Server {
	s := livingRoom
	s.Sections = []fakeplex.Section{{Key: "1", Title: "Shelves"}}
	return s
}()

// memberRig is the plexRig with Sam's and Bo's Plex accounts on plex.tv,
// the Plex server shown to Members, and Sam and Lee, two Members.
type memberRig struct {
	plexRig
	server   string
	sam, lee mstest.Member
}

func newMemberRig(t *testing.T) memberRig {
	t.Helper()
	base := newRig(t)
	r := plexRig{rig: base, plex: base.w.Plex(t, sharedRoom, fakeplextv.Account{Username: "plex-admin", PlexPass: true}, samOnPlex, fakeplextv.Account{Username: "bo"})}
	r.signIn(t)
	server := r.addPlex(t, r.w.Home.String()).ID
	r.show(t, server)
	return memberRig{plexRig: r, server: server, sam: r.w.Member(t, "sam"), lee: r.w.Member(t, "lee")}
}

func (r plexRig) startMemberPin(t *testing.T, m mstest.Member) mediaservers.MemberPin {
	t.Helper()
	status, body := r.o.CallAs(t, http.MethodPost, mine+"/plex/pin", "", m.Session)
	if status != http.StatusCreated {
		t.Fatalf("start a Member's Plex sign-in = %d %s", status, body)
	}
	return decode[mediaservers.MemberPin](t, body)
}

func (r plexRig) memberPin(t *testing.T, m mstest.Member, id int64) mediaservers.MemberPinState {
	t.Helper()
	status, body := r.o.CallAs(t, http.MethodGet, mine+"/plex/pin/"+jsonInt(id), "", m.Session)
	if status != http.StatusOK {
		t.Fatalf("read a Member's Plex sign-in = %d %s", status, body)
	}
	return decode[mediaservers.MemberPinState](t, body)
}

// memberSignIn runs a Member's Plex sign-in as the Watch step does: start,
// approve as username on plex.tv, and read the PIN once the Controller
// polled it.
func (r plexRig) memberSignIn(t *testing.T, m mstest.Member, username string) mediaservers.MemberPinState {
	t.Helper()
	pin := r.startMemberPin(t, m)
	r.plex.ApproveAs(t, pin.AuthURL, username)
	time.Sleep(time.Second)
	synctest.Wait()
	return r.memberPin(t, m, pin.ID)
}

// plexName is the Plex account m signed in as, "" for none.
func (r memberRig) plexName(t *testing.T, m mstest.Member) string {
	t.Helper()
	_, body := r.o.CallAs(t, http.MethodGet, mine, "", m.Session)
	a := decode[mediaservers.MyAccess](t, body)
	if a.Plex == nil {
		return ""
	}
	return a.Plex.Username
}

// TestARefusedPlexSignInKeepsNoToken signs Sam in with a Plex account
// SlipStream refuses: the PIN reads refused for its reason, plex.tv no
// longer holds the token it gave, and Sam keeps the Plex account held
// before.
func TestARefusedPlexSignInKeepsNoToken(t *testing.T) {
	for _, c := range []struct {
		name    string
		before  func(t *testing.T, r memberRig)
		as      string
		refusal mediaservers.PinRefusal
		held    string
	}{
		{"the Admin's account", func(*testing.T, memberRig) {}, "plex-admin", mediaservers.PlexAccountOwner, ""},
		{"an account Lee holds", func(t *testing.T, r memberRig) {
			if s := r.memberSignIn(t, r.lee, "sam"); s.State != mediaservers.PinSignedIn {
				t.Fatalf("Lee's sign-in is %+v", s)
			}
		}, "sam", mediaservers.PlexAccountTaken, ""},
		{"another account once shared", func(t *testing.T, r memberRig) {
			if status, body := r.offer(t, r.sam.ID, r.server, false); status != http.StatusOK {
				t.Fatalf("offer = %d %s", status, body)
			}
			r.w.ShareOnPlexTV(t, r.plex, plexID, "sam")
			if s := r.memberSignIn(t, r.sam, "sam"); s.State != mediaservers.PinSignedIn {
				t.Fatalf("Sam's first sign-in is %+v", s)
			}
		}, "bo", mediaservers.PlexAccountOther, "sam"},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := newMemberRig(t)
				c.before(t, r)
				pin := r.startMemberPin(t, r.sam)
				r.plex.ApproveAs(t, pin.AuthURL, c.as)
				time.Sleep(time.Second)
				synctest.Wait()
				if s := r.memberPin(t, r.sam, pin.ID); s != (mediaservers.MemberPinState{State: mediaservers.PinRefused, Refusal: c.refusal}) {
					t.Errorf("the sign-in is %+v, want refused %s", s, c.refusal)
				}
				if _, held := r.w.PlexTVUser(t, r.w.MemberPinToken(t, r.sam.ID, pin.ID)); held {
					t.Error("plex.tv still holds the refused sign-in's token")
				}
				if got := r.plexName(t, r.sam); got != c.held {
					t.Errorf("Sam is signed in to Plex as %q, want %q", got, c.held)
				}
			})
		})
	}
}

// TestTwoAccountsSignInToOnePlexAccountAtOnce approves Sam's and Lee's
// PINs as one Plex account before the Controller polls either, each
// offered the Plex server: one signs in and keeps the token for the share,
// the other is refused as taken.
func TestTwoAccountsSignInToOnePlexAccountAtOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newMemberRig(t)
		pins := map[string]mediaservers.MemberPin{}
		for name, m := range map[string]mstest.Member{"sam": r.sam, "lee": r.lee} {
			if status, body := r.offer(t, m.ID, r.server, false); status != http.StatusOK {
				t.Fatalf("offer to %s = %d %s", name, status, body)
			}
			pins[name] = r.startMemberPin(t, m)
			r.plex.ApproveAs(t, pins[name].AuthURL, "sam")
		}
		time.Sleep(time.Second)
		synctest.Wait()
		var states []mediaservers.MemberPinState
		for name, m := range map[string]mstest.Member{"sam": r.sam, "lee": r.lee} {
			s := r.memberPin(t, m, pins[name].ID)
			states = append(states, s)
			_, held := r.w.PlexTVUser(t, r.w.MemberPinToken(t, m.ID, pins[name].ID))
			if held != (s.State == mediaservers.PinSignedIn) {
				t.Errorf("%s's sign-in is %+v and plex.tv holds its token: %t", name, s, held)
			}
		}
		want := []mediaservers.MemberPinState{{State: mediaservers.PinSignedIn}, {State: mediaservers.PinRefused, Refusal: mediaservers.PlexAccountTaken}}
		if !slices.Contains(states, want[0]) || !slices.Contains(states, want[1]) {
			t.Errorf("the sign-ins are %+v, want %+v", states, want)
		}
	})
}

// TestAMemberReadsAnothersPlexSignInAsExpired reads Sam's waiting PIN as
// Lee.
func TestAMemberReadsAnothersPlexSignInAsExpired(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newMemberRig(t)
		pin := r.startMemberPin(t, r.sam)
		if s := r.memberPin(t, r.lee, pin.ID); s != (mediaservers.MemberPinState{State: mediaservers.PinExpired}) {
			t.Errorf("Lee reads Sam's sign-in as %+v", s)
		}
		if s := r.memberPin(t, r.sam, pin.ID); s != (mediaservers.MemberPinState{State: mediaservers.PinWaiting}) {
			t.Errorf("Sam reads their sign-in as %+v", s)
		}
	})
}

// TestAnAdminSignInAsAnotherPlexAccountAsksFirst signs the Admin in again
// as Bo while Sam's share waits on Sam's token: the sign-in waits as
// other_account and changes nothing until the replace, which signs
// SlipStream in as Bo, removes the Plex server and its access, and signs
// Sam's token out on plex.tv.
func TestAnAdminSignInAsAnotherPlexAccountAsksFirst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newMemberRig(t)
		if status, body := r.offer(t, r.sam.ID, r.server, false); status != http.StatusOK {
			t.Fatalf("offer = %d %s", status, body)
		}
		samPin := r.startMemberPin(t, r.sam)
		r.plex.ApproveAs(t, samPin.AuthURL, "sam")
		time.Sleep(time.Second)
		synctest.Wait()
		pin := decode[mediaservers.PlexPin](t, r.expect(t, http.MethodPost, servers+"/plex/pin", `{"forward":"https://slipstream.example.test/settings/system/media-servers"}`, http.StatusCreated))
		r.plex.ApproveAs(t, pin.AuthURL, "bo")
		time.Sleep(time.Second)
		if s := decode[mediaservers.PlexPinState](t, r.expect(t, http.MethodGet, servers+"/plex/pin/"+jsonInt(pin.ID), "", http.StatusOK)).State; s != "other_account" {
			t.Fatalf("the sign-in as Bo is %s", s)
		}
		if a := r.account(t); a.Name != "plex-admin" {
			t.Fatalf("before the replace SlipStream is signed in as %s", a.Name)
		}
		if status, body := r.o.CallSignedIn(t, http.MethodPost, servers+"/plex/pin/"+jsonInt(pin.ID)+"/replace", ""); status != http.StatusOK {
			t.Fatalf("replace = %d %s", status, body)
		}
		synctest.Wait()
		if a := r.account(t); a.Name != "bo" {
			t.Errorf("after the replace SlipStream is signed in as %s", a.Name)
		}
		if list := decode[struct{ Servers []serverOut }](t, r.expectOK(t, servers)).Servers; len(list) != 0 {
			t.Errorf("after the replace the servers are %+v", list)
		}
		if rows := decode[struct{ Rows []walkRow }](t, r.expectOK(t, servers+"/access")).Rows; len(rows) != 0 {
			t.Errorf("after the replace the access rows are %+v", rows)
		}
		if _, held := r.w.PlexTVUser(t, r.w.MemberPinToken(t, r.sam.ID, samPin.ID)); held {
			t.Error("plex.tv still holds Sam's token")
		}
	})
}
