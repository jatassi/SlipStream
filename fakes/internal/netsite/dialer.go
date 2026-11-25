package netsite

import "net"

// Dialer connects from 127.0.0.1, the proxy every Node in a Scenario with
// a network trusts for X-Forwarded-For. Left to the kernel, macOS picks the
// Node's own 127.0.0.N as the source, so the Node would ignore the header.
var Dialer = &net.Dialer{LocalAddr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)}}
