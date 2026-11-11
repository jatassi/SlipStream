package worksapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/filesapi"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/works"
)

type libraryInput struct {
	ContentType string   `query:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$" doc:"Only this Content Type's Works"`
	WorkIDs     []string `query:"workIds" format:"uuid" maxItems:"200" doc:"Only these Works, such as the ones a works.changed named; not with contentType"`
}

type libraryRow struct {
	Work
	Monitored   string                                 `json:"monitored" enum:"all,some,none" doc:"How many of the Items its source still lists are monitored"`
	ProfileID   string                                 `json:"profileId" format:"uuid"`
	RootID      *string                                `json:"rootId" format:"uuid" doc:"Null while the Work has no root"`
	PosterURL   *string                                `json:"posterUrl" doc:"The artwork route for its poster, without a size; null when it has none"`
	Rating      api.Nullable[rowRating]                `json:"rating" doc:"The first of its Content Type's ratings sources that it holds; null when it holds none"`
	AvailableOn *string                                `json:"availableOn" pattern:"^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$" doc:"Its one Item's availability date, or for many Items the earliest not yet out, else the latest out, read over its monitored Items, or over every Item while none is monitored; null while none of those has a date"`
	Bytes       int64                                  `json:"bytes" minimum:"0" doc:"The size of its library files, Kept files included"`
	Files       int                                    `json:"files" minimum:"0" doc:"Its library files, Kept files included"`
	Kept        int                                    `json:"kept" minimum:"0" doc:"Its Kept files"`
	GoneFiles   int                                    `json:"goneFiles" minimum:"0" doc:"Its files a library scan found missing from its root, which the other counts leave out"`
	AddedBy     string                                 `json:"addedBy" enum:"admin,scan" doc:"Who added it: an Admin, or a library scan that matched a folder on disk"`
	Status      api.Nullable[api.Enum[filesapi.State]] `json:"status" doc:"The first status its monitored Items hold, in the glossary's order; null while none holds a cell"`
	Cells       api.Counts[filesapi.State]             `json:"cells" doc:"Its monitored Items' cells by state"`
	Items       int                                    `json:"items" minimum:"0" doc:"Its monitored Items that its source still lists, the only Items the status and counts read"`
	ItemStatus  api.Counts[filesapi.State]             `json:"itemStatus" doc:"Its monitored Items by status; an Item with no cell isn't counted"`
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
	var scope works.Scope
	if len(in.WorkIDs) > 0 {
		ws, err := workIDsOf(in.WorkIDs)
		if err != nil {
			return nil, err
		}
		scope.Works = ws
	}
	if in.ContentType != "" {
		t := works.TypeID(in.ContentType)
		scope.Type = &t
	}
	rows, err := composer(h.shelf).Rows(ctx, by, scope)
	if err != nil {
		return nil, answer(err)
	}
	return &libraryRowsOutput{Body: libraryRows{Rows: rowsOf(minter(h.shelf), rows)}}, nil
}

func rowsOf(m artwork.Minter, rows []library.Row) []libraryRow {
	out := make([]libraryRow, len(rows))
	for i, row := range rows {
		out[i] = rowOf(m, row)
	}
	return out
}

func rowOf(m artwork.Minter, row library.Row) libraryRow {
	w, r := row.Work, row.Rollup
	out := libraryRow{
		Work: workOf(w.Summary), Monitored: w.Monitored.String(), ProfileID: r.Profile.String(), RootID: idOrNull(r.Root.UUID),
		PosterURL: urlOrNull(m, w.Poster), Bytes: r.Bytes, Files: r.Files, Kept: r.Kept, GoneFiles: row.Gone, AddedBy: string(w.AddedBy),
		Status: filesapi.StatusOf(r.Status), Cells: filesapi.CountsOf(r.Cells),
		Items: r.Items, ItemStatus: filesapi.CountsOf(r.ByItem), NeedsYou: len(r.Needs), Values: valuesOf(w.Values),
	}
	if rt := w.Rating; rt != nil {
		out.Rating = api.Some(rowRating{Source: string(rt.Source), Value: rt.Value, Scale: works.RatingSources[rt.Source].Scale})
	}
	if d, known := r.Available.Get(); known {
		out.AvailableOn = new(d.String())
	}
	return out
}

var bareOf = map[works.FieldKind]func(works.FieldValue) api.Bare{
	works.KindText:     func(v works.FieldValue) api.Bare { return api.BareString(v.Text) },
	works.KindDate:     func(v works.FieldValue) api.Bare { return api.BareString(v.Date.String()) },
	works.KindNumber:   func(v works.FieldValue) api.Bare { return api.BareNumber(v.Number) },
	works.KindDuration: func(v works.FieldValue) api.Bare { return api.BareNumber(float64(v.Duration.Milliseconds())) },
	works.KindTexts:    func(v works.FieldValue) api.Bare { return api.BareStrings(v.Texts) },
}

func valuesOf(vs []works.FieldValue) map[string]api.Bare {
	out := make(map[string]api.Bare, len(vs))
	for _, v := range vs {
		out[v.Key] = bareOf[v.Kind](v)
	}
	return out
}
