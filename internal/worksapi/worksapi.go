package worksapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

type Shelf interface {
	// Works is the Work tree, false unless the Node is a Controller.
	Works() (*works.Works, bool)
}

func open(s Shelf) *works.Works {
	w, ok := s.Works()
	if !ok {
		panic("worksapi: an operation ran on a Node that is no Controller")
	}
	return w
}

var (
	contentTypeNotFound = api.NewCode(http.StatusNotFound, "content_type_not_found", "No Content Type has that id")
	contentTypeDisabled = api.NewCode(http.StatusConflict, "content_type_disabled", "That Content Type is turned off")
	contentTypeInUse    = api.NewCode(http.StatusConflict, "content_type_in_use", "Remove its Works before turning this Content Type off")
	workNotFound        = api.NewCode(http.StatusNotFound, "work_not_found", "No Work has that id")
	workGoneUpstream    = api.NewCode(http.StatusConflict, "work_gone_upstream", "The metadata source no longer has this Work")
	metadataNotFound    = api.NewCode(http.StatusNotFound, "metadata_not_found", "The metadata source has no record with that key")
	metadataUnavailable = api.NewCode(http.StatusBadGateway, "metadata_unavailable", "The metadata source didn't answer. Try again later.")
	metadataInvalid     = api.NewCode(http.StatusBadGateway, "metadata_invalid", "The metadata source sent a record SlipStream can't use")
)

var codes = []struct {
	err  error
	code *api.Code
}{
	{works.ErrNoType, contentTypeNotFound},
	{works.ErrTypeOff, contentTypeDisabled},
	{works.ErrTypeInUse, contentTypeInUse},
	{works.ErrNoWork, workNotFound},
	{works.ErrGoneUpstream, workGoneUpstream},
	{works.ErrNotFound, metadataNotFound},
	{works.ErrUnavailable, metadataUnavailable},
	{works.ErrInvalid, metadataInvalid},
}

func answer(err error) error {
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
	Kind  string `json:"kind" enum:"text,number,date,duration"`
}

type contentType struct {
	ID      string      `json:"id" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Name    string      `json:"name"`
	Enabled bool        `json:"enabled"`
	Work    noun        `json:"work"`
	Levels  []noun      `json:"levels" nullable:"false" doc:"Outermost first; empty when Items hang off the Work"`
	Item    noun        `json:"item"`
	Fields  []fieldSpec `json:"fields" nullable:"false"`
}

func contentTypeOf(t works.ContentType) contentType {
	out := contentType{
		ID: string(t.ID), Name: t.Name, Enabled: t.Enabled,
		Work: noun(t.Work), Item: noun(t.Item), Levels: []noun{}, Fields: []fieldSpec{},
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
	return "kind", []any{textValue{}, numberValue{}, dateValue{}, durationValue{}}
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

var valueOf = map[works.FieldKind]func(works.FieldValue) any{
	works.KindText:   func(v works.FieldValue) any { return textValue{Key: v.Key, Kind: v.Kind.String(), Text: v.Text} },
	works.KindNumber: func(v works.FieldValue) any { return numberValue{Key: v.Key, Kind: v.Kind.String(), Number: v.Number} },
	works.KindDate: func(v works.FieldValue) any {
		return dateValue{Key: v.Key, Kind: v.Kind.String(), Date: v.Date.String()}
	},
	works.KindDuration: func(v works.FieldValue) any {
		return durationValue{Key: v.Key, Kind: v.Kind.String(), Ms: v.Duration.Milliseconds()}
	},
}

func fieldsOf(vs []works.FieldValue) []api.OneOf[fieldValue] {
	out := make([]api.OneOf[fieldValue], len(vs))
	for i, v := range vs {
		out[i] = api.NewOneOf[fieldValue](valueOf[v.Kind](v))
	}
	return out
}

type work struct {
	ID          string    `json:"id" format:"uuid"`
	ContentType string    `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Key         string    `json:"key"`
	Label       string    `json:"label"`
	SortLabel   string    `json:"sortLabel"`
	AddedAt     time.Time `json:"addedAt"`
	RefreshedAt time.Time `json:"refreshedAt"`
}

func workOf(s works.Summary) work {
	return work{
		ID: s.ID.String(), ContentType: string(s.ContentType), Key: string(s.Key), Label: s.Label, SortLabel: s.SortLabel,
		AddedAt: s.AddedAt.Time(), RefreshedAt: s.RefreshedAt.Time(),
	}
}

