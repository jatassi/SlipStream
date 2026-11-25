package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/gateway"
	"github.com/jatassi/slipstream/fakes/outside"
	"github.com/jatassi/slipstream/fakes/plex"
	"github.com/jatassi/slipstream/fakes/plextv"
	"github.com/jatassi/slipstream/fakes/proxy"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/stun"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

// TestRemoteAccess is the verdict matrix of S51 (Reachability diagnostic):
// each remote-* Scenario launches with doctor green and nothing sent,
// wants its routes, then plays its faults, cues and Node stops in turn,
// each row asserting what GET /remote-access answers for one route.
func TestRemoteAccess(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name     string
		scenario string
		edit     func(*scenario.Scenario)
		// ui and plex are the routes' public addresses, each {port} the
		// port the UI or Plex listens on; an empty ui wants no UI and portal.
		ui, plex string
		rows     []matrixRow
	}{
		{
			name: "remote-open", scenario: "remote-open", ui: "http://slipstream.open.localhost:{port}", plex: "http://plex.open.localhost:{port}",
			rows: []matrixRow{
				{subject: uiName, want: verdict{"fixable", "http.plain", "plain-http", "warn", true}},
				{subject: plexName, want: verdict{"reachable", "", "", "none", true}},
				{play: denNeverTests, subject: plexName, want: verdict{"reachable", "", "", "none", true}},
				{play: testerAway("seedbox"), subject: plexName, want: verdict{"reachable", "", "", "none", true}, facts: []string{"tester-away", "same-network"}},
				{play: start("seedbox"), subject: plexName, want: verdict{"reachable", "", "", "none", true}},
				{play: networkCue(`{"site":"home","wan":"203.0.113.10"}`), subject: plexName, want: verdict{"fixable", "outside.connect", "stale-a", "warn", true}, fails: "dns.a stale-a"},
				{play: secondCue(`{"site":"home","wan":"203.0.113.9"}`), subject: plexName, want: verdict{"reachable", "", "", "none", true}, facts: []string{"address-churn"}},
			},
		},
		{
			name: "remote-mappable", scenario: "remote-mappable", plex: "http://plex.mappable.localhost:{port}",
			rows: []matrixRow{
				{subject: plexName, want: verdict{"fixable", "outside.connect", "mappable-plex", "warn", true}},
				{play: mapPorts("atlas", true), subject: plexName, want: verdict{"reachable", "", "", "none", true}, by: "seedbox", writes: []string{"pcp map"}},
				{play: mapPorts("atlas", false), subject: plexName, want: verdict{"fixable", "outside.connect", "mappable-plex", "warn", true}, writes: []string{"pcp map", "pcp delete"}},
				{play: mapPorts("atlas", true), subject: plexName, want: verdict{"reachable", "", "", "none", true}, by: "seedbox", writes: []string{"pcp map", "pcp delete", "pcp map"}},
				{play: removePlex, writes: []string{"pcp map", "pcp delete", "pcp map", "pcp delete"}},
			},
		},
		{
			name: "remote-mappable on a Worker", scenario: "remote-mappable", plex: "http://plex.mappable.localhost:{port}",
			edit: plexOnDen,
			rows: []matrixRow{
				{subject: plexName, want: verdict{"fixable", "outside.connect", "mappable-plex", "warn", true}},
				{play: mapPorts("den", true), subject: plexName, want: verdict{"reachable", "", "", "none", true}, by: "seedbox", writes: []string{"pcp map"}},
				{play: removeNode("den"), writes: []string{"pcp map", "pcp delete"}},
			},
		},
		{
			name: "remote-cgnat", scenario: "remote-cgnat", plex: "http://plex.cgnat.localhost:{port}",
			rows: []matrixRow{{subject: plexName, want: verdict{"needs-relay", "outside.connect", "cgnat", "fail", true}, facts: []string{"plex-relay-quality"}}},
		},
		{
			name: "remote-double-nat", scenario: "remote-double-nat", plex: "http://plex.double.localhost:{port}",
			rows: []matrixRow{{subject: plexName, want: verdict{"fixable", "outside.connect", "double-nat", "warn", true}}},
		},
		{
			name: "remote-double-nat forwarded", scenario: "remote-double-nat", plex: "http://plex.double.localhost:{port}",
			edit: func(s *scenario.Scenario) { s.Network.Sites[0].Upstream = "forwarded" },
			rows: []matrixRow{{subject: plexName, want: verdict{"reachable", "", "", "none", true}}},
		},
		{
			name: "remote-vpn", scenario: "remote-vpn", plex: "http://plex.vpn.localhost:{port}",
			rows: []matrixRow{{subject: plexName, want: verdict{"fixable", "outside.connect", "vpn", "warn", true}}},
		},
		{
			name: "remote-own-proxy", scenario: "remote-own-proxy", ui: "https://slipstream.proxy.fake.localhost", plex: "https://plex.proxy.fake.localhost",
			rows: []matrixRow{
				{subject: uiName, want: verdict{"reachable", "", "", "none", true}, facts: []string{"tested-through-proxy"}},
				{subject: plexName, want: verdict{"reachable", "", "", "none", true}},
				{play: proxyFault("tls=expired"), subject: plexName, want: verdict{"fixable", "tls.certificate", "certificate", "warn", true}},
				{play: proxyFault("tls=wrong-name"), subject: uiName, want: verdict{"fixable", "tls.certificate", "certificate", "warn", true}},
				{play: proxyFault("status=401"), subject: uiName, want: verdict{"fixable", "identity", "sign-in-ui", "untested", false}},
				{subject: plexName, want: verdict{"fixable", "identity", "sign-in-server", "warn", true}},
				{play: plexPublic("https://media.proxy.fake.localhost"), subject: plexName, want: verdict{"fixable", "plex.custom-url", "custom-url", "warn", true}},
			},
		},
		{
			name: "remote-single-machine", scenario: "remote-single-machine", plex: "http://plex.single.localhost:{port}",
			rows: []matrixRow{
				{subject: plexName, want: verdict{"fixable", "origin.class", "forward-not-seen", "untested", false}},
				{play: plexMapping("mapped"), subject: plexName, want: verdict{"reachable", "", "", "none", true}},
			},
		},
		{
			name: "remote-ipv6", scenario: "remote-ipv6", plex: "http://plex.v6.localhost:{port}",
			rows: []matrixRow{
				{subject: plexName, want: verdict{"fixable", "outside.connect/ipv6", "firewall", "warn", true}},
				{play: networkCue(`{"site":"home","prefix6":"2001:db8:2::/64"}`), subject: plexName, want: verdict{"fixable", "outside.connect/ipv6", "prefix-change", "warn", true}, fails: "dns.aaaa/ipv6 prefix-change"},
			},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s, err := scenario.Load(c.scenario)
			if err != nil {
				t.Fatal(err)
			}
			if c.edit != nil {
				c.edit(&s)
			}
			w := &remoteWorld{run: worldtest.LaunchScenario(t, s)}
			w.a = api(t, w.run)
			w.nothingWanted(t)
			doctor(t, w.run)
			w.want(t, c.ui, c.plex, s.Nodes[0].MediaServers[0].RunsOn)
			for _, r := range c.rows {
				var since time.Time
				if r.play != nil {
					since = r.play(t, w)
				}
				w.await(t, r, since)
			}
		})
	}
}

