package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/netip"
	"os"
	"runtime"
	"runtime/pprof"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/curve25519"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun/netstack"
)

const (
	mtu      = 1420
	dataPort = 5201
	upload   = 'U'
	download = 'D'
)

type key [32]byte

func newKey() key {
	var k key
	rand.Read(k[:])
	k[0] &= 248
	k[31] = (k[31] & 127) | 64
	return k
}

func (k key) public() key {
	var p key
	pub, _ := curve25519.X25519(k[:], curve25519.Basepoint)
	copy(p[:], pub)
	return p
}

func parseKey(s string) key {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil || len(b) != 32 {
		log.Fatalf("bad key %q", s)
	}
	return key(b)
}

type peerConfig struct {
	priv, peerPub   key
	listenPort      int
	endpoint        string
	localIP, peerIP netip.Addr
}

type peer struct {
	dev  *device.Device
	tnet *netstack.Net
}

func newPeer(c peerConfig) *peer {
	tdev, tnet, err := netstack.CreateNetTUN([]netip.Addr{c.localIP}, nil, mtu)
	if err != nil {
		log.Fatal(err)
	}
	dev := device.NewDevice(tdev, conn.NewDefaultBind(), device.NewLogger(device.LogLevelError, ""))
	cfg := fmt.Sprintf("private_key=%s\nlisten_port=%d\npublic_key=%s\nallowed_ip=%s/32\n",
		hex.EncodeToString(c.priv[:]), c.listenPort, hex.EncodeToString(c.peerPub[:]), c.peerIP)
	if c.endpoint != "" {
		cfg += "endpoint=" + c.endpoint + "\npersistent_keepalive_interval=25\n"
	}
	if err := dev.IpcSet(cfg); err != nil {
		log.Fatal(err)
	}
	if err := dev.Up(); err != nil {
		log.Fatal(err)
	}
	return &peer{dev, tnet}
}

func cpuSeconds() float64 {
	var ru syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano()).Seconds()
}

type halfCloser interface {
	net.Conn
	CloseWrite() error
}

var payload = make([]byte, 128<<10)

func pump(c net.Conn, d time.Duration) int64 {
	var n int64
	for end := time.Now().Add(d); time.Now().Before(end); {
		w, err := c.Write(payload)
		n += int64(w)
		if err != nil {
			log.Fatalf("write: %v", err)
		}
	}
	c.(halfCloser).CloseWrite()
	return n
}

func drain(c net.Conn) (int64, time.Duration) {
	start := time.Now()
	n, err := io.CopyBuffer(io.Discard, c, make([]byte, 128<<10))
	if err != nil {
		log.Fatalf("read: %v", err)
	}
	return n, time.Since(start)
}

func serve(ln net.Listener, verbose bool) {
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		go func() {
			defer c.Close()
			var hdr [5]byte
			if _, err := io.ReadFull(c, hdr[:]); err != nil {
				return
			}
			d := time.Duration(binary.BigEndian.Uint32(hdr[1:])) * time.Millisecond
			cpu0 := cpuSeconds()
			switch hdr[0] {
			case upload:
				n, el := drain(c)
				fmt.Fprintf(c, "%d %d\n", n, el)
				if verbose {
					report("server recv (upload)", n, el, cpuSeconds()-cpu0)
				}
			case download:
				start := time.Now()
				n := pump(c, d)
				io.Copy(io.Discard, c)
				if verbose {
					report("server send (download)", n, time.Since(start), cpuSeconds()-cpu0)
				}
			}
		}()
	}
}

type result struct {
	bytes  int64
	secs   float64
	cpuSec float64
}

func (r result) mbps() float64     { return float64(r.bytes) * 8 / r.secs / 1e6 }
func (r result) cpuPerGB() float64 { return r.cpuSec / (float64(r.bytes) / 1e9) }

