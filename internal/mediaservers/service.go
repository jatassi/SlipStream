package mediaservers

import (
	"context"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/setting"
)

// Service is what internal/mediaserversapi's operations call. It renders
// every words field, and its errors are the ones errors.go declares.
type Service interface {
	List(ctx context.Context) ([]ServerView, error)
	Get(ctx context.Context, id link.ServerID) (ServerView, error)
	// Add tests the draft from its reaching Node and the Controller, checks
	// the version floor and saves it.
	Add(ctx context.Context, d Draft) (ServerView, error)
	Update(ctx context.Context, id link.ServerID, p Patch) (ServerView, error)
	// Remove forgets the server's changes, index and play history.
	Remove(ctx context.Context, id link.ServerID) error
	Test(ctx context.Context, id link.ServerID) (Tested, error)
	ConfirmIdentity(ctx context.Context, id link.ServerID) (ServerView, error)
	Libraries(ctx context.Context, id link.ServerID) (Libraries, error)
	SaveMappings(ctx context.Context, id link.ServerID, ms []Mapping) ([]MappingView, error)
	// Refresh queues a whole library refresh of every shown, confirmed
	// mapping's library.
	Refresh(ctx context.Context, id link.ServerID) error
	Sessions(ctx context.Context, id link.ServerID) (Sessions, error)
	Changes(ctx context.Context, id link.ServerID, cursor string) (Changes, error)
	CredentialHolders(ctx context.Context, id link.ServerID) ([]HolderView, error)
	SignIn(ctx context.Context, d SignInDraft) (SignedIn, error)
	StartPlexPin(ctx context.Context, forward string) (PlexPin, error)
	PlexPin(ctx context.Context, id int64) (PlexPinState, error)
	PlexAccount(ctx context.Context) (AccountView, error)
	SignOutOfPlex(ctx context.Context) error
	PlexServers(ctx context.Context) ([]PlexServerView, error)
	Settings() setting.Panel
}

// Mapping is one root's mapping to save, parsed from the request.
type Mapping struct {
	Root   roots.ID
	Shown  bool
	Folder string
}

// Unbuilt answers 501 for each operation not built yet. The Service
// embeds it for the operations still in TestNotYetLedger, and each that is
// built deletes its method here and its ledger line.
type Unbuilt struct{}

func (Unbuilt) Refresh(context.Context, link.ServerID) error { return ErrNotBuilt }

func (Unbuilt) Sessions(context.Context, link.ServerID) (Sessions, error) {
	return Sessions{}, ErrNotBuilt
}

func (Unbuilt) Changes(context.Context, link.ServerID, string) (Changes, error) {
	return Changes{}, ErrNotBuilt
}
