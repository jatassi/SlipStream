package indexersapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/settingapi"
)

type settings struct {
	Settings []api.OneOf[settingapi.Field] `json:"settings" nullable:"false" doc:"How long every request to an Indexer may fail before Health warns"`
}

type settingsOutput struct{ Body settings }

type settingsInput struct{ Body settingsChange }

type settingsChange struct {
	Values map[string]json.RawMessage `json:"values" doc:"Each setting's new value by key, all or none; null resets one to its default"`
}

func settingsRoutes(o *api.Owner, h handlers) {
	api.Register(o, api.Admin, api.Op{
		ID: "getIndexerSettings", Method: http.MethodGet, Path: "/indexer-settings", Summary: "Get the Indexer settings", Status: http.StatusOK,
	}, h.getSettings)

	api.Register(o, api.Admin, api.Op{
		ID:      "updateIndexerSettings",
		Method:  http.MethodPatch,
		Path:    "/indexer-settings",
		Summary: "Change Indexer settings by key, all or none",
		Status:  http.StatusOK,
		Errors:  []*api.Code{settingapi.ErrOutOfRange, settingapi.ErrUnknown},
	}, h.updateSettings)
}

func (h handlers) getSettings(ctx context.Context, _ identity.Admin, _ *struct{}) (*settingsOutput, error) {
	return settingsOf(ctx, open(h.registry).Settings())
}

func (h handlers) updateSettings(ctx context.Context, _ identity.Admin, in *settingsInput) (*settingsOutput, error) {
	p := open(h.registry).Settings()
	if err := p.Apply(ctx, in.Body.Values); err != nil {
		return nil, settingapi.Refused(err)
	}
	return settingsOf(ctx, p)
}

func settingsOf(ctx context.Context, p setting.Panel) (*settingsOutput, error) {
	fields, err := p.Fields(ctx)
	if err != nil {
		return nil, err
	}
	return &settingsOutput{Body: settings{Settings: settingapi.FieldsOf(fields)}}, nil
}
