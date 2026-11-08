// Package formatsapi is the Custom Formats operations: list, read, make,
// change, delete and paste them, and the kinds and values each app's
// formats take.
package formatsapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabsapi"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/targetsapi"
)

var (
	formatNotFound     = api.NewCode(http.StatusNotFound, "format_not_found", "No Custom Format has that id")
	patternNotCompiled = api.NewCode(http.StatusBadRequest, "format_pattern_not_compiled", "The pattern in this condition doesn't compile")
	nameTaken          = api.NewCode(http.StatusConflict, "format_name_taken", "Another Custom Format has this name")
	kindUnknown        = api.NewCode(http.StatusUnprocessableEntity, "format_kind_unknown", "SlipStream doesn't have this kind of condition")
	appNeeded          = api.NewCode(http.StatusUnprocessableEntity, "format_app_needed", "Say whether this was written for Radarr or Sonarr")
	formatEmpty        = api.NewCode(http.StatusUnprocessableEntity, "format_empty", "A Custom Format needs at least one condition")
	workNotFound       = api.NewCode(http.StatusNotFound, "work_not_found", "No Work has that id")
)

// Shelf is what the operations run against: the Targets and files, and
// the Grabs the tester judges names with.
type Shelf interface {
	targetsapi.Goals
	grabsapi.Pipeline
}

func open(g targetsapi.Goals) *targets.Targets {
	t, ok := g.Targets()
	if !ok {
		panic("formatsapi: an operation ran on a Node that is no Controller")
	}
	return t
}

func shelf(g targetsapi.Goals) *files.Files {
	f, ok := g.Files()
	if !ok {
		panic("formatsapi: an operation ran on a Node that is no Controller")
	}
	return f
}

func grabber(s Shelf) *grabs.Grabs {
	g, ok := s.Grabs()
	if !ok {
		panic("formatsapi: an operation ran on a Node that is no Controller")
	}
	return g
}

func Routes(r *api.Router, s Shelf) {
	o := r.Owner("formats", api.Admin)
	h := handlers{s}

	api.Register(o, api.Admin, api.Op{
		ID:      "listCustomFormats",
		Method:  http.MethodGet,
		Path:    "/custom-formats",
		Summary: "List the Custom Formats, each with the Targets that score it",
		Status:  http.StatusOK,
	}, h.listCustomFormats)
	api.Register(o, api.Admin, api.Op{
		ID:      "customFormatSchema",
		Method:  http.MethodGet,
		Path:    "/custom-formats/schema",
		Summary: "List the condition kinds each app's Custom Formats take, with each kind's fields and values in words",
		Status:  http.StatusOK,
	}, h.customFormatSchema)
	api.Register(o, api.Admin, api.Op{
		ID:      "createCustomFormat",
		Method:  http.MethodPost,
		Path:    "/custom-formats",
		Summary: "Make a Custom Format in SlipStream's own values. No Target scores it until one is given a score for it",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{patternNotCompiled, nameTaken, kindUnknown, formatEmpty},
	}, h.createCustomFormat)
	api.Register(o, api.Admin, api.Op{
		ID:      "getCustomFormat",
		Method:  http.MethodGet,
		Path:    "/custom-formats/{formatId}",
		Summary: "Get a Custom Format",
		Status:  http.StatusOK,
		Errors:  []*api.Code{formatNotFound},
	}, h.getCustomFormat)
	api.Register(o, api.Admin, api.Op{
		ID:      "updateCustomFormat",
		Method:  http.MethodPut,
		Path:    "/custom-formats/{formatId}",
		Summary: "Replace a Custom Format's name, renaming flag and conditions, refilling the Works of the Targets that score it, or preview the change's Impact",
		Status:  http.StatusOK,
		Errors:  []*api.Code{formatNotFound, patternNotCompiled, nameTaken, kindUnknown, formatEmpty, targetsapi.ImpactChanged},
	}, h.updateCustomFormat)
	api.Register(o, api.Admin, api.Op{
		ID:      "deleteCustomFormat",
		Method:  http.MethodDelete,
		Path:    "/custom-formats/{formatId}",
		Summary: "Delete a Custom Format and every Target's score for it, refilling the Works of the Targets that scored it, or preview the change's Impact",
		Status:  http.StatusOK,
		Errors:  []*api.Code{formatNotFound, targetsapi.ImpactChanged},
	}, h.deleteCustomFormat)
	api.Register(o, api.Admin, api.Op{
		ID:      "pasteCustomFormats",
		Method:  http.MethodPost,
		Path:    "/custom-formats/paste",
		Summary: "Add Custom Formats from Radarr, Sonarr or TRaSH Guides JSON, replacing or keeping beside each one of the same name, or preview what the paste adds and replaces",
		Status:  http.StatusOK,
		Errors:  []*api.Code{appNeeded, kindUnknown, patternNotCompiled, formatEmpty, nameTaken, targetsapi.ImpactChanged},
		MaxBody: 16 << 20,
	}, h.pasteCustomFormats)
	api.Register(o, api.Admin, api.Op{
		ID:      "parseReleases",
		Method:  http.MethodPost,
		Path:    "/releases/parse",
		Summary: "Read release or file names as a search would: the facts, the Work each matches, every Custom Format with each condition's result, and the score and verdict at each Target",
		Status:  http.StatusOK,
		Errors:  []*api.Code{workNotFound, formatNotFound, patternNotCompiled, kindUnknown, formatEmpty},
	}, h.parseReleases)
}

