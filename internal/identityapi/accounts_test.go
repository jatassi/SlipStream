package identityapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/storage"
)

var (
	lastAdmin       = problem(409, "last_admin", "SlipStream needs at least one Admin who is turned on")
	accountDisabled = problem(403, "account_disabled", "This Account is turned off. Ask an Admin to turn it back on.")
	accountNotFound = problem(404, "account_not_found", "No Account has that id")
	sessionNotFound = problem(404, "session_not_found", "The Account has no session with that id")
)

func (n *node) withKey(key, method, target, body string) *httptest.ResponseRecorder {
	n.t.Helper()
	return n.do(call{method: method, target: target, body: body, headers: map[string]string{"X-Api-Key": key}})
}

type sessionList struct {
	Sessions []struct {
		ID      string `json:"id"`
		Device  string `json:"device"`
		Current bool   `json:"current"`
	} `json:"sessions"`
}

func (n *node) currentSession(token string) string {
	n.t.Helper()
	for _, s := range decode[sessionList](n.t, n.as(token, http.MethodGet, "/api/v1/sessions", ""), http.StatusOK).Sessions {
		if s.Current {
			return s.ID
		}
	}
	n.t.Fatal("no current session")
	return ""
}

func (n *node) stream(account, token string) *live.Subscription {
	n.t.Helper()
	viewer := live.Viewer{Account: uuid(n.t, account), Session: uuid(n.t, n.currentSession(token))}
	sub := n.hub.Subscribe(viewer, n.start)
	n.t.Cleanup(sub.Close)
	return sub
}

func uuid(t *testing.T, s string) storage.UUID {
	t.Helper()
	u, err := storage.ParseUUID(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func ending(t *testing.T, sub *live.Subscription) string {
	t.Helper()
	done, cancel := context.WithCancel(t.Context())
	cancel()
	for {
		msg, err := sub.Next(done)
		if err != nil {
			return ""
		}
		switch d := msg.Data.(type) {
		case live.SessionEnded:
			return msg.Name + " " + string(d.Reason)
		case live.AccessChanged:
			return msg.Name
		}
	}
}

func TestTheLastAdminCannotBeDemotedTurnedOffOrDeleted(t *testing.T) {
	n := newNode(t)
	res, a := n.claimed()
	n.setAddress(a)
	admin := "/api/v1/accounts/" + res.Account.ID
	expect(t, n.as(a, http.MethodPatch, admin, `{"admin":false}`), 409, lastAdmin)
	expect(t, n.as(a, http.MethodPatch, admin, `{"disabled":true}`), 409, lastAdmin)
	expect(t, n.as(a, http.MethodDelete, admin, ""), 409, lastAdmin)
	if w := n.as(a, http.MethodPatch, admin, `{"displayName":"Ada"}`); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"displayName":"Ada","admin":true,"disabled":false`) {
		t.Fatalf("renaming the last Admin = %d %s", w.Code, w.Body)
	}

	grace, _ := n.grace(a)
	if w := n.as(a, http.MethodPatch, "/api/v1/accounts/"+grace, `{"admin":true}`); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"admin":true`) {
		t.Fatalf("promoting Grace = %d %s", w.Code, w.Body)
	}
	codes := make([]int, 2)
	var wg sync.WaitGroup
	for i, id := range []string{res.Account.ID, grace} {
		wg.Go(func() {
			codes[i] = n.withKey(res.APIKey, http.MethodPatch, "/api/v1/accounts/"+id, `{"admin":false}`).Code
		})
	}
	wg.Wait()
	if codes[0]+codes[1] != http.StatusOK+http.StatusConflict {
		t.Fatalf("demoting both Admins at once = %v, want one 200 and one 409", codes)
	}
	list := n.withKey(res.APIKey, http.MethodGet, "/api/v1/accounts", "").Body.String()
	if got := strings.Count(list, `"admin":true`); got != 1 {
		t.Fatalf("%d Admins remain: %s", got, list)
	}
}

func TestATurnedOffAccountEndsItsSessionsAndIsRefusedEveryWayIn(t *testing.T) {
	n := newNode(t)
	_, a := n.claimed()
	n.setAddress(a)
	grace, g := n.grace(a)
	expect(t, n.as(g, http.MethodPost, "/api/v1/account/email", `{"email":"grace@hopper.example"}`), 200, `{"email":{"outcome":"sent"}}`+"\n")
	confirm := regexp.MustCompile(`Confirm this email: (\S+)`).FindStringSubmatch(n.mail.Wait(t, "grace@hopper.example").Text)
	reset := decode[struct {
		URL string `json:"url"`
	}](t, n.as(a, http.MethodPost, "/api/v1/accounts/"+grace+"/reset-link", `{}`), http.StatusCreated)
	n.mail.Wait(t, "grace@example.com")
	g = sessionOf(t, n.signIn("grace@example.com", "1357"))
	sub := n.stream(grace, g)

	w := n.as(a, http.MethodPatch, "/api/v1/accounts/"+grace, `{"disabled":true}`)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"disabled":true`) {
		t.Fatalf("turning Grace off = %d %s", w.Code, w.Body)
	}
	if got := ending(t, sub); got != "live.session-ended revoked" {
		t.Fatalf("Grace's stream = %q, want ended revoked", got)
	}
	expect(t, n.getSession(g), 401, signedOut)
	expect(t, n.signIn("grace@example.com", "1357"), 403, accountDisabled)
	expect(t, n.signIn("grace@example.com", "0000"), 401, pinWrong)
	expect(t, n.as(a, http.MethodPost, "/api/v1/accounts/"+grace+"/reset-link", `{}`), 403, accountDisabled)
	expect(t, n.do(call{method: http.MethodPost, target: "/api/v1/reset-links/by-token/" + tokenOf(t, reset.URL) + "/pin", body: `{"pin":"8642"}`}), 403, accountDisabled)
	expect(t, n.do(call{method: http.MethodPost, target: "/api/v1/email-links/by-token/" + tokenOf(t, confirm[1]) + "/confirm"}), 403, accountDisabled)

	if w := n.as(a, http.MethodPatch, "/api/v1/accounts/"+grace, `{"disabled":false}`); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"disabled":false`) {
		t.Fatalf("turning Grace on = %d %s", w.Code, w.Body)
	}
	if w := n.signIn("grace@example.com", "1357"); w.Code != http.StatusOK {
		t.Fatalf("Grace signing in once on = %d %s", w.Code, w.Body)
	}
}

