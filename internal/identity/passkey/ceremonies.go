package passkey

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// CeremonyID names a started ceremony; each finish carries it.
type CeremonyID string

type kind uint8

const (
	registration kind = iota
	signIn
	reconfirm
)

// Lifetime is how long a started ceremony waits for its finish.
const Lifetime = 5 * time.Minute

// capacity bounds the ceremonies held, since sign-in begins are Public.
// Past it the oldest is dropped.
const capacity = 1024

var (
	// ErrCeremony is a finish for no live ceremony: unknown, already
	// finished, expired, or begun for another purpose, address or user.
	ErrCeremony = errors.New("passkey: no such ceremony")
	// ErrFailed is an authenticator answer that did not verify, or a sign
	// count that went backwards (a cloned authenticator).
	ErrFailed = errors.New("passkey: the passkey couldn't be verified")
	// ErrNoCredential is a re-confirm for a user with no passkey here.
	ErrNoCredential = errors.New("passkey: no passkey for this address")
)

// Ceremonies holds each started ceremony's challenge in memory for
// Lifetime, keyed by its id and bound to its kind, its RP and, except for
// sign-in, its user. A finish takes it out whatever the outcome.
type Ceremonies struct {
	mu   sync.Mutex
	m    map[CeremonyID]pending
	next uint64
}

type pending struct {
	kind    kind
	rp      RP
	handle  []byte
	session webauthn.SessionData
	expires time.Time
	// seq orders ceremonies by start. Their expiries can tie, since
	// Windows' clock ticks too coarsely to tell apart quick begins.
	seq uint64
}

func NewCeremonies() *Ceremonies {
	return &Ceremonies{m: map[CeremonyID]pending{}}
}

// BeginRegistration starts adding a discoverable passkey for u, excluding
// the ones u already has here. The options are
// PublicKeyCredentialCreationOptionsJSON.
func (c *Ceremonies) BeginRegistration(rp RP, u User) (CeremonyID, json.RawMessage, error) {
	w, err := relyingParty(rp)
	if err != nil {
		return "", nil, err
	}
	creation, session, err := w.BeginRegistration(webUser{u},
		webauthn.WithExclusions(webCredentials(u.Credentials).CredentialDescriptors()),
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			RequireResidentKey: protocol.ResidentKeyRequired(),
			ResidentKey:        protocol.ResidentKeyRequirementRequired,
			UserVerification:   protocol.VerificationRequired,
		}))
	if err != nil {
		return "", nil, fmt.Errorf("passkey: begin registration: %w", err)
	}
	return c.start(registration, rp, u.Handle, *session, creation.Response)
}

// FinishRegistration verifies the authenticator's answer to the
// registration id began for u at rp and answers the new credential.
func (c *Ceremonies) FinishRegistration(rp RP, id CeremonyID, u User, response []byte) (Credential, error) {
	p, err := c.take(id, registration, rp, u.Handle)
	if err != nil {
		return Credential{}, err
	}
	w, err := relyingParty(rp)
	if err != nil {
		return Credential{}, err
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes(response)
	if err != nil {
		return Credential{}, fmt.Errorf("%w: %w", ErrFailed, err)
	}
	cred, err := w.CreateCredential(webUser{u}, p.session, parsed)
	if err != nil {
		return Credential{}, fmt.Errorf("%w: %w", ErrFailed, err)
	}
	return credentialOf(cred), nil
}

// BeginSignIn starts a discoverable assertion: no allowCredentials, so the
// browser offers any passkey it holds for rp (conditional mediation or the
// button). The options are PublicKeyCredentialRequestOptionsJSON.
func (c *Ceremonies) BeginSignIn(rp RP) (CeremonyID, json.RawMessage, error) {
	w, err := relyingParty(rp)
	if err != nil {
		return "", nil, err
	}
	assertion, session, err := w.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		return "", nil, fmt.Errorf("passkey: begin sign-in: %w", err)
	}
	return c.start(signIn, rp, nil, *session, assertion.Response)
}

// FinishSignIn verifies a sign-in assertion. find answers the user whose
// handle the passkey carries, with its credentials for rp, or false when
// there is none.
func (c *Ceremonies) FinishSignIn(rp RP, id CeremonyID, response []byte, find func(handle []byte) (User, bool, error)) (Asserted, error) {
	p, err := c.take(id, signIn, rp, nil)
	if err != nil {
		return Asserted{}, err
	}
	w, parsed, err := parseAssertion(rp, response)
	if err != nil {
		return Asserted{}, err
	}
	u, ok, err := find(parsed.Response.UserHandle)
	if err != nil {
		return Asserted{}, err
	}
	if !ok {
		return Asserted{}, fmt.Errorf("%w: no user for the passkey's handle", ErrFailed)
	}
	_, cred, err := w.ValidatePasskeyLogin(func([]byte, []byte) (webauthn.User, error) { return webUser{u}, nil }, p.session, parsed)
	return asserted(u, cred, err)
}

