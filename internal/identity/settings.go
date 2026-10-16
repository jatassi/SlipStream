package identity

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/identity/internal/db"
)

// Settings is identity_settings: every limit and lifetime, and the address
// links are built on. Each is read at use, so a change applies at once.
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

type BoundKind uint8

const (
	Count BoundKind = iota
	// Duration is carried in whole seconds.
	Duration
)

// Bound is one setting's default, floor and ceiling, in the unit the API
// carries it in. The floors keep a setting from locking everyone out and
// the ceilings keep it from loosening ADR 0005 into something guessable.
type Bound struct {
	Field   string
	Kind    BoundKind
	Default int64
	Floor   int64
	Ceiling int64
	get     func(Settings) int64
}

func seconds(d time.Duration) int64 { return int64(d / time.Second) }

// Bounds lists every bounded setting. identity_settings' CHECKs hold the
// same floors and ceilings.
var Bounds = []Bound{
	{"pinWaitAfter", Count, 5, 3, 10, func(s Settings) int64 { return int64(s.PINWaitAfter) }},
	{"pinWaitSeconds", Duration, 900, 60, 86400, func(s Settings) int64 { return seconds(s.PINWait) }},
	{"pinOffAfter", Count, 10, 4, 20, func(s Settings) int64 { return int64(s.PINOffAfter) }},
	{"addressFailures", Count, 30, 10, 1000, func(s Settings) int64 { return int64(s.AddressFailures) }},
	{"addressWindowSeconds", Duration, 900, 60, 86400, func(s Settings) int64 { return seconds(s.AddressWindow) }},
	{"reconfirmWindowSeconds", Duration, 900, 60, 86400, func(s Settings) int64 { return seconds(s.ReconfirmWindow) }},
	{"sessionLifetimeSeconds", Duration, 2592000, 3600, 31536000, func(s Settings) int64 { return seconds(s.SessionLifetime) }},
	{"signupLinkLifetimeSeconds", Duration, 604800, 60, 2592000, func(s Settings) int64 { return seconds(s.SignupLinkLifetime) }},
	{"resetLinkLifetimeSeconds", Duration, 86400, 60, 604800, func(s Settings) int64 { return seconds(s.ResetLinkLifetime) }},
	{"emailLinkLifetimeSeconds", Duration, 86400, 60, 604800, func(s Settings) int64 { return seconds(s.EmailLinkLifetime) }},
}

// FieldError names a field by its name in the API.
type FieldError struct{ Field, Message string }

type OutOfRangeError struct{ Fields []FieldError }

func (e *OutOfRangeError) Error() string {
	parts := make([]string, len(e.Fields))
	for i, f := range e.Fields {
		parts[i] = f.Field + ": " + f.Message
	}
	return "identity: " + strings.Join(parts, "; ")
}

func (s Settings) validate() error {
	var fields []FieldError
	for _, b := range Bounds {
		if v := b.get(s); v < b.Floor || v > b.Ceiling {
			fields = append(fields, FieldError{b.Field, fmt.Sprintf("expected %d to %d", b.Floor, b.Ceiling)})
		}
	}
	if s.PINOffAfter <= s.PINWaitAfter {
		fields = append(fields, FieldError{"pinOffAfter", "expected more than pinWaitAfter"})
	}
	if len(fields) > 0 {
		return &OutOfRangeError{Fields: fields}
	}
	return nil
}

func settingsOf(row db.IdentitySetting) Settings {
	d := func(n int64) time.Duration { return time.Duration(n) * time.Second }
	return Settings{
		PINWaitAfter:       int(row.PinWaitAfter),
		PINWait:            d(row.PinWaitSeconds),
		PINOffAfter:        int(row.PinOffAfter),
		AddressFailures:    int(row.AddressFailures),
		AddressWindow:      d(row.AddressWindowSeconds),
		ReconfirmWindow:    d(row.ReconfirmWindowSeconds),
		SessionLifetime:    d(row.SessionLifetimeSeconds),
		SignupLinkLifetime: d(row.SignupLinkLifetimeSeconds),
		ResetLinkLifetime:  d(row.ResetLinkLifetimeSeconds),
		EmailLinkLifetime:  d(row.EmailLinkLifetimeSeconds),
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

func (s *Service) Settings(ctx context.Context, _ Admin) (Settings, error) {
	return readSettings(ctx, db.New(s.lib))
}

// UpdateSettings applies change to the settings in one Write, refusing the
// result with *OutOfRangeError when a value is outside its Bound.
func (s *Service) UpdateSettings(ctx context.Context, _ Admin, change func(*Settings)) (Settings, error) {
	var out Settings
	err := s.write(ctx, func(ctx context.Context, q *db.Queries, _ *effects) error {
		var err error
		if out, err = readSettings(ctx, q); err != nil {
			return err
		}
		change(&out)
		if err := out.validate(); err != nil {
			return err
		}
		return q.SaveSettings(ctx, db.SaveSettingsParams{
			PinWaitAfter:              int64(out.PINWaitAfter),
			PinWaitSeconds:            seconds(out.PINWait),
			PinOffAfter:               int64(out.PINOffAfter),
			AddressFailures:           int64(out.AddressFailures),
			AddressWindowSeconds:      seconds(out.AddressWindow),
			ReconfirmWindowSeconds:    seconds(out.ReconfirmWindow),
			SessionLifetimeSeconds:    seconds(out.SessionLifetime),
			SignupLinkLifetimeSeconds: seconds(out.SignupLinkLifetime),
			ResetLinkLifetimeSeconds:  seconds(out.ResetLinkLifetime),
			EmailLinkLifetimeSeconds:  seconds(out.EmailLinkLifetime),
			Address:                   out.Address.origin,
		})
	})
	return out, err
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
