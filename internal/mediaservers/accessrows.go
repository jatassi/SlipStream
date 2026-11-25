package mediaservers

import (
	"context"
	"errors"
	"fmt"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaservers/access"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/providers/plextv"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
)

// memberTokenColumn seals a Member's Plex token, keyed by their Account.
var memberTokenColumn = secret.NewColumn("mediaservers_member_plex.token_sealed")

// accessRow is an access row with what its moves read of its server: the
// name, kind, Plex machineIdentifier and Libraries for Members.
type accessRow struct {
	db.MediaserversMemberAccess
	server    db.MediaserversServer
	libraries []LibraryChoice
}

// turn is the Account's, or the row's own once the Account is gone.
func (r accessRow) turn() turn {
	if r.AccountID.IsZero() {
		return turn{row: r.ID}
	}
	return accountTurn(r.AccountID)
}

// step is what a run of remote calls hands to the Write.
type step struct {
	outcome access.Outcome
	problem *AccessProblem
	set     rowSet
}

// rowSet is what a step sets on the row besides its Mark.
type rowSet struct {
	shareID  *plextv.ShareID
	plex     *plextv.Who
	userID   *string
	username *string
	plexHome *bool
}

// runStep is the remote calls of one move, run under the row's turn.
type runStep func(context.Context, accessRow) (step, error)

// moved is a move once it commits.
type moved struct {
	row accessRow
	t   access.Transition
	// account is the row's Account before the move, zero for a row whose
	// Account is gone.
	account identity.AccountID
	// token is the Member's Plex token the move cleared, signed out after
	// the commit.
	token *clearedToken
}

type clearedToken struct {
	clientID string
	token    plextv.MemberToken
}

func (m *MediaServers) readRow(ctx context.Context, r storage.Reader[storage.Library], id AccessID) (accessRow, error) {
	q := db.New(r)
	a, err := q.GetAccess(ctx, id)
	if errors.Is(err, storage.ErrNoRows) {
		return accessRow{}, ErrAccessNotFound
	}
	if err != nil {
		return accessRow{}, err
	}
	return rowOn(ctx, q, a)
}

// rowOn is a with its server and the server's Libraries for Members.
func rowOn(ctx context.Context, q *db.Queries, a db.MediaserversMemberAccess) (accessRow, error) {
	srv, err := q.GetServer(ctx, a.ServerID)
	if err != nil {
		return accessRow{}, err
	}
	libs, err := q.ListMemberLibraries(ctx, a.ServerID)
	return accessRow{MediaserversMemberAccess: a, server: srv, libraries: choices(libs)}, err
}

func choices(libs []db.MediaserversMemberLibrary) []LibraryChoice {
	out := make([]LibraryChoice, len(libs))
	for i, l := range libs {
		out[i] = LibraryChoice{Key: l.LibraryKey, Name: l.Name}
	}
	return out
}

func (m *MediaServers) row(ctx context.Context, id AccessID) (accessRow, error) {
	var r accessRow
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		r, err = m.readRow(ctx, rd, id)
		return err
	})
	return r, err
}

// lockRow reads row id and takes its turn, reading it again under the
// turn. A row whose Account was deleted meanwhile moved to a turn of its
// own, which it takes instead.
func (m *MediaServers) lockRow(ctx context.Context, id AccessID) (accessRow, func(), error) {
	r, err := m.row(ctx, id)
	if err != nil {
		return accessRow{}, nil, err
	}
	for {
		release, err := m.turns.take(ctx, r.turn())
		if err != nil {
			return accessRow{}, nil, err
		}
		again, err := m.row(ctx, id)
		if err != nil {
			release()
			return accessRow{}, nil, err
		}
		if again.turn() == r.turn() {
			return again, release, nil
		}
		release()
		r = again
	}
}

// move is the only Write of an access row. It runs t, compares and sets
// on t.From, applies t.Mark and st, deletes the row when t.To is None, and
// clears the Member's Plex token once no waiting or shared row of theirs
// remains.
func (m *MediaServers) move(ctx context.Context, r accessRow, t access.Transition, st step) (moved, error) {
	var w moved
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		w, err = m.recordMove(ctx, db.New(tx), r, t, st)
		return err
	})
	return w, err
}

