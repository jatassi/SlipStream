package moves

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/moves/internal/db"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/reconciling"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

// scoping is how a wake narrows to the units its event names.
type scoping uint8

const (
	// everyUnit wakes every unit with the wait.
	everyUnit scoping = iota
	byWork
	byNode
	// byServerRoots wakes the units whose source root a named server
	// shows.
	byServerRoots
	// inTransfer wakes every unit that copies now, whatever its wait, so
	// its view follows its Transfer's.
	inTransfer
	// byTorrent wakes the torrent units whose source or destination
	// Torrent the event names.
	byTorrent
)

// waitRule is which events mark a unit with a wait due, and how a wake is
// scoped to the units it concerns. laneFreed is a wait the head walk
// ends: a unit waits for its lane, which the pass reads itself.
type waitRule struct {
	on        []live.Event
	scope     scoping
	laneFreed bool
	// every is whether the wait's Hold sets its own Until, so a fact no
	// event names is read again.
	every bool
}

// waits is the registry: every stored Wait and what ends it.
var waits = map[Wait]waitRule{
	WaitPairBusy:      {laneFreed: true},
	WaitNodeOffline:   {on: []live.Event{nodes.ChangedEvent}, scope: byNode, laneFreed: true},
	WaitLowPriority:   {on: []live.Event{transfers.ChangedEvent}, scope: inTransfer},
	WaitNoRoute:       {on: []live.Event{transfers.ChangedEvent}, scope: inTransfer},
	WaitSpace:         {on: []live.Event{roots.VolumesChangedEvent, transfers.ChangedEvent}},
	WaitRenames:       {on: []live.Event{files.MovedEvent}, scope: byWork},
	WaitDownloading:   {on: []live.Event{downloadclients.ChangedEvent}, scope: byTorrent, every: true},
	WaitImporting:     {on: []live.Event{imports.ChangedEvent}},
	WaitOpenFiles:     {on: []live.Event{transfers.ChangedEvent, imports.ChangedEvent, subtitles.ChangedEvent}},
	WaitPlaying:       {on: []live.Event{mediaservers.SessionsChangedEvent}, scope: byServerRoots, every: true},
	WaitMediaServers:  {on: []live.Event{mediaservers.ChangedEvent}, scope: byServerRoots, every: true},
	WaitGuard:         {on: []live.Event{downloadclients.ChangedEvent}, scope: byTorrent, every: true},
	WaitLinkedTorrent: {every: true},
}

// scope is what fired events named for one wait: every unit when all,
// else the ids.
type scope struct {
	all      bool
	works    []works.WorkID
	nodes    []role.NodeID
	servers  []link.ServerID
	torrents []storage.UUID
}

func (s *scope) add(o scope) {
	s.all = s.all || o.all
	s.works = append(s.works, o.works...)
	s.nodes = append(s.nodes, o.nodes...)
	s.servers = append(s.servers, o.servers...)
	s.torrents = append(s.torrents, o.torrents...)
}

// Wake follows every event the registry names, and reconciling.changed,
// until ctx ends. Between passes it gathers the waits they fire with the
// ids they name, and pokes the runner.
func (m *Moves) Wake(ctx context.Context) {
	var events []live.Event
	for _, r := range waits {
		for _, e := range r.on {
			if !slices.Contains(events, e) {
				events = append(events, e)
			}
		}
	}
	tap := m.cfg.Hub.Tap(ctx, append(events, reconciling.ChangedEvent)...)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-tap.Woken():
			}
			msgs, lost := tap.Take()
			m.gather(msgs, lost)
			m.Poke()
		}
	}()
}

// gather fires each wait msgs or a lost message ends, and notes a
// reconciling change.
func (m *Moves) gather(msgs []live.Message, lost bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for w, r := range waits {
		for _, msg := range msgs {
			if fired, s := r.named(msg); fired {
				m.fire(w, s)
			}
		}
		if lost && len(r.on) > 0 {
			m.fire(w, scope{all: true})
		}
	}
	m.reconciled = m.reconciled || lost || slices.ContainsFunc(msgs, func(msg live.Message) bool {
		return msg.Name == reconciling.ChangedEvent.Name()
	})
}

