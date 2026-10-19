package node

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jatassi/slipstream/internal/node/internal/logs"
	"github.com/jatassi/slipstream/internal/nodesystem"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/system"
)

// archiveStamp is an archive's time in its file name, in the Node's zone.
const archiveStamp = "20060102-150405"

// nodeHost is the Node's own system, which NodeSystemService serves on both
// halves. It exists before the halves start, and its facts appear when
// HTTP binds.
type nodeHost struct {
	n     *Node
	facts atomic.Pointer[system.Facts]
}

var errStarting = fmt.Errorf("%w: the Node is starting", nodesystem.ErrOffline)

func (h *nodeHost) Facts() (system.Facts, error) {
	f := h.facts.Load()
	if f == nil {
		return system.Facts{}, errStarting
	}
	return *f, nil
}

func (h *nodeHost) Config() (nodesystem.Config, error) { return h.n.configView() }

func (h *nodeHost) SetConfig(changes []nodesystem.Change) (nodesystem.Config, error) {
	if _, err := h.n.setConfig(changes); err != nil {
		return nodesystem.Config{}, err
	}
	return h.n.configView()
}

// Restart reports the port the Node serves on and, when the saved
// http.listen names another one, the port it moves to. After a fallback
// that is the configured port, which the successor tries first.
func (h *nodeHost) Restart() (nodesystem.Restarting, error) {
	f, err := h.Facts()
	if err != nil {
		return nodesystem.Restarting{}, err
	}
	saved, _, err := h.n.env.configFile().load()
	if err != nil {
		return nodesystem.Restarting{}, err
	}
	r := nodesystem.Restarting{StartedAt: h.n.started, BoundPort: f.Listeners.HTTP.Port()}
	if next := saved.HTTP.Listen.Port(); next != 0 && next != r.BoundPort {
		r.NextPort = next
	}
	switch err := h.n.Restart(); {
	case errors.Is(err, ErrAlreadyRestarting):
		return nodesystem.Restarting{}, nodesystem.ErrAlreadyRestarting
	case errors.Is(err, ErrStopping):
		return nodesystem.Restarting{}, fmt.Errorf("%w: %w", nodesystem.ErrOffline, err)
	case err != nil:
		return nodesystem.Restarting{}, err
	}
	return r, nil
}

func (h *nodeHost) ReadLogs(ctx context.Context, after uint64, wait time.Duration) (system.LogBatch, error) {
	b, err := h.n.logs.Read(ctx, after, wait)
	if errors.Is(err, logs.ErrClosed) {
		return system.LogBatch{}, fmt.Errorf("%w: %w", nodesystem.ErrOffline, err)
	}
	return b, err
}

func (h *nodeHost) Archive() (nodesystem.Archive, error) {
	stamp := time.Now().Format(archiveStamp)
	f, size, err := h.n.logs.Archive(h.n.env.DataDir.RunDir())
	if err != nil {
		return nodesystem.Archive{}, err
	}
	return nodesystem.Archive{File: f, Size: size, Stamp: stamp}, nil
}

// sections names the Nodes section each config.yaml section's rows show in.
var sections = map[string]nodesystem.SectionID{"http": nodesystem.Web, "log": nodesystem.Log}

// configView is the Node settings table as the Nodes sections show it:
// config.yaml as saved now, beside the running values.
func (n *Node) configView() (nodesystem.Config, error) {
	saved, keys, err := n.env.configFile().load()
	if err != nil {
		return nodesystem.Config{}, err
	}
	n.cfgMu.Lock()
	running := n.cfg
	n.cfgMu.Unlock()
	defaults := Defaults(n.env.Types, n.env.Install)
	v := view{saved: &saved, running: &running, defaults: &defaults, keys: keysByName(keys)}
	if f := n.host.facts.Load(); f != nil {
		v.boundHTTP = f.Listeners.HTTP.Port()
	}
	return viewConfig(v)
}

func viewConfig(v view) (nodesystem.Config, error) {
	var c nodesystem.Config
	at := map[nodesystem.SectionID]int{}
	for i, f := range v.fields() {
		prefix, _, _ := strings.Cut(nodeSettings[i].key(), ".")
		id, ok := sections[prefix]
		if !ok {
			panic("node: setting " + nodeSettings[i].key() + " is in no Nodes section")
		}
		b, err := json.Marshal(f)
		if err != nil {
			return nodesystem.Config{}, fmt.Errorf("encode setting %s: %w", nodeSettings[i].key(), err)
		}
		if _, seen := at[id]; !seen {
			at[id] = len(c.Sections)
			c.Sections = append(c.Sections, nodesystem.Section{ID: id})
		}
		s := &c.Sections[at[id]]
		s.Fields = append(s.Fields, b)
	}
	return c, nil
}

// Reach dials t on the Node's network, and finishes a TLS handshake
// there when t asks for TLS.
func (h *nodeHost) Reach(ctx context.Context, t nodesystem.Target) nodesystem.Reached {
	start := time.Now()
	conn, err := h.n.env.Net.DialContext(ctx, "tcp", t.String())
	if err == nil && t.TLS {
		secured := tls.Client(conn, &tls.Config{ServerName: t.Host, MinVersion: tls.VersionTLS12})
		err = secured.HandshakeContext(ctx)
		conn = secured
	}
	took := time.Since(start)
	if conn != nil {
		_ = conn.Close()
	}
	if err != nil {
		return nodesystem.Reached{Took: took, Err: reachError(err)}
	}
	return nodesystem.Reached{OK: true, Took: took}
}

// reachError is why a dial failed, without the addresses the test names.
func reachError(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "timed out"
	}
	if op, ok := errors.AsType[*net.OpError](err); ok {
		return op.Err.Error()
	}
	return err.Error()
}

// Probe runs every probe now and reports the verdicts, as each
// timings.role.probe does.
func (h *nodeHost) Probe() ([]role.ProbeResult, error) { return h.n.report() }

// reportHost reports the Node's host, if a half has started.
func (n *Node) reportHost() {
	if _, err := n.report(); err != nil && !errors.Is(err, errStarting) {
		n.log.Warn("settings revision not reported", "err", err.Error())
	}
}

// report tells the Controller, through the holder half's node facts,
// where the Node serves HTTP, which settings revision it shows and what
// its probes find now, and answers those verdicts.
// hostMu keeps a stale report from landing after a fresh one.
func (n *Node) report() ([]role.ProbeResult, error) {
	n.hostMu.Lock()
	defer n.hostMu.Unlock()
	h := n.half.Load()
	if h == nil {
		return nil, errStarting
	}
	var addr netip.AddrPort
	if f := n.host.facts.Load(); f != nil {
		addr = f.Listeners.HTTP
	}
	c, err := n.configView()
	if err != nil {
		return nil, err
	}
	probes := n.probe()
	h.holding.SetHost(role.Host{HTTPAddress: addr, SettingsRevision: c.Revision(), Probes: probes})
	return probes, nil
}
