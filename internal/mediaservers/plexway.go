package mediaservers

import (
	"context"
	"errors"
	"fmt"

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

// who is the row's Plex account, which every row past offered names.
func (r accessRow) who() plextv.Who { return plextv.Who{ID: *r.PlexID, UUID: deref(r.PlexUuid)} }

// remove takes the row's Plex access away: its share, which a waiting row
// or one shared by hand looks up first, and its Plex Home place when
// SlipStream added it there. A failure is the Failed outcome with the
// problem.
func (w plexWay) remove(ctx context.Context, r accessRow) (step, error) {
	if err := w.m.unshare(ctx, r); err != nil {
		return step{outcome: access.Failed, problem: plexProblem(err)}, nil
	}
	return step{}, nil
}

func (m *MediaServers) unshare(ctx context.Context, r accessRow) error {
	admin, err := m.signedIn(ctx)
	if err != nil {
		return err
	}
	tv, who := m.plexTV(admin), r.who()
	share := r.SharedServerID
	if share == nil {
		found, err := tv.FindShare(ctx, admin.jwt, r.server.Identity, who)
		if err != nil {
			return err
		}
		share = &found.Share
	}
	if *share != "" {
		if err := tv.Unshare(ctx, admin.jwt, r.server.Identity, *share, who); err != nil {
			return err
		}
	}
	if r.PlexHome && r.SharedServerID != nil {
		return tv.LeaveHome(ctx, admin.jwt, who)
	}
	return nil
}

// reoffer never runs: a Plex row is never removed.
func (plexWay) reoffer(context.Context, accessRow) (step, error) { return step{}, nil }

// plexProblem is a failed plex.tv call as an Admin reads it.
func plexProblem(err error) *AccessProblem {
	e, replied := errors.AsType[*plextv.Error](err)
	switch {
	case errors.Is(err, ErrPlexNotSignedIn):
		return &AccessProblem{Code: ProblemFailed, Detail: "SlipStream isn't signed in to Plex. Sign in with Plex on the Media Servers page, then try again."}
	case replied && plextv.Declined(err):
		return &AccessProblem{Code: ProblemPlexRefused, Detail: fmt.Sprintf("Plex refused the change. Its response was %q.", e.Detail)}
	case replied:
		return &AccessProblem{Code: ProblemFailed, Detail: fmt.Sprintf("plex.tv responded with status %d. Its response was %q.", e.Status, e.Detail)}
	}
	return &AccessProblem{Code: ProblemFailed, Detail: "plex.tv isn't responding to SlipStream."}
}