// fire runs under m.mu.
func (m *Moves) fire(w Wait, s scope) {
	if m.fired[w] == nil {
		m.fired[w] = &scope{}
	}
	m.fired[w].add(s)
}

// named is whether msg fires r, and the units it names.
func (r waitRule) named(msg live.Message) (bool, scope) {
	if !slices.ContainsFunc(r.on, func(e live.Event) bool { return e.Name() == msg.Name }) {
		return false, scope{}
	}
	switch r.scope {
	case byWork:
		if m, ok := msg.Data.(files.Moved); ok {
			return true, scope{works: parsed[works.WorkID](m.WorkIDs)}
		}
	case byNode:
		if n, ok := msg.Data.(nodes.Changed); ok {
			return true, scope{nodes: parsed[role.NodeID]([]string{n.NodeID})}
		}
	case byServerRoots:
		switch c := msg.Data.(type) {
		case mediaservers.Changed:
			return true, scope{servers: parsed[link.ServerID](c.ServerIDs)}
		case mediaservers.SessionsChanged:
			return true, scope{servers: parsed[link.ServerID](c.ServerIDs)}
		}
	case byTorrent:
		if c, ok := msg.Data.(downloadclients.Changed); ok {
			var ids []storage.UUID
			for _, s := range c.IDs {
				if id, err := storage.ParseUUID(s); err == nil {
					ids = append(ids, id)
				}
			}
			return true, scope{torrents: ids}
		}
	case everyUnit, inTransfer:
	}
	return true, scope{all: true}
}

// parsed is each of ss an id of type T, embedding storage.UUID, skipping
// any that doesn't parse.
func parsed[T ~struct{ storage.UUID }](ss []string) []T {
	var out []T
	for _, s := range ss {
		if id, err := storage.ParseUUID(s); err == nil {
			out = append(out, T{UUID: id})
		}
	}
	return out
}

// wakes takes the waits fired since the last pass.
func (m *Moves) wakes() (map[Wait]*scope, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	fired, reconciled := m.fired, m.reconciled
	m.fired, m.reconciled = map[Wait]*scope{}, false
	return fired, reconciled
}

// applyWakes marks due, in tx, every unit a fired wait names: one UPDATE
// per fired wait over moves_unit_waiting, scoped by the ids it named.
func applyWakes(ctx context.Context, tx *storage.Tx[file], fired map[Wait]*scope, reconciled bool, now storage.Time) error {
	q := db.New(tx)
	if reconciled {
		if err := q.WakeReconcile(ctx, now); err != nil {
			return err
		}
	}
	for w, s := range fired {
		r := waits[w]
		waiting := string(w)
		var err error
		switch {
		case r.scope == inTransfer:
			err = q.WakeTransfers(ctx, now)
		case s.all || r.scope == everyUnit:
			err = q.Wake(ctx, db.WakeParams{Now: now, Waiting: &waiting})
		case r.scope == byWork:
			err = q.WakeWorks(ctx, db.WakeWorksParams{Now: now, Waiting: &waiting, Works: s.works})
		case r.scope == byTorrent:
			err = q.WakeTorrents(ctx, db.WakeTorrentsParams{Now: now, Waiting: &waiting, Sources: s.torrents, Dests: s.torrents})
		case r.scope == byNode:
			err = q.WakeNodes(ctx, db.WakeNodesParams{Now: now, Waiting: &waiting, Froms: s.nodes, Tos: s.nodes})
		case r.scope == byServerRoots:
			var showing []roots.ID
			if showing, err = mediaservers.ShowingRoots(ctx, tx, s.servers); err == nil {
				err = q.WakeFromRoots(ctx, db.WakeFromRootsParams{Now: now, Waiting: &waiting, Roots: showing})
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}