// BeginReconfirm starts an assertion limited to u's passkeys for rp.
func (c *Ceremonies) BeginReconfirm(rp RP, u User) (CeremonyID, json.RawMessage, error) {
	if len(u.Credentials) == 0 {
		return "", nil, ErrNoCredential
	}
	w, err := relyingParty(rp)
	if err != nil {
		return "", nil, err
	}
	assertion, session, err := w.BeginLogin(webUser{u}, webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		return "", nil, fmt.Errorf("passkey: begin re-confirm: %w", err)
	}
	return c.start(reconfirm, rp, u.Handle, *session, assertion.Response)
}

func (c *Ceremonies) FinishReconfirm(rp RP, id CeremonyID, u User, response []byte) (Asserted, error) {
	p, err := c.take(id, reconfirm, rp, u.Handle)
	if err != nil {
		return Asserted{}, err
	}
	w, parsed, err := parseAssertion(rp, response)
	if err != nil {
		return Asserted{}, err
	}
	cred, err := w.ValidateLogin(webUser{u}, p.session, parsed)
	return asserted(u, cred, err)
}

func parseAssertion(rp RP, response []byte) (*webauthn.WebAuthn, *protocol.ParsedCredentialAssertionData, error) {
	w, err := relyingParty(rp)
	if err != nil {
		return nil, nil, err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(response)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", ErrFailed, err)
	}
	return w, parsed, nil
}

func asserted(u User, cred *webauthn.Credential, err error) (Asserted, error) {
	if err != nil {
		return Asserted{}, fmt.Errorf("%w: %w", ErrFailed, err)
	}
	if cred.Authenticator.CloneWarning {
		return Asserted{}, fmt.Errorf("%w: the sign count did not advance", ErrFailed)
	}
	return Asserted{Handle: u.Handle, Credential: credentialOf(cred)}, nil
}

// relyingParty builds go-webauthn's relying party for one ceremony at rp,
// with MDS unset: attestation is never checked against FIDO metadata.
func relyingParty(rp RP) (*webauthn.WebAuthn, error) {
	w, err := webauthn.New(&webauthn.Config{
		RPID:          rp.id,
		RPDisplayName: rp.id,
		RPOrigins:     []string{rp.origin},
	})
	if err != nil {
		return nil, fmt.Errorf("passkey: relying party %q: %w", rp.origin, err)
	}
	return w, nil
}

func (c *Ceremonies) start(k kind, rp RP, handle []byte, session webauthn.SessionData, options any) (CeremonyID, json.RawMessage, error) {
	raw, err := json.Marshal(options)
	if err != nil {
		return "", nil, fmt.Errorf("passkey: encode options: %w", err)
	}
	id := CeremonyID(rand.Text())
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.m) >= capacity {
		c.evict(now)
	}
	c.next++
	c.m[id] = pending{kind: k, rp: rp, handle: handle, session: session, expires: now.Add(Lifetime), seq: c.next}
	return id, raw, nil
}

// evict runs under mu.
func (c *Ceremonies) evict(now time.Time) {
	var oldest CeremonyID
	var first uint64
	for id, p := range c.m {
		if !p.expires.After(now) {
			delete(c.m, id)
			continue
		}
		if first == 0 || p.seq < first {
			oldest, first = id, p.seq
		}
	}
	if len(c.m) >= capacity {
		delete(c.m, oldest)
	}
}

func (c *Ceremonies) take(id CeremonyID, k kind, rp RP, handle []byte) (pending, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	p, ok := c.m[id]
	delete(c.m, id)
	if !ok || p.kind != k || p.rp != rp || !bytes.Equal(p.handle, handle) || !time.Now().Before(p.expires) {
		return pending{}, ErrCeremony
	}
	return p, nil
}

type webUser struct{ User }

func (u webUser) WebAuthnID() []byte          { return u.Handle }
func (u webUser) WebAuthnName() string        { return u.Name }
func (u webUser) WebAuthnDisplayName() string { return u.DisplayName }
func (u webUser) WebAuthnCredentials() []webauthn.Credential {
	return webCredentials(u.Credentials)
}

func webCredentials(cs []Credential) webauthn.Credentials {
	out := make(webauthn.Credentials, len(cs))
	for i, c := range cs {
		flags := protocol.FlagUserPresent | protocol.FlagUserVerified
		if c.BackupEligible {
			flags |= protocol.FlagBackupEligible
		}
		if c.BackupState {
			flags |= protocol.FlagBackupState
		}
		transports := make([]protocol.AuthenticatorTransport, len(c.Transports))
		for j, t := range c.Transports {
			transports[j] = protocol.AuthenticatorTransport(t)
		}
		out[i] = webauthn.Credential{
			ID:            c.ID,
			PublicKey:     c.PublicKey,
			Transport:     transports,
			Flags:         webauthn.NewCredentialFlags(flags),
			Authenticator: webauthn.Authenticator{AAGUID: c.AAGUID[:], SignCount: c.SignCount},
		}
	}
	return out
}

func credentialOf(c *webauthn.Credential) Credential {
	transports := make([]string, len(c.Transport))
	for i, t := range c.Transport {
		transports[i] = string(t)
	}
	out := Credential{
		ID:             c.ID,
		PublicKey:      c.PublicKey,
		SignCount:      c.Authenticator.SignCount,
		Transports:     transports,
		BackupEligible: c.Flags.BackupEligible,
		BackupState:    c.Flags.BackupState,
	}
	copy(out.AAGUID[:], c.Authenticator.AAGUID)
	return out
}
