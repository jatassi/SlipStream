package worldtest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/internal/nodeapi"
	"github.com/jatassi/slipstream/internal/identity"
)

// API calls a world's Nodes the way a script does, one request each over
// one client, with the run's API key on Admin calls. Both tiers use it:
// the end-to-end tier over the OS network, the simulation tier over its
// in-memory one. On a run t launched, each call journals as an
// exec -- curl step whose tokens name the run's values.
type API struct {
	ctx     context.Context
	t       *testing.T
	client  *http.Client
	m       world.Manifest
	journal *world.Run
}

func NewAPI(ctx context.Context, t *testing.T, client *http.Client, m world.Manifest) *API {
	a := &API{ctx: ctx, t: t, client: client, m: m}
	for _, run := range launchedBy(t) {
		if run.Manifest().RunDir == m.RunDir {
			a.journal = run
		}
	}
	return a
}

// request is one call to a Node.
type request struct {
	node, method, path string
	body               string
	// bodyArg is body as the journal writes it.
	bodyArg string
	headers []header
}

type header struct {
	name, value string
	// arg is value as the journal writes it.
	arg string
}

func apiKey(key string) header { return header{"X-Api-Key", key, "{apiKey}"} }

func cookie(session string) header {
	v := identity.CookieName + "=" + session
	return header{"Cookie", v, world.Literal(v)}
}

// curl is q as the command that sends it from a shell.
func (q request) curl() []string {
	args := []string{"curl", "-sS"}
	if q.method != http.MethodGet {
		args = append(args, "-X", q.method)
	}
	for _, h := range q.headers {
		args = append(args, "-H", h.name+": "+h.arg)
	}
	if q.body != "" {
		args = append(args, "-H", "Content-Type: application/json", "--data-binary", q.bodyArg)
	}
	return append(args, "{nodes."+q.node+".url}"+world.Literal(q.path))
}

// step journals q, unless the API's run is not one t launched or q is an
// Until's polling, and returns what ends the step.
func (a *API) step(q request) func(exit int, stdout []byte) {
	a.t.Helper()
	if a.journal == nil || waiting(a.t) {
		return func(int, []byte) {}
	}
	return startStep(a.t, a.journal, world.StepStart{Verb: "exec", Args: append([]string{"--"}, q.curl()...)})
}

type Reply struct {
	Status int
	Body   string
}

func (a *API) Node(name string) world.Node {
	a.t.Helper()
	i := slices.IndexFunc(a.m.Nodes, func(n world.Node) bool { return n.Name == name })
	if i < 0 {
		a.t.Fatalf("the manifest has no node %s", name)
	}
	return a.m.Nodes[i]
}

// Get calls path on the Node named name with no credential.
func (a *API) Get(name, path string) Reply {
	a.t.Helper()
	r, _ := a.do(request{node: name, method: http.MethodGet, path: path})
	return r
}

// Admin calls path on the Node named name with the run's API key, sending
// body as JSON unless it is empty.
func (a *API) Admin(name, method, path, body string) Reply {
	a.t.Helper()
	r, _ := a.do(request{node: name, method: method, path: path, body: body, bodyArg: world.Literal(body), headers: []header{apiKey(a.m.APIKey)}})
	return r
}

// SignIn signs in to the Node named name as the run's Admin, by email and
// PIN, and returns the session.
func (a *API) SignIn(name string) string {
	a.t.Helper()
	if a.m.Admin == nil {
		a.t.Fatal("the run has no Admin to sign in as")
	}
	return a.signIn(name, a.m.Admin.Email, a.m.Admin.PIN, `{"email":"{admin.email}","pin":"{admin.pin}"}`)
}

// SignInAs signs in to the Node named name by email and PIN, and returns
// the session.
func (a *API) SignInAs(name, email, pin string) string {
	a.t.Helper()
	return a.signIn(name, email, pin, "")
}

func (a *API) signIn(name, email, pin, bodyArg string) string {
	a.t.Helper()
	body, err := json.Marshal(struct {
		Email string `json:"email"`
		PIN   string `json:"pin"`
	}{email, pin})
	if err != nil {
		a.t.Fatal(err)
	}
	if bodyArg == "" {
		bodyArg = world.Literal(string(body))
	}
	reply, cookies := a.do(request{node: name, method: http.MethodPost, path: "/api/v1/session", body: string(body), bodyArg: bodyArg})
	for _, c := range cookies {
		if c.Name == identity.CookieName && reply.Status == http.StatusOK {
			return c.Value
		}
	}
	a.t.Fatalf("signIn as %s = %d %s, want 200 with a session cookie", email, reply.Status, reply.Body)
	return ""
}

