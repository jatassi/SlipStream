package mediaservers

import (
	"context"

	"github.com/jatassi/slipstream/internal/mediaserverlink"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/access"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// memberRoute is where one operation's user calls run. sender.go's via is
// the route of a Job pass, and these are its own names.
type memberRoute uint8

const (
	// memberDirect is the Controller in process.
	memberDirect memberRoute = iota
	// memberThroughNode is the reaching Node's Members RPC.
	memberThroughNode
	// memberHeld is neither: media_server_unreached.
	memberHeld
)

// membersRoute is where row's user calls run. It reads the stored result
// of the Controller's own last test of the fallback address, not a live
// probe, so one operation never switches route after a call that may have
// landed, and a Member's password stays on the Controller whenever it can.
func membersRoute(row db.MediaserversServer, self role.NodeID, nodeUp bool) memberRoute {
	switch {
	case row.ReachingNodeID == self, row.Fallback == "available":
		return memberDirect
	case nodeUp:
		return memberThroughNode
	}
	return memberHeld
}

// usersAt is a server's users along the route one operation decided on
// once.
type usersAt struct {
	link.Users
	server string
	// fallback is the Controller in process at the fallback address, for a
	// server another Node reaches.
	fallback bool
}

// usersOf is server id's users along its route: ErrNotFound for a server
// whose Members get in some other way, AccessUnreachedError for no route.
func (m *MediaServers) usersOf(ctx context.Context, id link.ServerID) (usersAt, error) {
	var sv saved
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		sv, err = m.load(ctx, r, id)
		return err
	})
	if err != nil {
		return usersAt{}, err
	}
	if m.wayOf(sv.row.Kind).way() != access.WayUsers {
		return usersAt{}, ErrNotFound
	}
	self := m.cfg.Roster.Self()
	at := usersAt{server: sv.row.Name}
	switch membersRoute(sv.row, self, m.up(sv.row.ReachingNodeID)) {
	case memberDirect:
		address := sv.row.Address
		if sv.row.ReachingNodeID != self {
			address, at.fallback = cmpOr(deref(sv.row.FallbackAddress), sv.row.Address), true
		}
		conn := sv.kind.Dial(link.Dialed{
			Server: sv.ref(), Address: address, Credential: sv.cred, DeviceID: sv.device, HTTP: m.cfg.HTTP, Timings: m.cfg.Timings,
		})
		at.Users, _ = mediaserverlink.UsersOf(conn)
	case memberThroughNode:
		at.Users = mediaserverlink.RemoteUsers(m.cfg.Link(sv.row.ReachingNodeID), sv.ref())
	case memberHeld:
		return usersAt{}, &AccessUnreachedError{Server: sv.row.Name}
	}
	return at, nil
}

// ref is the server as the link names it.
func (sv saved) ref() link.ServerRef {
	return link.ServerRef{ID: sv.row.ID, Kind: sv.row.Kind, Identity: sv.row.Identity, SettingsVersion: sv.row.SettingsVersion}
}
