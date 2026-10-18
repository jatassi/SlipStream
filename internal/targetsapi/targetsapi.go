package targetsapi

import (
	"errors"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type Goals interface {
	// Targets is the Targets and Profiles, false unless the Node is a
	// Controller.
	Targets() (*targets.Targets, bool)
	// Files is the library's files, for disk estimates, false unless the
	// Node is a Controller.
	Files() (*files.Files, bool)
}

func open(g Goals) *targets.Targets {
	t, ok := g.Targets()
	if !ok {
		panic("targetsapi: an operation ran on a Node that is no Controller")
	}
	return t
}

func shelf(g Goals) *files.Files {
	f, ok := g.Files()
	if !ok {
		panic("targetsapi: an operation ran on a Node that is no Controller")
	}
	return f
}

var (
	targetNameTaken     = api.NewCode(http.StatusConflict, "target_name_taken", "A Target already has that name")
	profileNameTaken    = api.NewCode(http.StatusConflict, "profile_name_taken", "A Profile already has that name")
	contentTypeNotFound = api.NewCode(http.StatusNotFound, "content_type_not_found", "No Content Type has that id")
	targetNotFound      = api.NewCode(http.StatusUnprocessableEntity, "target_not_found", "No Target has that id")
	noSuchTarget        = api.NewCode(http.StatusNotFound, "target_not_found", "No Target has that id")
	qualityNotInTarget  = api.NewCode(http.StatusUnprocessableEntity, "quality_not_in_target", "The Target doesn't accept that quality")
	targetInUse         = api.NewCode(http.StatusConflict, "target_in_use", "Profiles hold the Target")
	noSuchProfile       = api.NewCode(http.StatusNotFound, "profile_not_found", "No Profile has that id")
	workNotFound        = api.NewCode(http.StatusNotFound, "work_not_found", "No Work has that id")
)

var codes = []struct {
	err  error
	code *api.Code
}{
	{targets.ErrTargetNameTaken, targetNameTaken},
	{targets.ErrProfileNameTaken, profileNameTaken},
	{targets.ErrNoType, contentTypeNotFound},
	{targets.ErrNoTarget, targetNotFound},
	{targets.ErrNoProfile, noSuchProfile},
	{targets.ErrNoCut, cutNotFound},
	{works.ErrNoWork, workNotFound},
}

func answer(err error) error {
	if inUse, ok := errors.AsType[*targets.InUseError](err); ok {
		return targetInUse.Because(inUse.Error())
	}
	for _, bad := range []struct {
		err   error
		field string
	}{{targets.ErrBadTiers, "tiers"}, {targets.ErrBadSize, "sizes"}, {targets.ErrBadFilter, "filters"}, {targets.ErrTargetTwice, "targets"}, {targets.ErrNoCuts, "editions.cuts"}, {targets.ErrCutTwice, "editions.cuts"}} {
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

func Routes(r *api.Router, goals Goals) {
	o := r.Owner("targets", api.Admin)
	h := handlers{goals}

	api.Register(o, api.Admin, api.Op{
		ID:      "listQualities",
		Method:  http.MethodGet,
		Path:    "/qualities",
		Summary: "List the qualities a Target can accept, and the presets that order them",
		Status:  http.StatusOK,
	}, h.listQualities)
	api.Register(o, api.Admin, api.Op{
		ID:      "createTarget",
		Method:  http.MethodPost,
		Path:    "/targets",
		Summary: "Create a Target from its tiers of qualities, cutoff, filters and sizes",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{targetNameTaken, qualityNotInTarget},
	}, h.createTarget)
	api.Register(o, api.Admin, api.Op{
		ID:      "listTargets",
		Method:  http.MethodGet,
		Path:    "/targets",
		Summary: "List the Targets",
		Status:  http.StatusOK,
	}, h.listTargets)
	api.Register(o, api.Admin, api.Op{
		ID:      "getTarget",
		Method:  http.MethodGet,
		Path:    "/targets/{targetId}",
		Summary: "Get a Target",
		Status:  http.StatusOK,
		Errors:  []*api.Code{noSuchTarget},
	}, h.getTarget)
	api.Register(o, api.Admin, api.Op{
		ID:      "updateTarget",
		Method:  http.MethodPut,
		Path:    "/targets/{targetId}",
		Summary: "Replace a Target's name and rules, renaming and refilling its Works' files, or preview the change's Impact",
		Status:  http.StatusOK,
		Errors:  []*api.Code{noSuchTarget, targetNameTaken, qualityNotInTarget, impactChanged},
	}, h.updateTarget)
	api.Register(o, api.Admin, api.Op{
		ID:      "duplicateTarget",
		Method:  http.MethodPost,
		Path:    "/targets/{targetId}/duplicate",
		Summary: "Copy a Target's rules under the first free name ending in copy",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{noSuchTarget},
	}, h.duplicateTarget)
	api.Register(o, api.Admin, api.Op{
		ID:      "deleteTarget",
		Method:  http.MethodDelete,
		Path:    "/targets/{targetId}",
		Summary: "Delete a Target no Profile holds",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{noSuchTarget, targetInUse},
	}, h.deleteTarget)

	api.Register(o, api.Admin, api.Op{
		ID:      "createProfile",
		Method:  http.MethodPost,
		Path:    "/profiles",
		Summary: "Create a Profile of Targets in Profile order; it is never the default",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{profileNameTaken, contentTypeNotFound, targetNotFound, cutNotFound},
	}, h.createProfile)
	api.Register(o, api.Admin, api.Op{
		ID:      "listProfiles",
		Method:  http.MethodGet,
		Path:    "/profiles",
		Summary: "List the Profiles, each with the count of Works wanting from it",
		Status:  http.StatusOK,
	}, h.listProfiles)
	api.Register(o, api.Admin, api.Op{
		ID:      "getProfile",
		Method:  http.MethodGet,
		Path:    "/profiles/{profileId}",
		Summary: "Get a Profile with the disk each of its Targets costs across its Works",
		Status:  http.StatusOK,
		Errors:  []*api.Code{noSuchProfile},
	}, h.getProfile)
	api.Register(o, api.Admin, api.Op{
		ID:      "updateProfile",
		Method:  http.MethodPut,
		Path:    "/profiles/{profileId}",
		Summary: "Replace a Profile's name, Targets and Edition list, refilling and renaming its Works' files, or preview the change's Impact",
		Status:  http.StatusOK,
		Errors:  []*api.Code{noSuchProfile, profileNameTaken, targetNotFound, cutNotFound, profileWrongType, impactChanged},
	}, h.updateProfile)
	api.Register(o, api.Admin, api.Op{
		ID:      "deleteProfile",
		Method:  http.MethodDelete,
		Path:    "/profiles/{profileId}",
		Summary: "Delete a Profile that isn't its Content Type's default, moving its Works to another of the type, or preview the change's Impact",
		Status:  http.StatusOK,
		Errors:  []*api.Code{noSuchProfile, profileNotFound, profileIsDefault, profileWrongType, impactChanged},
	}, h.deleteProfile)
	api.Register(o, api.Admin, api.Op{
		ID:      "makeDefaultProfile",
		Method:  http.MethodPost,
		Path:    "/profiles/{profileId}/default",
		Summary: "Make a Profile its Content Type's default; existing Works keep theirs",
		Status:  http.StatusOK,
		Errors:  []*api.Code{noSuchProfile},
	}, h.makeDefaultProfile)
	api.Register(o, api.Admin, api.Op{
		ID:      "listCuts",
		Method:  http.MethodGet,
		Path:    "/cuts",
		Summary: "List the cuts an Edition list can name: built in, added by a Content Type, or custom",
		Status:  http.StatusOK,
	}, h.listCuts)
	api.Register(o, api.Admin, api.Op{
		ID:      "createCut",
		Method:  http.MethodPost,
		Path:    "/cuts",
		Summary: "Add a custom cut, keyed by its name's words, which every title parse reads from then on",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{cutNameTaken},
	}, h.createCut)
	api.Register(o, api.Admin, api.Op{
		ID:      "updateCut",
		Method:  http.MethodPut,
		Path:    "/cuts/{cutKey}",
		Summary: "Rename a custom cut or replace its aliases, renaming the files of it, or preview the change's Impact",
		Status:  http.StatusOK,
		Errors:  []*api.Code{noSuchCut, standardIsFixed, cutNotCustom, cutNameTaken, impactChanged},
	}, h.updateCut)
	api.Register(o, api.Admin, api.Op{
		ID:      "deleteCut",
		Method:  http.MethodDelete,
		Path:    "/cuts/{cutKey}",
		Summary: "Remove a custom cut no Edition list names and no file is of",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{noSuchCut, standardIsFixed, cutNotCustom, cutInUse},
	}, h.deleteCut)
	api.Register(o, api.Admin, api.Op{
		ID:      "getWorkWants",
		Method:  http.MethodGet,
		Path:    "/works/{workId}/wants",
		Summary: "Get the Profile a Work wants from and its Edition list",
		Status:  http.StatusOK,
		Errors:  []*api.Code{workNotFound},
	}, h.getWorkWants)
	api.Register(o, api.Admin, api.Op{
		ID:      "setWorkWants",
		Method:  http.MethodPut,
		Path:    "/works/{workId}/wants",
		Summary: "Set the Profile a Work wants from and its own Edition list, refilling and renaming its files, or preview the change's Impact",
		Status:  http.StatusOK,
		Errors:  []*api.Code{workNotFound, profileNotFound, profileWrongType, cutNotFound, impactChanged},
	}, h.setWorkWants)
}

type handlers struct{ goals Goals }
