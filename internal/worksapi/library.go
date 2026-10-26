package worksapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/filesapi"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/works"
)

type libraryInput struct {
	ContentType string   `query:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$" doc:"Only this Content Type's Works"`
	WorkIDs     []string `query:"workIds" format:"uuid" maxItems:"200" doc:"Only these Works, such as the ones a works.changed named; not with contentType"`
}

type libraryRow struct {
	Work
	Monitored   string                                 `json:"monitored" enum:"all,some,none" doc:"How many of its Items are monitored"`
	ProfileID   string                                 `json:"profileId" format:"uuid"`
	RootID      *string                                `json:"rootId" format:"uuid" doc:"Null while the Work has no root"`
	PosterURL   *string                                `json:"posterUrl" doc:"The artwork route for its poster, without a size; null when it has none"`
	Rating      api.Nullable[rowRating]                `json:"rating" doc:"The first of its Content Type's ratings sources that it holds; null when it holds none"`
	AvailableOn *string                                `json:"availableOn" pattern:"^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$" doc:"Its one Item's availability date, or for many Items the earliest not yet out, else the latest out; null while no Item has a date"`
	Bytes       int64                                  `json:"bytes" minimum:"0" doc:"The size of its library files, Kept files included"`
	Files       int                                    `json:"files" minimum:"0" doc:"Its library files, Kept files included"`
	Kept        int                                    `json:"kept" minimum:"0" doc:"Its Kept files"`
	Status      api.Nullable[api.Enum[filesapi.State]] `json:"status" doc:"The first status its Items hold, in the glossary's order; null for a Work with no cell"`
	Cells       api.Counts[filesapi.State]             `json:"cells" doc:"Its cells by state"`
	Failed      int                                    `json:"failed" minimum:"0" doc:"Its wanted cells whose download failed"`
	Items       int                                    `json:"items" minimum:"0"`
	ItemStatus  api.Counts[filesapi.State]             `json:"itemStatus" doc:"Its Items by status; an Item with no cell isn't counted"`
	NeedsYou    int                                    `json:"needsYou" minimum:"0" doc:"The entries of its Needs you"`
	Values      map[string]api.Bare                    `json:"values" doc:"Its listed fields by key: a string for text and date, a number for number and duration in ms, a list of strings for texts; a field it lacks is left out"`
}

type rowRating struct {
	Source string  `json:"source" doc:"The ratings source, such as imdb"`
	Value  float64 `json:"value" minimum:"0"`
	Scale  float64 `json:"scale" exclusiveMinimum:"0" doc:"The value's maximum"`
}

type libraryRows struct {
	Rows []libraryRow `json:"rows" nullable:"false" doc:"By sort label, then id"`
}

type libraryRowsOutput struct{ Body libraryRows }

func (h handlers) listLibrary(ctx context.Context, by identity.Admin, in *libraryInput) (*libraryRowsOutput, error) {
	if in.ContentType != "" && len(in.WorkIDs) > 0 {
		return nil, api.Invalid.Fields(api.FieldError{Location: "query.workIds", Message: "Use contentType or workIds, not both."})
	}
	ws, err := workIDsOf(in.WorkIDs)
	if err != nil {
		return nil, err
	}
	scope := library.Scope{Works: ws}
	if in.ContentType != "" {
		t := works.TypeID(in.ContentType)
		scope.Type = &t
	}
	if _, err := composer(h.shelf).Rows(ctx, by, scope); err != nil {
		return nil, answer(err)
	}
	return nil, notImplemented
}