const (
	uiName   = "SlipStream UI and portal"
	plexName = "Plex"
)

// verdict is what a row asserts of a route: its verdict, its deciding
// check (with /ipv6 for the IPv6 family's), its fix, its card, and
// whether every family was tested from outside.
type verdict struct {
	Verdict, Deciding, Fix, Card string
	Tested                       bool
}

// matrixRow plays play, then waits until subject's route, judged after
// the time play answers, reads want, carries facts, has the check fails
// names failing with its fix beside the deciding one, and was tested from
// outside by by; and until the routers' write requests are writes, each
// its protocol and op, in order. An empty subject waits for writes alone.
type matrixRow struct {
	play    func(*testing.T, *remoteWorld) time.Time
	subject string
	want    verdict
	facts   []string
	fails   string
	by      string
	writes  []string
}

type remoteWorld struct {
	run *world.Run
	a   *worldtest.API
	// plex is the Plex route's Exposure id.
	plex string
}

type idOnly struct {
	ID string `json:"id"`
}

type remoteRoute struct {
	Exposure struct {
		Subject string `json:"subject"`
	} `json:"exposure"`
	Checked  bool       `json:"checked"`
	Verdict  string     `json:"verdict"`
	Tested   bool       `json:"tested"`
	Card     string     `json:"card"`
	Fix      *idOnly    `json:"fix"`
	JudgedAt *time.Time `json:"judgedAt"`
	Hops     []struct {
		Checks []remoteCheck `json:"checks"`
		Facts  []idOnly      `json:"facts"`
	} `json:"hops"`
	Facts []idOnly `json:"facts"`
}