// Signed calls path on the Node named name in session, sending body as
// JSON unless it is empty.
func (a *API) Signed(name, session, method, path, body string) Reply {
	a.t.Helper()
	r, _ := a.do(request{node: name, method: method, path: path, body: body, bodyArg: world.Literal(body), headers: []header{cookie(session)}})
	return r
}

func (a *API) send(q request) *http.Response {
	a.t.Helper()
	var r io.Reader = http.NoBody
	if q.body != "" {
		r = strings.NewReader(q.body)
	}
	req, err := http.NewRequestWithContext(a.ctx, q.method, a.Node(q.node).URL+q.path, r)
	if err != nil {
		a.t.Fatal(err)
	}
	if q.body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, h := range q.headers {
		req.Header.Set(h.name, h.value)
	}
	res, err := a.client.Do(req)
	if err != nil {
		a.t.Fatal(err)
	}
	return res
}

func (a *API) do(q request) (Reply, []*http.Cookie) {
	a.t.Helper()
	end := a.step(q)
	res := a.send(q)
	defer func() { _ = res.Body.Close() }()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		a.t.Fatal(err)
	}
	end(0, data)
	return Reply{res.StatusCode, string(data)}, res.Cookies()
}

func Expect(t *testing.T, what string, got, want Reply) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %d %q, want %d %q", what, got.Status, got.Body, want.Status, want.Body)
	}
}

// Row is a listNodes row, as much of it as the tiers assert.
type Row struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Self          bool               `json:"self"`
	Key           string             `json:"key"`
	Addresses     []string           `json:"addresses"`
	Roles         []nodeapi.RoleView `json:"roles"`
	IntentVersion string             `json:"intentVersion"`
	Link          struct {
		State string `json:"state"`
		Since string `json:"since"`
		Build *struct {
			Version string `json:"version"`
			OS      string `json:"os"`
			Arch    string `json:"arch"`
		} `json:"build"`
	} `json:"link"`
	Facts *struct {
		Assigned       []string `json:"assigned"`
		Running        []string `json:"running"`
		Restarting     []string `json:"restarting"`
		AppliedVersion string   `json:"appliedVersion"`
	} `json:"facts"`
}

// Assigned is the Roles ticked on the Node.
func (r Row) Assigned() []string { return nodeapi.Assigned(r.Roles) }

// Nodes is listNodes on the Node named name.
func (a *API) Nodes(name string) []Row {
	a.t.Helper()
	var list struct {
		Nodes []Row `json:"nodes"`
	}
	a.decode(a.Admin(name, http.MethodGet, "/api/v1/nodes", ""), http.StatusOK, &list)
	return list.Nodes
}

// Outcome is a setRoles answer, as much of it as the tiers assert.
type Outcome struct {
	Basis   string `json:"basis"`
	Effects []struct {
		Kind  string `json:"kind"`
		Words string `json:"words"`
	} `json:"effects"`
}

// Words is what each of o's effects says.
func (o Outcome) Words() []string {
	out := []string{}
	for _, e := range o.Effects {
		out = append(out, e.Words)
	}
	return out
}

// SetRoles gives the Node with id roles through setRoles on the Node named
// name, on the basis getRoleBoard reads first.
func (a *API) SetRoles(name, id string, roles ...string) Outcome {
	a.t.Helper()
	var board struct {
		Basis string `json:"basis"`
	}
	a.decode(a.Admin(name, http.MethodGet, "/api/v1/nodes/roles", ""), http.StatusOK, &board)
	body, err := json.Marshal(struct {
		Basis string               `json:"basis"`
		Set   []nodeapi.Assignment `json:"set"`
	}{board.Basis, []nodeapi.Assignment{{NodeID: id, Roles: append([]string{}, roles...)}}})
	if err != nil {
		a.t.Fatal(err)
	}
	var out Outcome
	a.decode(a.Admin(name, http.MethodPut, "/api/v1/nodes/roles", string(body)), http.StatusOK, &out)
	return out
}

// Join joins the Unclaimed Node named name as a Worker of the Controller
// named controller, through the Node's setupCode and a Join Code the
// Controller mints, and returns the id the Controller gave it.
func (a *API) Join(controller, name, setupCode string) string {
	a.t.Helper()
	var minted struct {
		Code string `json:"code"`
	}
	a.decode(a.Admin(controller, http.MethodPost, "/api/v1/nodes/join-codes", ""), http.StatusCreated, &minted)
	body := `{"code":"` + minted.Code + `"}`
	reply, _ := a.do(request{
		node: name, method: http.MethodPost, path: "/api/v1/setup/worker", body: body, bodyArg: world.Literal(body),
		headers: []header{{"X-Setup-Code", setupCode, world.Literal(setupCode)}},
	})
	var joined struct {
		NodeID string `json:"nodeId"`
	}
	a.decode(reply, http.StatusCreated, &joined)
	return joined.NodeID
}

