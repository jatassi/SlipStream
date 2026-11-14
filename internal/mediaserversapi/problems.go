package mediaserversapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jatassi/slipstream/internal/api"
)

var (
	serverNotFound     = api.NewCode(http.StatusNotFound, "media_server_not_found", "No Media Server has this id")
	nameTaken          = api.NewCode(http.StatusConflict, "media_server_name_taken", "Another Media Server has this name")
	kindUnknown        = api.NewCode(http.StatusUnprocessableEntity, "media_server_kind_unknown", "SlipStream doesn't support this kind of Media Server")
	tooOld             = api.NewCode(http.StatusUnprocessableEntity, "media_server_too_old", "This server is older than SlipStream supports")
	otherServer        = api.NewCode(http.StatusConflict, "media_server_other_server", "A different server responded at this address")
	unreached          = api.NewCode(http.StatusBadGateway, "media_server_unreached", "The server didn't respond")
	signInRefused      = api.NewCode(http.StatusUnprocessableEntity, "media_server_sign_in_refused", "The server refused SlipStream's sign-in")
	localhostOnWorker  = api.NewCode(http.StatusUnprocessableEntity, "media_server_localhost_on_worker", "The Controller can't use a localhost address on a Worker")
	nodeNotFound       = api.NewCode(http.StatusNotFound, "node_not_found", "No Node has this id")
	nodeNotLinked      = api.NewCode(http.StatusConflict, "node_not_media_server_link", "That Node doesn't hold Media Server link")
	rootNotFound       = api.NewCode(http.StatusNotFound, "root_not_found", "No root has that id")
	outsideLibraries   = api.NewCode(http.StatusUnprocessableEntity, "mapping_outside_libraries", "A folder isn't inside any of the server's library folders")
	plexNotSignedIn    = api.NewCode(http.StatusConflict, "plex_not_signed_in", "SlipStream isn't signed in to Plex")
	plexServerNotFound = api.NewCode(http.StatusNotFound, "plex_server_not_found", "The Plex account has no server with this id")
	plexPinNotFound    = api.NewCode(http.StatusNotFound, "plex_pin_not_found", "No Plex sign-in has this id")
	plexTVUnreached    = api.NewCode(http.StatusBadGateway, "plex_tv_unreached", "plex.tv didn't respond")
	notBuilt           = api.NewCode(http.StatusNotImplemented, "media_servers_not_built", "This Media Servers operation isn't built yet")
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
	errNotYet             = errors.New("not built yet")
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

func notYet() error { return errNotYet }

// answer is err as the problem an Admin reads.
func answer(err error) error {
	var (
		old      *TooOldError
		other    *OtherServerError
		away     *UnreachedError
		refused  *SignInRefusedError
		local    *LocalhostOnWorkerError
		unlinked *NodeNotLinkedError
		outside  *OutsideLibrariesError
	)
	switch {
	case errors.As(err, &old):
		return tooOld.Because(fmt.Sprintf("%s %s is older than SlipStream supports. Update it to %s or later, then save again.", old.Label, old.Found, old.Floor))
	case errors.As(err, &other):
		return otherServer.Because(fmt.Sprintf("A different %s server responds at %s. Check the address, or confirm the new server under Details.", other.Label, other.Address))
	case errors.As(err, &away):
		return unreached.Because(fmt.Sprintf("%s at %s didn't respond to %s. Check that %s is running and that %s can reach %s.", away.Label, away.Address, away.Node, away.Label, away.Node, away.Address))
	case errors.As(err, &refused):
		return signInRefused.Because(refusedWords(refused))
	case errors.As(err, &local):
		return localhostOnWorker.On("address", fmt.Sprintf("The Controller can't use %s, because localhost on %s is %s itself. Enter %s's network address, or a fallback address the Controller can reach.", local.Address, local.Node, local.Node, local.Node))
	case errors.As(err, &unlinked):
		return nodeNotLinked.Because(fmt.Sprintf("%s doesn't hold Media Server link. Tick the Role on %s, or reach the server from the Controller.", unlinked.Node, unlinked.Node))
	case errors.As(err, &outside):
		return outsideFields(outside)
	case errors.Is(err, ErrNotFound):
		return serverNotFound
	case errors.Is(err, ErrNameTaken):
		return nameTaken
	case errors.Is(err, ErrKindUnknown):
		return kindUnknown
	case errors.Is(err, ErrNodeNotFound):
		return nodeNotFound
	case errors.Is(err, ErrRootNotFound):
		return rootNotFound
	case errors.Is(err, ErrPlexNotSignedIn):
		return plexNotSignedIn.Because("SlipStream isn't signed in to a Plex account. Sign in with Plex, then try again.")
	case errors.Is(err, ErrPlexServerNotFound):
		return plexServerNotFound
	case errors.Is(err, ErrPlexPinNotFound):
		return plexPinNotFound
	case errors.Is(err, ErrPlexTVUnreached):
		return plexTVUnreached.Because("plex.tv didn't respond to the Controller. Check the Controller's internet connection and try again.")
	case errors.Is(err, errNotYet):
		return notBuilt
	}
	return err
}

func refusedWords(e *SignInRefusedError) string {
	switch e.With {
	case RefusedPlex:
		return "SlipStream's sign-in to " + e.Server + " has expired or was revoked. Sign in with Plex again."
	case RefusedKey:
		return e.Server + " refused the API key. Paste a new key, or sign in with an administrator's name and password."
	case RefusedPassword:
	}
	return e.Server + " refused that name and password. Check them and sign in again."
}

func outsideFields(e *OutsideLibrariesError) error {
	found := strings.Join(e.Found, ", ")
	errs := make([]api.FieldError, len(e.Outside))
	for i, o := range e.Outside {
		errs[i] = api.FieldError{
			Location: fmt.Sprintf("body.mappings[%d].serverFolder", o.Index),
			Message:  fmt.Sprintf("%s isn't inside a library folder on %s. Its library folders are %s. Add the folder to a library on the server first.", o.Folder, e.Server, found),
		}
	}
	return outsideLibraries.Fields(errs...)
}
