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
)

// pinChecked are the codes any operation that checks a PIN may answer.
var pinChecked = []*api.Code{pinWrong, pinWait, pinOff, tooManyAttempts}

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
	switch {
	case errors.Is(err, identity.ErrPINWrong):
		return pinWrong
	case errors.Is(err, identity.ErrPINOff):
		return pinOff
	case errors.Is(err, identity.ErrAddressIP):
		return addressIP
	case errors.Is(err, identity.ErrAddressShape):
		return api.Invalid.Fields(api.FieldError{Location: "body.address", Message: "expected http:// or https:// then a host name and an optional port"})
	}
	return err
}
