package mediaservers

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/providers/plextv"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Record is a Media Server as S51 (Reachability diagnostic) reads it.
type Record struct {
	ID       link.ServerID
	Kind     link.KindName
	Name     string
	Address  string
	Identity string
	Reaching role.NodeID
	// RunsOn is zero when the Admin named no Node. It may name a removed
	// Node, which the reader checks.
	RunsOn role.NodeID
}

// ServersIn is every Media Server's record in rd, by name.
func ServersIn(ctx context.Context, rd storage.Reader[storage.Library]) ([]Record, error) {
	rows, err := db.New(rd).ListServers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Record, len(rows))
	for i, r := range rows {
		out[i] = Record{
			ID: r.ID, Kind: r.Kind, Name: r.Name, Address: r.Address, Identity: r.Identity, Reaching: r.ReachingNodeID, RunsOn: r.RunsOnNodeID,
		}
	}
	return out, nil
}

// Exposure is what one Media Server publishes for use away from home, as
// its reaching Node and plex.tv report it now.
type Exposure struct {
	Server   link.ServerID
	Kind     link.KindName
	Name     string
	Reaching role.NodeID
	// RunsOn is zero when the Admin named no Node or its Node was removed.
	RunsOn role.NodeID
	// Exposed is nil while the reaching Node is away: it is never read
	// from the Controller in its place.
	Exposed *link.Exposed
	// Resource is plex.tv's listing of the server, nil unless it is a Plex
	// and SlipStream is signed in to Plex. It carries no token.
	Resource *plextv.Server
	At       time.Time
}

// Exposure reads server id's Exposed through its reaching Node and, for a
// Plex, its plex.tv resource. A server's own failure comes back as a
// *link.ServerError.
func (m *MediaServers) Exposure(ctx context.Context, id link.ServerID) (Exposure, error) {
	var row db.MediaserversServer
	removed := false
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		if row, err = db.New(r).GetServer(ctx, id); err != nil {
			return err
		}
		names, err := m.names(ctx, r)
		removed = names[row.RunsOnNodeID].Removed
		return err
	})
	if errors.Is(err, storage.ErrNoRows) {
		return Exposure{}, ErrNotFound
	}
	if err != nil {
		return Exposure{}, err
	}
	x := Exposure{Server: row.ID, Kind: row.Kind, Name: row.Name, Reaching: row.ReachingNodeID, RunsOn: row.RunsOnNodeID, At: time.Now()}
	if removed {
		x.RunsOn = role.NodeID{}
	}
	if m.up(row.ReachingNodeID) {
		resp, err := m.cfg.Link(row.ReachingNodeID).Exposed(ctx, &msv1.ExposedRequest{ServerId: row.ID.Bytes(), SettingsVersion: row.SettingsVersion})
		if err != nil {
			return Exposure{}, err
		}
		exposed, err := mediaserverlink.ExposedOf(resp)
		if err != nil {
			return Exposure{}, err
		}
		x.Exposed = &exposed
	}
	if row.Kind != "plex" {
		return x, nil
	}
	a, err := m.signedIn(ctx)
	if errors.Is(err, ErrPlexNotSignedIn) {
		return x, nil
	}
	if err != nil {
		return Exposure{}, err
	}
	servers, err := m.plexServers(ctx, a)
	if err != nil {
		return Exposure{}, err
	}
	if i := slices.IndexFunc(servers, func(s plextv.Server) bool { return s.Identifier == row.Identity }); i >= 0 {
		srv := servers[i]
		srv.Token = ""
		x.Resource = &srv
	}
	return x, nil
}
