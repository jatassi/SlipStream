package mediaservers

import (
	"context"
	"log/slog"
	"net/http"
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
	cfg  Config
	pins pins
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
	Holds(name role.Name, node role.NodeID) bool
	Holding(name role.Name) []role.NodeID
}

func New(cfg Config) *MediaServers { return &MediaServers{cfg: cfg} }

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

// Run follows the roster and the roots: at its start and at each change
// of the roster it names the Controller as the reaching Node of every
// server whose reaching Node the Admin removed with a write-off, and at
// its start and at each roots.changed it unconfirms the mappings of
// re-homed roots.
func (m *MediaServers) Run(ctx context.Context) error {
	rehomed := m.cfg.RootsChanged(ctx)
	m.unconfirm(ctx)
	for {
		changed := m.cfg.Roster.Changed()
		m.repoint(ctx)
		select {
		case <-ctx.Done():
			return nil
		case <-changed:
		case <-rehomed:
			m.unconfirm(ctx)
		}
	}
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

// repointFrom names self as the reaching Node of node's servers, and
// answers which they are.
func repointFrom(ctx context.Context, q *db.Queries, node, self role.NodeID) ([]link.ServerID, error) {
	ids, err := q.ListReachedIDs(ctx, node)
	if err != nil {
		return nil, err
	}
	_, err = q.Repoint(ctx, db.RepointParams{Controller: self, At: storage.Now(), NodeID: node})
	return ids, err
}
