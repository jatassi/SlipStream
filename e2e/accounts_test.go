package e2e

import (
	"net"
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

type accountRow struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Admin       bool   `json:"admin"`
}

func accounts(t *testing.T, a *worldtest.API) []accountRow {
	t.Helper()
	var listed struct {
		Accounts []accountRow `json:"accounts"`
	}
	decode(t, "listAccounts", a.Admin("home", http.MethodGet, "/api/v1/accounts", ""), &listed)
	return listed.Accounts
}

func mailed(t *testing.T, run *world.Run) []events.Record {
	t.Helper()
	recs, err := events.Select(run.Manifest().RunDir, events.Query{Kind: "smtp.message"})
	if err != nil {
		t.Fatal(err)
	}
	return recs
}

func TestHomeMakesGraceAMemberBySignupLinkAndSavesTheMailFake(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "home")
	m := run.Manifest()
	a := api(t, run)
	if len(m.Members) != 1 {
		t.Fatalf("manifest members = %+v, want Grace alone", m.Members)
	}
	grace := m.Members[0]
	if want := (world.Member{ID: grace.ID, Email: "grace@example.test", DisplayName: "Grace Hopper", PIN: "1357"}); grace != want || grace.ID == "" {
		t.Fatalf("manifest member = %+v, want %+v with an id", grace, want)
	}
	listed := accounts(t, a)
	if len(listed) != 2 || listed[1] != (accountRow{grace.ID, "grace@example.test", "Grace Hopper", false}) {
		t.Fatalf("listAccounts = %+v, want Ada and then Grace as a Member", listed)
	}
	worldtest.Expect(t, "listAccountSessions", a.Admin("home", http.MethodGet, "/api/v1/accounts/"+grace.ID+"/sessions", ""), worldtest.Reply{Status: 200, Body: `{"sessions":[]}` + "\n"})
	var address struct {
		Address string `json:"address"`
	}
	decode(t, "getAddress", a.Admin("home", http.MethodGet, "/api/v1/identity/address", ""), &address)
	if address.Address != m.Nodes[0].BrowserURL {
		t.Fatalf("address = %s, want the Controller's browser URL %s", address.Address, m.Nodes[0].BrowserURL)
	}

	session := a.SignInAs("home", grace.Email, grace.PIN)
	worldtest.Expect(t, "getSession as Grace", a.Signed("home", session, http.MethodGet, "/api/v1/session", ""), worldtest.Reply{Status: 200,
		Body: `{"account":{"id":"` + grace.ID + `","email":"grace@example.test","displayName":"Grace Hopper","admin":false},"pinRequired":false,"keyPrompt":"ask"}` + "\n"})

	var smtpFront world.Front
	for _, f := range m.Fronts {
		if f.Target == "smtp" && f.Node == "home" {
			smtpFront = f
		}
	}
	host, port, err := net.SplitHostPort(smtpFront.Address)
	if err != nil {
		t.Fatal(err)
	}
	worldtest.Expect(t, "getMailSettings", a.Admin("home", http.MethodGet, "/api/v1/mail/settings", ""), worldtest.Reply{Status: 200,
		Body: `{"host":"` + host + `","port":` + port + `,"security":"none","username":"","passwordSet":false,"fromAddress":"slipstream@home.localhost","fromName":"SlipStream"}` + "\n"})
	if recs := mailed(t, run); len(recs) != 0 {
		t.Fatalf("the launch mailed %d messages, want none", len(recs))
	}
	worldtest.Expect(t, "sendTestEmail", a.Signed("home", a.SignIn("home"), http.MethodPost, "/api/v1/mail/test", ""), worldtest.Reply{Status: 200, Body: `{"outcome":"sent"}` + "\n"})
	recs := mailed(t, run)
	if len(recs) != 1 || recs[0].Node != "home" {
		t.Fatalf("mail capture = %+v, want one message from home", recs)
	}
}

func TestTheWorldPromotesAMemberMarkedAdmin(t *testing.T) {
	t.Parallel()
	s, err := scenario.Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	s.Name = "two-admins"
	s.Members = []scenario.Member{
		{Email: "grace@example.test", DisplayName: "Grace Hopper", PIN: "1357", Admin: true},
		{Email: "alan@example.test", DisplayName: "Alan Turing", PIN: "8642"},
	}
	run := worldtest.LaunchScenario(t, s)
	a := api(t, run)
	m := run.Manifest()
	want := []accountRow{
		{ID: accounts(t, a)[0].ID, Email: "ada@example.test", DisplayName: "Ada Okafor", Admin: true},
		{ID: m.Members[0].ID, Email: "grace@example.test", DisplayName: "Grace Hopper", Admin: true},
		{ID: m.Members[1].ID, Email: "alan@example.test", DisplayName: "Alan Turing", Admin: false},
	}
	if got := accounts(t, a); len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("listAccounts = %+v, want %+v", got, want)
	}
	if !m.Members[0].Admin || m.Members[1].Admin {
		t.Fatalf("manifest members = %+v, want Grace an Admin and Alan a Member", m.Members)
	}
	worldtest.Expect(t, "getMailSettings", a.Admin("home", http.MethodGet, "/api/v1/mail/settings", ""), worldtest.Reply{Status: 200,
		Body: `{"host":"","port":587,"security":"starttls","username":"","passwordSet":false,"fromAddress":"","fromName":"SlipStream"}` + "\n"})
}
