package nodesapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/joincode"
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

func (unadvertised) Reset(link.Key) {}

func (unadvertised) Keep(link.Target) {}

func (unadvertised) Drop(link.Key) {}

func (unadvertised) Invite(context.Context, link.Target) error { return nil }
func (unadvertised) Stats(link.Key) (link.Stats, bool)         { return link.Stats{}, false }
func (unadvertised) LaneBytes(link.Key) link.Stats             { return link.Stats{} }

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
	err = gate.Run(t.Context(), identity.Caller{}, code, func(c identity.Claim) {
		_, err = svc.ClaimController(t.Context(), c, "ada@example.test", "Ada", pin)
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc, gate, identitytest.APIKey(t, lib)
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
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/join-codes", nil)
	req.Header.Set("X-Api-Key", apiKey)
	w := httptest.NewRecorder()
	r.Handler().ServeHTTP(w, req)
	if want := `{"status":409,"code":"no_link_address","title":"The link has no address to put in a Join Code"}` + "\n"; w.Body.String() != want {
		t.Fatalf("createJoinCode = %d %s, want %s", w.Code, w.Body, want)
	}
}

func TestTheNodesSettingsAreSettingFields(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), role.LibrarySchema())
	ns, err := nodes.Open(t.Context(), nodes.Config{Library: lib, Name: "home", Link: unadvertised{}, Changed: changes{}, Owners: nodestest.Unowned(), Log: log})
	if err != nil {
		t.Fatal(err)
	}
	s := ownertest.Serve(t, lib, func(r *api.Router) { nodesapi.Routes(r, roster{ns}) })
	fields := func(lifetime, grace int64) string {
		return fmt.Sprintf(`{"joinCodeLifetime":{"key":"nodes.joinCodeLifetime","label":"Join Code lifetime","source":"database","applies":"live","isDefault":%t,"pendingRestart":false,"kind":"duration","value":%d,"running":%d,"default":3600,"min":600,"max":604800,"step":60},`+
			`"offlineGrace":{"key":"nodes.offlineGrace","label":"Offline grace","source":"database","applies":"live","isDefault":%t,"pendingRestart":false,"kind":"duration","value":%d,"running":%d,"default":900,"min":60,"max":86400,"step":60}}`,
			lifetime == 3600, lifetime, lifetime, grace == 900, grace, grace)
	}
	const path = "/api/v1/nodes/settings"
	s.Expect(t, http.MethodGet, path, "", http.StatusOK, fields(3600, 900))
	s.Expect(t, http.MethodPatch, path, `{"joinCodeLifetime":540}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"setting_out_of_range","title":"The value is outside the setting's range","detail":"Between 10 min and 7 days."}`)
	s.Expect(t, http.MethodPatch, path, `{"joinCodeLifetime":7200}`, http.StatusOK, fields(7200, 900))
	s.Expect(t, http.MethodPatch, path, `{"joinCodeLifetime":3600,"offlineGrace":30}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"setting_out_of_range","title":"The value is outside the setting's range","detail":"Between 1 min and 1 day."}`)
	s.Expect(t, http.MethodPatch, path, `{"offlineGrace":60}`, http.StatusOK, fields(7200, 60))
	s.Expect(t, http.MethodPatch, path, `{}`, http.StatusOK, fields(7200, 60))
	s.Expect(t, http.MethodPatch, path, `{"joinCodeLifetime":null,"offlineGrace":null}`, http.StatusOK, fields(3600, 900))
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

type roleEntry struct {
	Role   string `json:"role"`
	Label  string `json:"label"`
	Probe  string `json:"probe"`
	Pinned bool   `json:"pinned"`
}

type roleGroup struct {
	Group string      `json:"group"`
	Label string      `json:"label"`
	Roles []roleEntry `json:"roles"`
}

