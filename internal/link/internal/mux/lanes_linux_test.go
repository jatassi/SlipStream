//go:build linux

package mux_test

import (
	"cmp"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/jatassi/slipstream/internal/link/internal/mux"
	"github.com/jatassi/slipstream/internal/link/internal/wire"
)

// The lanes measurement settles spec S31's first-build fact: a Transfer's
// chunks over N plain TLS connections against one stream on the Node link,
// across round trip, random loss and line rate. It reuses the stream
// measurement's namespaces and child processes. Data flows from the
// server, which plays the sender, to the client, which plays the receiver.
const (
	lanesEnv  = "TEST_LINK_MEASURE_LANES"
	cellsEnv  = "TEST_LINK_LANES_CELLS"
	lanesRuns = 5
	chunkSize = 4 << 20
	// ahead is timings.transfers.ahead, the chunks each lane keeps wanted.
	ahead                  = 2
	transferKind wire.Kind = 2
	stockRmem              = "4096 131072 6291456"
	stockWmem              = "4096 16384 4194304"
)

type protocol int

const (
	// chunks is transfer.v1's lockstep: the receiver keeps ahead 4 MiB
	// chunks wanted and wants the next as each arrives.
	chunks protocol = iota
	// stream writes as fast as the receiver reads, the ceiling of one
	// connection with no protocol limit.
	stream
)

type laneArm struct {
	name  string
	conns int
	// link carries the bytes on one go-yamux stream of a Node link session
	// with link.stream_window's default, as route 3 does.
	link  bool
	proto protocol
}

var laneArms = []laneArm{
	{"link-chunks", 1, true, chunks},
	{"link-stream", 1, true, stream},
	{"tls-stream", 1, false, stream},
	{"lanes-1", 1, false, chunks},
	{"lanes-2", 2, false, chunks},
	{"lanes-4", 4, false, chunks},
	{"lanes-8", 8, false, chunks},
	{"lanes-16", 16, false, chunks},
}

// cell is one path between the namespaces.
type cell struct {
	rtt  time.Duration
	loss string
	// rate caps each end with tbf. Empty leaves the line uncapped.
	rate string
	// stock keeps Linux's default socket buffer limits instead of 64 MiB.
	stock bool
}

func (c cell) String() string {
	rate, buffers := "uncapped", "64MiB"
	if c.rate != "" {
		rate = c.rate
	}
	if c.stock {
		buffers = "stock"
	}
	return fmt.Sprintf("%v/%s/%s/%s", c.rtt, c.loss, rate, buffers)
}

func lanesCells() []cell {
	var out []cell
	for _, rtt := range rtts {
		for _, loss := range []string{"0%", "0.1%"} {
			out = append(out,
				cell{rtt: rtt, loss: loss, rate: "1gbit"},
				cell{rtt: rtt, loss: loss},
				cell{rtt: rtt, loss: loss, rate: "1gbit", stock: true},
			)
		}
	}
	return out
}

// shape delays each end by half the round trip, drops loss of its packets
// at random and caps it at rate with 50 ms of queue. Segmentation offloads
// are off because netem drops a whole TSO segment, up to 44 packets, as one.
func (c cell) shape(t *testing.T) func(ns, dev string) {
	return func(ns, dev string) {
		in := func(args ...string) { ip(t, append([]string{"netns", "exec", ns}, args...)...) }
		in("ethtool", "-K", dev, "tso", "off", "gso", "off", "gro", "off")
		in("tc", "qdisc", "add", "dev", dev, "root", "handle", "1:", "netem", "delay", half(c.rtt), "loss", c.loss, "limit", "100000")
		if c.rate != "" {
			in("tc", "qdisc", "add", "dev", dev, "parent", "1:1", "handle", "10:", "tbf", "rate", c.rate, "burst", "256kb", "latency", "50ms")
		}
		rmem, wmem := tunedRmem, tunedWmem
		if c.stock {
			rmem, wmem = stockRmem, stockWmem
		}
		in("sysctl", "-q", "-w", "net.ipv4.tcp_congestion_control=cubic", "net.ipv4.tcp_rmem="+rmem, "net.ipv4.tcp_wmem="+wmem)
	}
}

// laneResult is one run of one arm in one cell. The client measures Mbits
// and Chunks; the server, as sender, reads its sockets' TCP_INFO over the
// same window.
type laneResult struct {
	Cell, Arm string
	Run       int
	Mbits     float64
	Chunks    int64
	Errors    int
	ClientCPU float64
	ServerCPU float64
	TCP       tcpWindow
	// Sent and dropped packets at the server end's netem (data direction)
	// and tbf, and the client end's netem (acknowledgements).
	DataPkts, LossDrops, CapDrops, AckLossDrops int64
}

