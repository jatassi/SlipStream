package host

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/netmodel"
)

// procRoot is where the readers find /proc and /sys, so a test can hand
// them a tree of its own.
var procRoot = "/"

// arphrdNone is the link type of a WireGuard or TUN interface, which has
// no link-layer header.
const arphrdNone = "65534"

func tunnels() (func(net.Interface) bool, error) { return tunnel, nil }

func tunnel(i net.Interface) bool {
	if i.Flags&net.FlagPointToPoint != 0 {
		return true
	}
	kind, err := os.ReadFile(filepath.Join(procRoot, "sys", "class", "net", i.Name, "type"))
	return err == nil && strings.TrimSpace(string(kind)) == arphrdNone
}

// Route is the interface that holds local and that interface's default
// gateway from /proc/net/route. IPv6 routes aren't read: STUN, and so the
// egress, is IPv4 only.
func Route(ctx context.Context, local netip.Addr) (string, netip.Addr, error) {
	iface, err := ifaceOf(ctx, local)
	if err != nil || iface == "" {
		return iface, netip.Addr{}, err
	}
	data, err := os.ReadFile(filepath.Join(procRoot, "proc", "net", "route"))
	if err != nil {
		return iface, netip.Addr{}, fmt.Errorf("read the routes: %w", err)
	}
	return iface, defaultGateway(data, iface), nil
}

func defaultGateway(table []byte, iface string) netip.Addr {
	lines := bufio.NewScanner(bytes.NewReader(table))
	for lines.Scan() {
		f := strings.Fields(lines.Text())
		if len(f) < 3 || f[0] != iface || f[1] != "00000000" {
			continue
		}
		b, err := hex.DecodeString(f[2])
		if err != nil || len(b) != 4 {
			continue
		}
		if gw := netip.AddrFrom4([4]byte{b[3], b[2], b[1], b[0]}); !gw.IsUnspecified() {
			return gw
		}
		return netip.Addr{}
	}
	return netip.Addr{}
}

// procSocket is a listening socket in /proc/net/tcp or tcp6.
type procSocket struct {
	local netip.AddrPort
	inode string
}

// tcpListen is the state of a listening socket in /proc/net/tcp.
const tcpListen = "0A"

// Sockets is what listens on each port, from /proc/net/tcp and tcp6, with
// its owner where /proc lets this process see it.
func Sockets(_ context.Context, ports []uint16) ([]netmodel.Listener, error) {
	socks, err := procSockets(ports)
	return listenersOn(ports, socks), err
}

func procSockets(ports []uint16) ([]socket, error) {
	var found []procSocket
	for _, name := range []string{"tcp", "tcp6"} {
		data, err := os.ReadFile(filepath.Join(procRoot, "proc", "net", name)) //nolint:gosec // G304: a /proc path
		if err != nil {
			return nil, fmt.Errorf("read the listening sockets: %w", err)
		}
		found = append(found, listening(data, ports)...)
	}
	owners := ownersOf(found)
	v4 := netmodel.Unknown
	if bindV6Only() == "0" {
		v4 = netmodel.Yes
	}
	out := make([]socket, len(found))
	for i, s := range found {
		out[i] = socket{local: s.local, owner: owners[s.inode], v4: v4}
	}
	return out, nil
}

// listening is the sockets in a /proc/net/tcp or tcp6 table that listen
// on one of ports.
func listening(table []byte, ports []uint16) []procSocket {
	var out []procSocket
	lines := bufio.NewScanner(bytes.NewReader(table))
	for lines.Scan() {
		f := strings.Fields(lines.Text())
		if len(f) < 10 || f[3] != tcpListen {
			continue
		}
		at, ok := procAddr(f[1])
		if ok && slices.Contains(ports, at.Port()) {
			out = append(out, procSocket{local: at, inode: f[9]})
		}
	}
	return out
}

// procAddr parses /proc's ADDR:PORT, whose address is in 32-bit words of
// host order.
func procAddr(s string) (netip.AddrPort, bool) {
	a, p, ok := strings.Cut(s, ":")
	b, err1 := hex.DecodeString(a)
	port, err2 := strconv.ParseUint(p, 16, 16)
	if !ok || err1 != nil || err2 != nil || (len(b) != 4 && len(b) != 16) {
		return netip.AddrPort{}, false
	}
	for i := 0; i < len(b); i += 4 {
		binary.BigEndian.PutUint32(b[i:], binary.NativeEndian.Uint32(b[i:]))
	}
	addr, _ := netip.AddrFromSlice(b)
	return netip.AddrPortFrom(addr, uint16(port)), true
}

// ownersOf names each socket's owning process by its inode, where this
// process may read that process's descriptors.
func ownersOf(socks []procSocket) map[string]string {
	want := map[string]bool{}
	for _, s := range socks {
		want["socket:["+s.inode+"]"] = true
	}
	out := map[string]string{}
	procs, _ := filepath.Glob(filepath.Join(procRoot, "proc", "[0-9]*"))
	for _, proc := range procs {
		fds, _ := os.ReadDir(filepath.Join(proc, "fd"))
		for _, fd := range fds {
			link, err := os.Readlink(filepath.Join(proc, "fd", fd.Name()))
			if err != nil || !want[link] {
				continue
			}
			comm, err := os.ReadFile(filepath.Join(proc, "comm")) //nolint:gosec // G304: a /proc path
			if err == nil {
				out[strings.TrimSuffix(strings.TrimPrefix(link, "socket:["), "]")] = strings.TrimSpace(string(comm))
			}
		}
	}
	return out
}

func bindV6Only() string {
	b, err := os.ReadFile(filepath.Join(procRoot, "proc", "sys", "net", "ipv6", "bindv6only"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// Firewall reads Unknown: nftables and ufw can't be read without
// privilege, and an Allows from firewalld alone would hide every other
// rule.
func Firewall(_ context.Context, _ string, asks []netmodel.Rule) ([]netmodel.Rule, error) {
	return asks, nil
}