type remoteCheck struct {
	ID     string  `json:"id"`
	Family string  `json:"family"`
	Result string  `json:"result"`
	Points string  `json:"points"`
	Fact   bool    `json:"fact"`
	Fix    *idOnly `json:"fix"`
	By     string  `json:"by"`
}

// failing are r's failing checks, each its id, /ipv6 for the IPv6
// family's, and its fix.
func (r remoteRoute) failing() []string {
	var out []string
	for _, h := range r.Hops {
		for _, c := range h.Checks {
			if !c.Fact && c.Result == "fails" && c.Fix != nil {
				out = append(out, c.ID+map[string]string{"ipv6": "/ipv6"}[c.Family]+" "+c.Fix.ID)
			}
		}
	}
	return out
}

// verdict is what r reads. Its deciding check is the first failing check
// that points to the verdict with the route's fix, or with none failing
// the first such Not tested one.
func (r remoteRoute) verdict() verdict {
	v := verdict{Verdict: r.Verdict, Card: r.Card, Tested: r.Tested}
	if r.Fix == nil {
		return v
	}
	v.Fix = r.Fix.ID
	for _, result := range []string{"fails", "not-tested"} {
		for _, h := range r.Hops {
			for _, c := range h.Checks {
				if !c.Fact && c.Result == result && c.Points == r.Verdict && c.Fix != nil && c.Fix.ID == v.Fix {
					v.Deciding = c.ID + map[string]string{"ipv6": "/ipv6"}[c.Family]
					return v
				}
			}
		}
	}
	return v
}

// testedBy is the Node whose connection from outside passed.
func (r remoteRoute) testedBy() string {
	for _, h := range r.Hops {
		for _, c := range h.Checks {
			if c.ID == "outside.connect" && c.Result == "passes" {
				return c.By
			}
		}
	}
	return ""
}

func (r remoteRoute) facts() []string {
	out := []string{}
	for _, f := range r.Facts {
		out = append(out, f.ID)
	}
	for _, h := range r.Hops {
		for _, f := range h.Facts {
			out = append(out, f.ID)
		}
	}
	return out
}

func (w *remoteWorld) route(t *testing.T, subject string) remoteRoute {
	t.Helper()
	var page struct {
		Routes []remoteRoute `json:"routes"`
	}
	routes := decode(t, "getRemoteAccess", w.a.Admin("atlas", http.MethodGet, "/api/v1/remote-access", ""), &page).Routes
	i := slices.IndexFunc(routes, func(r remoteRoute) bool { return r.Exposure.Subject == subject })
	if i < 0 {
		t.Fatalf("getRemoteAccess has no route for %s: %+v", subject, routes)
	}
	return routes[i]
}

