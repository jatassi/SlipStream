package netfacts

import (
	"context"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/netfacts/gateway"
	"github.com/jatassi/slipstream/internal/netmodel"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

type Config struct {
	Env       Env
	Endpoints Endpoints
	Timings   Timings
	Rules     netmodel.Rules
	// HTTP reaches the router's UPnP description and control.
	HTTP *http.Client
	// Roots verify a tested address's certificate; nil is the system's.
	Roots *x509.CertPool
	// Local holds the router mappings this Node asked for.
	Local *storage.DB[storage.Local]
	// Node names this Node's UPnP entries.
	Node role.NodeID
	Log  *slog.Logger
}

// Netfacts is one Node's network facts: the passes that learn them, the
// watch that publishes a summary once a change held, and the service the
// Controller reads them through.
type Netfacts struct {
	cfg     Config
	gateway *gateway.Client
	mapper  *mapper
	// passing holds one pass at a time, so Reads that arrive together
	// share it.
	passing chan struct{}

	mu       sync.Mutex
	watching bool
	stun     []string
	maps     []Want
	last     pass
}

// pass is what one full pass observed.
type pass struct {
	at         time.Time
	interfaces []netmodel.Iface
	egress     netmodel.Egress
	gateway    netmodel.Gateway
	stun       []netmodel.STUNAnswer
	// router is the router the pass read, nil when it read none.
	router *gateway.Target
}

func New(cfg Config) *Netfacts {
	client := &gateway.Client{Packets: cfg.Env.Packets, HTTP: cfg.HTTP, Timings: gateway.Timings{Exchange: cfg.Timings.Gateway, SSDPWait: cfg.Timings.SSDPWait}}
	return &Netfacts{
		cfg:     cfg,
		gateway: client,
		mapper: &mapper{
			local: cfg.Local, client: client, description: "SlipStream " + cfg.Node.String(), timings: cfg.Timings,
			at: func(ctx context.Context, at netip.AddrPort) (gateway.Target, error) {
				return client.Toward(ctx, at, len(cfg.Endpoints.Gateway) > 0)
			},
		},
		passing: make(chan struct{}, 1),
	}
}

// Facts reads this Node's facts: interfaces, STUN and the router from the
// last full pass when it is younger than maxAge, else from a fresh one,
// and listeners and firewall fresh for ports. Local, or while the watch's
// intent says not to watch, nothing leaves the machine: interfaces are
// read fresh, and a local Read while watching carries the last pass's
// egress, router and STUN answers.
func (n *Netfacts) Facts(ctx context.Context, ports []uint16, maxAge time.Duration, local bool) (netmodel.NetworkFacts, error) {
	n.mu.Lock()
	watching, last := n.watching, n.last
	n.mu.Unlock()
	var f netmodel.NetworkFacts
	if local || !watching {
		ifaces, err := n.interfaces(ctx)
		if err != nil {
			return f, err
		}
		f = netmodel.NetworkFacts{At: time.Now(), Interfaces: ifaces}
		if watching {
			f.Egress, f.Gateway, f.STUN = last.egress, last.gateway, last.stun
		}
	} else {
		p, err := n.recent(ctx, maxAge)
		if err != nil {
			return f, err
		}
		f = p.facts()
	}
	listeners, rules, err := n.listening(ctx, ports, f.Egress.Iface)
	if err != nil {
		return f, err
	}
	f.Listeners, f.Firewall = listeners, rules
	f.Mappings, err = n.mapper.mappings(ctx)
	return f, err
}

func (p pass) facts() netmodel.NetworkFacts {
	return netmodel.NetworkFacts{At: p.at, Watched: true, Interfaces: p.interfaces, Egress: p.egress, Gateway: p.gateway, STUN: p.stun}
}

// recent is the last pass when it is younger than maxAge, else a fresh
// one. A Read waiting on another's pass takes that pass.
func (n *Netfacts) recent(ctx context.Context, maxAge time.Duration) (pass, error) {
	asked := time.Now()
	select {
	case n.passing <- struct{}{}:
	case <-ctx.Done():
		return pass{}, fmt.Errorf("wait for the network pass: %w", ctx.Err())
	}
	defer func() { <-n.passing }()
	n.mu.Lock()
	last := n.last
	n.mu.Unlock()
	if !last.at.IsZero() && (maxAge > 0 && time.Since(last.at) < maxAge || !last.at.Before(asked)) {
		return last, nil
	}
	return n.pass(ctx)
}

// pass observes the interfaces, asks each STUN server, and reads the
// router the STUN socket's route leaves by.
func (n *Netfacts) pass(ctx context.Context) (pass, error) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, n.cfg.Timings.Pass)
	defer cancel()
	n.mu.Lock()
	servers := slices.Clone(n.stun)
	n.mu.Unlock()
	ifaces, err := n.interfaces(ctx)
	if err != nil {
		return pass{}, err
	}
	p := pass{at: start, interfaces: ifaces}
	var local netip.Addr
	p.stun, local = n.askSTUN(ctx, servers)
	if local.IsValid() {
		route, err := n.cfg.Env.Route(ctx, local)
		if err != nil {
			return pass{}, fmt.Errorf("read the route: %w", err)
		}
		p.egress = netmodel.Egress{Iface: route.Iface, Local: local}
		if gw, ok := n.target(route, local); ok {
			if p.gateway, err = n.gateway.Read(ctx, gw, nil); err != nil {
				return pass{}, err
			}
			p.gateway.Addr = route.Gateway
			p.router = &gw
		}
	}
	n.mu.Lock()
	n.last = p
	maps := n.maps
	n.mu.Unlock()
	if err := n.mapper.reconcile(ctx, p.router, p.gateway.Protocols, maps, p.gateway.Forwards); err != nil {
		n.cfg.Log.Warn("couldn't keep this Node's router mappings", "err", err.Error())
	}
	return p, nil
}

