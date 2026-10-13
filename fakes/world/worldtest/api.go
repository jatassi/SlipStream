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
)

// API calls a world's Nodes the way a script does, one request each over
// one client, with the run's API key on Admin calls. Both tiers use it:
// the end-to-end tier over the OS network, the simulation tier over its
// in-memory one.
type API struct {
	ctx    context.Context
	t      *testing.T
	client *http.Client
	m      world.Manifest
}

func NewAPI(ctx context.Context, t *testing.T, client *http.Client, m world.Manifest) *API {
	return &API{ctx: ctx, t: t, client: client, m: m}
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
	return a.do(name, http.MethodGet, path, "", "")
}

// Admin calls path on the Node named name with the run's API key, sending
// body as JSON unless it is empty.
func (a *API) Admin(name, method, path, body string) Reply {
	a.t.Helper()
	return a.do(name, method, path, body, a.m.APIKey)
}

func (a *API) do(name, method, path, body, apiKey string) Reply {
	a.t.Helper()
	var r io.Reader = http.NoBody
	if body != "" {
		r = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(a.ctx, method, a.Node(name).URL+path, r)
	if err != nil {
		a.t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if apiKey != "" {
		req.Header.Set("X-Api-Key", apiKey)
	}
	res, err := a.client.Do(req)
	if err != nil {
		a.t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		a.t.Fatal(err)
	}
	return Reply{res.StatusCode, string(data)}
}

func Expect(t *testing.T, what string, got, want Reply) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %d %q, want %d %q", what, got.Status, got.Body, want.Status, want.Body)
	}
}

// Row is a listNodes row, as much of it as the tiers assert.
type Row struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Self          bool     `json:"self"`
	Key           string   `json:"key"`
	Addresses     []string `json:"addresses"`
	Roles         []string `json:"roles"`
	IntentVersion string   `json:"intentVersion"`
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

// Nodes is listNodes on the Node named name.
func (a *API) Nodes(name string) []Row {
	a.t.Helper()
	var list struct {
		Nodes []Row `json:"nodes"`
	}
	a.decode(a.Admin(name, http.MethodGet, "/api/v1/nodes", ""), http.StatusOK, &list)
	return list.Nodes
}

// SetRoles is setNodeRoles of the Node with id on the Node named name.
func (a *API) SetRoles(name, id string, roles ...string) Row {
	a.t.Helper()
	body, err := json.Marshal(struct {
		Roles []string `json:"roles"`
	}{append([]string{}, roles...)})
	if err != nil {
		a.t.Fatal(err)
	}
	var row Row
	a.decode(a.Admin(name, http.MethodPut, "/api/v1/nodes/"+id+"/roles", string(body)), http.StatusOK, &row)
	return row
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
// setNodeRoles that answers once seedbox holds the new Roles.
func CheckHome(t *testing.T, a *API) {
	t.Helper()
	box := a.Node("seedbox")
	if link := netip.MustParseAddrPort(box.Listeners["link"]).Addr(); box.Claim != world.ClaimWorker || box.Setup != scenario.SetupJoin || link != netip.MustParseAddr("127.0.0.3") {
		t.Fatalf("the manifest has seedbox with claim %s, setup %s and listeners.link on %s; want worker, join and 127.0.0.3", box.Claim, box.Setup, link)
	}
	Expect(t, "seedbox GET /api/v1/status", a.Get("seedbox", "/api/v1/status"), Reply{200, `{"claim":"worker"}` + "\n"})
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
	if set.Facts == nil || !slices.Equal(set.Facts.Assigned, []string{"searching"}) || set.Facts.AppliedVersion != set.IntentVersion {
		t.Fatalf("setNodeRoles(seedbox, [searching]) answered facts %+v at intentVersion %s, want searching applied", set.Facts, set.IntentVersion)
	}

	for name, want := range map[string]bool{"node.key": true, "pairing.json": true, "local.db": true, "library.db": false} {
		if _, err := os.Stat(filepath.Join(box.DataDir, name)); (err == nil) != want {
			t.Errorf("seedbox's data dir holds %s: %v, want %v", name, err == nil, want)
		}
	}
}

func checkRow(t *testing.T, r Row, link string, roles []string) {
	t.Helper()
	if r.Link.State != link || !slices.Equal(r.Roles, roles) || r.Facts == nil || !slices.Equal(r.Facts.Assigned, roles) {
		t.Fatalf("listNodes row %s: link %s, roles %v, facts %+v; want link %s with %v assigned", r.Name, r.Link.State, r.Roles, r.Facts, link, roles)
	}
}
