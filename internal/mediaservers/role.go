package mediaservers

import (
	"context"
	"log/slog"

	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/nodes"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// MediaServers is the Controller's side of the Media Server link Role:
// each Node's intent, the reports it applies and the reaching Nodes it
// follows on the roster.
type MediaServers struct{ cfg Config }

type Config struct {
	Library *storage.DB[storage.Library]
	// Roster is read once it opens, after MediaServers is made.
	Roster Roster
	// Touch follows every commit that changes a Node's intent.
	Touch func(nodes ...role.NodeID)
	Log   *slog.Logger
}

// Roster is what MediaServers reads of the roster.
type Roster interface {
	Self() role.NodeID
	Changed() <-chan struct{}
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
	in := &msv1.Intent{}
	for _, row := range rows {
		s := &msv1.Server{
			Id: row.ID.Bytes(), Kind: string(row.Kind), Name: row.Name, Address: row.Address,
			Identity: row.Identity, SettingsVersion: row.SettingsVersion,
		}
		if row.DeviceID != nil {
			s.DeviceId = *row.DeviceID
		}
		if row.MonitorDelayMs != nil {
			s.MonitorDelayMs = *row.MonitorDelayMs
		}
		in.Servers = append(in.Servers, s)
	}
	return in, nil
}

// Run names the Controller as the reaching Node of every server whose
// reaching Node the Admin removed with a write-off, at its start and at
// each change of the roster.
func (m *MediaServers) Run(ctx context.Context) error {
	for {
		changed := m.cfg.Roster.Changed()
		m.repoint(ctx)
		select {
		case <-ctx.Done():
			return nil
		case <-changed:
		}
	}
}

// repoint moves every server of a written-off reaching Node to the
// Controller. A second pass finds none left to move.
func (m *MediaServers) repoint(ctx context.Context) {
	self := m.cfg.Roster.Self()
	var moved int64
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		moved = 0
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
			n, err := q.Repoint(ctx, db.RepointParams{Controller: self, At: storage.Now(), NodeID: node})
			if err != nil {
				return err
			}
			moved += n
		}
		return nil
	})
	if err != nil {
		if ctx.Err() == nil {
			m.cfg.Log.Error("name the Controller in place of removed reaching Nodes", "err", err.Error())
		}
		return
	}
	if moved > 0 {
		m.cfg.Log.Info("named the Controller as the reaching Node of the servers of removed Nodes", "servers", moved)
		m.cfg.Touch(self)
	}
}
