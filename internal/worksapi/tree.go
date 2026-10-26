package worksapi

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type fieldValue struct{}

func (fieldValue) Variants() (discriminator string, variants []any) {
	return "kind", []any{textValue{}, numberValue{}, dateValue{}, durationValue{}, textsValue{}, peopleValue{}, linkValue{}}
}

type textValue struct {
	Key  string `json:"key"`
	Kind string `json:"kind" enum:"text"`
	Text string `json:"text"`
}

type numberValue struct {
	Key    string  `json:"key"`
	Kind   string  `json:"kind" enum:"number"`
	Number float64 `json:"number"`
}

type dateValue struct {
	Key  string `json:"key"`
	Kind string `json:"kind" enum:"date"`
	Date string `json:"date" pattern:"^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$" doc:"YYYY, YYYY-MM or YYYY-MM-DD"`
}

type durationValue struct {
	Key  string `json:"key"`
	Kind string `json:"kind" enum:"duration"`
	Ms   int64  `json:"ms" minimum:"0"`
}

type textsValue struct {
	Key   string   `json:"key"`
	Kind  string   `json:"kind" enum:"texts"`
	Texts []string `json:"texts" nullable:"false" doc:"In the source's order"`
}

type peopleValue struct {
	Key    string   `json:"key"`
	Kind   string   `json:"kind" enum:"people"`
	People []person `json:"people" nullable:"false" doc:"In the source's order"`
}

type person struct {
	Name       string  `json:"name"`
	Role       string  `json:"role" doc:"The character played or the job done; empty when the source names none"`
	Department string  `json:"department" doc:"Empty when the source names none"`
	PhotoURL   *string `json:"photoUrl" doc:"The artwork route for the person's photo, without a size; null when the source has none"`
}

type linkValue struct {
	Key   string `json:"key"`
	Kind  string `json:"kind" enum:"link"`
	Label string `json:"label"`
	URL   string `json:"url" format:"uri"`
}

var valueOf = map[works.FieldKind]func(works.FieldValue, artwork.Minter) any{
	works.KindText: func(v works.FieldValue, _ artwork.Minter) any {
		return textValue{Key: v.Key, Kind: v.Kind.String(), Text: v.Text}
	},
	works.KindNumber: func(v works.FieldValue, _ artwork.Minter) any {
		return numberValue{Key: v.Key, Kind: v.Kind.String(), Number: v.Number}
	},
	works.KindDate: func(v works.FieldValue, _ artwork.Minter) any {
		return dateValue{Key: v.Key, Kind: v.Kind.String(), Date: v.Date.String()}
	},
	works.KindDuration: func(v works.FieldValue, _ artwork.Minter) any {
		return durationValue{Key: v.Key, Kind: v.Kind.String(), Ms: v.Duration.Milliseconds()}
	},
	works.KindTexts: func(v works.FieldValue, _ artwork.Minter) any {
		return textsValue{Key: v.Key, Kind: v.Kind.String(), Texts: append([]string{}, v.Texts...)}
	},
	works.KindPeople: func(v works.FieldValue, m artwork.Minter) any {
		people := make([]person, len(v.People))
		for i, p := range v.People {
			people[i] = person{Name: p.Name, Role: p.Role, Department: p.Department, PhotoURL: urlOrNull(m, p.Photo)}
		}
		return peopleValue{Key: v.Key, Kind: v.Kind.String(), People: people}
	},
	works.KindLink: func(v works.FieldValue, _ artwork.Minter) any {
		return linkValue{Key: v.Key, Kind: v.Kind.String(), Label: v.Link.Label, URL: v.Link.URL}
	},
}

func fieldsOf(m artwork.Minter, vs []works.FieldValue) []api.OneOf[fieldValue] {
	out := make([]api.OneOf[fieldValue], len(vs))
	for i, v := range vs {
		out[i] = api.NewOneOf[fieldValue](valueOf[v.Kind](v, m))
	}
	return out
}

type art struct {
	Kind string `json:"kind" doc:"An artwork kind of the Content Type"`
	URL  string `json:"url" doc:"The artwork route for the image, without a size"`
}

// artOf is each kind's image, by kind key.
func artOf(m artwork.Minter, refs map[string]artwork.Ref) []art {
	out := []art{}
	for _, kind := range slices.Sorted(maps.Keys(refs)) {
		out = append(out, art{Kind: kind, URL: m.URL(refs[kind])})
	}
	return out
}

