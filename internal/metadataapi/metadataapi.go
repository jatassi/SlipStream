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
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

type Registry interface {
	// Metadata is the Controller's metadata, false unless the Node is a
	// Controller.
	Metadata() (*metadata.Metadata, bool)
	// Tasks is the Controller's Scheduled tasks, false unless the Node is a
	// Controller.
	Tasks() (*tasks.Scheduler, bool)
}

func open(r Registry) *metadata.Metadata {
	m, ok := r.Metadata()
	if !ok {
		panic("metadataapi: an operation ran on a Node that is no Controller")
	}
	return m
}

func scheduler(r Registry) *tasks.Scheduler {
	s, ok := r.Tasks()
	if !ok {
		panic("metadataapi: an operation ran on a Node that is no Controller")
	}
	return s
}

var (
	providerNotFound    = api.NewCode(http.StatusNotFound, "metadata_provider_not_found", "No metadata provider has that id")
	keyRefused          = api.NewCode(http.StatusUnprocessableEntity, "metadata_key_refused", "The metadata provider refused that key, so SlipStream didn't save it")
	pinRefused          = api.NewCode(http.StatusUnprocessableEntity, "metadata_pin_refused", "The metadata provider refused that PIN, so SlipStream didn't save it")
	metadataUnavailable = api.NewCode(http.StatusBadGateway, "metadata_unavailable", "The metadata source didn't respond. Try again later.")
	contentTypeNotFound = api.NewCode(http.StatusNotFound, "content_type_not_found", "No Content Type has that id")
	taskUnknown         = api.NewCode(http.StatusNotFound, "task_unknown", "No task has this id")
	artworkUnavailable  = api.NewCode(http.StatusNotFound, "artwork_unavailable", "SlipStream has no such image to show")
)

func answer(err error) error {
	se, isSource := errors.AsType[*works.SourceError](err)
	switch {
	case errors.Is(err, metadata.ErrNoProvider):
		return providerNotFound
	case isSource && errors.Is(err, metadata.ErrKeyRefused):
		return keyRefused.Because(se.Source + " " + se.Words)
	case isSource && errors.Is(err, metadata.ErrPinRefused):
		return pinRefused.Because(se.Source + " " + se.Words + ". Check the PIN on your " + se.Source + " account page.")
	case isSource && errors.Is(err, metadata.ErrTakesNoPin):
		return api.Invalid.Fields(api.FieldError{Location: "body.pin", Message: se.Source + " keys don't use a PIN. Leave the PIN out."})
	case errors.Is(err, works.ErrUnavailable):
		return Unavailable(metadataUnavailable, err)
	case errors.Is(err, works.ErrNoType):
		return contentTypeNotFound
	case errors.Is(err, tasks.ErrUnknownTask):
		return taskUnknown.Because("Metadata refresh starts once the library has something in it")
	}
	return err
}

// Unavailable answers code, a metadata_unavailable code, for err, a call
// that didn't get through to a metadata source or to the Node reading
// metadata.
func Unavailable(code *api.Code, err error) error {
	if holder, ok := errors.AsType[*metadata.HolderError](err); ok {
		return code.Logged(holder, "No Node with the Metadata Role is responding. Check the Nodes page, then try again.")
	}
	se, ok := errors.AsType[*works.SourceError](err)
	switch {
	case !ok:
		return code
	case !se.Until.IsZero():
		return code.At(se.Until, func(at string) string {
			return se.Source + " is rate limiting SlipStream until " + at + ". Try again then."
		})
	}
	return code.Logged(se, se.Source+" isn't available right now. Try again later.")
}

// Refused answers code, a metadata_refused code, for err, a call its
// source refused the key or PIN of.
func Refused(code *api.Code, err error) error {
	se, ok := errors.AsType[*works.SourceError](err)
	switch {
	case !ok:
		return code
	case errors.Is(err, works.ErrPinRefused):
		return code.Because(se.Source + " refused the PIN. Check the PIN in Settings > Metadata.")
	}
	return code.Because(se.Source + " refused the API key. Check the key in Settings > Metadata.")
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
	metadata.StatePinRefused:     "key_refused",
	metadata.StateRateLimited:    "rate_limited",
	metadata.StateOutOfAllowance: "out_of_allowance",
	metadata.StateUnreachable:    "unreachable",
}

