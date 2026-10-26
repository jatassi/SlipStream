package identityapi

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
)

type pinState struct {
	State string     `json:"state" enum:"fine,waiting,off"`
	Until *time.Time `json:"until,omitempty" doc:"When the wait ends, while it waits"`
}

type accountRow struct {
	ID           string     `json:"id" format:"uuid"`
	Email        string     `json:"email" format:"email" maxLength:"254"`
	DisplayName  string     `json:"displayName" minLength:"1" maxLength:"100"`
	Admin        bool       `json:"admin"`
	Disabled     bool       `json:"disabled" doc:"Turned off: it can't sign in and holds no session"`
	PasskeyCount int        `json:"passkeyCount"`
	LastSignInAt *time.Time `json:"lastSignInAt,omitempty" doc:"Absent until it first signs in"`
	PIN          pinState   `json:"pin"`
}

func timeOrNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func accountRowOf(r identity.AccountRow) accountRow {
	return accountRow{
		ID: r.ID.String(), Email: r.Email, DisplayName: r.DisplayName, Admin: r.Admin, Disabled: r.Disabled,
		PasskeyCount: r.PasskeyCount, LastSignInAt: timeOrNil(r.LastSignInAt),
		PIN: pinState{State: string(r.PIN.Kind), Until: timeOrNil(r.PIN.Until)},
	}
}

