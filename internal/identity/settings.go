package identity

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/identity/internal/db"
	"github.com/jatassi/slipstream/internal/setting"
)

const day = 24 * time.Hour

var wrongPINs = setting.Noun{One: "wrong PIN", Many: "wrong PINs"}

// The sign-in settings. Each floor keeps a setting from locking everyone
// out and each ceiling keeps it from loosening ADR 0005 into something
// guessable. PINOffAfter's floor is also one more than PINWaitAfter's
// value; PINLimits applies it.
var (
	PINWaitAfter       = setting.Count{Key: "identity.pinWaitAfter", Label: "Wrong PINs before a wait", Default: 5, Min: 3, Max: 10, Unit: wrongPINs}
	PINWaitLength      = setting.Duration{Key: "identity.pinWait", Label: "Wait after too many wrong PINs", Default: 15 * time.Minute, Min: time.Minute, Max: day, Step: time.Minute}
	PINOffAfter        = setting.Count{Key: "identity.pinOffAfter", Label: "Wrong PINs in a row before the PIN turns off", Default: 10, Min: 4, Max: 20, Unit: wrongPINs}
	AddressFailures    = setting.Count{Key: "identity.addressFailures", Label: "Failed attempts per address", Default: 30, Min: 10, Max: 1000, Unit: setting.Noun{One: "attempt", Many: "attempts"}}
	AddressWindow      = setting.Duration{Key: "identity.addressWindow", Label: "Per-address window", Default: 15 * time.Minute, Min: time.Minute, Max: day, Step: time.Minute}
	ReconfirmWindow    = setting.Duration{Key: "identity.reconfirmWindow", Label: "Re-confirm window", Default: 15 * time.Minute, Min: time.Minute, Max: day, Step: time.Minute}
	SessionLifetime    = setting.Duration{Key: "identity.sessionLifetime", Label: "Session lifetime", Default: 30 * day, Min: time.Hour, Max: 365 * day, Step: time.Hour}
	SignupLinkLifetime = setting.Duration{Key: "identity.signupLinkLifetime", Label: "Signup link lifetime", Default: 7 * day, Min: time.Minute, Max: 30 * day, Step: time.Minute}
	ResetLinkLifetime  = setting.Duration{Key: "identity.resetLinkLifetime", Label: "Reset link lifetime", Default: day, Min: time.Minute, Max: 7 * day, Step: time.Minute}
	EmailLinkLifetime  = setting.Duration{Key: "identity.emailLinkLifetime", Label: "Email confirm link lifetime", Default: day, Min: time.Minute, Max: 7 * day, Step: time.Minute}
)

// PINLimits is PINWaitAfter and PINOffAfter narrowed by each other's value
// under o, so the PIN turns off at least one wrong PIN after the wait.
func PINLimits(o SignInOverrides) (wait, off setting.Count) {
	wait, off = PINWaitAfter, PINOffAfter
	off.Min = max(off.Min, PINWaitAfter.Value(o.PINWaitAfter)+1)
	wait.Max = min(wait.Max, PINOffAfter.Value(o.PINOffAfter)-1)
	return wait, off
}

// SignInOverrides is each sign-in setting's stored override, nil at its
// default.
type SignInOverrides struct {
	PINWaitAfter, PINWait, PINOffAfter, AddressFailures, AddressWindow, ReconfirmWindow *int64
	SessionLifetime, SignupLinkLifetime, ResetLinkLifetime, EmailLinkLifetime           *int64
}

// SignInChange is the sign-in settings to store, each nil to leave.
type SignInChange struct {
	PINWaitAfter, PINWait, PINOffAfter, AddressFailures, AddressWindow, ReconfirmWindow *setting.Stored[int64]
	SessionLifetime, SignupLinkLifetime, ResetLinkLifetime, EmailLinkLifetime           *setting.Stored[int64]
}

