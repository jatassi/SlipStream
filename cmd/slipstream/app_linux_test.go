package main

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/testbin"
	"github.com/jatassi/slipstream/internal/node"
)

// browserStub is a $BROWSER that records its arguments in the file it
// returns, as the AppImage smoke's stub does.
func browserStub(t *testing.T) (browser, opened string) {
	t.Helper()
	dir := t.TempDir()
	browser, opened = filepath.Join(dir, "browser"), filepath.Join(dir, "opened")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" > " + opened + ".tmp && mv " + opened + ".tmp " + opened + "\n"
	if err := os.WriteFile(browser, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return browser, opened
}

// awaitOpened is what the browser stub was asked to open.
func awaitOpened(t *testing.T, opened string) string {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		b, err := os.ReadFile(opened)
		if err == nil {
			return string(b)
		}
		if time.Now().After(deadline) {
			t.Fatalf("the browser stub opened nothing after 30s: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// logged is how many records with message msg the Node has logged.
func (p *nodeProc) logged(msg string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := 0
	for _, line := range p.stderr {
		var rec map[string]any
		if json.Unmarshal([]byte(line), &rec) == nil && rec["msg"] == msg {
			n++
		}
	}
	return n
}

func TestRunNeverOpensTheBrowserNorShowsATray(t *testing.T) {
	t.Parallel()
	browser, opened := browserStub(t)
	dir := dataDir(t)
	p := startNode(t, dir, "BROWSER="+browser)
	p.ready()
	if got := statusOf(t, dir); got["mode"] != "run" || got["tray"] != nil || got["start_at_login"] != nil {
		t.Fatalf("status = %v, want mode run with no tray and no start_at_login", got)
	}
	if res := slipstream(t, childEnv(), "stop", "--data-dir", dir); res.code != node.ExitOK {
		t.Fatalf("stop: exit %d, stderr %q", res.code, res.stderr)
	}
	p.wait()
	if _, err := os.Stat(opened); !os.IsNotExist(err) {
		t.Fatalf("run opened the browser: %v", err)
	}
	if n := p.logged("tray"); n != 0 {
		t.Fatalf("run logged %d tray records:\n%s", n, p.output())
	}
}

func TestAppOpensTheSetupPageAndASecondLaunchOpensItAgain(t *testing.T) {
	t.Parallel()
	browser, opened := browserStub(t)
	dir := dataDir(t)
	p := startArgs(t, testbin.Slipstream(t), []string{"app", "--data-dir", dir}, "BROWSER="+browser)
	ready := p.ready()
	url, ok := p.record("setup page")["url"].(string)
	if !ok {
		t.Fatalf("setup page record has no url:\n%s", p.output())
	}
	if got := awaitOpened(t, opened); got != url+"\n" {
		t.Fatalf("app opened %q, want the setup page %s", got, url)
	}
	if rec := p.record("tray"); rec["state"] != "none" {
		t.Fatalf("tray record = %v, want state none", rec)
	}
	want := map[string]any{"pid": float64(ready.PID), "version": ready.Version, "install": "dev", "mode": "app", "claim": "unclaimed", "tray": "none", "url": url}
	if got := statusOf(t, dir); !maps.Equal(got, want) {
		t.Fatalf("status = %v, want %v", got, want)
	}

	second, secondOpened := browserStub(t)
	res := slipstream(t, childEnv(append(slices.Clone(localListeners), "BROWSER="+second)...), "app", "--data-dir", dir)
	if res.code != node.ExitOK {
		t.Fatalf("second app: exit %d, stderr %q", res.code, res.stderr)
	}
	if got := awaitOpened(t, secondOpened); got != url+"\n" {
		t.Fatalf("the second app opened %q, want the running Node's %s", got, url)
	}

	if res := slipstream(t, childEnv(), "stop", "--data-dir", dir); res.code != node.ExitOK {
		t.Fatalf("stop: exit %d, stderr %q", res.code, res.stderr)
	}
	if code := p.wait(); code != node.ExitOK {
		t.Fatalf("app exit %d after stop, want %d:\n%s", code, node.ExitOK, p.output())
	}
}