// tcpWindow sums the sender's sockets over the measured window. The
// limited shares are fractions of the window, averaged over connections.
type tcpWindow struct {
	AckedMbits    float64
	DataSegs      uint32
	Retrans       uint32
	Busy          float64
	RwndLimited   float64
	SndbufLimited float64
	// CwndMbits is the sum over connections of cwnd times MSS over the
	// smoothed RTT at the window's end.
	CwndMbits float64
	SRTT      time.Duration
}

func TestMeasureLanes(t *testing.T) {
	if os.Getenv(lanesEnv) != "1" {
		t.Skip(lanesEnv + "=1 measures Transfer lanes between network namespaces. It needs root, ip, tc with netem and ethtool.")
	}
	switch os.Getenv(roleEnv) {
	case "server":
		laneServer(t, laneArmNamed(t, os.Getenv(armEnv)))
	case "client":
		laneClient(t, laneArmNamed(t, os.Getenv(armEnv)))
	default:
		measureLanes(t)
	}
}

func laneArmNamed(t *testing.T, name string) laneArm {
	for _, a := range laneArms {
		if a.name == name {
			return a
		}
	}
	t.Fatalf("no lane arm %q", name)
	return laneArm{}
}

// measureLanes runs every arm lanesRuns times in each selected cell,
// rotating the order each run so warm-up and drift favor no arm. Each run
// gets fresh namespaces, so no cached TCP metrics carry between arms.
func measureLanes(t *testing.T) {
	only := regexp.MustCompile(os.Getenv(cellsEnv))
	var results []laneResult
	for _, c := range lanesCells() {
		if !only.MatchString(c.String()) {
			continue
		}
		for run := range lanesRuns {
			for i := range laneArms {
				a := laneArms[(i+run)%len(laneArms)]
				topology(t, c.shape(t))
				r := measureLane(t, a)
				r.Cell, r.Run = c.String(), run
				b, err := json.Marshal(r)
				if err != nil {
					t.Fatal(err)
				}
				fmt.Printf("lanes-result %s\n", b)
				results = append(results, r)
			}
		}
	}
	lanesReport(results)
	for _, r := range results {
		if r.Errors > 0 || r.Mbits == 0 {
			t.Errorf("%s run %d of %s had %d errors and carried %.1f Mbit/s", r.Arm, r.Run, r.Cell, r.Errors, r.Mbits)
		}
	}
}

func measureLane(t *testing.T, a laneArm) laneResult {
	out, _ := spawn(t, "TestMeasureLanes", a.name)
	var r, sender laneResult
	for path, into := range map[string]*laneResult{out: &r, out + ".server": &sender} {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, into); err != nil {
			t.Fatal(err)
		}
	}
	r.Arm, r.ServerCPU, r.TCP = a.name, sender.ServerCPU, sender.TCP
	srv, cli := qdiscs(t, serverNS, serverDev), qdiscs(t, clientNS, clientDev)
	r.DataPkts, r.LossDrops, r.CapDrops, r.AckLossDrops = srv["netem"].pkts, srv["netem"].dropped, srv["tbf"].dropped, cli["netem"].dropped
	return r
}

type qdiscStats struct{ pkts, dropped int64 }

var qdiscLine = regexp.MustCompile(`(?m)^qdisc (\w+) [^\n]*\n\s*Sent \d+ bytes (\d+) pkt \(dropped (\d+)`)

func qdiscs(t *testing.T, ns, dev string) map[string]qdiscStats {
	out, err := exec.CommandContext(t.Context(), "ip", "netns", "exec", ns, "tc", "-s", "qdisc", "show", "dev", dev).CombinedOutput()
	if err != nil {
		t.Fatalf("tc -s qdisc: %v\n%s", err, out)
	}
	stats := map[string]qdiscStats{}
	for _, m := range qdiscLine.FindAllStringSubmatch(string(out), -1) {
		pkts, _ := strconv.ParseInt(m[2], 10, 64)
		dropped, _ := strconv.ParseInt(m[3], 10, 64)
		stats[m[1]] = qdiscStats{pkts, dropped}
	}
	return stats
}

