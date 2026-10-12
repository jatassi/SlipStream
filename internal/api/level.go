package api

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
)

// Level is an operation's access level. Its type parameter is the principal
// its handlers receive. The four values below are the whole set.
type Level[P any] struct {
	*access
	admit admitFunc[P]
}

// admitFunc runs next with the caller narrowed to the level's principal, or
// returns the identity error that refuses it.
type admitFunc[P any] func(ctx context.Context, gate *identity.Gate, c identity.Caller, setupCode string, next func(P)) error

type access struct {
	name     string
	rank     int
	security []map[string][]string
	codes    []*Code
	// authenticates is false only for public, whose principal needs no
	// credential.
	authenticates bool
}

// Floor is the lowest level an owner's operations may use.
type Floor interface {
	level() *access
}

func (l Level[P]) level() *access { return l.access }

var (
	Public = Level[identity.Anyone]{
		&access{name: "public", rank: 0, security: []map[string][]string{}},
		narrowed(func(c identity.Caller) (identity.Anyone, error) { return c.Anyone(), nil }),
	}
	Setup = Level[identity.Claim]{
		&access{name: "setup", rank: 1, security: []map[string][]string{{schemeSetupCode: {}}}, codes: []*Code{APIKeyWrong, SetupCodeWrong, SetupBusy, AlreadyClaimed}, authenticates: true},
		func(ctx context.Context, gate *identity.Gate, c identity.Caller, setupCode string, next func(identity.Claim)) error {
			return gate.Run(ctx, c, setupCode, next)
		},
	}
	Account = Level[identity.Account]{
		&access{name: "account", rank: 2, security: []map[string][]string{{schemeSession: {}}}, codes: []*Code{SignedOut, APIKeyWrong, NeedsAccount}, authenticates: true},
		narrowed(identity.Caller.Account),
	}
	Admin = Level[identity.Admin]{
		&access{name: "admin", rank: 3, security: []map[string][]string{{schemeSession: {"admin"}}, {schemeAPIKey: {}}}, codes: []*Code{SignedOut, APIKeyWrong, NeedsAdmin}, authenticates: true},
		narrowed(identity.Caller.Admin),
	}
)

// zeroLevel ends the panic for a Level that is none of the four, such as
// another package's api.Level[identity.Admin]{}.
const zeroLevel = "a zero api.Level; use api.Public, api.Setup, api.Account or api.Admin"

var levels = []*access{Public.access, Setup.access, Account.access, Admin.access}

func narrowed[P any](narrow func(identity.Caller) (P, error)) admitFunc[P] {
	return func(_ context.Context, _ *identity.Gate, c identity.Caller, _ string, next func(P)) error {
		p, err := narrow(c)
		if err != nil {
			return err
		}
		next(p)
		return nil
	}
}

const (
	schemeSession   = "session"
	schemeAPIKey    = "apiKey"
	schemeSetupCode = "setupCode"
)