type sessionRow struct {
	ID         string    `json:"id" format:"uuid"`
	Device     string    `json:"device" doc:"Such as Safari on iPhone, from the browser at sign-in"`
	Address    string    `json:"address" doc:"The client address it signed in from"`
	CreatedAt  time.Time `json:"createdAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
	Current    bool      `json:"current" doc:"The session making this request"`
}

func sessionRowsOf(sessions []identity.Session) sessionRows {
	out := sessionRows{Sessions: make([]sessionRow, len(sessions))}
	for i, s := range sessions {
		out.Sessions[i] = sessionRow{
			ID: s.ID.String(), Device: s.Device, Address: s.Address,
			CreatedAt: s.CreatedAt, LastSeenAt: s.LastSeenAt, Current: s.Current,
		}
	}
	return out
}

type accountRows struct {
	Accounts []accountRow `json:"accounts" nullable:"false"`
}

type sessionRows struct {
	Sessions []sessionRow `json:"sessions" nullable:"false"`
}

type accountDetail struct {
	Account  accountRow   `json:"account"`
	Passkeys []passkey    `json:"passkeys" nullable:"false"`
	Sessions []sessionRow `json:"sessions" nullable:"false"`
}

type accountIDInput struct {
	ID string `path:"id" format:"uuid"`
}

type accountSessionInput struct {
	ID      string `path:"id" format:"uuid"`
	Session string `path:"session" format:"uuid"`
}

type sessionIDInput struct {
	ID string `path:"id" format:"uuid"`
}

type otherSessionsInput struct {
	Others bool `query:"others" required:"true" doc:"Must be true: every session but this one ends"`
}

type accountPatchInput struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		DisplayName *string `json:"displayName,omitempty" minLength:"1" maxLength:"100"`
		Email       *string `json:"email,omitempty" format:"email" maxLength:"254" doc:"Changed at once. SlipStream emails the old address about the change"`
		Admin       *bool   `json:"admin,omitempty"`
		Disabled    *bool   `json:"disabled,omitempty" doc:"Turning it off ends every session it holds"`
	}
}

type ownPatchInput struct {
	Body struct {
		DisplayName *string `json:"displayName,omitempty" minLength:"1" maxLength:"100"`
		KeyPrompt   *string `json:"keyPrompt,omitempty" enum:"ask,never" doc:"Whether the prompt to add a passkey may show after sign-in"`
	}
}

type (
	accountRowsOutput   struct{ Body accountRows }
	accountRowOutput    struct{ Body accountRow }
	accountDetailOutput struct{ Body accountDetail }
	sessionRowsOutput   struct{ Body sessionRows }
)

func sessionID(s string) (identity.SessionID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return identity.SessionID{}, sessionNotFound
	}
	return identity.SessionID{UUID: u}, nil
}

func listAccounts(lib Library) func(context.Context, identity.Admin, *struct{}) (*accountRowsOutput, error) {
	return func(ctx context.Context, a identity.Admin, _ *struct{}) (*accountRowsOutput, error) {
		rows, err := open(lib).Accounts(ctx, a)
		if err != nil {
			return nil, err
		}
		out := accountRows{Accounts: make([]accountRow, len(rows))}
		for i, r := range rows {
			out.Accounts[i] = accountRowOf(r)
		}
		return &accountRowsOutput{Body: out}, nil
	}
}

func getAccount(lib Library) func(context.Context, identity.Admin, *accountIDInput) (*accountDetailOutput, error) {
	return func(ctx context.Context, a identity.Admin, in *accountIDInput) (*accountDetailOutput, error) {
		id, err := accountID(in.ID)
		if err != nil {
			return nil, err
		}
		d, err := open(lib).AccountOf(ctx, a, id)
		if err != nil {
			return nil, answer(err)
		}
		return &accountDetailOutput{Body: accountDetail{Account: accountRowOf(d.AccountRow), Passkeys: passkeysOf(d.Passkeys), Sessions: sessionRowsOf(d.Sessions).Sessions}}, nil
	}
}

func updateAccount(lib Library) func(context.Context, identity.Admin, *accountPatchInput) (*accountRowOutput, error) {
	return func(ctx context.Context, a identity.Admin, in *accountPatchInput) (*accountRowOutput, error) {
		id, err := accountID(in.ID)
		if err != nil {
			return nil, err
		}
		b := in.Body
		row, err := open(lib).UpdateAccount(ctx, a, id, identity.AccountPatch{DisplayName: b.DisplayName, Email: b.Email, Admin: b.Admin, Disabled: b.Disabled})
		if err != nil {
			return nil, answer(err)
		}
		return &accountRowOutput{Body: accountRowOf(row)}, nil
	}
}

func deleteAccount(lib Library) func(context.Context, identity.Admin, *accountIDInput) (*struct{}, error) {
	return func(ctx context.Context, a identity.Admin, in *accountIDInput) (*struct{}, error) {
		id, err := accountID(in.ID)
		if err != nil {
			return nil, err
		}
		return nil, answer(open(lib).DeleteAccount(ctx, a, id))
	}
}

func listAccountSessions(lib Library) func(context.Context, identity.Admin, *accountIDInput) (*sessionRowsOutput, error) {
	return func(ctx context.Context, a identity.Admin, in *accountIDInput) (*sessionRowsOutput, error) {
		id, err := accountID(in.ID)
		if err != nil {
			return nil, err
		}
		sessions, err := open(lib).AccountSessions(ctx, a, id)
		if err != nil {
			return nil, answer(err)
		}
		return &sessionRowsOutput{Body: sessionRowsOf(sessions)}, nil
	}
}

func endAccountSessions(lib Library) func(context.Context, identity.Admin, *accountIDInput) (*struct{}, error) {
	return func(ctx context.Context, a identity.Admin, in *accountIDInput) (*struct{}, error) {
		id, err := accountID(in.ID)
		if err != nil {
			return nil, err
		}
		return nil, answer(open(lib).EndAccountSessions(ctx, a, id))
	}
}

func endAccountSession(lib Library) func(context.Context, identity.Admin, *accountSessionInput) (*struct{}, error) {
	return func(ctx context.Context, a identity.Admin, in *accountSessionInput) (*struct{}, error) {
		id, err := accountID(in.ID)
		if err != nil {
			return nil, err
		}
		session, err := sessionID(in.Session)
		if err != nil {
			return nil, err
		}
		return nil, answer(open(lib).EndAccountSession(ctx, a, id, session))
	}
}

func listSessions(lib Library) func(context.Context, identity.Account, *struct{}) (*sessionRowsOutput, error) {
	return func(ctx context.Context, a identity.Account, _ *struct{}) (*sessionRowsOutput, error) {
		sessions, err := open(lib).Sessions(ctx, a)
		if err != nil {
			return nil, err
		}
		return &sessionRowsOutput{Body: sessionRowsOf(sessions)}, nil
	}
}

func endSession(lib Library) func(context.Context, identity.Account, *sessionIDInput) (*struct{}, error) {
	return func(ctx context.Context, a identity.Account, in *sessionIDInput) (*struct{}, error) {
		session, err := sessionID(in.ID)
		if err != nil {
			return nil, err
		}
		return nil, answer(open(lib).EndSession(ctx, a, session))
	}
}

func endOtherSessions(lib Library) func(context.Context, identity.Account, *otherSessionsInput) (*struct{}, error) {
	return func(ctx context.Context, a identity.Account, in *otherSessionsInput) (*struct{}, error) {
		if !in.Others {
			return nil, api.Invalid.Fields(api.FieldError{Location: "query.others", Message: "expected true"})
		}
		return nil, open(lib).EndOtherSessions(ctx, a)
	}
}

func updateOwnAccount(lib Library) func(context.Context, identity.Account, *ownPatchInput) (*getSessionOutput, error) {
	return func(ctx context.Context, a identity.Account, in *ownPatchInput) (*getSessionOutput, error) {
		p := identity.OwnPatch{DisplayName: in.Body.DisplayName}
		if in.Body.KeyPrompt != nil {
			k := identity.KeyPrompt(*in.Body.KeyPrompt)
			p.KeyPrompt = &k
		}
		profile, prompt, err := open(lib).UpdateOwnAccount(ctx, a, p)
		if err != nil {
			return nil, err
		}
		s, err := withFirstRun(ctx, lib, Session{Account: accountOf(profile), KeyPrompt: string(prompt)})
		if err != nil {
			return nil, err
		}
		return &getSessionOutput{Body: s}, nil
	}
}