func laneServer(t *testing.T, a laneArm) {
	cert, _ := certificate(t, serverSeed)
	_, clientKey := certificate(t, clientSeed)
	ln, err := new(net.ListenConfig).Listen(t.Context(), "tcp", net.JoinHostPort(serverAddr, measurePort))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	fmt.Println("ready")
	raw := make([]net.Conn, a.conns)
	for i := range raw {
		c, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		raw[i] = c
	}
	conns := handshake(t, raw, func(c net.Conn) *tls.Conn { return tls.Server(c, wire.ServerConfig(cert, pinned(clientKey))) })
	served := make(chan error, len(conns))
	for _, c := range conns {
		go func() { served <- sendLane(a, c) }()
	}
	time.Sleep(warmUp)
	from, cpuFrom := tcpInfos(t, raw), cpuTime()
	time.Sleep(measureFor)
	to, cpuTo := tcpInfos(t, raw), cpuTime()
	for range conns {
		if err := <-served; err != nil {
			t.Logf("%s server: %v", a.name, err)
		}
	}
	r := laneResult{ServerCPU: (cpuTo - cpuFrom).Seconds() / measureFor.Seconds(), TCP: window(from, to)}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv(outEnv)+".server", b, 0o600); err != nil {
		t.Fatal(err)
	}
}

// handshake runs every connection's TLS handshake at once, as a Transfer
// dials its lanes.
func handshake(t *testing.T, raw []net.Conn, wrap func(net.Conn) *tls.Conn) []*tls.Conn {
	conns := make([]*tls.Conn, len(raw))
	errs := make([]error, len(raw))
	var wg sync.WaitGroup
	for i, c := range raw {
		wg.Go(func() {
			conns[i] = wrap(c)
			errs[i] = conns[i].HandshakeContext(t.Context())
		})
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	return conns
}

func sendLane(a laneArm, c net.Conn) error {
	rw := c
	if a.link {
		s, err := mux.Server(c, shippedConfig())
		if err != nil {
			return err
		}
		defer func() { _ = s.Close() }()
		st, err := s.Accept()
		if err != nil {
			return err
		}
		if _, err := wire.ReadKind(st); err != nil {
			return err
		}
		rw = st
	}
	if a.proto == stream {
		return write(rw)
	}
	return sendChunks(rw)
}

// sendChunks answers each 4-byte Want with the chunk's index and 4 MiB of
// random bytes, in one write.
func sendChunks(rw io.ReadWriter) error {
	frame := make([]byte, 4+chunkSize)
	_, _ = rand.Read(frame[4:])
	for {
		if _, err := io.ReadFull(rw, frame[:4]); err != nil {
			return fmt.Errorf("read want: %w", err)
		}
		if _, err := rw.Write(frame); err != nil {
			return fmt.Errorf("write chunk: %w", err)
		}
	}
}

func laneClient(t *testing.T, a laneArm) {
	cert, _ := certificate(t, clientSeed)
	_, serverKey := certificate(t, serverSeed)
	raw := make([]net.Conn, a.conns)
	dialed := make([]error, a.conns)
	var wg sync.WaitGroup
	for i := range raw {
		wg.Go(func() {
			c, err := new(net.Dialer).DialContext(t.Context(), "tcp", net.JoinHostPort(serverAddr, measurePort))
			if err == nil {
				raw[i] = c
			}
			dialed[i] = err
		})
	}
	wg.Wait()
	for _, err := range dialed {
		if err != nil {
			t.Fatal(err)
		}
	}
	conns := handshake(t, raw, func(c net.Conn) *tls.Conn { return tls.Client(c, wire.ClientConfig(cert, pinned(serverKey))) })
	var stopped atomic.Bool
	var got atomic.Int64
	moved := make([]*counted, len(conns))
	failed := make(chan error, len(conns))
	var receiving sync.WaitGroup
	for i, c := range conns {
		moved[i] = &counted{Conn: c}
		receiving.Go(func() {
			err := receiveLane(t, a, moved[i], &got)
			if !stopped.Load() {
				failed <- err
			}
		})
	}
	total := func() int64 {
		var n int64
		for _, m := range moved {
			n += m.n.Load()
		}
		return n
	}
	time.Sleep(warmUp)
	from, chunksFrom, cpuFrom := total(), got.Load(), cpuTime()
	time.Sleep(measureFor)
	to, chunksTo, cpuTo := total(), got.Load(), cpuTime()
	// The server's window starts when its handshakes finish, half a round
	// trip after the client's, and it samples its sockets at the end.
	time.Sleep(time.Second)
	stopped.Store(true)
	for _, c := range raw {
		_ = c.Close()
	}
	receiving.Wait()
	close(failed)
	r := laneResult{
		Mbits:     float64(to-from) * 8 / measureFor.Seconds() / 1e6,
		Chunks:    chunksTo - chunksFrom,
		ClientCPU: (cpuTo - cpuFrom).Seconds() / measureFor.Seconds(),
	}
	for err := range failed {
		r.Errors++
		t.Logf("%s client: %v", a.name, err)
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv(outEnv), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func receiveLane(t *testing.T, a laneArm, c net.Conn, got *atomic.Int64) error {
	rw := c
	if a.link {
		s, err := mux.Client(c, shippedConfig())
		if err != nil {
			return err
		}
		defer func() { _ = s.Close() }()
		st, err := wire.Open(t.Context(), s, transferKind)
		if err != nil {
			return err
		}
		rw = st
	}
	if a.proto == stream {
		_, err := io.Copy(io.Discard, rw)
		return fmt.Errorf("discard: %w", cmp.Or(err, io.EOF))
	}
	return wantChunks(rw, got)
}

// wantChunks keeps ahead chunks wanted and checks each arrives whole and
// in order.
func wantChunks(rw io.ReadWriter, got *atomic.Int64) error {
	var next uint32
	want := func() error {
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], next)
		next++
		if _, err := rw.Write(b[:]); err != nil {
			return fmt.Errorf("write want: %w", err)
		}
		return nil
	}
	for range ahead {
		if err := want(); err != nil {
			return err
		}
	}
	frame := make([]byte, 4+chunkSize)
	for expect := uint32(0); ; expect++ {
		if _, err := io.ReadFull(rw, frame); err != nil {
			return fmt.Errorf("read chunk: %w", err)
		}
		if i := binary.BigEndian.Uint32(frame); i != expect {
			return fmt.Errorf("chunk %d arrived when %d was next", i, expect)
		}
		got.Add(1)
		if err := want(); err != nil {
			return err
		}
	}
}

func cpuTime() time.Duration {
	var u syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &u)
	return time.Duration(u.Utime.Nano() + u.Stime.Nano())
}