// Work is exported so the document carries the fields tree embeds.
type Work struct {
	ID             string     `json:"id" format:"uuid"`
	ContentType    string     `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Key            string     `json:"key"`
	Label          string     `json:"label"`
	SortLabel      string     `json:"sortLabel"`
	AddedAt        time.Time  `json:"addedAt"`
	RefreshedAt    time.Time  `json:"refreshedAt"`
	GoneUpstreamAt *time.Time `json:"goneUpstreamAt" doc:"When the metadata source last reported that it no longer has this Work; null while it has it"`
}

func workOf(s works.Summary) Work {
	return Work{
		ID: s.ID.String(), ContentType: string(s.ContentType), Key: string(s.Key), Label: s.Label, SortLabel: s.SortLabel,
		AddedAt: s.AddedAt.Time(), RefreshedAt: s.RefreshedAt.Time(), GoneUpstreamAt: timeOrNull(s.GoneUpstreamAt),
	}
}

type rating struct {
	Source    string    `json:"source" doc:"The ratings source, such as imdb"`
	Label     string    `json:"label" doc:"The source's name as a person reads it"`
	Value     float64   `json:"value" minimum:"0"`
	Scale     float64   `json:"scale" exclusiveMinimum:"0" doc:"The value's maximum"`
	Votes     *int64    `json:"votes" minimum:"0" doc:"Null when the source counts none"`
	FetchedAt time.Time `json:"fetchedAt"`
}

func ratingsOf(rs []works.StoredRating) []rating {
	out := make([]rating, len(rs))
	for i, r := range rs {
		s := works.RatingSources[r.Source]
		out[i] = rating{Source: string(r.Source), Label: s.Label, Value: r.Value, Scale: s.Scale, Votes: r.Votes, FetchedAt: r.FetchedAt.Time()}
	}
	return out
}

type tree struct {
	Work
	Fields  []api.OneOf[fieldValue] `json:"fields" nullable:"false"`
	Ratings []rating                `json:"ratings" nullable:"false" doc:"The Work's own, by source"`
	Artwork []art                   `json:"artwork" nullable:"false" doc:"The Work's own images, by kind"`
	Levels  []level                 `json:"levels" nullable:"false" doc:"By depth, then their parent's order, live before withdrawn, then position"`
	Items   []item                  `json:"items" nullable:"false" doc:"By their level's order, live before withdrawn, then position"`
}

type level struct {
	ID          string                  `json:"id" format:"uuid"`
	ParentID    *string                 `json:"parentId" format:"uuid" doc:"Null under the Work"`
	Depth       int                     `json:"depth" minimum:"1"`
	Key         string                  `json:"key"`
	Label       string                  `json:"label"`
	Position    int                     `json:"position" minimum:"1"`
	WithdrawnAt *time.Time              `json:"withdrawnAt" doc:"When the source stopped listing it; null while it lists it"`
	Fields      []api.OneOf[fieldValue] `json:"fields" nullable:"false"`
}

type item struct {
	ID           string                  `json:"id" format:"uuid"`
	LevelID      *string                 `json:"levelId" format:"uuid" doc:"Null when the Content Type declares no levels"`
	Key          string                  `json:"key"`
	Label        string                  `json:"label"`
	Position     int                     `json:"position" minimum:"1"`
	WithdrawnAt  *time.Time              `json:"withdrawnAt" doc:"When the source stopped listing it; null while it lists it"`
	Monitored    bool                    `json:"monitored"`
	Fields       []api.OneOf[fieldValue] `json:"fields" nullable:"false"`
	Ratings      []rating                `json:"ratings" nullable:"false" doc:"By source"`
	Artwork      []art                   `json:"artwork" nullable:"false" doc:"By kind"`
	Availability *availability           `json:"availability" doc:"The day the Item counts as released and the rule that chose it; null while no date is known"`
}

type availability struct {
	Date  string `json:"date" pattern:"^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$" doc:"YYYY, YYYY-MM or YYYY-MM-DD"`
	Basis string `json:"basis" doc:"The rule that chose the date, in words, such as Digital release in United States; empty when the Content Type names none"`
}

func availabilityOf(t targets.Timing) *availability {
	d, known := t.Released.Get()
	if !known {
		return nil
	}
	return &availability{Date: d.String(), Basis: t.Basis}
}

func treeOf(m artwork.Minter, t works.Tree, timings map[works.ItemID]targets.Timing) tree {
	out := tree{Work: workOf(t.Summary), Fields: fieldsOf(m, t.Fields), Ratings: ratingsOf(t.Ratings), Artwork: artOf(m, t.Artwork), Levels: []level{}, Items: []item{}}
	for _, l := range t.Levels {
		out.Levels = append(out.Levels, level{
			ID: l.ID.String(), ParentID: idOrNull(l.ParentID.UUID), Depth: l.Depth, Key: string(l.Key), Label: l.Label,
			Position: l.Position, WithdrawnAt: timeOrNull(l.WithdrawnAt), Fields: fieldsOf(m, l.Fields),
		})
	}
	for _, it := range t.Items {
		out.Items = append(out.Items, item{
			ID: it.ID.String(), LevelID: idOrNull(it.LevelID.UUID), Key: string(it.Key), Label: it.Label,
			Position: it.Position, WithdrawnAt: timeOrNull(it.WithdrawnAt), Monitored: it.Monitored, Fields: fieldsOf(m, it.Fields),
			Ratings: ratingsOf(it.Ratings), Artwork: artOf(m, it.Artwork), Availability: availabilityOf(timings[it.ID]),
		})
	}
	return out
}

func (h handlers) treeOf(ctx context.Context, by identity.Admin, t works.Tree) (tree, error) {
	timings, err := goals(h.shelf).Timings(ctx, by, t.ContentType, t.ID)
	if err != nil {
		return tree{}, err
	}
	return treeOf(minter(h.shelf), t, timings), nil
}
