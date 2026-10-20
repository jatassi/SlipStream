package metadataapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/works"
)

type Registry interface {
	// Metadata is the Controller's metadata, false unless the Node is a
	// Controller.
	Metadata() (*metadata.Metadata, bool)
}

func open(r Registry) *metadata.Metadata {
	m, ok := r.Metadata()
	if !ok {
		panic("metadataapi: an operation ran on a Node that is no Controller")
	}
	return m
}

var (
	providerNotFound    = api.NewCode(http.StatusNotFound, "metadata_provider_not_found", "No metadata provider has that id")
	keyRefused          = api.NewCode(http.StatusUnprocessableEntity, "metadata_key_refused", "The metadata provider refused that key, so SlipStream didn't save it")
	metadataUnavailable = api.NewCode(http.StatusBadGateway, "metadata_unavailable", "The metadata source didn't answer. Try again later.")
	artworkUnavailable  = api.NewCode(http.StatusNotFound, "artwork_unavailable", "SlipStream has no such image to show")
)

func answer(err error) error {
	se, isSource := errors.AsType[*works.SourceError](err)
	switch {
	case errors.Is(err, metadata.ErrNoProvider):
		return providerNotFound
	case isSource && errors.Is(err, metadata.ErrKeyRefused):
		return keyRefused.Because(se.Source + " " + se.Words)
	case isSource && errors.Is(err, works.ErrUnavailable):
		return metadataUnavailable.Because(se.Error())
	}
	return err
}

var keys = map[metadata.KeyInUse]string{
	metadata.KeyNone:    "none",
	metadata.KeyBundled: "bundled",
	metadata.KeyYours:   "yours",
}

var states = map[metadata.State]string{
	metadata.StateUnused:         "unused",
	metadata.StateWorking:        "working",
	metadata.StateKeyRefused:     "key_refused",
	metadata.StateRateLimited:    "rate_limited",
	metadata.StateOutOfAllowance: "out_of_allowance",
	metadata.StateUnreachable:    "unreachable",
}

type provider struct {
	ID      string                  `json:"id" doc:"The provider, such as tmdb"`
	Label   string                  `json:"label"`
	Summary string                  `json:"summary" doc:"How the provider stands and what to do, in plain words: the sentence a page leads with, the rest under Details"`
	Key     string                  `json:"key" enum:"none,bundled,yours" doc:"Whose key calls carry: none in this build, SlipStream's, or the Admin's"`
	State   string                  `json:"state" enum:"unused,working,key_refused,rate_limited,out_of_allowance,unreachable" doc:"From the latest outcome of a call or test, unused until one"`
	Until   api.Nullable[time.Time] `json:"until" doc:"When a rate limited provider, or one out of its daily allowance, takes calls again"`
	Details details                 `json:"details"`
}

type details struct {
	LastSuccess api.Nullable[time.Time] `json:"lastSuccess"`
	LastError   api.Nullable[lastError] `json:"lastError"`
	Nodes       []reach                 `json:"nodes" nullable:"false" doc:"Each holder that called the provider since the Controller started or the key changed, by id"`
}

type lastError struct {
	Words string    `json:"words" doc:"In plain words, the provider named"`
	At    time.Time `json:"at"`
}

type reach struct {
	NodeID  string    `json:"nodeId" format:"uuid"`
	Reached bool      `json:"reached" doc:"Whether the Node reached the provider on its latest call"`
	At      time.Time `json:"at"`
}

func providerOf(s metadata.ProviderStatus) provider {
	out := provider{ID: string(s.Provider), Label: s.Label, Summary: s.Summary, Key: keys[s.Key], State: states[s.State], Details: details{Nodes: []reach{}}}
	if !s.Until.IsZero() {
		out.Until = api.Some(s.Until)
	}
	if !s.LastSuccess.IsZero() {
		out.Details.LastSuccess = api.Some(s.LastSuccess)
	}
	if s.LastError != "" {
		out.Details.LastError = api.Some(lastError{Words: s.LastError, At: s.LastErrorAt})
	}
	for _, n := range s.Nodes {
		out.Details.Nodes = append(out.Details.Nodes, reach{NodeID: n.Node.String(), Reached: n.Reached, At: n.At})
	}
	return out
}

type providerList struct {
	Providers []provider `json:"providers" nullable:"false" doc:"In display order"`
}

type providerListOutput struct{ Body providerList }

type providerOutput struct{ Body provider }

type providerInput struct {
	Provider string `path:"provider"`
}

type newKey struct {
	Key string `json:"key" minLength:"1" maxLength:"256"`
}

type keyInput struct {
	Provider string `path:"provider"`
	Body     newKey
}

type settings struct {
	Settings []api.OneOf[settingapi.Field] `json:"settings" nullable:"false" doc:"Language and region first, then the cache, OMDb's ratings and the artwork store"`
}

type settingsOutput struct{ Body settings }

type settingsInput struct{ Body settingsChange }

type settingsChange struct {
	Values map[string]json.RawMessage `json:"values" doc:"Each setting's new value by key, all or none; null resets one to its default. A new language or region makes every Work due at the next Metadata refresh."`
}