type handlers struct{ goals Shelf }

type condition struct {
	Name     string                     `json:"name"`
	Kind     string                     `json:"kind" doc:"Radarr's and Sonarr's implementation name, such as ReleaseTitleSpecification"`
	Negate   bool                       `json:"negate" doc:"The condition matches when its test fails"`
	Required bool                       `json:"required" doc:"The format matches only when this condition does, beside the others of its kind"`
	Fields   map[string]json.RawMessage `json:"fields" nullable:"false" doc:"Each field by name, as Radarr and Sonarr export them, such as value, or min and max; a value is a number in the radarr and sonarr dialects and a name in slipstream's"`
}

type targetRef struct {
	ID   string `json:"id" format:"uuid"`
	Name string `json:"name"`
}

type customFormat struct {
	ID                  string      `json:"id" format:"uuid"`
	Name                string      `json:"name"`
	IncludeWhenRenaming bool        `json:"includeWhenRenaming" doc:"The Custom Formats naming token lists it"`
	Dialect             string      `json:"dialect" enum:"slipstream,radarr,sonarr" doc:"The app whose values its conditions hold: radarr or sonarr for a pasted format, slipstream for one made here"`
	Conditions          []condition `json:"conditions" nullable:"false"`
	ScoredBy            []targetRef `json:"scoredBy" nullable:"false" doc:"Every Target that scores it, by name"`
	SlowReleases        int         `json:"slowReleases" minimum:"0" doc:"Releases one of its patterns ran past the match limit on since its last save; they count as not matching it"`
}

func formatOf(f targets.StoredFormat) customFormat {
	out := customFormat{
		ID: f.ID.String(), Name: f.Name, IncludeWhenRenaming: f.IncludeWhenRenaming, Dialect: string(f.Dialect),
		Conditions: make([]condition, len(f.Conditions)), ScoredBy: make([]targetRef, len(f.ScoredBy)), SlowReleases: f.Timeouts,
	}
	for i, c := range f.Conditions {
		fields := map[string]json.RawMessage{}
		if len(c.Fields) > 0 {
			if err := json.Unmarshal(c.Fields, &fields); err != nil {
				panic(fmt.Sprintf("formatsapi: stored fields of %s: %v", f.ID, err))
			}
		}
		out.Conditions[i] = condition{Name: c.Name, Kind: string(c.Kind), Negate: c.Negate, Required: c.Required, Fields: fields}
	}
	for i, t := range f.ScoredBy {
		out.ScoredBy[i] = targetRef{ID: t.ID.String(), Name: t.Name}
	}
	return out
}

type formatList struct {
	Formats    []customFormat `json:"formats" nullable:"false" doc:"By name"`
	MatchLimit int64          `json:"matchLimitMilliseconds" minimum:"0" doc:"How long one pattern may run against one name before it counts as not matching"`
}

type formatListOutput struct{ Body formatList }

