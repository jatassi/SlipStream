package notificationsapi_test

import (
	"net/http"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/notifications"
	"github.com/jatassi/slipstream/internal/notifications/notificationstest"
	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/notificationsapi"
)

type notes struct{ w *notificationstest.World }

func (n notes) Notifications() (*notifications.Service, bool) { return n.w.Service, true }

type served struct {
	*ownertest.Owner
	w *notificationstest.World
}

func serve(t *testing.T) served {
	t.Helper()
	w := notificationstest.New(t, notifications.DefaultTimings())
	return served{ownertest.ServeOver(t, w.Lib, w.Identity, func(r *api.Router) { notificationsapi.Routes(r, notes{w}) }), w}
}

// call answers and lets the dispatcher run what the call woke.
func (s served) call(t *testing.T, method, path, body string) (int, string) {
	t.Helper()
	code, got := s.Call(t, method, path, body)
	synctest.Wait()
	return code, got
}

func (s served) want(t *testing.T, method, path, body string, status int, want string) {
	t.Helper()
	code, got := s.call(t, method, path, body)
	if code != status || ownertest.Masked(got) != want+"\n" {
		t.Fatalf("%s %s = %d %s, want %d %s", method, path, code, ownertest.Masked(got), status, want)
	}
}

const family = `{"name":"Family","provider":"hook","on":true,"posters":true,"values":{"url":"https://hook.example/Family"},"secrets":{"token":"token-of-Family","signing":"sig-of-Family"}}`

func (s served) add(t *testing.T) string {
	t.Helper()
	code, got := s.call(t, http.MethodPost, "/api/v1/notifications/connections", family)
	if code != http.StatusCreated {
		t.Fatalf("addNotificationConnection = %d %s", code, got)
	}
	return ownertest.FirstID(got)
}

func (s served) headers(t *testing.T, target string) []http.Header {
	t.Helper()
	var out []http.Header
	for _, r := range s.w.Courier.Received(target) {
		out = append(out, r.Header)
	}
	return out
}

func TestNoAnswerOrLogHoldsASecret(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := serve(t)
		id := s.add(t)
		s.w.Courier.Queue(notificationstest.URL("Family"), notificationstest.Reply{Status: http.StatusUnauthorized})
		var answers []string
		for _, c := range []struct{ method, path, body string }{
			{http.MethodGet, "/api/v1/notifications/connections", ""},
			{http.MethodGet, "/api/v1/notifications/connections/" + id, ""},
			{http.MethodPost, "/api/v1/notifications/connections/" + id + "/test", "{}"},
			{http.MethodPost, "/api/v1/notifications/connections/" + id + "/test", "{}"},
			{http.MethodPost, "/api/v1/notifications/connections/test", family},
			{http.MethodPatch, "/api/v1/notifications/connections/" + id, `{"name":"Home","secrets":{"token":"token-of-Home"}}`},
			{http.MethodGet, "/api/v1/notifications/providers", ""},
		} {
			_, got := s.call(t, c.method, c.path, c.body)
			answers = append(answers, got)
		}
		s.want(t, http.MethodGet, "/api/v1/notifications/connections/"+id, "", http.StatusOK,
			`{"id":"id1","name":"Home","provider":"hook","on":true,"events":["health_item_cleared","new_health_item","ping","pong"],"warnings":false,"posters":true,"node":null,`+
				`"values":{"url":"https://hook.example/Family"},"sealed":["token","signing"],"shown":{},`+
				`"status":{"shown":"ok","since":null,"until":null,"reason":"","waiting":0,"givenUp":0,"lastSent":null}}`)
		everything := strings.Join(answers, "\n") + s.Log.String() + s.w.Logs()
		for _, secret := range []string{"token-of-Family", "sig-of-Family", "token-of-Home"} {
			if strings.Contains(everything, secret) {
				t.Fatalf("%s reached an answer or a log:\n%s", secret, everything)
			}
		}
	})
}

func TestASecretLeftOutOrEmptyKeepsItsValueAndNullRemovesOnlyAnOptionalOne(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := serve(t)
		id := s.add(t)
		for _, edit := range []string{`{"name":"Home"}`, `{"secrets":{"token":"","signing":""}}`} {
			if code, got := s.call(t, http.MethodPatch, "/api/v1/notifications/connections/"+id, edit); code != http.StatusOK {
				t.Fatalf("updateNotificationConnection %s = %d %s", edit, code, got)
			}
		}
		s.want(t, http.MethodPatch, "/api/v1/notifications/connections/"+id, `{"secrets":{"token":null}}`, http.StatusUnprocessableEntity,
			`{"status":422,"code":"notification_field_refused","title":"A field of the connection can't be saved","detail":"Token can't be removed. Enter a new one instead.","field":"token"}`)
		if code, got := s.call(t, http.MethodPatch, "/api/v1/notifications/connections/"+id, `{"secrets":{"signing":null}}`); code != http.StatusOK || !strings.Contains(got, `"sealed":["token"]`) {
			t.Fatalf("removing the signing key = %d %s, want it saved without one", code, got)
		}
		if code, got := s.call(t, http.MethodPost, "/api/v1/notifications/connections/"+id+"/test", "{}"); code != http.StatusOK {
			t.Fatalf("testNotificationConnection = %d %s", code, got)
		}
		sent := s.headers(t, notificationstest.URL("Family"))
		if len(sent) != 1 || sent[0].Get("Authorization") != "Bearer token-of-Family" || sent[0].Get("X-Signing") != "" {
			t.Fatalf("the Test sent %v, want the saved token and no signing key", sent)
		}
	})
}

