package grabsapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/settingapi"
)

type searchSettings struct {
	Settings []api.OneOf[settingapi.Field] `json:"settings" nullable:"false" doc:"Backlog search and how often it runs, RSS sync, how long a search waits for each Indexer and when SlipStream stops searching again after failed Grabs"`
}

type searchSettingsOutput struct{ Body searchSettings }

type searchSettingsInput struct {
	Body struct {
		Values map[string]json.RawMessage `json:"values" doc:"Each setting's new value by key, all or none; null resets one to its default"`
	}
}

var getSearchSettingsOp = api.Op{
	ID:      "getSearchSettings",
	Method:  http.MethodGet,
	Path:    "/search-settings",
	Summary: "Get the Automatic search settings",
	Status:  http.StatusOK,
}

var updateSearchSettingsOp = api.Op{
	ID:      "updateSearchSettings",
	Method:  http.MethodPatch,
	Path:    "/search-settings",
	Summary: "Change Automatic search settings by key, all or none",
	Status:  http.StatusOK,
	Errors:  []*api.Code{settingapi.ErrOutOfRange, settingapi.ErrUnknown, gapTooShort},
}

var gapTooShort = api.NewCode(http.StatusUnprocessableEntity, "longest_gap_too_short", "The longest gap is shorter than the Backlog search interval")

func (h handlers) getSearchSettings(ctx context.Context, _ identity.Admin, _ *struct{}) (*searchSettingsOutput, error) {
	return searchSettingsOf(ctx, open(h.p).Settings())
}

func (h handlers) updateSearchSettings(ctx context.Context, _ identity.Admin, in *searchSettingsInput) (*searchSettingsOutput, error) {
	p := open(h.p).Settings()
	if err := p.Apply(ctx, in.Body.Values); err != nil {
		if short, ok := errors.AsType[*grabs.GapTooShortError](err); ok {
			return nil, gapTooShort.On("values."+short.Key, short.Words())
		}
		return nil, settingapi.Refused(err)
	}
	return searchSettingsOf(ctx, p)
}

func searchSettingsOf(ctx context.Context, p setting.Panel) (*searchSettingsOutput, error) {
	fields, err := p.Fields(ctx)
	if err != nil {
		return nil, err
	}
	return &searchSettingsOutput{Body: searchSettings{Settings: settingapi.FieldsOf(fields)}}, nil
}
