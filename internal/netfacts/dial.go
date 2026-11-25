package netfacts

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/proxy"

	"github.com/jatassi/slipstream/internal/netmodel"
	netv1 "github.com/jatassi/slipstream/internal/pb/slipstream/network/v1"
)

// dial connects from this Node as a player would: to each address of the
// asked family that the host resolves to here, or to the vetted address,
// through the outside dialer entry when it is set. It finishes TLS with
// the public host as SNI, records the chain whatever its validity, sends
// one GET of the path without following a redirect, and keeps at most
// timings.netfacts.read_bytes of the body.
func (n *Netfacts) dial(ctx context.Context, req *netv1.DialRequest) (*netv1.DialResponse, error) {
	fam, err := netmodel.ParseFamily(req.GetFamily())
	if err != nil {
		return nil, err
	}
	port := uint16(req.GetPort()) //nolint:gosec // G115: checked to fit just below
	if port == 0 || uint32(port) != req.GetPort() {
		return nil, fmt.Errorf("port %d", req.GetPort())
	}
	if req.GetScheme() != netv1.Scheme_SCHEME_HTTP && req.GetScheme() != netv1.Scheme_SCHEME_HTTPS {
		return nil, fmt.Errorf("scheme %v", req.GetScheme())
	}
	sni := req.GetSni()
	var addrs []netip.Addr
	switch t := req.GetTarget().(type) {
	case *netv1.DialRequest_Host:
		sni = t.Host
		r := n.resolve(ctx, t.Host)
		raw, e := r.GetA(), r.GetErrorA()
		if fam == netmodel.IPv6 {
			raw, e = r.GetAaaa(), r.GetErrorAaaa()
		}
		if e != netv1.DNSError_DNS_ERROR_UNSPECIFIED {
			return &netv1.DialResponse{Error: e}, nil
		}
		for _, b := range raw {
			a, _ := netip.AddrFromSlice(b)
			addrs = append(addrs, a)
		}
	case *netv1.DialRequest_Addr:
		a, err := netmodel.ParseAddr(t.Addr)
		if err != nil || !a.IsValid() {
			return nil, fmt.Errorf("address: %w", err)
		}
		addrs = append(addrs, a)
	default:
		return nil, errors.New("no target")
	}
	out := &netv1.DialResponse{}
	for _, a := range addrs {
		out.Attempts = append(out.Attempts, n.attempt(ctx, netip.AddrPortFrom(a, port), fam, req.GetScheme(), sni, req.GetPath()))
	}
	return out, nil
}

func (n *Netfacts) attempt(ctx context.Context, at netip.AddrPort, fam netmodel.Family, scheme netv1.Scheme, sni, path string) *netv1.Attempt {
	a := &netv1.Attempt{Addr: at.Addr().AsSlice(), Stage: netv1.Stage_STAGE_RESOLVED}
	network := map[netmodel.Family]string{netmodel.IPv4: "tcp4", netmodel.IPv6: "tcp6"}[fam]
	cctx, cancel := context.WithTimeout(ctx, n.cfg.Timings.Connect)
	conn, err := n.dialer().DialContext(cctx, network, at.String())
	cancel()
	if err != nil {
		a.Error = connectError(err)
		return a
	}
	defer func() { _ = conn.Close() }()
	a.Stage = netv1.Stage_STAGE_CONNECTED
	if scheme == netv1.Scheme_SCHEME_HTTPS {
		tc := tls.Client(conn, &tls.Config{ServerName: sni, InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}) //nolint:gosec // G402: the chain is verified below, so a bad one is recorded rather than refused
		hctx, cancel := context.WithTimeout(ctx, n.cfg.Timings.TLS)
		err := tc.HandshakeContext(hctx)
		cancel()
		if err != nil {
			a.Error, a.Tls = netv1.DialError_DIAL_ERROR_TLS, &netv1.Certificate{Problem: netv1.TLSProblem_TLS_PROBLEM_HANDSHAKE}
			return a
		}
		a.Tls = n.certificate(tc.ConnectionState().PeerCertificates, sni)
		a.Stage, conn = netv1.Stage_STAGE_TLS, tc
	}
	n.fetch(ctx, conn, a, at, scheme, sni, path)
	return a
}

