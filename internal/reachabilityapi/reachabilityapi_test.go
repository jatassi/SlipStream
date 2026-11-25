package reachabilityapi_test

import (
	"context"
	"encoding/json"
	"math/rand/v2"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/reachability"
	"github.com/jatassi/slipstream/internal/reachability/judge"
	"github.com/jatassi/slipstream/internal/reachability/judge/judgetest"
	"github.com/jatassi/slipstream/internal/reachability/reachabilitytest"
	"github.com/jatassi/slipstream/internal/reachabilityapi"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

func world(t *testing.T, b *judgetest.B) *reachabilitytest.World {
	t.Helper()
	return reachabilitytest.New(t, b, reachabilitytest.Config{
		Timings: reachability.DefaultTimings(),
		Rand:    rand.New(rand.NewPCG(1, 2)), //nolint:gosec // a fixed draw the test replays
	})
}

// plex adds a Plex on atlas.
func plex(t *testing.T, w *reachabilitytest.World) string {
	t.Helper()
	id := storage.NewUUID()
	err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO mediaservers_server (id, kind, name, address, identity, version, reaching_node_id, secrets_sealed, created_at, updated_at)
			VALUES (?, 'plex', 'Plex', 'http://192.168.1.20:32400', ?, '1.43.5', ?, x'00', 1, 1)`, id, reachabilitytest.Identity, judgetest.Atlas)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return id.String()
}

type controller struct{ r *reachability.Reachability }

func (c controller) Reachability() (*reachability.Reachability, bool) { return c.r, c.r != nil }

func codeOf(reply string) string {
	var p struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal([]byte(reply), &p)
	return p.Code
}

func subjectOf(reply string) string {
	var e struct {
		Subject string `json:"subject"`
	}
	_ = json.Unmarshal([]byte(reply), &e)
	return e.Subject
}

func idOf(reply string) string {
	var e struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal([]byte(reply), &e)
	return e.ID
}

// Every operation in the order an Admin meets them, each problem
// included. Each step's path and body may name what an earlier step made.
func TestEveryOperationAndProblem(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		b := judgetest.Open().Inside()
		w := world(t, b)
		o := ownertest.Serve(t, w.Lib, func(r *api.Router) { reachabilityapi.Routes(r, controller{w.Reach}) })
		server := plex(t, w)
		other := storage.NewUUID().String()
		var exposure string
		// The Exposure a write answers with is named.
		subjects := map[string]string{"a Media Server": "Plex", "the UI and portal": "SlipStream UI and portal", "its address": "Plex"}
		for _, c := range []struct {
			name, method string
			path, body   func() string
			status       int
			code         string
		}{
			{"nothing wanted reads empty", http.MethodGet, fixed("/api/v1/remote-access"), none, 200, ""},
			{"a run with nothing wanted", http.MethodPost, fixed("/api/v1/remote-access/run"), none, 409, "nothing_wanted"},
			{"no public address", http.MethodPost, fixed("/api/v1/remote-access/exposures"), fixed(`{"kind":"media-server","serverId":"` + server + `"}`), 422, "public_required"},
			{"a public address that isn't a URL", http.MethodPost, fixed("/api/v1/remote-access/exposures"), fixed(`{"kind":"media-server","serverId":"` + server + `","public":"ftp://plex.example.org"}`), 422, "public_not_url"},
			{"an unknown Media Server", http.MethodPost, fixed("/api/v1/remote-access/exposures"), fixed(`{"kind":"media-server","serverId":"` + other + `","public":"https://plex.example.org"}`), 404, "server_not_found"},
			{"a Media Server", http.MethodPost, fixed("/api/v1/remote-access/exposures"), fixed(`{"kind":"media-server","serverId":"` + server + `","public":"https://plex.example.org"}`), 201, ""},
			{"the same Media Server again", http.MethodPost, fixed("/api/v1/remote-access/exposures"), fixed(`{"kind":"media-server","serverId":"` + server + `","public":"https://plex.example.org"}`), 409, "exposure_exists"},
			{"the UI and portal", http.MethodPost, fixed("/api/v1/remote-access/exposures"), fixed(`{"kind":"ui-portal"}`), 201, ""},
			{"an unknown Exposure's address", http.MethodPatch, fixed("/api/v1/remote-access/exposures/" + other), fixed(`{"public":"https://plex.example.org"}`), 404, "exposure_not_found"},
			{"its address", http.MethodPatch, func() string { return "/api/v1/remote-access/exposures/" + exposure }, fixed(`{"public":"https://plex2.example.org"}`), 200, ""},
			{"its address emptied", http.MethodPatch, func() string { return "/api/v1/remote-access/exposures/" + exposure }, fixed(`{"public":""}`), 422, "public_required"},
			{"a run of an unknown Exposure", http.MethodPost, fixed("/api/v1/remote-access/run?exposure=" + other), none, 404, "exposure_not_found"},
			{"a run of one", http.MethodPost, func() string { return "/api/v1/remote-access/run?exposure=" + exposure }, none, 202, ""},
			{"the page", http.MethodGet, fixed("/api/v1/remote-access"), none, 200, ""},
			{"an unknown Node's network", http.MethodGet, fixed("/api/v1/remote-access/nodes/" + other), none, 404, "node_not_found"},
			{"den's network", http.MethodGet, fixed("/api/v1/remote-access/nodes/" + judgetest.Den.String()), none, 200, ""},
			{"den's settings", http.MethodGet, fixed("/api/v1/remote-access/nodes/" + judgetest.Den.String() + "/settings"), none, 200, ""},
			{"den's stable address inside the home", http.MethodPatch, fixed("/api/v1/remote-access/nodes/" + judgetest.Den.String() + "/settings"), fixed(`{"values":{"reachability.stableIPv6":"fd00::1"}}`), 422, "setting_out_of_range"},
			{"den's stable address", http.MethodPatch, fixed("/api/v1/remote-access/nodes/" + judgetest.Den.String() + "/settings"), fixed(`{"values":{"reachability.stableIPv6":"2001:db8:1::30"}}`), 200, ""},
			{"the settings", http.MethodGet, fixed("/api/v1/remote-access/settings"), none, 200, ""},
			{"an unknown setting", http.MethodPatch, fixed("/api/v1/remote-access/settings"), fixed(`{"values":{"reachability.nothing":1}}`), 422, "unknown_setting"},
			{"the warning's wait", http.MethodPatch, fixed("/api/v1/remote-access/settings"), fixed(`{"values":{"reachability.warnAfter":3600}}`), 200, ""},
			{"an unknown suggestion", http.MethodPost, fixed("/api/v1/remote-access/suggestions/" + other + "/dismiss"), none, 404, "server_not_found"},
			{"the UI and portal's suggestion", http.MethodPost, fixed("/api/v1/remote-access/suggestions/ui-portal/dismiss"), none, 204, ""},
			{"an echo no check issued", http.MethodGet, fixed("/api/v1/remote-access/echo/" + strings.Repeat("0", 32)), none, 404, "echo_not_found"},
			{"an unknown Exposure's removal", http.MethodDelete, fixed("/api/v1/remote-access/exposures/" + other), none, 404, "exposure_not_found"},
			{"its removal", http.MethodDelete, func() string { return "/api/v1/remote-access/exposures/" + exposure }, none, 204, ""},
		} {
			status, reply := o.Call(t, c.method, c.path(), c.body())
			synctest.Wait()
			if status != c.status || codeOf(reply) != c.code {
				t.Fatalf("%s: %d %s, want %d %q", c.name, status, reply, c.status, c.code)
			}
			if want, ok := subjects[c.name]; ok && subjectOf(reply) != want {
				t.Errorf("%s answered subject %q, want %q", c.name, subjectOf(reply), want)
			}
			if c.name == "a Media Server" {
				exposure = idOf(reply)
			}
		}
	})
}

func fixed(s string) func() string { return func() string { return s } }

func none() string { return "" }

// The page answers at once while the Controller's resolver hasn't answered
// for Address, and suggests the UI and portal once it has.
func TestThePageNeverWaitsOnTheResolver(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, judgetest.Open())
		o := ownertest.Serve(t, w.Lib, func(r *api.Router) { reachabilityapi.Routes(r, controller{w.Reach}) })
		answer := make(chan struct{})
		w.Site(judgetest.Atlas, func(s *reachabilitytest.Site) { s.Resolving = answer })
		err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, "UPDATE identity_settings SET address = 'https://slipstream.example.org' WHERE id = 1")
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		plex(t, w)
		kinds := func() []string {
			start := time.Now()
			status, reply := o.Call(t, http.MethodGet, "/api/v1/remote-access", "")
			if status != http.StatusOK || time.Since(start) != 0 {
				t.Fatalf("GET /remote-access: %d after %s, want 200 at once", status, time.Since(start))
			}
			var page struct {
				Suggestions []struct {
					Kind string `json:"kind"`
				} `json:"suggestions"`
			}
			if err := json.Unmarshal([]byte(reply), &page); err != nil {
				t.Fatal(err)
			}
			var out []string
			for _, s := range page.Suggestions {
				out = append(out, s.Kind)
			}
			slices.Sort(out)
			return out
		}
		kinds()
		synctest.Wait()
		if got := kinds(); !slices.Equal(got, []string{"media-server"}) {
			t.Errorf("suggestions while the resolver hasn't answered = %v, want the Media Server alone", got)
		}
		close(answer)
		synctest.Wait()
		if got := kinds(); !slices.Equal(got, []string{"media-server", "ui-portal"}) {
			t.Errorf("suggestions once it answered = %v, want the Media Server and the UI and portal", got)
		}
	})
}

// Every operation but the echo refuses a Member; the echo answers anyone.
func TestOnlyTheEchoAnswersBelowAdmin(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, judgetest.Open())
		o := ownertest.Serve(t, w.Lib, func(r *api.Router) { reachabilityapi.Routes(r, controller{w.Reach}) })
		member := identitytest.Session(t, w.Lib, identitytest.Member(t, w.Lib, "member@example.test"))
		id := storage.NewUUID().String()
		for _, c := range []struct{ method, path string }{
			{http.MethodGet, "/api/v1/remote-access"},
			{http.MethodPost, "/api/v1/remote-access/exposures"},
			{http.MethodPatch, "/api/v1/remote-access/exposures/" + id},
			{http.MethodDelete, "/api/v1/remote-access/exposures/" + id},
			{http.MethodPost, "/api/v1/remote-access/suggestions/ui-portal/dismiss"},
			{http.MethodPost, "/api/v1/remote-access/run"},
			{http.MethodGet, "/api/v1/remote-access/nodes/" + id},
			{http.MethodGet, "/api/v1/remote-access/nodes/" + id + "/settings"},
			{http.MethodPatch, "/api/v1/remote-access/nodes/" + id + "/settings"},
			{http.MethodGet, "/api/v1/remote-access/settings"},
			{http.MethodPatch, "/api/v1/remote-access/settings"},
		} {
			if status, reply := o.CallAs(t, c.method, c.path, "{}", member); status != http.StatusForbidden {
				t.Fatalf("a Member's %s %s: %d %s, want 403", c.method, c.path, status, reply)
			}
		}
		if status, reply := o.CallAs(t, http.MethodGet, "/api/v1/remote-access/echo/"+strings.Repeat("0", 32), "", ""); status != http.StatusNotFound || codeOf(reply) != "echo_not_found" {
			t.Fatalf("nobody's echo: %d %s, want 404 echo_not_found", status, reply)
		}
	})
}

// On a Node that is no Controller the echo answers that the check ended.
func TestAWorkerAnswersNoEcho(t *testing.T) {
	w := world(t, judgetest.Open())
	o := ownertest.Serve(t, w.Lib, func(r *api.Router) { reachabilityapi.Routes(r, controller{}) })
	if status, reply := o.CallAs(t, http.MethodGet, "/api/v1/remote-access/echo/"+strings.Repeat("a", 32), "", ""); status != http.StatusNotFound || codeOf(reply) != "echo_not_found" {
		t.Fatalf("%d %s, want 404 echo_not_found", status, reply)
	}
}

// servedBy is a Serve stream's connection from node.
type servedBy struct {
	net.Conn
	node role.NodeID
}

func (s servedBy) ServedFor() role.NodeID { return s.node }

// arrival is how a tester's echo fetch reaches the Controller: on the
// bound listener, or through via's Serve stream, from source, which is
// empty for the tester's public address.
type arrival struct {
	listener api.Listener
	via      role.NodeID
	source   string
}

// fetch serves a tester's echo fetch to *o as a arrives.
func (a arrival) fetch(t *testing.T, o **ownertest.Owner) func(context.Context, string, netip.Addr) string {
	return func(ctx context.Context, path string, from netip.Addr) string {
		source := from.String()
		if a.source != "" {
			source = a.source
		}
		var req *http.Request
		if a.listener == api.Bound {
			req = httptest.NewRequestWithContext(api.OnListener(ctx, nil), http.MethodGet, path, nil)
			req.RemoteAddr = netip.AddrPortFrom(netip.MustParseAddr(source), 50000).String()
		} else {
			req = httptest.NewRequestWithContext(api.OverLink(ctx, servedBy{node: a.via}), http.MethodGet, path, nil)
			req.Header.Set("Forwarded", "for="+source)
		}
		status, reply := (*o).Send(req)
		if status != http.StatusOK {
			t.Errorf("the echo answered %d %s", status, reply)
		}
		return reply
	}
}

// wantUI sets the Address to b's public address and wants the UI and
// portal, which a check then runs on.
func wantUI(t *testing.T, w *reachabilitytest.World, o *ownertest.Owner, b *judgetest.B) {
	t.Helper()
	err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, "UPDATE identity_settings SET address = ? WHERE id = 1", b.Facts().Exposure.Public.String())
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	identity.AddressChangedEvent.Bind(w.Hub).Publish(t.Context(), identity.AddressChanged{})
	if status, reply := o.Call(t, http.MethodPost, "/api/v1/remote-access/exposures", `{"kind":"ui-portal"}`); status != http.StatusCreated {
		t.Fatalf("wanting the UI and portal: %d %s", status, reply)
	}
	synctest.Wait()
}

// A tester's echo fetch answers on the bound listener and through a
// Serve stream, and the check counts it only where AcceptEcho allows:
// never through the tester's own stream, and on a Direct route only from
// the tester's public address or through another program on the origin.
func TestTheEchoCountsByHowItArrived(t *testing.T) {
	for _, c := range []struct {
		name    string
		shape   func() *judgetest.B
		arrival arrival
		tested  bool
		fact    judge.FactID
	}{
		{name: "on the bound listener from the tester", shape: judgetest.OpenUI, arrival: arrival{listener: api.Bound}, tested: true},
		{name: "through den's Serve stream from the tester", shape: func() *judgetest.B { return judgetest.OpenUI().Inside() }, arrival: arrival{listener: api.Served, via: judgetest.Den}, tested: true},
		{name: "on the bound listener through Caddy", shape: func() *judgetest.B { return judgetest.OpenUI().Caddy().Loopback() }, arrival: arrival{listener: api.Bound, source: "127.0.0.1"}, tested: true, fact: judge.FactEchoThroughProgram},
		{name: "through the tester's own Serve stream", shape: judgetest.OpenUI, arrival: arrival{listener: api.Served, via: judgetest.Seedbox}, fact: judge.FactEchoOwnStream},
		{name: "on the bound listener over loopback with nothing in front", shape: judgetest.OpenUI, arrival: arrival{listener: api.Bound, source: "127.0.0.1"}, fact: judge.FactEchoSource},
		{name: "on the bound listener from another address", shape: judgetest.OpenUI, arrival: arrival{listener: api.Bound, source: "198.51.100.20"}, fact: judge.FactEchoSource},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				b := c.shape()
				var o *ownertest.Owner
				w := reachabilitytest.New(t, b, reachabilitytest.Config{
					Timings: reachability.DefaultTimings(),
					Rand:    rand.New(rand.NewPCG(1, 2)), //nolint:gosec // a fixed draw the test replays
					Fetch:   c.arrival.fetch(t, &o),
				})
				o = ownertest.Serve(t, w.Lib, func(r *api.Router) { reachabilityapi.Routes(r, controller{w.Reach}) })
				wantUI(t, w, o, b)
				rt := w.Route(t, "SlipStream UI and portal")
				shown := judgetest.Shown(rt.Result)
				if !rt.Judged || rt.Result.Tested != c.tested || (c.fact != "" && !slices.Contains(shown, c.fact)) {
					t.Fatalf("judged %v, tested %v, facts %v; want tested %v showing %q", rt.Judged, rt.Result.Tested, shown, c.tested, c.fact)
				}
			})
		})
	}
}
