package api

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
)

// Level is an operation's access level. Its type parameter is the principal
// its handlers receive. The seven values below are the whole set.
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
	// byClaim answers the check from the Node's claim first: only a
	// Controller has Accounts.
	byClaim bool
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
	// Bearer admits a session that must set a new PIN, which every other
	// session level refuses with pin_required.
	Bearer = Level[identity.Bearer]{
		&access{name: "bearer", rank: 2, security: []map[string][]string{{schemeSession: {}}}, codes: []*Code{SignedOut, APIKeyWrong, NeedsAccount, WorkerNode}, authenticates: true, byClaim: true},
		narrowed(identity.Caller.Bearer),
	}
	Account = Level[identity.Account]{
		&access{name: "account", rank: 3, security: []map[string][]string{{schemeSession: {}}}, codes: []*Code{SignedOut, APIKeyWrong, NeedsAccount, PINRequired, WorkerNode}, authenticates: true, byClaim: true},
		narrowed(identity.Caller.Account),
	}
	Confirmed = Level[identity.Account]{
		&access{name: "confirmed", rank: 4, security: []map[string][]string{{schemeSession: {scopeReconfirmed}}}, codes: []*Code{SignedOut, APIKeyWrong, NeedsAccount, PINRequired, NeedsReconfirm, WorkerNode}, authenticates: true, byClaim: true},
		narrowed(identity.Caller.Confirmed),
	}
	Admin = Level[identity.Admin]{
		&access{name: "admin", rank: 5, security: []map[string][]string{{schemeSession: {scopeAdmin}}, {schemeAPIKey: {}}}, codes: []*Code{SignedOut, APIKeyWrong, PINRequired, NeedsAdmin, WorkerNode}, authenticates: true, byClaim: true},
		narrowed(identity.Caller.Admin),
	}
	Reconfirmed = Level[identity.Admin]{
		&access{name: "reconfirmed", rank: 6, security: []map[string][]string{{schemeSession: {scopeAdmin, scopeReconfirmed}}, {schemeAPIKey: {}}}, codes: []*Code{SignedOut, APIKeyWrong, PINRequired, NeedsAdmin, NeedsReconfirm, WorkerNode}, authenticates: true, byClaim: true},
		narrowed(identity.Caller.Reconfirmed),
	}
)

// zeroLevel ends the panic for a Level that is none of the seven, such as
// another package's api.Level[identity.Admin]{}.
const zeroLevel = "a zero api.Level; use api.Public, api.Setup, api.Bearer, api.Account, api.Confirmed, api.Admin or api.Reconfirmed"

var levels = []*access{Public.access, Setup.access, Bearer.access, Account.access, Confirmed.access, Admin.access, Reconfirmed.access}

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

	scopeAdmin       = "admin"
	scopeReconfirmed = "reconfirmed"
)
