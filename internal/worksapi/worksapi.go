package worksapi

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type Shelf interface {
	// Works is the Work tree, false unless the Node is a Controller.
	Works() (*works.Works, bool)
	// Minter signs the artwork URLs the answers carry, false unless the
	// Node is a Controller.
	Minter() (artwork.Minter, bool)
	// Targets answers each Item's availability, false unless the Node is a
	// Controller.
	Targets() (*targets.Targets, bool)
}

func open(s Shelf) *works.Works {
	w, ok := s.Works()
	if !ok {
		panic("worksapi: an operation ran on a Node that is no Controller")
	}
	return w
}

func minter(s Shelf) artwork.Minter {
	m, ok := s.Minter()
	if !ok {
		panic("worksapi: an operation ran on a Node that is no Controller")
	}
	return m
}

func goals(s Shelf) *targets.Targets {
	t, ok := s.Targets()
	if !ok {
		panic("worksapi: an operation ran on a Node that is no Controller")
	}
	return t
}

var (
	contentTypeNotFound = api.NewCode(http.StatusNotFound, "content_type_not_found", "No Content Type has that id")
	contentTypeDisabled = api.NewCode(http.StatusConflict, "content_type_disabled", "That Content Type is turned off")
	contentTypeInUse    = api.NewCode(http.StatusConflict, "content_type_in_use", "Remove its Works before turning this Content Type off")
	contentTypeLast     = api.NewCode(http.StatusConflict, "content_type_last", "Turn on another Content Type before turning this one off")
	workNotFound        = api.NewCode(http.StatusNotFound, "work_not_found", "No Work has that id")
	workGoneUpstream    = api.NewCode(http.StatusConflict, "work_gone_upstream", "The metadata source no longer has this Work")
	metadataNotFound    = api.NewCode(http.StatusNotFound, "metadata_not_found", "The metadata source has no record with that key")
	metadataUnavailable = api.NewCode(http.StatusBadGateway, "metadata_unavailable", "The metadata source didn't respond. Try again later.")
	metadataInvalid     = api.NewCode(http.StatusBadGateway, "metadata_invalid", "The metadata source sent a record SlipStream can't use")
	metadataRefused     = api.NewCode(http.StatusBadGateway, "metadata_refused", "The metadata source refused SlipStream's key")
	profileNotFound     = api.NewCode(http.StatusUnprocessableEntity, "profile_not_found", "No Profile has that id")
	profileWrongType    = api.NewCode(http.StatusUnprocessableEntity, "profile_wrong_type", "The Profile is for another Content Type")
	rootNotFound        = api.NewCode(http.StatusNotFound, "root_not_found", "No root has that id")
	rootWrongType       = api.NewCode(http.StatusUnprocessableEntity, "root_wrong_type", "The root holds another Content Type's Works")
)

var codes = []struct {
	err  error
	code *api.Code
}{
	{works.ErrNoType, contentTypeNotFound},
	{works.ErrTypeOff, contentTypeDisabled},
	{works.ErrTypeInUse, contentTypeInUse},
	{works.ErrLastType, contentTypeLast},
	{works.ErrNoWork, workNotFound},
	{works.ErrGoneUpstream, workGoneUpstream},
	{works.ErrNotFound, metadataNotFound},
	{works.ErrUnavailable, metadataUnavailable},
	{works.ErrInvalid, metadataInvalid},
	{works.ErrRefused, metadataRefused},
	{targets.ErrNoProfile, profileNotFound},
	{targets.ErrProfileWrongType, profileWrongType},
	{roots.ErrNotFound, rootNotFound},
	{roots.ErrWrongType, rootWrongType},
}

func answer(err error) error {
	if source, ok := errors.AsType[*works.SourceError](err); ok {
		switch {
		case errors.Is(err, works.ErrRefused):
			return metadataRefused.Because(source.Source + " refused SlipStream's key")
		case errors.Is(err, works.ErrUnavailable):
			return metadataUnavailable.Because(source.Error())
		}
	}
	for _, c := range codes {
		if errors.Is(err, c.err) {
			return c.code
		}
	}
	return err
}

type noun struct {
	One  string `json:"one"`
	Many string `json:"many"`
}

