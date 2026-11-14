package mediaserversapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/mediaservers"
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

// answer is err as the problem an Admin reads.
func answer(err error) error {
	var (
		old      *mediaservers.TooOldError
		other    *mediaservers.OtherServerError
		away     *mediaservers.UnreachedError
		refused  *mediaservers.SignInRefusedError
		local    *mediaservers.LocalhostOnWorkerError
		unlinked *mediaservers.NodeNotLinkedError
		outside  *mediaservers.OutsideLibrariesError
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
	case errors.Is(err, mediaservers.ErrNotFound):
		return serverNotFound
	case errors.Is(err, mediaservers.ErrNameTaken):
		return nameTaken
	case errors.Is(err, mediaservers.ErrKindUnknown):
		return kindUnknown
	case errors.Is(err, mediaservers.ErrNodeNotFound):
		return nodeNotFound
	case errors.Is(err, mediaservers.ErrRootNotFound):
		return rootNotFound
	case errors.Is(err, mediaservers.ErrPlexNotSignedIn):
		return plexNotSignedIn.Because("SlipStream isn't signed in to a Plex account. Sign in with Plex, then try again.")
	case errors.Is(err, mediaservers.ErrPlexServerNotFound):
		return plexServerNotFound
	case errors.Is(err, mediaservers.ErrPlexPinNotFound):
		return plexPinNotFound
	case errors.Is(err, mediaservers.ErrPlexTVUnreached):
		return plexTVUnreached.Because("plex.tv didn't respond to the Controller. Check the Controller's internet connection and try again.")
	case errors.Is(err, mediaservers.ErrNotBuilt):
		return notBuilt
	}
	return err
}

func refusedWords(e *mediaservers.SignInRefusedError) string {
	switch e.With {
	case mediaservers.RefusedPlex:
		return "SlipStream's sign-in to " + e.Server + " has expired or was revoked. Sign in with Plex again."
	case mediaservers.RefusedKey:
		return e.Server + " refused the API key. Paste a new key, or sign in with an administrator's name and password."
	case mediaservers.RefusedPassword:
	}
	return e.Server + " refused that name and password. Check them and sign in again."
}

func outsideFields(e *mediaservers.OutsideLibrariesError) error {
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
