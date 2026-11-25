// Package reachabilityapi is S51 (Reachability diagnostic)'s operations:
// Remote access, its Exposures and suggestions, Run again, each Node's
// Network section and both settings panels, and the echo a tester fetches.
// Every operation translates HTTP; reachability decides.
package reachabilityapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/reachability"
	"github.com/jatassi/slipstream/internal/reachability/judge"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/storage"
)

// Owner is the Node's Remote access.
type Owner interface {
	// Reachability is Remote access, false unless the Node is a Controller.
	Reachability() (*reachability.Reachability, bool)
}

type handlers struct{ n Owner }

func (h handlers) open() *reachability.Reachability {
	r, ok := h.n.Reachability()
	if !ok {
		panic("reachabilityapi: an Admin operation ran on a Node that is no Controller")
	}
	return r
}

var (
	exposureExists   = api.NewCode(http.StatusConflict, "exposure_exists", "This is already set up for use away from home")
	publicRequired   = api.NewCode(http.StatusUnprocessableEntity, "public_required", "Enter the address players outside your home use")
	publicNotURL     = api.NewCode(http.StatusUnprocessableEntity, "public_not_url", "Enter an http:// or https:// address with a host name")
	serverNotFound   = api.NewCode(http.StatusNotFound, "server_not_found", "No Media Server has this id")
	exposureNotFound = api.NewCode(http.StatusNotFound, "exposure_not_found", "Nothing set up for use away from home has this id")
	nothingWanted    = api.NewCode(http.StatusConflict, "nothing_wanted", "Nothing is set up for use away from home, so there is nothing to check")
	nodeNotFound     = api.NewCode(http.StatusNotFound, "node_not_found", "No Node has this id")
	echoNotFound     = api.NewCode(http.StatusNotFound, "echo_not_found", "This check has ended")
)

func answer(err error) error {
	switch {
	case errors.Is(err, reachability.ErrExists):
		return exposureExists
	case errors.Is(err, reachability.ErrPublicRequired):
		return publicRequired
	case errors.Is(err, reachability.ErrPublicNotURL):
		return publicNotURL
	case errors.Is(err, reachability.ErrServerNotFound):
		return serverNotFound
	case errors.Is(err, reachability.ErrNotFound):
		return exposureNotFound
	case errors.Is(err, reachability.ErrNothingWanted):
		return nothingWanted
	case errors.Is(err, reachability.ErrNodeNotFound):
		return nodeNotFound
	}
	return err
}

func Routes(r *api.Router, n Owner) {
	o := r.Owner("reachability", api.Public)
	h := handlers{n}

	api.Register(o, api.Admin, api.Op{
		ID: "getRemoteAccess", Method: http.MethodGet, Path: "/remote-access",
		Summary: "Get Remote access: every route set up for use away from home with its verdict, the suggestions and each Node's class",
		Status:  http.StatusOK,
	}, h.get)
	api.Register(o, api.Admin, api.Op{
		ID: "addExposure", Method: http.MethodPost, Path: "/remote-access/exposures",
		Summary: "Set the UI and portal, or a Media Server, up for use away from home, and check it",
		Status:  http.StatusCreated, Errors: []*api.Code{exposureExists, publicRequired, publicNotURL, serverNotFound},
	}, h.add)
	api.Register(o, api.Admin, api.Op{
		ID: "editExposure", Method: http.MethodPatch, Path: "/remote-access/exposures/{id}",
		Summary: "Change a Media Server's public address, and check it again",
		Status:  http.StatusOK, Errors: []*api.Code{exposureNotFound, publicRequired, publicNotURL},
	}, h.edit)
	api.Register(o, api.Admin, api.Op{
		ID: "removeExposure", Method: http.MethodDelete, Path: "/remote-access/exposures/{id}",
		Summary: "Stop checking a route, and forget its result",
		Status:  http.StatusNoContent, Errors: []*api.Code{exposureNotFound},
	}, h.remove)
	api.Register(o, api.Admin, api.Op{
		ID: "dismissSuggestion", Method: http.MethodPost, Path: "/remote-access/suggestions/{id}/dismiss",
		Summary: "Hide a suggestion: ui-portal, or a Media Server's id",
		Status:  http.StatusNoContent, Errors: []*api.Code{serverNotFound},
	}, h.dismiss)
	api.Register(o, api.Admin, api.Op{
		ID: "runRemoteAccess", Method: http.MethodPost, Path: "/remote-access/run",
		Summary: "Check every route again, or one, with a fresh look at each Node's network",
		Status:  http.StatusAccepted, Errors: []*api.Code{nothingWanted, exposureNotFound},
	}, h.run)
	api.Register(o, api.Admin, api.Op{
		ID: "getNodeNetwork", Method: http.MethodGet, Path: "/remote-access/nodes/{id}",
		Summary: "Get a Node's Network section, read now with nothing sent outside the machine",
		Status:  http.StatusOK, Errors: []*api.Code{nodeNotFound},
	}, h.node)
	api.Register(o, api.Admin, api.Op{
		ID: "getNodeNetworkSettings", Method: http.MethodGet, Path: "/remote-access/nodes/{id}/settings",
		Summary: "Get a Node's Remote access settings",
		Status:  http.StatusOK, Errors: []*api.Code{nodeNotFound},
	}, h.nodeSettings)
	api.Register(o, api.Admin, api.Op{
		ID: "patchNodeNetworkSettings", Method: http.MethodPatch, Path: "/remote-access/nodes/{id}/settings",
		Summary: "Change a Node's Remote access settings by key, all or none",
		Status:  http.StatusOK, Errors: []*api.Code{nodeNotFound, settingapi.ErrOutOfRange, settingapi.ErrUnknown},
	}, h.patchNodeSettings)
	api.Register(o, api.Admin, api.Op{
		ID: "getRemoteAccessSettings", Method: http.MethodGet, Path: "/remote-access/settings",
		Summary: "Get the Remote access settings of Settings > Access",
		Status:  http.StatusOK,
	}, h.settings)
	api.Register(o, api.Admin, api.Op{
		ID: "patchRemoteAccessSettings", Method: http.MethodPatch, Path: "/remote-access/settings",
		Summary: "Change the Remote access settings by key, all or none",
		Status:  http.StatusOK, Errors: []*api.Code{settingapi.ErrOutOfRange, settingapi.ErrUnknown},
	}, h.patchSettings)
	api.RegisterDownload(o, api.Public, api.Op{
		ID: "remoteAccessEcho", Method: http.MethodGet, Path: "/remote-access/echo/{id}",
		Summary: "Answer a route test's single-use check once, so the Controller sees the test arrived from outside",
		Status:  http.StatusOK, Errors: []*api.Code{echoNotFound},
	}, "text/plain; charset=utf-8", h.echo)
}

