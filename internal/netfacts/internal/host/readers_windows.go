package host

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/jatassi/slipstream/internal/netmodel"
)

// ifTypePropVirtual is IF_TYPE_PROP_VIRTUAL, which WireGuard and Wintun
// adapters report. Go's interface flags show neither as a tunnel.
const ifTypePropVirtual = 53

// tunnelComponents are the driver ComponentIds of VPN adapters, among
// them OpenVPN's TAP adapters, which report an Ethernet IfType.
var tunnelComponents = map[string]bool{"wireguard": true, "wintun": true, "tap0901": true, "tap0801": true, "ovpn-dco": true}

// netClass is the device class of network adapters, whose subkeys name
// each adapter's NetCfgInstanceId and ComponentId.
const netClass = `SYSTEM\CurrentControlSet\Control\Class\{4d36e972-e325-11ce-bfc1-08002be10318}`

func tunnels() (func(net.Interface) bool, error) {
	adapters, err := adapterAddresses()
	if err != nil {
		return nil, err
	}
	components := componentIDs()
	tunnel := map[int]bool{}
	for _, a := range adapters {
		if a.IfType == ifTypePropVirtual || tunnelComponents[components[strings.ToLower(windows.BytePtrToString(a.AdapterName))]] {
			tunnel[adapterIndex(a)] = true
		}
	}
	return func(i net.Interface) bool { return tunnel[i.Index] }, nil
}

// adapterIndex is the index Go's net.Interface gives the adapter.
func adapterIndex(a *windows.IpAdapterAddresses) int {
	if a.IfIndex != 0 {
		return int(a.IfIndex)
	}
	return int(a.Ipv6IfIndex)
}

// componentIDs maps each adapter's lowercase NetCfgInstanceId to its
// lowercase ComponentId. A subkey this user can't open is left out.
func componentIDs() map[string]string {
	out := map[string]string{}
	class, err := registry.OpenKey(registry.LOCAL_MACHINE, netClass, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return out
	}
	defer class.Close() //nolint:errcheck // a read-only key
	names, _ := class.ReadSubKeyNames(-1)
	for _, name := range names {
		k, err := registry.OpenKey(class, name, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		id, _, err1 := k.GetStringValue("NetCfgInstanceId")
		component, _, err2 := k.GetStringValue("ComponentId")
		_ = k.Close()
		if err1 == nil && err2 == nil {
			out[strings.ToLower(id)] = strings.ToLower(component)
		}
	}
	return out
}

func adapterAddresses() ([]*windows.IpAdapterAddresses, error) {
	const flags = windows.GAA_FLAG_INCLUDE_GATEWAYS | windows.GAA_FLAG_SKIP_ANYCAST | windows.GAA_FLAG_SKIP_MULTICAST | windows.GAA_FLAG_SKIP_DNS_SERVER
	size := uint32(15000)
	for {
		buf := make([]byte, size)
		first := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0])) //nolint:gosec // G103: the buffer GetAdaptersAddresses fills
		err := windows.GetAdaptersAddresses(windows.AF_UNSPEC, flags, 0, first, &size)
		if errors.Is(err, windows.ERROR_BUFFER_OVERFLOW) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read the network adapters: %w", err)
		}
		var out []*windows.IpAdapterAddresses
		for a := first; a != nil; a = a.Next {
			out = append(out, a)
		}
		return out, nil
	}
}

// Route is the adapter that holds local and that adapter's IPv4 default
// gateway.
func Route(_ context.Context, local netip.Addr) (string, netip.Addr, error) {
	adapters, err := adapterAddresses()
	if err != nil {
		return "", netip.Addr{}, err
	}
	for _, a := range adapters {
		if !holds(a, local) {
			continue
		}
		for g := a.FirstGatewayAddress; g != nil; g = g.Next {
			if gw, ok := netip.AddrFromSlice(g.Address.IP()); ok && gw.Unmap().Is4() && !gw.IsUnspecified() {
				return windows.UTF16PtrToString(a.FriendlyName), gw.Unmap(), nil
			}
		}
		return windows.UTF16PtrToString(a.FriendlyName), netip.Addr{}, nil
	}
	return "", netip.Addr{}, nil
}

