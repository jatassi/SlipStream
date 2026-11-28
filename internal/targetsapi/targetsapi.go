package targetsapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/conversions"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/nodes"
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
	// Conversions is the Conversions, for what a Target's editor shows of
	// them, false unless the Node is a Controller.
	Conversions() (*conversions.Conversions, bool)
	// Nodes is the roster, for the Nodes that convert, false unless the
	// Node is a Controller.
	Nodes() (*nodes.Service, bool)
}

func open(g Goals) *targets.Targets {
	t, ok := g.Targets()
	if !ok {
		panic("targetsapi: an operation ran on a Node that is no Controller")
	}
	return t
}

func converts(g Goals) *conversions.Conversions {
	c, ok := g.Conversions()
	if !ok {
		panic("targetsapi: an operation ran on a Node that is no Controller")
	}
	return c
}

func roster(g Goals) *nodes.Service {
	n, ok := g.Nodes()
	if !ok {
		panic("targetsapi: an operation ran on a Node that is no Controller")
	}
	return n
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
	targetInUse         = api.NewCode(http.StatusConflict, "target_in_use", "Profiles use the Target")
	noSuchProfile       = api.NewCode(http.StatusNotFound, "profile_not_found", "No Profile has that id")
	workNotFound        = api.NewCode(http.StatusNotFound, "work_not_found", "No Work has that id")
	formatNotFound      = api.NewCode(http.StatusUnprocessableEntity, "format_not_found", "No Custom Format has that id")
	duplicateLanguage   = api.NewCode(http.StatusUnprocessableEntity, "duplicate_language", "The Profile lists a subtitle language twice")
	noSubtitlesForType  = api.NewCode(http.StatusUnprocessableEntity, "no_subtitles_for_type", "The Content Type has no subtitles")
)

var codes = []struct {
	err  error
	code *api.Code
}{
	{targets.ErrProfileNameTaken, profileNameTaken},
	{targets.ErrNoType, contentTypeNotFound},
	{targets.ErrNoTarget, targetNotFound},
	{targets.ErrNoProfile, noSuchProfile},
	{targets.ErrNoCut, cutNotFound},
	{works.ErrNoWork, workNotFound},
	{targets.ErrNoFormat, formatNotFound},
}

func answer(err error) error {
	if refused := conversionAnswer(err); refused != nil {
		return refused
	}
	if dup, ok := errors.AsType[*targets.DuplicateLanguageError](err); ok {
		return duplicateLanguage.On(fmt.Sprintf("languages[%d]", dup.Index), kindWords(dup.Wanted.Forced)+" "+dup.Wanted.Language.Name()+" subtitles are already on this list.")
	}
	if none, ok := errors.AsType[*targets.NoSubtitlesError](err); ok {
		return noSubtitlesForType.On("languages", none.Type+" has no subtitles, so its Profiles can't list subtitle languages.")
	}
	if inUse, ok := errors.AsType[*targets.InUseError](err); ok {
		fix := " Remove it from that Profile first."
		if len(inUse.By) > 1 {
			fix = " Remove it from those Profiles first."
		}
		return targetInUse.Because(usesWords(inUse.By, "this Target") + fix)
	}
	if taken, ok := errors.AsType[*targets.TargetNameTakenError](err); ok {
		fix := "Choose another name."
		if taken.Name != taken.Other {
			fix = "Use a name that differs by more than case."
		}
		return targetNameTaken.Because(fmt.Sprintf("Another Target is named %s. %s", taken.Other, fix))
	}
	if errors.Is(err, targets.ErrQualityNotInTarget) {
		return qualityNotInTarget.Because(specWords(err))
	}
	for _, bad := range []struct {
		err   error
		field string
	}{{targets.ErrBadTiers, "tiers"}, {targets.ErrBadSize, "sizes"}, {targets.ErrBadFilter, "filters"}, {targets.ErrTargetTwice, "targets"}, {targets.ErrNoCuts, "editions.cuts"}, {targets.ErrCutTwice, "editions.cuts"}, {targets.ErrBadIncrement, "scores.increment"}} {
		if errors.Is(err, bad.err) {
			return api.Invalid.On(bad.field, specWords(err))
		}
	}
	for _, c := range codes {
		if errors.Is(err, c.err) {
			return c.code
		}
	}
	return err
}

// kindWords names a subtitle kind at the start of a sentence.
func kindWords(forced bool) string {
	if forced {
		return "Forced-only"
	}
	return "Full"
}

// usesWords is the sentence that the things by names use thing.
func usesWords(by []string, thing string) string {
	if len(by) == 1 {
		return by[0] + " uses " + thing + "."
	}
	return strings.Join(by[:len(by)-1], ", ") + " and " + by[len(by)-1] + " use " + thing + "."
}