// await waits until r.subject's route, judged after since, reads r.want,
// carries r.facts, fails r.fails and was tested by r.by, and until the
// routers took r.writes.
func (w *remoteWorld) await(t *testing.T, r matrixRow, since time.Time) {
	t.Helper()
	var last remoteRoute
	var writes []string
	defer func() {
		if t.Failed() {
			t.Logf("%s's route last read %+v; the routers took %q", r.subject, last, writes)
		}
	}()
	worldtest.Until(t, fmt.Sprintf("%s reads %+v and the routers took %q", r.subject, r.want, r.writes), func() (bool, error) {
		writes = w.writes(t)
		if !slices.Equal(writes, r.writes) {
			return false, nil
		}
		if r.subject == "" {
			return true, nil
		}
		last = w.route(t, r.subject)
		has := last.facts()
		return last.Checked && last.JudgedAt.After(since) && last.verdict() == r.want &&
			!slices.ContainsFunc(r.facts, func(f string) bool { return !slices.Contains(has, f) }) &&
			(r.fails == "" || slices.Contains(last.failing(), r.fails)) &&
			(r.by == "" || last.testedBy() == r.by), nil
	})
}

// writes are the write requests the routers took, each its protocol and
// op, in order.
func (w *remoteWorld) writes(t *testing.T) []string {
	t.Helper()
	recs, err := events.Select(w.run.Manifest().RunDir, events.Query{Kind: gateway.Requests.Kind, Match: map[string]string{"write": "true"}})
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, rec := range recs {
		var req gateway.Request
		if err := json.Unmarshal(rec.Data, &req); err != nil {
			t.Fatal(err)
		}
		out = append(out, req.Protocol+" "+req.Op)
	}
	return out
}

// nothingWanted checks that with nothing wanted no Node sent a STUN or
// router packet, and Run again refuses.
func (w *remoteWorld) nothingWanted(t *testing.T) {
	t.Helper()
	for _, kind := range []string{stun.Requests.Kind, gateway.Requests.Kind} {
		recs, err := events.Select(w.run.Manifest().RunDir, events.Query{Kind: kind})
		if err != nil || len(recs) > 0 {
			t.Fatalf("with nothing wanted the run holds %d %s records, %v; want none", len(recs), kind, err)
		}
	}
	if r := w.a.Admin("atlas", http.MethodPost, "/api/v1/remote-access/run", ""); r.Status != http.StatusConflict {
		t.Fatalf("runRemoteAccess with nothing wanted = %d %s, want 409", r.Status, r.Body)
	}
}

// want sets Address to ui and wants the UI and portal, then wants Plex,
// which runs on plexOn, at plexAt.
func (w *remoteWorld) want(t *testing.T, ui, plexAt, plexOn string) {
	t.Helper()
	m := w.run.Manifest()
	if ui != "" {
		address := withPort(ui, netip.MustParseAddrPort(w.a.Node("atlas").Listeners["http"]).Port())
		if r := w.a.Admin("atlas", http.MethodPut, "/api/v1/identity/address", `{"address":"`+address+`"}`); r.Status != http.StatusOK {
			t.Fatalf("setAddress %s = %d %s", address, r.Status, r.Body)
		}
		w.post(t, `{"kind":"ui-portal"}`)
	}
	i := slices.IndexFunc(m.Fronts, func(f world.Front) bool { return f.Target == plex.Fake.Name && f.Node == plexOn })
	public := withPort(plexAt, netip.MustParseAddrPort(m.Fronts[i].Address).Port())
	var servers struct {
		Servers []idOnly `json:"servers"`
	}
	decode(t, "listMediaServers", w.a.Admin("atlas", http.MethodGet, "/api/v1/media-servers", ""), &servers)
	w.plex = w.post(t, fmt.Sprintf(`{"kind":"media-server","serverId":%q,"public":%q}`, servers.Servers[0].ID, public))
}

func withPort(address string, port uint16) string {
	return strings.ReplaceAll(address, "{port}", strconv.Itoa(int(port)))
}

func (w *remoteWorld) post(t *testing.T, body string) string {
	t.Helper()
	r := w.a.Admin("atlas", http.MethodPost, "/api/v1/remote-access/exposures", body)
	var e idOnly
	if r.Status != http.StatusCreated || json.Unmarshal([]byte(r.Body), &e) != nil {
		t.Fatalf("addExposure %s = %d %s, want 201", body, r.Status, r.Body)
	}
	return e.ID
}

