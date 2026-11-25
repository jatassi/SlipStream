package mediaservers

import (
	"errors"
	"fmt"
)

// The errors of Member access to Media Servers that the API answers as
// problems.
var (
	ErrAccessNotFound    = errors.New("no Media Server access has this id")
	ErrNoMemberLibraries = errors.New("no library for Members is picked while Show to Members is on")
	// ErrNotYet is an operation whose lane of S43 (Member access to Media
	// Servers) isn't built yet. Its method sits in a notyet_*.go file.
	ErrNotYet = errors.New("not built yet")
)

// AccessStateError is an operation that can't start from the row's state:
// the row moved since the caller read it.
type AccessStateError struct{ Member, Server string }

func (e *AccessStateError) Error() string {
	return fmt.Sprintf("%s's access to %s changed", e.Member, e.Server)
}

// LibrariesShortError is a Library for Members that plex.tv doesn't list
// for the server yet.
type LibrariesShortError struct{ Library, Server string }

func (e *LibrariesShortError) Error() string {
	return fmt.Sprintf("plex.tv doesn't list %s on %s", e.Library, e.Server)
}

// PlexRefusedError is plex.tv refusing a share or a Plex Home invite.
// Reply is plex.tv's, which the Admin reads.
type PlexRefusedError struct{ Server, Reply string }

func (e *PlexRefusedError) Error() string {
	return fmt.Sprintf("plex.tv refused the share of %s: %s", e.Server, e.Reply)
}

// AccessUnreachedError is a server an access operation found no route to.
type AccessUnreachedError struct{ Server string }

func (e *AccessUnreachedError) Error() string { return e.Server + " didn't respond" }
