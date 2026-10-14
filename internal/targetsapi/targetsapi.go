package targetsapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type Goals interface {
	// Targets is the Targets and Profiles, false unless the Node is a
	// Controller.
	Targets() (*targets.Targets, bool)
}

func open(g Goals) *targets.Targets {
	t, ok := g.Targets()
	if !ok {
		panic("targetsapi: an operation ran on a Node that is no Controller")
	}
	return t
}

var (
	targetNameTaken     = api.NewCode(http.StatusConflict, "target_name_taken", "A Target already has that name")
	profileNameTaken    = api.NewCode(http.StatusConflict, "profile_name_taken", "A Profile already has that name")
	contentTypeNotFound = api.NewCode(http.StatusNotFound, "content_type_not_found", "No Content Type has that id")
	targetNotFound      = api.NewCode(http.StatusUnprocessableEntity, "target_not_found", "No Target has that id")
)

var codes = []struct {
	err  error
	code *api.Code
}{
	{targets.ErrTargetNameTaken, targetNameTaken},
	{targets.ErrProfileNameTaken, profileNameTaken},
	{targets.ErrNoType, contentTypeNotFound},
	{targets.ErrNoTarget, targetNotFound},
}

func answer(err error) error {
	for _, c := range codes {
		if errors.Is(err, c.err) {
			return c.code
		}
	}
	return err
}

type target struct {
	ID          string    `json:"id" format:"uuid"`
	Name        string    `json:"name"`
	Resolutions []string  `json:"resolutions" nullable:"false" enum:"2160p,1080p,720p,576p,480p" doc:"Highest ranked first"`
	CreatedAt   time.Time `json:"createdAt"`
}

func targetOf(t targets.Target) target {
	out := target{ID: t.ID.String(), Name: t.Name, Resolutions: []string{}, CreatedAt: t.CreatedAt.Time()}
	for _, r := range t.Resolutions {
		out.Resolutions = append(out.Resolutions, r.String())
	}
	return out
}

type profile struct {
	ID          string   `json:"id" format:"uuid"`
	Name        string   `json:"name"`
	ContentType string   `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Default     bool     `json:"default" doc:"Whether the Content Type's Works use it"`
	TargetIDs   []string `json:"targetIds" nullable:"false" format:"uuid" doc:"Highest ranked first"`
}

func profileOf(p targets.Profile) profile {
	out := profile{ID: p.ID.String(), Name: p.Name, ContentType: string(p.ContentType), Default: p.Default, TargetIDs: []string{}}
	for _, id := range p.Targets {
		out.TargetIDs = append(out.TargetIDs, id.String())
	}
	return out
}

type newTarget struct {
	Name        string   `json:"name" minLength:"1" maxLength:"100"`
	Resolutions []string `json:"resolutions" nullable:"false" minItems:"1" uniqueItems:"true" enum:"2160p,1080p,720p,576p,480p" doc:"Highest ranked first"`
}

type createTargetInput struct{ Body newTarget }

type targetOutput struct{ Body target }

type targetList struct {
	Targets []target `json:"targets" nullable:"false" doc:"By name"`
}

type targetListOutput struct{ Body targetList }

type newProfile struct {
	Name        string   `json:"name" minLength:"1" maxLength:"100"`
	ContentType string   `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	TargetIDs   []string `json:"targetIds" nullable:"false" minItems:"1" uniqueItems:"true" format:"uuid" doc:"Highest ranked first"`
}

type createProfileInput struct{ Body newProfile }

type profileOutput struct{ Body profile }

type profileList struct {
	Profiles []profile `json:"profiles" nullable:"false" doc:"By Content Type, then name"`
}

type profileListOutput struct{ Body profileList }

func Routes(r *api.Router, goals Goals) {
	o := r.Owner("targets", api.Admin)
	h := handlers{goals}

	api.Register(o, api.Admin, api.Op{
		ID:      "createTarget",
		Method:  http.MethodPost,
		Path:    "/targets",
		Summary: "Create a Target from its resolutions",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{targetNameTaken},
	}, h.createTarget)

	api.Register(o, api.Admin, api.Op{
		ID:      "listTargets",
		Method:  http.MethodGet,
		Path:    "/targets",
		Summary: "List the Targets",
		Status:  http.StatusOK,
	}, h.listTargets)

	api.Register(o, api.Admin, api.Op{
		ID:      "createProfile",
		Method:  http.MethodPost,
		Path:    "/profiles",
		Summary: "Create a Profile; a Content Type's first is its default",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{profileNameTaken, contentTypeNotFound, targetNotFound},
	}, h.createProfile)

	api.Register(o, api.Admin, api.Op{
		ID:      "listProfiles",
		Method:  http.MethodGet,
		Path:    "/profiles",
		Summary: "List the Profiles",
		Status:  http.StatusOK,
	}, h.listProfiles)
}

type handlers struct{ goals Goals }

func (h handlers) createTarget(ctx context.Context, by identity.Admin, in *createTargetInput) (*targetOutput, error) {
	resolutions := make([]targets.Resolution, len(in.Body.Resolutions))
	for i, s := range in.Body.Resolutions {
		r, err := targets.ParseResolution(s)
		if err != nil {
			return nil, api.Invalid
		}
		resolutions[i] = r
	}
	t, err := open(h.goals).CreateTarget(ctx, by, in.Body.Name, resolutions)
	if err != nil {
		return nil, answer(err)
	}
	return &targetOutput{Body: targetOf(t)}, nil
}

func (h handlers) listTargets(ctx context.Context, by identity.Admin, _ *struct{}) (*targetListOutput, error) {
	list, err := open(h.goals).ListTargets(ctx, by)
	if err != nil {
		return nil, err
	}
	out := targetList{Targets: []target{}}
	for _, t := range list {
		out.Targets = append(out.Targets, targetOf(t))
	}
	return &targetListOutput{Body: out}, nil
}

func (h handlers) createProfile(ctx context.Context, by identity.Admin, in *createProfileInput) (*profileOutput, error) {
	ids := make([]targets.TargetID, len(in.Body.TargetIDs))
	for i, s := range in.Body.TargetIDs {
		u, err := storage.ParseUUID(s)
		if err != nil {
			return nil, targetNotFound
		}
		ids[i] = targets.TargetID{UUID: u}
	}
	p, err := open(h.goals).CreateProfile(ctx, by, in.Body.Name, works.TypeID(in.Body.ContentType), ids)
	if err != nil {
		return nil, answer(err)
	}
	return &profileOutput{Body: profileOf(p)}, nil
}

func (h handlers) listProfiles(ctx context.Context, by identity.Admin, _ *struct{}) (*profileListOutput, error) {
	list, err := open(h.goals).ListProfiles(ctx, by)
	if err != nil {
		return nil, err
	}
	out := profileList{Profiles: []profile{}}
	for _, p := range list {
		out.Profiles = append(out.Profiles, profileOf(p))
	}
	return &profileListOutput{Body: out}, nil
}
