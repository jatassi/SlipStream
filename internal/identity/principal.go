package identity

import (
	"errors"
	"net/netip"

	"github.com/jatassi/slipstream/internal/identity/passkey"
)

// client is who a request came from, after http.trusted_proxies.
type client struct {
	address   netip.Addr
	https     bool
	host      string
	userAgent string
}

func (c client) Address() netip.Addr { return c.address }

// SuggestedAddress is what Settings > Address offers this caller.
func (c client) SuggestedAddress() string { return suggestAddress(c) }

// Passkeys is whether the address c reached SlipStream at can hold a
// passkey, and why not.
func (c client) Passkeys() passkey.Verdict {
	_, v := c.relyingParty()
	return v
}

func (c client) relyingParty() (passkey.RP, passkey.Verdict) { return passkey.Decide(c.host, c.https) }

// Anyone is whoever called a public operation.
type Anyone struct{ client }

// Claim is a caller the Setup gate admitted while the Node is Unclaimed.
type Claim struct{ client }

type signedIn struct {
	profile     Profile
	session     SessionID
	confirmed   bool
	pinRequired bool
	keyPrompt   KeyPrompt
}

// Bearer is any signed-in session, one that must set a new PIN included.
// Only setting the PIN, reading the session and Sign out take one.
type Bearer struct {
	client
	s signedIn
}

func (b Bearer) Profile() Profile     { return b.s.profile }
func (b Bearer) Session() SessionID   { return b.s.session }
func (b Bearer) PINRequired() bool    { return b.s.pinRequired }
func (b Bearer) KeyPrompt() KeyPrompt { return b.s.keyPrompt }

// Account is a signed-in session, an Admin's or a Member's, that has a PIN.
type Account struct {
	client
	s signedIn
}

func (a Account) Profile() Profile   { return a.s.profile }
func (a Account) Session() SessionID { return a.s.session }

func (a Account) AsAdmin() (Admin, bool) {
	if !a.s.profile.Admin {
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

func (a Admin) session() SessionID {
	if a.account == nil {
		return SessionID{}
	}
	return a.account.s.session
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
	kind callerKind
	s    signedIn
}

var (
	ErrSignedOut      = errors.New("identity: no valid session and no API key")
	ErrAPIKeyWrong    = errors.New("identity: the X-Api-Key matches nothing")
	ErrNeedsAccount   = errors.New("identity: the API key has no Account")
	ErrNeedsAdmin     = errors.New("identity: a Member is not an Admin")
	ErrSetupCodeWrong = errors.New("identity: the Setup code is missing or wrong")
	ErrSetupBusy      = errors.New("identity: another setup operation is running")
	ErrAlreadyClaimed = errors.New("identity: the Node is claimed")
	ErrWorkerNode     = errors.New("identity: this Node is a Worker, with no Accounts")
	ErrPINWrong       = errors.New("identity: the email or PIN is wrong")
	ErrPINRequired    = errors.New("identity: the session must set a new PIN first")
	ErrNeedsReconfirm = errors.New("identity: the session's last confirm is older than the window")
)

func (c Caller) Anyone() Anyone { return Anyone{c.client} }

func (c Caller) Bearer() (Bearer, error) {
	switch c.kind {
	case nobody:
		return Bearer{}, ErrSignedOut
	case sessionCaller:
		return Bearer{client: c.client, s: c.s}, nil
	case apiKeyCaller:
		return Bearer{}, ErrNeedsAccount
	case wrongAPIKey:
		return Bearer{}, ErrAPIKeyWrong
	}
	panic("identity: unknown caller kind")
}

func (c Caller) Account() (Account, error) {
	if _, err := c.Bearer(); err != nil {
		return Account{}, err
	}
	if c.s.pinRequired {
		return Account{}, ErrPINRequired
	}
	return Account{client: c.client, s: c.s}, nil
}

// Confirmed is an Account whose session confirmed within the re-confirm
// window.
func (c Caller) Confirmed() (Account, error) {
	a, err := c.Account()
	if err != nil {
		return Account{}, err
	}
	if !c.s.confirmed {
		return Account{}, ErrNeedsReconfirm
	}
	return a, nil
}

func (c Caller) Admin() (Admin, error) {
	switch c.kind {
	case nobody:
		return Admin{}, ErrSignedOut
	case sessionCaller:
		a, err := c.Account()
		if err != nil {
			return Admin{}, err
		}
		if admin, ok := a.AsAdmin(); ok {
			return admin, nil
		}
		return Admin{}, ErrNeedsAdmin
	case apiKeyCaller:
		return Admin{client: c.client}, nil
	case wrongAPIKey:
		return Admin{}, ErrAPIKeyWrong
	}
	panic("identity: unknown caller kind")
}

// Reconfirmed is an Admin whose session confirmed within the re-confirm
// window, or the API key.
func (c Caller) Reconfirmed() (Admin, error) {
	a, err := c.Admin()
	if err != nil {
		return Admin{}, err
	}
	if c.kind == sessionCaller && !c.s.confirmed {
		return Admin{}, ErrNeedsReconfirm
	}
	return a, nil
}
