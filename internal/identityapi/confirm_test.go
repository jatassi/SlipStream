package identityapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
)

var (
	needsReconfirm = problem(403, "needs_reconfirm", "Confirm it's you to continue")
	pinRequired    = problem(403, "pin_required", "Set a new PIN to continue")
)

func (n *node) reconfirm(session, pin string) answer {
	n.t.Helper()
	return answerOf(n.do(call{method: http.MethodPost, target: "/api/v1/session/reconfirm", session: session, body: `{"pin":"` + pin + `"}`}))
}

func (n *node) patchSettingsAs(session, body string) *httptest.ResponseRecorder {
	n.t.Helper()
	return n.do(call{method: http.MethodPatch, target: "/api/v1/identity/settings", session: session, body: body})
}

func (n *node) changePIN(session, body string) *httptest.ResponseRecorder {
	n.t.Helper()
	return n.do(call{method: http.MethodPost, target: "/api/v1/account/pin", session: session, body: body})
}

func TestAConfirmLastsTheWindowAndReconfirmRenewsIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t)
		_, session := n.claimed()
		if w := n.patchSettingsAs(session, `{"pinWaitAfter":5}`); w.Code != http.StatusOK {
			t.Fatalf("straight after the claim, updateIdentitySettings = %d %s", w.Code, w.Body)
		}
		time.Sleep(15*time.Minute + time.Second)
		expect(t, n.patchSettingsAs(session, `{"pinWaitAfter":5}`), 403, needsReconfirm)
		if w := n.getSession(session); w.Code != http.StatusOK {
			t.Fatalf("a lapsed confirm still reads its session, got %d %s", w.Code, w.Body)
		}
		want(t, "re-confirm with the right PIN", n.reconfirm(session, "1234"), answer{status: 204})
		if w := n.patchSettingsAs(session, `{"reconfirmWindow":3600}`); w.Code != http.StatusOK {
			t.Fatalf("after re-confirm, updateIdentitySettings = %d %s", w.Code, w.Body)
		}
		time.Sleep(59 * time.Minute)
		if w := n.patchSettingsAs(session, `{"pinWaitAfter":5}`); w.Code != http.StatusOK {
			t.Fatalf("59 minutes into an hour's window, updateIdentitySettings = %d %s", w.Code, w.Body)
		}
	})
}

// TestReconfirmFailuresCountTowardThePINsLimits guesses through the
// re-confirm dialog: the guesses share sign-in's ledger, so they wait and
// turn the PIN off the same way.
func TestReconfirmFailuresCountTowardThePINsLimits(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t)
		_, session := n.claimed()
		for i := 1; i <= 5; i++ {
			want(t, fmt.Sprintf("wrong re-confirm %d", i), n.reconfirm(session, "9999"), wrongAnswer)
		}
		until := time.Now().Add(15 * time.Minute)
		want(t, "the right PIN at sign-in during the wait", n.signInFrom("192.0.2.1:1", "admin@example.com", "1234", nil), pinWaitAnswer(until, 900))
		want(t, "the right PIN at re-confirm during the wait", n.reconfirm(session, "1234"), pinWaitAnswer(until, 900))
		for i := 6; i <= 10; i++ {
			time.Sleep(time.Until(until))
			want(t, fmt.Sprintf("wrong re-confirm %d", i), n.reconfirm(session, "9999"), wrongAnswer)
			until = time.Now().Add(15 * time.Minute)
		}
		time.Sleep(time.Until(until))
		want(t, "the right PIN at sign-in once off", n.signInFrom("192.0.2.1:1", "admin@example.com", "1234", nil), offAnswer)
	})
}

func TestChangePINNeedsTheCurrentOne(t *testing.T) {
	n := newNode(t)
	_, session := n.claimed()
	expect(t, n.changePIN(session, `{"new":"5678"}`), 422,
		`{"status":422,"code":"invalid","title":"Some fields have values SlipStream can't accept","errors":[{"location":"body.current","message":"expected the current PIN"}]}`+"\n")
	want(t, "a wrong current PIN", answerOf(n.changePIN(session, `{"current":"9999","new":"5678"}`)), wrongAnswer)
	want(t, "the right current PIN", answerOf(n.changePIN(session, `{"current":"1234","new":"5678"}`)), answer{status: 204})
	want(t, "the old PIN", n.signInFrom("192.0.2.1:1", "admin@example.com", "1234", nil), wrongAnswer)
	if got := n.signInFrom("192.0.2.1:1", "admin@example.com", "5678", nil); got.status != http.StatusOK {
		t.Fatalf("the new PIN = %d %s, want 200", got.status, got.body)
	}
}

