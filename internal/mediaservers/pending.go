package mediaservers

import (
	"context"
	"slices"
	"sync"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/role"
)

// pendingKey is whose Write changed a server: a Node's receive Write, a
// server's sender, or writeSeen. Each takes only its own.
type pendingKey struct {
	node   role.NodeID
	sender link.ServerID
	seen   bool
}

func byNode(node role.NodeID) pendingKey   { return pendingKey{node: node} }
func bySender(id link.ServerID) pendingKey { return pendingKey{sender: id} }
func bySeen() pendingKey                   { return pendingKey{seen: true} }

// pendingServers is the servers each Write changed, until it commits.
type pendingServers struct {
	mu sync.Mutex
	by map[pendingKey][]string
}

func newPendingServers() pendingServers {
	return pendingServers{by: map[pendingKey][]string{}}
}

func (p *pendingServers) add(key pendingKey, id link.ServerID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.by[key] = append(p.by[key], id.String())
}

func (p *pendingServers) take(key pendingKey) []string {
	p.mu.Lock()
	ids := p.by[key]
	delete(p.by, key)
	p.mu.Unlock()
	slices.Sort(ids)
	return slices.Compact(ids)
}

// playsCommitted publishes media-servers.plays-changed for the servers
// key's Write changed, once that Write committed, and forgets them when
// it failed.
func (m *MediaServers) playsCommitted(ctx context.Context, key pendingKey, err error) {
	if ids := m.played.take(key); err == nil && len(ids) > 0 {
		m.cfg.Events.PlaysChanged.Publish(ctx, PlaysChanged{ServerIDs: ids})
	}
}

// playsEnded publishes media-servers.plays-changed for the servers whose
// plays a Controller-side end ended.
func (m *MediaServers) playsEnded(ctx context.Context, ended []link.ServerID) {
	if len(ended) == 0 {
		return
	}
	ids := make([]string, len(ended))
	for i, id := range ended {
		ids[i] = id.String()
	}
	slices.Sort(ids)
	m.cfg.Events.PlaysChanged.Publish(ctx, PlaysChanged{ServerIDs: slices.Compact(ids)})
}
