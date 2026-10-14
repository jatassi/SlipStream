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

	"github.com/jatassi/slipstream/fakes/internal/fixture"
	"github.com/jatassi/slipstream/fakes/internal/nodeenv"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/testbin"
)

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
	probe := testbin.Build(t, dir, "./fakes/fake/faketest/testdata/probe")
	env := nodeenv.Environ(caFile)

	if got := runProbe(t, probe, url, env); got != "exit 0: ok fixture\n" {
		t.Fatalf("probe with SSL_CERT_FILE: %q", got)
	}
	without := slices.DeleteFunc(slices.Clone(env), func(kv string) bool { return strings.HasPrefix(kv, "SSL_CERT_FILE=") })
	want := "exit 1: x509: certificate signed by unknown authority\n"
	if runtime.GOOS == "darwin" {
		want = "exit 1: x509: “127.0.0.1” certificate is not trusted\n"
	}
	if got := runProbe(t, probe, url, without); got != want {
		t.Fatalf("probe without SSL_CERT_FILE: %q, want %q", got, want)
	}
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
