package mediaservers

import (
	"errors"
	"fmt"
)

// The errors a Service returns, which the API answers as problems. Any
// other error answers 500.
var (
	ErrNotFound           = errors.New("no Media Server has this id")
	ErrNameTaken          = errors.New("another Media Server has this name")
	ErrKindUnknown        = errors.New("not a kind of Media Server SlipStream supports")
	ErrNodeNotFound       = errors.New("no Node has this id")
	ErrRootNotFound       = errors.New("no root has this id")
	ErrPlexNotSignedIn    = errors.New("SlipStream isn't signed in to Plex")
	ErrPlexServerNotFound = errors.New("the Plex account has no server with this id")
	ErrPlexPinNotFound    = errors.New("no Plex sign-in has this id")
	ErrPlexTVUnreached    = errors.New("plex.tv didn't respond")
)

// TooOldError is a server below its kind's version floor.
type TooOldError struct {
	// Label is the kind's label, such as Jellyfin.
	Label, Found, Floor string
}

func (e *TooOldError) Error() string {
	return fmt.Sprintf("%s %s is older than the supported %s", e.Label, e.Found, e.Floor)
}

// OtherServerError is another identity responding at a server's address.
type OtherServerError struct{ Label, Address string }

func (e *OtherServerError) Error() string {
	return fmt.Sprintf("a different %s server responds at %s", e.Label, e.Address)
}

// UnreachedError is no response at Address from Node.
type UnreachedError struct{ Label, Node, Address string }

func (e *UnreachedError) Error() string {
	return fmt.Sprintf("%s at %s didn't respond to %s", e.Label, e.Address, e.Node)
}

// Refused is the credential a server refused.
type Refused uint8

const (
	// RefusedPlex is a saved Plex server refusing the account's token.
	RefusedPlex Refused = iota + 1
	// RefusedKey is a Jellyfin or Emby server refusing its API key.
	RefusedKey
	// RefusedPassword is an administrator sign-in refused.
	RefusedPassword
)

// SignInRefusedError is a server refusing SlipStream's credential. Server
// is the saved server's name, or for a draft the kind's label and address.
type SignInRefusedError struct {
	Server string
	With   Refused
}

func (e *SignInRefusedError) Error() string { return e.Server + " refused SlipStream's sign-in" }

// LocalhostOnWorkerError is a localhost address on a Worker with no
// fallback address for the Controller.
type LocalhostOnWorkerError struct{ Node, Address string }

func (e *LocalhostOnWorkerError) Error() string {
	return fmt.Sprintf("%s is localhost on %s, which the Controller can't use", e.Address, e.Node)
}

// NodeNotLinkedError is a reaching Node that doesn't hold Media Server
// link.
type NodeNotLinkedError struct{ Node string }

func (e *NodeNotLinkedError) Error() string { return e.Node + " doesn't hold Media Server link" }

// Outside is one shown mapping whose folder isn't inside a library folder.
type Outside struct {
	// Index is the mapping's place in the saved list.
	Index  int
	Root   string
	Folder string
}

// OutsideLibrariesError names the mappings outside every library folder,
// and the folders the server has.
type OutsideLibrariesError struct {
	Server  string
	Outside []Outside
	Found   []string
}

func (e *OutsideLibrariesError) Error() string {
	return fmt.Sprintf("%d folders aren't inside a library folder on %s", len(e.Outside), e.Server)
}
