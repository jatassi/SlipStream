//go:build !windows

package host

import (
	"context"
	"net/netip"
)

// ifaceOf is the name of the interface that holds local, empty when none
// does.
func ifaceOf(ctx context.Context, local netip.Addr) (string, error) {
	ifs, err := Interfaces(ctx)
	if err != nil {
		return "", err
	}
	for _, i := range ifs {
		for _, a := range i.Addrs {
			if a.Prefix.Addr() == local.Unmap() {
				return i.Name, nil
			}
		}
	}
	return "", nil
}
