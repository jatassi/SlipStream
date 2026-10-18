package node

import (
	"context"
	"os"
	"time"

	"github.com/jatassi/slipstream/internal/node/internal/host"
	"github.com/jatassi/slipstream/internal/role"
)

// probeChecks is the check each probe the Role table names runs on this
// Node: its verdict, and why it doesn't fit.
var probeChecks = map[role.Probe]func(*Node) (role.Fit, role.ProbeReason){
	role.EngineStarts: (*Node).engineStarts,
	role.DataWritable: (*Node).dataWritable,
	role.ListenerUp:   (*Node).listenerUp,
}

// probe runs every probe the Role table names.
func (n *Node) probe() []role.ProbeResult {
	now := time.Now()
	out := make([]role.ProbeResult, 0, len(probeChecks))
	for _, p := range role.Probes() {
		fit, why := probeChecks[p](n)
		out = append(out, role.ProbeResult{Probe: p, Fit: fit, Reason: why, At: now})
	}
	return out
}

// reprobe reports the probes again every timings.role.probe until ctx
// ends.
func (n *Node) reprobe(ctx context.Context) error {
	tick := time.NewTicker(n.cfg.Timings.Role.Probe)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
			n.reportHost()
		}
	}
}

func (n *Node) engineStarts() (role.Fit, role.ProbeReason) {
	if _, err := n.torrent.ListenAddr(); err != nil {
		return role.Fails, role.EngineNotListening
	}
	dir := n.downloads.Dir()
	if !writable(dir) {
		return role.Fails, role.DownloadsNotWritable
	}
	if remote, err := host.OnNetworkShare(dir); err == nil && remote {
		return role.Weak, role.DownloadsOnNetworkShare
	}
	return role.Fits, ""
}

func (n *Node) dataWritable() (role.Fit, role.ProbeReason) {
	if !writable(n.env.DataDir.Path()) {
		return role.Fails, role.DataNotWritable
	}
	return role.Fits, ""
}

func (n *Node) listenerUp() (role.Fit, role.ProbeReason) {
	if f := n.host.facts.Load(); f == nil || !f.Listeners.HTTP.IsValid() {
		return role.Fails, role.HTTPNotListening
	}
	return role.Fits, ""
}

// writable creates dir when it is missing, then writes a file in it and
// removes it.
func writable(dir string) bool {
	if os.MkdirAll(dir, 0o750) != nil {
		return false
	}
	f, err := os.CreateTemp(dir, ".probe-*")
	if err != nil {
		return false
	}
	_, werr := f.WriteString("slipstream")
	cerr := f.Close()
	rerr := os.Remove(f.Name())
	return werr == nil && cerr == nil && rerr == nil
}
