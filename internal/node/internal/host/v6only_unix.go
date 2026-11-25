//go:build unix

package host

import "syscall"

func v6Only(rc syscall.RawConn) (bool, error) {
	var v int
	var opt error
	if err := rc.Control(func(fd uintptr) {
		v, opt = syscall.GetsockoptInt(int(fd), syscall.IPPROTO_IPV6, syscall.IPV6_V6ONLY)
	}); err != nil {
		return false, err //nolint:wrapcheck // FamiliesOf names the read
	}
	return v != 0, opt //nolint:wrapcheck // FamiliesOf names the read
}
