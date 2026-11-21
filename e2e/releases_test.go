package e2e

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/testbin"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/system"
)

// releases holds one Backup per release, each written by that release's
// binary through go run ./cmd/release tag -backup.
const releases = "../internal/backups/testdata/releases"

type releaseFixture struct {
	Version    string `json:"version"`
	Backup     string `json:"backup"`
	Passphrase string `json:"passphrase"`
	Email      string `json:"email"`
	PIN        string `json:"pin"`
}

// TestEveryReleasedBackupRestores restores each release's Backup with
// slipstream restore onto a fresh data dir and starts the binary under
// test on it: the library migrates to this build's schema, the Node starts
// as the Controller, and the Admin signs in with the Backup's PIN and
// lists the Controller.
func TestEveryReleasedBackupRestores(t *testing.T) {
	t.Parallel()
	dirs, err := os.ReadDir(releases)
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) == 0 {
		t.Fatalf("%s holds no release Backup", releases)
	}
	binary := testbin.Slipstream(t)
	for _, d := range dirs {
		t.Run(d.Name(), func(t *testing.T) {
			t.Parallel()
			restoreRelease(t, binary, filepath.Join(releases, d.Name()))
		})
	}
}

// restoreRelease restores the release Backup in dir and signs in to it.
func restoreRelease(t *testing.T, binary, dir string) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	var f releaseFixture
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	short, err := os.MkdirTemp("", "ss") //nolint:usetesting // t.TempDir's path is too long for the socket limit on macOS
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(short) })
	data := filepath.Join(short, "data")
	pass := filepath.Join(t.TempDir(), "pass")
	if err := os.WriteFile(pass, []byte(f.Passphrase), 0o600); err != nil {
		t.Fatal(err)
	}
	env := []string{
		"SystemRoot=" + os.Getenv("SystemRoot"),
		"SLIPSTREAM_HTTP_LISTEN=127.0.0.1:0", "SLIPSTREAM_LINK_LISTEN=127.0.0.1:0", "SLIPSTREAM_TORRENT_LISTEN=127.0.0.1:0",
		"SLIPSTREAM_ENDPOINTS_INDEXERS_CATALOG=https://127.0.0.1:9/master",
	}
	restore := exec.CommandContext(worldtest.Context(t), binary, "restore", "--data-dir", data, "--passphrase-file", pass, "--old-controller-off", filepath.Join(dir, f.Backup))
	restore.Env = env
	if out, err := restore.CombinedOutput(); err != nil || !strings.HasPrefix(string(out), "Prepared the restore of the Backup from ") {
		t.Fatalf("slipstream restore of the %s Backup = %v: %s", f.Version, err, out)
	}
	ready := startOn(t, binary, data, env)
	if ready.Claim != system.Controller {
		t.Fatalf("the restored %s Backup starts as %s, want the Controller", f.Version, ready.Claim)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	c := &http.Client{Jar: jar}
	base := "http://" + ready.Listeners.HTTP.String() + "/api/v1"
	resp, err := c.Post(base+"/session", "application/json", strings.NewReader(`{"email":`+quote(f.Email)+`,"pin":`+quote(f.PIN)+`}`)) //nolint:noctx // the Node this test started
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("signIn with the %s Backup's PIN = %d, want 200", f.Version, resp.StatusCode)
	}
	resp, err = c.Get(base + "/nodes") //nolint:noctx // as above
	if err != nil {
		t.Fatal(err)
	}
	var rows struct {
		Nodes []struct {
			Self bool `json:"self"`
		} `json:"nodes"`
	}
	err = json.NewDecoder(resp.Body).Decode(&rows)
	_ = resp.Body.Close()
	if err != nil || len(rows.Nodes) != 1 || !rows.Nodes[0].Self {
		t.Fatalf("listNodes after restoring the %s Backup = %d %+v %v, want the Controller alone", f.Version, resp.StatusCode, rows, err)
	}
}

// startOn runs binary on data until the test ends and answers its
// readiness record.
func startOn(t *testing.T, binary, data string, env []string) node.Ready {
	t.Helper()
	run := exec.CommandContext(worldtest.Context(t), binary, "run", "--data-dir", data)
	run.Env = env
	stderr, err := run.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := run.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		stop := exec.Command(binary, "stop", "--data-dir", data) //nolint:noctx // the test's context has ended
		stop.Env = env
		_ = stop.Run()
		_ = run.Wait()
	})
	lines := bufio.NewScanner(stderr)
	lines.Buffer(make([]byte, 0, 1<<16), 1<<20)
	for lines.Scan() {
		if r, err := node.ParseReady(lines.Bytes()); err == nil {
			go func() { _, _ = io.Copy(io.Discard, stderr) }()
			return r
		}
	}
	t.Fatalf("slipstream run on %s exited before it was ready", data)
	return node.Ready{}
}
