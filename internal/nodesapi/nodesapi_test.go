package nodesapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/nodesapi"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// Every member the document gives the Role enum names exactly the table's
// Roles, in its order.
func TestEveryRoleEnumIsRoleNames(t *testing.T) {
	r := api.NewRouter(api.Services{})
	nodesapi.Routes(r, nil)
	type enum struct {
		Enum  []string `json:"enum"`
		Items struct {
			Enum []string `json:"enum"`
		} `json:"items"`
	}
	var doc struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]enum `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(r.Document(), &doc); err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, n := range role.Names() {
		want = append(want, string(n))
	}
	var seen []string
	for name, schema := range doc.Components.Schemas {
		for member, p := range schema.Properties {
			for _, got := range [][]string{p.Enum, p.Items.Enum} {
				if !slices.Contains(got, string(role.Searching)) {
					continue
				}
				seen = append(seen, name+"."+member)
				if !slices.Equal(got, want) {
					t.Errorf("%s.%s enum = %v, want role.Names() %v", name, member, got, want)
				}
			}
		}
	}
	slices.Sort(seen)
	for _, member := range []string{"NodesAssignment.roles", "NodesBoardNode.assigned", "NodesGesture.role", "NodesRoleView.role"} {
		if _, ok := slices.BinarySearch(seen, member); !ok {
			t.Errorf("the Role enum is not on %s; it is on %v", member, seen)
		}
	}
}

// unadvertised is a link with no address to advertise yet.
type unadvertised struct{}

func (unadvertised) State(link.Key) link.State { return link.Down{} }

func (unadvertised) Subscribe(context.Context) <-chan link.Change { return nil }

func (unadvertised) Advertised() (link.Addresses, bool) { return link.Addresses{}, false }

type changes struct{}

func (changes) Publish(context.Context, nodes.Changed) {}

type roster struct{ svc *nodes.Service }

func (r roster) Nodes() (*nodes.Service, bool) { return r.svc, true }

// claimed is an identity service holding an Admin, and the API key.
func claimed(t *testing.T, lib *storage.DB[storage.Library], log *slog.Logger) (*identity.Service, *identity.Gate, string) {
	t.Helper()
	keys, err := secret.Open(t.Context(), lib, filepath.Join(t.TempDir(), secret.KeyFile))
	if err != nil {
		t.Fatal(err)
	}
	svc, err := identity.New(t.Context(), identity.Config{Library: lib, Pepper: keys.PINPepper(), Hub: identitytest.Hub(t)})
	if err != nil {
		t.Fatal(err)
	}
	gate, code, err := identity.NewGate(t.Context(), identitytest.Node{Service: svc}.Claim, log)
	if err != nil {
		t.Fatal(err)
	}
	pin, err := identity.ParsePIN("2468")
	if err != nil {
		t.Fatal(err)
	}
	var admin identity.Claimed
	err = gate.Run(t.Context(), identity.Caller{}, code, func(c identity.Claim) {
		admin, err = svc.ClaimController(t.Context(), c, "ada@example.test", "Ada", pin)
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc, gate, admin.APIKey.Reveal()
}

func TestAJoinCodeNeedsALinkAddress(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), role.LibrarySchema())
	svc, gate, apiKey := claimed(t, lib, log)
	ns, err := nodes.Open(t.Context(), nodes.Config{Library: lib, Name: "home", Link: unadvertised{}, Changed: changes{}, Owners: nodestest.Unowned(), Log: log})
	if err != nil {
		t.Fatal(err)
	}
	r := api.NewRouter(api.Services{Node: identitytest.Node{Service: svc}, Gate: gate, Web: fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}}, Log: log})
	nodesapi.Routes(r, roster{ns})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/nodes/join-codes", nil)
	req.Header.Set("X-Api-Key", apiKey)
	w := httptest.NewRecorder()
	r.Handler().ServeHTTP(w, req)
	if want := `{"status":409,"code":"no_link_address","title":"The link has no address to put in a Join Code"}` + "\n"; w.Body.String() != want {
		t.Fatalf("createJoinCode = %d %s, want %s", w.Code, w.Body, want)
	}
}

func TestTheJoinCodeLifetimeIsASettingField(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), role.LibrarySchema())
	ns, err := nodes.Open(t.Context(), nodes.Config{Library: lib, Name: "home", Link: unadvertised{}, Changed: changes{}, Owners: nodestest.Unowned(), Log: log})
	if err != nil {
		t.Fatal(err)
	}
	s := ownertest.Serve(t, lib, func(r *api.Router) { nodesapi.Routes(r, roster{ns}) })
	field := func(value int64, isDefault bool) string {
		return fmt.Sprintf(`{"joinCodeLifetime":{"key":"nodes.joinCodeLifetime","label":"Join Code lifetime","source":"database","applies":"live","isDefault":%t,"pendingRestart":false,"kind":"duration","value":%d,"running":%d,"default":3600,"min":600,"max":604800,"step":60}}`, isDefault, value, value)
	}
	const path = "/api/v1/nodes/settings"
	s.Expect(t, http.MethodGet, path, "", http.StatusOK, field(3600, true))
	s.Expect(t, http.MethodPatch, path, `{"joinCodeLifetime":540}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"setting_out_of_range","title":"The value is outside the setting's range","detail":"Between 10 min and 7 days."}`)
	s.Expect(t, http.MethodPatch, path, `{"joinCodeLifetime":7200}`, http.StatusOK, field(7200, false))
	s.Expect(t, http.MethodPatch, path, `{}`, http.StatusOK, field(7200, false))
	s.Expect(t, http.MethodPatch, path, `{"joinCodeLifetime":null}`, http.StatusOK, field(3600, true))
}

type effect struct {
	Kind  string `json:"kind"`
	Words string `json:"words"`
}

type outcome struct {
	Basis   string   `json:"basis"`
	Effects []effect `json:"effects"`
	Checks  []struct {
		Reason string `json:"reason"`
		Words  string `json:"words"`
	} `json:"checks"`
}

type problem struct {
	Code    string  `json:"code"`
	Outcome outcome `json:"outcome"`
}

type boardNode struct {
	Name       string   `json:"name"`
	Controller bool     `json:"controller"`
	Away       bool     `json:"away"`
	Assigned   []string `json:"assigned"`
	Roles      []struct {
		State string `json:"state"`
	} `json:"roles"`
}

type gesture struct {
	Role    string `json:"role"`
	From    string `json:"from"`
	Verdict string `json:"verdict"`
	Word    string `json:"word"`
}

type board struct {
	Basis    string      `json:"basis"`
	Nodes    []boardNode `json:"nodes"`
	Gestures []gesture   `json:"gestures"`
}

// call sends a request to o and decodes its answer into v, failing unless
// it answers want.
func call(t *testing.T, o *ownertest.Owner, method, path, body string, want int, v any) {
	t.Helper()
	status, reply := o.Call(t, method, path, body)
	if status != want {
		t.Fatalf("%s %s answered %d %s, want %d", method, path, status, reply, want)
	}
	if err := json.Unmarshal([]byte(reply), v); err != nil {
		t.Fatalf("%s: %v", reply, err)
	}
}

func set(basis string, assignments ...string) string {
	return fmt.Sprintf(`{"basis":%q,"set":[%s]}`, basis, strings.Join(assignments, ","))
}

func assign(id, roles string) string {
	return fmt.Sprintf(`{"nodeId":%q,"roles":[%s]}`, id, roles)
}

// The Controller home and box, out of step and holding Relaying, through
// the three Role operations: the board, a plan and a set, each refused for
// a stale basis, a block and an unknown Node.
func TestTheRoleOperations(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), role.LibrarySchema())
	boxID := storage.NewUUID()
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'box', 0)", boxID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "INSERT INTO nodes_role (node_id, role, assigned_at) VALUES (?, 'relaying', 0)", boxID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	ns, err := nodes.Open(t.Context(), nodes.Config{Library: lib, Name: "home", Link: unadvertised{}, Changed: changes{}, Owners: nodestest.Unowned(), Log: log})
	if err != nil {
		t.Fatal(err)
	}
	s := ownertest.Serve(t, lib, func(r *api.Router) { nodesapi.Routes(r, roster{ns}) })
	box, home := boxID.String(), ns.Self().String()
	const roles, plan = "/api/v1/nodes/roles", "/api/v1/nodes/roles/plan"

	var b board
	call(t, s, http.MethodGet, roles, "", http.StatusOK, &b)
	if len(b.Nodes) != 2 || b.Nodes[0].Name != "box" || !b.Nodes[0].Away || !slices.Equal(b.Nodes[0].Assigned, []string{"relaying"}) ||
		len(b.Nodes[0].Roles) != 1 || b.Nodes[0].Roles[0].State != "away" || !b.Nodes[1].Controller || b.Nodes[1].Away {
		t.Fatalf("the board's nodes are %+v, want box away holding Relaying, then home the Controller", b.Nodes)
	}
	if !slices.Contains(b.Gestures, gesture{Role: "relaying", Verdict: "block", Word: "box already relays"}) {
		t.Fatalf("the gestures %+v have no blocked add of Relaying onto home", b.Gestures)
	}

	var planned outcome
	call(t, s, http.MethodPost, plan, set(b.Basis, assign(box, `"relaying","searching"`)), http.StatusOK, &planned)
	if len(planned.Effects) != 1 || planned.Effects[0].Kind != "gains" || planned.Effects[0].Words != "box starts Searching." || planned.Basis != b.Basis {
		t.Fatalf("planRoles answered %+v, want box gaining Searching on the board's basis", planned)
	}
	var blocked problem
	call(t, s, http.MethodPut, roles, set(b.Basis, assign(home, `"relaying","serving"`)), http.StatusConflict, &blocked)
	if blocked.Code != "roles_blocked" || len(blocked.Outcome.Checks) != 1 || blocked.Outcome.Checks[0].Reason != "role_taken" {
		t.Fatalf("a second relay answered %+v, want roles_blocked naming box", blocked)
	}
	var unknown problem
	call(t, s, http.MethodPut, roles, set(b.Basis, assign(storage.NewUUID().String(), "")), http.StatusNotFound, &unknown)
	if unknown.Code != "node_not_found" {
		t.Fatalf("a set naming an unknown Node answered %+v, want node_not_found", unknown)
	}

	var applied outcome
	call(t, s, http.MethodPut, roles, set(b.Basis, assign(box, `"relaying","searching"`)), http.StatusOK, &applied)
	if len(applied.Effects) != 1 || applied.Effects[0].Words != "box starts Searching." || applied.Basis != b.Basis {
		t.Fatalf("setRoles answered %+v, want box gaining Searching", applied)
	}
	var stale problem
	call(t, s, http.MethodPost, plan, set(b.Basis, assign(box, "")), http.StatusConflict, &stale)
	if stale.Code != "roles_stale" || stale.Outcome.Basis == b.Basis || len(stale.Outcome.Effects) != 2 {
		t.Fatalf("a plan on the old basis answered %+v, want roles_stale with the plan on the new basis", stale)
	}
	var twice problem
	call(t, s, http.MethodPut, roles, set(stale.Outcome.Basis, assign(home, ""), assign(home, "")), http.StatusUnprocessableEntity, &twice)
	if twice.Code != "invalid" {
		t.Fatalf("a set naming home twice answered %+v, want invalid", twice)
	}
}

// box and seed both download and only box holds a torrent: writing off
// seed's Downloading is refused for holding nothing, and writing off box's
// gives it up.
func TestTheWriteOffOperation(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), role.LibrarySchema())
	boxID, seedID := storage.NewUUID(), storage.NewUUID()
	pairDownloading(t, lib, map[storage.UUID]string{boxID: "box", seedID: "seed"})
	owners := nodestest.Unowned()
	for i, o := range owners {
		if o.Role == role.Downloading {
			owners[i].Held = func(_ context.Context, _ storage.Reader[storage.Library], node nodes.ID) (nodes.Held, error) {
				if node.UUID == boxID {
					return nodes.Held{Units: 1, Noun: nodes.Noun{One: "torrent", Many: "torrents"}}, nil
				}
				return nodes.Held{}, nil
			}
		}
	}
	ns, err := nodes.Open(t.Context(), nodes.Config{Library: lib, Name: "home", Link: unadvertised{}, Changed: changes{}, Owners: owners, Log: log})
	if err != nil {
		t.Fatal(err)
	}
	s := ownertest.Serve(t, lib, func(r *api.Router) { nodesapi.Routes(r, roster{ns}) })
	writeOff := func(id string) string { return "/api/v1/nodes/" + id + "/write-offs" }
	const downloads = `{"roles":["downloading"]}`

	var empty problem
	call(t, s, http.MethodPost, writeOff(seedID.String()), downloads, http.StatusConflict, &empty)
	if empty.Code != "nothing_held" || len(empty.Outcome.Checks) != 1 || empty.Outcome.Checks[0].Reason != "nothing_held" {
		t.Fatalf("writing off what seed doesn't hold answered %+v, want nothing_held", empty)
	}
	var wrote outcome
	call(t, s, http.MethodPost, writeOff(boxID.String()), downloads, http.StatusOK, &wrote)
	if !slices.ContainsFunc(wrote.Effects, func(e effect) bool { return e.Kind == "writes-off" }) {
		t.Fatalf("writing off box's Downloading answered %+v, want a writes-off effect", wrote)
	}
	var unknown problem
	call(t, s, http.MethodPost, writeOff(storage.NewUUID().String()), downloads, http.StatusNotFound, &unknown)
	if unknown.Code != "node_not_found" {
		t.Fatalf("writing off on an unknown Node answered %+v, want node_not_found", unknown)
	}
	var bad problem
	call(t, s, http.MethodPost, writeOff(boxID.String()), `{"roles":["juggling"]}`, http.StatusUnprocessableEntity, &bad)
	if bad.Code != "invalid" {
		t.Fatalf("writing off an unknown Role answered %+v, want invalid", bad)
	}
}

// pairDownloading pairs each named Node into lib with Downloading ticked.
func pairDownloading(t *testing.T, lib *storage.DB[storage.Library], named map[storage.UUID]string) {
	t.Helper()
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for id, name := range named {
			if _, err := tx.ExecContext(ctx, "INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), ?, 0)", id, name); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, "INSERT INTO nodes_role (node_id, role, assigned_at) VALUES (?, 'downloading', 0)", id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
