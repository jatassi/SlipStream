// Package namingapi is each Content Type's naming settings over the API:
// the templates and modes, their tokens and a preview of a draft, with a
// Work's files and tested file names named under it.
package namingapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

type Namer interface {
	// Naming is each Content Type's naming, false unless the Node is a
	// Controller.
	Naming() (*naming.Naming, bool)
	// Files is the library's files, which a preview names a Work's files
	// from, false unless the Node is a Controller.
	Files() (*files.Files, bool)
	// Imports is Pending imports, which a preview names tested file names
	// through, false unless the Node is a Controller.
	Imports() (*imports.Imports, bool)
}

func open[T any](of func() (T, bool)) T {
	v, ok := of()
	if !ok {
		panic("namingapi: an operation ran on a Node that is no Controller")
	}
	return v
}

var (
	contentTypeNotFound = api.NewCode(http.StatusNotFound, "content_type_not_found", "No Content Type has that id")
	workNotFound        = api.NewCode(http.StatusNotFound, "work_not_found", "No Work has that id")
	templateRefused     = api.NewCode(http.StatusUnprocessableEntity, "naming_template_refused", "The naming settings don't pass the save check")
)

type problem struct {
	Key      string `json:"key" doc:"The setting key, such as <type>.naming.file.standard"`
	Sentence string `json:"sentence" doc:"What is wrong, then what passes"`
}

type typeNaming struct {
	ContentType string                        `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Settings    []api.OneOf[settingapi.Field] `json:"settings" nullable:"false" doc:"The rename switch, the folder templates outermost first, the file templates, the colon mode and its text, and the multi-episode style when the type's file templates number its Items"`
	Problems    []problem                     `json:"problems" nullable:"false" doc:"What the save check refuses in the stored settings once renaming is on; empty when every file of a Work shows as one item on Plex, Jellyfin and Emby"`
}

type namingList struct {
	ContentTypes []typeNaming `json:"contentTypes" nullable:"false" doc:"In id order"`
}

type namingListOutput struct{ Body namingList }

type typeNamingOutput struct{ Body typeNaming }

type contentTypeInput struct {
	ContentType string `path:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
}

type setNamingInput struct {
	ContentType string `path:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Body        struct {
		Values map[string]json.RawMessage `json:"values" doc:"Each setting's new value by key, all or none; null resets one to its default"`
	}
}

type token struct {
	Name     string   `json:"name"`
	Aliases  []string `json:"aliases" nullable:"false" doc:"Other names that match the same token"`
	Kind     string   `json:"kind" enum:"title,year,number,date,text,id,list"`
	Scope    string   `json:"scope" enum:"work,level,item,file" doc:"The tier the value comes from"`
	Group    string   `json:"group" enum:"work,level,item,target,release,media" doc:"The token builder's heading"`
	Example  string   `json:"example" doc:"A value the token prints"`
	About    string   `json:"about" doc:"What the token prints, as a sentence"`
	Modifier *string  `json:"modifier" enum:"pad,cut,languages,formats,format" doc:"What it takes after a colon; null when it takes nothing"`
	Tags     []string `json:"tags" nullable:"false" doc:"The words of the tags that may wrap it, as in {imdb-{ImdbId}}"`
}

type fieldTokens struct {
	Key    string  `json:"key" doc:"The template setting's key"`
	Tokens []token `json:"tokens" nullable:"false" doc:"Every token the template may name, the Content Type's own first"`
}

type tokenList struct {
	Fields []fieldTokens `json:"fields" nullable:"false" doc:"Each template setting, in the settings' order"`
}

type tokenListOutput struct{ Body tokenList }

type namingPreviewInput struct {
	ContentType string `path:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Body        struct {
		Draft  map[string]json.RawMessage `json:"draft" doc:"Values over the stored settings by key, as a save would send them; null reads a setting's default"`
		WorkID string                     `json:"workId,omitempty" format:"uuid" doc:"A Work whose files to name under the draft. A Work of another Content Type is named under its own stored settings."`
		Names  []testName                 `json:"names,omitempty" maxItems:"100" doc:"File names to name under the draft, each in the Work it matches"`
	}
}

