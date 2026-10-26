package worksapi

import (
	"context"
	"encoding/json"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/works"
)

type noun struct {
	One  string `json:"one"`
	Many string `json:"many"`
}

type fieldSpec struct {
	Key   string   `json:"key"`
	Label string   `json:"label"`
	Tier  string   `json:"tier" enum:"work,level,item"`
	Depth *int     `json:"depth,omitempty" minimum:"1" doc:"The level's depth, 1 for the outermost; present only on tier level"`
	Kind  string   `json:"kind" enum:"text,number,date,duration,texts,people,link"`
	In    []string `json:"in" enum:"facts,tagline,overview,about,credits,details" nullable:"false" doc:"Every place the Work page shows it"`
	List  listing  `json:"list" doc:"Where the Library uses it; all false for a field of a level or an Item"`
	With  *string  `json:"with" doc:"The key of a text field of the same tier shown in brackets after it; null when none"`
	Art   *string  `json:"art" doc:"The key of a Work artwork kind shown beside it, such as a studio's logo; null when none"`
}

type listing struct {
	Column bool `json:"column" doc:"A column of the Library's table"`
	Sort   bool `json:"sort" doc:"A sort, with sections by its value"`
	Filter bool `json:"filter" doc:"A filter: picked values for text and texts, a range for number, date and duration"`
}

type artRoles struct {
	Poster   *string `json:"poster" doc:"The kind the Library and the Work page show as its poster; null when none"`
	Backdrop *string `json:"backdrop" doc:"The kind shown behind the Work page's header; null when none"`
	Title    *string `json:"title" doc:"The kind of logo shown as the Work's title; null when none"`
}

type artworkKind struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Tier  string `json:"tier" enum:"work,item"`
	Shape string `json:"shape" enum:"poster,backdrop,logo,photo,cover"`
}

type contentType struct {
	ID      string        `json:"id" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Name    string        `json:"name"`
	Icon    string        `json:"icon" enum:"film,tv,note,open-book,headphones,panels,image,shapes"`
	Enabled bool          `json:"enabled"`
	Work    noun          `json:"work"`
	Levels  []noun        `json:"levels" nullable:"false" doc:"Outermost first; empty when Items hang off the Work"`
	Item    noun          `json:"item"`
	Fields  []fieldSpec   `json:"fields" nullable:"false"`
	Artwork []artworkKind `json:"artwork" nullable:"false" doc:"The kinds of image its Works and Items carry"`
	Source  string        `json:"source" doc:"Its metadata source's name, such as TMDB"`
	Items   string        `json:"items" enum:"one,many" doc:"How many Items a Work holds, which sets the Work page's layout"`
	Art     artRoles      `json:"art"`
	Ratings []string      `json:"ratings" nullable:"false" doc:"The ratings sources its Works show, in order. The first a Work holds is its hero rating"`
}

func contentTypeOf(t works.ContentType) contentType {
	out := contentType{
		ID: string(t.ID), Name: t.Name, Icon: t.Icon.String(), Enabled: t.Enabled,
		Work: noun(t.Work), Item: noun(t.Item), Levels: []noun{}, Fields: []fieldSpec{}, Artwork: []artworkKind{},
		Source: t.Source, Items: t.Items.String(), Ratings: []string{},
		Art: artRoles{Poster: stringOrNull(t.Art.Poster), Backdrop: stringOrNull(t.Art.Backdrop), Title: stringOrNull(t.Art.Title)},
	}
	for _, r := range t.Ratings {
		out.Ratings = append(out.Ratings, string(r))
	}
	for _, k := range t.Artwork {
		tier := "work"
		if k.Item {
			tier = "item"
		}
		out.Artwork = append(out.Artwork, artworkKind{Key: k.Key, Label: k.Label, Tier: tier, Shape: string(k.Shape)})
	}
	for _, l := range t.Levels {
		out.Levels = append(out.Levels, noun(l))
	}
	for _, f := range t.Fields {
		spec := fieldSpec{
			Key: f.Key, Label: f.Label, Kind: f.Kind.String(), In: []string{}, With: stringOrNull(f.With), Art: stringOrNull(f.Art),
			List: listing{Column: f.List.Has(works.ListColumn), Sort: f.List.Has(works.ListSort), Filter: f.List.Has(works.ListFilter)},
		}
		for _, p := range f.In {
			spec.In = append(spec.In, p.String())
		}
		switch f.Tier {
		case works.TierWork:
			spec.Tier = "work"
		case works.TierItem:
			spec.Tier = "item"
		default:
			depth := int(f.Tier)
			spec.Tier, spec.Depth = "level", &depth
		}
		out.Fields = append(out.Fields, spec)
	}
	return out
}

type contentTypeList struct {
	ContentTypes []contentType `json:"contentTypes" nullable:"false"`
}

type contentTypeListOutput struct{ Body contentTypeList }

type contentTypeOutput struct{ Body contentType }

type enabledInput struct {
	ContentType string `path:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Body        enabled
}

type enabled struct {
	Enabled bool `json:"enabled"`
}

type contentTypeInput struct {
	ContentType string `path:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
}

type typeSettings struct {
	Settings []api.OneOf[settingapi.Field] `json:"settings" nullable:"false" doc:"In the Content Type's order; empty when it has none"`
}

type typeSettingsOutput struct{ Body typeSettings }

type settingsInput struct {
	ContentType string `path:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Body        settingsChange
}

type settingsChange struct {
	Values map[string]json.RawMessage `json:"values" doc:"Each setting's new value by key, all or none; null resets one to its default"`
}

func (h handlers) listContentTypes(ctx context.Context, _ identity.Account, _ *struct{}) (*contentTypeListOutput, error) {
	types, err := open(h.shelf).Types(ctx)
	if err != nil {
		return nil, err
	}
	out := contentTypeList{ContentTypes: []contentType{}}
	for _, t := range types {
		out.ContentTypes = append(out.ContentTypes, contentTypeOf(t))
	}
	return &contentTypeListOutput{Body: out}, nil
}

func (h handlers) setContentTypeEnabled(ctx context.Context, by identity.Admin, in *enabledInput) (*contentTypeOutput, error) {
	t, err := open(h.shelf).SetEnabled(ctx, by, works.TypeID(in.ContentType), in.Body.Enabled)
	if err != nil {
		return nil, answer(err)
	}
	return &contentTypeOutput{Body: contentTypeOf(t)}, nil
}

func (h handlers) getContentTypeSettings(ctx context.Context, by identity.Admin, in *contentTypeInput) (*typeSettingsOutput, error) {
	fields, err := open(h.shelf).Settings(ctx, by, works.TypeID(in.ContentType))
	if err != nil {
		return nil, answer(err)
	}
	return &typeSettingsOutput{Body: typeSettings{Settings: settingapi.FieldsOf(fields)}}, nil
}

func (h handlers) updateContentTypeSettings(ctx context.Context, by identity.Admin, in *settingsInput) (*typeSettingsOutput, error) {
	fields, err := open(h.shelf).SetSettings(ctx, by, works.TypeID(in.ContentType), in.Body.Values)
	if err != nil {
		return nil, settingapi.Refused(answer(err))
	}
	return &typeSettingsOutput{Body: typeSettings{Settings: settingapi.FieldsOf(fields)}}, nil
}

func stringOrNull(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
