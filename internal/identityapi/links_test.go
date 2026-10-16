package identityapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/mail"
)

const address = "https://slipstream.example.com"

var (
	linkDead      = problem(410, "link_dead", "This link no longer works")
	addressUnset  = problem(409, "address_unset", "Set SlipStream's address first, so links point somewhere people can reach")
	accountExists = problem(409, "account_exists", "An Account already has that email")
	emailTaken    = problem(409, "email_taken", "Another Account has that email")
	linkNotFound  = problem(404, "link_not_found", "No pending Signup link has that id")
	unknownLink   = `{"state":"dead","reason":"unknown"}` + "\n"
	tokenURL      = regexp.MustCompile(`^https://slipstream\.example\.com/(signup|reset|confirm-email)#token=([A-Za-z0-9_-]{43})$`)
)

func (n *node) as(session, method, target, body string) *httptest.ResponseRecorder {
	n.t.Helper()
	return n.do(call{method: method, target: target, session: session, body: body})
}

func (n *node) setAddress(session string) {
	n.t.Helper()
	if w := n.as(session, http.MethodPut, "/api/v1/identity/address", `{"address":"`+address+`"}`); w.Code != http.StatusOK {
		n.t.Fatalf("setAddress = %d %s", w.Code, w.Body)
	}
}

type issuedLink struct {
	Link struct {
		ID        string    `json:"id"`
		Email     string    `json:"email"`
		IssuedBy  string    `json:"issuedBy"`
		IssuedAt  time.Time `json:"issuedAt"`
		ExpiresAt time.Time `json:"expiresAt"`
	} `json:"link"`
	URL   string `json:"url"`
	Email struct {
		Outcome string `json:"outcome"`
		Detail  string `json:"detail"`
	} `json:"email"`
}