func Routes(r *api.Router, registry Registry) {
	o := r.Owner("metadata", api.Account)
	h := handlers{registry}

	api.Register(o, api.Admin, api.Op{
		ID:      "listMetadataProviders",
		Method:  http.MethodGet,
		Path:    "/metadata/providers",
		Summary: "List each metadata provider's status and whose key it is called with",
		Status:  http.StatusOK,
	}, h.listProviders)

	api.Register(o, api.Admin, api.Op{
		ID:      "setMetadataKey",
		Method:  http.MethodPut,
		Path:    "/metadata/providers/{provider}/key",
		Summary: "Use your own key for a provider, once the provider takes it on a test call",
		Status:  http.StatusOK,
		Errors:  []*api.Code{providerNotFound, keyRefused, metadataUnavailable},
	}, h.setKey)

	api.Register(o, api.Admin, api.Op{
		ID:      "clearMetadataKey",
		Method:  http.MethodDelete,
		Path:    "/metadata/providers/{provider}/key",
		Summary: "Remove your key for a provider and go back to SlipStream's",
		Status:  http.StatusOK,
		Errors:  []*api.Code{providerNotFound},
	}, h.clearKey)

	api.Register(o, api.Admin, api.Op{
		ID:      "testMetadataProvider",
		Method:  http.MethodPost,
		Path:    "/metadata/providers/{provider}/test",
		Summary: "Make one cheap call to a provider with the key in use, and answer its status after it",
		Status:  http.StatusOK,
		Errors:  []*api.Code{providerNotFound, metadataUnavailable},
	}, h.test)

	api.Register(o, api.Admin, api.Op{
		ID:      "getMetadataSettings",
		Method:  http.MethodGet,
		Path:    "/metadata/settings",
		Summary: "Get the metadata settings, with the language and region options",
		Status:  http.StatusOK,
	}, h.getSettings)

	api.Register(o, api.Admin, api.Op{
		ID:      "updateMetadataSettings",
		Method:  http.MethodPatch,
		Path:    "/metadata/settings",
		Summary: "Change metadata settings by key, all or none",
		Status:  http.StatusOK,
		Errors:  []*api.Code{settingapi.ErrOutOfRange, settingapi.ErrUnknown},
	}, h.updateSettings)

	api.Register(o, api.Admin, api.Op{
		ID:      "clearMetadataCache",
		Method:  http.MethodDelete,
		Path:    "/metadata/cache",
		Summary: "Have every Node reading metadata drop what it cached, the next time it reads",
		Status:  http.StatusNoContent,
	}, h.clearCache)

	api.RegisterImage(o, api.Account, api.Op{
		ID:      "getArtwork",
		Method:  http.MethodGet,
		Path:    "/artwork/{token}",
		Summary: "Get an image SlipStream handed out, at a size; the address never changes what it answers",
		Status:  http.StatusOK,
		Errors:  []*api.Code{artworkUnavailable},
	}, h.artwork)
}

type handlers struct{ registry Registry }

func (h handlers) listProviders(ctx context.Context, by identity.Admin, _ *struct{}) (*providerListOutput, error) {
	out := providerList{Providers: []provider{}}
	for _, s := range open(h.registry).Providers(ctx, by) {
		out.Providers = append(out.Providers, providerOf(s))
	}
	return &providerListOutput{Body: out}, nil
}

func (h handlers) setKey(ctx context.Context, by identity.Admin, in *keyInput) (*providerOutput, error) {
	s, err := open(h.registry).SetKey(ctx, by, metadata.ProviderID(in.Provider), in.Body.Key)
	if err != nil {
		return nil, answer(err)
	}
	return &providerOutput{Body: providerOf(s)}, nil
}

func (h handlers) clearKey(ctx context.Context, by identity.Admin, in *providerInput) (*providerOutput, error) {
	s, err := open(h.registry).ClearKey(ctx, by, metadata.ProviderID(in.Provider))
	if err != nil {
		return nil, answer(err)
	}
	return &providerOutput{Body: providerOf(s)}, nil
}

func (h handlers) test(ctx context.Context, by identity.Admin, in *providerInput) (*providerOutput, error) {
	s, err := open(h.registry).Test(ctx, by, metadata.ProviderID(in.Provider))
	if err != nil {
		return nil, answer(err)
	}
	return &providerOutput{Body: providerOf(s)}, nil
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

func (h handlers) clearCache(_ context.Context, by identity.Admin, _ *struct{}) (*struct{}, error) {
	open(h.registry).ClearCache(by)
	return nil, nil
}

type artworkInput struct {
	Token string `path:"token" doc:"From an artwork URL the API answered"`
	Size  string `query:"size" required:"true" enum:"small,medium,large,original"`
}

func (h handlers) artwork(ctx context.Context, _ identity.Account, in *artworkInput) (*api.Image, error) {
	img, err := open(h.registry).Artwork(ctx, in.Token, artwork.Size(in.Size))
	if errors.Is(err, artwork.ErrForged) || errors.Is(err, artwork.ErrMissing) {
		return nil, artworkUnavailable.Because(err.Error())
	}
	if err != nil {
		return nil, err
	}
	return &api.Image{Type: img.Type, Body: img.Body}, nil
}
