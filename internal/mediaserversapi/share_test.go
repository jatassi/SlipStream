package mediaserversapi_test

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	fakeplextv "github.com/jatassi/slipstream/fakes/plextv"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaservers/access"
	"github.com/jatassi/slipstream/internal/providers/plextv"
	"github.com/jatassi/slipstream/internal/storage"
)

// checkAgain is how long Check again answers a row without asking
// plex.tv.
const checkAgain = 10 * time.Second

// plexHolds is what plex.tv holds for one account on the Admin's account:
// its shares of the Living room, whether invites/requested lists it, and
// whether it is in the Plex Home.
type plexHolds struct {
	shares        int
	invited, home bool
}

// holdsFor reads plex.tv's listings with the Admin's token.
func (r plexRig) holdsFor(t *testing.T, username string) plexHolds {
	t.Helper()
	token := r.w.PlexTVToken(t, r.plex, "").Reveal()
	get := func(path string) []byte {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://plextv.fake.localhost"+path, http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-Plex-Token", token)
		req.Header.Set("Accept", "application/json")
		resp, err := r.plex.TV.Client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		b, err := io.ReadAll(resp.Body)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s = %d %s, %v", path, resp.StatusCode, b, err)
		}
		return b
	}
	var h plexHolds
	for _, listing := range []string{"accepted", "pending"} {
		var shares []struct {
			Machine string `json:"machineIdentifier"`
			Invited struct {
				Username string `json:"username"`
			} `json:"invited"`
		}
		if err := json.Unmarshal(get("/api/v2/shared_servers/owned/"+listing), &shares); err != nil {
			t.Fatal(err)
		}
		for _, s := range shares {
			if s.Machine == plexID && s.Invited.Username == username {
				h.shares++
			}
		}
	}
	type named struct {
		Username string `xml:"username,attr"`
	}
	var invites struct {
		Invites []named `xml:"Invite"`
	}
	var home struct {
		Users []named `xml:"User"`
	}
	if err := xml.Unmarshal(get("/api/invites/requested"), &invites); err != nil {
		t.Fatal(err)
	}
	if err := xml.Unmarshal(get("/api/home/users"), &home); err != nil {
		t.Fatal(err)
	}
	h.invited = slices.Contains(invites.Invites, named{username})
	h.home = slices.Contains(home.Users, named{username})
	return h
}

// waitingPlexRow offers the Plex server to Sam, as Plex Home when home,
// signs Sam in to Plex and answers the row, waiting for the share.
func (r walkRig) waitingPlexRow(t *testing.T, home bool) string {
	t.Helper()
	if status, body := r.offer(t, r.member.ID, r.servers[access.WayPlexTV], home); status != http.StatusOK {
		t.Fatalf("offer = %d %s", status, body)
	}
	if state := r.memberSignIn(t, r.member, samOnPlex.Username); state.State != mediaservers.PinSignedIn {
		t.Fatalf("Sam's Plex sign-in is %+v", state)
	}
	row, _ := r.row(t, access.WayPlexTV)
	if row.State != access.Waiting {
		t.Fatalf("after Sam's sign-in the row is %s", row.State)
	}
	return row.ID
}

func (r walkRig) shareAs(t *testing.T, row, as string) (status int, body string) {
	t.Helper()
	return r.o.CallSignedIn(t, http.MethodPost, servers+"/access/"+row+"/share", `{"as":"`+as+`"}`)
}

// sharedOut is the fields of a share's answer the tests read: the row and
// the email's outcome, or the problem's code.
type sharedOut struct {
	Code string `json:"code"`
	Row  struct {
		State access.State `json:"state"`
	} `json:"row"`
	Mail *struct {
		Outcome string `json:"outcome"`
	} `json:"mail"`
}

// TestAShareEmailsTheMember shares the Plex server with Sam: a share
// SlipStream accepts, and one whose accept plex.tv refused, each email Sam
// and answer the email's outcome; a share plex.tv refuses answers
// plex_refused with plex.tv's reason and emails nobody.
func TestAShareEmailsTheMember(t *testing.T) {
	for _, c := range []struct {
		name  string
		cue   string
		state access.State
		code  string
	}{
		{"accepted", "", access.Active, ""},
		{"accept refused", fakeplextv.RefuseNextAccept.Name, access.Shared, ""},
		{"share refused", fakeplextv.RefuseNextShare.Name, access.Waiting, "plex_refused"},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := newWalkRig(t)
				row := r.waitingPlexRow(t, false)
				sent := len(r.w.Mail.Sent())
				if c.cue != "" {
					r.plex.TV.Cue(t, c.cue, map[string]string{})
				}
				status, body := r.shareAs(t, row, "friend")
				expectSharedAnswer(t, status, body, r.w.Mail.Sent()[sent:], c.state, c.code)
				if got, _ := r.row(t, access.WayPlexTV); got.State != c.state {
					t.Errorf("the row is %s, want %s", got.State, c.state)
				}
			})
		})
	}
}

