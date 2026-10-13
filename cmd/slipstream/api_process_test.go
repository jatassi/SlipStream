package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/testbin"
	"github.com/jatassi/slipstream/internal/buildinfo"
	"github.com/jatassi/slipstream/internal/node"
)

var setupCodeShape = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{4}-[0-9A-HJKMNP-TV-Z]{4}-[0-9A-HJKMNP-TV-Z]{4}$`)

type httpReply struct {
	status int
	header http.Header
	body   string
}

// call sends one request as curl does: no Origin, no Sec-Fetch-Site.
func call(t *testing.T, ready node.Ready, method, path, body string, headers map[string]string) httpReply {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, "http://"+ready.Listeners.HTTP.String()+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", "curl/8.5.0")
	req.Header.Set("Accept", "*/*")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return httpReply{status: res.StatusCode, header: res.Header, body: string(b)}
}

func expectReply(t *testing.T, r httpReply, status int, body string) {
	t.Helper()
	if r.status != status || r.body != body {
		t.Fatalf("got %d %q, want %d %q", r.status, r.body, status, body)
	}
}

func claimController(t *testing.T, ready node.Ready, code string) httpReply {
	t.Helper()
	return call(t, ready, http.MethodPost, "/api/v1/setup/controller", `{"email":"admin@example.com","displayName":"Admin","pin":"1234"}`, map[string]string{"X-Setup-Code": code})
}

func stopNode(t *testing.T, p *nodeProc, dir string) {
	t.Helper()
	if res := slipstream(t, childEnv(), "stop", "--data-dir", dir); res.code != node.ExitOK {
		t.Fatalf("stop: exit %d, stderr %q", res.code, res.stderr)
	}
	if code := p.wait(); code != node.ExitOK {
		t.Fatalf("node exit %d after stop, want 0:\n%s", code, p.output())
	}
}

// readyRecord is the readiness record as logged, to see which keys it has.
func (p *nodeProc) readyRecord() map[string]any {
	p.t.Helper()
	ready, ok := p.record("ready")["ready"].(map[string]any)
	if !ok {
		p.t.Fatalf("the ready record has no ready object:\n%s", p.output())
	}
	return ready
}

func TestAFreshNodeIsClaimedOverHTTP(t *testing.T) {
	t.Parallel()
	dir := dataDir(t)
	p := startNode(t, dir, "SLIPSTREAM_SEED=fixed")
	ready := p.ready()
	if ready.Listeners.HTTP.Port() == 0 || !ready.Listeners.HTTP.Addr().IsLoopback() {
		t.Fatalf("listeners.http = %s, want a bound loopback address", ready.Listeners.HTTP)
	}
	if !setupCodeShape.MatchString(ready.SetupCode) {
		t.Fatalf("setup_code = %q, want %s", ready.SetupCode, setupCodeShape)
	}
	if logged := p.record("setup code")["code"]; logged != ready.SetupCode {
		t.Fatalf("the setup code record logs %v, readiness reports %s", logged, ready.SetupCode)
	}
	expectReply(t, call(t, ready, http.MethodGet, "/api/v1/health", "", nil), 200, `{"status":"ok"}`+"\n")
	expectReply(t, call(t, ready, http.MethodGet, "/api/v1/status", "", nil), 200, `{"claim":"unclaimed"}`+"\n")

	claimed := claimController(t, ready, ready.SetupCode)
	var res struct {
		APIKey string `json:"apiKey"`
	}
	if err := json.Unmarshal([]byte(claimed.body), &res); err != nil || claimed.status != http.StatusCreated {
		t.Fatalf("claimController = %d %s, want 201", claimed.status, claimed.body)
	}
	if cookie := claimed.header.Get("Set-Cookie"); !strings.HasSuffix(cookie, "; Path=/; Max-Age=2592000; HttpOnly; SameSite=Lax") {
		t.Fatalf("Set-Cookie = %q", cookie)
	}
	if !regexp.MustCompile(`^ssk_[A-Za-z0-9_-]{43}$`).MatchString(res.APIKey) {
		t.Fatalf("apiKey = %q", res.APIKey)
	}

	sys := call(t, ready, http.MethodGet, "/api/v1/system", "", map[string]string{"X-Api-Key": res.APIKey})
	var report struct {
		Build     map[string]any `json:"build"`
		DataDir   string         `json:"dataDir"`
		Seed      string         `json:"seed"`
		Listeners struct {
			HTTP string `json:"http"`
		} `json:"listeners"`
	}
	if err := json.Unmarshal([]byte(sys.body), &report); err != nil || sys.status != http.StatusOK {
		t.Fatalf("getSystem with the API key = %d %s, want 200", sys.status, sys.body)
	}
	if report.DataDir != dir || report.Seed != "fixed" || report.Listeners.HTTP != ready.Listeners.HTTP.String() {
		t.Fatalf("getSystem = %s, want dataDir %s, seed fixed and listeners.http %s", sys.body, dir, ready.Listeners.HTTP)
	}
	if stamped, _ := report.Build["stamped"].(bool); stamped != binaryIsStamped(t) {
		t.Fatalf("getSystem reports build %v, and slipstream version says stamped is %t", report.Build, binaryIsStamped(t))
	}
	expectReply(t, call(t, ready, http.MethodGet, "/api/v1/system", "", nil), 401, `{"status":401,"code":"signed_out","title":"Sign in to continue"}`+"\n")

	stopNode(t, p, dir)
	again := startNode(t, dir)
	readyAgain := again.ready()
	if _, has := again.readyRecord()["setup_code"]; has {
		t.Fatalf("a claimed Node's readiness record has a setup_code: %v", again.readyRecord())
	}
	expectReply(t, call(t, readyAgain, http.MethodGet, "/api/v1/status", "", nil), 200, `{"claim":"controller"}`+"\n")
	expectReply(t, claimController(t, readyAgain, ready.SetupCode), 409, `{"status":409,"code":"already_claimed","title":"This Node is already set up"}`+"\n")
}

func TestTheReleaseBinaryListsTheMovieType(t *testing.T) {
	t.Parallel()
	dir := dataDir(t)
	ready := startNode(t, dir).ready()
	claimed := claimController(t, ready, ready.SetupCode)
	session, _, _ := strings.Cut(strings.TrimPrefix(claimed.header.Get("Set-Cookie"), "slipstream_session="), ";")
	if claimed.status != http.StatusCreated || session == "" {
		t.Fatalf("claimController = %d %s, Set-Cookie %q", claimed.status, claimed.body, claimed.header.Get("Set-Cookie"))
	}
	r := call(t, ready, http.MethodGet, "/api/v1/content-types", "", map[string]string{"Cookie": "slipstream_session=" + session})
	expectReply(t, r, http.StatusOK, `{"contentTypes":[`+
		`{"id":"movie","name":"Movies","enabled":true,"work":{"one":"Movie","many":"Movies"},"levels":[],"item":{"one":"Movie","many":"Movies"},`+
		`"fields":[{"key":"year","label":"Year","tier":"work","kind":"number"},{"key":"released","label":"Released","tier":"work","kind":"date"},`+
		`{"key":"runtime","label":"Runtime","tier":"work","kind":"duration"}]}]}`+"\n")
}

func binaryIsStamped(t *testing.T) bool {
	t.Helper()
	out, err := exec.CommandContext(t.Context(), testbin.Slipstream(t), "version").Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(out) != "slipstream dev (unstamped build)\n"
}

// TestSystemReportsTheBinarysStamp links a stamped binary, as the release
// build does, and proves getSystem reports what slipstream version prints.
func TestSystemReportsTheBinarysStamp(t *testing.T) {
	t.Parallel()
	stamp, err := buildinfo.Encode(buildinfo.Stamp{
		Version:      "2.0.0-test",
		Commit:       strings.Repeat("a", 40),
		Fingerprint:  "sha256:" + strings.Repeat("f", 64),
		Target:       runtime.GOOS + "-" + runtime.GOARCH,
		NativeBundle: "sha256:" + strings.Repeat("b", 64),
		Libtorrent:   "2.0.11",
	})
	if err != nil {
		t.Fatal(err)
	}
	stamped := testbin.Build(t, t.TempDir(), "./cmd/slipstream", "-ldflags", "-X=github.com/jatassi/slipstream/internal/buildinfo.stamp="+stamp)
	version, err := exec.CommandContext(t.Context(), stamped, "version").Output()
	if err != nil {
		t.Fatal(err)
	}
	fingerprint := regexp.MustCompile(`(?m)^fingerprint +(\S+)$`).FindSubmatch(version)
	if fingerprint == nil {
		t.Fatalf("slipstream version printed no fingerprint line:\n%s", version)
	}
	dir := dataDir(t)
	ready := startBinary(t, stamped, dir).ready()
	r := claimController(t, ready, ready.SetupCode)
	var res struct {
		APIKey string `json:"apiKey"`
	}
	if err := json.Unmarshal([]byte(r.body), &res); err != nil || r.status != http.StatusCreated {
		t.Fatalf("claimController = %d %s, want 201", r.status, r.body)
	}
	sys := call(t, ready, http.MethodGet, "/api/v1/system", "", map[string]string{"X-Api-Key": res.APIKey})
	var report struct {
		Build struct {
			Stamped     bool   `json:"stamped"`
			Fingerprint string `json:"fingerprint"`
		} `json:"build"`
	}
	if err := json.Unmarshal([]byte(sys.body), &report); err != nil || sys.status != http.StatusOK {
		t.Fatalf("getSystem = %d %s, want 200", sys.status, sys.body)
	}
	if !report.Build.Stamped || report.Build.Fingerprint != string(fingerprint[1]) {
		t.Fatalf("build = %+v, want stamped with fingerprint %s", report.Build, fingerprint[1])
	}
}

func TestAnUnclaimedRestartDrawsANewCode(t *testing.T) {
	t.Parallel()
	dir := dataDir(t)
	p := startNode(t, dir)
	first := p.ready().SetupCode
	stopNode(t, p, dir)
	again := startNode(t, dir)
	ready := again.ready()
	if ready.SetupCode == first || again.record("setup code")["code"] != ready.SetupCode {
		t.Fatalf("the restart logged %v and reports %s, want a code other than %s", again.record("setup code")["code"], ready.SetupCode, first)
	}
	expectReply(t, claimController(t, ready, first), 403, `{"status":403,"code":"setup_code_wrong","title":"The Setup code is wrong"}`+"\n")
}

func TestAMissingKeyFileRefusesTheStart(t *testing.T) {
	t.Parallel()
	dir := dataDir(t)
	p := startNode(t, dir)
	ready := p.ready()
	if r := claimController(t, ready, ready.SetupCode); r.status != http.StatusCreated {
		t.Fatalf("claimController = %d %s, want 201", r.status, r.body)
	}
	stopNode(t, p, dir)
	if err := os.Remove(filepath.Join(dir, "library.key")); err != nil {
		t.Fatal(err)
	}
	res := slipstream(t, childEnv(localListeners...), "run", "--data-dir", dir)
	want := "library.key is missing, but library.db was set up with one; restore library.key from the Backup that holds this library.db"
	var failed struct{ Msg, Err string }
	for line := range strings.SplitSeq(strings.TrimSpace(res.stderr), "\n") {
		if err := json.Unmarshal([]byte(line), &failed); err == nil && failed.Msg == "start failed" {
			break
		}
	}
	if res.code != node.ExitLibraryRefused || failed.Err != want {
		t.Fatalf("exit %d, start failed with %q; want exit %d with %q\n%s", res.code, failed.Err, node.ExitLibraryRefused, want, res.stderr)
	}
}
