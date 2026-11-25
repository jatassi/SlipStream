package mediaservers

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaservers/access"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
)

// usersWay gives access with a user on the server, along a route.
type usersWay struct{ m *MediaServers }

func (usersWay) way() access.Way { return access.WayUsers }

func (usersWay) offered(context.Context, *db.Queries, identity.AccountID) (step, error) {
	return step{}, nil
}

func (w usersWay) howToWatch(a db.MediaserversMemberAccess, srv db.MediaserversServer, _ db.MediaserversMemberPlex) HowToWatch {
	return HowToWatch{App: w.m.kind(srv.Kind).About().Label, SignInAs: deref(a.Username), Address: deref(srv.MemberAddress)}
}
