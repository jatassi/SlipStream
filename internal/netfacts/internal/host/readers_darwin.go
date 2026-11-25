package host

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"net/netip"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/net/route"
	"golang.org/x/sys/unix"

	"github.com/jatassi/slipstream/internal/netmodel"
)

// tunnels are utun interfaces, which WireGuard, Tailscale and the system
// VPN clients make, and point-to-point links.
func tunnels() (func(net.Interface) bool, error) {
	return func(i net.Interface) bool {
		return strings.HasPrefix(i.Name, "utun") || i.Flags&net.FlagPointToPoint != 0
	}, nil
}

// Route is the interface that holds local and that interface's IPv4
// default gateway from the routing table.
func Route(ctx context.Context, local netip.Addr) (string, netip.Addr, error) {
	name, err := ifaceOf(ctx, local)
	if err != nil || name == "" {
		return name, netip.Addr{}, err
	}
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return name, netip.Addr{}, fmt.Errorf("read %s: %w", name, err)
	}
	rib, err := route.FetchRIB(syscall.AF_INET, route.RIBTypeRoute, 0)
	if err != nil {
		return name, netip.Addr{}, fmt.Errorf("read the routes: %w", err)
	}
	msgs, err := route.ParseRIB(route.RIBTypeRoute, rib)
	if err != nil {
		return name, netip.Addr{}, fmt.Errorf("read the routes: %w", err)
	}
	return name, defaultGateway(msgs, iface.Index), nil
}

func defaultGateway(msgs []route.Message, index int) netip.Addr {
	for _, m := range msgs {
		rm, ok := m.(*route.RouteMessage)
		if !ok || rm.Index != index || rm.Flags&syscall.RTF_GATEWAY == 0 || len(rm.Addrs) <= syscall.RTAX_NETMASK {
			continue
		}
		dst, ok1 := rm.Addrs[syscall.RTAX_DST].(*route.Inet4Addr)
		gw, ok2 := rm.Addrs[syscall.RTAX_GATEWAY].(*route.Inet4Addr)
		mask, _ := rm.Addrs[syscall.RTAX_NETMASK].(*route.Inet4Addr)
		if ok1 && ok2 && dst.IP == [4]byte{} && (mask == nil || mask.IP == [4]byte{}) {
			return netip.AddrFrom4(gw.IP)
		}
	}
	return netip.Addr{}
}

// Sockets is what listens on each port, from the tables netstat -anv
// reads, with the owner's executable where this user may read its
// arguments. tcp46 marks a wildcard that takes both families.
func Sockets(ctx context.Context, ports []uint16) ([]netmodel.Listener, error) {
	socks, err := tcpListeners(ctx)
	return listenersOn(ports, socks), err
}

func tcpListeners(ctx context.Context) ([]socket, error) {
	out, err := exec.CommandContext(ctx, "/usr/sbin/netstat", "-anv", "-p", "tcp").Output()
	if err != nil {
		return nil, fmt.Errorf("read the listening sockets: %w", err)
	}
	rows := netstatListening(out)
	socks := make([]socket, len(rows))
	paths := map[int]string{}
	for i, r := range rows {
		if _, ok := paths[r.pid]; !ok {
			paths[r.pid] = execPath(r.pid)
		}
		socks[i] = r.socket
		socks[i].owner = paths[r.pid]
	}
	return socks, nil
}

type netstatRow struct {
	socket
	pid int
}

// netstatListening is the LISTEN rows of netstat -anv -p tcp. The pid
// column is found by its header, "pid" or "process:pid", since its place
// moves between macOS versions.
func netstatListening(out []byte) []netstatRow {
	var rows []netstatRow
	pidAt := -1
	lines := bufio.NewScanner(bytes.NewReader(out))
	for lines.Scan() {
		f := strings.Fields(lines.Text())
		if len(f) > 0 && f[0] == "Proto" {
			// "Local Address" and "Foreign Address" are two header fields
			// each and one row field each.
			pidAt = slices.IndexFunc(f, func(h string) bool { return h == "pid" || strings.HasSuffix(h, ":pid") }) - 2
			continue
		}
		if len(f) < 6 || !strings.HasPrefix(f[0], "tcp") || f[5] != "LISTEN" {
			continue
		}
		at, ok := netstatAddr(f[3], f[0])
		if !ok {
			continue
		}
		r := netstatRow{socket: socket{local: at, v4: netmodel.No}}
		if f[0] == "tcp46" {
			r.v4 = netmodel.Yes
		}
		if pidAt >= 0 && pidAt < len(f) {
			_, pid, _ := strings.Cut(f[pidAt], ":")
			if pid == "" {
				pid = f[pidAt]
			}
			r.pid, _ = strconv.Atoi(pid)
		}
		rows = append(rows, r)
	}
	return rows
}

// netstatAddr parses netstat's address.port, where * is the wildcard of
// the row's family.
func netstatAddr(s, proto string) (netip.AddrPort, bool) {
	i := strings.LastIndexByte(s, '.')
	if i < 0 {
		return netip.AddrPort{}, false
	}
	port, err := strconv.ParseUint(s[i+1:], 10, 16)
	if err != nil {
		return netip.AddrPort{}, false
	}
	host, _, _ := strings.Cut(s[:i], "%")
	var a netip.Addr
	switch {
	case host == "*" && proto == "tcp4":
		a = netip.IPv4Unspecified()
	case host == "*":
		a = netip.IPv6Unspecified()
	default:
		if a, err = netip.ParseAddr(host); err != nil {
			return netip.AddrPort{}, false
		}
	}
	return netip.AddrPortFrom(a, uint16(port)), true
}

// execPath is the executable of process pid from kern.procargs2, which
// starts with argc and then the path. It is empty for a process of
// another user.
func execPath(pid int) string {
	if pid <= 0 {
		return ""
	}
	b, err := unix.SysctlRaw("kern.procargs2", pid)
	if err != nil || len(b) < 4 {
		return ""
	}
	path, _, _ := bytes.Cut(b[4:], []byte{0})
	return string(path)
}