// runAgain presses Run again and answers when.
func (w *remoteWorld) runAgain(t *testing.T) time.Time {
	t.Helper()
	at := time.Now()
	if r := w.a.Admin("atlas", http.MethodPost, "/api/v1/remote-access/run", ""); r.Status != http.StatusAccepted {
		t.Fatalf("runRemoteAccess = %d %s, want 202", r.Status, r.Body)
	}
	return at
}

// denNeverTests checks that the outside dialer saw connections from
// seedbox and none from den, in atlas's home.
func denNeverTests(t *testing.T, w *remoteWorld) time.Time {
	t.Helper()
	for node, some := range map[string]bool{"den": false, "seedbox": true} {
		recs, err := events.Select(w.run.Manifest().RunDir, events.Query{Kind: outside.Connections.Kind, Node: node})
		if err != nil || (len(recs) > 0) != some {
			t.Fatalf("the outside dialer holds %d connections from %s, %v; want some: %t", len(recs), node, err, some)
		}
	}
	return time.Time{}
}

// testerAway stops node, the only Worker that tests from outside, once
// Plex's own check has mapped Plex, which then tests the route alone.
func testerAway(node string) func(*testing.T, *remoteWorld) time.Time {
	return func(t *testing.T, w *remoteWorld) time.Time {
		worldtest.Until(t, "Plex's own check maps Plex", func() (bool, error) {
			recs, err := events.Select(w.run.Manifest().RunDir, events.Query{Kind: plextv.ConnectBacks.Kind, Match: map[string]string{"state": "mapped"}})
			return len(recs) > 0, err
		})
		act(t, w.run, node, world.StopNode)
		return w.runAgain(t)
	}
}

func start(node string) func(*testing.T, *remoteWorld) time.Time {
	return func(t *testing.T, w *remoteWorld) time.Time {
		id := act(t, w.run, node, world.StartNode).NodeID
		worldtest.Until(t, "atlas lists "+node+"'s link up", func() (bool, error) {
			return row(t, w.a.Nodes("atlas"), id).Link.State == "up", nil
		})
		return w.runAgain(t)
	}
}

// networkCue sets the network watch to its shortest interval with no
// wait, and plays the network cue with change: the Node's next watch
// pass sees it and the Controller reruns its routes once.
func networkCue(change string) func(*testing.T, *remoteWorld) time.Time {
	return func(t *testing.T, w *remoteWorld) time.Time {
		body := `{"values":{"reachability.watchEvery":60,"reachability.changeWait":0}}`
		if r := w.a.Admin("atlas", http.MethodPatch, "/api/v1/remote-access/settings", body); r.Status != http.StatusOK {
			t.Fatalf("patchRemoteAccessSettings %s = %d %s", body, r.Status, r.Body)
		}
		at := time.Now()
		w.cue(t, gateway.Network.Name, change)
		return at
	}
}

// secondCue plays the network cue again inside the watch interval of the
// rerun the last cue started, so the Controller marks atlas's address as
// churning.
func secondCue(change string) func(*testing.T, *remoteWorld) time.Time {
	return func(t *testing.T, w *remoteWorld) time.Time {
		rerun := *w.route(t, plexName).JudgedAt
		w.cue(t, gateway.Network.Name, change)
		return rerun
	}
}

func (w *remoteWorld) cue(t *testing.T, name, args string) {
	t.Helper()
	if _, err := w.run.Cue(worldtest.Context(t), name, json.RawMessage(args)); err != nil {
		t.Fatalf("cue %s %s: %v", name, args, err)
	}
}

// proxyFault sets the fault on the Admin's proxy as atlas, the tester,
// reaches it, replacing the last, and presses Run again.
func proxyFault(spelled string) func(*testing.T, *remoteWorld) time.Time {
	return func(t *testing.T, w *remoteWorld) time.Time {
		f, err := fault.Parse(spelled)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.run.SetFault(worldtest.Context(t), fault.Rule{ID: "the-vps", Target: fault.Target(proxy.Fake.Name), Origin: "atlas", Fault: f}); err != nil {
			t.Fatal(err)
		}
		return w.runAgain(t)
	}
}

