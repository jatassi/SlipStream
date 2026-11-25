package mediaservers

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaservers/access"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/providers/plextv"
	"github.com/jatassi/slipstream/internal/storage"
)

var (
	// errPlexIDTaken is another Account holding the Plex account by the
	// time the sign-in records it.
	errPlexIDTaken = errors.New("another Account signed in with this Plex account")
	// errAccountGone is a sign-in whose Account was deleted meanwhile.
	errAccountGone = errors.New("the Account was deleted")
)

// plexStanding is an Account's Plex identity and Plex rows as a sign-in
// finds them.
type plexStanding struct {
	held  *plextv.AccountID
	taken bool
	rows  []accessRow
}

// refuses is why a sign-in as id is refused, "" when it isn't: the
// Admin's own account, one another Account holds, or another account while
// a share names the one held.
func (st plexStanding) refuses(id plextv.AccountID, owner *plextv.AccountID) PinRefusal {
	switch {
	case owner != nil && *owner == id:
		return PlexAccountOwner
	case st.taken:
		return PlexAccountTaken
	case st.held != nil && *st.held != id && slices.ContainsFunc(st.rows, func(r accessRow) bool {
		return r.State == access.Shared || r.State == access.Active
	}):
		return PlexAccountOther
	}
	return ""
}

func (m *MediaServers) plexStanding(ctx context.Context, account identity.AccountID, id plextv.AccountID) (plexStanding, error) {
	var st plexStanding
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		q := db.New(rd)
		p, err := q.GetMemberPlex(ctx, account)
		if errors.Is(err, storage.ErrNoRows) {
			return errAccountGone
		}
		if err != nil {
			return err
		}
		st.held = p.PlexID
		holder, err := q.HolderOfPlex(ctx, &id)
		switch {
		case err == nil:
			st.taken = holder != account
		case !errors.Is(err, storage.ErrNoRows):
			return err
		}
		as, err := q.ListAccessOf(ctx, account)
		if err != nil {
			return err
		}
		for _, a := range as {
			if a.Way != access.WayPlexTV {
				continue
			}
			r, err := rowOn(ctx, q, a)
			if err != nil {
				return err
			}
			st.rows = append(st.rows, r)
		}
		return nil
	})
	return st, err
}

// signInRow is a Plex row and what the sign-in's remote calls found for it.
type signInRow struct {
	accessRow
	outcome access.Outcome
}

// memberSignedIn signs account in with the Plex account u, or refuses it
// and signs token out. It looks up each offered and waiting row's server
// for a share or a Plex Home place, and accepts each shared row's share,
// then records the identity, the token and every row's move in one Write.
func (m *MediaServers) memberSignedIn(ctx context.Context, account identity.AccountID, tv *plextv.Member, u plextv.User, token plextv.MemberToken) (MemberPinState, error) {
	release, err := m.turns.take(ctx, accountTurn(account))
	if err != nil {
		return MemberPinState{}, err
	}
	defer release()
	admin, _, err := m.account(ctx)
	if err != nil {
		return MemberPinState{}, err
	}
	st, err := m.plexStanding(ctx, account, u.ID)
	if errors.Is(err, errAccountGone) {
		m.signOutMember(ctx, tv, token)
		return MemberPinState{State: PinExpired}, nil
	}
	if err != nil {
		return MemberPinState{}, err
	}
	if refusal := st.refuses(u.ID, admin.plexID); refusal != "" {
		m.signOutMember(ctx, tv, token)
		return MemberPinState{State: PinRefused, Refusal: refusal}, nil
	}
	who := plextv.Who{ID: u.ID, UUID: u.UUID}
	var rows []signInRow
	for _, r := range st.rows {
		switch r.State {
		case access.Offered, access.Waiting:
			rows = append(rows, signInRow{r, m.alreadyThere(ctx, admin, r, who)})
		case access.Shared:
			rows = append(rows, signInRow{r, m.accept(ctx, tv, token, admin, r, who)})
		case access.None, access.Making, access.Active, access.Removing, access.Removed:
		}
	}
	res, err := m.recordSignIn(ctx, account, u, token, rows)
	switch {
	case errors.Is(err, errPlexIDTaken):
		m.signOutMember(ctx, tv, token)
		return MemberPinState{State: PinRefused, Refusal: PlexAccountTaken}, nil
	case errors.Is(err, errAccountGone):
		m.signOutMember(ctx, tv, token)
		return MemberPinState{State: PinExpired}, nil
	case err != nil:
		return MemberPinState{}, err
	}
	m.cfg.Log.Info("a Member signed in with Plex", "account", account.String())
	m.signOut(ctx, res.signOut)
	m.toAddMail(ctx, u, res.toAdd)
	m.accessChanged(ctx, account)
	return MemberPinState{State: PinSignedIn}, nil
}