// recordMove is move's Write, for a caller already inside one.
func (m *MediaServers) recordMove(ctx context.Context, q *db.Queries, r accessRow, t access.Transition, st step) (moved, error) {
	next := r
	next.State, next.ChangedAt = t.To, storage.Now()
	st.set.onto(&next.MediaserversMemberAccess)
	mark(&next.MediaserversMemberAccess, t.Mark, st.problem)
	a := next.MediaserversMemberAccess
	var n int64
	var err error
	switch {
	case t.From == access.None:
		n = 1
		err = q.InsertAccess(ctx, db.InsertAccessParams{
			ID: a.ID, ServerID: a.ServerID, Way: a.Way, AccountID: a.AccountID, MemberName: a.MemberName, State: a.State,
			At: a.ChangedAt, OfferedBy: a.OfferedBy, PlexID: a.PlexID, PlexUuid: a.PlexUuid, PlexHome: a.PlexHome,
		})
	case t.To == access.None:
		n, err = q.DeleteAccessIn(ctx, db.DeleteAccessInParams{ID: a.ID, State: t.From})
	default:
		n, err = q.MoveAccess(ctx, db.MoveAccessParams{
			ID: a.ID, FromState: t.From, State: a.State, ChangedAt: a.ChangedAt, AccountID: a.AccountID,
			PlexID: a.PlexID, PlexUuid: a.PlexUuid, SharedServerID: a.SharedServerID, ByHand: a.ByHand, PlexHome: a.PlexHome,
			ServerUserID: a.ServerUserID, Username: a.Username, ProblemCode: a.ProblemCode, ProblemDetail: a.ProblemDetail,
		})
	}
	if err != nil {
		return moved{}, err
	}
	if n == 0 {
		return moved{}, lost(ctx, q, r)
	}
	w := moved{row: next, t: t, account: r.AccountID}
	if !r.AccountID.IsZero() {
		w.token, err = m.clearIdleToken(ctx, q, r.AccountID)
	}
	return w, err
}

// lost is the error of a move whose row left its state: not found once
// it is gone, the state changed otherwise.
func lost(ctx context.Context, q *db.Queries, r accessRow) error {
	_, err := q.GetAccess(ctx, r.ID)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return ErrAccessNotFound
	case err != nil:
		return err
	}
	return &AccessStateError{Member: r.MemberName, Server: r.server.Name}
}

func (s rowSet) onto(a *db.MediaserversMemberAccess) {
	if s.shareID != nil {
		a.SharedServerID = s.shareID
	}
	if s.plex != nil {
		a.PlexID, a.PlexUuid = &s.plex.ID, &s.plex.UUID
	}
	if s.userID != nil {
		a.ServerUserID = s.userID
	}
	if s.username != nil {
		a.Username = s.username
	}
	if s.plexHome != nil {
		a.PlexHome = *s.plexHome
	}
}

func mark(a *db.MediaserversMemberAccess, mk access.Mark, p *AccessProblem) {
	if mk&access.MarkByHand != 0 {
		a.ByHand = true
	}
	if mk&access.SetPlexHome != 0 {
		a.PlexHome = true
	}
	if mk&access.ClearPlexHome != 0 {
		a.PlexHome = false
	}
	if mk&access.ClearPlex != 0 {
		a.PlexID, a.PlexUuid, a.SharedServerID = nil, nil, nil
	}
	if mk&access.ClearUser != 0 {
		a.ServerUserID, a.Username = nil, nil
	}
	if mk&access.ClearAccount != 0 {
		a.AccountID = identity.AccountID{}
	}
	a.ProblemCode, a.ProblemDetail = nil, nil
	if mk&access.KeepProblem != 0 && p != nil {
		a.ProblemCode, a.ProblemDetail = &p.Code, &p.Detail
	}
}

// clearIdleToken clears account's Plex token when no waiting or shared row
// of theirs needs it, and answers it for signing out.
func (m *MediaServers) clearIdleToken(ctx context.Context, q *db.Queries, account identity.AccountID) (*clearedToken, error) {
	idle, err := q.IdleMemberToken(ctx, account)
	if errors.Is(err, storage.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	token, err := m.cfg.Sealer.Open(idle.TokenSealed, memberTokenColumn.Row(account.String()))
	if err != nil {
		return nil, fmt.Errorf("open a Member's Plex token: %w", err)
	}
	return &clearedToken{clientID: idle.ClientIdentifier, token: plextv.NewMemberToken(string(token))}, q.ClearMemberToken(ctx, account)
}

// afterMove is a committed move's effects: the cleared token signed out
// and the access events.
func (m *MediaServers) afterMove(ctx context.Context, w moved) {
	if w.token != nil {
		tv := plextv.NewMember(m.cfg.PlexTV, plexHeaders(w.token.clientID), m.cfg.HTTP)
		if err := tv.SignOut(ctx, w.token.token); err != nil {
			m.cfg.Log.Warn("sign a Member's Plex token out", "err", err.Error())
		}
	}
	m.accessChanged(ctx, w.account)
}

// accessChanged tells the Admins, and account's Member when it is set,
// that access rows changed.
func (m *MediaServers) accessChanged(ctx context.Context, account identity.AccountID) {
	m.cfg.Events.AccessChanged.Publish(ctx, AccessChanged{})
	if !account.IsZero() {
		m.cfg.Events.MyAccessChanged.Publish(ctx, account.UUID, MyAccessChanged{})
	}
}

// newRow is a row not yet offered: account's access to srv by way.
func newRow(srv db.MediaserversServer, way access.Way, account identity.AccountID, name string, by identity.AccountID) accessRow {
	return accessRow{
		MediaserversMemberAccess: db.MediaserversMemberAccess{
			ID: AccessID{UUID: storage.NewUUID()}, ServerID: srv.ID, Way: way, AccountID: account, MemberName: name, OfferedBy: by,
		},
		server: srv,
	}
}
