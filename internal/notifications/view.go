package notifications

import (
	"encoding/json"
	"net/url"
	"time"

	"github.com/jatassi/slipstream/internal/notifications/providers"
)

// Providers is the provider table, in the order the Add sheet lists it.
func (s *Service) Providers() []providers.Provider { return s.cfg.Providers }

// Described is a connection as the API shows it. It holds no secret value.
type Described struct {
	View
	// Sealed are the secret fields that have a value.
	Sealed []string
	// Shown is what of each secret its field lets the API show.
	Shown map[string]string
	// Reason is why it isn't sending, worded now. "" when it is.
	Reason string
}

// Describe opens v's secrets to say which have a value. A seal that won't
// open has none: its line is paused until the Admin enters them again.
func (s *Service) Describe(v View) Described {
	d := Described{View: v, Sealed: []string{}, Shown: map[string]string{}}
	if v.Status.Reason.Code != "" {
		d.Reason = s.wording(time.Now(), 0).reason(s.provider(v.Provider), v.Status.Reason)
	}
	secrets, err := s.open(v.ID, v.Sealed)
	if err != nil {
		return d
	}
	for _, f := range s.provider(v.Provider).Fields {
		raw, ok := secrets[f.Key]
		if !f.Secret || !ok {
			continue
		}
		d.Sealed = append(d.Sealed, f.Key)
		if f.Shown == providers.ShownHost {
			var text string
			if json.Unmarshal(raw, &text) == nil {
				if u, err := url.Parse(text); err == nil && u.Host != "" {
					d.Shown[f.Key] = u.Host
				}
			}
		}
	}
	return d
}