type remoteAccessOutput struct{ Body remoteAccess }

func (h handlers) get(ctx context.Context, admin identity.Admin, _ *struct{}) (*remoteAccessOutput, error) {
	p, err := h.open().RemoteAccess(ctx, admin)
	if err != nil {
		return nil, err
	}
	return &remoteAccessOutput{Body: pageOf(p)}, nil
}

type newExposure struct {
	Kind     string `json:"kind" enum:"ui-portal,media-server"`
	ServerID string `json:"serverId,omitempty" format:"uuid" doc:"The Media Server, for kind media-server"`
	Public   string `json:"public,omitempty" doc:"The address players outside the home use, for kind media-server"`
}

type addInput struct{ Body newExposure }

type exposureOutput struct{ Body exposureView }

func (h handlers) add(ctx context.Context, admin identity.Admin, in *addInput) (*exposureOutput, error) {
	kind, _ := reachability.ParseKind(in.Body.Kind)
	w := reachability.Wanted{Kind: kind, Public: in.Body.Public}
	if kind == judge.MediaServer {
		id, err := storage.ParseUUID(in.Body.ServerID)
		if err != nil {
			return nil, serverNotFound
		}
		w.Server = link.ServerID{UUID: id}
	}
	e, err := h.open().Want(ctx, admin, w)
	if err != nil {
		return nil, answer(err)
	}
	return &exposureOutput{Body: exposureViewOf(e.Exposure, e.Subject)}, nil
}

type idInput struct {
	ID string `path:"id" format:"uuid"`
}

func exposureID(s string) (judge.ExposureID, error) {
	id, err := storage.ParseUUID(s)
	if err != nil {
		return judge.ExposureID{}, exposureNotFound
	}
	return judge.ExposureID{UUID: id}, nil
}

type exposurePublic struct {
	Public string `json:"public" doc:"The address players outside the home use"`
}

type editInput struct {
	ID   string `path:"id" format:"uuid"`
	Body exposurePublic
}

func (h handlers) edit(ctx context.Context, admin identity.Admin, in *editInput) (*exposureOutput, error) {
	id, err := exposureID(in.ID)
	if err != nil {
		return nil, err
	}
	e, err := h.open().SetPublic(ctx, admin, id, in.Body.Public)
	if err != nil {
		return nil, answer(err)
	}
	return &exposureOutput{Body: exposureViewOf(e.Exposure, e.Subject)}, nil
}

func (h handlers) remove(ctx context.Context, admin identity.Admin, in *idInput) (*struct{}, error) {
	id, err := exposureID(in.ID)
	if err != nil {
		return nil, err
	}
	return nil, answer(h.open().Unwant(ctx, admin, id))
}

type dismissInput struct {
	ID string `path:"id" doc:"ui-portal, or a Media Server's id"`
}

