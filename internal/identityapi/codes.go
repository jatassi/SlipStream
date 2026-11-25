package identityapi

import (
	"errors"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
)

var (
	pinWrong           = api.NewCode(http.StatusUnauthorized, "pin_wrong", "The email or PIN is wrong")
	pinWait            = api.NewWaiting(http.StatusTooManyRequests, "pin_wait", "Too many wrong PINs")
	pinOff             = api.NewCode(http.StatusForbidden, "pin_off", "This PIN is locked after too many wrong PINs. Sign in with a passkey or ask an Admin for a Reset link.")
	tooManyAttempts    = api.NewWaiting(http.StatusTooManyRequests, "too_many_attempts", "Too many failed sign-ins from this IP address")
	addressIP          = api.NewCode(http.StatusUnprocessableEntity, "address_ip", "The address must be a name, not an IP address")
	accountDisabled    = api.NewCode(http.StatusForbidden, "account_disabled", "This Account is turned off. Ask an Admin to turn it back on.")
	addressUnset       = api.NewCode(http.StatusConflict, "address_unset", "Set SlipStream's address first, so links point somewhere people can reach")
	accountExists      = api.NewCode(http.StatusConflict, "account_exists", "An Account already has that email")
	emailTaken         = api.NewCode(http.StatusConflict, "email_taken", "Another Account has that email")
	lastAdmin          = api.NewCode(http.StatusConflict, "last_admin", "SlipStream needs at least one active Admin. Make another Account an Admin first.")
	linkDead           = api.NewCode(http.StatusGone, "link_dead", "This link no longer works")
	linkNotFound       = api.NewCode(http.StatusNotFound, "link_not_found", "No pending Signup link has that id")
	accountNotFound    = api.NewCode(http.StatusNotFound, "account_not_found", "No Account has that id")
	sessionNotFound    = api.NewCode(http.StatusNotFound, "session_not_found", "The Account has no session with that id")
	passkeyNotFound    = api.NewCode(http.StatusNotFound, "passkey_not_found", "You have no passkey with that id")
	passkeyUnavailable = api.NewCode(http.StatusConflict, "passkey_unavailable", "You have no passkey for this address, or passkeys don't work here. Sign in with your PIN.")
	passkeyFailed      = api.NewCode(http.StatusUnauthorized, "passkey_failed", "The passkey couldn't be verified")
	notShownToMembers  = api.NewCode(http.StatusConflict, "not_shown_to_members", "The Media Server isn't shown to Members")
	plexHomeNotPlex    = api.NewCode(http.StatusUnprocessableEntity, "plex_home_not_plex", "Plex Home is a choice for Plex servers only")
	notBuilt           = api.NewCode(http.StatusNotImplemented, "not_built_yet", "This part of SlipStream isn't built yet")
)

var pinChecked = []*api.Code{pinWrong, pinWait.Code, pinOff, tooManyAttempts.Code}

var errorCodes = []struct {
	err  error
	code *api.Code
}{
	{identity.ErrPINWrong, pinWrong},
	{identity.ErrPINOff, pinOff},
	{identity.ErrAddressIP, addressIP},
	{identity.ErrAccountDisabled, accountDisabled},
	{identity.ErrAddressUnset, addressUnset},
	{identity.ErrAccountExists, accountExists},
	{identity.ErrEmailTaken, emailTaken},
	{identity.ErrLastAdmin, lastAdmin},
	{identity.ErrNoLink, linkNotFound},
	{identity.ErrNoAccount, accountNotFound},
	{identity.ErrNoSession, sessionNotFound},
	{identity.ErrNoPasskey, passkeyNotFound},
	{identity.ErrPasskeyUnavailable, passkeyUnavailable},
	{identity.ErrPasskeyFailed, passkeyFailed},
}

// answer is the one map from identity's errors to codes. An operation lists
// in its Op the codes it may produce.
func answer(err error) error {
	if wait, ok := errors.AsType[*identity.WaitError](err); ok {
		if wait.Kind == identity.TooManyAttempts {
			return tooManyAttempts.Until(wait.Until)
		}
		return pinWait.Until(wait.Until)
	}
	if _, ok := errors.AsType[*identity.DeadError](err); ok {
		return linkDead
	}
	for _, e := range errorCodes {
		if errors.Is(err, e.err) {
			return e.code
		}
	}
	switch {
	case errors.Is(err, identity.ErrCurrentPINMissing):
		return api.Invalid.Fields(api.FieldError{Location: "body.current", Message: "Enter your current PIN."})
	case errors.Is(err, identity.ErrAddressShape):
		return api.Invalid.Fields(api.FieldError{Location: "body.address", Message: "Start with http:// or https://, then a host name and an optional port, such as https://slipstream.example.com."})
	}
	return err
}
