package mediaserversapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
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
	nodeNotLinked      = api.NewCode(http.StatusConflict, "node_not_media_server_link", "That Node doesn't have the Media Server link Role")
	rootNotFound       = api.NewCode(http.StatusNotFound, "root_not_found", "No root has that id")
	outsideLibraries   = api.NewCode(http.StatusUnprocessableEntity, "mapping_outside_libraries", "A folder isn't inside any of the server's library folders")
	plexNotSignedIn    = api.NewCode(http.StatusConflict, "plex_not_signed_in", "SlipStream isn't signed in to Plex")
	plexServerNotFound = api.NewCode(http.StatusNotFound, "plex_server_not_found", "The Plex account has no server with this id")
	plexPinNotFound    = api.NewCode(http.StatusNotFound, "plex_pin_not_found", "No Plex sign-in has this id")
	plexTVUnreached    = api.NewCode(http.StatusBadGateway, "plex_tv_unreached", "plex.tv didn't respond")
	marginTooShort     = api.NewCode(http.StatusUnprocessableEntity, "renew_margin_too_short", "The renewal window isn't longer than the renewal interval")
	accountNotFound    = api.NewCode(http.StatusNotFound, "account_not_found", "No Account has that id")
	accessNotFound     = api.NewCode(http.StatusNotFound, "media_server_access_not_found", "No Media Server access has this id")
	accessState        = api.NewCode(http.StatusConflict, "access_state", "The access changed since this page loaded")
	plexRefused        = api.NewCode(http.StatusUnprocessableEntity, "plex_refused", "Plex refused the share")
	plexLibsShort      = api.NewCode(http.StatusConflict, "plex_libraries_short", "plex.tv doesn't list a library for Members yet")
	usernameTaken      = api.NewCode(http.StatusConflict, "username_taken", "The server has a user with this name")
	serverRefused      = api.NewCode(http.StatusUnprocessableEntity, "server_refused", "The server refused this name")
	notShownToMembers  = api.NewCode(http.StatusConflict, "not_shown_to_members", "The Media Server isn't shown to Members")
	plexHomeNotPlex    = api.NewCode(http.StatusUnprocessableEntity, "plex_home_not_plex", "Plex Home is a choice for Plex servers only")
	noMemberLibraries  = api.NewCode(http.StatusUnprocessableEntity, "no_member_libraries", "Show to Members needs at least one library for Members")
	playNotFound       = api.NewCode(http.StatusNotFound, "play_not_found", "No play on this Media Server has this id")
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
	if a := accessAnswer(err); a != nil {
		return a
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
	case errors.Is(err, mediaservers.ErrNoSuchPlay):
		return playNotFound
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

// accessAnswer is an error of Member access to Media Servers as the
// problem an Admin reads, nil for any other error.
func accessAnswer(err error) error {
	if e, ok := errors.AsType[*mediaservers.AccessStateError](err); ok {
		return accessState.Because(fmt.Sprintf("%s's access to %s changed since this page loaded. Reload the page to see it.", e.Member, e.Server))
	}
	if e, ok := errors.AsType[*mediaservers.LibrariesShortError](err); ok {
		return plexLibsShort.Because(fmt.Sprintf("plex.tv doesn't list %s on %s yet. Try again in a few minutes.", e.Library, e.Server))
	}
	if e, ok := errors.AsType[*mediaservers.PlexRefusedError](err); ok {
		return plexRefused.Because(e.Detail)
	}
	if e, ok := errors.AsType[*mediaservers.AccessUnreachedError](err); ok {
		return unreached.Because(e.Server + " isn't responding to SlipStream. Check that it is running, then try again.")
	}
	if e, ok := errors.AsType[*identity.NotGrantableError](err); ok {
		if e.Why == identity.NotPlexWay {
			return plexHomeNotPlex.Because(e.Name + " isn't a Plex server, so it has no Plex Home. Offer it without Plex Home.")
		}
		return notShownToMembers.Because(e.Name + " isn't shown to Members. Turn on Show to Members on its page first.")
	}
	switch {
	case errors.Is(err, mediaservers.ErrAccessNotFound):
		return accessNotFound
	case errors.Is(err, mediaservers.ErrNoMemberLibraries):
		return noMemberLibraries.On("memberLibraries", "Pick at least one library for Members.")
	case errors.Is(err, mediaservers.ErrMemberAddressOnPlex):
		return api.Invalid.On("memberAddress", "A Plex server has no Address for Members, because Plex apps connect through the Member's Plex account. Clear the address.")
	case errors.Is(err, identity.ErrNoAccount):
		return accountNotFound
	}
	return nil
}

// answerMine is err as the problem a Member reads about their own access,
// which never carries a reply meant for an Admin.
func answerMine(err error) error {
	if e, ok := errors.AsType[*mediaservers.AccessStateError](err); ok {
		return accessState.Because(fmt.Sprintf("Your access to %s changed since this page loaded. Reload the page to see it.", e.Server))
	}
	if e, ok := errors.AsType[*mediaservers.AccessUnreachedError](err); ok {
		return unreached.Because(e.Server + " isn't responding to SlipStream. Try again in a few minutes.")
	}
	if e, ok := errors.AsType[*link.NameRefusedError](err); ok {
		if e.Taken {
			return usernameTaken.On("username", "The server has a user with this name. Choose another username.")
		}
		return serverRefused.On("username", fmt.Sprintf("The server refused this username. Its response was %q. Choose another username.", e.Reply))
	}
	switch {
	case errors.Is(err, mediaservers.ErrPlexNotSignedIn):
		return plexNotSignedIn.Because("SlipStream isn't signed in to Plex. Ask an Admin to sign in with Plex on the Media Servers page.")
	case errors.Is(err, mediaservers.ErrPlexTVUnreached):
		return plexTVUnreached.Because("plex.tv isn't responding. Try again in a few minutes.")
	}
	return answer(err)
}