func runTest(dial func() (net.Conn, error), dir byte, d time.Duration) result {
	c, err := dial()
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer c.Close()
	var hdr [5]byte
	hdr[0] = dir
	binary.BigEndian.PutUint32(hdr[1:], uint32(d.Milliseconds()))
	if _, err := c.Write(hdr[:]); err != nil {
		log.Fatal(err)
	}
	cpu0 := cpuSeconds()
	if dir == download {
		n, el := drain(c)
		c.(halfCloser).CloseWrite()
		io.Copy(io.Discard, c)
		return result{n, el.Seconds(), cpuSeconds() - cpu0}
	}
	sent := pump(c, d)
	line, err := bufio.NewReader(c).ReadString('\n')
	if err != nil {
		log.Fatalf("upload ack: %v", err)
	}
	var n, el int64
	fmt.Sscanf(line, "%d %d", &n, &el)
	if n != sent {
		log.Fatalf("server received %d bytes, client sent %d", n, sent)
	}
	return result{n, time.Duration(el).Seconds(), cpuSeconds() - cpu0}
}

func report(label string, n int64, el time.Duration, cpu float64) {
	r := result{n, el.Seconds(), cpu}
	fmt.Printf("%-24s %8.1f Mbit/s  %6.2f GB in %5.2fs  cpu %6.2fs  %5.2f cpu-s/GB\n", label, r.mbps(), float64(n)/1e9, r.secs, cpu, r.cpuPerGB())
}

func freeUDPPort() int {
	c, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	return c.LocalAddr().(*net.UDPAddr).Port
}

var (
	serverIP = netip.MustParseAddr("10.77.0.1")
	clientIP = netip.MustParseAddr("10.77.0.2")
)

func loopbackRun(kind string, d time.Duration) result {
	if kind == "tcp" {
		ln, err := net.Listen("tcp4", "127.0.0.1:0")
		if err != nil {
			log.Fatal(err)
		}
		defer ln.Close()
		go serve(ln, false)
		return runTest(func() (net.Conn, error) { return net.Dial("tcp4", ln.Addr().String()) }, upload, d)
	}
	sk, ck := newKey(), newKey()
	sp, cp := freeUDPPort(), freeUDPPort()
	var s, c *peer
	pprof.Do(context.Background(), pprof.Labels("peer", "server"), func(context.Context) {
		s = newPeer(peerConfig{sk, ck.public(), sp, fmt.Sprintf("127.0.0.1:%d", cp), serverIP, clientIP})
	})
	pprof.Do(context.Background(), pprof.Labels("peer", "client"), func(context.Context) {
		c = newPeer(peerConfig{ck, sk.public(), cp, fmt.Sprintf("127.0.0.1:%d", sp), clientIP, serverIP})
	})
	defer s.dev.Close()
	defer c.dev.Close()
	ln, err := s.tnet.ListenTCPAddrPort(netip.AddrPortFrom(serverIP, dataPort))
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	go serve(ln, false)
	return runTest(func() (net.Conn, error) {
		return c.tnet.DialContext(context.Background(), "tcp", netip.AddrPortFrom(serverIP, dataPort).String())
	}, upload, d)
}

