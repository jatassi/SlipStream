package identityapi

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
)

type settings struct {
	PINWaitAfter              int64  `json:"pinWaitAfter" doc:"Wrong PINs before a wait"`
	PINWaitSeconds            int64  `json:"pinWaitSeconds" doc:"The wait after too many wrong PINs"`
	PINOffAfter               int64  `json:"pinOffAfter" doc:"Wrong PINs in a row before the PIN turns off"`
	AddressFailures           int64  `json:"addressFailures" doc:"Failed sign-in attempts one address may make in the window"`
	AddressWindowSeconds      int64  `json:"addressWindowSeconds"`
	ReconfirmWindowSeconds    int64  `json:"reconfirmWindowSeconds" doc:"How long a confirm with the PIN or a passkey lasts"`
	SessionLifetimeSeconds    int64  `json:"sessionLifetimeSeconds" doc:"How long a session lasts unused"`
	SignupLinkLifetimeSeconds int64  `json:"signupLinkLifetimeSeconds"`
	ResetLinkLifetimeSeconds  int64  `json:"resetLinkLifetimeSeconds"`
	EmailLinkLifetimeSeconds  int64  `json:"emailLinkLifetimeSeconds"`
	Address                   string `json:"address" doc:"The origin links are built on, empty until set"`
	SuggestedAddress          string `json:"suggestedAddress" doc:"The address this request reached SlipStream at, or this machine's name when that was an IP address"`
}

type settingsPatch struct {
	PINWaitAfter              *int64  `json:"pinWaitAfter,omitempty"`
	PINWaitSeconds            *int64  `json:"pinWaitSeconds,omitempty"`
	PINOffAfter               *int64  `json:"pinOffAfter,omitempty"`
	AddressFailures           *int64  `json:"addressFailures,omitempty"`
	AddressWindowSeconds      *int64  `json:"addressWindowSeconds,omitempty"`
	ReconfirmWindowSeconds    *int64  `json:"reconfirmWindowSeconds,omitempty"`
	SessionLifetimeSeconds    *int64  `json:"sessionLifetimeSeconds,omitempty"`
	SignupLinkLifetimeSeconds *int64  `json:"signupLinkLifetimeSeconds,omitempty"`
	ResetLinkLifetimeSeconds  *int64  `json:"resetLinkLifetimeSeconds,omitempty"`
	EmailLinkLifetimeSeconds  *int64  `json:"emailLinkLifetimeSeconds,omitempty"`
	Address                   *string `json:"address,omitempty" maxLength:"2048"`
}

type settingsInput struct{ Body settingsPatch }

type settingsOutput struct{ Body settings }

func secondsOf(d time.Duration) int64 { return int64(d / time.Second) }

func settingsOf(s identity.Settings, suggested string) *settingsOutput {
	return &settingsOutput{Body: settings{
		PINWaitAfter:              int64(s.PINWaitAfter),
		PINWaitSeconds:            secondsOf(s.PINWait),
		PINOffAfter:               int64(s.PINOffAfter),
		AddressFailures:           int64(s.AddressFailures),
		AddressWindowSeconds:      secondsOf(s.AddressWindow),
		ReconfirmWindowSeconds:    secondsOf(s.ReconfirmWindow),
		SessionLifetimeSeconds:    secondsOf(s.SessionLifetime),
		SignupLinkLifetimeSeconds: secondsOf(s.SignupLinkLifetime),
		ResetLinkLifetimeSeconds:  secondsOf(s.ResetLinkLifetime),
		EmailLinkLifetimeSeconds:  secondsOf(s.EmailLinkLifetime),
		Address:                   s.Address.String(),
		SuggestedAddress:          suggested,
	}}
}

func (p settingsPatch) apply(s *identity.Settings, address *identity.Address) {
	count := func(dst *int, v *int64) {
		if v != nil {
			*dst = int(*v)
		}
	}
	duration := func(dst *time.Duration, v *int64) {
		if v != nil {
			*dst = time.Duration(*v) * time.Second
		}
	}
	count(&s.PINWaitAfter, p.PINWaitAfter)
	duration(&s.PINWait, p.PINWaitSeconds)
	count(&s.PINOffAfter, p.PINOffAfter)
	count(&s.AddressFailures, p.AddressFailures)
	duration(&s.AddressWindow, p.AddressWindowSeconds)
	duration(&s.ReconfirmWindow, p.ReconfirmWindowSeconds)
	duration(&s.SessionLifetime, p.SessionLifetimeSeconds)
	duration(&s.SignupLinkLifetime, p.SignupLinkLifetimeSeconds)
	duration(&s.ResetLinkLifetime, p.ResetLinkLifetimeSeconds)
	duration(&s.EmailLinkLifetime, p.EmailLinkLifetimeSeconds)
	if address != nil {
		s.Address = *address
	}
}

func getSettings(lib Library) func(context.Context, identity.Admin, *struct{}) (*settingsOutput, error) {
	return func(ctx context.Context, a identity.Admin, _ *struct{}) (*settingsOutput, error) {
		s, err := open(lib).Settings(ctx, a)
		if err != nil {
			return nil, err
		}
		return settingsOf(s, a.SuggestedAddress()), nil
	}
}

func updateSettings(lib Library) func(context.Context, identity.Admin, *settingsInput) (*settingsOutput, error) {
	return func(ctx context.Context, a identity.Admin, in *settingsInput) (*settingsOutput, error) {
		var address *identity.Address
		if in.Body.Address != nil {
			parsed, err := identity.ParseAddress(*in.Body.Address)
			if err != nil {
				return nil, answer(err)
			}
			address = &parsed
		}
		s, err := open(lib).UpdateSettings(ctx, a, func(s *identity.Settings) { in.Body.apply(s, address) })
		if err != nil {
			return nil, answer(err)
		}
		return settingsOf(s, a.SuggestedAddress()), nil
	}
}