type testName struct {
	Name   string `json:"name" minLength:"1" doc:"A file name, or a path inside the Work's folder"`
	WorkID string `json:"workId" format:"uuid" doc:"The Work the name matches, as POST /releases/parse with kind file reads it"`
}

type samplePath struct {
	Label string `json:"label" doc:"The sample, and for a file template the file it names"`
	Path  string `json:"path" doc:"The path in the root, with \"/\" between folders"`
}

type fieldPreview struct {
	Key      string       `json:"key"`
	Samples  []samplePath `json:"samples" nullable:"false" doc:"The full paths a folder or file template names for the type's samples, with renaming on; empty for any other setting and for a template that doesn't parse"`
	Problems []string     `json:"problems" nullable:"false" doc:"What the save check refuses in the setting, each a sentence; empty when it passes"`
}

type workFile struct {
	From   string `json:"from" doc:"Where the file is now, relative to its root"`
	To     string `json:"to" doc:"Where the draft puts it; empty when its name doesn't fit"`
	Detail string `json:"detail,omitempty" doc:"Why its name doesn't fit, then what to do"`
}

type unplaced struct {
	Code  string `json:"code" enum:"no_number,no_such_item,no_rule" doc:"Why the Content Type can't tell which Item the name holds"`
	Names string `json:"names,omitempty" doc:"For no_such_item: the Item the name names, such as S03E14"`
}

type testedName struct {
	Name     string    `json:"name"`
	Path     string    `json:"path" doc:"Where a file of that name would go if a library scan found it now, relative to the root; empty when it would go nowhere"`
	Unplaced *unplaced `json:"unplaced,omitempty" doc:"Why it would go nowhere, when no Item of the Work holds the name"`
	Detail   string    `json:"detail,omitempty" doc:"Why it would go nowhere, then what to do, for any other reason"`
}

type preview struct {
	Fields []fieldPreview `json:"fields" nullable:"false" doc:"Every setting, in the settings' order"`
	Work   []workFile     `json:"work" nullable:"false" doc:"Each live file of workId's Work, in the order it was added; empty without workId and while a setting has a problem"`
	Tested []testedName   `json:"tested" nullable:"false" doc:"Each of names, in order; empty while a setting has a problem"`
}

type previewOutput struct{ Body preview }

func Routes(r *api.Router, n Namer) {
	o := r.Owner("naming", api.Admin)
	h := handlers{n}

	api.Register(o, api.Admin, api.Op{
		ID:      "listNaming",
		Method:  http.MethodGet,
		Path:    "/naming",
		Summary: "List each Content Type's naming settings, with what the save check refuses once renaming is on",
		Status:  http.StatusOK,
	}, h.list)

	api.Register(o, api.Admin, api.Op{
		ID:      "setNaming",
		Method:  http.MethodPatch,
		Path:    "/naming/{contentType}",
		Summary: "Change a Content Type's naming settings by key, all or none. Every setting must pass the save check as the change leaves them",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound, templateRefused, settingapi.ErrUnknown},
	}, h.set)

	api.Register(o, api.Admin, api.Op{
		ID:      "listNamingTokens",
		Method:  http.MethodGet,
		Path:    "/naming/{contentType}/tokens",
		Summary: "List the tokens each of a Content Type's templates may name",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound},
	}, h.tokens)

	api.Register(o, api.Admin, api.Op{
		ID:      "previewNaming",
		Method:  http.MethodPost,
		Path:    "/naming/{contentType}/preview",
		Summary: "Name a Content Type's samples, a Work's files and tested file names under a draft of its naming settings, with what the save check refuses. Nothing is stored",
		Status:  http.StatusOK,
		Errors:  []*api.Code{contentTypeNotFound, workNotFound, settingapi.ErrUnknown},
	}, h.preview)
}

type handlers struct{ namer Namer }