func TestAChangedRoleReGatesTheAccountsOpenTabs(t *testing.T) {
	n := newNode(t)
	_, a := n.claimed()
	n.setAddress(a)
	grace, g := n.grace(a)
	sub := n.stream(grace, g)
	if w := n.as(a, http.MethodPatch, "/api/v1/accounts/"+grace, `{"admin":true}`); w.Code != http.StatusOK {
		t.Fatalf("promoting Grace = %d %s", w.Code, w.Body)
	}
	if got := ending(t, sub); got != "live.access-changed" {
		t.Fatalf("Grace's stream = %q, want access-changed", got)
	}
	expect(t, n.getSession(g), 200, `{"account":{"id":"`+grace+`","email":"grace@example.com","displayName":"Grace Hopper","admin":true},"pinRequired":false,"keyPrompt":"ask","firstRun":"finished"}`+"\n")
	if w := n.as(g, http.MethodGet, "/api/v1/accounts", ""); w.Code != http.StatusOK {
		t.Fatalf("Grace listing Accounts as an Admin = %d %s", w.Code, w.Body)
	}
}

func TestSessionsAreListedAndEndedOneOrAllAtOnce(t *testing.T) {
	n := newNode(t)
	_, a := n.claimed()
	n.setAddress(a)
	grace, g := n.grace(a)
	phone := n.do(call{method: http.MethodPost, target: "/api/v1/session", body: `{"email":"grace@example.com","pin":"1357"}`,
		headers: map[string]string{"User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1"}})
	p := sessionOf(t, phone)
	list := decode[sessionList](t, n.as(g, http.MethodGet, "/api/v1/sessions", ""), http.StatusOK)
	if len(list.Sessions) != 2 || list.Sessions[0].Device != "Safari on iPhone" || list.Sessions[0].Current || !list.Sessions[1].Current {
		t.Fatalf("Grace's sessions = %+v", list)
	}
	phoneID, own := list.Sessions[0].ID, list.Sessions[1].ID
	sub := n.stream(grace, p)
	expect(t, n.as(g, http.MethodDelete, "/api/v1/sessions/"+phoneID, ""), 204, "")
	if got := ending(t, sub); got != "live.session-ended revoked" {
		t.Fatalf("the phone's stream = %q, want ended revoked", got)
	}
	expect(t, n.getSession(p), 401, signedOut)
	expect(t, n.as(g, http.MethodDelete, "/api/v1/sessions/"+phoneID, ""), 404, sessionNotFound)

	p = sessionOf(t, n.signIn("grace@example.com", "1357"))
	sub = n.stream(grace, p)
	expect(t, n.as(g, http.MethodDelete, "/api/v1/sessions?others=false", ""), 422,
		`{"status":422,"code":"invalid","title":"Some fields have values SlipStream can't accept","errors":[{"location":"query.others","message":"expected true"}]}`+"\n")
	expect(t, n.as(g, http.MethodDelete, "/api/v1/sessions?others=true", ""), 204, "")
	if got := ending(t, sub); got != "live.session-ended revoked" {
		t.Fatalf("the other stream = %q, want ended revoked", got)
	}
	if got := decode[sessionList](t, n.as(g, http.MethodGet, "/api/v1/sessions", ""), http.StatusOK); len(got.Sessions) != 1 || got.Sessions[0].ID != own {
		t.Fatalf("after ending the others = %+v", got)
	}

	sub = n.stream(grace, g)
	expect(t, n.as(g, http.MethodDelete, "/api/v1/sessions/"+own, ""), 204, "")
	if got := ending(t, sub); got != "live.session-ended signed-out" {
		t.Fatalf("the own stream = %q, want signed-out", got)
	}
	expect(t, n.getSession(g), 401, signedOut)
}

