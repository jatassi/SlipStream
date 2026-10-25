package world

import (
	"context"
	"encoding/json"
	"net/netip"
	"slices"
	"sync"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/internal/front"
	"github.com/jatassi/slipstream/fakes/internal/leecher"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/internal/bencode"
)

// LeecherTarget is the fault target of the leecher front of the Node named
// name: the leecher dials it to reach that Node's listen port.
func LeecherTarget(name string) fault.Target { return fault.Target("leecher/" + name) }

// Hello is a leecher.hello record's data: whether the leecher's Node
// answered its hello for the release, and why not.
type Hello struct {
	Release  string `json:"release"`
	Answered bool   `json:"answered"`
	Error    string `json:"error,omitempty"`
}

// leeching is the Scenario's leecher as the run plays it: one hello, after
// its Node's first announce of its release that fake-tracker accepted.
type leeching struct {
	spec   scenario.Leecher
	hash   bencode.InfoHash
	at     netip.AddrPort
	once   sync.Once
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// armLeecher readies the Scenario's leecher once the release catalog is
// built, and forwards its front to its Node's torrent listener as that
// Node runs now.
func (r *Run) armLeecher() {
	spec := r.scenario.Leecher
	if spec == nil {
		return
	}
	rel := r.releases.Releases[slices.IndexFunc(r.releases.Releases, func(rel releases.Release) bool { return rel.Name == spec.Release })]
	n := r.nodes[slices.IndexFunc(r.nodes, func(n *nodeRun) bool { return n.spec.Name == spec.Node })]
	i := slices.IndexFunc(r.listeners, func(l *front.Front) bool { return l.Fault().Target == LeecherTarget(spec.Node) })
	r.listeners[i].Serve(front.Forward(func() (netip.AddrPort, error) {
		ready, err := n.life.Live()
		return ready.Listeners.Torrent, err
	}))
	l := &leeching{spec: *spec, hash: rel.InfoHash, at: r.listeners[i].Addr()}
	l.ctx, l.cancel = context.WithCancel(r.life)
	r.leecher.Store(l)
}

// leech starts the leecher's hello on rec when it is the first announce
// of the leecher's release from its Node that fake-tracker accepted.
func (r *Run) leech(rec events.Record) {
	l := r.leecher.Load()
	if l == nil || rec.Kind != tracker.Announces.Kind || rec.Node != l.spec.Node {
		return
	}
	var a tracker.Announce
	if json.Unmarshal(rec.Data, &a) != nil || !a.Accepted || a.Event == "stopped" || a.InfoHash != l.hash.String() {
		return
	}
	l.once.Do(func() {
		l.wg.Go(func() {
			h := Hello{Release: l.spec.Release, Answered: true}
			if err := leecher.Hello(l.ctx, l.at, l.hash); err != nil {
				h.Answered, h.Error = false, err.Error()
			}
			r.appendRecord(events.Record{Kind: events.LeecherHello, Node: l.spec.Node, Data: data(h)})
		})
	})
}

// stopLeecher ends a hello in flight and waits for its record.
func (r *Run) stopLeecher() {
	if l := r.leecher.Load(); l != nil {
		l.cancel()
		l.wg.Wait()
	}
}
