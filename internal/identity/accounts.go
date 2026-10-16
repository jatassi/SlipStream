package identity

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/identity/internal/db"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/storage"
)

var (
	ErrLastAdmin = errors.New("identity: no Admin who is turned on would remain")
	ErrNoAccount = errors.New("identity: no Account has that id")
	ErrNoSession = errors.New("identity: the Account has no session with that id")
)

type KeyPrompt = db.KeyPrompt

const (
	KeyPromptAsk   KeyPrompt = "ask"
	KeyPromptNever KeyPrompt = "never"
)

type PINStateKind string

const (
	PINFine      PINStateKind = "fine"
	PINWaiting   PINStateKind = "waiting"
	PINTurnedOff PINStateKind = "off"
)

// PINState is what an Admin sees of an Account's PIN. Until is set while
// it waits.
type PINState struct {
	Kind  PINStateKind
	Until time.Time
}

func pinStateOf(row db.IdentityAccount, now time.Time) PINState {
	switch l := ledgerOf(row); {
	case !l.offAt.IsZero():
		return PINState{Kind: PINTurnedOff}
	case now.Before(l.waitUntil):
		return PINState{Kind: PINWaiting, Until: l.waitUntil}
	}
	return PINState{Kind: PINFine}
}

// AccountRow is an Account as the Accounts list shows it. LastSignInAt is
// zero for an Account that never signed in.
type AccountRow struct {
	Profile
	Disabled     bool
	PasskeyCount int
	LastSignInAt time.Time
	PIN          PINState
}

func rowOf(row db.IdentityAccount, passkeys int, now time.Time) AccountRow {
	return AccountRow{
		Profile:      profileOf(row),
		Disabled:     !row.DisabledAt.IsZero(),
		PasskeyCount: passkeys,
		LastSignInAt: row.LastSignInAt.Time(),
		PIN:          pinStateOf(row, now),
	}
}

// Session is one signed-in device. Current is whether it is the caller's.
type Session struct {
	ID         SessionID
	Device     string
	Address    string
	CreatedAt  time.Time
	LastSeenAt time.Time
	Current    bool
}

type AccountDetail struct {
	AccountRow
	Passkeys []Passkey
	Sessions []Session
}

func (s *Service) Accounts(ctx context.Context, _ Admin) ([]AccountRow, error) {
	rows, err := db.New(s.lib).ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := make([]AccountRow, len(rows))
	for i, r := range rows {
		out[i] = rowOf(accountOfRow(r), int(r.PasskeyCount), now)
	}
	return out, nil
}

func accountOfRow(r db.ListAccountsRow) db.IdentityAccount {
	return db.IdentityAccount{
		ID: r.ID, Email: r.Email, DisplayName: r.DisplayName, Admin: r.Admin, PinHash: r.PinHash, CreatedAt: r.CreatedAt,
		DisabledAt: r.DisabledAt, PinFailures: r.PinFailures, PinWaitUntil: r.PinWaitUntil, PinDisabledAt: r.PinDisabledAt,
		KeyPrompt: r.KeyPrompt, LastSignInAt: r.LastSignInAt,
	}
}

func (s *Service) AccountOf(ctx context.Context, by Admin, id AccountID) (AccountDetail, error) {
	q := db.New(s.lib)
	row, err := getAccount(ctx, q, id)
	if err != nil {
		return AccountDetail{}, err
	}
	keys, err := q.ListPasskeys(ctx, id)
	if err != nil {
		return AccountDetail{}, err
	}
	sessions, err := sessionsOf(ctx, q, id, by.session())
	if err != nil {
		return AccountDetail{}, err
	}
	return AccountDetail{AccountRow: rowOf(row, len(keys), time.Now()), Passkeys: passkeysOf(keys), Sessions: sessions}, nil
}

// AccountPatch is an Admin's change to an Account; a nil field stays.
type AccountPatch struct {
	DisplayName *string
	Email       *string
	Admin       *bool
	Disabled    *bool
}