func (a *API) decode(r Reply, status int, v any) {
	a.t.Helper()
	if r.Status != status {
		a.t.Fatalf("got %d %s, want %d", r.Status, r.Body, status)
	}
	if err := json.Unmarshal([]byte(r.Body), v); err != nil {
		a.t.Fatalf("%s: %v", r.Body, err)
	}
}

var (
	homeRoles    = []string{"storage", "converting", "subtitles", "metadata", "media-server-link", "notifying", "serving"}
	seedboxRoles = []string{"searching", "downloading"}
)

// CheckHome makes the home Scenario's assertions, which every tier that
// runs it shares: seedbox joined as a Worker on 127.0.0.3 holding only
// local.db, both Nodes listed on home with their Roles applied, and a
// setRoles that answers once seedbox holds the new Roles and home holds
// the Downloading seedbox let go.
func CheckHome(t *testing.T, a *API) {
	t.Helper()
	box := a.Node("seedbox")
	if link := netip.MustParseAddrPort(box.Listeners["link"]).Addr(); box.Claim != world.ClaimWorker || box.Setup != scenario.SetupJoin || link != netip.MustParseAddr("127.0.0.3") {
		t.Fatalf("the manifest has seedbox with claim %s, setup %s and listeners.link on %s; want worker, join and 127.0.0.3", box.Claim, box.Setup, link)
	}
	Expect(t, "seedbox GET /api/v1/status", a.Get("seedbox", "/api/v1/status"), Reply{200, `{"claim":"worker","passkeys":{"available":false,"reason":"ip_address"}}` + "\n"})
	Expect(t, "seedbox GET /api/v1/system", a.Admin("seedbox", http.MethodGet, "/api/v1/system", ""),
		Reply{409, `{"status":409,"code":"worker_node","title":"This Node is a Worker. Manage it from its Controller."}` + "\n"})

	rows := a.Nodes("home")
	if len(rows) != 2 {
		t.Fatalf("listNodes on home has %d rows, want home and seedbox", len(rows))
	}
	byName := map[string]Row{rows[0].Name: rows[0], rows[1].Name: rows[1]}
	home, seedbox := byName["home"], byName["seedbox"]
	checkRow(t, home, "local", homeRoles)
	checkRow(t, seedbox, "up", seedboxRoles)
	if seedbox.Link.Build == nil || home.Link.Build == nil || *seedbox.Link.Build != *home.Link.Build {
		t.Fatalf("seedbox's build %+v is not home's %+v", seedbox.Link.Build, home.Link.Build)
	}
	if seedbox.ID != box.NodeID || !slices.Equal(box.Roles, seedboxRoles) || box.NodeKey != seedbox.Key {
		t.Fatalf("the manifest has seedbox as %s with roles %v and key %q; want %s with %v and key %q", box.NodeID, box.Roles, box.NodeKey, seedbox.ID, seedboxRoles, seedbox.Key)
	}

	set := a.SetRoles("home", seedbox.ID, "searching")
	if want := []string{"seedbox stops Downloading.", "home takes over Downloading, since SlipStream always needs a Node that downloads."}; !slices.Equal(set.Words(), want) {
		t.Fatalf("setRoles(seedbox, [searching]) answered %q, want %q", set.Words(), want)
	}
	rows = a.Nodes("home")
	byName = map[string]Row{rows[0].Name: rows[0], rows[1].Name: rows[1]}
	checkRow(t, byName["seedbox"], "up", []string{"searching"})
	checkRow(t, byName["home"], "local", []string{"downloading", "storage", "converting", "subtitles", "metadata", "media-server-link", "notifying", "serving"})

	for name, want := range map[string]bool{"node.key": true, "pairing.json": true, "local.db": true, "library.db": false} {
		if _, err := os.Stat(filepath.Join(box.DataDir, name)); (err == nil) != want {
			t.Errorf("seedbox's data dir holds %s: %v, want %v", name, err == nil, want)
		}
	}
}

func checkRow(t *testing.T, r Row, link string, roles []string) {
	t.Helper()
	if r.Link.State != link || !slices.Equal(r.Assigned(), roles) || r.Facts == nil || !slices.Equal(r.Facts.Assigned, roles) || r.Facts.AppliedVersion != r.IntentVersion {
		t.Fatalf("listNodes row %s: link %s, roles %v, facts %+v at intentVersion %s; want link %s with %v applied", r.Name, r.Link.State, r.Roles, r.Facts, r.IntentVersion, link, roles)
	}
}
