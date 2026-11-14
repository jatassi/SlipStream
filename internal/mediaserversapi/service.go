package mediaserversapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/roots"
)

// Service is what the operations call. internal/mediaservers implements it
// and renders every words field. Its errors are the ones problems.go
// declares.
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
}

// Mapping is one root's mapping to save, parsed from the request.
type Mapping struct {
	Root   roots.ID
	Shown  bool
	Folder string
}

// Unbuilt is the Service until internal/mediaservers builds it: no
// servers, and every other operation answers 501. The built Service embeds
// it for the operations still in TestNotYetLedger, and each that it builds
// deletes its method here and its ledger line.
type Unbuilt struct{}

func (Unbuilt) List(context.Context) ([]ServerView, error) { return []ServerView{}, nil }

func (Unbuilt) Get(context.Context, link.ServerID) (ServerView, error) {
	return ServerView{}, ErrNotFound
}

func (Unbuilt) Add(context.Context, Draft) (ServerView, error) { return ServerView{}, notYet() }

func (Unbuilt) Update(context.Context, link.ServerID, Patch) (ServerView, error) {
	return ServerView{}, notYet()
}

func (Unbuilt) Remove(context.Context, link.ServerID) error { return notYet() }

func (Unbuilt) Test(context.Context, link.ServerID) (Tested, error) { return Tested{}, notYet() }

func (Unbuilt) ConfirmIdentity(context.Context, link.ServerID) (ServerView, error) {
	return ServerView{}, notYet()
}

func (Unbuilt) Libraries(context.Context, link.ServerID) (Libraries, error) {
	return Libraries{}, notYet()
}

func (Unbuilt) SaveMappings(context.Context, link.ServerID, []Mapping) ([]MappingView, error) {
	return nil, notYet()
}

func (Unbuilt) Refresh(context.Context, link.ServerID) error { return notYet() }

func (Unbuilt) Sessions(context.Context, link.ServerID) (Sessions, error) {
	return Sessions{}, notYet()
}

func (Unbuilt) Changes(context.Context, link.ServerID, string) (Changes, error) {
	return Changes{}, notYet()
}

func (Unbuilt) CredentialHolders(context.Context, link.ServerID) ([]HolderView, error) {
	return nil, notYet()
}

func (Unbuilt) SignIn(context.Context, SignInDraft) (SignedIn, error) { return SignedIn{}, notYet() }

func (Unbuilt) StartPlexPin(context.Context, string) (PlexPin, error) { return PlexPin{}, notYet() }

func (Unbuilt) PlexPin(context.Context, int64) (PlexPinState, error) {
	return PlexPinState{}, notYet()
}

func (Unbuilt) PlexAccount(context.Context) (AccountView, error) { return AccountView{}, notYet() }

func (Unbuilt) SignOutOfPlex(context.Context) error { return notYet() }

func (Unbuilt) PlexServers(context.Context) ([]PlexServerView, error) { return nil, notYet() }
