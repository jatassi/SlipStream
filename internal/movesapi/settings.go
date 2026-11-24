package movesapi

import (
	"context"
	"encoding/json"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/settingapi"
)

type moveSettings struct {
	Settings []api.OneOf[settingapi.Field] `json:"settings" nullable:"false" doc:"When a waiting switch shows in Needs you, then how long the old files wait for Plex"`
}

type moveSettingsOutput struct{ Body moveSettings }

type moveSettingsValues struct {
	Values map[string]json.RawMessage `json:"values" doc:"Each setting's new value by key, all or none; null resets one to its default"`
}

type moveSettingsInput struct{ Body moveSettingsValues }

func settingsOf(ctx context.Context, p setting.Panel) (*moveSettingsOutput, error) {
	fields, err := p.Fields(ctx)
	if err != nil {
		return nil, err
	}
	return &moveSettingsOutput{Body: moveSettings{Settings: settingapi.FieldsOf(fields)}}, nil
}

func (h handlers) getSettings(ctx context.Context, _ identity.Admin, _ *struct{}) (*moveSettingsOutput, error) {
	return settingsOf(ctx, open(h.n).Settings())
}

func (h handlers) patchSettings(ctx context.Context, _ identity.Admin, in *moveSettingsInput) (*moveSettingsOutput, error) {
	p := open(h.n).Settings()
	if err := p.Apply(ctx, in.Body.Values); err != nil {
		return nil, settingapi.Refused(err)
	}
	return settingsOf(ctx, p)
}
