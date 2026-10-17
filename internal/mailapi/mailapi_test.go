package mailapi_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/smtp"
	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/identityapi"
	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/mailapi"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

type library struct {
	identitytest.Node
	mail *mail.Mail
}

func (l library) Mail() (*mail.Mail, bool) { return l.mail, true }

type node struct {
	t       *testing.T
	h       http.Handler
	smtp    faketest.Served
	apiKey  string
	session string
}

func newNode(t *testing.T) *node {
	t.Helper()
	ctx := t.Context()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), mail.Schema())
	keys, err := secret.Open(ctx, lib, filepath.Join(t.TempDir(), secret.KeyFile))
	if err != nil {
		t.Fatal(err)
	}
	svc, err := identity.New(ctx, identity.Config{Library: lib, Pepper: keys.PINPepper(), Hub: identitytest.Hub(t)})
	if err != nil {
		t.Fatal(err)
	}
	f := faketest.Loopback(t, smtp.Fake, scenario.Scenario{Name: "mail", Seed: "mail"}, faketest.As("home"))
	l := library{Node: identitytest.Node{Service: svc}, mail: mail.New(mail.Config{Library: lib, Sealer: keys.Sealer(), Timings: mail.DefaultTimings(), Roots: f.Roots})}
	log := slog.New(slog.NewTextHandler(t.Output(), nil))
	gate, code, err := identity.NewGate(ctx, l.Claim, log)
	if err != nil {
		t.Fatal(err)
	}
	r := api.NewRouter(api.Services{Node: l, Gate: gate, Web: fstest.MapFS{"index.html": {}}, Log: log})
	identityapi.Routes(r, l)
	mailapi.Routes(r, l)
	n := &node{t: t, h: r.Handler(), smtp: f}
	w := n.do(http.MethodPost, "/api/v1/setup/controller", `{"email":"ada@example.com","displayName":"Ada Okafor","pin":"2468"}`, map[string]string{"X-Setup-Code": code})
	var claimed struct {
		APIKey string `json:"apiKey"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &claimed); err != nil || w.Code != http.StatusCreated {
		t.Fatalf("claimController = %d %s", w.Code, w.Body)
	}
	n.apiKey = claimed.APIKey
	for _, c := range w.Result().Cookies() {
		if c.Name == identity.CookieName {
			n.session = c.Value
		}
	}
	return n
}

func (n *node) do(method, target, body string, headers map[string]string) *httptest.ResponseRecorder {
	n.t.Helper()
	req := httptest.NewRequestWithContext(n.t.Context(), method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	n.h.ServeHTTP(w, req)
	return w
}

func (n *node) admin(method, target, body string) *httptest.ResponseRecorder {
	return n.do(method, target, body, map[string]string{"Cookie": identity.CookieName + "=" + n.session})
}

func expect(t *testing.T, w *httptest.ResponseRecorder, status int, body string) {
	t.Helper()
	if w.Code != status || w.Body.String() != body+"\n" {
		t.Fatalf("got %d %q, want %d %q", w.Code, w.Body, status, body)
	}
}

func TestTheSMTPSettingCarriesEveryEmail(t *testing.T) {
	n := newNode(t)
	expect(t, n.admin(http.MethodGet, "/api/v1/mail/settings", ""), 200,
		`{"host":"","port":587,"security":"starttls","username":"","passwordSet":false,"fromAddress":"","fromName":"SlipStream"}`)
	expect(t, n.admin(http.MethodPost, "/api/v1/mail/test", ""), 200, `{"outcome":"not-set-up"}`)

	port := strconv.Itoa(int(n.smtp.Address.AddrPort.Port()))
	expect(t, n.admin(http.MethodPatch, "/api/v1/mail/settings", `{"host":"127.0.0.1","port":`+port+`,"username":"slipstream","password":"hunter2","fromAddress":"slipstream@media-pc.example"}`), 200,
		`{"host":"127.0.0.1","port":`+port+`,"security":"starttls","username":"slipstream","passwordSet":true,"fromAddress":"slipstream@media-pc.example","fromName":"SlipStream"}`)
	if w := n.admin(http.MethodPut, "/api/v1/identity/address", `{"address":"http://media-pc:8484"}`); w.Code != http.StatusOK {
		t.Fatalf("setAddress = %d %s", w.Code, w.Body)
	}
	expect(t, n.admin(http.MethodPost, "/api/v1/mail/test", ""), 200, `{"outcome":"sent"}`)
	var got []smtp.Message
	for _, r := range n.smtp.Records(t) {
		var m smtp.Message
		if err := json.Unmarshal(r.Data, &m); err != nil {
			t.Fatal(err)
		}
		got = append(got, m)
	}
	if len(got) != 1 || got[0].Subject != "Test email from media-pc" || got[0].To[0] != "ada@example.com" || got[0].Username != "slipstream" || got[0].Security != "starttls" {
		t.Fatalf("captured %+v, want the test email to Ada over STARTTLS signed in as slipstream", got)
	}

	n.smtp.Cue(t, smtp.Reject.Name, smtp.Toggle{})
	expect(t, n.admin(http.MethodPost, "/api/v1/mail/test", ""), 200, `{"outcome":"failed","detail":"550 5.1.1 Mailbox unavailable"}`)

	expect(t, n.do(http.MethodPost, "/api/v1/mail/test", "", map[string]string{"X-Api-Key": n.apiKey}), 422,
		`{"status":422,"code":"no_recipient","title":"The API key has no email address to send a test to. Sign in to send one."}`)
	expect(t, n.do(http.MethodPatch, "/api/v1/mail/settings", `{"password":""}`, map[string]string{"X-Api-Key": n.apiKey}), 200,
		`{"host":"127.0.0.1","port":`+port+`,"security":"starttls","username":"slipstream","passwordSet":false,"fromAddress":"slipstream@media-pc.example","fromName":"SlipStream"}`)
}