func decode[T any](t *testing.T, w *httptest.ResponseRecorder, status int) T {
	t.Helper()
	var v T
	if w.Code != status {
		t.Fatalf("got %d %s, want %d", w.Code, w.Body, status)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

// tokenOf is the token in a link's URL, failing t unless the URL is one on
// the configured address.
func tokenOf(t *testing.T, url string) string {
	t.Helper()
	m := tokenURL.FindStringSubmatch(url)
	if m == nil {
		t.Fatalf("url %q is no link on %s", url, address)
	}
	return m[2]
}

func (n *node) issueSignup(session, email string) issuedLink {
	n.t.Helper()
	return decode[issuedLink](n.t, n.as(session, http.MethodPost, "/api/v1/signup-links", `{"email":"`+email+`"}`), http.StatusCreated)
}

func (n *node) join(token string) *httptest.ResponseRecorder {
	n.t.Helper()
	return n.do(call{method: http.MethodPost, target: "/api/v1/signup-links/by-token/" + token + "/account", body: `{"displayName":"Grace Hopper","pin":"1357"}`})
}

// grace joins Grace, grace@example.com with PIN 1357, through a real
// Signup link the Admin issues, and returns her id and session.
func (n *node) grace(admin string) (id, session string) {
	n.t.Helper()
	w := n.join(tokenOf(n.t, n.issueSignup(admin, "grace@example.com").URL))
	got := decode[struct {
		Account struct {
			ID string `json:"id"`
		} `json:"account"`
	}](n.t, w, http.StatusOK)
	n.mail.Wait(n.t, "grace@example.com")
	return got.Account.ID, sessionOf(n.t, w)
}

func view(n *node, kind, token string) string {
	n.t.Helper()
	w := n.do(call{method: http.MethodGet, target: "/api/v1/" + kind + "-links/by-token/" + token})
	if w.Code != http.StatusOK {
		n.t.Fatalf("GET %s link = %d %s", kind, w.Code, w.Body)
	}
	return w.Body.String()
}

func dead(reason string, issued, expires time.Time) string {
	return `{"state":"dead","reason":"` + reason + `","adminName":"Admin","issuedAt":"` + issued.UTC().Format(time.RFC3339Nano) + `","expiresAt":"` + expires.UTC().Format(time.RFC3339Nano) + `"}` + "\n"
}

func TestASignupLinksLife(t *testing.T) {
	n := newNode(t)
	_, a := n.claimed()
	expect(t, n.as(a, http.MethodPost, "/api/v1/signup-links", `{"email":"grace@example.com"}`), 409, addressUnset)
	n.setAddress(a)

	first := n.issueSignup(a, " Grace@Example.com ")
	if first.Link.Email != "grace@example.com" || first.Link.IssuedBy != "Admin" || first.Email.Outcome != "sent" ||
		!first.Link.ExpiresAt.Equal(first.Link.IssuedAt.Add(7*24*time.Hour)) {
		t.Fatalf("issued %+v", first)
	}
	firstToken := tokenOf(t, first.URL)
	invite := n.mail.Wait(t, "grace@example.com")
	if invite.Subject != "Admin invited you to slipstream.example.com" || !strings.Contains(invite.Text, "Make your Account: "+first.URL+"\n") {
		t.Fatalf("the Signup email = %q %q", invite.Subject, invite.Text)
	}
	expect(t, n.as(a, http.MethodGet, "/api/v1/signup-links/"+first.Link.ID, ""), 200,
		`{"link":{"id":"`+first.Link.ID+`","email":"grace@example.com","issuedBy":"Admin","issuedAt":"`+first.Link.IssuedAt.Format(time.RFC3339Nano)+
			`","expiresAt":"`+first.Link.ExpiresAt.Format(time.RFC3339Nano)+`"},"url":"`+first.URL+`"}`+"\n")
	expect(t, n.do(call{method: http.MethodGet, target: "/api/v1/signup-links/by-token/" + firstToken}), 200,
		`{"state":"live","adminName":"Admin","email":"grace@example.com","serverName":"slipstream.example.com","expiresAt":"`+first.Link.ExpiresAt.Format(time.RFC3339Nano)+`"}`+"\n")

	second := decode[issuedLink](t, n.as(a, http.MethodPost, "/api/v1/signup-links/"+first.Link.ID+"/resend", ""), http.StatusCreated)
	resent := n.mail.Wait(t, "grace@example.com")
	if !strings.Contains(resent.Text, "This replaces the link sent on ") {
		t.Fatalf("the resent email doesn't say it replaces the first:\n%s", resent.Text)
	}
	if got := view(n, "signup", firstToken); got != dead("withdrawn", first.Link.IssuedAt, first.Link.ExpiresAt) {
		t.Fatalf("the replaced link = %s", got)
	}
	expect(t, n.join(firstToken), 410, linkDead)
	listed := decode[struct {
		Links []struct {
			ID string `json:"id"`
		} `json:"links"`
	}](t, n.as(a, http.MethodGet, "/api/v1/signup-links", ""), 200)
	if len(listed.Links) != 1 || listed.Links[0].ID != second.Link.ID {
		t.Fatalf("pending links %+v, want only the resent one %s", listed.Links, second.Link.ID)
	}

	secondToken := tokenOf(t, second.URL)
	w := n.join(secondToken)
	got := decode[struct {
		Account struct {
			ID    string `json:"id"`
			Email string `json:"email"`
			Admin bool   `json:"admin"`
		} `json:"account"`
		PINRequired   bool `json:"pinRequired"`
		PasskeyPrompt bool `json:"passkeyPrompt"`
	}](t, w, http.StatusOK)
	if got.Account.Email != "grace@example.com" || got.Account.Admin || got.PINRequired || got.PasskeyPrompt {
		t.Fatalf("joined %+v", got)
	}
	expect(t, n.getSession(sessionOf(t, w)), 200,
		`{"account":{"id":"`+got.Account.ID+`","email":"grace@example.com","displayName":"Grace Hopper","admin":false},"pinRequired":false,"keyPrompt":"ask"}`+"\n")
	if w := n.signIn("grace@example.com", "1357"); w.Code != http.StatusOK {
		t.Fatalf("Grace's own PIN = %d %s", w.Code, w.Body)
	}
	expect(t, n.join(secondToken), 410, linkDead)
	if got := view(n, "signup", secondToken); !strings.HasPrefix(got, `{"state":"dead","reason":"used","adminName":"Admin",`) {
		t.Fatalf("the used link = %s", got)
	}
	expect(t, n.as(a, http.MethodPost, "/api/v1/signup-links", `{"email":"grace@example.com"}`), 409, accountExists)
	expect(t, n.as(a, http.MethodGet, "/api/v1/signup-links", ""), 200, `{"links":[]}`+"\n")
	expect(t, n.do(call{method: http.MethodGet, target: "/api/v1/signup-links/by-token/made-up"}), 200, unknownLink)
	expect(t, n.do(call{method: http.MethodGet, target: "/api/v1/reset-links/by-token/" + secondToken}), 200, unknownLink)
}

func TestACanceledLinkIsWithdrawn(t *testing.T) {
	n := newNode(t)
	res, a := n.claimed()
	n.setAddress(a)
	n.mail.Answer(mail.Outcome{Kind: mail.Failed, Reply: "550 5.1.1 Mailbox unavailable"})
	issued := decode[issuedLink](t, n.do(call{method: http.MethodPost, target: "/api/v1/signup-links", body: `{"email":"hopper@example.com"}`,
		headers: map[string]string{"X-Api-Key": res.APIKey}}), http.StatusCreated)
	if issued.Link.IssuedBy != "" || issued.Email.Outcome != "failed" || issued.Email.Detail != "550 5.1.1 Mailbox unavailable" {
		t.Fatalf("issued by the API key %+v", issued)
	}
	if m := n.mail.Wait(t, "hopper@example.com"); m.Subject != "Your Admin invited you to slipstream.example.com" {
		t.Fatalf("the Signup email from the API key = %q", m.Subject)
	}
	expect(t, n.as(a, http.MethodDelete, "/api/v1/signup-links/"+issued.Link.ID, ""), 204, "")
	if got := view(n, "signup", tokenOf(t, issued.URL)); !strings.HasPrefix(got, `{"state":"dead","reason":"withdrawn","issuedAt":`) {
		t.Fatalf("the canceled link = %s, want withdrawn naming nobody", got)
	}
	expect(t, n.as(a, http.MethodDelete, "/api/v1/signup-links/"+issued.Link.ID, ""), 404, linkNotFound)
	expect(t, n.as(a, http.MethodGet, "/api/v1/signup-links/"+issued.Link.ID, ""), 404, linkNotFound)
}

func TestASignupLinkExpiresAtItsLifetime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t)
		_, a := n.claimed()
		n.setAddress(a)
		if w := n.as(a, http.MethodPatch, "/api/v1/identity/settings", `{"signupLinkLifetime":60}`); w.Code != http.StatusOK {
			t.Fatalf("updateIdentitySettings = %d %s", w.Code, w.Body)
		}
		issued := n.issueSignup(a, "grace@example.com")
		token := tokenOf(t, issued.URL)
		time.Sleep(59 * time.Second)
		if got := view(n, "signup", token); !strings.HasPrefix(got, `{"state":"live"`) {
			t.Fatalf("a second before it expires = %s", got)
		}
		time.Sleep(time.Second)
		if got, want := view(n, "signup", token), dead("expired", issued.Link.IssuedAt, issued.Link.IssuedAt.Add(time.Minute)); got != want {
			t.Fatalf("at its lifetime = %s, want %s", got, want)
		}
		expect(t, n.join(token), 410, linkDead)
		expect(t, n.as(a, http.MethodPost, "/api/v1/signup-links/"+issued.Link.ID+"/resend", ""), 404, linkNotFound)
		if fresh := n.issueSignup(a, "grace@example.com"); fresh.Link.ID == issued.Link.ID {
			t.Fatal("a new link reused the expired one")
		}
	})
}

