//go:build windows || darwin

package host

import "github.com/jatassi/slipstream/internal/netmodel"

// ownerOf is the program listening on port, on fam when one does, else on
// either family.
func ownerOf(socks []socket, port uint16, fam netmodel.Family) string {
	var either string
	for _, s := range socks {
		if s.local.Port() != port || s.owner == "" {
			continue
		}
		if s.local.Addr().Is4() == (fam == netmodel.IPv4) {
			return s.owner
		}
		either = s.owner
	}
	return either
}
