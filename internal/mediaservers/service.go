package mediaservers

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/playback"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/works"
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
	// Changes lists the server's changes newest first, the page before
	// before, which is zero for the newest page.
	Changes(ctx context.Context, id link.ServerID, before link.ChangeID) (Changes, error)
	CredentialHolders(ctx context.Context, id link.ServerID) ([]HolderView, error)
	// Playback is the Playback report over the plays the window keeps.
	Playback(ctx context.Context, id link.ServerID) (playback.Report, error)
	// Plays lists the server's plays newest first, the page after cursor,
	// which is zero for the newest page. An empty player lists every
	// player's.
	Plays(ctx context.Context, id link.ServerID, player string, cursor link.PlayID) (Plays, error)
	SignIn(ctx context.Context, d SignInDraft) (SignedIn, error)
	StartPlexPin(ctx context.Context, forward string) (PlexPin, error)
	PlexPin(ctx context.Context, id int64) (PlexPinState, error)
	PlexAccount(ctx context.Context) (AccountView, error)
	SignOutOfPlex(ctx context.Context) error
	PlexServers(ctx context.Context) ([]PlexServerView, error)
	// OpenIn is the Work page's links to work on each server that holds
	// it.
	OpenIn(ctx context.Context, work works.WorkID) ([]OpenIn, error)
	Settings() setting.Panel
	// ReplacePlexAccount confirms an Admin's Plex sign-in as a different
	// Plex account, the pin state other_account.
	ReplacePlexAccount(ctx context.Context, pin int64) (AccountView, error)

	// Access lists the access rows, Needs you first. f.Account narrows it
	// to one Member and adds the servers they may be offered.
	Access(ctx context.Context, f AccessFilter) (AccessList, error)
	Offer(ctx context.Context, by identity.Admin, account identity.AccountID, server link.ServerID, plexHome bool) (Acted, error)
	Share(ctx context.Context, by identity.Admin, id AccessID, as Invite) (Acted, error)
	MarkAdded(ctx context.Context, by identity.Admin, id AccessID) (Acted, error)
	LinkUser(ctx context.Context, by identity.Admin, id AccessID, user string) (Acted, error)
	CheckShare(ctx context.Context, by identity.Admin, id AccessID) (Acted, error)
	RemoveAccess(ctx context.Context, by identity.Admin, id AccessID) (Acted, error)
	// LinkableUsers are the server's users an Admin may link to a Member:
	// no administrator, no disabled user and no user already linked.
	LinkableUsers(ctx context.Context, server link.ServerID) ([]ServerUser, error)

	// Mine is the Member's own access.
	Mine(ctx context.Context, me identity.Account) (MyAccess, error)
	StartMemberPin(ctx context.Context, me identity.Account) (MemberPin, error)
	MemberPin(ctx context.Context, me identity.Account, id int64) (MemberPinState, error)
	// AnotherPlex forgets the Member's Plex account, for Use another Plex
	// account.
	AnotherPlex(ctx context.Context, me identity.Account) error
	MakeAccount(ctx context.Context, me identity.Account, id AccessID, username string, pw link.Password) (MyAccessRow, error)
	CheckMine(ctx context.Context, me identity.Account, id AccessID) (MyAccessRow, error)
}

var ErrNoSuchPlay = errors.New("no play on this Media Server has this id")

// Mapping is one root's mapping to save, parsed from the request.
type Mapping struct {
	Root   roots.ID
	Shown  bool
	Folder string
}
