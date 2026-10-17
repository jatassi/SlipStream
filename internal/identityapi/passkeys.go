package identityapi

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
)

type passkey struct {
	ID         string     `json:"id" format:"uuid"`
	Name       string     `json:"name"`
	Host       string     `json:"host" doc:"The address's host the passkey signs in at"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
}

func passkeyOf(k identity.Passkey) passkey {
	return passkey{ID: k.ID.String(), Name: k.Name, Host: k.RelyingParty, CreatedAt: k.CreatedAt, LastUsedAt: timeOrNil(k.LastUsedAt)}
}

func passkeysOf(keys []identity.Passkey) []passkey {
	out := make([]passkey, len(keys))
	for i, k := range keys {
		out[i] = passkeyOf(k)
	}
	return out
}

type ceremony struct {
	Ceremony string          `json:"ceremony" doc:"The id the finish carries in its path"`
	Options  json.RawMessage `json:"options" doc:"PublicKeyCredentialCreationOptionsJSON for a registration, PublicKeyCredentialRequestOptionsJSON for an assertion: the browser's parseCreationOptionsFromJSON or parseRequestOptionsFromJSON takes it"`
}

type ceremonyOutput struct{ Body ceremony }

func ceremonyOf(c identity.Ceremony) *ceremonyOutput {
	return &ceremonyOutput{Body: ceremony{Ceremony: string(c.ID), Options: c.Options}}
}

type finishInput struct {
	Ceremony string          `path:"ceremony" maxLength:"64"`
	Body     json.RawMessage `doc:"The browser's PublicKeyCredential, as its toJSON() answers"`
}

type passkeyOutput struct{ Body passkey }

type passkeyRows struct {
	Passkeys []passkey `json:"passkeys" nullable:"false"`
}

type passkeyRowsOutput struct{ Body passkeyRows }

type passkeyIDInput struct {
	ID string `path:"id" format:"uuid"`
}

type passkeyRenameInput struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Name string `json:"name" minLength:"1" maxLength:"100"`
	}
}

func passkeyID(s string) (identity.PasskeyID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return identity.PasskeyID{}, passkeyNotFound
	}
	return identity.PasskeyID{UUID: u}, nil
}

func startPasskeyRegistration(lib Library) func(context.Context, identity.Account, *struct{}) (*ceremonyOutput, error) {
	return func(ctx context.Context, a identity.Account, _ *struct{}) (*ceremonyOutput, error) {
		c, err := open(lib).BeginPasskeyRegistration(ctx, a)
		if err != nil {
			return nil, answer(err)
		}
		return ceremonyOf(c), nil
	}
}

func finishPasskeyRegistration(lib Library) func(context.Context, identity.Account, *finishInput) (*passkeyOutput, error) {
	return func(ctx context.Context, a identity.Account, in *finishInput) (*passkeyOutput, error) {
		k, err := open(lib).FinishPasskeyRegistration(ctx, a, identity.CeremonyID(in.Ceremony), in.Body)
		if err != nil {
			return nil, answer(err)
		}
		return &passkeyOutput{Body: passkeyOf(k)}, nil
	}
}

func listPasskeys(lib Library) func(context.Context, identity.Account, *struct{}) (*passkeyRowsOutput, error) {
	return func(ctx context.Context, a identity.Account, _ *struct{}) (*passkeyRowsOutput, error) {
		keys, err := open(lib).Passkeys(ctx, a)
		if err != nil {
			return nil, err
		}
		return &passkeyRowsOutput{Body: passkeyRows{Passkeys: passkeysOf(keys)}}, nil
	}
}

func renamePasskey(lib Library) func(context.Context, identity.Account, *passkeyRenameInput) (*passkeyOutput, error) {
	return func(ctx context.Context, a identity.Account, in *passkeyRenameInput) (*passkeyOutput, error) {
		id, err := passkeyID(in.ID)
		if err != nil {
			return nil, err
		}
		k, err := open(lib).RenamePasskey(ctx, a, id, in.Body.Name)
		if err != nil {
			return nil, answer(err)
		}
		return &passkeyOutput{Body: passkeyOf(k)}, nil
	}
}

func deletePasskey(lib Library) func(context.Context, identity.Account, *passkeyIDInput) (*struct{}, error) {
	return func(ctx context.Context, a identity.Account, in *passkeyIDInput) (*struct{}, error) {
		id, err := passkeyID(in.ID)
		if err != nil {
			return nil, err
		}
		return nil, answer(open(lib).DeletePasskey(ctx, a, id))
	}
}

func startPasskeySignIn(lib Library) func(context.Context, identity.Anyone, *struct{}) (*ceremonyOutput, error) {
	return func(ctx context.Context, a identity.Anyone, _ *struct{}) (*ceremonyOutput, error) {
		s, ok := lib.Identity()
		if !ok {
			return nil, passkeyUnavailable
		}
		c, err := s.BeginPasskeySignIn(ctx, a)
		if err != nil {
			return nil, answer(err)
		}
		return ceremonyOf(c), nil
	}
}

func finishPasskeySignIn(lib Library) func(context.Context, identity.Anyone, *finishInput) (*signedInOutput, error) {
	return func(ctx context.Context, a identity.Anyone, in *finishInput) (*signedInOutput, error) {
		s, ok := lib.Identity()
		if !ok {
			return nil, passkeyUnavailable
		}
		got, err := s.FinishPasskeySignIn(ctx, a, identity.CeremonyID(in.Ceremony), in.Body)
		if err != nil {
			return nil, answer(err)
		}
		return signedInOf(got), nil
	}
}

func startPasskeyReconfirm(lib Library) func(context.Context, identity.Account, *struct{}) (*ceremonyOutput, error) {
	return func(ctx context.Context, a identity.Account, _ *struct{}) (*ceremonyOutput, error) {
		c, err := open(lib).BeginPasskeyReconfirm(ctx, a)
		if err != nil {
			return nil, answer(err)
		}
		return ceremonyOf(c), nil
	}
}

func finishPasskeyReconfirm(lib Library) func(context.Context, identity.Account, *finishInput) (*struct{}, error) {
	return func(ctx context.Context, a identity.Account, in *finishInput) (*struct{}, error) {
		return nil, answer(open(lib).FinishPasskeyReconfirm(ctx, a, identity.CeremonyID(in.Ceremony), in.Body))
	}
}