// target is the router to ask: the gateway entry, which names it by
// address, else the route's gateway, none when the route has none.
func (n *Netfacts) target(route Route, local netip.Addr) (gateway.Target, bool) {
	if len(n.cfg.Endpoints.Gateway) > 0 {
		e := n.cfg.Endpoints.Gateway[0]
		at, err := netip.ParseAddrPort(e.String())
		return gateway.At(at, local), err == nil
	}
	return gateway.Discovered(route.Gateway, local), route.Gateway.IsValid()
}

func (n *Netfacts) interfaces(ctx context.Context) ([]netmodel.Iface, error) {
	ifaces, err := n.cfg.Env.Interfaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("read the interfaces: %w", err)
	}
	for i := range ifaces {
		for j := range ifaces[i].Addrs {
			ifaces[i].Addrs[j].Class = n.cfg.Rules.Of(ifaces[i].Addrs[j].Prefix.Addr())
		}
	}
	return ifaces, nil
}

// listening reads what listens on ports and what the firewall says of
// each for connections arriving on egress, with SlipStream's own
// listeners as the Node knows them.
func (n *Netfacts) listening(ctx context.Context, ports []uint16, egress string) ([]netmodel.Listener, []netmodel.Rule, error) {
	if len(ports) == 0 {
		return nil, nil, nil
	}
	listeners, err := n.cfg.Env.Sockets(ctx, ports)
	if err != nil {
		return nil, nil, fmt.Errorf("read the listeners: %w", err)
	}
	for _, own := range n.cfg.Env.Self() {
		if !slices.Contains(ports, own.Port) {
			continue
		}
		listeners = slices.DeleteFunc(listeners, func(l netmodel.Listener) bool { return l.Port == own.Port })
		listeners = append(listeners, own)
	}
	qs := make([]Query, 0, 2*len(ports))
	for _, p := range ports {
		qs = append(qs, Query{Port: p, Family: netmodel.IPv4}, Query{Port: p, Family: netmodel.IPv6})
	}
	rules, err := n.cfg.Env.Firewall(ctx, egress, qs)
	if err != nil {
		return nil, nil, fmt.Errorf("read the firewall: %w", err)
	}
	return listeners, rules, nil
}
