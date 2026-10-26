package identityapi_test

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/identity/passkey/passkeytest"
)

const (
	home           = "http://home.localhost:8484"
	named          = "https://media.example.com"
	safariOnIPhone = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1"
)

var (
	passkeyUnavailable = problem(409, "passkey_unavailable", "A passkey can't be used at this address")
	passkeyNotFound    = problem(404, "passkey_not_found", "You have no passkey with that id")
)

func aaguid(t *testing.T, s string) [16]byte {
	t.Helper()
	b, err := hex.DecodeString(strings.ReplaceAll(s, "-", ""))
	if err != nil || len(b) != 16 {
		t.Fatalf("aaguid %q: %v", s, err)
	}
	return [16]byte(b)
}

// onePassword is an authenticator reporting 1Password's AAGUID.
func onePassword(t *testing.T) *passkeytest.Authenticator {
	return &passkeytest.Authenticator{AAGUID: aaguid(t, "bada5566-a7aa-401f-bd96-45619a55120d"), Synced: true}
}

func (n *node) signInAt(origin string) *httptest.ResponseRecorder {
	n.t.Helper()
	return n.do(call{method: http.MethodPost, target: origin + "/api/v1/session", body: `{"email":"admin@example.com","pin":"1234"}`})
}

type started struct {
	Ceremony string          `json:"ceremony"`
	Options  json.RawMessage `json:"options"`
}