type provider struct {
	ID          string                    `json:"id" doc:"The provider, such as tmdb"`
	Label       string                    `json:"label"`
	Summary     string                    `json:"summary" doc:"How the provider stands and what to do, in plain words: the sentence a page leads with, the rest under Details"`
	Key         string                    `json:"key" enum:"none,bundled,yours" doc:"Which key SlipStream sends: none (this build has none), the bundled key or the Admin's own key"`
	State       string                    `json:"state" enum:"unused,working,key_refused,rate_limited,out_of_allowance,unreachable" doc:"The provider's state from its latest call or test; unused until the first one"`
	Until       api.Nullable[time.Time]   `json:"until" doc:"When a provider that is rate limiting SlipStream, or whose daily request limit is used up, takes calls again"`
	KeySetAt    api.Nullable[time.Time]   `json:"keySetAt" doc:"When the Admin set the key in use; null unless it is theirs"`
	KeyPage     string                    `json:"keyPage" format:"uri" doc:"Where an Admin gets a key, opened in the browser"`
	TakesPin    bool                      `json:"takesPin" doc:"Set key offers a subscriber PIN beside the key"`
	PinSet      bool                      `json:"pinSet" doc:"The Admin's key in use carries a PIN"`
	Attribution api.Nullable[attribution] `json:"attribution" doc:"What the provider's terms require SlipStream to show; null when they require nothing"`
	Details     details                   `json:"details"`
}

type attribution struct {
	Text string               `json:"text" doc:"The sentence to show, word for word"`
	Logo api.Nullable[string] `json:"logo" enum:"tmdb" doc:"The logo the web app bundles to show beside it; null when it bundles none, so the text shows alone"`
}

type details struct {
	LastSuccess api.Nullable[time.Time] `json:"lastSuccess"`
	LastError   api.Nullable[lastError] `json:"lastError"`
	Nodes       []reach                 `json:"nodes" nullable:"false" doc:"Each Node that called the provider since the Controller started or the key changed, by id"`
}

type lastError struct {
	Words string    `json:"words" doc:"The last error in plain words, starting with the provider's name"`
	At    time.Time `json:"at"`
}

type reach struct {
	NodeID  string    `json:"nodeId" format:"uuid"`
	Reached bool      `json:"reached" doc:"Whether the Node reached the provider on its latest call"`
	At      time.Time `json:"at"`
}

func providerOf(s metadata.ProviderStatus) provider {
	out := provider{
		ID: string(s.Provider), Label: s.Label, Summary: s.Summary, Key: keys[s.Key], KeyPage: s.KeyPage, TakesPin: s.TakesPin, PinSet: s.PinSet,
		State: states[s.State], Details: details{Nodes: []reach{}},
	}
	if !s.KeySetAt.IsZero() {
		out.KeySetAt = api.Some(s.KeySetAt)
	}
	if a := s.Attribution; a != nil {
		credit := attribution{Text: a.Text}
		if a.Logo != "" {
			credit.Logo = api.Some(a.Logo)
		}
		out.Attribution = api.Some(credit)
	}
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
	Pin string `json:"pin,omitempty" maxLength:"256" doc:"A subscriber PIN to send beside the key, for a provider that takes one; left out, none"`
}

type keyInput struct {
	Provider string `path:"provider"`
	Body     newKey
}

type refreshInput struct {
	ContentType string `path:"contentType"`
}

type refreshStarted struct {
	RunID api.Nullable[int64] `json:"runId" doc:"The Metadata refresh run that refreshes every Work of the type; null while a run is already in progress, and the next run refreshes them"`
}

type refreshStartedOutput struct{ Body refreshStarted }

type refreshAllRun struct {
	LastRunAt api.Nullable[time.Time] `json:"lastRunAt" doc:"When the latest Refresh all of the type ran; null until one has"`
	Running   bool                    `json:"running" doc:"Whether a run that refreshes every Work of the type is in progress"`
}

