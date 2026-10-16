package identityapi_test

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

// signInAt signs the Admin in at origin with the PIN.
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

// addPasskey runs a registration at origin for session, answered by key,
// and returns the finish's answer.
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
	passkeyFailed := problem(401, "passkey_failed", "The passkey did not check out")
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