func holds(a *windows.IpAdapterAddresses, local netip.Addr) bool {
	for u := a.FirstUnicastAddress; u != nil; u = u.Next {
		if addr, ok := netip.AddrFromSlice(u.Address.IP()); ok && addr.Unmap() == local.Unmap() {
			return true
		}
	}
	return false
}

var (
	iphlpapi            = windows.NewLazySystemDLL("iphlpapi.dll")
	getExtendedTCPTable = iphlpapi.NewProc("GetExtendedTcpTable")
)

// tcpTableOwnerPIDListener is TCP_TABLE_OWNER_PID_LISTENER.
const tcpTableOwnerPIDListener = 3

// Sockets is what listens on each port, from GetExtendedTcpTable, with the
// owner's image path where this user may query the process. Windows sets
// IPV6_V6ONLY by default and the tables don't show it, so an IPv6
// wildcard of another program reads IPv4 Unknown.
func Sockets(_ context.Context, ports []uint16) ([]netmodel.Listener, error) {
	socks, err := tcpListeners()
	return listenersOn(ports, socks), err
}

func tcpListeners() ([]socket, error) {
	var out []socket
	paths := map[uint32]string{}
	for _, fam := range []uint32{windows.AF_INET, windows.AF_INET6} {
		rows, err := tcpTable(fam)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			if _, ok := paths[r.pid]; !ok {
				paths[r.pid] = imagePath(r.pid)
			}
			out = append(out, socket{local: r.local, owner: paths[r.pid], v4: netmodel.Unknown})
		}
	}
	return out, nil
}

type tcpRow struct {
	local netip.AddrPort
	pid   uint32
}

// tcpTable is the listening rows of MIB_TCPTABLE_OWNER_PID or
// MIB_TCP6TABLE_OWNER_PID.
func tcpTable(fam uint32) ([]tcpRow, error) {
	var size uint32
	var buf []byte
	for {
		var p *byte
		if len(buf) > 0 {
			p = &buf[0]
		}
		r, _, _ := getExtendedTCPTable.Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&size)), 0, uintptr(fam), tcpTableOwnerPIDListener, 0) //nolint:gosec // G103: the buffer GetExtendedTcpTable fills
		if errno := windows.Errno(r); errno == windows.ERROR_INSUFFICIENT_BUFFER {
			buf = make([]byte, size)
			continue
		} else if errno != 0 {
			return nil, fmt.Errorf("read the listening sockets: %w", errno)
		}
		break
	}
	return tcpRows(buf, fam), nil
}

func tcpRows(buf []byte, fam uint32) []tcpRow {
	if len(buf) < 4 {
		return nil
	}
	n := int(binary.LittleEndian.Uint32(buf))
	// MIB_TCPROW_OWNER_PID is six DWORDs. MIB_TCP6ROW_OWNER_PID is a
	// 16-byte address, scope, port, a remote address, scope and port,
	// state and pid.
	stride, addrAt, addrLen, portAt, pidAt := 24, 4, 4, 8, 20
	if fam == windows.AF_INET6 {
		stride, addrAt, addrLen, portAt, pidAt = 56, 0, 16, 20, 52
	}
	out := make([]tcpRow, 0, n)
	for i := range n {
		row := buf[4+i*stride : 4+(i+1)*stride]
		addr, _ := netip.AddrFromSlice(row[addrAt : addrAt+addrLen])
		port := binary.BigEndian.Uint16(row[portAt : portAt+2])
		out = append(out, tcpRow{local: netip.AddrPortFrom(addr, port), pid: binary.LittleEndian.Uint32(row[pidAt:])})
	}
	return out
}

// imagePath is the full path of the process's executable, empty when
// this user may not query it.
func imagePath(pid uint32) string {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(h) //nolint:errcheck // a query handle
	size := uint32(windows.MAX_LONG_PATH)
	buf := make([]uint16, size)
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:size])
}