// specWords is the sentence of err, a Target or Profile that targets
// refused as written.
func specWords(err error) string {
	se, _ := errors.AsType[*targets.SpecError](err)
	name := func() string { return se.Quality.Info().Name }
	switch {
	case errors.Is(err, targets.ErrQualityNotInTarget) && se.Size:
		return "No tier has " + name() + ", so it can't have a size. Add " + name() + " to a tier, or remove its size."
	case errors.Is(err, targets.ErrQualityNotInTarget):
		return "No tier has the cutoff, " + name() + ". Add it to a tier, or choose another cutoff."
	case errors.Is(err, targets.ErrBadTiers) && se == nil:
		return "Add at least one tier."
	case errors.Is(err, targets.ErrBadTiers) && se.Tier > 0:
		return fmt.Sprintf("Tier %d has no qualities. Add one, or remove the tier.", se.Tier)
	case errors.Is(err, targets.ErrBadTiers):
		return name() + " is in the tiers twice. Keep it in one place."
	case errors.Is(err, targets.ErrBadSize):
		return "The size for " + name() + " needs a minimum of 0 or more and a maximum no lower than the minimum."
	case errors.Is(err, targets.ErrBadFilter):
		return "Use only values SlipStream lists for the attribute, each once. A filter set to Any takes no values, and one set to Only or Not takes at least one."
	case errors.Is(err, targets.ErrTargetTwice):
		return "Add each Target to the Profile once."
	case errors.Is(err, targets.ErrNoCuts):
		return "Choose at least one cut."
	case errors.Is(err, targets.ErrCutTwice):
		return "Choose each cut once."
	case errors.Is(err, targets.ErrBadIncrement):
		return "Use a score increment of 1 or more."
	}
	panic("targetsapi: no words for " + err.Error())
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
		Summary: "Create a Target from its tiers of qualities, cutoff, filters, sizes, Custom Format scores and propers setting",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{targetNameTaken, qualityNotInTarget, formatNotFound, conversionRefused.Code},
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
		Errors:  []*api.Code{noSuchTarget, targetNameTaken, qualityNotInTarget, formatNotFound, ImpactChanged, conversionRefused.Code, targetIsASource.Code},
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
		Summary: "Create a Profile of Targets in Profile order. A new Profile is never the default",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{profileNameTaken, contentTypeNotFound, targetNotFound, cutNotFound, duplicateLanguage, noSubtitlesForType},
	}, h.createProfile)
	api.Register(o, api.Admin, api.Op{
		ID:      "listProfiles",
		Method:  http.MethodGet,
		Path:    "/profiles",
		Summary: "List the Profiles, each with the number of Works that use it",
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
		Summary: "Replace a Profile's name, Targets, Edition list and subtitle languages, refilling and renaming its Works' files when its Targets or Edition list change, or preview the change's Impact",
		Status:  http.StatusOK,
		Errors:  []*api.Code{noSuchProfile, profileNameTaken, targetNotFound, cutNotFound, profileWrongType, profileNotFound, duplicateLanguage, noSubtitlesForType, ImpactChanged, targetIsASource.Code},
	}, h.updateProfile)
	api.Register(o, api.Admin, api.Op{
		ID:      "deleteProfile",
		Method:  http.MethodDelete,
		Path:    "/profiles/{profileId}",
		Summary: "Delete a Profile that isn't its Content Type's default, moving its Works to another of the type, or preview the change's Impact",
		Status:  http.StatusOK,
		Errors:  []*api.Code{noSuchProfile, profileNotFound, profileIsDefault, profileWrongType, ImpactChanged},
	}, h.deleteProfile)
	api.Register(o, api.Admin, api.Op{
		ID:      "makeDefaultProfile",
		Method:  http.MethodPost,
		Path:    "/profiles/{profileId}/default",
		Summary: "Make a Profile its Content Type's default. Existing Works keep theirs",
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
		Errors:  []*api.Code{noSuchCut, standardIsFixed, cutNotCustom, cutNameTaken, ImpactChanged},
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
		Summary: "Get the Profile a Work uses and its Edition list",
		Status:  http.StatusOK,
		Errors:  []*api.Code{workNotFound},
	}, h.getWorkWants)
	api.Register(o, api.Admin, api.Op{
		ID:      "setWorkWants",
		Method:  http.MethodPut,
		Path:    "/works/{workId}/wants",
		Summary: "Set a Work's own Edition list, or its Profile's, refilling and renaming its files, or preview the change's Impact",
		Status:  http.StatusOK,
		Errors:  []*api.Code{workNotFound, cutNotFound, ImpactChanged},
	}, h.setWorkWants)
}

type handlers struct{ goals Goals }