func (h handlers) list(ctx context.Context, _ identity.Admin, _ *struct{}) (*namingListOutput, error) {
	nm := open(h.namer.Naming)
	out := namingList{ContentTypes: []typeNaming{}}
	for _, t := range nm.Types() {
		tn, err := typeNamingOf(ctx, nm, t)
		if err != nil {
			return nil, err
		}
		out.ContentTypes = append(out.ContentTypes, tn)
	}
	return &namingListOutput{Body: out}, nil
}

func (h handlers) set(ctx context.Context, _ identity.Admin, in *setNamingInput) (*typeNamingOutput, error) {
	nm, t, err := h.typeOf(in.ContentType)
	if err != nil {
		return nil, err
	}
	if err := nm.Panel(t).Apply(ctx, in.Body.Values); err != nil {
		return nil, refused(err)
	}
	tn, err := typeNamingOf(ctx, nm, t)
	if err != nil {
		return nil, err
	}
	return &typeNamingOutput{Body: tn}, nil
}

func (h handlers) tokens(_ context.Context, _ identity.Admin, in *contentTypeInput) (*tokenListOutput, error) {
	nm, t, err := h.typeOf(in.ContentType)
	if err != nil {
		return nil, err
	}
	fields, err := nm.Catalog(t)
	if err != nil {
		return nil, err
	}
	out := tokenList{Fields: make([]fieldTokens, len(fields))}
	for i, f := range fields {
		ft := fieldTokens{Key: f.Key, Tokens: make([]token, len(f.Tokens))}
		for j, e := range f.Tokens {
			ft.Tokens[j] = tokenOf(e)
		}
		out.Fields[i] = ft
	}
	return &tokenListOutput{Body: out}, nil
}

func (h handlers) preview(ctx context.Context, by identity.Admin, in *namingPreviewInput) (*previewOutput, error) {
	nm, t, err := h.typeOf(in.ContentType)
	if err != nil {
		return nil, err
	}
	p, err := nm.Preview(ctx, t, in.Body.Draft)
	if err != nil {
		return nil, settingapi.Refused(err)
	}
	out := preview{Fields: make([]fieldPreview, len(p.Fields)), Work: []workFile{}, Tested: []testedName{}}
	passes := true
	for i, f := range p.Fields {
		fp := fieldPreview{Key: f.Key, Samples: []samplePath{}, Problems: []string{}}
		for _, s := range f.Samples {
			fp.Samples = append(fp.Samples, samplePath{Label: s.Label, Path: s.Path})
		}
		fp.Problems = append(fp.Problems, f.Problems...)
		passes = passes && len(f.Problems) == 0
		out.Fields[i] = fp
	}
	if !passes {
		return &previewOutput{Body: out}, nil
	}
	draft := &files.Draft{Type: t, Patch: in.Body.Draft}
	if in.Body.WorkID != "" {
		if out.Work, err = h.workFiles(ctx, by, in.Body.WorkID, draft); err != nil {
			return nil, err
		}
	}
	for _, n := range in.Body.Names {
		tested, err := h.tested(ctx, by, n, draft)
		if err != nil {
			return nil, err
		}
		out.Tested = append(out.Tested, tested)
	}
	return &previewOutput{Body: out}, nil
}

// workFiles is each live file of the Work id names, named under draft.
func (h handlers) workFiles(ctx context.Context, by identity.Admin, id string, draft *files.Draft) ([]workFile, error) {
	w, err := workOf(id)
	if err != nil {
		return nil, err
	}
	named, err := open(h.namer.Files).NamesOf(ctx, by, w, draft)
	if errors.Is(err, works.ErrNoWork) {
		return nil, workNotFound
	}
	if err != nil {
		return nil, err
	}
	out := make([]workFile, len(named))
	for i, n := range named {
		out[i] = workFile{From: n.From, To: n.To}
		if n.Long != nil {
			out[i].Detail = n.Long.Sentence()
		}
	}
	return out, nil
}