// expectSharedAnswer checks a share's answer and the emails it sent: the
// problem code with plex.tv's reason and no email when code is set, else
// the row in state and one email to Sam, sent.
func expectSharedAnswer(t *testing.T, status int, body string, mails []mail.Message, state access.State, code string) {
	t.Helper()
	out := decode[sharedOut](t, body)
	if code != "" {
		if status != http.StatusUnprocessableEntity || out.Code != code || !strings.Contains(body, "couldn't be shared") || len(mails) != 0 {
			t.Errorf("share = %d %s with %d emails, want %s with plex.tv's reason and none", status, body, len(mails), code)
		}
		return
	}
	if status != http.StatusOK || out.Row.State != state || out.Mail == nil || out.Mail.Outcome != string(mail.Sent) ||
		len(mails) != 1 || mails[0].To.Email != "sam@example.test" {
		t.Errorf("share = %d %s with emails %+v, want %s and one email to Sam", status, body, mails, state)
	}
}

// TestTwoSharesAtOnceMakeOneShare has two Admins press Share on Plex
// together: both succeed and plex.tv holds one share.
func TestTwoSharesAtOnceMakeOneShare(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newWalkRig(t)
		row := r.waitingPlexRow(t, false)
		var wg sync.WaitGroup
		statuses := make([]int, 2)
		for i := range statuses {
			wg.Go(func() { statuses[i], _ = r.shareAs(t, row, "friend") })
		}
		wg.Wait()
		if statuses[0] != http.StatusOK || statuses[1] != http.StatusOK {
			t.Errorf("the two shares answered %v", statuses)
		}
		if holds := r.holdsFor(t, samOnPlex.Username); holds.shares != 1 {
			t.Errorf("plex.tv holds %+v for Sam, want one share", holds)
		}
	})
}

// TestAShareCutShortByARestartLeavesOneShare stops the Controller after
// plex.tv made Sam's share and before the row recorded it: the next press
// finds that share, so Share on Plex ends with one share and Remove access
// with none and no invite.
func TestAShareCutShortByARestartLeavesOneShare(t *testing.T) {
	for _, c := range []struct {
		name  string
		press func(t *testing.T, r walkRig, row string) (int, string)
		want  plexHolds
	}{
		{"share again", func(t *testing.T, r walkRig, row string) (int, string) { return r.shareAs(t, row, "friend") }, plexHolds{shares: 1}},
		{"remove", func(t *testing.T, r walkRig, row string) (int, string) {
			return r.o.CallSignedIn(t, http.MethodDelete, servers+"/access/"+row, "")
		}, plexHolds{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := newWalkRig(t)
				row := r.waitingPlexRow(t, false)
				r.stopMidShare(t, row)
				if got, _ := r.row(t, access.WayPlexTV); got.State != access.Waiting {
					t.Fatalf("after the restart the row is %s", got.State)
				}
				if status, body := c.press(t, r, row); status != http.StatusOK {
					t.Fatalf("press = %d %s", status, body)
				}
				if holds := r.holdsFor(t, samOnPlex.Username); holds != c.want {
					t.Errorf("plex.tv holds %+v for Sam, want %+v", holds, c.want)
				}
			})
		})
	}
}

// stopMidShare starts the share of row with Sam's accept hung, waits for
// it to stop there and ends it as a Controller that stops does, then
// starts the Controller again.
func (r walkRig) stopMidShare(t *testing.T, row string) {
	t.Helper()
	if _, err := r.plex.TV.Faults.Set(fault.Rule{ID: "hang", Target: fault.Target(fakeplextv.Fake.Name), Route: "pending-received", Fault: fault.Hang{}}); err != nil {
		t.Fatal(err)
	}
	id, err := storage.ParseUUID(row)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error)
	go func() {
		_, err := r.w.Service.Share(ctx, identity.Admin{}, mediaservers.AccessID{UUID: id}, mediaservers.InviteFriend)
		done <- err
	}()
	synctest.Wait()
	cancel()
	if err := <-done; err == nil {
		t.Fatal("the share ended before its hung accept")
	}
	r.plex.TV.Faults.Clear("hang")
	r.w.Restart(t)
}

