//go:build linux

package mux_test

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	hashicorp "github.com/hashicorp/yamux"

	"github.com/jatassi/slipstream/internal/link/internal/mux"
	"github.com/jatassi/slipstream/internal/link/internal/wire"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/link/v1/linkv1connect"
)

// The measurement runs the real stack between two network namespaces on a
// veth pair whose ends each delay by half the round trip. The parent test
// builds the topology and re-runs this binary in each namespace, one
// process per side and arm.
const (
	roleEnv     = "TEST_LINK_MEASURE_ROLE"
	armEnv      = "TEST_LINK_MEASURE_ARM"
	outEnv      = "TEST_LINK_MEASURE_OUT"
	serverNS    = "slipmeas-srv"
	clientNS    = "slipmeas-cli"
	serverDev   = "smv-srv"
	clientDev   = "smv-cli"
	serverAddr  = "10.77.0.1"
	clientAddr  = "10.77.0.2"
	serverMAC   = "02:00:0a:4d:00:01"
	clientMAC   = "02:00:0a:4d:00:02"
	measurePort = "8485"
	bufSize     = 1 << 20
	shipped     = "go-yamux"
	controlArm  = "hashicorp-default"
	uhdPeak     = 144.0
)

var (
	warmUp     = 5 * time.Second
	measureFor = 20 * time.Second
	rtts       = []time.Duration{30 * time.Millisecond, 100 * time.Millisecond, 300 * time.Millisecond}
	// streamWindow is link.stream_window's default.
	streamWindow uint32 = 16 << 20
	serverSeed          = [ed25519.SeedSize]byte{1}
	clientSeed          = [ed25519.SeedSize]byte{2}
)

// arm moves bytes over one RPC-kind stream of one mux library. serve runs
// the server's end of a TLS connection and run the client's until c
// closes.
type arm struct {
	name  string
	serve func(c net.Conn) error
	run   func(ctx context.Context, c net.Conn) error
}

var arms = []arm{
	{controlArm, hashicorpServe(256 << 10), hashicorpRun(256 << 10)},
	{shipped, shippedServe, shippedRun},
	{"hashicorp-raised", hashicorpServe(streamWindow), hashicorpRun(streamWindow)},
	{"rpc", rpcServe, rpcRun},
}

func TestMeasureStream(t *testing.T) {
	if os.Getenv("TEST_LINK_MEASURE") != "1" {
		t.Skip("TEST_LINK_MEASURE=1 measures one stream between network namespaces; it needs root, ip and tc")
	}
	switch os.Getenv(roleEnv) {
	case "server":
		measureServer(t, armNamed(t, os.Getenv(armEnv)))
	case "client":
		measureClient(t, armNamed(t, os.Getenv(armEnv)))
	default:
		measureAll(t)
	}
}

func armNamed(t *testing.T, name string) arm {
	for _, a := range arms {
		if a.name == name {
			return a
		}
	}
	t.Fatalf("no arm %q", name)
	return arm{}
}

type result struct {
	Arm       string
	RTT       time.Duration
	Connect   time.Duration
	Mbits     float64
	ServerRSS int64
	Drops     string
}

func measureAll(t *testing.T) {
	var results []result
	for _, rtt := range rtts {
		topology(t, rtt)
		for _, a := range arms {
			r := measureArm(t, a)
			r.RTT = rtt
			results = append(results, r)
		}
	}
	report(results)
	for _, r := range results {
		if r.Arm == controlArm {
			// hashicorp/yamux keeps the whole 256 KiB window outstanding, but
			// the receiver credits it back only in steps of at least half
			// (stream.go sendWindowUpdate), and what it consumed past the last
			// step waits for the next frame. Between half and all of the
			// window comes back each round trip, depending on where frames
			// fall against that step, and a run settles at half or above it.
			// The margins cover per-trip processing; a delay that netem did
			// not apply misses by far more.
			low, high := windowMbits(128<<10, r.RTT)*0.9, windowMbits(256<<10, r.RTT)*1.1
			if r.Mbits < low || r.Mbits > high {
				t.Errorf("%s at %v carried %.1f Mbit/s, outside the window's %.1f to %.1f: harness did not measure the yamux window", r.Arm, r.RTT, r.Mbits, low, high)
			}
		}
		if r.Arm == shipped && r.RTT == 100*time.Millisecond && r.Mbits < uhdPeak {
			t.Errorf("%s carried %.1f Mbit/s at 100 ms, under the %.0f Mbit/s UHD peak", r.Arm, r.Mbits, uhdPeak)
		}
	}
}

