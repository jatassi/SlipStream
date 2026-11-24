package netfacts

import (
	"context"
	"slices"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/jatassi/slipstream/internal/netmodel"
	netv1 "github.com/jatassi/slipstream/internal/pb/slipstream/network/v1"
	"github.com/jatassi/slipstream/internal/role"
)

// Watch is the network channel's run. While the intent says to watch it
// passes every interval, and publishes a summary only once a changed one
// held: a confirm pass at the change plus the intent's wait observes it
// again. Told not to watch, it never passes and publishes that once.
func (n *Netfacts) Watch(ctx context.Context, h role.Holder[*netv1.Intent, *netv1.Facts, *emptypb.Empty]) error {
	intents := make(chan *netv1.Intent)
	go func() {
		for s := range h.Intent(ctx) {
			select {
			case intents <- s.Intent:
			case <-ctx.Done():
				return
			}
		}
	}()
	w := &watcher{n: n, h: h}
	defer w.stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case next := <-intents:
			w.take(ctx, next)
		case <-w.tick:
			w.observe(ctx)
		case <-w.confirm:
			w.confirm = nil
			w.observe(ctx)
		}
	}
}

type watcher struct {
	n         *Netfacts
	h         role.Holder[*netv1.Intent, *netv1.Facts, *emptypb.Empty]
	intent    *netv1.Intent
	published *netmodel.Summary
	candidate *netmodel.Summary
	ticker    *time.Ticker
	tick      <-chan time.Time
	confirm   <-chan time.Time
}

func (w *watcher) stop() {
	if w.ticker != nil {
		w.ticker.Stop()
	}
	w.tick, w.confirm, w.candidate = nil, nil, nil
}

// take follows next, and starts the passes over when its watch or
// interval changed.
func (w *watcher) take(ctx context.Context, next *netv1.Intent) {
	changed := w.intent == nil || next.GetWatch() != w.intent.GetWatch() || next.GetIntervalMs() != w.intent.GetIntervalMs()
	remap := w.intent != nil && !slices.EqualFunc(next.GetMap(), w.intent.GetMap(), func(a, b *netv1.MapWant) bool { return proto.Equal(a, b) })
	w.intent = next
	w.n.follow(next)
	if !changed {
		if remap && next.GetWatch() {
			w.observe(ctx)
		}
		return
	}
	w.stop()
	if !next.GetWatch() {
		if err := w.n.Release(ctx); err != nil {
			w.n.cfg.Log.Warn("couldn't delete this Node's router mappings", "err", err.Error())
		}
		if w.published == nil || w.published.Watched {
			w.h.SetFacts(netmodel.SummaryOnWire(netmodel.Summary{}, time.Now()))
			w.published = &netmodel.Summary{}
		}
		return
	}
	w.ticker = time.NewTicker(time.Duration(next.GetIntervalMs()) * time.Millisecond)
	w.tick = w.ticker.C
	w.observe(ctx)
}

func (w *watcher) observe(ctx context.Context) {
	p, err := w.n.recent(ctx, 0)
	if err != nil {
		w.n.cfg.Log.Warn("couldn't read this Node's network", "err", err.Error())
		return
	}
	s := w.n.cfg.Rules.Summary(p.facts())
	switch {
	case w.published != nil && s.Revision == w.published.Revision:
		w.candidate, w.confirm = nil, nil
	case w.candidate != nil && s.Revision == w.candidate.Revision && w.confirm == nil:
		w.h.SetFacts(netmodel.SummaryOnWire(s, p.at))
		w.published, w.candidate = &s, nil
	case w.candidate == nil || s.Revision != w.candidate.Revision:
		w.candidate = &s
		w.confirm = time.After(max(time.Duration(w.intent.GetWaitMs())*time.Millisecond, w.n.cfg.Timings.ConfirmFloor))
	}
}

// follow takes the intent's watch and STUN list for every pass and Read
// after it.
func (n *Netfacts) follow(i *netv1.Intent) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.watching = i.GetWatch()
	n.maps = nil
	for _, m := range i.GetMap() {
		n.maps = append(n.maps, Want{External: uint16(m.GetExternal()), Internal: uint16(m.GetInternal())}) //nolint:gosec // G115: the Controller sends ports
	}
	n.stun = i.GetStun().GetServers()
	if i.GetStun() == nil {
		n.stun = nil
		for _, s := range n.cfg.Endpoints.STUN {
			n.stun = append(n.stun, s.String())
		}
	}
}

// Release is the channel's release at removal, and the watch's when told
// not to watch: it deletes every router mapping this Node holds, the
// entries of the last router it read among them.
func (n *Netfacts) Release(ctx context.Context) error {
	n.mu.Lock()
	router := n.last.router
	n.mu.Unlock()
	var listed []netmodel.Forward
	if router != nil {
		g, err := n.gateway.Read(ctx, *router, nil)
		if err != nil {
			return err
		}
		listed = g.Forwards
	}
	return n.mapper.reconcile(ctx, router, netmodel.Protocols{}, nil, listed)
}