func TestTwoTabsRacingOneLinkMakeOneAccount(t *testing.T) {
	n := newNode(t)
	_, a := n.claimed()
	n.setAddress(a)
	token := tokenOf(t, n.issueSignup(a, "grace@example.com").URL)
	var wg sync.WaitGroup
	codes := make([]int, 2)
	bodies := make([]string, 2)
	for i := range codes {
		wg.Go(func() {
			w := n.join(token)
			codes[i], bodies[i] = w.Code, w.Body.String()
		})
	}
	wg.Wait()
	won, lost := 0, 1
	if codes[1] == 200 {
		won, lost = 1, 0
	}
	if codes[won] != 200 || codes[lost] != 410 || bodies[lost] != linkDead {
		t.Fatalf("racing redeems = %v %v, want one 200 and one 410 link_dead", codes, bodies)
	}
	if got := view(n, "signup", token); !strings.HasPrefix(got, `{"state":"dead","reason":"used"`) {
		t.Fatalf("the raced link = %s", got)
	}
}

func TestAResetLinkEndsSessionsWhenIssuedAndWhenUsed(t *testing.T) {
	n := newNode(t)
	_, a := n.claimed()
	n.setAddress(a)
	grace, before := n.grace(a)
	expect(t, n.as(a, http.MethodPost, "/api/v1/accounts/00000000-0000-0000-0000-000000000000/reset-link", "{}"), 404,
		problem(404, "account_not_found", "No Account has that id"))

	for range 5 {
		n.signInFrom("192.0.2.9:1", "grace@example.com", "0000", nil)
	}
	issued := decode[struct {
		URL   string `json:"url"`
		Email struct {
			Outcome string `json:"outcome"`
		} `json:"email"`
	}](t, n.as(a, http.MethodPost, "/api/v1/accounts/"+grace+"/reset-link", `{}`), http.StatusCreated)
	if issued.Email.Outcome != "sent" {
		t.Fatalf("issued %+v", issued)
	}
	if m := n.mail.Wait(t, "grace@example.com"); m.Subject != "Reset your PIN for slipstream.example.com" || !strings.Contains(m.Text, "Admin sent you a link to set a new PIN.") {
		t.Fatalf("the Reset email = %q %q", m.Subject, m.Text)
	}
	expect(t, n.getSession(before), 401, signedOut)
	token := tokenOf(t, issued.URL)
	if got := view(n, "reset", token); !strings.HasPrefix(got, `{"state":"live","adminName":"Admin","email":"grace@example.com","serverName":"slipstream.example.com","expiresAt":`) {
		t.Fatalf("the Reset link = %s", got)
	}
	w := n.do(call{method: http.MethodPost, target: "/api/v1/reset-links/by-token/" + token + "/pin", body: `{"pin":"8642"}`})
	if w.Code != http.StatusOK {
		t.Fatalf("redeemResetLink = %d %s", w.Code, w.Body)
	}
	after := sessionOf(t, w)
	if s := n.getSession(after); s.Code != http.StatusOK {
		t.Fatalf("the Reset link's session = %d %s", s.Code, s.Body)
	}
	expect(t, n.signIn("grace@example.com", "1357"), 401, pinWrong)
	if w := n.signIn("grace@example.com", "8642"); w.Code != http.StatusOK {
		t.Fatalf("the new PIN, straight after five wrong ones = %d %s", w.Code, w.Body)
	}
	expect(t, n.do(call{method: http.MethodPost, target: "/api/v1/reset-links/by-token/" + token + "/pin", body: `{"pin":"1111"}`}), 410, linkDead)
	expect(t, n.do(call{method: http.MethodPost, target: "/api/v1/signup-links/by-token/" + token + "/account", body: `{"displayName":"X","pin":"1111"}`}), 410, linkDead)
}