type passkeyRow struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Host       string     `json:"host"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
}

type passkeyList struct {
	Passkeys []passkeyRow `json:"passkeys"`
}

func (n *node) addPasskey(origin, session string, key *passkeytest.Authenticator, headers map[string]string) *httptest.ResponseRecorder {
	n.t.Helper()
	s := decode[started](n.t, n.do(call{method: http.MethodPost, target: origin + "/api/v1/passkeys/registration", session: session, headers: headers}), http.StatusOK)
	response, err := key.Register(origin, s.Options)
	if err != nil {
		n.t.Fatal(err)
	}
	return n.do(call{method: http.MethodPost, target: origin + "/api/v1/passkeys/registration/" + s.Ceremony, session: session, headers: headers, body: string(response)})
}

func (n *node) passkeys(session string) []passkeyRow {
	n.t.Helper()
	return decode[passkeyList](n.t, n.as(session, http.MethodGet, "/api/v1/passkeys", ""), http.StatusOK).Passkeys
}

func hosts(keys []passkeyRow) string {
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = k.Name + " at " + k.Host
	}
	return strings.Join(out, ", ")
}

type prompted struct {
	PasskeyPrompt bool `json:"passkeyPrompt"`
}

type firstRunPrompted struct {
	FirstRun      *string `json:"firstRun"`
	PasskeyPrompt bool    `json:"passkeyPrompt"`
}

func TestFirstSetupHoldsThePasskeyPromptAndReachesOnlyAdmins(t *testing.T) {
	for _, open := range []bool{true, false} {
		n := newNodeWith(t, nodeOptions{setupOpen: open})
		n.claimed()
		state := map[bool]string{true: "open", false: "finished"}[open]
		got := decode[firstRunPrompted](t, n.signInAt(home), http.StatusOK)
		if got.FirstRun == nil || *got.FirstRun != state || got.PasskeyPrompt == open {
			t.Errorf("an Admin's sign-in with first setup %s = firstRun %v, passkeyPrompt %v; want %s and %v", state, got.FirstRun, got.PasskeyPrompt, state, !open)
		}
		member := identitytest.Session(t, n.lib, identitytest.Member(t, n.lib, "member@example.com"))
		if got := decode[firstRunPrompted](t, n.getSession(member), http.StatusOK); got.FirstRun != nil {
			t.Errorf("a Member's session carries firstRun %q", *got.FirstRun)
		}
	}
}

func TestThePasskeyPromptShowsOnlyWhereAPasskeyWorks(t *testing.T) {
	n := newNode(t)
	_, a := n.claimed()
	for origin, want := range map[string]bool{home: true, named: true, "http://media.example.com": false, "https://192.168.1.20:8484": false} {
		if got := decode[prompted](t, n.signInAt(origin), http.StatusOK); got.PasskeyPrompt != want {
			t.Errorf("signing in at %s: passkeyPrompt %v, want %v", origin, got.PasskeyPrompt, want)
		}
	}
	if w := n.as(a, http.MethodPatch, "/api/v1/account", `{"keyPrompt":"never"}`); w.Code != http.StatusOK {
		t.Fatalf("Don't ask again = %d %s", w.Code, w.Body)
	}
	if got := decode[prompted](t, n.signInAt(named), http.StatusOK); got.PasskeyPrompt {
		t.Error("the prompt showed after Don't ask again")
	}
}

func TestAPasskeyIsAddedOnlyWhereTheAddressCanHoldOne(t *testing.T) {
	n := newNode(t)
	res, a := n.claimed()
	key := onePassword(t)
	for _, origin := range []string{home, named} {
		got := decode[passkeyRow](t, n.addPasskey(origin, a, key, nil), http.StatusCreated)
		if got.Name != "1Password" || got.Host != strings.TrimSuffix(strings.SplitN(origin, "//", 2)[1], ":8484") || got.LastUsedAt != nil || got.CreatedAt.IsZero() {
			t.Fatalf("the passkey added at %s = %+v", origin, got)
		}
	}
	for _, origin := range []string{"https://192.168.1.20:8484", "http://[::1]:8484", "http://media.example.com", "https://nas:8484"} {
		expect(t, n.do(call{method: http.MethodPost, target: origin + "/api/v1/passkeys/registration", session: a}), 409, passkeyUnavailable)
		expect(t, n.do(call{method: http.MethodPost, target: origin + "/api/v1/passkeys/registration/ABC", session: a, body: "{}"}), 409, passkeyUnavailable)
	}
	if got := hosts(n.passkeys(a)); got != "1Password at home.localhost, 1Password at media.example.com" {
		t.Fatalf("listPasskeys = %s", got)
	}
	if got := n.heard(res.Account.ID, identity.PasskeysChangedEvent.Name()); got != 2 {
		t.Fatalf("passkeys-changed reached the Admin %d times, want once per passkey", got)
	}
	if got := decode[prompted](t, n.signInAt(home), http.StatusOK); got.PasskeyPrompt {
		t.Error("the prompt showed at an address that already has a passkey")
	}
	if got := decode[prompted](t, n.signInAt("http://den.localhost:8484"), http.StatusOK); !got.PasskeyPrompt {
		t.Error("the prompt did not show at an address with no passkey")
	}
}

func TestAPasskeyWithNoKnownProviderIsNamedForTheDevice(t *testing.T) {
	n := newNode(t)
	_, a := n.claimed()
	got := decode[passkeyRow](t, n.addPasskey(named, a, &passkeytest.Authenticator{}, map[string]string{"User-Agent": safariOnIPhone}), http.StatusCreated)
	if got.Name != "Safari on iPhone" {
		t.Fatalf("name = %q, want the device label", got.Name)
	}
}

func TestAddingAPasskeyNeedsARecentConfirmAndARealAnswer(t *testing.T) {
	n := newNode(t)
	_, a := n.claimed()
	n.setAddress(a)
	grace, _ := n.grace(a)
	lapsed := identitytest.LapsedSession(t, n.lib, identity.AccountID{UUID: uuid(t, grace)})
	expect(t, n.do(call{method: http.MethodPost, target: home + "/api/v1/passkeys/registration", session: lapsed}), 403,
		problem(403, "needs_reconfirm", "Confirm it's you to continue"))

	s := decode[started](t, n.do(call{method: http.MethodPost, target: home + "/api/v1/passkeys/registration", session: a}), http.StatusOK)
	response, err := onePassword(t).Register(named, s.Options)
	if err != nil {
		t.Fatal(err)
	}
	passkeyFailed := problem(401, "passkey_failed", "The passkey couldn't be verified")
	expect(t, n.do(call{method: http.MethodPost, target: home + "/api/v1/passkeys/registration/" + s.Ceremony, session: a, body: string(response)}), 401, passkeyFailed)
	expect(t, n.do(call{method: http.MethodPost, target: home + "/api/v1/passkeys/registration/" + s.Ceremony, session: a, body: string(response)}), 401, passkeyFailed)
	if got := n.passkeys(a); len(got) != 0 {
		t.Fatalf("a refused registration saved %s", hosts(got))
	}
}

func TestPasskeysAreRenamedAndDeletedByTheirOwnerOnly(t *testing.T) {
	n := newNode(t)
	res, a := n.claimed()
	n.setAddress(a)
	_, g := n.grace(a)
	key := decode[passkeyRow](t, n.addPasskey(home, a, onePassword(t), nil), http.StatusCreated)

	got := decode[passkeyRow](t, n.as(a, http.MethodPatch, "/api/v1/passkeys/"+key.ID, `{"name":"Work laptop"}`), http.StatusOK)
	if got.Name != "Work laptop" || got.ID != key.ID || got.Host != "home.localhost" {
		t.Fatalf("renamed = %+v", got)
	}
	expect(t, n.as(g, http.MethodPatch, "/api/v1/passkeys/"+key.ID, `{"name":"Mine now"}`), 404, passkeyNotFound)
	expect(t, n.as(g, http.MethodDelete, "/api/v1/passkeys/"+key.ID, ""), 404, passkeyNotFound)
	if got := hosts(n.passkeys(a)); got != "Work laptop at home.localhost" {
		t.Fatalf("listPasskeys = %s", got)
	}
	expect(t, n.as(a, http.MethodDelete, "/api/v1/passkeys/"+key.ID, ""), 204, "")
	expect(t, n.as(a, http.MethodDelete, "/api/v1/passkeys/"+key.ID, ""), 404, passkeyNotFound)
	if got := n.passkeys(a); len(got) != 0 {
		t.Fatalf("listPasskeys after the delete = %s", hosts(got))
	}
	if got := n.heard(res.Account.ID, identity.PasskeysChangedEvent.Name()); got != 3 {
		t.Fatalf("passkeys-changed reached the Admin %d times, want at the add, the rename and the delete", got)
	}
}

// TestTheSignupAndResetFlowsAddPasskeysOnTheirNewSession: each flow signs
// in, so its passkey step is already confirmed.
func TestTheSignupAndResetFlowsAddPasskeysOnTheirNewSession(t *testing.T) {
	n := newNode(t)
	_, a := n.claimed()
	n.setAddress(a)
	grace, joined := n.grace(a)
	if w := n.addPasskey(home, joined, onePassword(t), nil); w.Code != http.StatusCreated {
		t.Fatalf("the signup flow's passkey step = %d %s", w.Code, w.Body)
	}

	reset := func(removePasskeys bool) string {
		t.Helper()
		issued := decode[struct {
			URL string `json:"url"`
		}](t, n.as(a, http.MethodPost, "/api/v1/accounts/"+grace+"/reset-link", `{"removePasskeys":`+map[bool]string{true: "true", false: "false"}[removePasskeys]+`}`), http.StatusCreated)
		w := n.do(call{method: http.MethodPost, target: home + "/api/v1/reset-links/by-token/" + tokenOf(t, issued.URL) + "/pin", body: `{"pin":"8642"}`})
		if w.Code != http.StatusOK {
			t.Fatalf("redeemResetLink = %d %s", w.Code, w.Body)
		}
		return sessionOf(t, w)
	}
	kept := reset(false)
	keys := n.passkeys(kept)
	if hosts(keys) != "1Password at home.localhost" {
		t.Fatalf("the Reset flow's passkeys = %s", hosts(keys))
	}
	expect(t, n.as(kept, http.MethodDelete, "/api/v1/passkeys/"+keys[0].ID, ""), 204, "")
	if w := n.addPasskey(home, kept, onePassword(t), nil); w.Code != http.StatusCreated {
		t.Fatalf("the Reset flow's passkey step = %d %s", w.Code, w.Body)
	}

	before := n.heard(grace, identity.PasskeysChangedEvent.Name())
	if got := n.passkeys(reset(true)); len(got) != 0 {
		t.Fatalf("a Reset link that removes passkeys left %s", hosts(got))
	}
	if n.heard(grace, identity.PasskeysChangedEvent.Name()) != before+1 {
		t.Fatal("removing the passkeys by Reset link reached no stream")
	}
}

var passkeyFailed = problem(401, "passkey_failed", "The passkey couldn't be verified")

func (n *node) passkeySignIn(origin, from string, key *passkeytest.Authenticator) *httptest.ResponseRecorder {
	n.t.Helper()
	s := decode[started](n.t, n.do(call{method: http.MethodPost, target: origin + "/api/v1/passkeys/assertion", from: from}), http.StatusOK)
	response, err := key.Assert(origin, s.Options)
	if err != nil {
		n.t.Fatal(err)
	}
	return n.do(call{method: http.MethodPost, target: origin + "/api/v1/passkeys/assertion/" + s.Ceremony, from: from, body: string(response)})
}

func (n *node) passkeyReconfirm(origin, session string, key *passkeytest.Authenticator) *httptest.ResponseRecorder {
	n.t.Helper()
	s := decode[started](n.t, n.do(call{method: http.MethodPost, target: origin + "/api/v1/session/reconfirm/passkey", session: session}), http.StatusOK)
	response, err := key.Assert(origin, s.Options)
	if err != nil {
		n.t.Fatal(err)
	}
	return n.do(call{method: http.MethodPost, target: origin + "/api/v1/session/reconfirm/passkey/" + s.Ceremony, session: session, body: string(response)})
}

func (n *node) graceWithPasskeys(admin string, key *passkeytest.Authenticator, origins ...string) (id, session string) {
	n.t.Helper()
	id, session = n.grace(admin)
	for _, origin := range origins {
		if w := n.addPasskey(origin, session, key, nil); w.Code != http.StatusCreated {
			n.t.Fatalf("adding Grace's passkey at %s = %d %s", origin, w.Code, w.Body)
		}
	}
	return id, session
}

func graceSignedIn(id string, pinRequired bool) string {
	return `{"account":{"id":"` + id + `","email":"grace@example.com","displayName":"Grace Hopper","admin":false},"pinRequired":` +
		map[bool]string{true: "true", false: "false"}[pinRequired] + `,"keyPrompt":"ask","passkeyPrompt":false}` + "\n"
}

func TestAPasskeySignsInWhereTheAddressCanHoldOne(t *testing.T) {
	n := newNode(t)
	_, a := n.claimed()
	n.setAddress(a)
	key := &passkeytest.Authenticator{}
	grace, _ := n.graceWithPasskeys(a, key, home, named)
	for origin, secure := range map[string]bool{home: false, named: true} {
		w := n.passkeySignIn(origin, "192.0.2.1:1", key)
		expect(t, w, 200, graceSignedIn(grace, false))
		if got := strings.Contains(w.Header().Get("Set-Cookie"), "; Secure"); got != secure {
			t.Errorf("signing in at %s set %q, want Secure %v", origin, w.Header().Get("Set-Cookie"), secure)
		}
		if s := n.getSession(sessionOf(t, w)); s.Code != http.StatusOK {
			t.Fatalf("the passkey's session at %s = %d %s", origin, s.Code, s.Body)
		}
	}
	for _, origin := range []string{"https://192.168.1.20:8484", "http://media.example.com"} {
		expect(t, n.do(call{method: http.MethodPost, target: origin + "/api/v1/passkeys/assertion"}), 409, passkeyUnavailable)
		expect(t, n.do(call{method: http.MethodPost, target: origin + "/api/v1/passkeys/assertion/ABC", body: "{}"}), 409, passkeyUnavailable)
	}

	s := decode[started](t, n.do(call{method: http.MethodPost, target: home + "/api/v1/passkeys/assertion"}), http.StatusOK)
	response, err := key.Assert(home, s.Options)
	if err != nil {
		t.Fatal(err)
	}
	expect(t, n.do(call{method: http.MethodPost, target: named + "/api/v1/passkeys/assertion/" + s.Ceremony, body: string(response)}), 401, passkeyFailed)
	expect(t, n.do(call{method: http.MethodPost, target: home + "/api/v1/passkeys/assertion/" + s.Ceremony, body: string(response)}), 401, passkeyFailed)
}

// TestEachSignInSavesTheSignCountAndLastUse: a copy of the authenticator, as a
// cloned one would be, answers with a count the stored one passed.
func TestEachSignInSavesTheSignCountAndLastUse(t *testing.T) {
	n := newNode(t)
	_, a := n.claimed()
	n.setAddress(a)
	key := &passkeytest.Authenticator{}
	_, g := n.graceWithPasskeys(a, key, home, named)
	clone := key.Clone()
	for range 2 {
		if w := n.passkeySignIn(home, "192.0.2.1:1", key); w.Code != http.StatusOK {
			t.Fatalf("passkey sign-in = %d %s", w.Code, w.Body)
		}
	}
	expect(t, n.passkeySignIn(home, "192.0.2.1:1", clone), 401, passkeyFailed)
	keys := n.passkeys(g)
	if len(keys) != 2 || keys[0].LastUsedAt == nil || keys[1].LastUsedAt != nil {
		t.Fatalf("passkeys = %+v, want the one at home used and the other not", keys)
	}
}

func TestATurnedOffPINSignsInByPasskeyToSetANewOne(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t)
		_, a := n.claimed()
		n.setAddress(a)
		key := onePassword(t)
		grace, _ := n.graceWithPasskeys(a, key, home)
		for i := 1; i <= 10; i++ {
			n.signInFrom("192.0.2.1:1", "grace@example.com", "0000", nil)
			if i >= 5 {
				time.Sleep(15 * time.Minute)
			}
		}
		want(t, "the right PIN once off", n.signInFrom("192.0.2.1:1", "grace@example.com", "1357", nil), offAnswer)
		w := n.passkeySignIn(home, "192.0.2.1:1", key)
		expect(t, w, 200, graceSignedIn(grace, true))
		s := sessionOf(t, w)
		expect(t, n.as(s, http.MethodGet, "/api/v1/passkeys", ""), 403, pinRequired)
		want(t, "a new PIN", answerOf(n.changePIN(s, `{"new":"2468"}`)), answer{status: 204})
		if keys := n.passkeys(s); len(keys) != 1 {
			t.Fatalf("passkeys after the new PIN = %+v", keys)
		}
		if got := n.signInFrom("192.0.2.1:1", "grace@example.com", "2468", nil); got.status != http.StatusOK {
			t.Fatalf("sign-in with the new PIN = %d %s", got.status, got.body)
		}
	})
}

func TestATurnedOffAccountCantSignInByPasskey(t *testing.T) {
	n := newNode(t)
	_, a := n.claimed()
	n.setAddress(a)
	key := onePassword(t)
	grace, _ := n.graceWithPasskeys(a, key, named)
	if w := n.as(a, http.MethodPatch, "/api/v1/accounts/"+grace, `{"disabled":true}`); w.Code != http.StatusOK {
		t.Fatalf("turning Grace off = %d %s", w.Code, w.Body)
	}
	expect(t, n.passkeySignIn(named, "192.0.2.1:1", key), 403, accountDisabled)
	if w := n.as(a, http.MethodPatch, "/api/v1/accounts/"+grace, `{"disabled":false}`); w.Code != http.StatusOK {
		t.Fatalf("turning Grace on = %d %s", w.Code, w.Body)
	}
	expect(t, n.passkeySignIn(named, "192.0.2.1:1", key), 200, graceSignedIn(grace, false))
}

func TestFailedPasskeysCountWithWrongPINsPerAddress(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t)
		_, a := n.claimed()
		n.setAddress(a)
		key := onePassword(t)
		n.graceWithPasskeys(a, key, home)
		for i := range 15 {
			want(t, "a wrong PIN", n.signInFrom("192.0.2.10:1", fmt.Sprintf("guess%d@example.com", i), "9999", nil), wrongAnswer)
			expect(t, n.do(call{method: http.MethodPost, target: home + "/api/v1/passkeys/assertion/NOSUCHCEREMONY", from: "192.0.2.10:1", body: "{}"}), 401, passkeyFailed)
		}
		blocked := tooManyAnswer(time.Now().Add(15 * time.Minute))
		want(t, "starting a passkey sign-in", answerOf(n.do(call{method: http.MethodPost, target: home + "/api/v1/passkeys/assertion", from: "192.0.2.10:1"})), blocked)
		want(t, "finishing one", answerOf(n.do(call{method: http.MethodPost, target: home + "/api/v1/passkeys/assertion/NOSUCHCEREMONY", from: "192.0.2.10:1", body: "{}"})), blocked)
		want(t, "the right PIN", n.signInFrom("192.0.2.10:1", "grace@example.com", "1357", nil), blocked)
		if w := n.passkeySignIn(home, "192.0.2.11:1", key); w.Code != http.StatusOK {
			t.Fatalf("a passkey from the neighbor = %d %s", w.Code, w.Body)
		}
		time.Sleep(15 * time.Minute)
		if w := n.passkeySignIn(home, "192.0.2.10:1", key); w.Code != http.StatusOK {
			t.Fatalf("a passkey once the window passed = %d %s", w.Code, w.Body)
		}
	})
}

func TestAPasskeyReconfirmsTheSession(t *testing.T) {
	n := newNode(t)
	res, a := n.claimed()
	key := onePassword(t)
	if w := n.addPasskey(named, a, key, nil); w.Code != http.StatusCreated {
		t.Fatalf("addPasskey = %d %s", w.Code, w.Body)
	}
	lapsed := identitytest.LapsedSession(t, n.lib, identity.AccountID{UUID: uuid(t, res.Account.ID)})
	needsReconfirm := problem(403, "needs_reconfirm", "Confirm it's you to continue")
	expect(t, n.do(call{method: http.MethodPost, target: named + "/api/v1/passkeys/registration", session: lapsed}), 403, needsReconfirm)
	for _, origin := range []string{home, "https://192.168.1.20:8484"} {
		expect(t, n.do(call{method: http.MethodPost, target: origin + "/api/v1/session/reconfirm/passkey", session: lapsed}), 409, passkeyUnavailable)
	}
	expect(t, n.passkeyReconfirm(named, lapsed, key), 204, "")
	if w := n.do(call{method: http.MethodPost, target: named + "/api/v1/passkeys/registration", session: lapsed}); w.Code != http.StatusOK {
		t.Fatalf("after re-confirm, startPasskeyRegistration = %d %s", w.Code, w.Body)
	}
	if keys := n.passkeys(a); len(keys) != 1 || keys[0].LastUsedAt == nil {
		t.Fatalf("passkeys = %+v, want the one used to re-confirm", keys)
	}
	expect(t, n.do(call{method: http.MethodPost, target: named + "/api/v1/session/reconfirm/passkey/NOSUCHCEREMONY", session: lapsed, body: "{}"}), 401, passkeyFailed)
}
