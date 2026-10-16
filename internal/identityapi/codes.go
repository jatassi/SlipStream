package identityapi

import (
	"errors"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
)

var (
	pinWrong          = api.NewCode(http.StatusUnauthorized, "pin_wrong", "The email or PIN is wrong")
	pinWait           = api.NewCode(http.StatusTooManyRequests, "pin_wait", "Too many wrong PINs. Wait before trying again.")
	pinOff            = api.NewCode(http.StatusForbidden, "pin_off", "This PIN is turned off. Use a passkey or ask an Admin for a Reset link.")
	tooManyAttempts   = api.NewCode(http.StatusTooManyRequests, "too_many_attempts", "Too many failed attempts from this address. Wait before trying again.")
	settingOutOfRange = api.NewCode(http.StatusUnprocessableEntity, "setting_out_of_range", "A setting is outside its range")
	addressIP         = api.NewCode(http.StatusUnprocessableEntity, "address_ip", "The address must be a name, not an IP address")
	accountDisabled   = api.NewCode(http.StatusForbidden, "account_disabled", "This Account is turned off. Ask an Admin to turn it back on.")
	addressUnset      = api.NewCode(http.StatusConflict, "address_unset", "Set SlipStream's address first, so links point somewhere people can reach")
	accountExists     = api.NewCode(http.StatusConflict, "account_exists", "An Account already has that email")
	emailTaken        = api.NewCode(http.StatusConflict, "email_taken", "Another Account has that email")
	lastAdmin         = api.NewCode(http.StatusConflict, "last_admin", "SlipStream needs at least one Admin who is turned on")
	linkDead          = api.NewCode(http.StatusGone, "link_dead", "This link no longer works")
	linkNotFound      = api.NewCode(http.StatusNotFound, "link_not_found", "No pending Signup link has that id")
	accountNotFound   = api.NewCode(http.StatusNotFound, "account_not_found", "No Account has that id")
	sessionNotFound   = api.NewCode(http.StatusNotFound, "session_not_found", "The Account has no session with that id")
)

// pinChecked are the codes any operation that checks a PIN may answer.
var pinChecked = []*api.Code{pinWrong, pinWait, pinOff, tooManyAttempts}

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
	if out, ok := errors.AsType[*identity.OutOfRangeError](err); ok {
		fields := make([]api.FieldError, len(out.Fields))
		for i, f := range out.Fields {
			fields[i] = api.FieldError{Location: "body." + f.Field, Message: f.Message}
		}
		return settingOutOfRange.Fields(fields...)
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
		return api.Invalid.Fields(api.FieldError{Location: "body.current", Message: "expected the current PIN"})
	case errors.Is(err, identity.ErrAddressShape):
		return api.Invalid.Fields(api.FieldError{Location: "body.address", Message: "expected http:// or https:// then a host name and an optional port"})
	}
	return err
}
