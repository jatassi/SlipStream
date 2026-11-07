package clients

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// Address is a client's whole URL: scheme http or https, a host, an
// optional port and a path, with the kind's path filled in. Only
// ParseAddress makes one.
type Address struct {
	u    url.URL
	kind Kind
}

var (
	ErrAddressScheme   = errors.New("the address doesn't start with http:// or https://")
	ErrAddressNotURL   = errors.New("the address isn't a URL")
	ErrAddressHost     = errors.New("the address has no host")
	ErrAddressPort     = errors.New("the port in the address isn't between 1 and 65535")
	ErrSignInInAddress = errors.New("the address holds a username or password")
	ErrQueryInAddress  = errors.New("the address holds a query or fragment")
)

// ParseAddress is raw, with surrounding spaces trimmed, as kind's address.
// An IPv6 literal must be in brackets. A path of "/" or none becomes the
// kind's Path, and a trailing slash is dropped.
func ParseAddress(raw string, kind Kind) (Address, error) {
	raw = strings.TrimSpace(raw)
	lower := strings.ToLower(raw)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return Address{}, ErrAddressScheme
	}
	u, err := url.Parse(raw)
	switch {
	case err != nil:
		return Address{}, ErrAddressNotURL
	case u.User != nil:
		return Address{}, ErrSignInInAddress
	case u.RawQuery != "" || u.ForceQuery || strings.Contains(raw, "#"):
		return Address{}, ErrQueryInAddress
	case u.Hostname() == "":
		return Address{}, ErrAddressHost
	}
	if p := u.Port(); p != "" {
		if n, err := strconv.Atoi(p); err != nil || n < 1 || n > 65535 {
			return Address{}, ErrAddressPort
		}
	}
	u.Host = strings.TrimSuffix(u.Host, ":")
	u.Path, u.RawPath = strings.TrimSuffix(u.Path, "/"), ""
	if u.Path == "" {
		u.Path = kind.Facts().Path
	}
	return Address{u: *u, kind: kind}, nil
}

func (a Address) String() string { return a.u.String() }

// Key is scheme, lowercased host, port with the scheme's default filled
// in, and path: two addresses with one Key reach one client.
func (a Address) Key() string {
	return a.u.Scheme + "://" + strings.ToLower(a.HostPort()) + a.u.Path
}

// HostPort is what the Node dials, for sentences: "192.168.1.20:9091".
func (a Address) HostPort() string {
	port := a.u.Port()
	if port == "" {
		port = map[string]string{"http": "80", "https": "443"}[a.u.Scheme]
	}
	return net.JoinHostPort(a.u.Hostname(), port)
}

// Join is the address with path, such as "/api/v2/auth/login", after its
// own path.
func (a Address) Join(path string) string {
	u := a.u
	u.Path += path
	return u.String()
}

// WebUI is the client's own web UI, false for a kind with none.
func (a Address) WebUI() (string, bool) {
	path := a.kind.Facts().WebUI
	if path == "" {
		return "", false
	}
	u := url.URL{Scheme: a.u.Scheme, Host: a.u.Host, Path: path}
	return u.String(), true
}
