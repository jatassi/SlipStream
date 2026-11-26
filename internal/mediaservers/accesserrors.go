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
// Detail is the sentence the Admin reads, with plex.tv's reply.
type PlexRefusedError struct{ Server, Detail string }

func (e *PlexRefusedError) Error() string {
	return "plex.tv refused the share of " + e.Server
}

// AccessUnreachedError is a server an access operation found no route to.
type AccessUnreachedError struct{ Server string }

func (e *AccessUnreachedError) Error() string { return e.Server + " didn't respond" }
