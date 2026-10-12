package identity

import (
	"errors"
	"net/netip"
)

// client is the connection a request arrived on. The Relay Node later
// supplies the original client address for relayed requests.
type client struct {
	address netip.Addr
	tls     bool
}

func (c client) Address() netip.Addr { return c.address }

// Anyone is whoever called a public operation.
type Anyone struct{ client }

// Claim is a caller the Setup gate admitted while the Node is Unclaimed.
type Claim struct{ client }

// Account is a signed-in session, an Admin's or a Member's.
type Account struct {
	client
	profile Profile
	session SessionID
}

func (a Account) Profile() Profile   { return a.profile }
func (a Account) Session() SessionID { return a.session }

// AsAdmin narrows a to an Admin. It fails for a Member.
func (a Account) AsAdmin() (Admin, bool) {
	if !a.profile.Admin {
		return Admin{}, false
	}
	return Admin{client: a.client, account: &a}, true
}

// Admin is an Admin session or the API key.
type Admin struct {
	client
	account *Account
}

// Account is the Admin's session, or false when the API key called.
func (a Admin) Account() (Account, bool) {
	if a.account == nil {
		return Account{}, false
	}
	return *a.account, true
}

// Profile is an Account as its owner and the API see it.
type Profile struct {
	ID          AccountID
	Email       string
	DisplayName string
	Admin       bool
}

type callerKind uint8

const (
	nobody callerKind = iota
	sessionCaller
	apiKeyCaller
	wrongAPIKey
)

// Caller is a request's one resolved authentication: nobody, a session's
// Account, the API key or an X-Api-Key that matches nothing. Each access
// level narrows it to its principal.
type Caller struct {
	client
	kind    callerKind
	account Account
}

var (
	ErrSignedOut      = errors.New("identity: no valid session and no API key")
	ErrAPIKeyWrong    = errors.New("identity: the X-Api-Key matches nothing")
	ErrNeedsAccount   = errors.New("identity: the API key has no Account")
	ErrNeedsAdmin     = errors.New("identity: a Member is not an Admin")
	ErrSetupCodeWrong = errors.New("identity: the Setup code is missing or wrong")
	ErrSetupBusy      = errors.New("identity: another setup operation is running")
	ErrAlreadyClaimed = errors.New("identity: the Node is claimed")
	ErrPINWrong       = errors.New("identity: the email or PIN is wrong")
)

func (c Caller) Anyone() Anyone { return Anyone{c.client} }

func (c Caller) Account() (Account, error) {
	switch c.kind {
	case nobody:
		return Account{}, ErrSignedOut
	case sessionCaller:
		return c.account, nil
	case apiKeyCaller:
		return Account{}, ErrNeedsAccount
	case wrongAPIKey:
		return Account{}, ErrAPIKeyWrong
	}
	panic("identity: unknown caller kind")
}

func (c Caller) Admin() (Admin, error) {
	switch c.kind {
	case nobody:
		return Admin{}, ErrSignedOut
	case sessionCaller:
		if a, ok := c.account.AsAdmin(); ok {
			return a, nil
		}
		return Admin{}, ErrNeedsAdmin
	case apiKeyCaller:
		return Admin{client: c.client}, nil
	case wrongAPIKey:
		return Admin{}, ErrAPIKeyWrong
	}
	panic("identity: unknown caller kind")
}