type fieldSpec struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Tier  string `json:"tier" enum:"work,level,item"`
	Depth *int   `json:"depth,omitempty" minimum:"1" doc:"The level's depth, 1 for the outermost; present only on tier level"`
	Kind  string `json:"kind" enum:"text,number,date,duration,texts,people,link"`
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
}

func contentTypeOf(t works.ContentType) contentType {
	out := contentType{
		ID: string(t.ID), Name: t.Name, Icon: t.Icon.String(), Enabled: t.Enabled,
		Work: noun(t.Work), Item: noun(t.Item), Levels: []noun{}, Fields: []fieldSpec{}, Artwork: []artworkKind{},
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
		spec := fieldSpec{Key: f.Key, Label: f.Label, Kind: f.Kind.String()}
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

func urlOrNull(m artwork.Minter, r artwork.Ref) *string {
	if r.IsZero() {
		return nil
	}
	u := m.URL(r)
	return &u
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

func idOrNull(id storage.UUID) *string {
	if id.IsZero() {
		return nil
	}
	s := id.String()
	return &s
}

func timeOrNull(t storage.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	v := t.Time()
	return &v
}

type candidate struct {
	Key       string                  `json:"key"`
	Label     string                  `json:"label"`
	Original  string                  `json:"original" doc:"The label in its original language; empty when it is the label"`
	Overview  string                  `json:"overview" doc:"One line; empty when the source has none"`
	Fields    []api.OneOf[fieldValue] `json:"fields" nullable:"false" doc:"The Work's fields"`
	PosterURL *string                 `json:"posterUrl" doc:"The artwork route for the result's poster, without a size; null when the source has none"`
	WorkID    *string                 `json:"workId" format:"uuid" doc:"The Work, once it is in the library"`
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

type catalogInput struct {
	ContentType string `path:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Query       string `query:"query" required:"true" minLength:"1"`
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

type candidateList struct {
	Candidates []candidate `json:"candidates" nullable:"false" doc:"Best first"`
}

type candidateListOutput struct{ Body candidateList }

type recordInput struct {
	ContentType string `path:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Key         string `path:"key" minLength:"1" maxLength:"200" doc:"The Work's key at its metadata source"`
}

type previewed struct {
	InLibrary bool `json:"inLibrary" doc:"Whether the Work is in the library; while it isn't, the ids in work name nothing"`
	Work      tree `json:"work"`
}

type previewedOutput struct{ Body previewed }

type addInput struct {
	Body newWork
}

type newWork struct {
	ContentType string `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Key         string `json:"key" minLength:"1" maxLength:"200" doc:"The Work's key at its metadata source"`
	Monitor     bool   `json:"monitor,omitempty" doc:"Monitor a new Work's Items; adding a Work already in the library changes nothing"`
	ProfileID   string `json:"profileId,omitempty" format:"uuid" doc:"The Profile of the Content Type a new Work wants from; left out, the type's default"`
	RootID      string `json:"rootId,omitempty" format:"uuid" doc:"The root of the Content Type a new Work lives in; left out, the type's default, or none while the type has no root"`
}

type added struct {
	Created bool `json:"created" doc:"False when the Work was already in the library"`
	Work    tree `json:"work"`
}

type addedOutput struct{ Body added }

type listInput struct {
	ContentType string `query:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
}

type listedWork struct {
	Work
	Fields  []api.OneOf[fieldValue] `json:"fields" nullable:"false" doc:"The Work's own fields"`
	Artwork []art                   `json:"artwork" nullable:"false" doc:"The Work's own images, by kind"`
}

type workList struct {
	Works []listedWork `json:"works" nullable:"false" doc:"By sort label, then id"`
}

type workListOutput struct{ Body workList }

type workInput struct {
	WorkID string `path:"workId" format:"uuid"`
}

type treeOutput struct{ Body tree }

func workID(s string) (works.WorkID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return works.WorkID{}, workNotFound
	}
	return works.WorkID{UUID: u}, nil
}

func Routes(r *api.Router, shelf Shelf) {
	o := r.Owner("works", api.Account)
	h := handlers{shelf}

	api.Register(o, api.Account, api.Op{
		ID:      "listContentTypes",
		Method:  http.MethodGet,
		Path:    "/content-types",
		Summary: "List the Content Types this install has, in id order, off ones included",
		Status:  http.StatusOK,
	}, h.listContentTypes)

	api.Register(o, api.Admin, api.Op{
		ID:      "setContentTypeEnabled",
		Method:  http.MethodPut,
		Path:    "/content-types/{contentType}/enabled",
		Summary: "Turn a Content Type on or off. It can be off only while it holds no Works",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound, contentTypeInUse, contentTypeLast},
	}, h.setContentTypeEnabled)

	api.Register(o, api.Admin, api.Op{
		ID:      "getContentTypeSettings",
		Method:  http.MethodGet,
		Path:    "/content-types/{contentType}/settings",
		Summary: "Get a Content Type's own settings",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound},
	}, h.getContentTypeSettings)

	api.Register(o, api.Admin, api.Op{
		ID:      "updateContentTypeSettings",
		Method:  http.MethodPatch,
		Path:    "/content-types/{contentType}/settings",
		Summary: "Change a Content Type's own settings by key, all or none. Every Work of the type may then read differently",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound, settingapi.ErrOutOfRange, settingapi.ErrUnknown},
	}, h.updateContentTypeSettings)

	api.Register(o, api.Admin, api.Op{
		ID:      "searchCatalog",
		Method:  http.MethodGet,
		Path:    "/content-types/{contentType}/catalog",
		Summary: "Search a Content Type's metadata source",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound, contentTypeDisabled, metadataUnavailable, metadataRefused},
	}, h.searchCatalog)

	api.Register(o, api.Admin, api.Op{
		ID:      "previewCatalogRecord",
		Method:  http.MethodGet,
		Path:    "/content-types/{contentType}/catalog/{key}",
		Summary: "Read a record of a Content Type's metadata source as the Work adding it would make, without adding it or reading ratings",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound, contentTypeDisabled, metadataNotFound, metadataUnavailable, metadataInvalid, metadataRefused},
	}, h.previewCatalogRecord)

	api.Register(o, api.Admin, api.Op{
		ID:      "addWork",
		Method:  http.MethodPost,
		Path:    "/works",
		Summary: "Add a Work by its key with its whole tree, or get the Work already added",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound, contentTypeDisabled, profileNotFound, profileWrongType, rootNotFound, rootWrongType, metadataNotFound, metadataUnavailable, metadataInvalid, metadataRefused},
	}, h.addWork)

	api.Register(o, api.Admin, api.Op{
		ID:      "listWorks",
		Method:  http.MethodGet,
		Path:    "/works",
		Summary: "List the Works, or one Content Type's",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound},
	}, h.listWorks)

	api.Register(o, api.Admin, api.Op{
		ID:      "getWork",
		Method:  http.MethodGet,
		Path:    "/works/{workId}",
		Summary: "Get a Work's tree with each node's fields",
		Status:  http.StatusOK,
		Errors:  []*api.Code{workNotFound},
	}, h.getWork)

	api.Register(o, api.Admin, api.Op{
		ID:      "refreshWork",
		Method:  http.MethodPost,
		Path:    "/works/{workId}/refresh",
		Summary: "Refresh a Work from its metadata source, keeping every id",
		Status:  http.StatusOK,
		Errors:  []*api.Code{workNotFound, workGoneUpstream, metadataUnavailable, metadataInvalid, metadataRefused},
	}, h.refreshWork)
}

