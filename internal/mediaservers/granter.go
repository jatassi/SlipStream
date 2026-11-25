package mediaservers

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/access"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// granter is Media Server access as identity sees it: the Media Servers a
// Signup link offers, and the access an Account delete removes first.
type granter struct{ m *MediaServers }

var _ identity.Granter = granter{}

func (m *MediaServers) Granter() identity.Granter { return granter{m} }

// grantable refuses an offer of srv by w: a server not shown to Members,
// or Plex Home on a server that isn't Plex.
func grantable(srv db.MediaserversServer, w way, plexHome bool) error {
	if !srv.MembersSee {
		return &identity.NotGrantableError{Name: srv.Name, Why: identity.NotShown}
	}
	if plexHome && w.way() != access.WayPlexTV {
		return &identity.NotGrantableError{Name: srv.Name, Why: identity.NotPlexWay}
	}
	return nil
}

func (g granter) CheckGrants(ctx context.Context, r storage.Reader[storage.Library], gr identity.Grants) error {
	q := db.New(r)
	for _, sg := range gr.MediaServers {
		srv, err := q.GetServer(ctx, link.ServerID{UUID: sg.Server})
		if errors.Is(err, storage.ErrNoRows) {
			return &identity.NotGrantableError{Name: sg.Server.String(), Why: identity.NotShown}
		}
		if err != nil {
			return err
		}
		if err := grantable(srv, g.m.wayOf(srv.Kind), sg.PlexHome); err != nil {
			return err
		}
	}
	return nil
}

func (g granter) Describe(ctx context.Context, r storage.Reader[storage.Library], gr identity.Grants) (identity.Described, error) {
	q := db.New(r)
	out := identity.Described{MediaServers: make([]identity.GrantedServer, len(gr.MediaServers))}
	for i, sg := range gr.MediaServers {
		out.MediaServers[i] = identity.GrantedServer{Server: sg.Server, PlexHome: sg.PlexHome}
		srv, err := q.GetServer(ctx, link.ServerID{UUID: sg.Server})
		if errors.Is(err, storage.ErrNoRows) {
			continue
		}
		if err != nil {
			return identity.Described{}, err
		}
		d := &out.MediaServers[i]
		d.Name, d.Kind, d.Present, d.Shown = srv.Name, g.m.kind(srv.Kind).About().Label, true, srv.MembersSee
	}
	return out, nil
}

// Redeemed offers the new Account each server the link names that is
// still shown to Members, in identity's redemption Write.
func (g granter) Redeemed(ctx context.Context, tx *storage.Tx[storage.Library], red identity.Redemption) (func(context.Context), error) {
	q := db.New(tx)
	who, err := identity.Contact(ctx, tx, red.Account)
	if err != nil {
		return nil, err
	}
	for _, sg := range red.Grants.MediaServers {
		srv, err := q.GetServer(ctx, link.ServerID{UUID: sg.Server})
		if errors.Is(err, storage.ErrNoRows) || (err == nil && !srv.MembersSee) {
			continue
		}
		if err != nil {
			return nil, err
		}
		w := g.m.wayOf(srv.Kind)
		st, err := w.offered(ctx, q, red.Account)
		if err != nil {
			return nil, err
		}
		st.set.plexHome = &sg.PlexHome
		r := newRow(srv, w.way(), red.Account, who.Name, red.IssuedBy)
		if _, err := g.m.recordMove(ctx, q, r, access.Must(access.None, access.Offer, st.outcome, w.way()), st); err != nil {
			return nil, err
		}
	}
	return func(ctx context.Context) { g.m.accessChanged(ctx, red.Account) }, nil
}