// Settings is every limit and lifetime's value, and the address links are
// built on. Each is read at use, so a change applies at once.
type Settings struct {
	PINWaitAfter       int
	PINWait            time.Duration
	PINOffAfter        int
	AddressFailures    int
	AddressWindow      time.Duration
	ReconfirmWindow    time.Duration
	SessionLifetime    time.Duration
	SignupLinkLifetime time.Duration
	ResetLinkLifetime  time.Duration
	EmailLinkLifetime  time.Duration
	Address            Address
}

func overridesOf(row db.IdentitySetting) SignInOverrides {
	return SignInOverrides{
		PINWaitAfter: row.PinWaitAfter, PINWait: row.PinWaitSeconds, PINOffAfter: row.PinOffAfter,
		AddressFailures: row.AddressFailures, AddressWindow: row.AddressWindowSeconds, ReconfirmWindow: row.ReconfirmWindowSeconds,
		SessionLifetime: row.SessionLifetimeSeconds, SignupLinkLifetime: row.SignupLinkLifetimeSeconds,
		ResetLinkLifetime: row.ResetLinkLifetimeSeconds, EmailLinkLifetime: row.EmailLinkLifetimeSeconds,
	}
}

func settingsOf(row db.IdentitySetting) Settings {
	return Settings{
		PINWaitAfter:       int(PINWaitAfter.Value(row.PinWaitAfter)),
		PINWait:            PINWaitLength.Value(row.PinWaitSeconds),
		PINOffAfter:        int(PINOffAfter.Value(row.PinOffAfter)),
		AddressFailures:    int(AddressFailures.Value(row.AddressFailures)),
		AddressWindow:      AddressWindow.Value(row.AddressWindowSeconds),
		ReconfirmWindow:    ReconfirmWindow.Value(row.ReconfirmWindowSeconds),
		SessionLifetime:    SessionLifetime.Value(row.SessionLifetimeSeconds),
		SignupLinkLifetime: SignupLinkLifetime.Value(row.SignupLinkLifetimeSeconds),
		ResetLinkLifetime:  ResetLinkLifetime.Value(row.ResetLinkLifetimeSeconds),
		EmailLinkLifetime:  EmailLinkLifetime.Value(row.EmailLinkLifetimeSeconds),
		Address:            Address{origin: row.Address},
	}
}

func readSettings(ctx context.Context, q *db.Queries) (Settings, error) {
	row, err := q.GetSettings(ctx)
	if err != nil {
		return Settings{}, err
	}
	return settingsOf(row), nil
}

func (s *Service) SignInSettings(ctx context.Context, _ Admin) (SignInOverrides, error) {
	row, err := db.New(s.lib).GetSettings(ctx)
	return overridesOf(row), err
}

// SetSignInSettings stores c in one Write, refusing with
// *setting.RangeError a change that breaks PINLimits.
func (s *Service) SetSignInSettings(ctx context.Context, _ Admin, c SignInChange) (SignInOverrides, error) {
	var o SignInOverrides
	err := s.write(ctx, func(ctx context.Context, q *db.Queries, _ *effects) error {
		row, err := q.GetSettings(ctx)
		if err != nil {
			return err
		}
		o = overridesOf(row)
		for _, f := range []struct {
			change   *setting.Stored[int64]
			override **int64
		}{
			{c.PINWaitAfter, &o.PINWaitAfter},
			{c.PINWait, &o.PINWait},
			{c.PINOffAfter, &o.PINOffAfter},
			{c.AddressFailures, &o.AddressFailures},
			{c.AddressWindow, &o.AddressWindow},
			{c.ReconfirmWindow, &o.ReconfirmWindow},
			{c.SessionLifetime, &o.SessionLifetime},
			{c.SignupLinkLifetime, &o.SignupLinkLifetime},
			{c.ResetLinkLifetime, &o.ResetLinkLifetime},
			{c.EmailLinkLifetime, &o.EmailLinkLifetime},
		} {
			if f.change != nil {
				*f.override = f.change.Override()
			}
		}
		if err := pinLimitsHold(o, c.PINOffAfter != nil); err != nil {
			return err
		}
		return q.SetSignInSettings(ctx, db.SetSignInSettingsParams{
			PinWaitAfter: o.PINWaitAfter, PinWaitSeconds: o.PINWait, PinOffAfter: o.PINOffAfter,
			AddressFailures: o.AddressFailures, AddressWindowSeconds: o.AddressWindow, ReconfirmWindowSeconds: o.ReconfirmWindow,
			SessionLifetimeSeconds: o.SessionLifetime, SignupLinkLifetimeSeconds: o.SignupLinkLifetime,
			ResetLinkLifetimeSeconds: o.ResetLinkLifetime, EmailLinkLifetimeSeconds: o.EmailLinkLifetime,
		})
	})
	return o, err
}

