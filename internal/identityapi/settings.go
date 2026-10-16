package identityapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/settingapi"
)

type settings struct {
	PINWaitAfter       setting.CountField    `json:"pinWaitAfter"`
	PINWait            setting.DurationField `json:"pinWait"`
	PINOffAfter        setting.CountField    `json:"pinOffAfter"`
	AddressFailures    setting.CountField    `json:"addressFailures"`
	AddressWindow      setting.DurationField `json:"addressWindow"`
	ReconfirmWindow    setting.DurationField `json:"reconfirmWindow"`
	SessionLifetime    setting.DurationField `json:"sessionLifetime"`
	SignupLinkLifetime setting.DurationField `json:"signupLinkLifetime"`
	ResetLinkLifetime  setting.DurationField `json:"resetLinkLifetime"`
	EmailLinkLifetime  setting.DurationField `json:"emailLinkLifetime"`
}

type settingsPatch struct {
	PINWaitAfter       api.Patch[int64] `json:"pinWaitAfter,omitempty" doc:"null resets it to its default"`
	PINWait            api.Patch[int64] `json:"pinWait,omitempty" doc:"Seconds; null resets it to its default"`
	PINOffAfter        api.Patch[int64] `json:"pinOffAfter,omitempty" doc:"null resets it to its default"`
	AddressFailures    api.Patch[int64] `json:"addressFailures,omitempty" doc:"null resets it to its default"`
	AddressWindow      api.Patch[int64] `json:"addressWindow,omitempty" doc:"Seconds; null resets it to its default"`
	ReconfirmWindow    api.Patch[int64] `json:"reconfirmWindow,omitempty" doc:"Seconds; null resets it to its default"`
	SessionLifetime    api.Patch[int64] `json:"sessionLifetime,omitempty" doc:"Seconds; null resets it to its default"`
	SignupLinkLifetime api.Patch[int64] `json:"signupLinkLifetime,omitempty" doc:"Seconds; null resets it to its default"`
	ResetLinkLifetime  api.Patch[int64] `json:"resetLinkLifetime,omitempty" doc:"Seconds; null resets it to its default"`
	EmailLinkLifetime  api.Patch[int64] `json:"emailLinkLifetime,omitempty" doc:"Seconds; null resets it to its default"`
}

type settingsInput struct{ Body settingsPatch }

type settingsOutput struct{ Body settings }

func settingsOf(o identity.SignInOverrides) *settingsOutput {
	wait, off := identity.PINLimits(o)
	return &settingsOutput{Body: settings{
		PINWaitAfter:       wait.Field(o.PINWaitAfter),
		PINWait:            identity.PINWaitLength.Field(o.PINWait),
		PINOffAfter:        off.Field(o.PINOffAfter),
		AddressFailures:    identity.AddressFailures.Field(o.AddressFailures),
		AddressWindow:      identity.AddressWindow.Field(o.AddressWindow),
		ReconfirmWindow:    identity.ReconfirmWindow.Field(o.ReconfirmWindow),
		SessionLifetime:    identity.SessionLifetime.Field(o.SessionLifetime),
		SignupLinkLifetime: identity.SignupLinkLifetime.Field(o.SignupLinkLifetime),
		ResetLinkLifetime:  identity.ResetLinkLifetime.Field(o.ResetLinkLifetime),
		EmailLinkLifetime:  identity.EmailLinkLifetime.Field(o.EmailLinkLifetime),
	}}
}

func signInChange(b settingsPatch) (identity.SignInChange, error) {
	var c identity.SignInChange
	for _, f := range []struct {
		patch api.Patch[int64]
		parse func(*int64) (setting.Stored[int64], error)
		into  **setting.Stored[int64]
	}{
		{b.PINWaitAfter, identity.PINWaitAfter.Parse, &c.PINWaitAfter},
		{b.PINWait, identity.PINWaitLength.Parse, &c.PINWait},
		{b.PINOffAfter, identity.PINOffAfter.Parse, &c.PINOffAfter},
		{b.AddressFailures, identity.AddressFailures.Parse, &c.AddressFailures},
		{b.AddressWindow, identity.AddressWindow.Parse, &c.AddressWindow},
		{b.ReconfirmWindow, identity.ReconfirmWindow.Parse, &c.ReconfirmWindow},
		{b.SessionLifetime, identity.SessionLifetime.Parse, &c.SessionLifetime},
		{b.SignupLinkLifetime, identity.SignupLinkLifetime.Parse, &c.SignupLinkLifetime},
		{b.ResetLinkLifetime, identity.ResetLinkLifetime.Parse, &c.ResetLinkLifetime},
		{b.EmailLinkLifetime, identity.EmailLinkLifetime.Parse, &c.EmailLinkLifetime},
	} {
		if !f.patch.Sent() {
			continue
		}
		v, err := f.parse(f.patch.Value())
		if err != nil {
			return c, err
		}
		*f.into = &v
	}
	return c, nil
}

func getSettings(lib Library) func(context.Context, identity.Admin, *struct{}) (*settingsOutput, error) {
	return func(ctx context.Context, a identity.Admin, _ *struct{}) (*settingsOutput, error) {
		o, err := open(lib).SignInSettings(ctx, a)
		if err != nil {
			return nil, err
		}
		return settingsOf(o), nil
	}
}

func updateSettings(lib Library) func(context.Context, identity.Admin, *settingsInput) (*settingsOutput, error) {
	return func(ctx context.Context, a identity.Admin, in *settingsInput) (*settingsOutput, error) {
		c, err := signInChange(in.Body)
		if err != nil {
			return nil, settingapi.Refused(err)
		}
		o, err := open(lib).SetSignInSettings(ctx, a, c)
		if err != nil {
			return nil, settingapi.Refused(err)
		}
		return settingsOf(o), nil
	}
}

type address struct {
	Address          string `json:"address" doc:"The origin links are built on, empty until set"`
	SuggestedAddress string `json:"suggestedAddress" doc:"The address this request reached SlipStream at, or this machine's name when that was an IP address"`
}

type addressOutput struct{ Body address }

type addressChange struct {
	Address string `json:"address" maxLength:"2048"`
}

type addressInput struct{ Body addressChange }

func getAddress(lib Library) func(context.Context, identity.Admin, *struct{}) (*addressOutput, error) {
	return func(ctx context.Context, a identity.Admin, _ *struct{}) (*addressOutput, error) {
		current, err := open(lib).Address(ctx, a)
		if err != nil {
			return nil, err
		}
		return &addressOutput{Body: address{Address: current.String(), SuggestedAddress: a.SuggestedAddress()}}, nil
	}
}

func setAddress(lib Library) func(context.Context, identity.Admin, *addressInput) (*addressOutput, error) {
	return func(ctx context.Context, a identity.Admin, in *addressInput) (*addressOutput, error) {
		parsed, err := identity.ParseAddress(in.Body.Address)
		if err != nil {
			return nil, answer(err)
		}
		if err := open(lib).SetAddress(ctx, a, parsed); err != nil {
			return nil, err
		}
		return &addressOutput{Body: address{Address: parsed.String(), SuggestedAddress: a.SuggestedAddress()}}, nil
	}
}
