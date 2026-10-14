package tracker_test

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec // v1 piece hashes are SHA-1 by the protocol
	"crypto/tls"
	"encoding/json"
	"errors"
	"log/slog"
	"math/rand/v2"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/internal/ca"
	"github.com/jatassi/slipstream/fakes/internal/front"
	"github.com/jatassi/slipstream/fakes/internal/nodeenv"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/testbin"
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/internal/bencode"
)

// writeTorrent writes payload.torrent announcing to announce for payload
// into dir.
func writeTorrent(t *testing.T, dir, announce string, payload []byte) (string, bencode.InfoHash) {
	t.Helper()
	var pieces []byte
	for off := 0; off < len(payload); off += releases.PieceLength {
		sum := sha1.Sum(payload[off:min(off+releases.PieceLength, len(payload))]) //nolint:gosec // v1 piece hashes
		pieces = append(pieces, sum[:]...)
	}
	meta, err := bencode.Encode(map[string]any{
		"announce": announce,
		"info":     map[string]any{"length": len(payload), "name": "payload.mkv", "piece length": releases.PieceLength, "pieces": pieces, "private": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	m, err := bencode.DecodeMetainfo(meta)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "payload.torrent")
	if err := os.WriteFile(path, meta, 0o600); err != nil {
		t.Fatal(err)
	}
	return path, m.InfoHash
}

// world serves fake-tracker behind a front with a leaf from a run CA, as
// the world does, for one torrent of payload, and a forwarded seeder front
// the tracker hands a leecher, which reaches the peer that announced
// left=0. It returns the torrent file, the CA file and a reader of the
// tracker's announces capture.
func world(t *testing.T, payload []byte) (torrentFile, caFile string, records func() []events.Record) {
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
	runDir := t.TempDir()
	log, err := events.Create(runDir, func(events.Record) []events.Record { return nil })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	logger := slog.New(slog.NewTextHandler(t.Output(), nil))
	trackerFront := fault.Front{Target: fault.Target(tracker.Fake.Name), Origin: "peer", Layers: fault.Conn | fault.TLS | fault.HTTP}
	seedFront := fault.Front{Target: tracker.SeedersTarget("payload"), Origin: "peer", Layers: fault.Conn}
	plan := fault.NewPlan([]fault.Front{trackerFront, seedFront}, nil)
	set := front.NewSet(plan, func(r events.Record) { _, _ = log.Append(r) }, logger)
	tf, err := set.Listen(t.Context(), trackerFront)
	if err != nil {
		t.Fatal(err)
	}
	sf, err := set.Listen(t.Context(), seedFront)
	if err != nil {
		t.Fatal(err)
	}
	at := fake.Address{AddrPort: tf.Addr(), TLS: true}

	s := scenario.Scenario{Seed: "ssl"}
	passkey := releases.Passkey(s)
	torrentFile, hash := writeTorrent(t, dir, at.URL("/announce/"+passkey), payload)

	env := fake.NewEnv(s, tracker.Fake, logger, plan, log)
	env.Releases = &releases.Catalog{Passkey: passkey, Releases: []releases.Release{{Release: scenario.Release{Name: "payload"}, InfoHash: hash}}}
	env.Fronts = fake.Fronts{{Target: seedFront.Target, Node: seedFront.Origin, Address: fake.Address{AddrPort: sf.Addr()}}}
	server, err := tracker.Fake.New(s, env)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := authority.Leaf("127.0.0.1", "tracker.fake.localhost")
	if err != nil {
		t.Fatal(err)
	}
	in := front.NewInbox(tf.Addr())
	tf.Serve(front.Served(in, &tls.Config{Certificates: []tls.Certificate{leaf}, MinVersion: tls.VersionTLS12}))
	records = func() []events.Record {
		t.Helper()
		recs, err := events.Select(runDir, events.Query{Kind: tracker.Announces.Kind})
		if err != nil {
			t.Fatal(err)
		}
		return recs
	}
	sf.Serve(front.Forward(func() (netip.AddrPort, error) {
		for _, r := range records() {
			var a tracker.Announce
			if err := json.Unmarshal(r.Data, &a); err == nil && a.Left == 0 {
				return netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), a.Port), nil
			}
		}
		return netip.AddrPort{}, errors.New("no seeder announced")
	}))
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx, in) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("fake-tracker: %v", err)
		}
		set.Close()
	})
	return torrentFile, caFile, records
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
// listening on 127.0.0.3 then downloads from a seeder on 127.0.0.1 through
// the forwarded front the tracker named.
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
	for _, r := range records() {
		var a tracker.Announce
		if err := json.Unmarshal(r.Data, &a); err != nil {
			t.Fatal(err)
		}
		// The leecher's stopped announce leaves as it exits, so it may
		// reach the tracker after this read.
		if a.Event == "stopped" {
			continue
		}
		announces = append(announces, r.Node+" "+a.Event+" left="+strconv.FormatInt(a.Left, 10))
	}
	want := []string{
		"peer started left=0",
		"peer started left=" + strconv.Itoa(len(payload)),
		"peer completed left=0",
	}
	if !slices.Equal(announces, want) {
		t.Fatalf("the tracker recorded %q, want %q", announces, want)
	}
}
