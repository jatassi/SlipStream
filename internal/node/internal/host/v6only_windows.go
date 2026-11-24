package host

import (
	"syscall"

	"golang.org/x/sys/windows"
)

func v6Only(rc syscall.RawConn) (bool, error) {
	var v int
	var opt error
	if err := rc.Control(func(fd uintptr) {
		v, opt = windows.GetsockoptInt(windows.Handle(fd), windows.IPPROTO_IPV6, windows.IPV6_V6ONLY)
	}); err != nil {
		return false, err //nolint:wrapcheck // FamiliesOf names the read
	}
	return v != 0, opt //nolint:wrapcheck // FamiliesOf names the read
}
