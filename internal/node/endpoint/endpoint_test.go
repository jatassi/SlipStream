package endpoint_test

import (
	"strconv"
	"testing"

	"github.com/jatassi/slipstream/internal/node/endpoint"
)

func TestParseHostPort(t *testing.T) {
	for in, want := range map[string]string{
		"stun.example.net:3478": "stun.example.net 3478 <nil>",
		"[2001:db8::1]:6881":    "2001:db8::1 6881 <nil>",
		"192.0.2.1:80":          "192.0.2.1 80 <nil>",
		"stun.example.net":      "0 parse host and port: address stun.example.net: missing port in address",
		"bad_name.example:1":    "0 host is neither an IP literal nor a DNS name",
		"host.example:70000":    `0 parse port: strconv.ParseUint: parsing "70000": value out of range`,
	} {
		h, err := endpoint.ParseHostPort(in)
		var got string
		if err != nil {
			got = "0 " + err.Error()
		} else {
			got = h.Hostname() + " " + strconv.Itoa(int(h.Port())) + " <nil>"
		}
		if got != want {
			t.Errorf("ParseHostPort(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestParseURL(t *testing.T) {
	for in, want := range map[string]string{
		"https://api.example.com/3": "api.example.com",
		"udp://127.0.0.1:6969":      "127.0.0.1",
		"/relative":                 "not an absolute URL with a scheme and a host",
		"example.com":               "not an absolute URL with a scheme and a host",
	} {
		u, err := endpoint.ParseURL(in)
		got := u.Hostname()
		if err != nil {
			got = err.Error()
		}
		if got != want {
			t.Errorf("ParseURL(%q) = %s, want %s", in, got, want)
		}
	}
}
