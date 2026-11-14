package mediaservers

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/nodes"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1/mediaserverv1connect"
	"github.com/jatassi/slipstream/internal/providers/plextv"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
)

// MediaServers is the Controller's side of S35 (Media Servers): the server
// records and their Path mappings, each Node's intent of the Media Server
// link Role, the reports it applies and the reaching Nodes it follows on
// the roster.
type MediaServers struct {
	cfg     Config
	pins    pins
	seen    seenSessions
	senders *senders
}

type Config struct {
	Library *storage.DB[storage.Library]
	Sealer  *secret.Sealer
	// Roster is read once it opens, after MediaServers is made.
	Roster Roster
	Kinds  link.Registry
	// HTTP is the Controller's own client, for its route tests at the
	// fallback address.
	HTTP    *http.Client
	Timings link.Timings
	// PlexTV is where the Controller signs in to Plex.
	PlexTV plextv.Endpoints
	// Link is MediaServerLinkService on node, the Controller's own through
	// Local.
	Link func(node role.NodeID) mediaserverv1connect.MediaServerLinkServiceHandler
	// RootWords names each root the way copy does.
	RootWords func(ctx context.Context, r storage.Reader[storage.Library]) (map[roots.ID]string, error)
	// RootsChanged wakes after each roots.changed until ctx ends.
	RootsChanged func(ctx context.Context) <-chan struct{}
	// Tasks stores the renewal task's cadence. It is read once it opens.
	Tasks  Cadences
	Events Events
	// Touch follows every commit that changes a Node's intent.
	Touch func(nodes ...role.NodeID)
	// Facts are a Node's last facts of Media Server link, false while its
	// session is detached.
	Facts func(node role.NodeID) (*msv1.Facts, bool)
	// Zone is the Controller's, for the times copy names.
	Zone *time.Location
	Log  *slog.Logger
}

// Publisher is one of the Media Servers' live events, bound to the hub.
type Publisher[T any] interface {
	Publish(ctx context.Context, v T)
}

// Events are the five live events, bound once at the composition root.
type Events struct {
	Changed         Publisher[Changed]
	SessionsChanged Publisher[SessionsChanged]
	Unhealthy       Publisher[WentUnhealthy]
	Recovered       Publisher[Recovered]
	ItemsFound      Publisher[Found]
}

// Roster is what MediaServers reads of the roster.
type Roster interface {
	Self() role.NodeID
	Changed() <-chan struct{}
	InStep(node role.NodeID) bool
	// Standing is how node stands and since when: while away, since its
	// last Sync frame.
	Standing(node role.NodeID) (nodes.Standing, time.Time)
	Holds(name role.Name, node role.NodeID) bool
	Holding(name role.Name) []role.NodeID
}

func New(cfg Config) *MediaServers {
	return &MediaServers{cfg: cfg, seen: newSeenSessions(), senders: newSenders()}
}

// Owner keeps Media Server link on every Node that a server names as its
// reaching Node, until the Admin names another or removes the Node with a
// write-off.
func (m *MediaServers) Owner() nodes.Owner {
	return nodes.Owner{Role: role.MediaServerLink, Held: nodes.HoldsNothing, Keeps: serversOn}
}

// serversOn names the servers node reaches.
func serversOn(ctx context.Context, r storage.Reader[storage.Library], node role.NodeID) ([]string, error) {
	return db.New(r).ListReachedNames(ctx, node)
}

// Intent is the servers that name node as their reaching Node, in id
// order, with no secret.
func (m *MediaServers) Intent(ctx context.Context, r storage.Reader[storage.Library], node role.NodeID) (*msv1.Intent, error) {
	rows, err := db.New(r).ListReached(ctx, node)
	if err != nil {
		return nil, err
	}
	every, err := sessionsEvery(ctx, db.New(r))
	if err != nil {
		return nil, err
	}
	in := &msv1.Intent{SessionsEveryMs: every.Milliseconds()}
	for _, row := range rows {
		s := &msv1.Server{
			Id: row.ID.Bytes(), Kind: string(row.Kind), Name: row.Name, Address: row.Address,
			Identity: row.Identity, SettingsVersion: row.SettingsVersion,
		}
		if row.DeviceID != nil {
			s.DeviceId = *row.DeviceID
		} else {
			a, err := db.New(r).GetPlexAccount(ctx)
			if err != nil {
				return nil, err
			}
			s.DeviceId = a.ClientIdentifier
		}
		if row.MonitorDelayMs != nil {
			s.MonitorDelayMs = *row.MonitorDelayMs
		}
		in.Servers = append(in.Servers, s)
	}
	return in, nil
}