type refreshAllRunOutput struct{ Body refreshAllRun }

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
		Errors:  []*api.Code{providerNotFound, keyRefused, pinRefused, metadataUnavailable},
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
		Summary: "Make one cheap call to a provider with the key in use, and return the provider's status after it",
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
		Summary: "Clear the metadata cache on every Node that reads metadata",
		Status:  http.StatusNoContent,
	}, h.clearCache)

	api.Register(o, api.Admin, api.Op{
		ID:      "refreshContentType",
		Method:  http.MethodPost,
		Path:    "/content-types/{contentType}/refresh",
		Summary: "Refresh every Work of a Content Type in a Metadata refresh run now, or in the next run if one is in progress",
		Status:  http.StatusAccepted,
		Errors:  []*api.Code{contentTypeNotFound, taskUnknown},
	}, h.refreshAll)

	api.Register(o, api.Admin, api.Op{
		ID:      "getContentTypeRefresh",
		Method:  http.MethodGet,
		Path:    "/content-types/{contentType}/refresh",
		Summary: "Get when every Work of a Content Type was last refreshed together, and whether a run doing it is in progress",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound},
	}, h.getRefreshAll)

	api.RegisterImage(o, api.Account, api.Op{
		ID:      "getArtwork",
		Method:  http.MethodGet,
		Path:    "/artwork/{token}",
		Summary: "Get an artwork image at a size. The image at a URL never changes, so clients can cache it",
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
	s, err := open(h.registry).SetKey(ctx, by, metadata.ProviderID(in.Provider), metadata.NewKey{Key: in.Body.Key, Pin: in.Body.Pin})
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

func (h handlers) refreshAll(ctx context.Context, by identity.Admin, in *refreshInput) (*refreshStartedOutput, error) {
	if err := open(h.registry).RefreshAll(ctx, by, works.TypeID(in.ContentType)); err != nil {
		return nil, answer(err)
	}
	var who storage.UUID
	if account, ok := by.Account(); ok {
		who = account.Profile().ID.UUID
	}
	id, err := scheduler(h.registry).RunNow(ctx, metadata.RefreshTaskID, who)
	if _, ok := errors.AsType[*tasks.ErrRunning](err); ok {
		return &refreshStartedOutput{}, nil
	}
	if err != nil {
		return nil, answer(err)
	}
	return &refreshStartedOutput{Body: refreshStarted{RunID: api.Some(int64(id))}}, nil
}

// getRefreshAll reads the run that does the latest Refresh all: the first
// Metadata refresh run started at or after it was asked. Until that run
// starts, none has run since the ask, which is the only one kept.
func (h handlers) getRefreshAll(ctx context.Context, by identity.Admin, in *refreshInput) (*refreshAllRunOutput, error) {
	asked, err := open(h.registry).RefreshAllAt(ctx, by, works.TypeID(in.ContentType))
	if err != nil {
		return nil, answer(err)
	}
	var out refreshAllRun
	if asked.IsZero() {
		return &refreshAllRunOutput{Body: out}, nil
	}
	task, err := scheduler(h.registry).Task(ctx, metadata.RefreshTaskID)
	if errors.Is(err, tasks.ErrUnknownTask) {
		return &refreshAllRunOutput{Body: out}, nil
	}
	if err != nil {
		return nil, err
	}
	for _, run := range task.Runs {
		if run.Started.Before(asked) {
			break
		}
		out = refreshAllRun{LastRunAt: api.Some(run.Started), Running: run.Ended.IsZero()}
	}
	return &refreshAllRunOutput{Body: out}, nil
}

func (h handlers) clearCache(_ context.Context, by identity.Admin, _ *struct{}) (*struct{}, error) {
	open(h.registry).ClearCache(by)
	return nil, nil
}

type artworkInput struct {
	Token string `path:"token" doc:"From an artwork URL in an API response"`
	Size  string `query:"size" required:"true" enum:"small,medium,large,original"`
}

func (h handlers) artwork(ctx context.Context, _ identity.Account, in *artworkInput) (*api.Image, error) {
	img, err := open(h.registry).Artwork(ctx, in.Token, artwork.Size(in.Size))
	if errors.Is(err, artwork.ErrMissing) {
		return nil, artworkUnavailable.Logged(err, "SlipStream couldn't fetch this image. It tries again later.")
	}
	if errors.Is(err, artwork.ErrForged) {
		return nil, artworkUnavailable.Because("SlipStream didn't make this image link. Use an artwork URL from an API response.")
	}
	if err != nil {
		return nil, err
	}
	return &api.Image{Type: img.Type, Body: img.Body}, nil
}
