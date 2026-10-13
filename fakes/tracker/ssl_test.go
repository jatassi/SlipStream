package tracker_test

import (
	"bufio"
	"bytes"
	"crypto/sha1" //nolint:gosec // v1 piece hashes are SHA-1 by the protocol
	"crypto/tls"
	"encoding/json"
	"errors"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/ca"
	"github.com/jatassi/slipstream/fakes/internal/nodeenv"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/testbin"
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/internal/bencode"
)

// world serves fake-tracker behind a leaf from a run CA, as the world does,
// for one torrent of payload, and returns the torrent file, the CA file and
// a client that reads the tracker's records.
func world(t *testing.T, payload []byte) (torrentFile, caFile string, records func() []tracker.Announce) {
	t.Helper()
	authority, err := ca.New("ssl")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	caFile = filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(caFile, authority.PEM(), 0o600); err != nil {
		t.Fatal(err)
	}
	l, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	at := fake.Address{AddrPort: netip.MustParseAddrPort(l.Addr().String()), TLS: true}
	leaf, err := authority.Leaf("127.0.0.1", "tracker.fake.localhost")
	if err != nil {
		t.Fatal(err)
	}
	l = tls.NewListener(l, &tls.Config{Certificates: []tls.Certificate{leaf}, MinVersion: tls.VersionTLS12})

	var pieces []byte
	for off := 0; off < len(payload); off += releases.PieceLength {
		sum := sha1.Sum(payload[off:min(off+releases.PieceLength, len(payload))]) //nolint:gosec // v1 piece hashes
		pieces = append(pieces, sum[:]...)
	}
	s := scenario.Scenario{Seed: "ssl"}
	passkey := releases.Passkey(s)
	meta, err := bencode.Encode(map[string]any{
		"announce": at.URL("/announce/" + passkey),
		"info":     map[string]any{"length": len(payload), "name": "payload.mkv", "piece length": releases.PieceLength, "pieces": pieces, "private": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	m, err := bencode.DecodeMetainfo(meta)
	if err != nil {
		t.Fatal(err)
	}
	torrentFile = filepath.Join(dir, "payload.torrent")
	if err := os.WriteFile(torrentFile, meta, 0o600); err != nil {
		t.Fatal(err)
	}

	env := fake.NewEnv(s, tracker.Fake.Name, slog.New(slog.NewTextHandler(t.Output(), nil)))
	env.Releases = &releases.Catalog{Passkey: passkey, Releases: []releases.Release{{InfoHash: m.InfoHash}}}
	server, err := tracker.Fake.New(s, env)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(t.Context(), l) }()
	t.Cleanup(func() {
		if err := <-done; err != nil {
			t.Errorf("fake-tracker: %v", err)
		}
	})
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: authority.Pool(), MinVersion: tls.VersionTLS12}}}
	t.Cleanup(client.CloseIdleConnections)
	return torrentFile, caFile, func() []tracker.Announce {
		t.Helper()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, at.URL("/records"), http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = res.Body.Close() }()
		var got []tracker.Announce
		if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		return got
	}
}

func withoutCertFile(env []string) []string {
	return slices.DeleteFunc(slices.Clone(env), func(kv string) bool { return strings.HasPrefix(kv, "SSL_CERT_FILE=") })
}

func run(t *testing.T, probe string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), probe, args...)
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

// TestLibtorrentTrustsTheRunCAThroughSSLCertFile settles the spec's fact:
// libtorrent's OpenSSL verifies an HTTPS tracker's leaf, with the run CA's
// name constraints and an IP address host, against SSL_CERT_FILE. A peer
// listening on 127.0.0.3 then downloads from a seeder on 127.0.0.1 that the
// tracker named.
func TestLibtorrentTrustsTheRunCAThroughSSLCertFile(t *testing.T) {
	payload := make([]byte, 3*releases.PieceLength+1000)
	_, _ = rand.NewChaCha8([32]byte{'s', 's', 'l'}).Read(payload)
	torrentFile, caFile, records := world(t, payload)
	probe := testbin.Build(t, t.TempDir(), "./fakes/tracker/testdata/peer")
	env := nodeenv.Environ(caFile)

	if got, want := run(t, probe, withoutCertFile(env), "127.0.0.1", torrentFile, t.TempDir()), "exit 1: tracker failed: certificate verify failed (SSL routines)\n"; got != want {
		t.Fatalf("peer without SSL_CERT_FILE: %q, want %q", got, want)
	}
	if got := records(); len(got) != 0 {
		t.Fatalf("the tracker recorded %+v from a peer that does not trust it", got)
	}

	seedDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(seedDir, "payload.mkv"), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	seeder := exec.CommandContext(t.Context(), probe, "127.0.0.1", torrentFile, seedDir, "seed")
	seeder.Env = env
	out, err := seeder.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := seeder.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = seeder.Process.Kill()
		_ = seeder.Wait()
	})
	if line, err := bufio.NewReader(out).ReadString('\n'); line != "announced\n" {
		t.Fatalf("seeder printed %q, %v; want announced", line, err)
	}

	leechDir := t.TempDir()
	if got, want := run(t, probe, env, "127.0.0.3", torrentFile, leechDir), "exit 0: announced\nfinished\n"; got != want {
		t.Fatalf("peer with SSL_CERT_FILE: %q, want %q", got, want)
	}
	got, err := os.ReadFile(filepath.Join(leechDir, "payload.mkv"))
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("the leecher's payload is %d bytes, %v; want the seeder's %d", len(got), err, len(payload))
	}
	var announces []string
	for _, a := range records() {
		// The leecher's stopped announce leaves as it exits, so it may
		// reach the tracker after this read.
		if a.Event == "stopped" {
			continue
		}
		announces = append(announces, a.Source+" "+a.Event+" left="+strconv.FormatInt(a.Left, 10))
	}
	want := []string{
		"127.0.0.1 started left=0",
		"127.0.0.3 started left=" + strconv.Itoa(len(payload)),
		"127.0.0.3 completed left=0",
	}
	if !slices.Equal(announces, want) {
		t.Fatalf("the tracker recorded %q, want %q", announces, want)
	}
}