func TestAnAdminEndsAnAccountsSessions(t *testing.T) {
	n := newNode(t)
	_, a := n.claimed()
	n.setAddress(a)
	grace, g := n.grace(a)
	other := sessionOf(t, n.signIn("grace@example.com", "1357"))
	sessions := "/api/v1/accounts/" + grace + "/sessions"
	list := decode[sessionList](t, n.as(a, http.MethodGet, sessions, ""), http.StatusOK)
	if len(list.Sessions) != 2 || list.Sessions[0].Current || list.Sessions[1].Current {
		t.Fatalf("Grace's sessions as the Admin sees them = %+v", list)
	}
	sub := n.stream(grace, other)
	otherID := n.currentSession(other)
	expect(t, n.as(a, http.MethodDelete, sessions+"/"+otherID, ""), 204, "")
	if got := ending(t, sub); got != "live.session-ended revoked" {
		t.Fatalf("the ended stream = %q, want revoked", got)
	}
	expect(t, n.as(a, http.MethodDelete, sessions+"/"+otherID, ""), 404, sessionNotFound)

	sub = n.stream(grace, g)
	expect(t, n.as(a, http.MethodDelete, sessions, ""), 204, "")
	if got := ending(t, sub); got != "live.session-ended revoked" {
		t.Fatalf("Sign out everywhere = %q, want revoked", got)
	}
	expect(t, n.getSession(g), 401, signedOut)
	expect(t, n.as(a, http.MethodGet, "/api/v1/accounts/"+missing+"/sessions", ""), 404, accountNotFound)
}

const missing = "01a14564-0000-7000-8000-000000000000"

type accountList struct {
	Accounts []struct {
		ID           string     `json:"id"`
		Admin        bool       `json:"admin"`
		PasskeyCount int        `json:"passkeyCount"`
		LastSignInAt *time.Time `json:"lastSignInAt"`
		PIN          struct {
			State string     `json:"state"`
			Until *time.Time `json:"until"`
		} `json:"pin"`
	} `json:"accounts"`
}

