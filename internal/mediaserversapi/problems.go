package mediaserversapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/settingapi"
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
	marginTooShort     = api.NewCode(http.StatusUnprocessableEntity, "renew_margin_too_short", "The renewal margin isn't longer than the renewal schedule")
)

// answer is err as the problem an Admin reads.
func answer(err error) error {
	words, _ := mediaservers.Words(err)
	switch {
	case errors.As(err, new(*mediaservers.TooOldError)):
		return tooOld.Because(words)
	case errors.As(err, new(*mediaservers.OtherServerError)):
		return otherServer.Because(words)
	case errors.As(err, new(*mediaservers.UnreachedError)):
		return unreached.Because(words)
	case errors.As(err, new(*mediaservers.SignInRefusedError)):
		return signInRefused.Because(words)
	case errors.As(err, new(*mediaservers.LocalhostOnWorkerError)):
		return localhostOnWorker.On("address", words)
	case errors.As(err, new(*mediaservers.NodeNotLinkedError)):
		return nodeNotLinked.Because(words)
	case errors.As(err, new(*mediaservers.MarginTooShortError)):
		return marginTooShort.On("values."+mediaservers.RenewMargin.Key, words)
	}
	if outside, ok := errors.AsType[*mediaservers.OutsideLibrariesError](err); ok {
		return outsideFields(outside)
	}
	switch {
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
	}
	return settingapi.Refused(err)
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
