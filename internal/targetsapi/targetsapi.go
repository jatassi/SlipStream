package targetsapi

import (
	"context"
	"errors"
	"net/http"

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
	noSuchTarget        = api.NewCode(http.StatusNotFound, "target_not_found", "No Target has that id")
	qualityNotInTarget  = api.NewCode(http.StatusUnprocessableEntity, "quality_not_in_target", "The Target doesn't accept that quality")
	targetInUse         = api.NewCode(http.StatusConflict, "target_in_use", "Profiles hold the Target")
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
	if inUse, ok := errors.AsType[*targets.InUseError](err); ok {
		return targetInUse.Because(inUse.Error())
	}
	for _, bad := range []struct {
		err   error
		field string
	}{{targets.ErrBadTiers, "tiers"}, {targets.ErrBadSize, "sizes"}, {targets.ErrBadFilter, "filters"}} {
		if errors.Is(err, bad.err) {
			return api.Invalid.On(bad.field, err.Error())
		}
	}
	if errors.Is(err, targets.ErrQualityNotInTarget) {
		return qualityNotInTarget.Because(err.Error())
	}
	for _, c := range codes {
		if errors.Is(err, c.err) {
			return c.code
		}
	}
	return err
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

	registerTargets(o, h)

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