func TestAResetLinkNeedsTheAddress(t *testing.T) {
	n := newNode(t)
	_, a := n.claimed()
	grace := identitytest.Member(t, n.lib, "grace@example.com")
	expect(t, n.as(a, http.MethodPost, "/api/v1/accounts/"+grace.String()+"/reset-link", "{}"), 409, addressUnset)
}

func TestUsingAResetLinkEndsTheSessionsSinceItWasIssued(t *testing.T) {
	n := newNode(t)
	_, a := n.claimed()
	n.setAddress(a)
	grace, _ := n.grace(a)
	issued := decode[struct {
		URL string `json:"url"`
	}](t, n.as(a, http.MethodPost, "/api/v1/accounts/"+grace+"/reset-link", `{}`), http.StatusCreated)
	since := sessionOf(t, n.signIn("grace@example.com", "1357"))
	if w := n.do(call{method: http.MethodPost, target: "/api/v1/reset-links/by-token/" + tokenOf(t, issued.URL) + "/pin", body: `{"pin":"8642"}`}); w.Code != http.StatusOK {
		t.Fatalf("redeemResetLink = %d %s", w.Code, w.Body)
	}
	expect(t, n.getSession(since), 401, signedOut)
}

func TestAnEmailChangeIsConfirmedByALinkToTheNewAddress(t *testing.T) {
	n := newNode(t)
	_, a := n.claimed()
	expect(t, n.as(a, http.MethodPost, "/api/v1/account/email", `{"email":"ada@hopper.example"}`), 409, addressUnset)
	n.setAddress(a)
	grace, g := n.grace(a)
	expect(t, n.as(g, http.MethodPost, "/api/v1/account/email", `{"email":"admin@example.com"}`), 409, emailTaken)
	expect(t, n.as(g, http.MethodPost, "/api/v1/account/email", `{"email":"Grace@Hopper.example"}`), 200, `{"email":{"outcome":"sent"}}`+"\n")
	confirm := n.mail.Wait(t, "grace@hopper.example")
	if confirm.Subject != "Confirm your new email for slipstream.example.com" {
		t.Fatalf("the confirm email = %q", confirm.Subject)
	}
	link := regexp.MustCompile(`Confirm this email: (\S+)`).FindStringSubmatch(confirm.Text)
	if link == nil {
		t.Fatalf("no confirm link in:\n%s", confirm.Text)
	}
	token := tokenOf(t, link[1])
	if got := view(n, "email", token); !strings.HasPrefix(got, `{"state":"live","email":"grace@hopper.example","serverName":"slipstream.example.com",`) {
		t.Fatalf("the confirm link = %s", got)
	}
	expect(t, n.getSession(g), 200, `{"account":{"id":"`+grace+`","email":"grace@example.com","displayName":"Grace Hopper","admin":false},"pinRequired":false,"keyPrompt":"ask"}`+"\n")
	expect(t, n.do(call{method: http.MethodPost, target: "/api/v1/email-links/by-token/" + token + "/confirm"}), 204, "")
	expect(t, n.getSession(g), 200, `{"account":{"id":"`+grace+`","email":"grace@hopper.example","displayName":"Grace Hopper","admin":false},"pinRequired":false,"keyPrompt":"ask"}`+"\n")
	told := n.mail.Wait(t, "grace@example.com")
	if told.Subject != "Your SlipStream email was changed" || !strings.HasPrefix(told.Text, "Your SlipStream email was changed to grace@hopper.example.") {
		t.Fatalf("the old address was told %q %q", told.Subject, told.Text)
	}
	expect(t, n.do(call{method: http.MethodPost, target: "/api/v1/email-links/by-token/" + token + "/confirm"}), 410, linkDead)
	if w := n.signIn("grace@hopper.example", "1357"); w.Code != http.StatusOK {
		t.Fatalf("signing in with the new email = %d %s", w.Code, w.Body)
	}
}
