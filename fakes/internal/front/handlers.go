package front

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/netip"
	"sync"

	"github.com/jatassi/slipstream/fakes/fault"
)

// Inbox is a fake's fan-in listener: the connections each of its served
// fronts accepted, in the order they arrive.
type Inbox struct {
	addr  net.Addr
	conns chan net.Conn
	done  chan struct{}
	once  sync.Once
}

// NewInbox is an Inbox whose Addr is addr, the fake's world front.
func NewInbox(addr netip.AddrPort) *Inbox {
	return &Inbox{addr: net.TCPAddrFromAddrPort(addr), conns: make(chan net.Conn), done: make(chan struct{})}
}

func (in *Inbox) Accept() (net.Conn, error) {
	select {
	case c := <-in.conns:
		return c, nil
	case <-in.done:
		return nil, net.ErrClosed
	}
}

func (in *Inbox) Close() error {
	in.once.Do(func() { close(in.done) })
	return nil
}

func (in *Inbox) Addr() net.Addr { return in.addr }

// Served pushes each connection into in, behind TLS from cfg when cfg is
// not nil.
func Served(in *Inbox, cfg *tls.Config) Handler {
	return func(ctx context.Context, c net.Conn) {
		if cfg != nil {
			c = tls.Server(c, cfg)
		}
		select {
		case in.conns <- c:
		case <-in.done:
			_ = c.Close()
		case <-ctx.Done():
			_ = c.Close()
		}
	}
}

// TLS is a served front's TLS: good, or the bad leaf a TLS rule on f
// picks for one handshake.
func TLS(plan *fault.Plan, f fault.Front, good tls.Certificate, bad map[fault.TLSFailure]tls.Certificate) *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			failure, ok := plan.TLS(f)
			if !ok {
				return &good, nil
			}
			leaf := bad[fault.TLSFailure(failure)]
			return &leaf, nil
		},
	}
}

// Forward dials upstream for each connection and copies bytes both ways
// until either side closes, then closes both.
func Forward(upstream func() (netip.AddrPort, error)) Handler {
	return func(ctx context.Context, c net.Conn) {
		addr, err := upstream()
		var up net.Conn
		if err == nil {
			up, err = new(net.Dialer).DialContext(ctx, "tcp", addr.String())
		}
		if err != nil {
			if a, ok := c.(*app); ok {
				a.t.end(fmt.Sprintf("upstream: %v", err))
			}
			_ = c.Close()
			return
		}
		var wg sync.WaitGroup
		closeBoth := func() {
			_ = up.Close()
			_ = c.Close()
		}
		wg.Go(func() {
			_, _ = io.Copy(up, c)
			closeBoth()
		})
		wg.Go(func() {
			_, _ = io.Copy(c, up)
			closeBoth()
		})
		wg.Wait()
	}
}