// plexPublic clears the proxy's fault and moves the Plex route to public.
func plexPublic(public string) func(*testing.T, *remoteWorld) time.Time {
	return func(t *testing.T, w *remoteWorld) time.Time {
		if _, err := w.run.ClearFault(worldtest.Context(t), "the-vps"); err != nil {
			t.Fatal(err)
		}
		at := time.Now()
		if r := w.a.Admin("atlas", http.MethodPatch, "/api/v1/remote-access/exposures/"+w.plex, `{"public":"`+public+`"}`); r.Status != http.StatusOK {
			t.Fatalf("editExposure %s = %d %s, want 200", public, r.Status, r.Body)
		}
		return at
	}
}

// plexMapping plays Plex's own check of its public address.
func plexMapping(state string) func(*testing.T, *remoteWorld) time.Time {
	return func(t *testing.T, w *remoteWorld) time.Time {
		w.cue(t, plextv.PlexMapping.Name, `{"server":"`+plexName+`","state":"`+state+`"}`)
		return w.runAgain(t)
	}
}

// mapPorts sets node's "Open router ports automatically". The check that
// change starts may read node's facts before node holds the intent that
// maps or deletes, so once a router took that write it presses Run again.
func mapPorts(node string, on bool) func(*testing.T, *remoteWorld) time.Time {
	return func(t *testing.T, w *remoteWorld) time.Time {
		took := len(w.writes(t))
		body := fmt.Sprintf(`{"values":{"reachability.mapPorts":%t}}`, on)
		if r := w.a.Admin("atlas", http.MethodPatch, "/api/v1/remote-access/nodes/"+w.a.Node(node).NodeID+"/settings", body); r.Status != http.StatusOK {
			t.Fatalf("patchNodeNetworkSettings %s %s = %d %s", node, body, r.Status, r.Body)
		}
		worldtest.Until(t, "a router takes "+node+"'s next write", func() (bool, error) {
			return len(w.writes(t)) > took, nil
		})
		return w.runAgain(t)
	}
}

func removePlex(t *testing.T, w *remoteWorld) time.Time {
	t.Helper()
	if r := w.a.Admin("atlas", http.MethodDelete, "/api/v1/remote-access/exposures/"+w.plex, ""); r.Status != http.StatusNoContent {
		t.Fatalf("removeExposure Plex = %d %s, want 204", r.Status, r.Body)
	}
	return time.Time{}
}

// removeNode removes the Worker node, and checks that a router took its
// one delete before node restarted unpaired.
func removeNode(node string) func(*testing.T, *remoteWorld) time.Time {
	return func(t *testing.T, w *remoteWorld) time.Time {
		was := w.a.Node(node)
		if r := w.a.Admin("atlas", http.MethodDelete, "/api/v1/nodes/"+was.NodeID, ""); r.Status != http.StatusNoContent {
			t.Fatalf("removeNode %s = %d %s, want 204", node, r.Status, r.Body)
		}
		dir := w.run.Manifest().RunDir
		restarted, err := events.Wait(worldtest.Context(t), dir, events.Query{Kind: events.NodeRestarted, Node: node, Match: map[string]string{"oldPid": strconv.Itoa(was.PID)}})
		if err != nil {
			t.Fatalf("events.jsonl has no %s record for %s: %v", events.NodeRestarted, node, err)
		}
		deleted, err := events.Select(dir, events.Query{Kind: gateway.Requests.Kind, Node: node, Match: map[string]string{"op": "delete"}})
		if err != nil || len(deleted) != 1 || deleted[0].Seq > restarted.Seq {
			t.Fatalf("the routers hold %+v, %v from %s; want one delete before its restart at seq %d", deleted, err, node, restarted.Seq)
		}
		return time.Time{}
	}
}

// plexOnDen runs Plex on den, a Worker in atlas's home.
func plexOnDen(s *scenario.Scenario) {
	s.Nodes[0].MediaServers[0].RunsOn = "den"
	s.Nodes = append(s.Nodes, scenario.Node{Name: "den", Setup: scenario.SetupJoin, Site: "home", Firewall: s.Nodes[0].Firewall})
}