func windowMbits(bytes float64, rtt time.Duration) float64 {
	return bytes * 8 / rtt.Seconds() / 1e6
}

// topology makes the two namespaces for rtt, removing them at cleanup.
// Each end knows the other's MAC, so the first connection does not wait a
// round trip for ARP through the delay.
func topology(t *testing.T, rtt time.Duration) {
	cleanup := func() {
		_ = exec.CommandContext(context.Background(), "ip", "netns", "del", serverNS).Run()
		_ = exec.CommandContext(context.Background(), "ip", "netns", "del", clientNS).Run()
	}
	cleanup()
	t.Cleanup(cleanup)
	half := fmt.Sprintf("%dus", rtt.Microseconds()/2)
	ip(t, "netns", "add", serverNS)
	ip(t, "netns", "add", clientNS)
	ip(t, "link", "add", serverDev, "netns", serverNS, "address", serverMAC, "type", "veth", "peer", "name", clientDev, "netns", clientNS, "address", clientMAC)
	for _, end := range [][5]string{{serverNS, serverDev, serverAddr, clientAddr, clientMAC}, {clientNS, clientDev, clientAddr, serverAddr, serverMAC}} {
		ns, dev, addr, peer, peerMAC := end[0], end[1], end[2], end[3], end[4]
		ip(t, "-n", ns, "addr", "add", addr+"/24", "dev", dev)
		ip(t, "-n", ns, "neigh", "add", peer, "lladdr", peerMAC, "dev", dev, "nud", "permanent")
		ip(t, "-n", ns, "link", "set", dev, "up")
		ip(t, "-n", ns, "link", "set", "lo", "up")
		ip(t, "netns", "exec", ns, "tc", "qdisc", "add", "dev", dev, "root", "netem", "delay", half, "limit", "100000")
		ip(t, "netns", "exec", ns, "sysctl", "-q", "-w", "net.ipv4.tcp_rmem=4096 131072 67108864", "net.ipv4.tcp_wmem=4096 16384 67108864")
	}
}

