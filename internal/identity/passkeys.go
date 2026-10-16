package identity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/identity/internal/db"
	"github.com/jatassi/slipstream/internal/identity/passkey"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/storage"
)

type CeremonyID = passkey.CeremonyID

// Ceremony is a started passkey ceremony: the id its finish carries, and
// the options the browser parses.
type Ceremony struct {
	ID      CeremonyID
	Options json.RawMessage
}

// Passkey is one of an Account's passkeys. RelyingParty is the host it
// signs in at.
type Passkey struct {
	ID           PasskeyID
	Name         string
	RelyingParty string
	CreatedAt    time.Time
	LastUsedAt   time.Time
}

// PasskeysChanged reaches an Account's streams when one of its passkeys is
// added, renamed or removed.
type PasskeysChanged struct {
	Account string `json:"account" format:"uuid"`
}

var PasskeysChangedEvent = live.AccountEvent[PasskeysChanged]("identity.passkeys-changed")

var (
	ErrPasskeyUnavailable = errors.New("identity: this address can't hold a passkey, or holds none of the Account's")
	ErrPasskeyFailed      = errors.New("identity: the passkey did not check out")
	ErrNoPasskey          = errors.New("identity: the Account has no passkey with that id")
)

// ceremonyRP is the relying party c's ceremonies run as, refused where c's
// address can't hold a passkey.
func ceremonyRP(c client) (passkey.RP, error) {
	rp, v := c.relyingParty()
	if !v.Available {
		return passkey.RP{}, ErrPasskeyUnavailable
	}
	return rp, nil
}

func failed(err error) error {
	if errors.Is(err, passkey.ErrCeremony) || errors.Is(err, passkey.ErrFailed) {
		return fmt.Errorf("%w: %w", ErrPasskeyFailed, err)
	}
	return err
}

// userAt is p as a ceremony at rp sees it: its passkeys there only.
func userAt(ctx context.Context, q *db.Queries, p Profile, rp passkey.RP) (passkey.User, error) {
	rows, err := q.ListCredentialsAt(ctx, db.ListCredentialsAtParams{AccountID: p.ID, RelyingParty: rp.ID()})
	if err != nil {
		return passkey.User{}, err
	}
	u := passkey.User{Handle: p.ID.Bytes(), Name: p.Email, DisplayName: p.DisplayName, Credentials: make([]passkey.Credential, len(rows))}
	for i, r := range rows {
		u.Credentials[i] = credentialOf(r)
	}
	return u, nil
}

func credentialOf(r db.IdentityPasskey) passkey.Credential {
	c := passkey.Credential{
		ID:             r.CredentialID,
		PublicKey:      r.PublicKey,
		SignCount:      uint32(r.SignCount), //nolint:gosec // stored from a uint32
		BackupEligible: r.BackupEligible,
		BackupState:    r.BackupState,
	}
	copy(c.AAGUID[:], r.Aaguid)
	if r.Transports != "" {
		c.Transports = strings.Split(r.Transports, ",")
	}
	return c
}

// BeginPasskeyRegistration starts adding a passkey for a's Account at the
// address a reached, excluding the ones it already holds there.
func (s *Service) BeginPasskeyRegistration(ctx context.Context, a Account) (Ceremony, error) {
	rp, err := ceremonyRP(a.client)
	if err != nil {
		return Ceremony{}, err
	}
	u, err := userAt(ctx, db.New(s.lib), a.s.profile, rp)
	if err != nil {
		return Ceremony{}, err
	}
	id, options, err := s.ceremonies.BeginRegistration(rp, u)
	return Ceremony{ID: id, Options: options}, err
}

// FinishPasskeyRegistration verifies the authenticator's answer and saves
// the passkey, named for its provider or else for a's device.
func (s *Service) FinishPasskeyRegistration(ctx context.Context, a Account, id CeremonyID, response []byte) (Passkey, error) {
	rp, err := ceremonyRP(a.client)
	if err != nil {
		return Passkey{}, err
	}
	u, err := userAt(ctx, db.New(s.lib), a.s.profile, rp)
	if err != nil {
		return Passkey{}, err
	}
	cred, err := s.ceremonies.FinishRegistration(rp, id, u, response)
	if err != nil {
		return Passkey{}, failed(err)
	}
	name, ok := passkey.ProviderName(cred.AAGUID)
	if !ok {
		name = deviceLabel(a.userAgent)
	}
	key := Passkey{ID: PasskeyID{UUID: storage.NewUUID()}, Name: name, RelyingParty: rp.ID()}
	err = s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
		key.CreatedAt = fx.now.Time()
		fx.passkeysChanged(a.s.profile.ID)
		return q.InsertPasskey(ctx, db.InsertPasskeyParams{
			ID:             key.ID,
			AccountID:      a.s.profile.ID,
			RelyingParty:   key.RelyingParty,
			CredentialID:   cred.ID,
			PublicKey:      cred.PublicKey,
			SignCount:      int64(cred.SignCount),
			Aaguid:         cred.AAGUID[:],
			Transports:     strings.Join(cred.Transports, ","),
			BackupEligible: cred.BackupEligible,
			BackupState:    cred.BackupState,
			Name:           key.Name,
			CreatedAt:      fx.now,
		})
	})
	return key, err
}

func passkeyOf(r db.ListPasskeysRow) Passkey {
	return Passkey{ID: r.ID, Name: r.Name, RelyingParty: r.RelyingParty, CreatedAt: r.CreatedAt.Time(), LastUsedAt: r.LastUsedAt.Time()}
}

func passkeysOf(rows []db.ListPasskeysRow) []Passkey {
	out := make([]Passkey, len(rows))
	for i, r := range rows {
		out[i] = passkeyOf(r)
	}
	return out
}

// Passkeys lists a's passkeys at every address.
func (s *Service) Passkeys(ctx context.Context, a Account) ([]Passkey, error) {
	rows, err := db.New(s.lib).ListPasskeys(ctx, a.s.profile.ID)
	if err != nil {
		return nil, err
	}
	return passkeysOf(rows), nil
}

func (s *Service) RenamePasskey(ctx context.Context, a Account, id PasskeyID, name string) (Passkey, error) {
	var out Passkey
	err := s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
		row, err := q.RenamePasskey(ctx, db.RenamePasskeyParams{Name: name, AccountID: a.s.profile.ID, ID: id})
		if errors.Is(err, storage.ErrNoRows) {
			return ErrNoPasskey
		}
		if err != nil {
			return err
		}
		fx.passkeysChanged(a.s.profile.ID)
		out = passkeyOf(db.ListPasskeysRow(row))
		return nil
	})
	return out, err
}

func (s *Service) DeletePasskey(ctx context.Context, a Account, id PasskeyID) error {
	return s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
		n, err := q.DeletePasskey(ctx, db.DeletePasskeyParams{AccountID: a.s.profile.ID, ID: id})
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrNoPasskey
		}
		fx.passkeysChanged(a.s.profile.ID)
		return nil
	})
}