func TestTheAccountsListShowsEachAccountsState(t *testing.T) {
	n := newNode(t)
	res, a := n.claimed()
	n.setAddress(a)
	grace, _ := n.grace(a)
	for range 5 {
		n.signIn("grace@example.com", "0000")
	}
	list := decode[accountList](t, n.as(a, http.MethodGet, "/api/v1/accounts", ""), http.StatusOK)
	if len(list.Accounts) != 2 {
		t.Fatalf("accounts = %+v", list)
	}
	ada, g := list.Accounts[0], list.Accounts[1]
	if ada.ID != res.Account.ID || !ada.Admin || ada.PIN.State != "fine" || ada.LastSignInAt == nil {
		t.Fatalf("the Admin's row = %+v", ada)
	}
	if g.ID != grace || g.Admin || g.PasskeyCount != 0 || g.PIN.State != "waiting" || g.PIN.Until == nil {
		t.Fatalf("Grace's row = %+v", g)
	}
	detail := n.as(a, http.MethodGet, "/api/v1/accounts/"+grace, "").Body.String()
	if !strings.Contains(detail, `"passkeys":[]`) || strings.Count(detail, `"device":"A browser"`) != 1 {
		t.Fatalf("Grace's detail = %s", detail)
	}
	expect(t, n.as(a, http.MethodGet, "/api/v1/accounts/"+missing, ""), 404, accountNotFound)
	expect(t, n.as(a, http.MethodGet, "/api/v1/accounts/not-an-id", ""), 422,
		`{"status":422,"code":"invalid","title":"Some fields have values SlipStream can't accept","errors":[{"location":"path.id","message":"expected string to be RFC 4122 uuid: invalid UUID length: 9"}]}`+"\n")
}

func TestAnAdminChangesAnEmailDirectlyAndTheOldAddressIsTold(t *testing.T) {
	n := newNode(t)
	_, a := n.claimed()
	n.setAddress(a)
	grace, g := n.grace(a)
	expect(t, n.as(a, http.MethodPatch, "/api/v1/accounts/"+grace, `{"email":"admin@example.com"}`), 409, emailTaken)
	if w := n.as(a, http.MethodPatch, "/api/v1/accounts/"+grace, `{"email":"Grace@Hopper.example","displayName":"Grace"}`); w.Code != http.StatusOK ||
		!strings.Contains(w.Body.String(), `"email":"grace@hopper.example","displayName":"Grace"`) {
		t.Fatalf("changing Grace's email = %d %s", w.Code, w.Body)
	}
	if told := n.mail.Wait(t, "grace@example.com"); told.Subject != "Your SlipStream email was changed" {
		t.Fatalf("the old address was told %q", told.Subject)
	}
	expect(t, n.getSession(g), 200, `{"account":{"id":"`+grace+`","email":"grace@hopper.example","displayName":"Grace","admin":false},"pinRequired":false,"keyPrompt":"ask"}`+"\n")
	if n.published(grace) == 0 {
		t.Fatal("Grace's tabs never heard her Account changed")
	}
}

func TestDeletingAnAccountEndsItsSessionsAndFreesItsEmail(t *testing.T) {
	n := newNode(t)
	_, a := n.claimed()
	n.setAddress(a)
	grace, g := n.grace(a)
	sub := n.stream(grace, g)
	expect(t, n.as(a, http.MethodDelete, "/api/v1/accounts/"+grace, ""), 204, "")
	if got := ending(t, sub); got != "live.session-ended revoked" {
		t.Fatalf("Grace's stream = %q, want revoked", got)
	}
	expect(t, n.getSession(g), 401, signedOut)
	expect(t, n.signIn("grace@example.com", "1357"), 401, pinWrong)
	expect(t, n.as(a, http.MethodDelete, "/api/v1/accounts/"+grace, ""), 404, accountNotFound)
	if w := n.as(a, http.MethodPost, "/api/v1/signup-links", `{"email":"grace@example.com"}`); w.Code != http.StatusCreated {
		t.Fatalf("a new Signup link for the freed email = %d %s", w.Code, w.Body)
	}
}

func TestYourAccountChangesItsNameAndThePasskeyPrompt(t *testing.T) {
	n := newNode(t)
	res, a := n.claimed()
	want := `{"account":{"id":"` + res.Account.ID + `","email":"admin@example.com","displayName":"Ada","admin":true},"pinRequired":false,"keyPrompt":"never","firstRun":"finished"}` + "\n"
	expect(t, n.as(a, http.MethodPatch, "/api/v1/account", `{"displayName":"Ada","keyPrompt":"never"}`), 200, want)
	expect(t, n.getSession(a), 200, want)
}