// tested is where a file named n would go in n's Work under draft, or why
// it would go nowhere.
func (h handlers) tested(ctx context.Context, by identity.Admin, n testName, draft *files.Draft) (testedName, error) {
	w, err := workOf(n.WorkID)
	if err != nil {
		return testedName{}, err
	}
	out := testedName{Name: n.Name}
	out.Path, err = open(h.namer.Imports).Imagine(ctx, by, w, n.Name, draft)
	var (
		noItem *imports.UnplacedError
		taken  *files.NameTakenError
		long   *naming.TooLongError
	)
	switch {
	case errors.Is(err, works.ErrNoWork):
		return testedName{}, workNotFound
	case errors.As(err, &noItem):
		out.Unplaced = &unplaced{Code: string(noItem.Reason.Code), Names: noItem.Reason.Names}
	case errors.Is(err, imports.ErrNoQuality):
		out.Detail = "SlipStream reads no quality from this name. Add a quality such as Bluray-1080p, or a video extension such as .mkv."
	case errors.Is(err, roots.ErrNoRoot):
		out.Detail = "The Work is in no library root. Add a library root for its Content Type, then test the name again."
	case errors.As(err, &taken):
		out.Detail = fmt.Sprintf("Another file of the Work is already at %q. Turn on renaming to give each file its own name.", taken.Path)
	case errors.As(err, &long):
		out.Detail = long.Sentence()
	case err != nil:
		return testedName{}, err
	}
	return out, nil
}

func workOf(id string) (works.WorkID, error) {
	u, err := storage.ParseUUID(id)
	if err != nil {
		return works.WorkID{}, workNotFound
	}
	return works.WorkID{UUID: u}, nil
}

// typeOf is the Naming and the Content Type id, which must be one Naming
// names files for.
func (h handlers) typeOf(id string) (*naming.Naming, works.TypeID, error) {
	nm, t := open(h.namer.Naming), works.TypeID(id)
	if !slices.Contains(nm.Types(), t) {
		return nil, "", contentTypeNotFound
	}
	return nm, t, nil
}

func typeNamingOf(ctx context.Context, nm *naming.Naming, t works.TypeID) (typeNaming, error) {
	fields, err := nm.Panel(t).Fields(ctx)
	if err != nil {
		return typeNaming{}, err
	}
	problems, err := nm.Problems(ctx, t)
	if err != nil {
		return typeNaming{}, err
	}
	out := typeNaming{ContentType: string(t), Settings: settingapi.FieldsOf(fields), Problems: []problem{}}
	for _, p := range problems {
		out.Problems = append(out.Problems, problem{Key: p.Key, Sentence: p.Sentence})
	}
	return out, nil
}

// refused answers a *naming.RefusedError with one field error per
// problem, at the value's key.
func refused(err error) error {
	r, ok := errors.AsType[*naming.RefusedError](err)
	if !ok {
		return settingapi.Refused(err)
	}
	fields := make([]api.FieldError, len(r.Problems))
	for i, p := range r.Problems {
		fields[i] = api.FieldError{Location: "body.values." + p.Key, Message: p.Sentence}
	}
	return templateRefused.Fields(fields...)
}

var (
	kinds = map[naming.Kind]string{
		naming.KindTitle: "title", naming.KindYear: "year", naming.KindNumber: "number", naming.KindDate: "date",
		naming.KindText: "text", naming.KindID: "id", naming.KindList: "list",
	}
	scopes = map[naming.Scope]string{naming.WorkScope: "work", naming.LevelScope: "level", naming.ItemScope: "item", naming.FileScope: "file"}
	groups = map[naming.Group]string{
		naming.GroupWork: "work", naming.GroupLevel: "level", naming.GroupItem: "item",
		naming.GroupTarget: "target", naming.GroupRelease: "release", naming.GroupMedia: "media",
	}
	modifiers = map[naming.Modifier]string{
		naming.ModPad: "pad", naming.ModCut: "cut", naming.ModLanguages: "languages", naming.ModFormats: "formats", naming.ModFormat: "format",
	}
)

func tokenOf(e naming.Entry) token {
	out := token{
		Name: e.Name, Aliases: append([]string{}, e.Aliases...), Kind: kinds[e.Kind], Scope: scopes[e.Scope], Group: groups[e.Group],
		Example: e.Example, About: e.About, Tags: append([]string{}, e.Tags...),
	}
	if m, ok := modifiers[e.Modifier]; ok {
		out.Modifier = &m
	}
	return out
}