func loopback(args []string) {
	fs := flag.NewFlagSet("loopback", flag.ExitOnError)
	kinds := fs.String("kinds", "tcp,wg", "comma list of tcp,wg")
	procs := fs.String("procs", fmt.Sprint(runtime.NumCPU())+",2,1", "comma list of GOMAXPROCS values")
	reps := fs.Int("reps", 3, "repetitions per config")
	dur := fs.Duration("dur", 10*time.Second, "seconds per run")
	prof := fs.String("cpuprofile", "", "write a CPU profile covering all runs")
	fs.Parse(args)

	type cfg struct {
		kind  string
		procs int
	}
	var cfgs []cfg
	for _, k := range strings.Split(*kinds, ",") {
		for _, p := range strings.Split(*procs, ",") {
			n, _ := strconv.Atoi(p)
			cfgs = append(cfgs, cfg{k, n})
		}
	}
	if *prof != "" {
		f, err := os.Create(*prof)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	results := make(map[cfg][]result)
	for rep := 0; rep < *reps; rep++ {
		for _, c := range cfgs {
			runtime.GOMAXPROCS(c.procs)
			r := loopbackRun(c.kind, *dur)
			runtime.GOMAXPROCS(runtime.NumCPU())
			fmt.Printf("rep %d  %-3s procs=%-2d %9.1f Mbit/s  %6.2f cpu-s/GB  (%.2f GB, cpu %.2fs / wall %.2fs)\n",
				rep+1, c.kind, c.procs, r.mbps(), r.cpuPerGB(), float64(r.bytes)/1e9, r.cpuSec, r.secs)
			results[c] = append(results[c], r)
			runtime.GC()
		}
	}
	fmt.Printf("\n| config | Mbit/s median | min-max | CPU-s/GB median | cores busy |\n|---|---|---|---|---|\n")
	for _, c := range cfgs {
		rs := results[c]
		mb := make([]float64, len(rs))
		cg := make([]float64, len(rs))
		busy := make([]float64, len(rs))
		for i, r := range rs {
			mb[i], cg[i], busy[i] = r.mbps(), r.cpuPerGB(), r.cpuSec/r.secs
		}
		slices.Sort(mb)
		slices.Sort(cg)
		slices.Sort(busy)
		fmt.Printf("| %s GOMAXPROCS=%d | %.0f | %.0f-%.0f | %.2f | %.2f |\n", c.kind, c.procs, mb[len(mb)/2], mb[0], mb[len(mb)-1], cg[len(cg)/2], busy[len(busy)/2])
	}
}

func peerFlags(fs *flag.FlagSet, defLocal, defPeer string) (*string, *string, *int, *string, *string, *string) {
	return fs.String("key", "", "own private key (base64)"),
		fs.String("peer-pub", "", "peer public key (base64)"),
		fs.Int("listen-port", 0, "UDP listen port"),
		fs.String("endpoint", "", "peer UDP endpoint host:port"),
		fs.String("ip", defLocal, "own tunnel IP"),
		fs.String("peer-ip", defPeer, "peer tunnel IP")
}

func server(args []string) {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	k, pp, port, ep, ip, pip := peerFlags(fs, serverIP.String(), clientIP.String())
	fs.Parse(args)
	p := newPeer(peerConfig{parseKey(*k), parseKey(*pp), *port, *ep, netip.MustParseAddr(*ip), netip.MustParseAddr(*pip)})
	ln, err := p.tnet.ListenTCPAddrPort(netip.AddrPortFrom(netip.MustParseAddr(*ip), dataPort))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("listening on %s:%d inside the tunnel, UDP %d, GOMAXPROCS=%d\n", *ip, dataPort, *port, runtime.GOMAXPROCS(0))
	serve(ln, true)
}

func client(args []string) {
	fs := flag.NewFlagSet("client", flag.ExitOnError)
	k, pp, port, ep, ip, pip := peerFlags(fs, clientIP.String(), serverIP.String())
	reps := fs.Int("reps", 3, "repetitions per direction")
	dur := fs.Duration("dur", 10*time.Second, "seconds per run")
	fs.Parse(args)
	p := newPeer(peerConfig{parseKey(*k), parseKey(*pp), *port, *ep, netip.MustParseAddr(*ip), netip.MustParseAddr(*pip)})
	target := netip.AddrPortFrom(netip.MustParseAddr(*pip), dataPort).String()
	dial := func() (net.Conn, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return p.tnet.DialContext(ctx, "tcp", target)
	}
	for rep := 1; rep <= *reps; rep++ {
		for _, dir := range []byte{upload, download} {
			r := runTest(dial, dir, *dur)
			name := map[byte]string{upload: "upload   client->server", download: "download server->client"}[dir]
			fmt.Printf("rep %d %s %8.1f Mbit/s  %6.3f GB  client cpu %.2fs = %.2f cpu-s/GB\n", rep, name, r.mbps(), float64(r.bytes)/1e9, r.cpuSec, r.cpuPerGB())
		}
	}
}

func keygen() {
	k := newKey()
	pub := k.public()
	fmt.Printf("private %s\npublic  %s\n", base64.StdEncoding.EncodeToString(k[:]), base64.StdEncoding.EncodeToString(pub[:]))
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: wgbench loopback|server|client|keygen [flags]")
	}
	switch os.Args[1] {
	case "loopback":
		loopback(os.Args[2:])
	case "server":
		server(os.Args[2:])
	case "client":
		client(os.Args[2:])
	case "keygen":
		keygen()
	default:
		log.Fatalf("unknown mode %q", os.Args[1])
	}
}