func TestADraftTestAndASavedTestWithOneEdit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := serve(t)
		s.want(t, http.MethodPost, "/api/v1/notifications/connections/test", family, http.StatusOK, `{"node":"Den"}`)
		if got := s.w.Courier.Received(notificationstest.URL("Family")); len(got) != 1 || got[0].Title != "Test from SlipStream" {
			t.Fatalf("the draft's Test sent %+v, want the Test message", got)
		}
		s.want(t, http.MethodGet, "/api/v1/notifications/connections", "", http.StatusOK, `{"connections":[]}`)

		id := s.add(t)
		other := notificationstest.URL("Other")
		s.w.Courier.Queue(other, notificationstest.Reply{Status: http.StatusNotFound})
		s.want(t, http.MethodPost, "/api/v1/notifications/connections/"+id+"/test", `{"values":{"url":"`+other+`"}}`, http.StatusUnprocessableEntity,
			`{"status":422,"code":"notification_test_failed","title":"The Test failed","detail":"Hook responded that this address no longer exists. Edit the connection with a new one.","field":"url"}`)
		s.want(t, http.MethodPost, "/api/v1/notifications/connections/"+id+"/test", `{"values":{"url":"`+other+`"}}`, http.StatusOK, `{"node":"Den"}`)
		if sent := s.headers(t, other); len(sent) != 2 || sent[1].Get("Authorization") != "Bearer token-of-Family" {
			t.Fatalf("the edited Test sent %v, want the saved token at the edited URL", sent)
		}
		if _, got := s.call(t, http.MethodGet, "/api/v1/notifications/connections/"+id, ""); !strings.Contains(got, `"url":"https://hook.example/Family"`) {
			t.Fatalf("a Test with an edit saved it: %s", got)
		}
	})
}

func TestSeenNeedsAnAccount(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := serve(t)
		s.want(t, http.MethodPost, "/api/v1/notifications/feed/seen", `{"through":0}`, http.StatusForbidden,
			`{"status":403,"code":"needs_account","title":"The API key can't do this. Sign in to an Account to do it."}`)
		if code, got := s.CallSignedIn(t, http.MethodPost, "/api/v1/notifications/feed/seen", `{"through":0}`); code != http.StatusNoContent {
			t.Fatalf("markNotificationsSeen signed in = %d %s", code, got)
		}
		s.want(t, http.MethodGet, "/api/v1/notifications/feed", "", http.StatusOK, `{"rows":[],"unread":null,"before":null}`)
	})
}

func TestTheStatusReasonIsWordedWhenRead(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := serve(t)
		id := s.add(t)
		s.w.Courier.Queue(notificationstest.URL("Family"), notificationstest.Reply{Status: http.StatusNotFound})
		s.w.Record(notificationstest.PingRow, notificationstest.Ping{Name: "one"})
		reason := func() string {
			t.Helper()
			_, got := s.call(t, http.MethodGet, "/api/v1/notifications/connections/"+id, "")
			_, after, _ := strings.Cut(got, `"reason":`)
			before, _, _ := strings.Cut(after, `,"waiting"`)
			return before
		}
		if got := reason(); got != `"Hook responded that this address no longer exists. Edit the connection with a new one."` {
			t.Fatalf("a paused connection's reason = %s", got)
		}
		s.w.Words[providers.Gone] = "Hook responded that this hook was deleted. Paste a new URL."
		if got := reason(); got != `"Hook responded that this hook was deleted. Paste a new URL."` {
			t.Fatalf("after a copy change the reason = %s, want the new words", got)
		}
	})
}

func TestASavedWebhookSendsEveryHeaderTypedOneALine(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := serve(t)
		code, got := s.call(t, http.MethodPost, "/api/v1/notifications/connections",
			`{"name":"Home","provider":"webhook","on":true,"secrets":{"url":"https://hooks.example/home","headers":"X-Key: one\nX-Room:  den \n"}}`)
		if code != http.StatusCreated {
			t.Fatalf("addNotificationConnection = %d %s", code, got)
		}
		if code, got := s.call(t, http.MethodPost, "/api/v1/notifications/connections/"+ownertest.FirstID(got)+"/test", "{}"); code != http.StatusOK {
			t.Fatalf("testNotificationConnection = %d %s", code, got)
		}
		sent := s.headers(t, "https://hooks.example/home")
		if len(sent) != 1 || sent[0].Get("X-Key") != "one" || sent[0].Get("X-Room") != "den" {
			t.Fatalf("the Webhook sent headers %v, want X-Key and X-Room", sent)
		}
		code, got = s.call(t, http.MethodPost, "/api/v1/notifications/connections/test", `{"name":"Home","provider":"webhook","secrets":{"url":"https://hooks.example/home","headers":"X-Key one"}}`)
		if code != http.StatusUnprocessableEntity || !strings.Contains(got, "Write each header on its own line as Name: value.") {
			t.Fatalf("a header line with no colon = %d %s, want it refused at headers", code, got)
		}
	})
}