type tree struct {
	work
	Fields []api.OneOf[fieldValue] `json:"fields" nullable:"false"`
	Levels []level                 `json:"levels" nullable:"false" doc:"By depth, then their parent's order, live before withdrawn, then position"`
	Items  []item                  `json:"items" nullable:"false" doc:"By their level's order, live before withdrawn, then position"`
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
	ID          string                  `json:"id" format:"uuid"`
	LevelID     *string                 `json:"levelId" format:"uuid" doc:"Null when the Content Type declares no levels"`
	Key         string                  `json:"key"`
	Label       string                  `json:"label"`
	Position    int                     `json:"position" minimum:"1"`
	WithdrawnAt *time.Time              `json:"withdrawnAt" doc:"When the source stopped listing it; null while it lists it"`
	Fields      []api.OneOf[fieldValue] `json:"fields" nullable:"false"`
}

func treeOf(t works.Tree) tree {
	out := tree{work: workOf(t.Summary), Fields: fieldsOf(t.Fields), Levels: []level{}, Items: []item{}}
	for _, l := range t.Levels {
		out.Levels = append(out.Levels, level{
			ID: l.ID.String(), ParentID: idOrNull(l.ParentID.UUID), Depth: l.Depth, Key: string(l.Key), Label: l.Label,
			Position: l.Position, WithdrawnAt: timeOrNull(l.WithdrawnAt), Fields: fieldsOf(l.Fields),
		})
	}
	for _, it := range t.Items {
		out.Items = append(out.Items, item{
			ID: it.ID.String(), LevelID: idOrNull(it.LevelID.UUID), Key: string(it.Key), Label: it.Label,
			Position: it.Position, WithdrawnAt: timeOrNull(it.WithdrawnAt), Fields: fieldsOf(it.Fields),
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
	Key    string                  `json:"key"`
	Label  string                  `json:"label"`
	Fields []api.OneOf[fieldValue] `json:"fields" nullable:"false" doc:"The Work's fields"`
	WorkID *string                 `json:"workId" format:"uuid" doc:"The Work, once it is in the library"`
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

type candidateList struct {
	Candidates []candidate `json:"candidates" nullable:"false" doc:"Best first"`
}

type candidateListOutput struct{ Body candidateList }

type addInput struct {
	Body newWork
}

type newWork struct {
	ContentType string `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Key         string `json:"key" minLength:"1" maxLength:"200" doc:"The Work's key at its metadata source"`
}

type added struct {
	Created bool `json:"created" doc:"False when the Work was already in the library"`
	Work    tree `json:"work"`
}

type addedOutput struct{ Body added }

type listInput struct {
	ContentType string `query:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
}

type workList struct {
	Works []work `json:"works" nullable:"false" doc:"By sort label, then id"`
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
		Summary: "Turn a Content Type on or off; off only while it holds no Works",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound, contentTypeInUse},
	}, h.setContentTypeEnabled)

	api.Register(o, api.Admin, api.Op{
		ID:      "searchCatalog",
		Method:  http.MethodGet,
		Path:    "/content-types/{contentType}/catalog",
		Summary: "Search a Content Type's metadata source",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound, contentTypeDisabled, metadataUnavailable},
	}, h.searchCatalog)

	api.Register(o, api.Admin, api.Op{
		ID:      "addWork",
		Method:  http.MethodPost,
		Path:    "/works",
		Summary: "Add a Work by its key with its whole tree, or get the Work already added",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound, contentTypeDisabled, metadataNotFound, metadataUnavailable, metadataInvalid},
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
		Errors:  []*api.Code{workNotFound, workGoneUpstream, metadataUnavailable, metadataInvalid},
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
	out := candidateList{Candidates: []candidate{}}
	for _, c := range found {
		out.Candidates = append(out.Candidates, candidate{Key: string(c.Key), Label: c.Label, Fields: fieldsOf(c.Fields), WorkID: idOrNull(c.WorkID.UUID)})
	}
	return &candidateListOutput{Body: out}, nil
}

func (h handlers) addWork(ctx context.Context, by identity.Admin, in *addInput) (*addedOutput, error) {
	key, err := works.ParseKey(in.Body.Key)
	if err != nil {
		return nil, api.Invalid
	}
	t, created, err := open(h.shelf).Add(ctx, by, works.TypeID(in.Body.ContentType), key)
	if err != nil {
		return nil, answer(err)
	}
	return &addedOutput{Body: added{Created: created, Work: treeOf(t)}}, nil
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
	out := workList{Works: []work{}}
	for _, s := range list {
		out.Works = append(out.Works, workOf(s))
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
	return &treeOutput{Body: treeOf(t)}, nil
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
	return &treeOutput{Body: treeOf(t)}, nil
}