// alreadyThere is whether r's server already shares with who, or has them
// in the Admin's Plex Home. A lookup that fails reads as not there: the
// Admin's share looks up again first.
func (m *MediaServers) alreadyThere(ctx context.Context, admin account, r accessRow, who plextv.Who) access.Outcome {
	found, err := m.plexTV(admin).FindShare(ctx, admin.jwt, r.server.Identity, who)
	if err != nil {
		m.cfg.Log.Warn("look up a Member's Plex share", "server", r.server.Name, "err", err.Error())
		return access.NotThere
	}
	if found.Share != "" || found.Home {
		return access.AlreadyThere
	}
	return access.NotThere
}

// accept accepts r's share with the Member's token, and otherwise reads
// whether it is accepted already.
func (m *MediaServers) accept(ctx context.Context, tv *plextv.Member, token plextv.MemberToken, admin account, r accessRow, who plextv.Who) access.Outcome {
	accepted, err := tv.Accept(ctx, token, *r.SharedServerID)
	if err == nil && accepted {
		return access.Accepted
	}
	if err != nil {
		m.cfg.Log.Warn("accept a Member's Plex share", "server", r.server.Name, "err", err.Error())
	}
	found, err := m.plexTV(admin).FindShare(ctx, admin.jwt, r.server.Identity, who)
	if err == nil && found.Accepted {
		return access.Accepted
	}
	return access.NotAccepted
}

type signedIn struct {
	// signOut is the tokens the Write cleared or replaced.
	signOut []clearedToken
	// toAdd is the rows that reached waiting or now name another Plex
	// account.
	toAdd []accessRow
}

// signInMove is one row's move at a sign-in.
type signInMove struct {
	signInRow
	t access.Transition
}

// signInMoves is each row's move, those that leave the token needed
// first, so no move before them clears it.
func signInMoves(rows []signInRow) ([]signInMove, error) {
	var needs, rest []signInMove
	for _, r := range rows {
		t, err := access.Next(r.State, access.PlexSignIn, r.outcome, access.WayPlexTV)
		if err != nil {
			return nil, err
		}
		if t.To == access.Waiting || t.To == access.Shared {
			needs = append(needs, signInMove{r, t})
		} else {
			rest = append(rest, signInMove{r, t})
		}
	}
	return append(needs, rest...), nil
}

// recordSignIn is the sign-in's one Write: the identity and sealed token,
// each row's move, and the token cleared when no row waits on it.
func (m *MediaServers) recordSignIn(ctx context.Context, account identity.AccountID, u plextv.User, token plextv.MemberToken, rows []signInRow) (signedIn, error) {
	moves, err := signInMoves(rows)
	if err != nil {
		return signedIn{}, err
	}
	var res signedIn
	err = m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		old, err := m.signMemberIn(ctx, q, account, u, token)
		if err != nil {
			return err
		}
		res = signedIn{signOut: old}
		for _, mv := range moves {
			if err := m.signInMoved(ctx, q, u, mv, &res); err != nil {
				return err
			}
		}
		idle, err := m.clearIdleToken(ctx, q, account)
		if idle != nil {
			res.signOut = append(res.signOut, *idle)
		}
		return err
	})
	return res, err
}

// signMemberIn writes account's Plex identity and token, and answers the
// token it replaces.
func (m *MediaServers) signMemberIn(ctx context.Context, q *db.Queries, account identity.AccountID, u plextv.User, token plextv.MemberToken) ([]clearedToken, error) {
	_, err := q.GetMemberPlex(ctx, account)
	if errors.Is(err, storage.ErrNoRows) {
		return nil, errAccountGone
	}
	if err != nil {
		return nil, err
	}
	switch holder, err := q.HolderOfPlex(ctx, &u.ID); {
	case err == nil && holder != account:
		return nil, errPlexIDTaken
	case err != nil && !errors.Is(err, storage.ErrNoRows):
		return nil, err
	}
	old, err := m.heldToken(ctx, q, account)
	if err != nil {
		return nil, err
	}
	name := cmpOr(u.Username, u.Title)
	err = q.SignMemberIn(ctx, db.SignMemberInParams{
		PlexID: &u.ID, PlexUuid: &u.UUID, Username: &name, Email: &u.Email, SignedInAt: storage.Now(),
		TokenSealed: m.cfg.Sealer.Seal([]byte(token.Reveal()), memberTokenColumn.Row(account.String())), AccountID: account,
	})
	if old == nil {
		return nil, err
	}
	return []clearedToken{*old}, err
}

