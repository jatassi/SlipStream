package mediaservers

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaservers/access"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/providers/plextv"
	"github.com/jatassi/slipstream/internal/storage"
)

// plexWay gives access through plex.tv, from the Controller.
type plexWay struct{ m *MediaServers }

func (plexWay) way() access.Way { return access.WayPlexTV }

// offered waits for the Member's Plex sign-in, or for the Admin's share
// once the Member signed in.
func (plexWay) offered(ctx context.Context, q *db.Queries, account identity.AccountID) (step, error) {
	p, err := q.GetMemberPlex(ctx, account)
	if errors.Is(err, storage.ErrNoRows) || (err == nil && p.PlexID == nil) {
		return step{outcome: access.NoIdentity}, nil
	}
	if err != nil {
		return step{}, err
	}
	who := plextv.Who{ID: *p.PlexID, UUID: deref(p.PlexUuid)}
	return step{outcome: access.IdentityHeld, set: rowSet{plex: &who}}, nil
}

func (w plexWay) howToWatch(_ db.MediaserversMemberAccess, srv db.MediaserversServer, plex db.MediaserversMemberPlex) HowToWatch {
	return HowToWatch{App: w.m.kind(srv.Kind).About().Label, SignInAs: deref(plex.Username)}
}