// UpdateAccount applies p in one Write. A new email is set directly and the
// old address is told. A changed admin flag re-gates the Account's tabs.
// Turning it off ends every session it holds.
func (s *Service) UpdateAccount(ctx context.Context, _ Admin, id AccountID, p AccountPatch) (AccountRow, error) {
	var out AccountRow
	err := s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
		row, err := getAccount(ctx, q, id)
		if err != nil {
			return err
		}
		for _, change := range []func(context.Context, *db.Queries, *effects, db.IdentityAccount, AccountPatch) error{
			renameAccount, changeAccountEmail, changeRole, changeDisabled,
		} {
			if err := change(ctx, q, fx, row, p); err != nil {
				return err
			}
		}
		if row, err = getAccount(ctx, q, id); err != nil {
			return err
		}
		keys, err := q.ListPasskeys(ctx, id)
		out = rowOf(row, len(keys), fx.now.Time())
		return err
	})
	return out, err
}

func renameAccount(ctx context.Context, q *db.Queries, fx *effects, row db.IdentityAccount, p AccountPatch) error {
	if p.DisplayName == nil || *p.DisplayName == row.DisplayName {
		return nil
	}
	fx.accountsChanged()
	fx.sessionsChanged(row.ID)
	return q.SetDisplayName(ctx, db.SetDisplayNameParams{DisplayName: *p.DisplayName, ID: row.ID})
}

func changeAccountEmail(ctx context.Context, q *db.Queries, fx *effects, row db.IdentityAccount, p AccountPatch) error {
	if p.Email == nil {
		return nil
	}
	return changeEmail(ctx, q, fx, row, normalizeEmail(*p.Email))
}

func changeRole(ctx context.Context, q *db.Queries, fx *effects, row db.IdentityAccount, p AccountPatch) error {
	if p.Admin == nil || *p.Admin == row.Admin {
		return nil
	}
	fx.regate(row.ID)
	return q.SetAdmin(ctx, db.SetAdminParams{Admin: *p.Admin, ID: row.ID})
}

func changeDisabled(ctx context.Context, q *db.Queries, fx *effects, row db.IdentityAccount, p AccountPatch) error {
	off := p.Disabled != nil && *p.Disabled
	if p.Disabled == nil || off == !row.DisabledAt.IsZero() {
		return nil
	}
	var at storage.Time
	if off {
		at = fx.now
	}
	if err := q.SetDisabledAt(ctx, db.SetDisabledAtParams{DisabledAt: at, ID: row.ID}); err != nil {
		return err
	}
	fx.accountsChanged()
	if !off {
		return nil
	}
	return endAllSessions(ctx, q, fx, row.ID)
}

// DeleteAccount deletes id, and through the foreign keys its sessions,
// passkeys and links. Its streams end, revoked.
func (s *Service) DeleteAccount(ctx context.Context, _ Admin, id AccountID) error {
	return s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
		if _, err := getAccount(ctx, q, id); err != nil {
			return err
		}
		ids, err := q.ListSessionIDs(ctx, id)
		if err != nil {
			return err
		}
		fx.endSessions(id, ids, live.Revoked)
		fx.accountsChanged()
		return q.DeleteAccount(ctx, id)
	})
}

func sessionsOf(ctx context.Context, q *db.Queries, id AccountID, current SessionID) ([]Session, error) {
	rows, err := q.ListSessions(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Session, len(rows))
	for i, r := range rows {
		out[i] = Session{
			ID: r.ID, Device: r.Device, Address: r.Address,
			CreatedAt: r.CreatedAt.Time(), LastSeenAt: r.LastSeenAt.Time(), Current: r.ID == current,
		}
	}
	return out, nil
}