// pinLimitsHold refuses o when the PIN would turn off no later than the
// wait, blaming PINOffAfter when the change set it and PINWaitAfter
// otherwise.
func pinLimitsHold(o SignInOverrides, offChanged bool) error {
	wait, off := PINLimits(o)
	waitAfter, offAfter := PINWaitAfter.Value(o.PINWaitAfter), PINOffAfter.Value(o.PINOffAfter)
	if offAfter > waitAfter {
		return nil
	}
	if offChanged {
		_, err := off.Parse(&offAfter)
		return err
	}
	_, err := wait.Parse(&waitAfter)
	return err
}

// Address is the address links are built on, zero until set.
func (s *Service) Address(ctx context.Context, _ Admin) (Address, error) {
	set, err := readSettings(ctx, db.New(s.lib))
	return set.Address, err
}

func (s *Service) SetAddress(ctx context.Context, _ Admin, address Address) error {
	return s.write(ctx, func(ctx context.Context, q *db.Queries, _ *effects) error {
		return q.SetAddress(ctx, address.origin)
	})
}

// Address is the configured name, scheme://host[:port], whose host is a
// name. Signup, Reset and confirm links are built on it and nothing else
// (ADR 0005). The zero Address is unset.
type Address struct{ origin string }

var (
	ErrAddressIP    = errors.New("identity: the address's host is an IP address")
	ErrAddressShape = errors.New("identity: the address is not http:// or https:// then a host name and an optional port")
)

var defaultPorts = map[string]string{"http": "80", "https": "443"}

// ParseAddress reads an origin the way a person types one, any case and
// with or without a default port or a trailing slash, and refuses an IP
// host with ErrAddressIP.
func ParseAddress(s string) (Address, error) {
	u, err := url.Parse(strings.TrimSpace(s))
	if err != nil {
		return Address{}, ErrAddressShape
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())
	if defaultPorts[scheme] == "" || host == "" || u.User != nil || (u.Path != "" && u.Path != "/") ||
		u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || u.ForceQuery {
		return Address{}, ErrAddressShape
	}
	if _, err := netip.ParseAddr(host); err == nil {
		return Address{}, ErrAddressIP
	}
	if port := u.Port(); port != "" && port != defaultPorts[scheme] {
		host = net.JoinHostPort(host, port)
	}
	return Address{origin: scheme + "://" + host}, nil
}

func (a Address) IsZero() bool   { return a.origin == "" }
func (a Address) String() string { return a.origin }

// Host is the address's host without its port, which emails call the
// server's name until it has a display name.
func (a Address) Host() string {
	u, err := url.Parse(a.origin)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// suggestAddress is what Settings > Address offers: the address c reached
// SlipStream at, or this machine's name with that port when c used an IP.
func suggestAddress(c client) string {
	scheme := "http"
	if c.https {
		scheme = "https"
	}
	host, port, err := net.SplitHostPort(c.host)
	if err != nil {
		host, port = c.host, ""
	}
	if _, err := netip.ParseAddr(host); err == nil || host == "" {
		name, err := os.Hostname()
		if err != nil {
			return ""
		}
		host = strings.ToLower(name)
	}
	a, err := ParseAddress(scheme + "://" + net.JoinHostPort(host, port))
	if port == "" {
		a, err = ParseAddress(scheme + "://" + host)
	}
	if err != nil {
		return ""
	}
	return a.String()
}
