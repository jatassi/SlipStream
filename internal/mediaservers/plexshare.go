package mediaservers

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/mediaservers/access"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/providers/plextv"
	"github.com/jatassi/slipstream/internal/storage"
)

// checkAgain is how long after a row's last change Check again answers the
// row as it is, without asking plex.tv.
const checkAgain = 10 * time.Second

// plexWorks is what the Admin's Plex sign-in can do through plex.tv, as the
// recorded Plex run found it: share, and invite into the Plex Home.
func (m *MediaServers) plexWorks(plexPass bool) (shares, home bool) {
	f := m.cfg.PlexFacts
	return f.AdminShares, f.AdminInvitesHome && (plexPass || !f.HomeNeedsPlexPass)
}

// Share shares the row's Libraries for Members with the Member's Plex
// account, as a friend or in the Plex Home, accepts it with the Member's
// held token and emails the Member. A share plex.tv refuses keeps the row
// waiting with plex.tv's reply.
func (s service) Share(ctx context.Context, _ identity.Admin, id AccessID, as Invite) (Acted, error) {
	on := access.ShareFriend
	if as == InviteHome {
		on = access.ShareHome
	}
	var ran *step
	var refused error
	a, err := s.m.act(ctx, id, on, func(ctx context.Context, r accessRow) (step, error) {
		st, err := s.m.share(ctx, r, on)
		if err == nil && st.outcome == access.Refused {
			refused = &PlexRefusedError{Server: r.server.Name, Reply: st.problem.Detail}
		}
		ran = &st
		return st, err
	})
	switch {
	case err != nil:
		return Acted{}, err
	case refused != nil:
		return Acted{}, refused
	case ran != nil:
		a.Mail = s.m.canWatch(ctx, *a.Row, ran.outcome == access.Accepted)
	}
	return a, nil
}

// share is the remote part of a share: the plex.tv ids of the Libraries
// for Members, a share or Plex Home place the Member already has, the
// Home invite, the share, and the accept.
func (m *MediaServers) share(ctx context.Context, r accessRow, on access.Event) (step, error) {
	admin, err := m.signedIn(ctx)
	if err != nil {
		return step{}, err
	}
	works, home := m.plexWorks(admin.plexPass)
	if on == access.ShareHome {
		works = home
	}
	if !works {
		return step{}, &AccessStateError{Member: r.MemberName, Server: r.server.Name}
	}
	tv, who := m.plexTV(admin), r.who()
	sections, err := tv.Sections(ctx, admin.jwt, r.server.Identity)
	if err != nil {
		return step{}, fromPlexTV(err)
	}
	ids := make([]plextv.SectionID, len(r.libraries))
	for i, l := range r.libraries {
		id, ok := sections[l.Key]
		if !ok {
			return step{}, &LibrariesShortError{Library: l.Name, Server: r.server.Name}
		}
		ids[i] = id
	}
	found, err := tv.FindShare(ctx, admin.jwt, r.server.Identity, who)
	if err != nil {
		return step{}, fromPlexTV(err)
	}
	if on == access.ShareHome && !found.Home {
		invitee, err := m.invitee(ctx, r)
		if err != nil {
			return step{}, err
		}
		if err := tv.InviteHome(ctx, admin.jwt, invitee); err != nil {
			return declined(err)
		}
	}
	share := found.Share
	if share == "" {
		if share, err = tv.Share(ctx, admin.jwt, r.server.Identity, who.ID, ids); err != nil {
			return declined(err)
		}
	}
	st := step{outcome: access.NotAccepted, set: rowSet{shareID: &share}}
	if found.Accepted || m.acceptWithHeldToken(ctx, r, share) {
		st.outcome = access.Accepted
	}
	return st, nil
}

// declined is a sharing call plex.tv refused as the row keeps it, with
// plex.tv's reply, and any other failure as the operation's error.
func declined(err error) (step, error) {
	if e, ok := errors.AsType[*plextv.Error](err); ok && plextv.Declined(err) {
		return step{outcome: access.Refused, problem: &AccessProblem{Code: ProblemPlexRefused, Detail: e.Detail}}, nil
	}
	return step{}, fromPlexTV(err)
}

// invitee is who a Plex Home invite for r addresses.
func (m *MediaServers) invitee(ctx context.Context, r accessRow) (plextv.Invitee, error) {
	var p db.MediaserversMemberPlex
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		p, err = db.New(rd).GetMemberPlex(ctx, r.AccountID)
		return err
	})
	return plextv.Invitee{Who: r.who(), Username: deref(p.Username), Email: deref(p.Email)}, err
}

