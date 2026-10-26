package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/node"
)

// home is a claimed Controller named home whose listeners are on
// 127.0.0.2, as in the Fake world.
type home struct {
	dir    string
	ready  node.Ready
	apiKey string
}

func listenersOn(addr, name string) []string {
	return []string{
		"SLIPSTREAM_TORRENT_LISTEN=" + addr + ":0",
		"SLIPSTREAM_HTTP_LISTEN=" + addr + ":0",
		"SLIPSTREAM_LINK_LISTEN=" + addr + ":0",
		"SLIPSTREAM_NAME=" + name,
	}
}

var boxEnv = listenersOn("127.0.0.3", "box")

func startHome(t *testing.T) home {
	t.Helper()
	return startHomeWith(t, listenersOn("127.0.0.2", "home"))
}

func startHomeWith(t *testing.T, env []string) home {
	t.Helper()
	dir := dataDir(t)
	p := startNode(t, dir, env...)
	ready := p.ready()
	claimed := claimController(t, ready, ready.SetupCode)
	if claimed.status != http.StatusCreated {
		t.Fatalf("claimController = %d %s", claimed.status, claimed.body)
	}
	return home{dir: dir, ready: ready, apiKey: keyOf(t, ready, claimed)}
}

func (h home) admin(t *testing.T, method, path string, out any) {
	t.Helper()
	r := call(t, h.ready, method, path, "", map[string]string{"X-Api-Key": h.apiKey})
	if r.status/100 != 2 {
		t.Fatalf("%s %s = %d %s", method, path, r.status, r.body)
	}
	if err := json.Unmarshal([]byte(r.body), out); err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
}

func (h home) mint(t *testing.T) string {
	t.Helper()
	var res struct {
		Code string `json:"code"`
	}
	h.admin(t, http.MethodPost, "/api/v1/join-codes", &res)
	return res.Code
}

type listed struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (h home) boxID(t *testing.T) string {
	t.Helper()
	var res struct {
		Nodes []listed `json:"nodes"`
	}
	h.admin(t, http.MethodGet, "/api/v1/nodes", &res)
	for _, n := range res.Nodes {
		if n.Name == "box" {
			return n.ID
		}
	}
	t.Fatalf("home lists no box: %+v", res.Nodes)
	return ""
}

// await is the first record with msg, read or still to come.
func (p *nodeProc) await(msg string) map[string]any {
	p.t.Helper()
	p.mu.Lock()
	seen := slices.Clone(p.stderr)
	p.mu.Unlock()
	for _, line := range seen {
		var rec map[string]any
		if json.Unmarshal([]byte(line), &rec) == nil && rec["msg"] == msg {
			return rec
		}
	}
	timeout := time.After(60 * time.Second)
	for {
		select {
		case line, ok := <-p.lines:
			if !ok {
				p.t.Fatalf("node exited before a %q record:\n%s", msg, p.output())
			}
			var rec map[string]any
			if json.Unmarshal([]byte(line), &rec) == nil && rec["msg"] == msg {
				return rec
			}
		case <-timeout:
			p.t.Fatalf("no %q record after 60s:\n%s", msg, p.output())
		}
	}
}

// TestJoinByEnvironment launches single-machine and starts a second
// binary with SLIPSTREAM_JOIN_CODE: it joins as a Worker, the code reaches
// no log, and a restart with the same code changes nothing.
func TestJoinByEnvironment(t *testing.T) {
	t.Parallel()
	a := worldtest.NewAPI(worldtest.Context(t), t, http.DefaultClient, worldtest.Launch(t, "single-machine").Manifest())
	var minted struct {
		Code string `json:"code"`
	}
	if r := a.Admin("home", http.MethodPost, "/api/v1/join-codes", ""); r.Status != http.StatusCreated || json.Unmarshal([]byte(r.Body), &minted) != nil {
		t.Fatalf("createJoinCode = %d %s", r.Status, r.Body)
	}
	env := slices.Concat(boxEnv, []string{"SLIPSTREAM_JOIN_CODE=" + minted.Code})
	dir := dataDir(t)
	box := startNode(t, dir, env...)
	r := box.ready()
	if r.Claim.String() != "unclaimed" {
		t.Fatalf("box was ready as %s, want unclaimed until its Join", r.Claim)
	}
	if want := (node.Key{Key: "join.code", Value: "SLP1-[redacted]", Default: "", Source: "env", Env: "SLIPSTREAM_JOIN_CODE"}); !slices.Contains(r.Overrides, want) {
		t.Fatalf("overrides = %+v, want %+v among them", r.Overrides, want)
	}
	if claimed := box.await("claimed"); claimed["claim"] != "worker" {
		t.Fatalf("claimed record %v, want claim worker", claimed)
	}
	logged, err := os.ReadFile(filepath.Join(dir, "logs", "slipstream.log"))
	if err != nil {
		t.Fatal(err)
	}
	if holdsCode(box.output(), minted.Code) || holdsCode(string(logged), minted.Code) {
		t.Fatalf("a log holds the Join Code:\n%s\n%s", box.output(), logged)
	}
	var rows []string
	id := ""
	worldtest.Until(t, "home lists box's link up", func() (bool, error) {
		rows, id = nil, ""
		for _, n := range a.Nodes("home") {
			rows = append(rows, n.Name+" "+n.Link.State)
			if n.Name == "box" {
				id = n.ID
			}
		}
		return fmt.Sprint(rows) == fmt.Sprint([]string{"home local", "box up"}), nil
	})

	stopNode(t, box, dir)
	again := startNode(t, dir, env...)
	if r := again.ready(); r.Claim.String() != "worker" {
		t.Fatalf("box restarted as %s, want worker", r.Claim)
	}
	again.await("join code ignored: already joined")
	if rows := a.Nodes("home"); len(rows) != 2 || !slices.ContainsFunc(rows, func(r worldtest.Row) bool { return r.ID == id }) {
		t.Fatalf("after the restart listNodes has %+v, want home and box %s alone", rows, id)
	}
}