func (h handlers) listCustomFormats(ctx context.Context, by identity.Admin, _ *struct{}) (*formatListOutput, error) {
	t := open(h.goals)
	all, err := t.ListFormats(ctx, by)
	if err != nil {
		return nil, err
	}
	out := formatList{Formats: make([]customFormat, len(all)), MatchLimit: t.MatchLimit().Milliseconds()}
	for i, f := range all {
		out.Formats[i] = formatOf(f)
	}
	return &formatListOutput{Body: out}, nil
}

type formatInput struct {
	FormatID string `path:"formatId" format:"uuid"`
}

func formatID(s string) (formats.ID, error) {
	id, err := targets.ParseFormatID(s)
	if err != nil {
		return formats.ID{}, formatNotFound
	}
	return id, nil
}

type formatOutput struct{ Body customFormat }

func (h handlers) getCustomFormat(ctx context.Context, by identity.Admin, in *formatInput) (*formatOutput, error) {
	id, err := formatID(in.FormatID)
	if err != nil {
		return nil, err
	}
	f, err := open(h.goals).GetFormat(ctx, by, id)
	if err != nil {
		return nil, notFound(err)
	}
	return &formatOutput{Body: formatOf(f)}, nil
}

type newFormat struct {
	Name                string      `json:"name" minLength:"1" maxLength:"100" doc:"Unique among Custom Formats, ignoring case"`
	IncludeWhenRenaming bool        `json:"includeWhenRenaming,omitempty" doc:"The Custom Formats naming token lists it"`
	Conditions          []condition `json:"conditions" nullable:"false" doc:"Conditions of a kind match when one of them does and every required one does; the format matches when every kind's do"`
}

func (n newFormat) format() formats.Format {
	f := formats.Format{Name: n.Name, IncludeWhenRenaming: n.IncludeWhenRenaming, Conditions: make([]formats.Condition, len(n.Conditions))}
	for i, c := range n.Conditions {
		fields, err := json.Marshal(c.Fields)
		if err != nil {
			panic(fmt.Sprintf("formatsapi: the decoded fields of condition %d: %v", i, err))
		}
		f.Conditions[i] = formats.Condition{Name: c.Name, Kind: formats.Kind(c.Kind), Negate: c.Negate, Required: c.Required, Fields: fields}
	}
	return f
}

type createFormatInput struct{ Body newFormat }

func (h handlers) createCustomFormat(ctx context.Context, by identity.Admin, in *createFormatInput) (*formatOutput, error) {
	f := in.Body.format()
	f.ID = targets.NewFormatID()
	if _, err := shelf(h.goals).Change(ctx, by, targets.AddFormats{Dialect: formats.SlipStream, Add: []formats.Format{f}}, files.Mode{}); err != nil {
		return nil, formatRefused(f, err, "", "save")
	}
	stored, err := open(h.goals).GetFormat(ctx, by, f.ID)
	if err != nil {
		return nil, err
	}
	return &formatOutput{Body: formatOf(stored)}, nil
}

type updateFormatInput struct {
	FormatID string `path:"formatId" format:"uuid"`
	targetsapi.Changing
	Body newFormat
}

type formatChange struct {
	Impact targetsapi.Impact `json:"impact"`
	Format customFormat      `json:"format"`
}

type formatChangeOutput struct{ Body formatChange }

func (h handlers) updateCustomFormat(ctx context.Context, by identity.Admin, in *updateFormatInput) (*formatChangeOutput, error) {
	id, err := formatID(in.FormatID)
	if err != nil {
		return nil, err
	}
	f := in.Body.format()
	i, err := shelf(h.goals).Change(ctx, by, targets.UpdateFormat{ID: id, Format: f}, in.Mode())
	if err != nil {
		return nil, formatRefused(f, notFound(err), "", "save")
	}
	stored, err := open(h.goals).GetFormat(ctx, by, id)
	if err != nil {
		return nil, err
	}
	return &formatChangeOutput{Body: formatChange{Impact: targetsapi.ImpactOf(i), Format: formatOf(stored)}}, nil
}

type deleteFormatInput struct {
	FormatID string `path:"formatId" format:"uuid"`
	targetsapi.Changing
}