// Run sends every server's changes and follows the roster and the roots.
// It first returns the changes being sent at the last stop to due, then
// runs one sender per server. At its start and at each change of the
// roster it names the Controller as the reaching Node of every server
// whose reaching Node the Admin removed with a write-off, ends the plays
// of a reaching Node away longer than Mark unhealthy after, and wakes
// every sender to route again. At its start and at each roots.changed it
// unconfirms the mappings of re-homed roots. After each Node's new facts
// it writes when their open plays were last seen.
func (m *MediaServers) Run(ctx context.Context) error {
	rehomed := m.cfg.RootsChanged(ctx)
	m.revert(ctx)
	var wg sync.WaitGroup
	defer wg.Wait()
	m.unconfirm(ctx)
	for {
		changed := m.cfg.Roster.Changed()
		m.repoint(ctx)
		m.follow(ctx, &wg)
		m.senders.wakeAll()
		var away <-chan time.Time
		if next := m.endAway(ctx); !next.IsZero() {
			away = time.After(time.Until(next))
		}
		for wait := true; wait; {
			select {
			case <-ctx.Done():
				return nil
			case <-changed:
				wait = false
			case <-away:
				wait = false
			case <-m.senders.follow:
				wait = false
			case <-m.seen.wake:
				m.writeSeen(ctx)
			case <-rehomed:
				m.unconfirm(ctx)
				wait = false
			}
		}
	}
}

// endAway ends, at each one's last look, the open plays of every server
// whose reaching Node has been away longer than Mark unhealthy after, and
// answers when the next Node away will have been, zero for none.
func (m *MediaServers) endAway(ctx context.Context) time.Time {
	self := m.cfg.Roster.Self()
	var next time.Time
	var ended int64
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		next, ended = time.Time{}, 0
		q := db.New(tx)
		row, err := q.GetSettings(ctx)
		if err != nil {
			return err
		}
		after := UnhealthyAfter.Value(row.UnhealthyAfterSeconds)
		reaching, err := q.ListReachingNodes(ctx)
		if err != nil {
			return err
		}
		now := time.Now()
		for _, node := range reaching {
			due, away := m.awayUntil(node, self, after)
			switch {
			case !away:
			case now.Before(due):
				next = earlier(next, due)
			default:
				n, err := q.EndPlaysOn(ctx, node)
				if err != nil {
					return err
				}
				ended += n
			}
		}
		return nil
	})
	if err != nil {
		if ctx.Err() == nil {
			m.cfg.Log.Error("end the plays of reaching Nodes away too long", "err", err.Error())
		}
		return time.Time{}
	}
	if ended > 0 {
		m.cfg.Log.Info("ended the plays of reaching Nodes away longer than Mark unhealthy after", "plays", ended)
	}
	return next
}

// awayUntil is when node will have been away for after, false while it
// is in step. The Controller is never away.
func (m *MediaServers) awayUntil(node, self role.NodeID, after time.Duration) (time.Time, bool) {
	standing, since := m.cfg.Roster.Standing(node)
	if node == self || standing == nodes.InStep {
		return time.Time{}, false
	}
	return since.Add(after), true
}

// earlier is the earlier of a and b, where zero is none.
func earlier(a, b time.Time) time.Time {
	if a.IsZero() || (!b.IsZero() && b.Before(a)) {
		return b
	}
	return a
}

// repoint moves every server of a written-off reaching Node to the
// Controller. A second pass finds none left to move.
func (m *MediaServers) repoint(ctx context.Context) {
	self := m.cfg.Roster.Self()
	var moved []link.ServerID
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		moved = nil
		written, err := nodes.WrittenOffIn(ctx, tx, role.MediaServerLink)
		if err != nil {
			return err
		}
		q := db.New(tx)
		reaching, err := q.ListReachingNodes(ctx)
		if err != nil {
			return err
		}
		for _, node := range reaching {
			if !written[node] {
				continue
			}
			ids, err := repointFrom(ctx, q, node, self)
			if err != nil {
				return err
			}
			moved = append(moved, ids...)
		}
		return nil
	})
	if err != nil {
		if ctx.Err() == nil {
			m.cfg.Log.Error("name the Controller in place of removed reaching Nodes", "err", err.Error())
		}
		return
	}
	if len(moved) > 0 {
		m.cfg.Log.Info("named the Controller as the reaching Node of the servers of removed Nodes", "servers", len(moved))
		m.cfg.Touch(self)
		m.changed(ctx, moved...)
	}
}

// repointFrom names self as the reaching Node of node's servers, ends the
// plays node saw, and answers which servers they are.
func repointFrom(ctx context.Context, q *db.Queries, node, self role.NodeID) ([]link.ServerID, error) {
	ids, err := q.ListReachedIDs(ctx, node)
	if err != nil {
		return nil, err
	}
	if _, err := q.EndPlaysOn(ctx, node); err != nil {
		return nil, err
	}
	_, err = q.Repoint(ctx, db.RepointParams{Controller: self, At: storage.Now(), NodeID: node})
	return ids, err
}
