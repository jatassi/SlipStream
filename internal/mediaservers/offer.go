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

// Offer offers account access to server, or changes a Plex offer's Plex
// Home before the share, or offers a removed Jellyfin or Emby access
// again.
func (s service) Offer(ctx context.Context, by identity.Admin, account identity.AccountID, server link.ServerID, plexHome bool) (Acted, error) {
	release, err := s.m.turns.take(ctx, accountTurn(account))
	if err != nil {
		return Acted{}, err
	}
	defer release()
	var r accessRow
	err = s.m.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		q := db.New(rd)
		srv, err := q.GetServer(ctx, server)
		if errors.Is(err, storage.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		who, err := identity.Contact(ctx, rd, account)
		if errors.Is(err, storage.ErrNoRows) {
			return identity.ErrNoAccount
		}
		if err != nil {
			return err
		}
		w := s.m.wayOf(srv.Kind)
		if err := grantable(srv, w, plexHome); err != nil {
			return err
		}
		if who.Admin {
			return &AccessStateError{Member: who.Name, Server: srv.Name}
		}
		a, err := q.GetAccessOn(ctx, db.GetAccessOnParams{ServerID: server, AccountID: account})
		if errors.Is(err, storage.ErrNoRows) {
			r = newRow(srv, w.way(), account, who.Name, adminID(by))
			return nil
		}
		if err != nil {
			return err
		}
		r, err = rowOn(ctx, q, a)
		return err
	})
	if err != nil {
		return Acted{}, err
	}
	return s.m.actLocked(ctx, r, access.Offer, s.m.offerStep(plexHome))
}

// offerStep is the remote part of an offer: a new row's first state, a
// removed account turned back on, or the Plex Home choice.
func (m *MediaServers) offerStep(plexHome bool) runStep {
	return func(ctx context.Context, r accessRow) (step, error) {
		w := m.wayOf(r.server.Kind)
		if r.State == access.Removed {
			return w.reoffer(ctx, r)
		}
		var st step
		if r.State == access.None {
			err := m.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
				var err error
				st, err = w.offered(ctx, db.New(rd), r.AccountID)
				return err
			})
			if err != nil {
				return step{}, err
			}
		}
		st.set.plexHome = &plexHome
		return st, nil
	}
}

// adminID is the Admin's Account, zero for the API key.
func adminID(by identity.Admin) identity.AccountID {
	if a, ok := by.Account(); ok {
		return a.Profile().ID
	}
	return identity.AccountID{}
}

// RemoveAccess takes the row's access away, or tries again on a row whose
// removal failed.
func (s service) RemoveAccess(ctx context.Context, _ identity.Admin, id AccessID) (Acted, error) {
	r, release, err := s.m.lockRow(ctx, id)
	if err != nil {
		return Acted{}, err
	}
	defer release()
	on := access.Remove
	if r.State == access.Removing {
		on = access.RetryRemove
	}
	return s.m.actLocked(ctx, r, on, s.m.takeAway)
}