type board struct {
	Basis    string      `json:"basis"`
	Groups   []roleGroup `json:"groups"`
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
	if len(b.Groups) != 5 || b.Groups[0].Label != "Acquire" ||
		!slices.Equal(b.Groups[0].Roles, []roleEntry{{Role: "searching", Label: "Searching"}, {Role: "downloading", Label: "Downloading", Probe: "engine-starts"}}) ||
		b.Groups[4].Roles[1] != (roleEntry{Role: "serving", Label: "Serving", Probe: "listener-up", Pinned: true}) {
		t.Fatalf("the board's groups are %+v, want the Role table from Acquire to Reach", b.Groups)
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

// advertised is a link that advertises home's address and whose invites
// wait for their end.
type advertised struct{ unadvertised }

func (advertised) Advertised() (link.Addresses, bool) {
	a, err := link.ParseAddresses([]netip.AddrPort{netip.MustParseAddrPort("127.0.0.2:8485")})
	return a, err == nil
}

func (advertised) Invite(ctx context.Context, _ link.Target) error {
	<-ctx.Done()
	return fmt.Errorf("invite: %w", ctx.Err())
}

func TestTheJoinCodeOperations(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), role.LibrarySchema())
	ns, err := nodes.Open(t.Context(), nodes.Config{Library: lib, Name: "home", Link: advertised{}, Changed: changes{}, Owners: nodestest.Unowned(), Timings: nodes.DefaultTimings(), Log: log})
	if err != nil {
		t.Fatal(err)
	}
	s := ownertest.Serve(t, lib, func(r *api.Router) { nodesapi.Routes(r, roster{ns}) })
	const codes = "/api/v1/join-codes"
	var minted struct {
		ID   string `json:"id"`
		Code string `json:"code"`
	}
	call(t, s, http.MethodPost, codes, "", http.StatusCreated, &minted)
	var list struct {
		Codes []struct {
			ID string `json:"id"`
		} `json:"codes"`
	}
	call(t, s, http.MethodGet, codes, "", http.StatusOK, &list)
	if len(list.Codes) != 1 || list.Codes[0].ID != minted.ID {
		t.Fatalf("listJoinCodes = %+v, want the minted code %s", list, minted.ID)
	}
	code := codes + "/" + minted.ID
	var st struct {
		State   string `json:"state"`
		Dialing *struct {
			Address     string   `json:"address"`
			Addresses   []string `json:"addresses"`
			Unreachable bool     `json:"unreachable"`
		} `json:"dialing"`
	}
	call(t, s, http.MethodGet, code, "", http.StatusOK, &st)
	if st.State != "open" || st.Dialing != nil {
		t.Fatalf("getJoinCode = %+v, want open with no dial", st)
	}

	box, err := joincode.NewNodeAddress(bytes.Repeat([]byte{7}, 32), []netip.AddrPort{netip.MustParseAddrPort("127.0.0.3:8485")})
	if err != nil {
		t.Fatal(err)
	}
	var damaged problem
	call(t, s, http.MethodPost, code+"/dial", `{"nodeAddress":"SLN1-40GJ4"}`, http.StatusUnprocessableEntity, &damaged)
	if damaged.Code != "node_address_damaged" {
		t.Fatalf("a dial for damaged text answered %+v, want node_address_damaged", damaged)
	}
	dial := fmt.Sprintf(`{"nodeAddress":%q}`, box)
	if status, reply := s.Call(t, http.MethodPost, code+"/dial", dial); status != http.StatusAccepted {
		t.Fatalf("dialForCode answered %d %s, want 202", status, reply)
	}
	call(t, s, http.MethodGet, code, "", http.StatusOK, &st)
	if st.Dialing == nil || st.Dialing.Address != box.String() || !slices.Equal(st.Dialing.Addresses, []string{"127.0.0.3:8485"}) || st.Dialing.Unreachable {
		t.Fatalf("getJoinCode after the dial = %+v, want dialing %s", st, box)
	}

	if status, reply := s.Call(t, http.MethodDelete, code, ""); status != http.StatusNoContent {
		t.Fatalf("revokeJoinCode answered %d %s, want 204", status, reply)
	}
	for _, c := range []struct{ method, path, body string }{
		{http.MethodGet, code, ""},
		{http.MethodDelete, code, ""},
		{http.MethodPost, code + "/dial", dial},
	} {
		var gone problem
		call(t, s, c.method, c.path, c.body, http.StatusNotFound, &gone)
		if gone.Code != "join_code_not_found" {
			t.Fatalf("%s %s on a revoked code answered %+v, want join_code_not_found", c.method, c.path, gone)
		}
	}

	nodeDial := "/api/v1/nodes/" + ns.Self().String() + "/dial"
	var mismatch problem
	call(t, s, http.MethodPut, nodeDial, dial, http.StatusConflict, &mismatch)
	if mismatch.Code != "node_key_mismatch" {
		t.Fatalf("setNodeDial with another Node's address answered %+v, want node_key_mismatch", mismatch)
	}
	var neither problem
	call(t, s, http.MethodPut, nodeDial, `{}`, http.StatusUnprocessableEntity, &neither)
	if neither.Code != "invalid" {
		t.Fatalf("setNodeDial with neither member answered %+v, want invalid", neither)
	}
	var row struct {
		Dial struct {
			By              string   `json:"by"`
			ControllerDials []string `json:"controllerDials"`
		} `json:"dial"`
	}
	call(t, s, http.MethodPut, nodeDial, `{"addresses":[]}`, http.StatusOK, &row)
	if row.Dial.By != "" || row.Dial.ControllerDials == nil || len(row.Dial.ControllerDials) != 0 {
		t.Fatalf("setNodeDial to stop dialing answered %+v, want no session and no dials", row.Dial)
	}
}