type formatDeletion struct {
	Impact   targetsapi.Impact `json:"impact"`
	ScoredBy []targetRef       `json:"scoredBy" nullable:"false" doc:"The Targets whose scores for it the delete drops, by name"`
}

type formatDeletionOutput struct{ Body formatDeletion }

func (h handlers) deleteCustomFormat(ctx context.Context, by identity.Admin, in *deleteFormatInput) (*formatDeletionOutput, error) {
	id, err := formatID(in.FormatID)
	if err != nil {
		return nil, err
	}
	f, err := open(h.goals).GetFormat(ctx, by, id)
	if err != nil {
		return nil, notFound(err)
	}
	i, err := shelf(h.goals).Change(ctx, by, targets.DeleteFormat{ID: id}, in.Mode())
	if err != nil {
		return nil, changed(notFound(err))
	}
	return &formatDeletionOutput{Body: formatDeletion{Impact: targetsapi.ImpactOf(i), ScoredBy: formatOf(f).ScoredBy}}, nil
}

func notFound(err error) error {
	if errors.Is(err, targets.ErrNoFormat) {
		return formatNotFound
	}
	return err
}

func changed(err error) error {
	if e, ok := errors.AsType[*files.ImpactChangedError](err); ok {
		return targetsapi.ImpactRefused(e)
	}
	return err
}

// formatRefused answers err from compiling f, made or changed in the
// editor, with the problem on the field that caused it. at is where f
// sits in the body and again what the Admin does once they fix it.
func formatRefused(f formats.Format, err error, at, again string) error {
	if e, ok := errors.AsType[*formats.PatternError](err); ok {
		return patternNotCompiled.On(fmt.Sprintf("%sconditions[%d].fields.value", at, e.Condition), patternWords(e, "")+" Fix the pattern and "+again+" again.")
	}
	if e, ok := errors.AsType[*formats.KindError](err); ok {
		i := 0
		for j, c := range f.Conditions {
			if string(c.Kind) == e.Kind {
				i = j
				break
			}
		}
		return kindUnknown.On(fmt.Sprintf("%sconditions[%d].kind", at, i), kindWords(e)+" Pick one of the kinds the schema lists.")
	}
	if e, ok := errors.AsType[*formats.EmptyError](err); ok {
		return formatEmpty.On(at+"conditions", fmt.Sprintf("The Custom Format %q has no conditions. Add a condition and %s again.", e.Name, again))
	}
	return refused(err, at+"conditions", again)
}

// refused answers the errors whose problem is the same from the editor
// and a paste, on field. again is what the Admin does once they fix it.
func refused(err error, field, again string) error {
	if errors.Is(err, targets.ErrFormatNameTaken) {
		return nameTaken
	}
	if e, ok := errors.AsType[*formats.FieldError](err); ok {
		return api.Invalid.On(field, fmt.Sprintf("The condition %q has the field %s, which %s conditions don't have. Remove the field and %s again.", e.Name, e.Field, kindName(e.Kind), again))
	}
	if e, ok := errors.AsType[*formats.ValueError](err); ok {
		return api.Invalid.On(field, fmt.Sprintf("The condition %q has the %s %s, which isn't %s. Change it and %s again.", e.Name, e.Field, e.Value, e.Want, again))
	}
	return changed(err)
}

// patternWords is a pattern that doesn't compile, in words, with of
// naming its format when the Admin can't see which. regexp2 can't place
// every error, and one it can't place has no position.
func patternWords(e *formats.PatternError, of string) string {
	if e.Offset > 0 {
		return fmt.Sprintf("The pattern in %q%s doesn't compile at position %d (%s).", e.Name, of, e.Offset, e.Reason)
	}
	return fmt.Sprintf("The pattern in %q%s doesn't compile (%s).", e.Name, of, e.Reason)
}

// kindName is k's words as they read inside a sentence.
func kindName(k formats.Kind) string {
	i := slices.IndexFunc(formats.Kinds(), func(info formats.KindInfo) bool { return info.Kind == k })
	return strings.ToLower(formats.Kinds()[i].Words)
}

func kindWords(e *formats.KindError) string {
	return fmt.Sprintf("The condition %q has the kind %s.", e.Name, e.Kind)
}
