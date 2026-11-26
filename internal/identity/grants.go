package identity

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/storage"
)

// Grants are what a Signup link offers besides the Account. Each field
// belongs to one Granter: MediaServers to mediaservers.
type Grants struct {
	MediaServers []ServerGrant `json:"mediaServers,omitempty"`
}

// ServerGrant is one Media Server a Signup link offers, and for a Plex
// server whether the Member joins the Admin's Plex Home.
type ServerGrant struct {
	Server   storage.UUID `json:"server"`
	PlexHome bool         `json:"plexHome,omitempty"`
}

func (g Grants) Empty() bool { return len(g.MediaServers) == 0 }

// Described is Grants as the pending link row, the by-token view and Send
// again show them.
type Described struct {
	MediaServers []GrantedServer
}

type GrantedServer struct {
	Server storage.UUID
	Name   string
	// Kind is the kind's label, for copy.
	Kind string
	// Present is false once the server is removed.
	Present bool
	// Shown is whether Show to Members is on.
	Shown    bool
	PlexHome bool
}

// Redemption is one Signup link becoming an Account.
type Redemption struct {
	Account AccountID
	// IssuedBy is zero for a link the API key issued.
	IssuedBy AccountID
	Grants   Grants
}

// Granter is what owns a Grants field. identity calls it and never imports
// the owner.
type Granter interface {
	// CheckGrants refuses a grant at issue, inside the issue Write, with
	// *NotGrantableError.
	CheckGrants(ctx context.Context, r storage.Reader[file], g Grants) error
	Describe(ctx context.Context, r storage.Reader[file], g Grants) (Described, error)
	// Redeemed runs inside the redemption Write. identity runs after once
	// the Write commits.
	Redeemed(ctx context.Context, tx *storage.Tx[file], r Redemption) (after func(context.Context), err error)
	// Leaving runs before DeleteAccount's Write and removes what the Account
	// was given. identity calls done once its Write ends, committed or not.
	Leaving(ctx context.Context, account AccountID) (done func(), err error)
}

// NotGrantableError is a grant its owner refuses.
type NotGrantableError struct {
	// Name is the server's, for the problem detail, and empty for Gone.
	Name string
	Why  NotGrantable
}

type NotGrantable uint8

const (
	// NotShown is a server not shown to Members.
	NotShown NotGrantable = iota + 1
	// NotPlexWay is Plex Home asked of a server that isn't Plex.
	NotPlexWay
	// Gone is a server that was removed.
	Gone
)

func (e *NotGrantableError) Error() string {
	switch e.Why {
	case NotShown:
	case NotPlexWay:
		return e.Name + " isn't a Plex server and has no Plex Home"
	case Gone:
		return "a Media Server on the link was removed"
	}
	return e.Name + " isn't shown to Members"
}

// ErrNoGranter is a call with grants, or an Account delete, before
// SetGranter.
var ErrNoGranter = errors.New("nothing has taken the Signup link grants yet")

// SetGranter hands grants to g. startController calls it once. The offline
// admin reset never does.
func (s *Service) SetGranter(g Granter) { s.granter.Store(&g) }

// NoGrants is the Granter of a Library with no Media Servers, for test
// harnesses that build identity alone. It refuses every non-empty Grants
// and leaves nothing behind.
type NoGrants struct{}

var _ Granter = NoGrants{}

func (NoGrants) CheckGrants(_ context.Context, _ storage.Reader[file], g Grants) error {
	if g.Empty() {
		return nil
	}
	return &NotGrantableError{Why: Gone}
}

func (NoGrants) Describe(context.Context, storage.Reader[file], Grants) (Described, error) {
	return Described{}, nil
}

func (NoGrants) Redeemed(context.Context, *storage.Tx[file], Redemption) (func(context.Context), error) {
	return func(context.Context) {}, nil
}

func (NoGrants) Leaving(context.Context, AccountID) (func(), error) { return func() {}, nil }