// box holds a torrent for Downloading and seed holds nothing. Renaming
// refuses a taken or malformed name; removing refuses the Controller and
// box while it holds work, and writing that work off removes box, whose
// name is then free.
func TestTheRenameAndRemoveOperations(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), role.LibrarySchema())
	boxID, seedID := storage.NewUUID(), storage.NewUUID()
	pairDownloading(t, lib, map[storage.UUID]string{boxID: "box", seedID: "seed"})
	owners := nodestest.Unowned()
	for i, o := range owners {
		if o.Role == role.Downloading {
			owners[i].Held = func(_ context.Context, _ storage.Reader[storage.Library], node nodes.ID) (nodes.Held, error) {
				if node.UUID == boxID {
					return nodes.Held{Units: 3, Noun: nodes.Noun{One: "torrent", Many: "torrents"}, Bytes: 41 << 30}, nil
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
	node := func(id fmt.Stringer) string { return "/api/v1/nodes/" + id.String() }
	refused := func(method, path, body string, status int, code string) problem {
		t.Helper()
		var p problem
		call(t, s, method, path, body, status, &p)
		if p.Code != code {
			t.Fatalf("%s %s answered %+v, want %s", method, path, p, code)
		}
		return p
	}

	refused(http.MethodPatch, node(seedID), `{"name":"box"}`, http.StatusConflict, "node_name_taken")
	for _, name := range []string{"", " seed", "seed ", strings.Repeat("s", 64)} {
		refused(http.MethodPatch, node(seedID), fmt.Sprintf(`{"name":%q}`, name), http.StatusUnprocessableEntity, "node_name_invalid")
	}
	refused(http.MethodPatch, node(storage.NewUUID()), `{"name":"new"}`, http.StatusNotFound, "node_not_found")
	var renamed struct {
		Name string `json:"name"`
	}
	call(t, s, http.MethodPatch, node(seedID), `{"name":"`+strings.Repeat("ü", 63)+`"}`, http.StatusOK, &renamed)
	call(t, s, http.MethodPatch, node(seedID), `{"name":"seedbox"}`, http.StatusOK, &renamed)
	if renamed.Name != "seedbox" {
		t.Fatalf("renameNode answered %+v, want seedbox", renamed)
	}

	refused(http.MethodDelete, node(ns.Self()), "", http.StatusConflict, "node_is_controller")
	holds := refused(http.MethodDelete, node(boxID), "", http.StatusConflict, "node_holds_work")
	want := "box still holds 3 torrents (41 GB) for Downloading. Untick its Roles and remove it when they finish, or remove it now and write them off."
	if len(holds.Outcome.Checks) != 1 || holds.Outcome.Checks[0].Words != want {
		t.Fatalf("removing box while it holds work answered %+v, want the check %q", holds, want)
	}
	if status, reply := s.Call(t, http.MethodDelete, node(boxID)+"?writeOff=true", ""); status != http.StatusNoContent {
		t.Fatalf("removing box with writeOff answered %d %s, want 204", status, reply)
	}
	refused(http.MethodGet, node(boxID), "", http.StatusNotFound, "node_not_found")
	refused(http.MethodDelete, node(boxID), "", http.StatusNotFound, "node_not_found")
	var list struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	}
	call(t, s, http.MethodGet, "/api/v1/nodes", "", http.StatusOK, &list)
	var names []string
	for _, n := range list.Nodes {
		names = append(names, n.Name)
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"home", "seedbox"}) {
		t.Fatalf("listNodes answered %v, want seedbox and home without box", names)
	}
	call(t, s, http.MethodPatch, node(seedID), `{"name":"box"}`, http.StatusOK, &renamed)
	if !ns.WrittenOff(role.Downloading, nodes.ID{UUID: boxID}) {
		t.Fatal("box's Downloading reads as kept, want written off once removed")
	}
}