func tcpInfos(t *testing.T, conns []net.Conn) []unix.TCPInfo {
	infos := make([]unix.TCPInfo, len(conns))
	for i, c := range conns {
		tc, ok := c.(syscall.Conn)
		if !ok {
			t.Fatalf("%T has no socket", c)
		}
		sc, err := tc.SyscallConn()
		if err != nil {
			t.Fatal(err)
		}
		var info *unix.TCPInfo
		var ierr error
		if err := sc.Control(func(fd uintptr) { info, ierr = unix.GetsockoptTCPInfo(int(fd), unix.IPPROTO_TCP, unix.TCP_INFO) }); err != nil {
			t.Fatal(err)
		}
		if ierr != nil {
			t.Fatal(ierr)
		}
		infos[i] = *info
	}
	return infos
}

func window(from, to []unix.TCPInfo) tcpWindow {
	var w tcpWindow
	usec := float64(measureFor.Microseconds())
	n := float64(len(to))
	for i := range to {
		a, b := from[i], to[i]
		w.AckedMbits += float64(b.Bytes_acked-a.Bytes_acked) * 8 / measureFor.Seconds() / 1e6
		w.DataSegs += b.Data_segs_out - a.Data_segs_out
		w.Retrans += b.Total_retrans - a.Total_retrans
		w.Busy += float64(b.Busy_time-a.Busy_time) / usec / n
		w.RwndLimited += float64(b.Rwnd_limited-a.Rwnd_limited) / usec / n
		w.SndbufLimited += float64(b.Sndbuf_limited-a.Sndbuf_limited) / usec / n
		if b.Rtt > 0 {
			w.CwndMbits += float64(b.Snd_cwnd) * float64(b.Snd_mss) * 8 / float64(b.Rtt)
		}
		w.SRTT += time.Duration(b.Rtt) * time.Microsecond / time.Duration(len(to))
	}
	return w
}

// lanesReport prints each cell's arms as median Mbit/s with the range over
// runs.
func lanesReport(results []laneResult) {
	var b strings.Builder
	b.WriteString("| cell | arm | median Mbit/s | min | max | runs |\n|---|---|---|---|---|---|\n")
	var cellNames []string
	for _, r := range results {
		if !slices.Contains(cellNames, r.Cell) {
			cellNames = append(cellNames, r.Cell)
		}
	}
	for _, c := range cellNames {
		for _, a := range laneArms {
			var mbits []float64
			for _, r := range results {
				if r.Cell == c && r.Arm == a.name {
					mbits = append(mbits, r.Mbits)
				}
			}
			slices.Sort(mbits)
			fmt.Fprintf(&b, "| %s | %s | %.1f | %.1f | %.1f | %d |\n", c, a.name, mbits[len(mbits)/2], mbits[0], mbits[len(mbits)-1], len(mbits))
		}
	}
	fmt.Print("\n" + b.String() + "\n")
	if path := os.Getenv("GITHUB_STEP_SUMMARY"); path != "" {
		_ = os.WriteFile(path, []byte("## lanes-measure\n\n"+b.String()), 0o600)
	}
}