func (n *Netfacts) dialer() proxy.ContextDialer {
	direct := &net.Dialer{}
	if len(n.cfg.Endpoints.OutsideDialer) == 0 {
		return direct
	}
	socks, err := proxy.SOCKS5("tcp", n.cfg.Endpoints.OutsideDialer[0].String(), nil, direct)
	if err != nil {
		panic(fmt.Sprintf("netfacts: SOCKS5 to %s: %v", n.cfg.Endpoints.OutsideDialer[0], err))
	}
	cd, _ := socks.(proxy.ContextDialer)
	return cd
}

// fetch sends one GET of path and records its answer in a.
func (n *Netfacts) fetch(ctx context.Context, conn net.Conn, a *netv1.Attempt, at netip.AddrPort, scheme netv1.Scheme, host, path string) {
	if host == "" {
		host = at.Addr().String()
	}
	u := &url.URL{Scheme: map[netv1.Scheme]string{netv1.Scheme_SCHEME_HTTP: "http", netv1.Scheme_SCHEME_HTTPS: "https"}[scheme], Host: hostPort(host, at.Port(), scheme), Path: path}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		a.Error = netv1.DialError_DIAL_ERROR_HTTP
		return
	}
	_ = conn.SetDeadline(time.Now().Add(n.cfg.Timings.Read))
	r := bufio.NewReader(conn)
	if err := req.Write(conn); err != nil {
		a.Error = netv1.DialError_DIAL_ERROR_HTTP
		return
	}
	res, err := http.ReadResponse(r, req)
	if err != nil {
		a.Error = netv1.DialError_DIAL_ERROR_HTTP
		return
	}
	defer func() { _ = res.Body.Close() }()
	a.Status = int32(res.StatusCode) //nolint:gosec // G115: an HTTP status is three digits
	if loc, err := res.Location(); err == nil {
		a.LocationHost = loc.Hostname()
	}
	a.Body, err = io.ReadAll(io.LimitReader(res.Body, n.cfg.Timings.ReadBytes))
	if err != nil {
		a.Error = netv1.DialError_DIAL_ERROR_HTTP
		return
	}
	a.Stage = netv1.Stage_STAGE_HTTP
}

// hostPort is the Host a player sends: the host alone on the scheme's
// own port.
func hostPort(host string, port uint16, scheme netv1.Scheme) string {
	if (scheme == netv1.Scheme_SCHEME_HTTPS && port == 443) || (scheme == netv1.Scheme_SCHEME_HTTP && port == 80) {
		if strings.Contains(host, ":") {
			return "[" + host + "]"
		}
		return host
	}
	return net.JoinHostPort(host, strconv.Itoa(int(port)))
}

// certificate is the chain's leaf as a player would judge it: expired,
// not from a trusted authority, or for another name.
func (n *Netfacts) certificate(chain []*x509.Certificate, name string) *netv1.Certificate {
	leaf := chain[0]
	c := &netv1.Certificate{Names: leaf.DNSNames, NotAfterMs: leaf.NotAfter.UnixMilli()}
	inter := x509.NewCertPool()
	for _, cert := range chain[1:] {
		inter.AddCert(cert)
	}
	_, err := leaf.Verify(x509.VerifyOptions{Roots: n.cfg.Roots, Intermediates: inter})
	var invalid x509.CertificateInvalidError
	switch {
	case errors.As(err, &invalid) && invalid.Reason == x509.Expired:
		c.Problem = netv1.TLSProblem_TLS_PROBLEM_EXPIRED
	case err != nil:
		c.Problem = netv1.TLSProblem_TLS_PROBLEM_UNTRUSTED
	case leaf.VerifyHostname(name) != nil:
		c.Problem = netv1.TLSProblem_TLS_PROBLEM_WRONG_NAME
	}
	return c
}

// connectError is why a connection failed. The SOCKS5 client reports its
// proxy's reply only as text, so a refusal behind the outside dialer is
// matched by its words.
func connectError(err error) netv1.DialError {
	var timeout interface{ Timeout() bool }
	switch {
	case errors.Is(err, refused) || strings.HasSuffix(err.Error(), "connection refused"):
		return netv1.DialError_DIAL_ERROR_REFUSED
	case errors.Is(err, context.DeadlineExceeded) || errors.As(err, &timeout) && timeout.Timeout():
		return netv1.DialError_DIAL_ERROR_TIMEOUT
	}
	return netv1.DialError_DIAL_ERROR_UNREACHABLE
}