type handlers struct{ shelf Shelf }

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

func (h handlers) searchCatalog(ctx context.Context, by identity.Admin, in *catalogInput) (*candidateListOutput, error) {
	found, err := open(h.shelf).Search(ctx, by, works.TypeID(in.ContentType), in.Query)
	if err != nil {
		return nil, answer(err)
	}
	m := minter(h.shelf)
	out := candidateList{Candidates: []candidate{}}
	for _, c := range found {
		out.Candidates = append(out.Candidates, candidate{
			Key: string(c.Key), Label: c.Label, Original: c.Original, Overview: c.Overview, Fields: fieldsOf(m, c.Fields),
			PosterURL: urlOrNull(m, c.Poster), WorkID: idOrNull(c.WorkID.UUID),
		})
	}
	return &candidateListOutput{Body: out}, nil
}

func (h handlers) previewCatalogRecord(ctx context.Context, by identity.Admin, in *recordInput) (*previewedOutput, error) {
	key, err := works.ParseKey(in.Key)
	if err != nil {
		return nil, api.Invalid
	}
	var timings map[works.ItemID]targets.Timing
	t, stored, err := open(h.shelf).Preview(ctx, by, works.TypeID(in.ContentType), key, func(ctx context.Context, r storage.Reader[storage.Library], t works.Tree) error {
		var err error
		timings, err = goals(h.shelf).TimingsIn(ctx, r, t.ContentType, []works.WorkID{t.ID})
		return err
	})
	if err != nil {
		return nil, answer(err)
	}
	return &previewedOutput{Body: previewed{InLibrary: stored, Work: treeOf(minter(h.shelf), t, timings)}}, nil
}

