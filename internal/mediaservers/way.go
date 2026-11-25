package mediaservers

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/access"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
)

// way is how one family of kinds gives and takes access. One per
// access.Way.
type way interface {
	way() access.Way
	// offered is a new offer to account: Plex reads whether the Member
	// signed in to Plex.
	offered(ctx context.Context, q *db.Queries, account identity.AccountID) (step, error)
	// remove takes the row's access away on the server: Done, Left or
	// Failed with the problem.
	remove(ctx context.Context, r accessRow) (step, error)
	// reoffer turns a removed row's access back on: Done, Gone or Failed.
	reoffer(ctx context.Context, r accessRow) (step, error)
	// howToWatch is what the Member needs to watch in the kind's app.
	howToWatch(a db.MediaserversMemberAccess, srv db.MediaserversServer, plex db.MediaserversMemberPlex) HowToWatch
}

var ways = map[access.Way]func(*MediaServers) way{
	access.WayPlexTV: func(m *MediaServers) way { return plexWay{m} },
	access.WayUsers:  func(m *MediaServers) way { return usersWay{m} },
}

// wayOf is the one place a kind's link.Access becomes an access.Way.
func (m *MediaServers) wayOf(k link.KindName) way {
	w := access.WayUsers
	if m.kind(k).About().Access == link.AccessPlexTV {
		w = access.WayPlexTV
	}
	return ways[w](m)
}

func (m *MediaServers) kind(k link.KindName) link.Kind {
	kind, _ := m.cfg.Kinds.Of(k)
	return kind
}