// signInMoved records mv. A row its server's removal deleted meanwhile is
// left out.
func (m *MediaServers) signInMoved(ctx context.Context, q *db.Queries, u plextv.User, mv signInMove, res *signedIn) error {
	w, err := m.recordMove(ctx, q, mv.accessRow, mv.t, step{outcome: mv.outcome, set: rowSet{plex: &plextv.Who{ID: u.ID, UUID: u.UUID}}})
	if errors.Is(err, ErrAccessNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if w.token != nil {
		res.signOut = append(res.signOut, *w.token)
	}
	if mv.t.To == access.Waiting && (mv.State != access.Waiting || *mv.PlexID != u.ID) {
		res.toAdd = append(res.toAdd, w.row)
	}
	return nil
}

// signOutMember signs token out on plex.tv. A failure is a log line:
// SlipStream keeps no copy.
func (m *MediaServers) signOutMember(ctx context.Context, tv *plextv.Member, token plextv.MemberToken) {
	if err := tv.SignOut(ctx, token); err != nil {
		m.cfg.Log.Warn("sign a Member's Plex token out", "err", err.Error())
	}
}

func (m *MediaServers) signOut(ctx context.Context, tokens []clearedToken) {
	for _, t := range tokens {
		m.signOutMember(ctx, m.memberDevice(t.clientID), t.token)
	}
}

// toAddMail tells every Admin, through the Outbox, of each row now waiting
// for a share to u's Plex account.
func (m *MediaServers) toAddMail(ctx context.Context, u plextv.User, rows []accessRow) {
	if len(rows) == 0 {
		return
	}
	admins, err := m.cfg.Admins(ctx)
	if err != nil {
		m.cfg.Log.Error("couldn't read the Admins to email a Member's Plex account", "err", err.Error())
		return
	}
	var address identity.Address
	err = m.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		address, err = identity.AddressIn(ctx, rd)
		return err
	})
	if err != nil {
		m.cfg.Log.Error("couldn't read the Address to email a Member's Plex account", "err", err.Error())
		return
	}
	accounts := address.URL().JoinPath("accounts").String()
	for _, r := range rows {
		for _, to := range admins {
			msg, err := plexAccountToAdd.Render(to, toAddData{
				Member: r.MemberName, PlexUsername: cmpOr(u.Username, u.Title), PlexEmail: u.Email, Server: r.server.Name, AccountsURL: accounts,
			})
			if err != nil {
				m.cfg.Log.Error("couldn't write the email of a Member's Plex account", "err", err.Error())
				continue
			}
			m.cfg.Outbox.Post(msg)
		}
	}
}

// AnotherPlex forgets the Member's Plex account while no share names it:
// each waiting row goes back to offered, and the old token is signed out.
func (s service) AnotherPlex(ctx context.Context, me identity.Account) error {
	account := me.Profile().ID
	release, err := s.m.turns.take(ctx, accountTurn(account))
	if err != nil {
		return err
	}
	defer release()
	rows, err := s.m.rowsOf(ctx, account)
	if err != nil {
		return err
	}
	for _, r := range rows {
		if r.Way == access.WayPlexTV && (r.State == access.Shared || r.State == access.Active) {
			return &AccessStateError{Member: r.MemberName, Server: r.server.Name}
		}
	}
	var tokens []clearedToken
	err = s.m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		tokens, err = s.m.forgetPlex(ctx, db.New(tx), account, rows)
		return err
	})
	if err != nil {
		return err
	}
	s.m.signOut(ctx, tokens)
	s.m.accessChanged(ctx, account)
	return nil
}

// forgetPlex clears account's Plex identity and token, moves each waiting
// row back to offered, and answers the token to sign out.
func (m *MediaServers) forgetPlex(ctx context.Context, q *db.Queries, account identity.AccountID, rows []accessRow) ([]clearedToken, error) {
	old, err := m.heldToken(ctx, q, account)
	if err != nil {
		return nil, err
	}
	if err := q.ForgetMemberPlex(ctx, account); err != nil {
		return nil, err
	}
	for _, r := range rows {
		if r.Way != access.WayPlexTV || r.State != access.Waiting {
			continue
		}
		if _, err := m.recordMove(ctx, q, r, access.Must(access.Waiting, access.AnotherPlex, access.Done, access.WayPlexTV), step{}); err != nil {
			return nil, err
		}
	}
	if old == nil {
		return nil, nil
	}
	return []clearedToken{*old}, nil
}

// heldToken is account's Plex token, nil for none.
func (m *MediaServers) heldToken(ctx context.Context, q *db.Queries, account identity.AccountID) (*clearedToken, error) {
	p, err := q.GetMemberPlex(ctx, account)
	if errors.Is(err, storage.ErrNoRows) || (err == nil && !p.TokenSealed.Set()) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	token, err := m.cfg.Sealer.Open(p.TokenSealed, memberTokenColumn.Row(account.String()))
	if err != nil {
		return nil, fmt.Errorf("open a Member's Plex token: %w", err)
	}
	return &clearedToken{clientID: p.ClientIdentifier, token: plextv.NewMemberToken(string(token))}, nil
}

// memberTokens opens every Member's Plex token and clears them all, for a
// sign-out of the Admin's Plex account.
func (m *MediaServers) memberTokens(ctx context.Context, q *db.Queries) ([]clearedToken, error) {
	rows, err := q.ListMemberTokens(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]clearedToken, len(rows))
	for i, r := range rows {
		token, err := m.cfg.Sealer.Open(r.TokenSealed, memberTokenColumn.Row(r.AccountID.String()))
		if err != nil {
			return nil, fmt.Errorf("open a Member's Plex token: %w", err)
		}
		out[i] = clearedToken{clientID: r.ClientIdentifier, token: plextv.NewMemberToken(string(token))}
	}
	return out, q.ClearMemberTokens(ctx)
}
