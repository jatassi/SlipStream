package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

// firstRunView is getFirstRun's reply with each step's state, in order.
func firstRunView(open bool, steps ...string) worldtest.Reply {
	rows := make([]string, len(steps))
	for i, s := range steps {
		id, state, _ := strings.Cut(s, "=")
		rows[i] = `{"id":"` + id + `","state":"` + state + `"}`
	}
	o := "false"
	if open {
		o = "true"
	}
	return worldtest.Reply{Status: 200, Body: `{"open":` + o + `,"steps":[` + strings.Join(rows, ",") + `]}` + "\n"}
}

func TestFirstSetupRunsFromTheClaimUntilTheAdminFinishesIt(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "first-run")
	a := api(t, run)
	code := worldtest.SetupCode(t, run, "home")
	checkCode := func(code string) worldtest.Reply {
		return a.Setup("home", code, http.MethodPost, "/api/v1/setup/code", "")
	}

	worldtest.Expect(t, "checkSetupCode with a wrong code", checkCode("0000-0000-0000"), problem(403, "setup_code_wrong", "The Setup code is wrong"))
	worldtest.Expect(t, "checkSetupCode", checkCode(code), worldtest.Reply{Status: 204})
	worldtest.Expect(t, "checkSetupCode typed loosely", checkCode(strings.ToLower(strings.ReplaceAll(code, "-", " "))), worldtest.Reply{Status: 204})

	claimed := a.Setup("home", code, http.MethodPost, "/api/v1/setup/controller", `{"email":"ada@example.test","displayName":"Ada Okafor","pin":"2468"}`)
	var ada struct {
		Account struct {
			ID string `json:"id"`
		} `json:"account"`
	}
	if err := json.Unmarshal([]byte(claimed.Body), &ada); err != nil {
		t.Fatalf("claimController = %d %s", claimed.Status, claimed.Body)
	}
	adaAccount := `{"id":"` + ada.Account.ID + `","email":"ada@example.test","displayName":"Ada Okafor","admin":true}`
	worldtest.Expect(t, "claimController", claimed, worldtest.Reply{Status: 201,
		Body: `{"account":` + adaAccount + `,"pinRequired":false,"keyPrompt":"ask","firstRun":"open","passkeyPrompt":false}` + "\n"})
	worldtest.Expect(t, "checkSetupCode once claimed", checkCode(code), problem(409, "already_claimed", "This Node is already set up"))

	session := a.SignInAs("home", "ada@example.test", "2468")
	as := func(method, path, body string) worldtest.Reply { return a.Signed("home", session, method, path, body) }
	getFirstRun := func() worldtest.Reply { return as(http.MethodGet, "/api/v1/first-run", "") }
	setStep := func(id, outcome string) worldtest.Reply {
		return as(http.MethodPut, "/api/v1/first-run/steps/"+id, `{"outcome":"`+outcome+`"}`)
	}

	worldtest.Expect(t, "getFirstRun after the claim", getFirstRun(),
		firstRunView(true, "address=todo", "passkey=todo", "language=todo", "roots=todo", "downloads=todo", "targets=todo"))
	worldtest.Expect(t, "Continue on language", setStep("language", "done"),
		firstRunView(true, "address=todo", "passkey=todo", "language=done", "roots=todo", "downloads=todo", "targets=todo"))
	worldtest.Expect(t, "Skip on roots", setStep("roots", "skipped"),
		firstRunView(true, "address=todo", "passkey=todo", "language=done", "roots=skipped", "downloads=todo", "targets=todo"))
	worldtest.Expect(t, "Continue on an unknown step", setStep("trash", "done"), problem(404, "step_unknown", "No first setup step has that id"))
	worldtest.Expect(t, "getFirstRun after the step writes", getFirstRun(),
		firstRunView(true, "address=todo", "passkey=todo", "language=done", "roots=skipped", "downloads=todo", "targets=todo"))

	var nodes struct {
		Nodes []struct {
			ID string `json:"id"`
		} `json:"nodes"`
	}
	decode(t, "listNodes", as(http.MethodGet, "/api/v1/nodes", ""), &nodes)
	root, err := json.Marshal(struct {
		ContentType string `json:"contentType"`
		NodeID      string `json:"nodeId"`
		Path        string `json:"path"`
	}{"movie", nodes.Nodes[0].ID, t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if r := as(http.MethodPost, "/api/v1/roots", string(root)); r.Status != http.StatusCreated {
		t.Fatalf("createRoot = %d %s, want 201", r.Status, r.Body)
	}
	worldtest.Expect(t, "getFirstRun once the movie root exists", getFirstRun(),
		firstRunView(true, "address=todo", "passkey=todo", "language=done", "roots=done", "downloads=todo", "targets=todo"))

	var key struct {
		APIKey string `json:"apiKey"`
	}
	decode(t, "regenerateAPIKey", as(http.MethodPost, "/api/v1/api-key", ""), &key)
	keyed := func(method, path, body string) worldtest.Reply {
		return a.Keyed("home", key.APIKey, method, path, body)
	}
	worldtest.Expect(t, "Continue on downloads with the API key", keyed(http.MethodPut, "/api/v1/first-run/steps/downloads", `{"outcome":"done"}`),
		firstRunView(true, "address=todo", "passkey=todo", "language=done", "roots=done", "downloads=done", "targets=todo"))

	home := a.Node("home")
	if r := as(http.MethodPut, "/api/v1/identity/address", `{"address":"`+home.BrowserURL+`"}`); r.Status != http.StatusOK {
		t.Fatalf("setAddress = %d %s, want 200", r.Status, r.Body)
	}
	member := memberSession(t, a, as)
	needsAdmin := problem(403, "needs_admin", "Only an Admin can do this")
	worldtest.Expect(t, "getFirstRun as a Member", a.Signed("home", member, http.MethodGet, "/api/v1/first-run", ""), needsAdmin)
	worldtest.Expect(t, "setFirstRunStep as a Member", a.Signed("home", member, http.MethodPut, "/api/v1/first-run/steps/targets", `{"outcome":"done"}`), needsAdmin)
	worldtest.Expect(t, "finishFirstRun as a Member", a.Signed("home", member, http.MethodPost, "/api/v1/first-run/finish", ""), needsAdmin)

	act(t, run, "home", world.RestartNode)
	opened := firstRunView(true, "address=done", "passkey=todo", "language=done", "roots=done", "downloads=done", "targets=todo")
	worldtest.Expect(t, "getFirstRun after a restart", getFirstRun(), opened)
	worldtest.Expect(t, "getFirstRun with the API key", keyed(http.MethodGet, "/api/v1/first-run", ""), opened)

	worldtest.Expect(t, "finishFirstRun", as(http.MethodPost, "/api/v1/first-run/finish", ""), worldtest.Reply{Status: 204})
	worldtest.Expect(t, "finishFirstRun again with the API key", keyed(http.MethodPost, "/api/v1/first-run/finish", ""), worldtest.Reply{Status: 204})
	worldtest.Expect(t, "getFirstRun once finished", getFirstRun(),
		firstRunView(false, "address=done", "passkey=skipped", "language=done", "roots=done", "downloads=done", "targets=skipped"))
	worldtest.Expect(t, "Continue once finished", setStep("targets", "done"), problem(409, "first_run_finished", "First setup is already finished"))
	worldtest.Expect(t, "getSession once finished", as(http.MethodGet, "/api/v1/session", ""),
		worldtest.Reply{Status: 200, Body: `{"account":` + adaAccount + `,"pinRequired":false,"keyPrompt":"ask","firstRun":"finished"}` + "\n"})
}

// memberSession signs in a Member made through a Signup link that the
// Admin of as issues.
func memberSession(t *testing.T, a *worldtest.API, as func(method, path, body string) worldtest.Reply) string {
	t.Helper()
	r := as(http.MethodPost, "/api/v1/signup-links", `{"email":"grace@example.test"}`)
	var issued struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(r.Body), &issued); err != nil || r.Status != http.StatusCreated {
		t.Fatalf("createSignupLink = %d %s, want 201", r.Status, r.Body)
	}
	_, token, _ := strings.Cut(issued.URL, "#token=")
	if r := a.Anyone("home", http.MethodPost, "/api/v1/signup-links/by-token/"+token+"/account", `{"displayName":"Grace Hopper","pin":"1357"}`); r.Status != http.StatusOK {
		t.Fatalf("redeemSignupLink = %d %s, want 200", r.Status, r.Body)
	}
	session := a.SignInAs("home", "grace@example.test", "1357")
	var got struct {
		FirstRun *string `json:"firstRun"`
	}
	decode(t, "getSession as a Member", a.Signed("home", session, http.MethodGet, "/api/v1/session", ""), &got)
	if got.FirstRun != nil {
		t.Fatalf("a Member's session carries firstRun %q", *got.FirstRun)
	}
	return session
}