func (h handlers) addWork(ctx context.Context, by identity.Admin, in *addInput) (*addedOutput, error) {
	key, err := works.ParseKey(in.Body.Key)
	if err != nil {
		return nil, api.Invalid
	}
	var profile *targets.ProfileID
	if in.Body.ProfileID != "" {
		u, err := storage.ParseUUID(in.Body.ProfileID)
		if err != nil {
			return nil, profileNotFound
		}
		profile = &targets.ProfileID{UUID: u}
	}
	var root *roots.ID
	if in.Body.RootID != "" {
		u, err := storage.ParseUUID(in.Body.RootID)
		if err != nil {
			return nil, rootNotFound
		}
		root = &roots.ID{UUID: u}
	}
	t, created, err := open(h.shelf).Add(ctx, by, works.TypeID(in.Body.ContentType), key, works.Adding{
		Monitor: in.Body.Monitor,
		Also: func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
			if err := targets.Assign(ctx, tx, w, profile); err != nil {
				return err
			}
			return roots.Assign(ctx, tx, w, root)
		},
	})
	if err != nil {
		return nil, answer(err)
	}
	out, err := h.treeOf(ctx, by, t)
	if err != nil {
		return nil, err
	}
	return &addedOutput{Body: added{Created: created, Work: out}}, nil
}

func (h handlers) listWorks(ctx context.Context, by identity.Admin, in *listInput) (*workListOutput, error) {
	var only *works.TypeID
	if in.ContentType != "" {
		t := works.TypeID(in.ContentType)
		only = &t
	}
	list, err := open(h.shelf).List(ctx, by, only)
	if err != nil {
		return nil, answer(err)
	}
	m := minter(h.shelf)
	out := workList{Works: []listedWork{}}
	for _, l := range list {
		out.Works = append(out.Works, listedWork{Work: workOf(l.Summary), Fields: fieldsOf(m, l.Fields), Artwork: artOf(m, l.Artwork)})
	}
	return &workListOutput{Body: out}, nil
}

func (h handlers) getWork(ctx context.Context, by identity.Admin, in *workInput) (*treeOutput, error) {
	id, err := workID(in.WorkID)
	if err != nil {
		return nil, err
	}
	t, err := open(h.shelf).Get(ctx, by, id)
	if err != nil {
		return nil, answer(err)
	}
	out, err := h.treeOf(ctx, by, t)
	if err != nil {
		return nil, err
	}
	return &treeOutput{Body: out}, nil
}

func (h handlers) refreshWork(ctx context.Context, by identity.Admin, in *workInput) (*treeOutput, error) {
	id, err := workID(in.WorkID)
	if err != nil {
		return nil, err
	}
	t, err := open(h.shelf).Refresh(ctx, by, id)
	if err != nil {
		return nil, answer(err)
	}
	out, err := h.treeOf(ctx, by, t)
	if err != nil {
		return nil, err
	}
	return &treeOutput{Body: out}, nil
}

func (h handlers) treeOf(ctx context.Context, by identity.Admin, t works.Tree) (tree, error) {
	timings, err := goals(h.shelf).Timings(ctx, by, t.ContentType, t.ID)
	if err != nil {
		return tree{}, err
	}
	return treeOf(minter(h.shelf), t, timings), nil
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
