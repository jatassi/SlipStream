package faketest

import (
	"bytes"
	"errors"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/fake/internal/fixture"
	"github.com/jatassi/slipstream/fakes/internal/nodeenv"
	"github.com/jatassi/slipstream/fakes/scenario"
)

// TestCATrustedThroughSSLCertFile settles, per OS, that a Node trusts the
// run CA through SSL_CERT_FILE in exactly the world's Node environment.
func TestCATrustedThroughSSLCertFile(t *testing.T) {
	s, err := scenario.Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	authority := newCA(t)
	l, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	url := serve(t, fixture.Fake, s, l, &http.Transport{}, authority).Address.URL("/whoami")

	dir := t.TempDir()
	caFile := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(caFile, authority.PEM(), 0o600); err != nil {
		t.Fatal(err)
	}
	probe := buildProbe(t, dir)
	env := nodeenv.Environ(caFile)

	if got := runProbe(t, probe, url, env); got != "exit 0: ok fixture\n" {
		t.Fatalf("probe with SSL_CERT_FILE: %q", got)
	}
	without := slices.DeleteFunc(slices.Clone(env), func(kv string) bool { return strings.HasPrefix(kv, "SSL_CERT_FILE=") })
	if got, want := runProbe(t, probe, url, without), "exit 1: x509: certificate signed by unknown authority\n"; got != want {
		t.Fatalf("probe without SSL_CERT_FILE: %q, want %q", got, want)
	}
}

func buildProbe(t *testing.T, dir string) string {
	t.Helper()
	out := filepath.Join(dir, "probe")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	build := exec.CommandContext(t.Context(), "go", "build", "-o", out, "./testdata/probe")
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build probe: %v\n%s", err, b)
	}
	return out
}

func runProbe(t *testing.T, probe, url string, env []string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), probe, url)
	cmd.Env = env
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	code := 0
	if exit, ok := errors.AsType[*exec.ExitError](err); ok {
		code = exit.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return "exit " + strconv.Itoa(code) + ": " + out.String()
}