func (h handlers) dismiss(ctx context.Context, admin identity.Admin, in *dismissInput) (*struct{}, error) {
	var server link.ServerID
	if in.ID != reachability.KindName(judge.UIPortal) {
		id, err := storage.ParseUUID(in.ID)
		if err != nil {
			return nil, serverNotFound
		}
		server = link.ServerID{UUID: id}
	}
	return nil, answer(h.open().Dismiss(ctx, admin, server))
}

type runInput struct {
	Exposure string `query:"exposure" format:"uuid" doc:"Left out, every route"`
}

type runStarted struct {
	RunID int64 `json:"runId" doc:"The Scheduled task run the routes were queued in"`
}

type runOutput struct{ Body runStarted }

func (h handlers) run(ctx context.Context, admin identity.Admin, in *runInput) (*runOutput, error) {
	var only *judge.ExposureID
	if in.Exposure != "" {
		id, err := exposureID(in.Exposure)
		if err != nil {
			return nil, err
		}
		only = &id
	}
	var by storage.UUID
	if account, ok := admin.Account(); ok {
		by = account.Profile().ID.UUID
	}
	id, err := h.open().RunNow(ctx, by, only)
	if err != nil {
		return nil, answer(err)
	}
	return &runOutput{Body: runStarted{RunID: int64(id)}}, nil
}

func nodeID(s string) (role.NodeID, error) {
	id, err := storage.ParseUUID(s)
	if err != nil {
		return role.NodeID{}, nodeNotFound
	}
	return role.NodeID{UUID: id}, nil
}

type nodeNetworkOutput struct{ Body nodeNetwork }

func (h handlers) node(ctx context.Context, admin identity.Admin, in *idInput) (*nodeNetworkOutput, error) {
	id, err := nodeID(in.ID)
	if err != nil {
		return nil, err
	}
	n, err := h.open().NodeNetwork(ctx, admin, id)
	if err != nil {
		return nil, answer(err)
	}
	return &nodeNetworkOutput{Body: nodeNetworkOf(n)}, nil
}

type settingsBody struct {
	Settings []api.OneOf[settingapi.Field] `json:"settings" nullable:"false"`
}

type settingsOutput struct{ Body settingsBody }

type settingsValues struct {
	Values map[string]json.RawMessage `json:"values" doc:"Each setting's new value by key, all or none; null resets one to its default"`
}

type settingsInput struct{ Body settingsValues }

type nodeSettingsInput struct {
	ID   string `path:"id" format:"uuid"`
	Body settingsValues
}

func settingsOf(ctx context.Context, p setting.Panel) (*settingsOutput, error) {
	fields, err := p.Fields(ctx)
	if err != nil {
		return nil, answer(err)
	}
	return &settingsOutput{Body: settingsBody{Settings: settingapi.FieldsOf(fields)}}, nil
}

func (h handlers) nodeSettings(ctx context.Context, _ identity.Admin, in *idInput) (*settingsOutput, error) {
	id, err := nodeID(in.ID)
	if err != nil {
		return nil, err
	}
	return settingsOf(ctx, h.open().NodePanel(id))
}

func (h handlers) patchNodeSettings(ctx context.Context, _ identity.Admin, in *nodeSettingsInput) (*settingsOutput, error) {
	id, err := nodeID(in.ID)
	if err != nil {
		return nil, err
	}
	p := h.open().NodePanel(id)
	if err := p.Apply(ctx, in.Body.Values); err != nil {
		if errors.Is(err, reachability.ErrNodeNotFound) {
			return nil, nodeNotFound
		}
		return nil, settingapi.Refused(err)
	}
	return settingsOf(ctx, p)
}

func (h handlers) settings(ctx context.Context, _ identity.Admin, _ *struct{}) (*settingsOutput, error) {
	return settingsOf(ctx, h.open().Panel())
}

func (h handlers) patchSettings(ctx context.Context, _ identity.Admin, in *settingsInput) (*settingsOutput, error) {
	p := h.open().Panel()
	if err := p.Apply(ctx, in.Body.Values); err != nil {
		return nil, settingapi.Refused(err)
	}
	return settingsOf(ctx, p)
}

type echoInput struct {
	ID string `path:"id" pattern:"^[0-9a-f]{32}$"`
}

// echo answers on every Node: one with no Remote access answers that the
// check has ended, never a panic.
func (h handlers) echo(ctx context.Context, _ identity.Anyone, in *echoInput) (*api.Download, error) {
	r, ok := h.n.Reachability()
	if !ok {
		return nil, echoNotFound
	}
	body, ok := r.Echo(ctx, in.ID, api.ArrivalOf(ctx))
	if !ok {
		return nil, echoNotFound
	}
	return &api.Download{Name: "echo.txt", Size: int64(len(body)), Body: io.NopCloser(bytes.NewReader([]byte(body)))}, nil
}