// TestWhatTheAdminMayShareFollowsThePlexRun lists a waiting row's actions
// under each fact the recorded Plex run could settle and each Plex Pass of
// the Admin: a share the row doesn't offer answers access_state when
// pressed anyway, and Mark as added stays.
func TestWhatTheAdminMayShareFollowsThePlexRun(t *testing.T) {
	for _, c := range []struct {
		name     string
		facts    plextv.Facts
		plexPass bool
		offered  []access.Event
	}{
		{"as recorded", plextv.Recorded, true, []access.Event{access.ShareFriend, access.ShareHome, access.MarkAdded}},
		{"as recorded, no Plex Pass", plextv.Recorded, false, []access.Event{access.ShareFriend, access.MarkAdded}},
		{"Plex Home without Plex Pass", plextv.Facts{AdminShares: true, AdminInvitesHome: true}, false,
			[]access.Event{access.ShareFriend, access.ShareHome, access.MarkAdded}},
		{"the Admin's sign-in can't share", plextv.Facts{}, true, []access.Event{access.MarkAdded}},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := newWalkRig(t)
				r.w.UsePlexFacts(t, c.facts)
				if !c.plexPass {
					r.plex.TV.Cue(t, fakeplextv.PlexPass.Name, map[string]bool{"active": false})
					if err := r.renew(t); err != nil {
						t.Fatal(err)
					}
				}
				id := r.waitingPlexRow(t, false)
				r.expectOffered(t, id, c.offered)
			})
		})
	}
}

// expectOffered checks that row id offers the Admin each share in offered,
// and that a share it doesn't offer answers access_state.
func (r walkRig) expectOffered(t *testing.T, id string, offered []access.Event) {
	t.Helper()
	rows := r.accessOf(t, r.member.ID).Rows
	row := rows[slices.IndexFunc(rows, func(a accessOut) bool { return a.ID == id })]
	for _, on := range []access.Event{access.ShareFriend, access.ShareHome, access.MarkAdded} {
		if slices.Contains(row.Actions, on) != slices.Contains(offered, on) {
			t.Errorf("the row's actions are %v, want %v", row.Actions, offered)
		}
	}
	for as, on := range map[string]access.Event{"friend": access.ShareFriend, "home": access.ShareHome} {
		if slices.Contains(offered, on) {
			continue
		}
		if status, body := r.shareAs(t, id, as); status != http.StatusConflict || !strings.Contains(body, "access_state") {
			t.Errorf("a share as %s = %d %s, want access_state", as, status, body)
		}
	}
}

// TestAFriendShareOnAPlexHomeOfferLeavesTheHome shares as a friend a row
// offered as Plex Home, with Sam already in the Admin's Plex Home: Remove
// access then deletes the share and leaves Sam in the Home.
func TestAFriendShareOnAPlexHomeOfferLeavesTheHome(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newWalkRig(t)
		row := r.waitingPlexRow(t, true)
		r.w.HomeOnPlexTV(t, r.plex, samOnPlex.Username)
		if status, body := r.shareAs(t, row, "friend"); status != http.StatusOK {
			t.Fatalf("share = %d %s", status, body)
		}
		if status, body := r.o.CallSignedIn(t, http.MethodDelete, servers+"/access/"+row, ""); status != http.StatusOK {
			t.Fatalf("remove = %d %s", status, body)
		}
		if holds := r.holdsFor(t, samOnPlex.Username); holds != (plexHolds{home: true}) {
			t.Errorf("plex.tv holds %+v for Sam, want only the Plex Home place", holds)
		}
	})
}

// TestAnAcceptedShareIsFinishedByEachChecker leaves Sam's share
// unaccepted, then has each checker look again once Check again's pause
// passed: each accepts it with Sam's token and the row is active.
func TestAnAcceptedShareIsFinishedByEachChecker(t *testing.T) {
	for _, c := range []struct {
		name  string
		check func(t *testing.T, r walkRig, row string)
	}{
		{"the Admin", func(t *testing.T, r walkRig, row string) {
			r.o.CallSignedIn(t, http.MethodPost, servers+"/access/"+row+"/check", "")
		}},
		{"the Member", func(t *testing.T, r walkRig, row string) {
			r.o.CallAs(t, http.MethodPost, mine+"/"+row+"/check", "", r.member.Session)
		}},
		{"the Plex sign-in task", func(t *testing.T, r walkRig, _ string) {
			if err := r.renew(t); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := newWalkRig(t)
				row := r.waitingPlexRow(t, false)
				r.plex.TV.Cue(t, fakeplextv.RefuseNextAccept.Name, map[string]string{})
				if status, body := r.shareAs(t, row, "friend"); status != http.StatusOK {
					t.Fatalf("share = %d %s", status, body)
				}
				time.Sleep(checkAgain)
				c.check(t, r, row)
				if got, _ := r.row(t, access.WayPlexTV); got.State != access.Active {
					t.Errorf("after the check the row is %s, want active", got.State)
				}
			})
		})
	}
}