// acceptWithHeldToken accepts share with the Member's held Plex token:
// false when no token is held or plex.tv doesn't accept it.
func (m *MediaServers) acceptWithHeldToken(ctx context.Context, r accessRow, share plextv.ShareID) bool {
	var held *clearedToken
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		held, err = m.heldToken(ctx, db.New(rd), r.AccountID)
		return err
	})
	if err != nil {
		m.cfg.Log.Error("read a Member's Plex token", "err", err.Error())
		return false
	}
	if held == nil {
		return false
	}
	accepted, err := m.memberDevice(held.clientID).Accept(ctx, held.token, share)
	if err != nil {
		m.cfg.Log.Warn("accept a Member's Plex share", "server", r.server.Name, "err", err.Error())
		return false
	}
	return accepted
}

// MarkAdded records that an Admin shared the server with the Member in Plex
// itself, and emails the Member.
func (s service) MarkAdded(ctx context.Context, _ identity.Admin, id AccessID) (Acted, error) {
	ran := false
	a, err := s.m.act(ctx, id, access.MarkAdded, func(context.Context, accessRow) (step, error) {
		ran = true
		return step{}, nil
	})
	if err == nil && ran {
		a.Mail = s.m.canWatch(ctx, *a.Row, false)
	}
	return a, err
}

// canWatch emails v's Member that they can watch its server on Plex, and
// waits for the outcome. accepted is a share SlipStream accepted for them.
func (m *MediaServers) canWatch(ctx context.Context, v AccessView, accepted bool) *mail.Outcome {
	admin, _, err := m.account(ctx)
	if err != nil {
		m.cfg.Log.Error("couldn't read the Plex account to email a Member", "err", err.Error())
		return &mail.Outcome{Kind: mail.Failed}
	}
	msg, err := canWatchOnPlex.Render(mail.Address{Name: v.Member.Name, Email: v.Member.Email}, canWatchData{
		Server: v.Server.Name, PlexUsername: v.Plex.Username, Accepted: accepted, PlexPassLine: !admin.plexPass,
	})
	if err != nil {
		m.cfg.Log.Error("couldn't write the email that a Member can watch on Plex", "err", err.Error())
		return &mail.Outcome{Kind: mail.Failed}
	}
	out, err := m.cfg.Mail.Send(ctx, msg)
	if err != nil {
		m.cfg.Log.Error("couldn't send the email that a Member can watch on Plex", "err", err.Error())
		return &mail.Outcome{Kind: mail.Failed}
	}
	return &out
}

// CheckShare moves a shared row to active once the Member accepted the
// share, accepting it with their held token when SlipStream can.
func (s service) CheckShare(ctx context.Context, _ identity.Admin, id AccessID) (Acted, error) {
	r, err := s.m.row(ctx, id)
	if err != nil {
		return Acted{}, err
	}
	if checkedLately(r) {
		return s.m.acted(ctx, id)
	}
	return s.m.act(ctx, id, access.Check, s.m.checkAccept)
}

// CheckMine is CheckShare on the Member's own row.
func (s service) CheckMine(ctx context.Context, me identity.Account, id AccessID) (MyAccessRow, error) {
	r, release, err := s.m.mine(ctx, me, id)
	if err != nil {
		return MyAccessRow{}, err
	}
	defer release()
	if !checkedLately(r) {
		if _, err := s.m.actLocked(ctx, r, access.Check, s.m.checkAccept); err != nil {
			return MyAccessRow{}, err
		}
	}
	return s.myRowOf(ctx, me, id)
}

// checkedLately is a shared row shared or checked less than checkAgain
// ago: a Check of it moves it, or keeps it shared with a new time.
func checkedLately(r accessRow) bool {
	return r.State == access.Shared && time.Since(r.ChangedAt.Time()) < checkAgain
}

// checkAccept is whether a shared row's share is accepted now: plex.tv lists it
// accepted, or the Member's held token accepts it.
func (m *MediaServers) checkAccept(ctx context.Context, r accessRow) (step, error) {
	admin, err := m.signedIn(ctx)
	if err != nil {
		return step{}, err
	}
	found, err := m.plexTV(admin).FindShare(ctx, admin.jwt, r.server.Identity, r.who())
	if err != nil {
		return step{}, fromPlexTV(err)
	}
	if found.Accepted || m.acceptWithHeldToken(ctx, r, *r.SharedServerID) {
		return step{outcome: access.Accepted}, nil
	}
	return step{outcome: access.NotAccepted}, nil
}

// checkShared checks every shared row, so a share its Member accepted in
// Plex's email moves to active.
func (m *MediaServers) checkShared(ctx context.Context) {
	var ids []AccessID
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		as, err := db.New(r).ListAccessIn(ctx, access.Shared)
		for _, a := range as {
			ids = append(ids, a.ID)
		}
		return err
	})
	if err != nil {
		m.cfg.Log.Error("read the Plex shares Members haven't accepted", "err", err.Error())
		return
	}
	for _, id := range ids {
		_, err := m.act(ctx, id, access.Check, m.checkAccept)
		if _, moved := errors.AsType[*AccessStateError](err); err != nil && !moved && !errors.Is(err, ErrAccessNotFound) {
			m.cfg.Log.Warn("check whether a Member accepted a Plex share", "err", err.Error())
		}
	}
}
