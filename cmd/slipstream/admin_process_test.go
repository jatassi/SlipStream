package main

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/node"
)

var resetLink = regexp.MustCompile(`^http://home\.localhost:8484/reset#token=([A-Za-z0-9_-]{43})\n$`)

func reset(t *testing.T, dir string, args ...string) result {
	t.Helper()
	return slipstream(t, childEnv(), append([]string{"admin", "reset", "--data-dir", dir}, args...)...)
}

func printedToken(t *testing.T, res result) string {
	t.Helper()
	m := resetLink.FindStringSubmatch(res.stdout)
	if res.code != node.ExitOK || m == nil {
		t.Fatalf("admin reset: exit %d, stdout %q, stderr %q; want exit 0 and one Reset link", res.code, res.stdout, res.stderr)
	}
	return m[1]
}

func expectRefused(t *testing.T, res result, code int, stderr string) {
	t.Helper()
	if res.code != code || res.stdout != "" || res.stderr != "slipstream admin reset: "+stderr+"\n" {
		t.Fatalf("admin reset: exit %d, stdout %q, stderr %q; want exit %d, stderr %q", res.code, res.stdout, res.stderr, code, stderr)
	}
}

func signIn(t *testing.T, ready node.Ready, pin string) httpReply {
	t.Helper()
	return call(t, ready, http.MethodPost, "/api/v1/session", `{"email":"admin@example.com","pin":"`+pin+`"}`, nil)
}

func newPIN(t *testing.T, ready node.Ready, token string) httpReply {
	t.Helper()
	return call(t, ready, http.MethodPost, "/api/v1/reset-links/by-token/"+token+"/pin", `{"pin":"5678"}`, nil)
}

func TestAdminResetOffline(t *testing.T) {
	t.Parallel()
	dir := dataDir(t)
	p := startNode(t, dir, listenersOn("127.0.0.2", "home")...)
	setup := p.ready()
	if r := claimController(t, setup, setup.SetupCode); r.status != http.StatusCreated {
		t.Fatalf("claimController = %d %s", r.status, r.body)
	}
	stopNode(t, p, dir)

	expectRefused(t, reset(t, dir, "admin@example.com"), node.ExitUsage,
		"SlipStream has no address for links yet; pass --address <origin>, such as --address https://slipstream.example.com")
	expectRefused(t, reset(t, dir, "--address", "http://10.0.0.1:8484", "admin@example.com"), node.ExitUsage,
		"--address http://10.0.0.1:8484: the address must be a name, not an IP address")
	first := printedToken(t, reset(t, dir, "admin@example.com", "--address", "http://HOME.localhost:8484/"))
	expectRefused(t, reset(t, dir, "--address", "https://other.example", "admin@example.com"), node.ExitUsage,
		"SlipStream's address is already http://home.localhost:8484; leave out --address, or change it in Settings > Address")
	expectRefused(t, reset(t, dir, "nobody@example.com"), node.ExitFailed, "no Account has the email nobody@example.com")
	second := printedToken(t, reset(t, dir, "Admin@Example.com"))

	p = startNode(t, dir, listenersOn("127.0.0.2", "home")...)
	ready := p.ready()
	if r := call(t, ready, http.MethodGet, "/api/v1/reset-links/by-token/"+first, "", nil); !strings.HasPrefix(r.body, `{"state":"dead","reason":"withdrawn","issuedAt":`) {
		t.Fatalf("the replaced link = %d %s, want withdrawn naming nobody", r.status, r.body)
	}
	if r := newPIN(t, ready, first); r.status != http.StatusGone {
		t.Fatalf("the replaced link = %d %s, want 410", r.status, r.body)
	}
	if r := newPIN(t, ready, second); r.status != http.StatusOK {
		t.Fatalf("redeemResetLink = %d %s", r.status, r.body)
	}
	if r := signIn(t, ready, "5678"); r.status != http.StatusOK {
		t.Fatalf("signing in with the new PIN = %d %s", r.status, r.body)
	}
}

func TestAdminResetRunning(t *testing.T) {
	t.Parallel()
	h := startHome(t)
	signedIn := signIn(t, h.ready, "1234")
	cookie := strings.Split(signedIn.header.Get("Set-Cookie"), ";")[0]
	if r := call(t, h.ready, http.MethodPut, "/api/v1/identity/address", `{"address":"http://home.localhost:8484"}`, map[string]string{"X-Api-Key": h.apiKey}); r.status != http.StatusOK {
		t.Fatalf("setAddress = %d %s", r.status, r.body)
	}
	token := printedToken(t, reset(t, h.dir, "admin@example.com"))
	if r := call(t, h.ready, http.MethodGet, "/api/v1/session", "", map[string]string{"Cookie": cookie}); r.status != http.StatusUnauthorized {
		t.Fatalf("the session the reset ended = %d %s, want 401", r.status, r.body)
	}
	expectRefused(t, reset(t, h.dir, "--address", "https://other.example", "admin@example.com"), node.ExitUsage,
		"SlipStream's address is already http://home.localhost:8484; leave out --address, or change it in Settings > Address")
	expectRefused(t, reset(t, h.dir, "nobody@example.com"), node.ExitFailed, "identity.reset: failed: no Account has the email nobody@example.com")
	if r := newPIN(t, h.ready, token); r.status != http.StatusOK {
		t.Fatalf("redeemResetLink = %d %s", r.status, r.body)
	}
	if r := signIn(t, h.ready, "5678"); r.status != http.StatusOK {
		t.Fatalf("signing in with the new PIN = %d %s", r.status, r.body)
	}
}

func TestAdminResetRefusesADataDirThatIsNotAController(t *testing.T) {
	t.Parallel()
	const refusal = "this machine isn't a Controller, so it has no Accounts to reset"
	fresh := dataDir(t)
	expectRefused(t, reset(t, fresh, "admin@example.com"), node.ExitUsage, refusal)
	if _, err := os.Stat(filepath.Join(fresh, "library.db")); !os.IsNotExist(err) {
		t.Fatalf("a refused reset made library.db: %v", err)
	}

	worker := dataDir(t)
	if err := os.WriteFile(filepath.Join(worker, "pairing.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	expectRefused(t, reset(t, worker, "admin@example.com"), node.ExitUsage, refusal)

	unclaimed := dataDir(t)
	startNode(t, unclaimed).ready()
	expectRefused(t, reset(t, unclaimed, "admin@example.com"), node.ExitUsage, refusal)
}