// TestASessionThatMustSetAPINCanDoNothingElse is the session a passkey
// sign-in starts on a turned-off PIN. Setting a new PIN turns it back on.
func TestASessionThatMustSetAPINCanDoNothingElse(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t)
		res, _ := n.claimed()
		admin := identity.AccountID{UUID: mustUUID(t, res.Account.ID)}
		for i := 1; i <= 10; i++ {
			n.signInFrom("192.0.2.1:1", "admin@example.com", "9999", nil)
			if i >= 5 {
				time.Sleep(15 * time.Minute)
			}
		}
		want(t, "the right PIN once off", n.signInFrom("192.0.2.1:1", "admin@example.com", "1234", nil), offAnswer)
		session := identitytest.PINRequiredSession(t, n.lib, admin)
		account := `{"id":"` + res.Account.ID + `","email":"admin@example.com","displayName":"Admin","admin":true}`
		expect(t, n.getSession(session), 200, `{"account":`+account+`,"pinRequired":true,"keyPrompt":"ask"}`+"\n")
		expect(t, n.do(call{method: http.MethodGet, target: "/api/v1/identity/settings", session: session}), 403, pinRequired)
		want(t, "re-confirm", n.reconfirm(session, "1234"), answer{status: 403, body: pinRequired})
		want(t, "a new PIN with no current one", answerOf(n.changePIN(session, `{"new":"5678"}`)), answer{status: 204})
		expect(t, n.getSession(session), 200, `{"account":`+account+`,"pinRequired":false,"keyPrompt":"ask"}`+"\n")
		if w := n.do(call{method: http.MethodGet, target: "/api/v1/identity/settings", session: session}); w.Code != http.StatusOK {
			t.Fatalf("after the new PIN, getIdentitySettings = %d %s", w.Code, w.Body)
		}
		if got := n.signInFrom("192.0.2.1:1", "admin@example.com", "5678", nil); got.status != http.StatusOK {
			t.Fatalf("sign-in with the new PIN = %d %s, want 200", got.status, got.body)
		}
	})
}

func TestRegeneratingTheAPIKeyStopsTheOldOne(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t)
		res, session := n.claimed()
		claimedAt := time.Now().UTC()
		expect(t, n.do(call{method: http.MethodGet, target: "/api/v1/api-key", headers: map[string]string{"X-Api-Key": res.APIKey}}), 200,
			`{"createdAt":"`+claimedAt.Format(time.RFC3339)+`"}`+"\n")
		time.Sleep(time.Hour)
		expect(t, n.do(call{method: http.MethodPost, target: "/api/v1/api-key", session: session}), 403, needsReconfirm)
		want(t, "re-confirm", n.reconfirm(session, "1234"), answer{status: 204})
		w := n.do(call{method: http.MethodPost, target: "/api/v1/api-key", session: session})
		var got struct {
			APIKey    string    `json:"apiKey"`
			CreatedAt time.Time `json:"createdAt"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || w.Code != http.StatusOK {
			t.Fatalf("regenerateAPIKey = %d %s", w.Code, w.Body)
		}
		if got.APIKey == res.APIKey || !got.CreatedAt.Equal(time.Now()) {
			t.Fatalf("regenerateAPIKey = %+v, want a new key made now", got)
		}
		expect(t, n.do(call{method: http.MethodGet, target: "/api/v1/api-key", headers: map[string]string{"X-Api-Key": res.APIKey}}), 401,
			problem(401, "api_key_wrong", "The API key is wrong"))
		expect(t, n.do(call{method: http.MethodGet, target: "/api/v1/api-key", headers: map[string]string{"X-Api-Key": got.APIKey}}), 200,
			`{"createdAt":"`+time.Now().UTC().Format(time.RFC3339)+`"}`+"\n")
	})
}