func TestJoinOffline(t *testing.T) {
	t.Parallel()
	h := startHome(t)
	code := h.mint(t)
	dir := dataDir(t)
	res := slipstream(t, childEnv(boxEnv...), "join", "--data-dir", dir, code)
	id := h.boxID(t)
	if want := fmt.Sprintf("joined home as box (node %s), linked to %s\n", id, h.ready.Listeners.Link); res.code != node.ExitOK || res.stdout != want {
		t.Fatalf("join: exit %d, stdout %q, stderr %q; want exit 0, stdout %q", res.code, res.stdout, res.stderr, want)
	}
	if runtime.GOOS != "windows" {
		for _, name := range []string{"node.key", "pairing.json"} {
			info, err := os.Stat(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}
			if mode := info.Mode().Perm(); mode != 0o600 {
				t.Errorf("%s has mode %o, want 600", name, mode)
			}
		}
	}

	used := slipstream(t, childEnv(listenersOn("127.0.0.4", "other")...), "join", "--data-dir", dataDir(t), code)
	if want := "join refused: the Join Code was already used\n"; used.code != node.ExitFailed || used.stderr != want {
		t.Fatalf("a used code: exit %d, stderr %q; want exit 1, stderr %q", used.code, used.stderr, want)
	}

	box := startNode(t, dir, boxEnv...)
	if r := box.ready(); r.Claim.String() != "worker" {
		t.Fatalf("the joined Node started as %s, want worker", r.Claim)
	}
	stopNode(t, box, dir)
	if _, err := os.Stat(filepath.Join(dir, "library.db")); !os.IsNotExist(err) {
		t.Fatalf("a Worker's run made library.db: %v", err)
	}
}

func TestJoinRunning(t *testing.T) {
	t.Parallel()
	h := startHome(t)
	dir := dataDir(t)
	box := startNode(t, dir, boxEnv...)
	ready := box.ready()
	expectReply(t, call(t, ready, http.MethodGet, "/api/v1/status", "", nil), 200, `{"claim":"unclaimed","passkeys":{"available":false,"reason":"ip_address"}}`+"\n")
	res := slipstream(t, childEnv(), "join", h.mint(t), "--data-dir", dir)
	id := h.boxID(t)
	if want := fmt.Sprintf("joined home as box (node %s), linked to %s\n", id, h.ready.Listeners.Link); res.code != node.ExitOK || res.stdout != want {
		t.Fatalf("join: exit %d, stdout %q, stderr %q; want exit 0, stdout %q", res.code, res.stdout, res.stderr, want)
	}
	expectReply(t, call(t, ready, http.MethodGet, "/api/v1/status", "", nil), 200, `{"claim":"worker","passkeys":{"available":false,"reason":"ip_address"}}`+"\n")
	if claimed := box.await("claimed"); claimed["claim"] != "worker" {
		t.Fatalf("claimed record %v, want claim worker", claimed)
	}
}

// TestJoinWaitsAtTheNodePortForTheDial joins a stopped Node with a code
// naming an address it can't reach. --node-port writes link.listen, and
// the join prints the Node address at that port, which node-address
// prints too, then fails once await_dial passes with no dial.
func TestJoinWaitsAtTheNodePortForTheDial(t *testing.T) {
	t.Parallel()
	h := startHomeWith(t, append(listenersOn("127.0.0.2", "home"), "SLIPSTREAM_LINK_ADVERTISE=127.0.0.250:8485"))
	dir := dataDir(t)
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("link:\n  listen: 127.0.0.6:0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if res := slipstream(t, childEnv(), "join", "--data-dir", dir, "--node-port", "0", h.mint(t)); res.code != node.ExitUsage {
		t.Fatalf("join --node-port 0: exit %d, stderr %q; want the usage code", res.code, res.stderr)
	}
	free, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.6:0")
	if err != nil {
		t.Fatal(err)
	}
	at := netip.MustParseAddrPort(free.Addr().String())
	if err := free.Close(); err != nil {
		t.Fatal(err)
	}

	env := childEnv("SLIPSTREAM_NAME=box", "SLIPSTREAM_TIMINGS_LINK_JOIN_DIAL=1s", "SLIPSTREAM_TIMINGS_LINK_AWAIT_DIAL=2s")
	res := slipstream(t, env, "join", "--data-dir", dir, "--node-port", strconv.Itoa(int(at.Port())), h.mint(t))
	lines := strings.Split(strings.TrimSuffix(res.stdout, "\n"), "\n")
	if res.code != node.ExitFailed || len(lines) != 4 || lines[0] != "Can't reach the Controller at 127.0.0.250:8485." || !strings.HasPrefix(res.stderr, "join failed: the Controller didn't dial this Node by ") {
		t.Fatalf("join: exit %d, stdout %q, stderr %q; want exit 1 after the awaiting lines", res.code, res.stdout, res.stderr)
	}
	printed, err := joincode.ParseNodeAddress(strings.TrimPrefix(lines[2], "  "))
	if want := []netip.AddrPort{at}; err != nil || !slices.Equal(printed.Addresses(), want) {
		t.Fatalf("join printed the Node address %q (%v), want one at %v", lines[2], err, want)
	}
	if res := slipstream(t, childEnv(), "node-address", "--data-dir", dir); res.code != node.ExitOK || res.stdout != printed.String()+"\n" {
		t.Fatalf("node-address: exit %d, stdout %q, stderr %q; want %s", res.code, res.stdout, res.stderr, printed)
	}
}
