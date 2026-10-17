package identityapi

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
)

type reconfirmInput struct{ Body reconfirmation }

type reconfirmation struct {
	PIN string `json:"pin" pattern:"^[0-9]{4}$"`
}

type changePINInput struct{ Body pinChange }

type pinChange struct {
	Current *string `json:"current,omitempty" pattern:"^[0-9]{4}$" doc:"The current PIN, left out only by a session that must set a new PIN"`
	New     string  `json:"new" pattern:"^[0-9]{4}$"`
}

type apiKeyOutput struct{ Body apiKeyMade }

type apiKeyMade struct {
	CreatedAt time.Time `json:"createdAt"`
}

type newAPIKeyOutput struct{ Body regeneratedAPIKey }

type regeneratedAPIKey struct {
	APIKey    string    `json:"apiKey" pattern:"^ssk_[A-Za-z0-9_-]{43}$"`
	CreatedAt time.Time `json:"createdAt"`
}

func reconfirm(lib Library) func(context.Context, identity.Account, *reconfirmInput) (*struct{}, error) {
	return func(ctx context.Context, a identity.Account, in *reconfirmInput) (*struct{}, error) {
		pin, err := identity.ParsePIN(in.Body.PIN)
		if err != nil {
			return nil, err
		}
		return nil, answer(open(lib).Reconfirm(ctx, a, pin))
	}
}

func changePIN(lib Library) func(context.Context, identity.Bearer, *changePINInput) (*struct{}, error) {
	return func(ctx context.Context, b identity.Bearer, in *changePINInput) (*struct{}, error) {
		next, err := identity.ParsePIN(in.Body.New)
		if err != nil {
			return nil, err
		}
		var current *identity.PIN
		if in.Body.Current != nil {
			pin, err := identity.ParsePIN(*in.Body.Current)
			if err != nil {
				return nil, err
			}
			current = &pin
		}
		return nil, answer(open(lib).ChangePIN(ctx, b, current, next))
	}
}

func getAPIKey(lib Library) func(context.Context, identity.Admin, *struct{}) (*apiKeyOutput, error) {
	return func(ctx context.Context, a identity.Admin, _ *struct{}) (*apiKeyOutput, error) {
		at, err := open(lib).APIKeyCreatedAt(ctx, a)
		if err != nil {
			return nil, err
		}
		return &apiKeyOutput{Body: apiKeyMade{CreatedAt: at}}, nil
	}
}

func regenerateAPIKey(lib Library) func(context.Context, identity.Admin, *struct{}) (*newAPIKeyOutput, error) {
	return func(ctx context.Context, a identity.Admin, _ *struct{}) (*newAPIKeyOutput, error) {
		key, at, err := open(lib).RegenerateAPIKey(ctx, a)
		if err != nil {
			return nil, err
		}
		return &newAPIKeyOutput{Body: regeneratedAPIKey{APIKey: key.Reveal(), CreatedAt: at}}, nil
	}
}
