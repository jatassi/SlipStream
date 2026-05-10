// Package netutil provides small networking helpers shared across HTTP clients.
package netutil

import (
	"net/url"
	"strings"
)

// NormalizeLoopbackHost returns "127.0.0.1" if host is "localhost"
// (case-insensitive), otherwise returns host unchanged.
//
// On platforms where the system resolver returns ::1 first for "localhost"
// (notably Windows), Go's HTTP transport can fail to connect when the target
// service binds IPv4 only. Routing through 127.0.0.1 explicitly avoids the
// ambiguity. Callers that have an explicit IPv6 literal (e.g. "[::1]") are
// preserved as written.
func NormalizeLoopbackHost(host string) string {
	if strings.EqualFold(host, "localhost") {
		return "127.0.0.1"
	}
	return host
}

// NormalizeLoopbackURL parses rawURL and rewrites a "localhost" host to
// "127.0.0.1", preserving scheme, port, path, query, and userinfo. If the
// URL cannot be parsed, the input is returned unchanged.
func NormalizeLoopbackURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return rawURL
	}
	if !strings.EqualFold(u.Hostname(), "localhost") {
		return rawURL
	}
	if port := u.Port(); port != "" {
		u.Host = "127.0.0.1:" + port
	} else {
		u.Host = "127.0.0.1"
	}
	return u.String()
}
