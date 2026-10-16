package identityapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	home  = "http://home.localhost:8484"
	named = "https://media.example.com"
)

func (n *node) signInAt(origin, email, pin string) *httptest.ResponseRecorder {
	n.t.Helper()
	return n.do(call{method: http.MethodPost, target: origin + "/api/v1/session", body: `{"email":"` + email + `","pin":"` + pin + `"}`})
}

type prompted struct {
	PasskeyPrompt bool `json:"passkeyPrompt"`
}

func TestThePasskeyPromptShowsOnlyWhereAPasskeyWorks(t *testing.T) {
	n := newNode(t)
	_, a := n.claimed()
	for origin, want := range map[string]bool{home: true, named: true, "http://media.example.com": false, "https://192.168.1.20:8484": false} {
		if got := decode[prompted](t, n.signInAt(origin, "admin@example.com", "1234"), http.StatusOK); got.PasskeyPrompt != want {
			t.Errorf("signing in at %s: passkeyPrompt %v, want %v", origin, got.PasskeyPrompt, want)
		}
	}
	if w := n.as(a, http.MethodPatch, "/api/v1/account", `{"keyPrompt":"never"}`); w.Code != http.StatusOK {
		t.Fatalf("Don't ask again = %d %s", w.Code, w.Body)
	}
	if got := decode[prompted](t, n.signInAt(named, "admin@example.com", "1234"), http.StatusOK); got.PasskeyPrompt {
		t.Error("the prompt showed after Don't ask again")
	}
}