func ip(t *testing.T, args ...string) {
	t.Helper()
	if out, err := exec.CommandContext(t.Context(), "ip", args...).CombinedOutput(); err != nil {
		t.Fatalf("ip %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// measureArm runs a's server and client in their namespaces.
func measureArm(t *testing.T, a arm) result {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "result.json")
	child := func(ns, role string) *exec.Cmd {
		cmd := exec.CommandContext(t.Context(), "ip", "netns", "exec", ns, self, "-test.run", "^TestMeasureStream$")
		cmd.Env = append(os.Environ(), roleEnv+"="+role, armEnv+"="+a.name, outEnv+"="+out)
		return cmd
	}
	server := child(serverNS, "server")
	server.Stderr = os.Stderr
	ready, err := server.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	if err := awaitReady(ready); err != nil {
		t.Fatalf("%s server: %v", a.name, err)
	}
	go func() { _, _ = io.Copy(io.Discard, ready) }()
	if out, err := child(clientNS, "client").CombinedOutput(); err != nil {
		t.Fatalf("%s client: %v\n%s", a.name, err, out)
	}
	if err := server.Wait(); err != nil {
		t.Fatalf("%s server: %v", a.name, err)
	}
	var r result
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &r); err != nil {
		t.Fatal(err)
	}
	if usage, ok := server.ProcessState.SysUsage().(*syscall.Rusage); ok {
		r.ServerRSS = usage.Maxrss << 10
	}
	r.Drops = drops(t, serverNS, serverDev) + "/" + drops(t, clientNS, clientDev)
	return r
}

// awaitReady reads the server's output up to its "ready" line.
func awaitReady(r io.Reader) error {
	lines := bufio.NewScanner(r)
	for lines.Scan() {
		if lines.Text() == "ready" {
			return nil
		}
	}
	return errors.New("exited before it was ready")
}

var dropped = regexp.MustCompile(`dropped (\d+)`)

func drops(t *testing.T, ns, dev string) string {
	out, err := exec.CommandContext(t.Context(), "ip", "netns", "exec", ns, "tc", "-s", "qdisc", "show", "dev", dev).CombinedOutput()
	if err != nil {
		t.Fatalf("tc -s qdisc: %v\n%s", err, out)
	}
	if m := dropped.FindSubmatch(out); m != nil {
		return string(m[1])
	}
	return "?"
}

func report(results []result) {
	var b strings.Builder
	b.WriteString("| arm | RTT | TCP connect | Mbit/s | server peak RSS | netem drops (srv/cli) |\n|---|---|---|---|---|---|\n")
	for _, r := range results {
		fmt.Fprintf(&b, "| %s | %v | %v | %.1f | %.1f MiB | %s |\n", r.Arm, r.RTT, r.Connect.Round(time.Millisecond), r.Mbits, float64(r.ServerRSS)/(1<<20), r.Drops)
	}
	fmt.Print("\n" + b.String() + "\n")
	if path := os.Getenv("GITHUB_STEP_SUMMARY"); path != "" {
		_ = os.WriteFile(path, []byte("## link-measure\n\n"+b.String()), 0o600)
	}
}

func certificate(t *testing.T, seed [ed25519.SeedSize]byte) (tls.Certificate, ed25519.PublicKey) {
	key := ed25519.NewKeyFromSeed(seed[:])
	cert, err := wire.Certificate(key)
	if err != nil {
		t.Fatal(err)
	}
	return cert, ed25519.PublicKey(key[ed25519.SeedSize:])
}

func pinned(want ed25519.PublicKey) wire.Accept {
	return func(key ed25519.PublicKey) error {
		if !key.Equal(want) {
			return errors.New("not the pinned key")
		}
		return nil
	}
}

func measureServer(t *testing.T, a arm) {
	cert, _ := certificate(t, serverSeed)
	_, clientKey := certificate(t, clientSeed)
	ln, err := new(net.ListenConfig).Listen(t.Context(), "tcp", net.JoinHostPort(serverAddr, measurePort))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	fmt.Println("ready")
	c, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	tc := tls.Server(c, wire.ServerConfig(cert, pinned(clientKey)))
	if err := tc.HandshakeContext(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := a.serve(tc); err != nil {
		t.Logf("%s server: %v", a.name, err)
	}
}

func measureClient(t *testing.T, a arm) {
	cert, _ := certificate(t, clientSeed)
	_, serverKey := certificate(t, serverSeed)
	start := time.Now()
	c, err := new(net.Dialer).DialContext(t.Context(), "tcp", net.JoinHostPort(serverAddr, measurePort))
	if err != nil {
		t.Fatal(err)
	}
	dialTime := time.Since(start)
	tc := tls.Client(c, wire.ClientConfig(cert, pinned(serverKey)))
	if err := tc.HandshakeContext(t.Context()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	moved := &counted{Conn: tc}
	done := make(chan error, 1)
	go func() { done <- a.run(ctx, moved) }()
	time.Sleep(warmUp)
	from := moved.n.Load()
	time.Sleep(measureFor)
	mbits := float64(moved.n.Load()-from) * 8 / measureFor.Seconds() / 1e6
	cancel()
	_ = tc.Close()
	<-done
	b, err := json.Marshal(result{Arm: a.name, Connect: dialTime, Mbits: mbits})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv(outEnv), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

// counted counts the bytes crossing the client's end of the TLS connection
// either way, as the mux frames them. A mux Write returns only once its
// whole buffer is sent, so counting there would move in 1 MiB steps.
type counted struct {
	net.Conn
	n atomic.Int64
}

func (c *counted) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	c.n.Add(int64(n))
	return n, err //nolint:wrapcheck // the connection's own error, as net.Conn returns it
}

func (c *counted) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	c.n.Add(int64(n))
	return n, err //nolint:wrapcheck // the connection's own error, as net.Conn returns it
}

// incompressible is a 1 MiB random buffer, so a negotiated gzip on the rpc
// arm cannot shrink what the connection carries.
func incompressible() []byte {
	b := make([]byte, bufSize)
	_, _ = rand.Read(b)
	return b
}

func write(st net.Conn) error {
	buf := incompressible()
	for {
		if _, err := st.Write(buf); err != nil {
			return fmt.Errorf("write: %w", err)
		}
	}
}

// discard reads st's kind byte, then discards the rest.
func discard(st net.Conn) error {
	if _, err := wire.ReadKind(st); err != nil {
		return err
	}
	if _, err := io.Copy(io.Discard, st); err != nil {
		return fmt.Errorf("discard: %w", err)
	}
	return nil
}

func hashicorpConfig(window uint32) *hashicorp.Config {
	cfg := hashicorp.DefaultConfig()
	cfg.MaxStreamWindowSize = window
	cfg.LogOutput = io.Discard
	return cfg
}

func hashicorpServe(window uint32) func(net.Conn) error {
	return func(c net.Conn) error {
		s, err := hashicorp.Server(c, hashicorpConfig(window))
		if err != nil {
			return fmt.Errorf("hashicorp server: %w", err)
		}
		defer func() { _ = s.Close() }()
		st, err := s.Accept()
		if err != nil {
			return fmt.Errorf("hashicorp accept: %w", err)
		}
		return discard(st)
	}
}

func hashicorpRun(window uint32) func(context.Context, net.Conn) error {
	return func(_ context.Context, c net.Conn) error {
		s, err := hashicorp.Client(c, hashicorpConfig(window))
		if err != nil {
			return fmt.Errorf("hashicorp client: %w", err)
		}
		defer func() { _ = s.Close() }()
		st, err := s.Open()
		if err != nil {
			return fmt.Errorf("hashicorp open: %w", err)
		}
		if _, err := st.Write([]byte{byte(wire.RPC)}); err != nil {
			return fmt.Errorf("write stream kind: %w", err)
		}
		return write(st)
	}
}

func shippedConfig() mux.Config {
	return mux.Config{Keepalive: 25 * time.Second, WriteTimeout: 10 * time.Second, StreamWindow: streamWindow}
}

func shippedServe(c net.Conn) error {
	s, err := mux.Server(c, shippedConfig())
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()
	st, err := s.Accept()
	if err != nil {
		return err
	}
	return discard(st)
}

func shippedRun(ctx context.Context, c net.Conn) error {
	s, err := mux.Client(c, shippedConfig())
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()
	st, err := wire.Open(ctx, s, wire.RPC)
	if err != nil {
		return err
	}
	return write(st)
}

// artifacts streams 1 MiB FetchArtifact messages until the call ends, the
// shape of an update download.
type artifacts struct {
	linkv1connect.UnimplementedUpdateServiceHandler
}

func (artifacts) FetchArtifact(_ context.Context, _ *linkv1.FetchArtifactRequest, stream linkv1connect.UpdateServiceFetchArtifactServerStream) error {
	msg := &linkv1.FetchArtifactResponse{Data: incompressible()}
	for {
		if err := stream.Send(msg); err != nil {
			return err
		}
	}
}

func rpcServe(c net.Conn) error {
	s, err := mux.Server(c, shippedConfig())
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()
	srv := connect.NewServer()
	linkv1connect.RegisterUpdateServiceHandler(srv, artifacts{})
	routes := http.NewServeMux()
	connecthttp.Mount(routes, srv)
	rpc := wire.NewConns()
	h := wire.Server(streamWindow, routes, nil, slog.New(slog.DiscardHandler))
	go func() { _ = h.Serve(rpc) }()
	defer func() { _ = h.Close() }()
	for {
		st, err := s.Accept()
		if err != nil {
			return nil
		}
		if _, err := wire.ReadKind(st); err != nil {
			return err
		}
		go rpc.Push(st)
	}
}

func rpcRun(ctx context.Context, c net.Conn) error {
	s, err := mux.Client(c, shippedConfig())
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()
	tr := wire.Transport(streamWindow, func(ctx context.Context) (net.Conn, error) { return wire.Open(ctx, s, wire.RPC) })
	defer tr.CloseIdleConnections()
	client := linkv1connect.NewUpdateServiceClient(connect.NewClient(connecthttp.NewTransport(&http.Client{Transport: tr}, "http://link")))
	stream, err := client.FetchArtifact(ctx, &linkv1.FetchArtifactRequest{Name: "measure"})
	if err != nil {
		return err
	}
	defer func() { _ = stream.Close() }()
	for {
		if _, err := stream.Receive(); err != nil {
			return err
		}
	}
}