func (s *Service) AccountSessions(ctx context.Context, by Admin, id AccountID) ([]Session, error) {
	q := db.New(s.lib)
	if _, err := getAccount(ctx, q, id); err != nil {
		return nil, err
	}
	return sessionsOf(ctx, q, id, by.session())
}

// EndAccountSessions is an Admin's Sign out everywhere for id.
func (s *Service) EndAccountSessions(ctx context.Context, _ Admin, id AccountID) error {
	return s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
		if _, err := getAccount(ctx, q, id); err != nil {
			return err
		}
		return endAllSessions(ctx, q, fx, id)
	})
}

func (s *Service) EndAccountSession(ctx context.Context, _ Admin, id AccountID, session SessionID) error {
	return s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
		if _, err := getAccount(ctx, q, id); err != nil {
			return err
		}
		return endOneSession(ctx, q, fx, id, session, live.Revoked)
	})
}

func endOneSession(ctx context.Context, q *db.Queries, fx *effects, id AccountID, session SessionID, why live.EndReason) error {
	n, err := q.DeleteAccountSession(ctx, db.DeleteAccountSessionParams{AccountID: id, ID: session})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNoSession
	}
	fx.endSession(id, session, why)
	return nil
}

func (s *Service) Sessions(ctx context.Context, a Account) ([]Session, error) {
	return sessionsOf(ctx, db.New(s.lib), a.s.profile.ID, a.s.session)
}

// EndSession ends one of a's sessions: signed out when it is a's own,
// revoked on the device it signed in.
func (s *Service) EndSession(ctx context.Context, a Account, session SessionID) error {
	why := live.Revoked
	if session == a.s.session {
		why = live.SignedOut
	}
	return s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
		return endOneSession(ctx, q, fx, a.s.profile.ID, session, why)
	})
}

// EndOtherSessions ends every session of a's Account but a's own.
func (s *Service) EndOtherSessions(ctx context.Context, a Account) error {
	me := a.s.profile.ID
	return s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
		ids, err := q.ListSessionIDs(ctx, me)
		if err != nil {
			return err
		}
		for _, id := range ids {
			if id != a.s.session {
				fx.endSession(me, id, live.Revoked)
			}
		}
		return q.DeleteOtherSessions(ctx, db.DeleteOtherSessionsParams{AccountID: me, Keep: a.s.session})
	})
}

// OwnPatch is a change to one's own Account; a nil field stays.
type OwnPatch struct {
	DisplayName *string
	KeyPrompt   *KeyPrompt
}

func (s *Service) UpdateOwnAccount(ctx context.Context, a Account, p OwnPatch) (Profile, KeyPrompt, error) {
	me := a.s.profile.ID
	var row db.IdentityAccount
	err := s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
		if p.DisplayName != nil {
			if err := q.SetDisplayName(ctx, db.SetDisplayNameParams{DisplayName: *p.DisplayName, ID: me}); err != nil {
				return err
			}
			fx.accountsChanged()
		}
		if p.KeyPrompt != nil {
			if err := q.SetKeyPrompt(ctx, db.SetKeyPromptParams{KeyPrompt: *p.KeyPrompt, ID: me}); err != nil {
				return err
			}
		}
		fx.sessionsChanged(me)
		var err error
		row, err = getAccount(ctx, q, me)
		return err
	})
	return profileOf(row), row.KeyPrompt, err
}

func getAccount(ctx context.Context, q *db.Queries, id AccountID) (db.IdentityAccount, error) {
	row, err := q.GetAccount(ctx, id)
	if errors.Is(err, storage.ErrNoRows) {
		return db.IdentityAccount{}, ErrNoAccount
	}
	return row, err
}

// emailFree refuses ErrEmailTaken when an Account other than self has
// email.
func emailFree(ctx context.Context, q *db.Queries, email string, self AccountID) error {
	row, err := q.GetAccountByEmail(ctx, email)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return nil
	case err != nil:
		return err
	case row.ID != self:
		return ErrEmailTaken
	}
	return nil
}
